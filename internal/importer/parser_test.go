package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplateContent_SimpleYAML(t *testing.T) {
	content := []byte(`
AWSTemplateFormatVersion: "2010-09-09"
Description: Test template

Parameters:
  BucketPrefix:
    Type: String
    Default: test

Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub "${BucketPrefix}-bucket"

Outputs:
  BucketArn:
    Value: !GetAtt MyBucket.Arn
    Description: The bucket ARN
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	assert.Equal(t, "Test template", ir.Description)
	assert.Equal(t, "2010-09-09", ir.AWSTemplateFormatVersion)

	// Check parameter
	assert.Len(t, ir.Parameters, 1)
	param := ir.Parameters["BucketPrefix"]
	assert.NotNil(t, param)
	assert.Equal(t, "String", param.Type)
	assert.Equal(t, "test", param.Default)

	// Check resource
	assert.Len(t, ir.Resources, 1)
	resource := ir.Resources["MyBucket"]
	assert.NotNil(t, resource)
	assert.Equal(t, "AWS::S3::Bucket", resource.ResourceType)
	assert.Equal(t, "S3", resource.Service())
	assert.Equal(t, "Bucket", resource.TypeName())

	// Check BucketName property is a Sub intrinsic
	bucketNameProp := resource.Properties["BucketName"]
	require.NotNil(t, bucketNameProp)
	intrinsic, ok := bucketNameProp.Value.(*IRIntrinsic)
	require.True(t, ok, "BucketName should be an intrinsic")
	assert.Equal(t, IntrinsicSub, intrinsic.Type)

	// Check output
	assert.Len(t, ir.Outputs, 1)
	output := ir.Outputs["BucketArn"]
	assert.NotNil(t, output)
	assert.Equal(t, "The bucket ARN", output.Description)
	getAttIntrinsic, ok := output.Value.(*IRIntrinsic)
	require.True(t, ok, "Output value should be a GetAtt intrinsic")
	assert.Equal(t, IntrinsicGetAtt, getAttIntrinsic.Type)

	// Check reference graph
	assert.Contains(t, ir.ReferenceGraph["MyBucket"], "BucketPrefix")
	assert.Contains(t, ir.ReferenceGraph["BucketArn"], "MyBucket")
}

func TestParseTemplateContent_JSON(t *testing.T) {
	content := []byte(`{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "MyVPC": {
      "Type": "AWS::EC2::VPC",
      "Properties": {
        "CidrBlock": "10.0.0.0/16"
      }
    },
    "MySubnet": {
      "Type": "AWS::EC2::Subnet",
      "Properties": {
        "VpcId": {"Ref": "MyVPC"},
        "CidrBlock": "10.0.1.0/24"
      }
    }
  }
}`)

	ir, err := ParseTemplateContent(content, "test.json")
	require.NoError(t, err)

	assert.Len(t, ir.Resources, 2)

	vpc := ir.Resources["MyVPC"]
	assert.Equal(t, "AWS::EC2::VPC", vpc.ResourceType)

	subnet := ir.Resources["MySubnet"]
	assert.Equal(t, "AWS::EC2::Subnet", subnet.ResourceType)

	// Check VpcId is a Ref intrinsic
	vpcIdProp := subnet.Properties["VpcId"]
	require.NotNil(t, vpcIdProp)
	refIntrinsic, ok := vpcIdProp.Value.(*IRIntrinsic)
	require.True(t, ok)
	assert.Equal(t, IntrinsicRef, refIntrinsic.Type)
	assert.Equal(t, "MyVPC", refIntrinsic.Args)
}

func TestParseTemplateContent_AllIntrinsics(t *testing.T) {
	content := []byte(`
Resources:
  Test:
    Type: AWS::CloudFormation::WaitConditionHandle
    Properties:
      RefProp: !Ref MyParam
      GetAttProp: !GetAtt OtherResource.Arn
      SubProp: !Sub "arn:aws:s3:::${BucketName}"
      JoinProp: !Join [",", ["a", "b", "c"]]
      SelectProp: !Select [0, ["first", "second"]]
      GetAZsProp: !GetAZs ""
      IfProp: !If [CreateProd, "prod", "dev"]
      EqualsProp: !Equals ["a", "b"]
      Base64Prop: !Base64 "Hello World"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	resource := ir.Resources["Test"]
	require.NotNil(t, resource)

	// Check each intrinsic type
	checkIntrinsic := func(propName string, expectedType IntrinsicType) {
		prop := resource.Properties[propName]
		require.NotNil(t, prop, "Property %s should exist", propName)
		intrinsic, ok := prop.Value.(*IRIntrinsic)
		require.True(t, ok, "Property %s should be an intrinsic", propName)
		assert.Equal(t, expectedType, intrinsic.Type, "Property %s should be %s", propName, expectedType)
	}

	checkIntrinsic("RefProp", IntrinsicRef)
	checkIntrinsic("GetAttProp", IntrinsicGetAtt)
	checkIntrinsic("SubProp", IntrinsicSub)
	checkIntrinsic("JoinProp", IntrinsicJoin)
	checkIntrinsic("SelectProp", IntrinsicSelect)
	checkIntrinsic("GetAZsProp", IntrinsicGetAZs)
	checkIntrinsic("IfProp", IntrinsicIf)
	checkIntrinsic("EqualsProp", IntrinsicEquals)
	checkIntrinsic("Base64Prop", IntrinsicBase64)
}

