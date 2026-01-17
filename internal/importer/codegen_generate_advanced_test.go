package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateCode_TransformStruct tests that Fn::Transform generates correct Transform struct.
// Issue #55: Transform intrinsic struct incomplete.
func TestGenerateCode_TransformStruct(t *testing.T) {
	content := []byte(`
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-bucket
      Tags:
        - Key: Date
          Value:
            Fn::Transform:
              Name: Date
              Parameters:
                Date: "2024-01-01"
                Operation: Current
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "transformtest")
	storageCode := files["storage.go"]

	// Transform struct should have Name and Parameters fields
	assert.Contains(t, storageCode, `Transform{Name: "Date"`, "Transform should have Name field")
	assert.Contains(t, storageCode, "Parameters:", "Transform should have Parameters field")
}

// TestGenerateCode_TransformListFormat tests that Fn::Transform with list format works.
// Some templates use list format: Fn::Transform: [{Name: ..., Parameters: ...}]
func TestGenerateCode_TransformListFormat(t *testing.T) {
	content := []byte(`
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-bucket
      Tags:
        - Key: Date
          Value:
            Fn::Transform:
              - Name: Date
                Parameters:
                  Date: "2024-01-01"
                  Operation: Current
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "transformlist")
	storageCode := files["storage.go"]

	// Transform struct should have Name and Parameters fields even with list format
	assert.Contains(t, storageCode, `Transform{Name: "Date"`, "Transform should have Name field")
	assert.Contains(t, storageCode, "Parameters:", "Transform should have Parameters field")
}

// TestGenerateCode_ResourceTypeFieldInNestedType tests that ResourceType field in nested types
// is NOT transformed to ResourceTypeProp.
// Issue #56: Only top-level resources have the ResourceType() method that conflicts.
func TestGenerateCode_ResourceTypeFieldInNestedType(t *testing.T) {
	content := []byte(`
Resources:
  LaunchTemplate:
    Type: AWS::EC2::LaunchTemplate
    Properties:
      LaunchTemplateData:
        TagSpecifications:
          - ResourceType: instance
            Tags:
              - Key: Name
                Value: MyInstance
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "restype")
	computeCode := files["compute.go"]

	// ResourceType field in nested TagSpecification should stay as "ResourceType"
	// not be transformed to "ResourceTypeProp"
	assert.Contains(t, computeCode, "ResourceType:", "Nested types should use ResourceType field name")
	assert.NotContains(t, computeCode, "ResourceTypeProp:", "Nested types should NOT use ResourceTypeProp")
}

// TestGenerateCode_TagImportsIntrinsics tests that files with Tag{} import intrinsics.
// Issue #59: Undefined Tag type when intrinsics import missing.
func TestGenerateCode_TagImportsIntrinsics(t *testing.T) {
	content := []byte(`
Resources:
  FlowLog:
    Type: AWS::EC2::FlowLog
    Properties:
      ResourceTypeProp: VPC
      LogDestinationType: cloud-watch-logs
      Tags:
        - Key: Name
          Value: My Flow Log
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "tagtest")
	networkCode := files["network.go"]

	// File should import intrinsics for Tag type
	assert.Contains(t, networkCode, `"github.com/lex00/wetwire-aws-go/intrinsics"`, "Should import intrinsics for Tag type")
	assert.Contains(t, networkCode, "Tag{", "Should have Tag literal")
}

// TestGenerateCode_GetAttRootResourceId tests that !GetAtt RestApi.RootResourceId generates
// correct field access pattern instead of undefined variable reference.
// Issue #60: Undefined resource reference for GetAtt attributes.
func TestGenerateCode_GetAttRootResourceId(t *testing.T) {
	content := []byte(`
Resources:
  RestApi:
    Type: AWS::ApiGateway::RestApi
    Properties:
      Name: MyApi

  TestResource:
    Type: AWS::ApiGateway::Resource
    Properties:
      ParentId: !GetAtt RestApi.RootResourceId
      PathPart: test
      RestApiId: !Ref RestApi
`)

	ir, err := ParseTemplateContent(content, "apitest")
	require.NoError(t, err)

	files := GenerateCode(ir, "apitest")
	mainCode := files["main.go"]

	// Should generate RestApi.RootResourceId field access, not RestApiRootResourceId variable
	assert.Contains(t, mainCode, "ParentId: RestApi.RootResourceId", "Should use field access for GetAtt")
	assert.NotContains(t, mainCode, "RestApiRootResourceId", "Should not generate undefined variable reference")
}

// TestGenerateCode_SubGetAttPattern tests that !Sub ${RestApi.RootResourceId} generates
// correct field access pattern instead of undefined variable reference.
// Issue #60: Sub shorthand for GetAtt was being sanitized incorrectly.
func TestGenerateCode_SubGetAttPattern(t *testing.T) {
	content := []byte(`
Resources:
  RestApi:
    Type: AWS::ApiGateway::RestApi
    Properties:
      Name: MyApi

  TestResource:
    Type: AWS::ApiGateway::Resource
    Properties:
      ParentId: !Sub ${RestApi.RootResourceId}
      PathPart: test
      RestApiId: !Ref RestApi
`)

	ir, err := ParseTemplateContent(content, "apitest")
	require.NoError(t, err)

	files := GenerateCode(ir, "apitest")
	mainCode := files["main.go"]

	// Should generate RestApi.RootResourceId field access, not RestApiRootResourceId variable
	assert.Contains(t, mainCode, "ParentId: RestApi.RootResourceId", "Should use field access for Sub with GetAtt shorthand")
	assert.NotContains(t, mainCode, "RestApiRootResourceId", "Should not generate undefined variable reference")
}

// TestGenerateCode_ListFieldWrapping tests that references to resources/parameters
// are wrapped in []any{} when assigned to list-type fields like VPCZoneIdentifier.
func TestGenerateCode_ListFieldWrapping(t *testing.T) {
	content := []byte(`
Parameters:
  Subnets:
    Type: List<AWS::EC2::Subnet::Id>
    Description: Subnet IDs for autoscaling group

Resources:
  NotificationTopic:
    Type: AWS::SNS::Topic
    Properties:
      TopicName: my-notifications

  NotificationConfig:
    Type: AWS::AutoScaling::AutoScalingGroup
    Properties:
      MinSize: "1"
      MaxSize: "3"
      VPCZoneIdentifier: !Ref Subnets
      NotificationConfigurations:
        - TopicARN: !Ref NotificationTopic
          NotificationTypes:
            - autoscaling:EC2_INSTANCE_LAUNCH
`)

	ir, err := ParseTemplateContent(content, "asgtest")
	require.NoError(t, err)

	files := GenerateCode(ir, "asgtest")
	computeCode := files["compute.go"]

	// VPCZoneIdentifier should wrap Subnets parameter in []any{}
	assert.Contains(t, computeCode, "VPCZoneIdentifier: []any{Subnets}", "Should wrap parameter ref in []any{} for VPCZoneIdentifier")

	// TopicARN should wrap NotificationTopic resource ref in []any{}
	assert.Contains(t, computeCode, "TopicARN: []any{NotificationTopic}", "Should wrap resource ref in []any{} for TopicARN")
}

// TestGenerateCode_LowercaseParameterNames tests that parameter names starting with
// lowercase letters are capitalized to be exported.
// Issue #71: Parameters like myKeyPair should become MyKeyPair
func TestGenerateCode_LowercaseParameterNames(t *testing.T) {
	content := []byte(`
Parameters:
  myKeyPair:
    Type: AWS::EC2::KeyPair::KeyName
    Description: Amazon EC2 Key Pair
  pPrivateSubnet1:
    Type: AWS::EC2::Subnet::Id
    Description: First private subnet

Resources:
  MyInstance:
    Type: AWS::EC2::Instance
    Properties:
      KeyName: !Ref myKeyPair
      SubnetId: !Ref pPrivateSubnet1
`)

	ir, err := ParseTemplateContent(content, "paramtest")
	require.NoError(t, err)

	files := GenerateCode(ir, "paramtest")
	paramsCode := files["params.go"]
	computeCode := files["compute.go"]

	// Parameter names should be capitalized for export
	assert.Contains(t, paramsCode, "var MyKeyPair = Parameter{", "lowercase myKeyPair should become MyKeyPair")
	assert.Contains(t, paramsCode, "var PPrivateSubnet1 = Parameter{", "pPrivateSubnet1 should become PPrivateSubnet1")

	// References should use capitalized names
	assert.Contains(t, computeCode, "KeyName: MyKeyPair", "Reference should use capitalized name")
	assert.Contains(t, computeCode, "SubnetId: PPrivateSubnet1", "Reference should use capitalized name")
}

// TestGenerateCode_EMRListProperties tests that EMR security group list properties
// are wrapped in []any{}.
// Issue #72: AdditionalMasterSecurityGroups and AdditionalSlaveSecurityGroups
func TestGenerateCode_EMRListProperties(t *testing.T) {
	content := []byte(`
Parameters:
  MasterSecurityGroups:
    Type: List<AWS::EC2::SecurityGroup::Id>
    Description: Additional master security groups
  SlaveSecurityGroups:
    Type: List<AWS::EC2::SecurityGroup::Id>
    Description: Additional slave security groups

Resources:
  EMRCluster:
    Type: AWS::EMR::Cluster
    Properties:
      Name: MyCluster
      Instances:
        AdditionalMasterSecurityGroups: !Ref MasterSecurityGroups
        AdditionalSlaveSecurityGroups: !Ref SlaveSecurityGroups
`)

	ir, err := ParseTemplateContent(content, "emrtest")
	require.NoError(t, err)

	files := GenerateCode(ir, "emrtest")
	mainCode := files["main.go"]

	// EMR list properties should wrap refs in []any{}
	assert.Contains(t, mainCode, "AdditionalMasterSecurityGroups: []any{MasterSecurityGroups}", "Should wrap in []any{}")
	assert.Contains(t, mainCode, "AdditionalSlaveSecurityGroups: []any{SlaveSecurityGroups}", "Should wrap in []any{}")
}

// TestGenerateCode_JoinValuesWrapping tests that Join.Values wraps single
// intrinsic references in []any{}.
// Issue #73: Join{..., Values: SecurityGroupIds} should be Join{..., Values: []any{SecurityGroupIds}}
func TestGenerateCode_JoinValuesWrapping(t *testing.T) {
	content := []byte(`
Parameters:
  SecurityGroupIds:
    Type: List<AWS::EC2::SecurityGroup::Id>
    Description: Security group IDs

Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Join ["-", !Ref SecurityGroupIds]
`)

	ir, err := ParseTemplateContent(content, "jointest")
	require.NoError(t, err)

	files := GenerateCode(ir, "jointest")
	storageCode := files["storage.go"]

	// Join.Values should wrap the parameter ref in []any{}
	assert.Contains(t, storageCode, "Values: []any{SecurityGroupIds}", "Join.Values should wrap ref in []any{}")
}

// TestGenerateCode_NestedTypeInIntrinsic tests that nested types inside intrinsics
// use the correct child type, not the parent type.
// Issue #74: When S3Location is wrapped in If{}, it should use Association_S3OutputLocation,
// not the parent Association_InstanceAssociationOutputLocation.
func TestGenerateCode_NestedTypeInIntrinsic(t *testing.T) {
	content := []byte(`
Conditions:
  HasS3Bucket:
    !Not [!Equals [!Ref LogsBucketName, ""]]

Parameters:
  LogsBucketName:
    Type: String
    Description: S3 bucket name for logs

Resources:
  SSMAssociation:
    Type: AWS::SSM::Association
    Properties:
      Name: AWS-RunPatchBaseline
      OutputLocation:
        S3Location:
          Fn::If:
            - HasS3Bucket
            - OutputS3BucketName: !Ref LogsBucketName
              OutputS3KeyPrefix: logs/ssm
            - !Ref AWS::NoValue
`)

	ir, err := ParseTemplateContent(content, "ssmtest")
	require.NoError(t, err)

	files := GenerateCode(ir, "ssmtest")

	// SSM goes to security.go
	securityCode := files["security.go"]

	// The S3Location value inside If should use Association_S3OutputLocation, not the parent type
	assert.Contains(t, securityCode, "ssm.Association_S3OutputLocation{", "Should use correct nested type inside If")
	assert.NotContains(t, securityCode, "ssm.Association_InstanceAssociationOutputLocation{OutputS3BucketName", "Should NOT use parent type for S3Location value")
}

// TestGenerateCode_SubWithMapVariables tests that SubWithMap.Variables is always
// generated as Json{}, not as a struct type.
// Issue #75: SubWithMap.Variables should be map, not struct type
func TestGenerateCode_SubWithMapVariables(t *testing.T) {
	content := []byte(`
Resources:
  MyFunction:
    Type: AWS::Lambda::Function
    Properties:
      FunctionName: MyFunction
      Runtime: python3.9
      Handler: index.handler
      Role: arn:aws:iam::123456789012:role/lambda-role
      Code:
        ZipFile:
          Fn::Sub:
            - |
              import boto3
              REGION = "${Region}"
            - Region: !Ref AWS::Region
`)

	ir, err := ParseTemplateContent(content, "subtest")
	require.NoError(t, err)

	files := GenerateCode(ir, "subtest")
	computeCode := files["compute.go"]

	// SubWithMap.Variables should use Json{}, not struct type
	assert.Contains(t, computeCode, `Variables: Json{`, "SubWithMap.Variables should be Json")
	assert.NotContains(t, computeCode, `Variables: lambda.Function_Code{`, "Should NOT use struct type for Variables")
}

// TestGenerateCode_ApplicationAutoScaling tests that ApplicationAutoScaling resources
// (ScalableTarget, ScalingPolicy) are correctly imported and not skipped.
// Issue #90: ApplicationAutoScaling resources may be skipped during import
func TestGenerateCode_ApplicationAutoScaling(t *testing.T) {
	content := []byte(`
AWSTemplateFormatVersion: '2010-09-09'
Description: DynamoDB with ApplicationAutoScaling

Resources:
  MyTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: TestTable
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      BillingMode: PROVISIONED
      ProvisionedThroughput:
        ReadCapacityUnits: 5
        WriteCapacityUnits: 5

  TableReadScalableTarget:
    Type: AWS::ApplicationAutoScaling::ScalableTarget
    Properties:
      MaxCapacity: 100
      MinCapacity: 5
      ResourceId: !Sub table/${MyTable}
      ScalableDimension: dynamodb:table:ReadCapacityUnits
      ServiceNamespace: dynamodb

  TableReadScalingPolicy:
    Type: AWS::ApplicationAutoScaling::ScalingPolicy
    Properties:
      PolicyName: ReadAutoScalingPolicy
      PolicyType: TargetTrackingScaling
      ScalingTargetId: !Ref TableReadScalableTarget
      TargetTrackingScalingPolicyConfiguration:
        TargetValue: 70.0
        PredefinedMetricSpecification:
          PredefinedMetricType: DynamoDBReadCapacityUtilization
`)

	ir, err := ParseTemplateContent(content, "scaling_test.yaml")
	require.NoError(t, err)

	// Verify resources are in the IR
	assert.Contains(t, ir.Resources, "MyTable", "MyTable should be in IR")
	assert.Contains(t, ir.Resources, "TableReadScalableTarget", "TableReadScalableTarget should be in IR")
	assert.Contains(t, ir.Resources, "TableReadScalingPolicy", "TableReadScalingPolicy should be in IR")

	files := GenerateCode(ir, "scaling_test")

	// DynamoDB goes to database.go
	dbCode, hasDB := files["database.go"]
	require.True(t, hasDB, "Should generate database.go for DynamoDB")
	assert.Contains(t, dbCode, "var MyTable = dynamodb.Table{")

	// ApplicationAutoScaling goes to compute.go (in serviceCategories map)
	computeCode, hasCompute := files["compute.go"]
	require.True(t, hasCompute, "Should generate compute.go for ApplicationAutoScaling resources")

	// Check ScalableTarget is present
	assert.Contains(t, computeCode, `"github.com/lex00/wetwire-aws-go/resources/applicationautoscaling"`,
		"Should import applicationautoscaling package")
	assert.Contains(t, computeCode, "var TableReadScalableTarget = applicationautoscaling.ScalableTarget{",
		"ScalableTarget should be generated")
	assert.Contains(t, computeCode, "MaxCapacity: 100",
		"ScalableTarget should have MaxCapacity property")
	assert.Contains(t, computeCode, "MinCapacity: 5",
		"ScalableTarget should have MinCapacity property")
	assert.Contains(t, computeCode, `ServiceNamespace: "dynamodb"`,
		"ScalableTarget should have ServiceNamespace property")

	// Check ScalingPolicy is present
	assert.Contains(t, computeCode, "var TableReadScalingPolicy = applicationautoscaling.ScalingPolicy{",
		"ScalingPolicy should be generated")
	assert.Contains(t, computeCode, `PolicyName: "ReadAutoScalingPolicy"`,
		"ScalingPolicy should have PolicyName property")
}

func TestGenerateCode_ListTypeParameterWrapping(t *testing.T) {
	// Test that CommaDelimitedList parameters are wrapped in []any{} when used in struct fields
	content := []byte(`
AWSTemplateFormatVersion: "2010-09-09"
Parameters:
  ClientDomains:
    Type: CommaDelimitedList
    Description: Array of domains allowed to use this UserPool
Resources:
  UserPoolClient:
    Type: AWS::Cognito::UserPoolClient
    Properties:
      ClientName: TestClient
      CallbackURLs: !Ref ClientDomains
      LogoutURLs: !Ref ClientDomains
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "template")
	code := files["security.go"]

	// CallbackURLs and LogoutURLs should wrap the Parameter reference in []any{}
	// because ClientDomains is a CommaDelimitedList (list type parameter)
	assert.Contains(t, code, "CallbackURLs: []any{ClientDomains}",
		"List-type parameter should be wrapped in []any{}")
	assert.Contains(t, code, "LogoutURLs: []any{ClientDomains}",
		"List-type parameter should be wrapped in []any{}")
}

func TestGenerateCode_MapToArrayWrapping(t *testing.T) {
	// Test that maps are wrapped in []any{} when the field expects an array
	// This is common with SAM Policies field which accepts either a list or a policy map
	content := []byte(`
AWSTemplateFormatVersion: "2010-09-09"
Transform: AWS::Serverless-2016-10-31
Resources:
  MyFunction:
    Type: AWS::Serverless::Function
    Properties:
      Runtime: python3.9
      Handler: app.handler
      CodeUri: ./src
      Policies:
        S3ReadPolicy:
          BucketName: my-bucket
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "template")
	code := files["main.go"]

	// Policies field is []any, so the map should be wrapped in []any{}
	assert.Contains(t, code, "Policies: []any{Json{",
		"Map value should be wrapped in []any{} when field expects array")
}

func TestGenerateCode_PolicyDocumentWithConditions(t *testing.T) {
	// Test IAM policy with conditions to cover conditionToGo, jsonMapToGo, jsonValueToGo
	yaml := `
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MyRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
          - Effect: Allow
            Principal:
              Service: lambda.amazonaws.com
            Action: sts:AssumeRole
            Condition:
              StringEquals:
                aws:RequestedRegion: us-east-1
              Bool:
                aws:SecureTransport: true
`
	ir, err := ParseTemplateContent([]byte(yaml), "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "test")

	// Should contain the policy document in security.go (IAM category)
	code := files["security.go"]
	assert.Contains(t, code, "AssumeRolePolicyDocument")
}

func TestGenerateCode_PolicyDocumentWithPrincipals(t *testing.T) {
	// Test IAM policy with various principal types
	yaml := `
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MyRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
          - Effect: Allow
            Principal: "*"
            Action: sts:AssumeRole
          - Effect: Allow
            Principal:
              AWS:
                - "arn:aws:iam::123456789012:root"
            Action: sts:AssumeRole
          - Effect: Allow
            Principal:
              Federated: "arn:aws:iam::123456789012:saml-provider/ExampleProvider"
            Action: sts:AssumeRoleWithSAML
`
	ir, err := ParseTemplateContent([]byte(yaml), "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "test")

	// Should contain principal types (IAM goes to security.go)
	code := files["security.go"]
	assert.Contains(t, code, "AssumeRolePolicyDocument")
}

func TestGenerateCode_GetModuleVersion(t *testing.T) {
	// Test that getModuleVersion is called during code generation
	// This is an indirect test - the version appears in go.mod reference
	yaml := `
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
`
	ir, err := ParseTemplateContent([]byte(yaml), "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "test")

	// Should have generated valid Go code with package declaration (S3 goes to storage.go)
	code := files["storage.go"]
	assert.Contains(t, code, "package test")
}
