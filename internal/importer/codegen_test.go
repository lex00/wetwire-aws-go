package importer

import (
	"fmt"
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

func TestGenerateCode_WithIntrinsics(t *testing.T) {
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
      AvailabilityZone: !Select [0, !GetAZs ""]
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "vpc")
	// VPC and Subnet go to network.go (EC2 network resources)
	code := files["network.go"]

	// Check imports - uses dot import for intrinsics
	assert.Contains(t, code, `"github.com/lex00/wetwire-aws-go/resources/ec2"`)
	assert.Contains(t, code, `. "github.com/lex00/wetwire-aws-go/intrinsics"`)

	// Check VPC
	assert.Contains(t, code, "var MyVPC = ec2.VPC{")

	// Check Subnet with Ref - bare resource name pattern
	assert.Contains(t, code, "var MySubnet = ec2.Subnet{")
	// VpcId should reference MyVPC directly (no-parens pattern)
	assert.Contains(t, code, "VpcId: MyVPC,")
}

func TestGenerateCode_WithPseudoParameters(t *testing.T) {
	content := []byte(`
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub "${AWS::StackName}-bucket-${AWS::Region}"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "stack")
	// S3 bucket goes to storage.go
	code := files["storage.go"]

	// Check intrinsics import (dot import)
	assert.Contains(t, code, `. "github.com/lex00/wetwire-aws-go/intrinsics"`)

	// Check Sub usage with keyed syntax (no intrinsics. prefix with dot import)
	assert.Contains(t, code, `Sub{String: "${AWS::StackName}-bucket-${AWS::Region}"}`)
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

func TestTopologicalSort(t *testing.T) {
	template := NewIRTemplate()

	// Create resources with dependencies
	template.Resources["C"] = &IRResource{LogicalID: "C"}
	template.Resources["B"] = &IRResource{LogicalID: "B"}
	template.Resources["A"] = &IRResource{LogicalID: "A"}

	// A depends on B, B depends on C
	template.ReferenceGraph["A"] = []string{"B"}
	template.ReferenceGraph["B"] = []string{"C"}

	sorted := topologicalSort(template)

	// C should come before B, B should come before A
	indexC := indexOf(sorted, "C")
	indexB := indexOf(sorted, "B")
	indexA := indexOf(sorted, "A")

	assert.Less(t, indexC, indexB, "C should come before B")
	assert.Less(t, indexB, indexA, "B should come before A")
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func TestResolveResourceType(t *testing.T) {
	tests := []struct {
		cfType     string
		wantModule string
		wantType   string
	}{
		{"AWS::S3::Bucket", "s3", "Bucket"},
		{"AWS::EC2::VPC", "ec2", "VPC"},
		{"AWS::Lambda::Function", "lambda", "Function"},
		{"AWS::IAM::Role", "iam", "Role"},
		{"AWS::CloudFormation::Stack", "cloudformation", "Stack"},
		{"Invalid::Type", "", ""},
		{"AWS::TooFew", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.cfType, func(t *testing.T) {
			module, typeName := resolveResourceType(tt.cfType)
			assert.Equal(t, tt.wantModule, module)
			assert.Equal(t, tt.wantType, typeName)
		})
	}
}

func TestValueToGo(t *testing.T) {
	ctx := &codegenContext{
		template: NewIRTemplate(),
		imports:  make(map[string]bool),
	}

	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"nil", nil, "nil"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"string", "hello", `"hello"`},
		{"empty slice", []any{}, "[]any{}"},
		{"empty map", map[string]any{}, "Json{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valueToGo(ctx, tt.value, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeGoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ValidName", "ValidName"},
		{"123Invalid", "N123Invalid"},             // Digits preserved with N prefix (exported)
		{"2RouteTableCondition", "N2RouteTableCondition"}, // Issue #76: digit prefix preserved (N for exported)
		{"with-dash", "Withdash"},                 // Capitalized for export
		{"with.dot", "Withdot"},                   // Capitalized for export
		{"type", "Type"},                          // Capitalized, no longer a keyword
		{"package", "Package"},                    // Capitalized, no longer a keyword
		{"", "_"},
		{"myBucket", "MyBucket"},                  // Lowercase capitalized
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeGoName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"BucketName", "bucket_name"},
		{"VPCId", "v_p_c_id"},
		{"Simple", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bucket_name", "BucketName"},
		{"simple", "Simple"},
		{"already-pascal", "AlreadyPascal"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCleanForVarName_NegativeNumbers tests that negative numbers are properly sanitized.
// Bug: Port-1 becomes Port-1ICMP which is an invalid Go identifier.
func TestCleanForVarName_NegativeNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-1", "Neg1"},
		{"22", "N22"},
		{"443", "N443"},
		{"-1ICMP", "Neg1ICMP"},
		{"port-80", "PortNeg80"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanForVarName(tt.input)
			assert.Equal(t, tt.expected, result)
			// Verify it's a valid Go identifier
			assert.True(t, isValidGoIdentifier(result), "Result should be valid Go identifier: %s", result)
		})
	}
}

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

func TestMapToIntrinsic_Ref(t *testing.T) {
	m := map[string]any{"Ref": "MyResource"}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicRef, result.Type)
	assert.Equal(t, "MyResource", result.Args)
}

func TestMapToIntrinsic_GetAtt(t *testing.T) {
	m := map[string]any{"Fn::GetAtt": []string{"MyResource", "Arn"}}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicGetAtt, result.Type)
}

func TestMapToIntrinsic_Sub(t *testing.T) {
	m := map[string]any{"Fn::Sub": "${AWS::StackName}-bucket"}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicSub, result.Type)
}

func TestMapToIntrinsic_Join(t *testing.T) {
	m := map[string]any{"Fn::Join": []any{"-", []any{"a", "b", "c"}}}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicJoin, result.Type)
}

func TestMapToIntrinsic_AllTypes(t *testing.T) {
	tests := []struct {
		key      string
		expected IntrinsicType
	}{
		{"Ref", IntrinsicRef},
		{"Fn::GetAtt", IntrinsicGetAtt},
		{"Fn::Sub", IntrinsicSub},
		{"Fn::Join", IntrinsicJoin},
		{"Fn::Select", IntrinsicSelect},
		{"Fn::GetAZs", IntrinsicGetAZs},
		{"Fn::If", IntrinsicIf},
		{"Fn::Equals", IntrinsicEquals},
		{"Fn::And", IntrinsicAnd},
		{"Fn::Or", IntrinsicOr},
		{"Fn::Not", IntrinsicNot},
		{"Fn::Base64", IntrinsicBase64},
		{"Fn::FindInMap", IntrinsicFindInMap},
		{"Fn::Cidr", IntrinsicCidr},
		{"Fn::ImportValue", IntrinsicImportValue},
		{"Fn::Split", IntrinsicSplit},
		{"Fn::Transform", IntrinsicTransform},
		{"Condition", IntrinsicCondition},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			m := map[string]any{tt.key: "value"}
			result := mapToIntrinsic(m)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result.Type)
		})
	}
}

func TestMapToIntrinsic_EmptyMap(t *testing.T) {
	m := map[string]any{}
	result := mapToIntrinsic(m)
	assert.Nil(t, result)
}

func TestMapToIntrinsic_MultipleKeys(t *testing.T) {
	m := map[string]any{"Ref": "A", "Fn::Sub": "B"}
	result := mapToIntrinsic(m)
	assert.Nil(t, result)
}

func TestMapToIntrinsic_UnknownKey(t *testing.T) {
	m := map[string]any{"Unknown::Function": "value"}
	result := mapToIntrinsic(m)
	assert.Nil(t, result)
}

func TestPseudoParameterToGo(t *testing.T) {
	ctx := &codegenContext{
		template: NewIRTemplate(),
		imports:  make(map[string]bool),
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"AWS::Region", "AWS_REGION"},
		{"AWS::AccountId", "AWS_ACCOUNT_ID"},
		{"AWS::StackName", "AWS_STACK_NAME"},
		{"AWS::StackId", "AWS_STACK_ID"},
		{"AWS::Partition", "AWS_PARTITION"},
		{"AWS::URLSuffix", "AWS_URL_SUFFIX"},
		{"AWS::NoValue", "AWS_NO_VALUE"},
		{"AWS::NotificationARNs", "AWS_NOTIFICATION_ARNS"},
		{"UnknownPseudo", "UnknownPseudo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pseudoParameterToGo(ctx, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapitalizeService(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ec2", "Ec2"},  // Switch case: ec2 -> Ec2
		{"s3", "S3"},    // Switch case: s3 -> S3
		{"rds", "Rds"},  // Switch case: rds -> Rds
		{"ecs", "Ecs"},  // Switch case: ecs -> Ecs
		{"acm", "Acm"},  // Switch case: acm -> Acm
		{"elbv2", "Elbv2"}, // Switch case: elbv2 -> Elbv2
		{"lambda", "Lambda"}, // Default: capitalize first letter
		{"unknown", "Unknown"}, // Default: capitalize first letter
		{"", ""},        // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capitalizeService(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimplifySubString(t *testing.T) {
	ctx := &codegenContext{
		template: NewIRTemplate(),
		imports:  make(map[string]bool),
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"pseudo param region", "${AWS::Region}", "AWS_REGION"},
		{"pseudo param account", "${AWS::AccountId}", "AWS_ACCOUNT_ID"},
		{"pseudo param stack", "${AWS::StackName}", "AWS_STACK_NAME"},
		{"complex sub", "${AWS::StackName}-bucket", `Sub{String: "${AWS::StackName}-bucket"}`},
		{"no variables", "plain-text", `Sub{String: "plain-text"}`},
		{"nested braces", "${foo${bar}}", `Sub{String: "${foo${bar}}"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := simplifySubString(ctx, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimplifySubString_ResourceRef(t *testing.T) {
	ctx := &codegenContext{
		template: NewIRTemplate(),
		imports:  make(map[string]bool),
	}
	ctx.template.Resources["MyBucket"] = &IRResource{LogicalID: "MyBucket"}

	// Single resource reference should simplify
	result := simplifySubString(ctx, "${MyBucket}")
	assert.Equal(t, "MyBucket", result)
}

func TestSimplifySubString_ResourceAttr(t *testing.T) {
	ctx := &codegenContext{
		template: NewIRTemplate(),
		imports:  make(map[string]bool),
	}
	ctx.template.Resources["MyBucket"] = &IRResource{LogicalID: "MyBucket"}

	// Resource.Attr pattern should simplify to field access
	result := simplifySubString(ctx, "${MyBucket.Arn}")
	assert.Equal(t, "MyBucket.Arn", result)
}

func TestIntrinsicNeedsArrayWrapping(t *testing.T) {
	tests := []struct {
		intrinsic IntrinsicType
		expected  bool
	}{
		{IntrinsicGetAZs, true},
		{IntrinsicSplit, true},
		{IntrinsicRef, true},
		{IntrinsicIf, true},
		{IntrinsicSub, false},
		{IntrinsicJoin, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("type_%d", tt.intrinsic), func(t *testing.T) {
			ir := &IRIntrinsic{Type: tt.intrinsic}
			result := intrinsicNeedsArrayWrapping(ir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsListTypeParameter(t *testing.T) {
	tests := []struct {
		paramType string
		expected  bool
	}{
		{"List<Number>", true},
		{"List<String>", true},
		{"CommaDelimitedList", true},
		{"AWS::SSM::Parameter::Value<List<String>>", true},
		{"String", false},
		{"Number", false},
	}

	for _, tt := range tests {
		t.Run(tt.paramType, func(t *testing.T) {
			param := &IRParameter{Type: tt.paramType}
			result := isListTypeParameter(param)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsEC2NetworkType(t *testing.T) {
	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"VPC", true},
		{"Subnet", true},
		{"SecurityGroup", true},
		{"RouteTable", true},
		{"InternetGateway", true},
		{"NatGateway", true},
		{"Instance", false},
		{"LaunchTemplate", false},
		{"Unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := isEC2NetworkType(tt.resourceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPropertyTypeName_EmptyCases(t *testing.T) {
	// Test early return conditions
	ctx := &codegenContext{
		template:        NewIRTemplate(),
		imports:         make(map[string]bool),
		currentResource: "AWS::Lambda::Function",
		currentTypeName: "", // Empty type name
	}

	// Empty currentTypeName returns empty
	result := getPropertyTypeName(ctx, "SomeProperty")
	assert.Equal(t, "", result)

	// Empty propName returns empty
	ctx.currentTypeName = "Function"
	result = getPropertyTypeName(ctx, "")
	assert.Equal(t, "", result)

	// Tags are skipped
	result = getPropertyTypeName(ctx, "Tags")
	assert.Equal(t, "", result)

	// Metadata is skipped
	result = getPropertyTypeName(ctx, "Metadata")
	assert.Equal(t, "", result)
}

func TestAllKeysValidIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		expected bool
	}{
		{"valid keys", map[string]any{"Key1": 1, "Key2": 2}, true},
		{"empty map", map[string]any{}, true},
		{"key with space", map[string]any{"key with space": 1}, false},
		{"key with dash", map[string]any{"key-dash": 1}, false},
		{"numeric key", map[string]any{"123": 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allKeysValidIdentifiers(tt.m)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidGoIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"validName", true},
		{"Valid123", true},
		{"_underscore", true},
		{"", false},
		{"123start", false},
		{"with-dash", false},
		{"with.dot", false},
		{"with space", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isValidGoIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Policies", "Policy"},
		{"Properties", "Property"},
		{"Entries", "Entry"},
		{"Rules", "Rule"},
		{"Tags", "Tag"},
		{"Items", "Item"},
		{"Singular", "Singular"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := singularize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetArrayElementTypeName_EmptyCases(t *testing.T) {
	// Test early return cases
	ctx := &codegenContext{
		template:        NewIRTemplate(),
		imports:         make(map[string]bool),
		currentResource: "AWS::EC2::SecurityGroup",
		currentTypeName: "", // Empty type name
	}

	// Empty currentTypeName returns empty
	result := getArrayElementTypeName(ctx, "SomeProperty")
	assert.Equal(t, "", result)

	// Empty prop name also returns empty
	ctx.currentTypeName = "SecurityGroup"
	result = getArrayElementTypeName(ctx, "")
	assert.Equal(t, "", result)
}

func TestIRResource_Service(t *testing.T) {
	tests := []struct {
		cfType   string
		expected string
	}{
		{"AWS::S3::Bucket", "S3"},
		{"AWS::Lambda::Function", "Lambda"},
		{"AWS::EC2::VPC", "EC2"},
		{"AWS::Serverless::Function", "Serverless"},
		{"Custom::MyResource", "MyResource"},
		{"Invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.cfType, func(t *testing.T) {
			res := &IRResource{ResourceType: tt.cfType}
			assert.Equal(t, tt.expected, res.Service())
		})
	}
}

func TestIRResource_TypeName(t *testing.T) {
	tests := []struct {
		cfType   string
		expected string
	}{
		{"AWS::S3::Bucket", "Bucket"},
		{"AWS::Lambda::Function", "Function"},
		{"AWS::EC2::VPC", "VPC"},
		{"Custom::MyResource", ""}, // Custom resources have no third part
		{"Invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.cfType, func(t *testing.T) {
			res := &IRResource{ResourceType: tt.cfType}
			assert.Equal(t, tt.expected, res.TypeName())
		})
	}
}
