package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateCode_SecurityGroupWithNegativePort tests code generation for
// security groups with negative port numbers (like -1 for all ports).
// Bug: Generates invalid variable names like "Port-1ICMP".
func TestGenerateCode_SecurityGroupWithNegativePort(t *testing.T) {
	content := []byte(`
Resources:
  MySecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Test SG
      SecurityGroupIngress:
        - IpProtocol: icmp
          FromPort: -1
          ToPort: -1
          CidrIp: "0.0.0.0/0"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "sg_test")
	code := files["network.go"]

	// Should NOT contain hyphen in variable name
	assert.NotContains(t, code, "Port-1", "Should not have hyphen in variable name")
	// Should generate typed struct for security group ingress rules
	assert.Contains(t, code, "ec2.SecurityGroup_Ingress{", "Should use typed SecurityGroup_Ingress struct")
	// Should contain the -1 port value
	assert.Contains(t, code, "FromPort: -1", "Should contain negative port value")
}

// TestGenerateCode_UnknownResourceType tests that unknown resource types are handled gracefully.
// Bug: Generates import to "resources/unknown" which doesn't exist.
func TestGenerateCode_UnknownResourceType(t *testing.T) {
	content := []byte(`
Resources:
  MyCustomResource:
    Type: Custom::MyResource
    Properties:
      Foo: bar
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "custom")

	// Should NOT generate import to resources/unknown
	for _, code := range files {
		assert.NotContains(t, code, `resources/unknown`, "Should not import resources/unknown")
	}

	// Should generate placeholder variable for unknown resource
	mainCode := files["main.go"]
	assert.Contains(t, mainCode, "var MyCustomResource any = nil", "Should generate placeholder variable")
	assert.Contains(t, mainCode, "placeholder for unknown resource type: Custom::MyResource", "Should have comment explaining placeholder")
}

