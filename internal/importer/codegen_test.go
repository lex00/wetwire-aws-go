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

	// Check Sub usage (no intrinsics. prefix with dot import)
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

	// Used parameters ARE generated as typed vars using Param()
	assert.Contains(t, paramsCode, "// Environment - Environment name")
	assert.Contains(t, paramsCode, `var Environment = Param("Environment")`)

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
		{"empty map", map[string]any{}, "map[string]any{}"},
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
		{"123Invalid", "_23Invalid"},
		{"with-dash", "withdash"},
		{"with.dot", "withdot"},
		{"type", "type_"},       // Go keyword
		{"package", "package_"}, // Go keyword
		{"", "_"},
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
	// Should have sanitized name
	assert.Contains(t, code, "PortNeg1", "Should use Neg prefix for negative numbers")
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
			// Verify there's actual usage (Sub, Ref, etc.)
			hasUsage := strings.Contains(code, "Sub{") ||
				strings.Contains(code, "Ref{") ||
				strings.Contains(code, "AWS_")
			assert.True(t, hasUsage, "If intrinsics is imported, it should be used")
		}
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