func TestParseTemplateContent_Conditions(t *testing.T) {
	content := []byte(`
Parameters:
  Environment:
    Type: String
    AllowedValues:
      - prod
      - dev

Conditions:
  IsProd: !Equals [!Ref Environment, "prod"]
  NotProd: !Not [!Condition IsProd]
  ProdOrStaging: !Or
    - !Equals [!Ref Environment, "prod"]
    - !Equals [!Ref Environment, "staging"]

Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Condition: IsProd
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	assert.Len(t, ir.Conditions, 3)

	// Check IsProd condition
	isProd := ir.Conditions["IsProd"]
	require.NotNil(t, isProd)
	equalsIntrinsic, ok := isProd.Expression.(*IRIntrinsic)
	require.True(t, ok)
	assert.Equal(t, IntrinsicEquals, equalsIntrinsic.Type)

	// Check NotProd condition
	notProd := ir.Conditions["NotProd"]
	require.NotNil(t, notProd)
	notIntrinsic, ok := notProd.Expression.(*IRIntrinsic)
	require.True(t, ok)
	assert.Equal(t, IntrinsicNot, notIntrinsic.Type)

	// Check resource condition
	bucket := ir.Resources["Bucket"]
	assert.Equal(t, "IsProd", bucket.Condition)
}

func TestParseTemplateContent_Mappings(t *testing.T) {
	content := []byte(`
Mappings:
  RegionAMI:
    us-east-1:
      HVM64: ami-0123456789abcdef0
      HVM32: ami-0fedcba9876543210
    us-west-2:
      HVM64: ami-abcdef0123456789a
      HVM32: ami-9876543210fedcba0
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	assert.Len(t, ir.Mappings, 1)

	regionAMI := ir.Mappings["RegionAMI"]
	require.NotNil(t, regionAMI)

	assert.Contains(t, regionAMI.MapData, "us-east-1")
	assert.Contains(t, regionAMI.MapData, "us-west-2")
	assert.Equal(t, "ami-0123456789abcdef0", regionAMI.MapData["us-east-1"]["HVM64"])
}

func TestParseTemplateContent_UnsupportedTags(t *testing.T) {
	content := []byte(`
Resources:
  Bucket:
    Type: !Rain::S3::Bucket
`)

	_, err := ParseTemplateContent(content, "test.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Rain-specific tags")
}

func TestParseTemplateContent_KubernetesManifest(t *testing.T) {
	content := []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test
`)

	_, err := ParseTemplateContent(content, "test.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Kubernetes manifest")
}

func TestDerivePackageName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"template.yaml", "template"},
		{"my-stack.yaml", "my_stack"},
		{"MyStack.json", "mystack"},
		{"123-invalid.yaml", "_23_invalid"},
		{"/path/to/vpc.template.yaml", "vpc_template"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := DerivePackageName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntrinsicType_String(t *testing.T) {
	tests := []struct {
		it       IntrinsicType
		expected string
	}{
		{IntrinsicRef, "Ref"},
		{IntrinsicGetAtt, "GetAtt"},
		{IntrinsicSub, "Sub"},
		{IntrinsicJoin, "Join"},
		{IntrinsicSelect, "Select"},
		{IntrinsicGetAZs, "GetAZs"},
		{IntrinsicIf, "If"},
		{IntrinsicEquals, "Equals"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.it.String())
		})
	}
}

func TestIRResource_ServiceAndTypeName(t *testing.T) {
	resource := &IRResource{
		ResourceType: "AWS::S3::Bucket",
	}

	assert.Equal(t, "S3", resource.Service())
	assert.Equal(t, "Bucket", resource.TypeName())

	resource2 := &IRResource{
		ResourceType: "AWS::Lambda::Function",
	}

	assert.Equal(t, "Lambda", resource2.Service())
	assert.Equal(t, "Function", resource2.TypeName())
}
