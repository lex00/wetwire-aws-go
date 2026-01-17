package importer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCode_SimpleTemplate(t *testing.T) {
	content := []byte(`
AWSTemplateFormatVersion: "2010-09-09"
Description: Simple S3 bucket

Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-test-bucket
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "mystack")

	// S3 bucket goes to storage.go with category-based splitting
	code, ok := files["storage.go"]
	require.True(t, ok, "Should generate storage.go for S3 bucket")

	// Check package declaration
	assert.Contains(t, code, "package mystack")

	// Check import
	assert.Contains(t, code, `"github.com/lex00/wetwire-aws-go/resources/s3"`)

	// Check resource variable
	assert.Contains(t, code, "var MyBucket = s3.Bucket{")
	assert.Contains(t, code, `BucketName: "my-test-bucket"`)
}

func TestGenerateCode_WithParameters(t *testing.T) {
	content := []byte(`
Parameters:
  Environment:
    Type: String
    Description: Environment name
    Default: dev
    AllowedValues:
      - dev
      - prod
  UnusedParam:
    Type: String
    Description: This param is not referenced

Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref Environment
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "params")
	// Parameters go to params.go
	paramsCode := files["params.go"]

	// Used parameters ARE generated as full Parameter{} structs
	assert.Contains(t, paramsCode, "// Environment - Environment name")
	assert.Contains(t, paramsCode, "var Environment = Parameter{")
	assert.Contains(t, paramsCode, `Type: "String",`)

	// Unused parameters are NOT generated
	assert.NotContains(t, paramsCode, "UnusedParam")

	// S3 bucket goes to storage.go
	storageCode := files["storage.go"]
	// Parameter is referenced by bare identifier
	assert.Contains(t, storageCode, "BucketName: Environment,")
}

func TestGenerateCode_WithOutputs(t *testing.T) {
	content := []byte(`
Resources:
  MyBucket:
    Type: AWS::S3::Bucket

Outputs:
  BucketArn:
    Description: The ARN of the bucket
    Value: !GetAtt MyBucket.Arn
    Export:
      Name: !Sub "${AWS::StackName}-BucketArn"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "outputs")
	// Outputs go to outputs.go
	code := files["outputs.go"]

	// Check output uses Output type
	assert.Contains(t, code, "// BucketArnOutput - The ARN of the bucket")
	assert.Contains(t, code, "var BucketArnOutput = Output{")
	assert.Contains(t, code, "Value:")
	assert.Contains(t, code, "Description:")
	assert.Contains(t, code, "ExportName:")
}

func TestGenerateCode_WithConditions(t *testing.T) {
	content := []byte(`
Parameters:
  Environment:
    Type: String

Conditions:
  IsProd: !Equals [!Ref Environment, "prod"]

Resources:
  ProdBucket:
    Type: AWS::S3::Bucket
    Condition: IsProd
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "cond")
	// Conditions go to params.go with parameters
	code := files["params.go"]

	// Check condition - uses bare parameter identifier (typed via Param())
	assert.Contains(t, code, `var IsProdCondition = Equals{Environment, "prod"}`)
}

func TestGenerateCode_WithMappings(t *testing.T) {
	content := []byte(`
Mappings:
  RegionMap:
    us-east-1:
      AMI: ami-12345
    us-west-2:
      AMI: ami-67890

Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !FindInMap [RegionMap, !Ref "AWS::Region", AMI]
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "mapping")

	// Mappings go to params.go; find the mapping in any generated file
	var foundMapping bool
	for _, code := range files {
		if strings.Contains(code, "var RegionMapMapping = ") {
			foundMapping = true
			assert.Contains(t, code, `"us-east-1"`)
			assert.Contains(t, code, `"ami-12345"`)
			break
		}
	}
	assert.True(t, foundMapping, "Should generate RegionMapMapping in some file")
}

// TestGenerateCode_NoUnusedImports tests that intrinsics import is only added when used.
// Bug: Intrinsics imported even when no intrinsic types are used.
func TestGenerateCode_NoUnusedImports(t *testing.T) {
	content := []byte(`
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: "static-bucket-name"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "nointrinsics")
	code := files["storage.go"]

	// Should NOT have intrinsics import since no intrinsics are used
	assert.NotContains(t, code, `wetwire-aws-go/intrinsics`, "Should not import intrinsics when not used")
}

