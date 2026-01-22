package importer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{"123Invalid", "N123Invalid"},                     // Digits preserved with N prefix (exported)
		{"2RouteTableCondition", "N2RouteTableCondition"}, // Issue #76: digit prefix preserved (N for exported)
		{"with-dash", "Withdash"},                         // Capitalized for export
		{"with.dot", "Withdot"},                           // Capitalized for export
		{"type", "Type"},                                  // Capitalized, no longer a keyword
		{"package", "Package"},                            // Capitalized, no longer a keyword
		{"", "_"},
		{"myBucket", "MyBucket"}, // Lowercase capitalized
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

func TestCapitalizeService(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ec2", "Ec2"},         // Switch case: ec2 -> Ec2
		{"s3", "S3"},           // Switch case: s3 -> S3
		{"rds", "Rds"},         // Switch case: rds -> Rds
		{"ecs", "Ecs"},         // Switch case: ecs -> Ecs
		{"acm", "Acm"},         // Switch case: acm -> Acm
		{"elbv2", "Elbv2"},     // Switch case: elbv2 -> Elbv2
		{"lambda", "Lambda"},   // Default: capitalize first letter
		{"unknown", "Unknown"}, // Default: capitalize first letter
		{"", ""},               // Empty string
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