// TestGenerateCode_UnknownResourceWithOutputRef tests that outputs can reference unknown resource types.
// Bug: Outputs referencing Custom::* resources cause undefined variable errors.
func TestGenerateCode_UnknownResourceWithOutputRef(t *testing.T) {
	content := []byte(`
Resources:
  ADConnectorResource:
    Type: Custom::ADConnectorResource
    Properties:
      ServiceToken: !Sub arn:aws:lambda:${AWS::Region}:${AWS::AccountId}:function:MyFunction
Outputs:
  DirectoryId:
    Value: !GetAtt ADConnectorResource.DirectoryId
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "adconnector")

	// Should generate placeholder variable
	mainCode := files["main.go"]
	assert.Contains(t, mainCode, "var ADConnectorResource any = nil", "Should generate placeholder variable for Custom:: resource")

	// Output should reference the placeholder (as GetAtt with quoted logical ID)
	outputsCode := files["outputs.go"]
	assert.Contains(t, outputsCode, `GetAtt{"ADConnectorResource"`, "Output should use GetAtt with quoted logical ID for custom resource")
}

// TestGenerateCode_SAMImplicitResources tests that outputs referencing SAM implicit
// resources (like auto-generated IAM roles) use explicit GetAtt{} instead of undefined variables.
func TestGenerateCode_SAMImplicitResources(t *testing.T) {
	content := []byte(`
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Resources:
  HelloWorldFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: bootstrap
      Runtime: go1.x
Outputs:
  FunctionArn:
    Value: !GetAtt HelloWorldFunction.Arn
  FunctionRoleArn:
    Description: Implicit IAM Role created by SAM
    Value: !GetAtt HelloWorldFunctionRole.Arn
`)

	ir, err := ParseTemplateContent(content, "sam-test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "samtest")

	outputsCode := files["outputs.go"]

	// Reference to existing function should use direct field access
	assert.Contains(t, outputsCode, "HelloWorldFunction.Arn", "Output should use direct field access for existing resource")

	// Reference to SAM implicit role should use explicit GetAtt{}
	assert.Contains(t, outputsCode, `GetAtt{"HelloWorldFunctionRole", "Arn"}`, "Output should use explicit GetAtt for SAM implicit role")
}

func TestGenerateCode_TransformVariableCollision(t *testing.T) {
	// Test that a resource named "Transform" doesn't collide with intrinsics.Transform type
	content := []byte(`
Resources:
  Transform:
    Type: AWS::CloudFormation::Macro
    Properties:
      Name: MyMacro
      FunctionName: arn:aws:lambda:us-east-1:123456789:function:MyMacro
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "transformtest")
	code := files["infra.go"]

	// Should rename "Transform" to "TransformResource" to avoid collision with intrinsics.Transform
	assert.Contains(t, code, "var TransformResource = cloudformation.Macro{", "Transform should be renamed to TransformResource")
	assert.NotContains(t, code, "var Transform = cloudformation.Macro{", "Should not use bare 'Transform' variable name")
}

func TestGenerateCode_ReservedNameReferences(t *testing.T) {
	// Test that references to resources with reserved names are also sanitized
	content := []byte(`
Resources:
  Transform:
    Type: AWS::CloudFormation::Macro
    Properties:
      Name: MyMacro
      FunctionName: arn:aws:lambda:us-east-1:123456789:function:MyMacro
  MyRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement: []
      Description: !Ref Transform
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "reftest")
	securityCode := files["security.go"]

	// Reference to Transform should also use the sanitized name
	assert.Contains(t, securityCode, "Description: TransformResource,", "Reference to Transform should use TransformResource")
	assert.NotContains(t, securityCode, "Description: Transform,", "Should not use bare 'Transform' reference")
}

// TestGenerateCode_DuplicateArrayElementNames tests that duplicate array elements get unique names.
// Issue #57: Variable name redeclaration.
func TestGenerateCode_DuplicateArrayElementNames(t *testing.T) {
	content := []byte(`
Resources:
  SecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Test SG
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 3306
          ToPort: 3306
          CidrIp: 10.0.0.0/24
        - IpProtocol: tcp
          FromPort: 3306
          ToPort: 3306
          CidrIp: 192.168.0.0/24
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "dupnames")
	networkCode := files["network.go"]

	// Should have two different variable names for the ingress rules
	assert.Contains(t, networkCode, "PortN3306 =", "First ingress rule should have PortN3306")
	assert.Contains(t, networkCode, "PortN3306_2 =", "Second ingress rule should have PortN3306_2")
}

// TestGenerateCode_LowercaseResourceName tests that lowercase resource names are capitalized.
// Issue #58: Lowercase variable prefix causes undefined errors.
func TestGenerateCode_LowercaseResourceName(t *testing.T) {
	content := []byte(`
Resources:
  myBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-bucket
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "lowercase")
	storageCode := files["storage.go"]

	// Variable name should be capitalized for export
	assert.Contains(t, storageCode, "var MyBucket =", "Lowercase resource name should be capitalized")
	assert.NotContains(t, storageCode, "var myBucket =", "Should not have lowercase variable name")
}

func TestGenerateCode_InitializationCycleBreaking(t *testing.T) {
	// Test that circular references between resources and property blocks
	// are broken by using explicit GetAtt{} instead of direct attribute access
	content := []byte(`
AWSTemplateFormatVersion: "2010-09-09"
Transform: AWS::Serverless-2016-10-31
Resources:
  # BaseFunction has DeploymentPreference that references PreHook
  BaseFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: base.handler
      Runtime: nodejs16.x
      DeploymentPreference:
        Type: AllAtOnce
        Hooks:
          PreTraffic: !Ref PreHook
  # PreHook references BaseFunction.Arn - this creates a cycle
  PreHook:
    Type: AWS::Serverless::Function
    Properties:
      Handler: prehook.handler
      Runtime: nodejs16.x
      Environment:
        Variables:
          FUNCTION_ARN: !GetAtt BaseFunction.Arn
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "template")
	code := files["main.go"]

	// The code should compile - cycles should be broken
	// Either by using GetAtt{} instead of BaseFunction.Arn
	// or by inlining the DeploymentPreference

	// Check that the cycle is broken - we expect GetAtt{} for the cyclic reference
	// instead of direct attribute access like BaseFunction.Arn
	assert.Contains(t, code, `GetAtt{"BaseFunction", "Arn"}`,
		"Cyclic reference should use explicit GetAtt{} to break initialization cycle")
}

func TestGenerateCode_DetectSAMImplicitResourcesEdgeCases(t *testing.T) {
	// Test SAM implicit resource detection with various event types
	yaml := `
AWSTemplateFormatVersion: "2010-09-09"
Transform: AWS::Serverless-2016-10-31
Resources:
  MyFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: index.handler
      Runtime: python3.12
      Events:
        HttpApi:
          Type: HttpApi
          Properties:
            Path: /hello
            Method: get
        Schedule:
          Type: Schedule
          Properties:
            Schedule: rate(1 hour)
`
	ir, err := ParseTemplateContent([]byte(yaml), "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "test")

	// Should generate function code (Serverless goes to main.go by default)
	code := files["main.go"]
	assert.Contains(t, code, "MyFunction")
}

func TestGenerateCode_ArrayElementTypeName(t *testing.T) {
	// Test getArrayElementTypeName through code with nested arrays
	yaml := `
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MySecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Test SG
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 0.0.0.0/0
`
	ir, err := ParseTemplateContent([]byte(yaml), "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "test")

	// Should have generated code with security group ingress rules (EC2 goes to network.go)
	code := files["network.go"]
	assert.Contains(t, code, "SecurityGroupIngress")
}