// TestGenerateCode_MappingsNoUnusedImports tests that mappings-only files don't have unused imports.
func TestGenerateCode_MappingsNoUnusedImports(t *testing.T) {
	content := []byte(`
Mappings:
  RegionMap:
    us-east-1:
      AMI: ami-12345
    us-west-2:
      AMI: ami-67890
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "mappingsonly")

	// If main.go is generated for mappings, it should not have unused intrinsics import
	if code, ok := files["main.go"]; ok {
		// If intrinsics is imported, there should be usage of it
		if strings.Contains(code, `wetwire-aws-go/intrinsics`) {
			// Verify there's actual usage (Sub, List, Param, AWS_*, etc.)
			// Note: Ref{} and GetAtt{} should NOT be generated (style violation)
			hasUsage := strings.Contains(code, "Sub{") ||
				strings.Contains(code, "List(") ||
				strings.Contains(code, "Param(") ||
				strings.Contains(code, "AWS_")
			assert.True(t, hasUsage, "If intrinsics is imported, it should be used")
		}
		// Verify we never generate explicit Ref{} or GetAtt{} (style violations)
		assert.False(t, strings.Contains(code, `Ref{"`), "Should not generate Ref{} - use direct variable refs")
		assert.False(t, strings.Contains(code, `GetAtt{"`), "Should not generate GetAtt{} - use Resource.Attr")
	}
}

func TestGenerateCode_DependencyOrder(t *testing.T) {
	content := []byte(`
Resources:
  # Define in reverse order to test sorting
  MySubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref MyVPC
      CidrBlock: "10.0.1.0/24"

  MyVPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: "10.0.0.0/16"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "network")
	// VPC and Subnet are EC2 network resources -> network.go
	code := files["network.go"]

	// VPC should appear before Subnet in the generated code
	vpcIdx := strings.Index(code, "var MyVPC")
	subnetIdx := strings.Index(code, "var MySubnet")

	assert.True(t, vpcIdx < subnetIdx, "VPC should be defined before Subnet")
}

// TestGenerateCode_GetAZsRegionStringField tests that GetAZs.Region uses empty string, not AWS_REGION.
// Issue #36: GetAZs{Region: AWS_REGION} causes type mismatch - Region field expects string, not Ref.
func TestGenerateCode_GetAZsRegionStringField(t *testing.T) {
	content := []byte(`
Resources:
  MyVPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: "10.0.0.0/16"

  MySubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref MyVPC
      CidrBlock: "10.0.1.0/24"
      AvailabilityZone:
        Fn::Select:
          - 0
          - Fn::GetAZs:
              Ref: "AWS::Region"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "aztest")
	code := files["network.go"]

	// Should NOT use AWS_REGION in GetAZs - Region field is string type, not any
	assert.NotContains(t, code, "GetAZs{Region: AWS_REGION}", "GetAZs.Region should not use AWS_REGION (Ref type)")

	// Should generate valid Go code - either GetAZs{} or GetAZs{Region: ""}
	// Both are valid and mean "use current region"
	hasValidGetAZs := strings.Contains(code, `GetAZs{}`) || strings.Contains(code, `GetAZs{Region: ""}`)
	assert.True(t, hasValidGetAZs, "GetAZs should be generated without AWS_REGION")
}

// TestGenerateCode_GetAZsEmptyString tests that GetAZs with empty string generates valid code.
func TestGenerateCode_GetAZsEmptyString(t *testing.T) {
	content := []byte(`
Resources:
  MySubnet:
    Type: AWS::EC2::Subnet
    Properties:
      CidrBlock: "10.0.1.0/24"
      AvailabilityZone: !Select [0, !GetAZs ""]
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "aztest2")
	code := files["network.go"]

	// Should generate valid GetAZs - either GetAZs{} or GetAZs{Region: ""}
	// Both forms are equivalent (empty string means current region)
	hasValidGetAZs := strings.Contains(code, `GetAZs{}`) || strings.Contains(code, `GetAZs{Region: ""}`)
	assert.True(t, hasValidGetAZs, "GetAZs with empty string should generate valid code")
	assert.NotContains(t, code, "AWS_REGION", "GetAZs should not use AWS_REGION")
}

// TestGenerateCode_ParamsOnlyNoUnusedImports tests that params-only templates don't have unused resource imports.
// Issue #36: When only parameters are used, resource imports should not be added.
func TestGenerateCode_ParamsOnlyNoUnusedImports(t *testing.T) {
	content := []byte(`
Parameters:
  Environment:
    Type: String
    Description: Environment name
    Default: dev

  BucketName:
    Type: String
    Description: S3 bucket name
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "paramsonly")

	// Check that no resource packages are imported (params-only template)
	for filename, code := range files {
		// Should not import any resource packages
		assert.NotContains(t, code, `"github.com/lex00/wetwire-aws-go/resources/`,
			"File %s should not import resource packages for params-only template", filename)
	}
}

// TestGenerateCode_GetAZsInListField tests that GetAZs is wrapped in []any{} for list-type fields.
// Issue #38: GetAZs{} is incompatible with []any fields like AvailabilityZones.
func TestGenerateCode_GetAZsInListField(t *testing.T) {
	content := []byte(`
Resources:
  WebServerGroup:
    Type: AWS::AutoScaling::AutoScalingGroup
    Properties:
      MinSize: 1
      MaxSize: 3
      AvailabilityZones: !GetAZs ""
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "listtest")
	code := files["compute.go"]

	// Should wrap GetAZs in []any{} for list-type fields
	assert.Contains(t, code, "[]any{GetAZs{}}", "GetAZs should be wrapped in []any{} for AvailabilityZones field")
	// Should NOT have bare GetAZs{} assignment to list field
	assert.NotContains(t, code, "AvailabilityZones: GetAZs{}", "GetAZs should not be assigned directly to []any field")
}

// TestGenerateCode_ParameterInListField tests that Parameters are wrapped for list-type fields.
// Issue #38: Parameter{} is incompatible with []any fields.
func TestGenerateCode_ParameterInListField(t *testing.T) {
	content := []byte(`
Parameters:
  Subnets:
    Type: CommaDelimitedList
    Description: List of subnet IDs

Resources:
  MyLambda:
    Type: AWS::Lambda::Function
    Properties:
      FunctionName: test
      Runtime: python3.9
      Handler: index.handler
      Role: arn:aws:iam::123456789012:role/lambda-role
      VpcConfig:
        SubnetIds: !Ref Subnets
        SecurityGroupIds: []
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "paramlisttest")
	code := files["compute.go"]

	// Should wrap Parameter ref in []any{} for list-type fields
	// Or use the parameter directly if it's a CommaDelimitedList (which serializes to a list)
	hasValidSubnets := strings.Contains(code, "[]any{Subnets}") || strings.Contains(code, "SubnetIds: Subnets")
	assert.True(t, hasValidSubnets, "Parameter should be usable in list-type fields")
}

// TestGenerateCode_SelectIndexAsInt tests that Select.Index is generated as int, not string.
// Issue #39: Select index was generated as "0" instead of 0.
func TestGenerateCode_SelectIndexAsInt(t *testing.T) {
	content := []byte(`
Resources:
  MySubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: vpc-123
      CidrBlock: "10.0.0.0/24"
      AvailabilityZone: !Select ["0", !GetAZs ""]
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "selecttest")
	code := files["network.go"]

	// Should have integer index, not string
	assert.Contains(t, code, "Select{Index: 0,", "Select index should be integer, not string")
	assert.NotContains(t, code, `Select{Index: "0"`, "Select index should not be quoted string")
}

func TestGenerateCode_IfIntrinsicWithPropertyType(t *testing.T) {
	// Test that If{} intrinsic on property type fields doesn't generate & prefix
	// Issue #41: If{} intrinsic incompatible with pointer fields
	content := []byte(`
Conditions:
  HasLogging:
    !Equals [!Ref EnableLogging, "true"]
Parameters:
  EnableLogging:
    Type: String
    Default: "false"
  LogBucket:
    Type: String
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-bucket
      LoggingConfiguration: !If
        - HasLogging
        - DestinationBucketName: !Ref LogBucket
        - !Ref "AWS::NoValue"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "iftest")
	code := files["storage.go"]

	// Should generate If{} without & prefix for property type fields
	assert.Contains(t, code, "LoggingConfiguration: If{", "Should generate If{} for conditional property")
	assert.NotContains(t, code, "&If{", "Should not use & prefix with If{}")
}

func TestGenerateCode_NestedGetAtt(t *testing.T) {
	// Test that nested GetAtt attributes like "Endpoint.Address" use explicit GetAtt{}
	// Issue #42: Nested GetAtt attributes not supported
	content := []byte(`
Resources:
  MyDB:
    Type: AWS::RDS::DBInstance
    Properties:
      DBInstanceClass: db.t3.micro
      Engine: mysql
  MyApp:
    Type: AWS::Lambda::Function
    Properties:
      Runtime: python3.12
      Handler: index.handler
      Role: arn:aws:iam::123456789:role/lambda
      Code:
        ZipFile: |
          def handler(event, context):
              pass
      Environment:
        Variables:
          DB_HOST: !GetAtt MyDB.Endpoint.Address
          DB_PORT: !GetAtt MyDB.Endpoint.Port
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "nestedtest")
	code := files["compute.go"]

	// Should generate explicit GetAtt{} for nested attributes with quoted logical ID
	assert.Contains(t, code, `GetAtt{"MyDB", "Endpoint.Address"}`, "Should use GetAtt{} with quoted logical ID for nested attribute")
	assert.Contains(t, code, `GetAtt{"MyDB", "Endpoint.Port"}`, "Should use GetAtt{} with quoted logical ID for nested attribute")
	// Should NOT generate field access for nested attributes
	assert.NotContains(t, code, "MyDB.Endpoint.Address", "Should not use field access for nested attribute")
}

func TestGenerateCode_NoUnusedIntrinsicsImport(t *testing.T) {
	// Test that files without intrinsic types don't import intrinsics
	// Issue #43: Unused intrinsics import in some resource files
	content := []byte(`
Parameters:
  BucketName:
    Type: String
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref BucketName
  MyFunction:
    Type: AWS::Lambda::Function
    Properties:
      Runtime: python3.12
      Handler: index.handler
      Role: arn:aws:iam::123456789:role/lambda
      Code:
        ZipFile: |
          def handler(event, context):
              pass
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "importtest")
	storageCode := files["storage.go"]
	computeCode := files["compute.go"]

	// Neither file should import intrinsics - they only have resource references
	assert.NotContains(t, storageCode, "github.com/lex00/wetwire-aws-go/intrinsics", "storage.go should not import intrinsics")
	assert.NotContains(t, computeCode, "github.com/lex00/wetwire-aws-go/intrinsics", "compute.go should not import intrinsics")
}

// TestGenerateCode_ParameterInArrayField tests that Parameters used in []any fields get wrapped.
// Issue #52: Parameter types incompatible with []any fields.
func TestGenerateCode_ParameterInArrayField(t *testing.T) {
	content := []byte(`
Parameters:
  SecurityGroups:
    Type: List<AWS::EC2::SecurityGroup::Id>
  Subnets:
    Type: List<AWS::EC2::Subnet::Id>
Resources:
  MyFunction:
    Type: AWS::Lambda::Function
    Properties:
      FunctionName: test
      Runtime: python3.12
      Handler: index.handler
      Role: arn:aws:iam::123456789:role/lambda
      Code:
        ZipFile: "def handler(e,c): pass"
      VpcConfig:
        SecurityGroupIds: !Ref SecurityGroups
        SubnetIds: !Ref Subnets
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "paramarray")
	computeCode := files["compute.go"]

	// Parameters used in list fields should be wrapped in []any{}
	assert.Contains(t, computeCode, "[]any{SecurityGroups}", "SecurityGroups parameter should be wrapped in []any{}")
	assert.Contains(t, computeCode, "[]any{Subnets}", "Subnets parameter should be wrapped in []any{}")
}

// TestGenerateCode_SplitInArrayField tests that Split{} used in []any fields gets wrapped.
// Issue #52: Split intrinsic incompatible with []any fields.
func TestGenerateCode_SplitInArrayField(t *testing.T) {
	content := []byte(`
Resources:
  MyInstance:
    Type: AWS::EC2::Instance
    Properties:
      ImageId: ami-12345678
      SecurityGroupIds: !Split [",", "sg-123,sg-456"]
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "splitarray")
	computeCode := files["compute.go"]

	// Split{} used in list fields should be wrapped in []any{}
	assert.Contains(t, computeCode, "[]any{Split{", "Split{} should be wrapped in []any{}")
}

// TestGenerateCode_IfInArrayField tests that If{} used in []any fields gets wrapped.
// Issue #52: If intrinsic incompatible with []any fields.
func TestGenerateCode_IfInArrayField(t *testing.T) {
	content := []byte(`
Conditions:
  UseMultiAZ:
    !Equals [!Ref "AWS::Region", "us-east-1"]
Resources:
  MyDB:
    Type: AWS::RDS::DBInstance
    Properties:
      DBInstanceClass: db.t3.micro
      Engine: mysql
      MasterUsername: admin
      MasterUserPassword: password123
      VPCSecurityGroups:
        !If
          - UseMultiAZ
          - - sg-primary
            - sg-secondary
          - - sg-single
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "ifarray")
	databaseCode := files["database.go"]

	// If{} used in list fields should be wrapped in []any{}
	assert.Contains(t, databaseCode, "[]any{If{", "If{} should be wrapped in []any{}")
}
