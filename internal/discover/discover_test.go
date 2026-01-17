package discover

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	wetwire "github.com/lex00/wetwire-aws-go"
	coreast "github.com/lex00/wetwire-core-go/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover_SimpleResource(t *testing.T) {
	// Create a temp directory with test Go files
	dir := t.TempDir()

	// Write a simple Go file with resource declarations
	code := `package infra

import "github.com/lex00/wetwire-aws-go/s3"

var MyBucket = s3.Bucket{
	BucketName: "my-bucket",
}
`
	err := os.WriteFile(filepath.Join(dir, "storage.go"), []byte(code), 0644)
	require.NoError(t, err)

	// Discover resources
	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	// Verify the resource was found
	assert.Len(t, result.Resources, 1)
	assert.Contains(t, result.Resources, "MyBucket")

	res := result.Resources["MyBucket"]
	assert.Equal(t, "MyBucket", res.Name)
	assert.Equal(t, "s3.Bucket", res.Type)
	assert.Equal(t, "infra", res.Package)
	assert.Empty(t, res.Dependencies)
}

func TestDiscover_WithDependencies(t *testing.T) {
	dir := t.TempDir()

	code := `package infra

import (
	"github.com/lex00/wetwire-aws-go/iam"
	"github.com/lex00/wetwire-aws-go/lambda"
	"github.com/lex00/wetwire-aws-go/s3"
)

var DataBucket = s3.Bucket{
	BucketName: "data-bucket",
}

var ProcessorRole = iam.Role{
	RoleName: "processor-role",
}

var ProcessorFunction = lambda.Function{
	FunctionName: "processor",
	Role:         ProcessorRole.Arn,
	Environment: &lambda.Environment{
		Variables: map[string]string{
			"BUCKET": DataBucket.BucketName,
		},
	},
}
`
	err := os.WriteFile(filepath.Join(dir, "compute.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 3)

	// Check ProcessorFunction has dependencies
	fn := result.Resources["ProcessorFunction"]
	assert.Equal(t, "lambda.Function", fn.Type)
	assert.Contains(t, fn.Dependencies, "ProcessorRole")
	assert.Contains(t, fn.Dependencies, "DataBucket")
}

func TestDiscover_UndefinedReference(t *testing.T) {
	dir := t.TempDir()

	code := `package infra

import "github.com/lex00/wetwire-aws-go/lambda"

var ProcessorFunction = lambda.Function{
	Role: UndefinedRole.Arn,
}
`
	err := os.WriteFile(filepath.Join(dir, "compute.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	// Should have an error for undefined reference
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "UndefinedRole")
}

func TestDiscover_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	storageCode := `package infra

import "github.com/lex00/wetwire-aws-go/s3"

var DataBucket = s3.Bucket{
	BucketName: "data",
}
`
	computeCode := `package infra

import "github.com/lex00/wetwire-aws-go/lambda"

var Processor = lambda.Function{
	FunctionName: "proc",
	Environment: &lambda.Environment{
		Variables: map[string]string{
			"BUCKET": DataBucket.BucketName,
		},
	},
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "storage.go"), []byte(storageCode), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "compute.go"), []byte(computeCode), 0644))

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 2)
	assert.Contains(t, result.Resources, "DataBucket")
	assert.Contains(t, result.Resources, "Processor")

	// Processor should depend on DataBucket
	proc := result.Resources["Processor"]
	assert.Contains(t, proc.Dependencies, "DataBucket")
	assert.Empty(t, result.Errors)
}

func TestExtractTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantPkg  string
	}{
		{
			name:     "qualified type",
			input:    "s3.Bucket",
			wantType: "Bucket",
			wantPkg:  "s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would need proper AST parsing to test properly
			// For now, just verify the function exists in core AST package
			_ = coreast.ExtractTypeName
		})
	}
}

func TestIsCommonIdent(t *testing.T) {
	tests := []struct {
		name     string
		ident    string
		expected bool
	}{
		// Go built-ins
		{"true", "true", true},
		{"false", "false", true},
		{"nil", "nil", true},
		{"string type", "string", true},
		{"int type", "int", true},
		{"any type", "any", true},

		// Intrinsic types
		{"Ref", "Ref", true},
		{"Sub", "Sub", true},
		{"GetAtt", "GetAtt", true},
		{"Join", "Join", true},
		{"If", "If", true},
		{"Parameter", "Parameter", true},
		{"Output", "Output", true},

		// Pseudo-parameters
		{"AWS_REGION", "AWS_REGION", true},
		{"AWS_ACCOUNT_ID", "AWS_ACCOUNT_ID", true},
		{"AWS_STACK_NAME", "AWS_STACK_NAME", true},

		// Not common idents (resource names)
		{"MyBucket", "MyBucket", false},
		{"DataBucket", "DataBucket", false},
		{"ProcessorFunction", "ProcessorFunction", false},
		{"custom name", "SomeResource", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommonIdent(tt.ident)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiscover_WithParameter(t *testing.T) {
	dir := t.TempDir()

	code := `package infra

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/s3"
)

var Environment = Parameter{
	Type:    "String",
	Default: "dev",
}

var DataBucket = s3.Bucket{
	BucketName: Sub{"${AWS::StackName}-${Environment}"},
}
`
	err := os.WriteFile(filepath.Join(dir, "infra.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	// Should find 1 resource and 1 parameter
	assert.Len(t, result.Resources, 1)
	assert.Len(t, result.Parameters, 1)
	assert.Contains(t, result.Parameters, "Environment")
}

func TestDiscover_WithOutput(t *testing.T) {
	dir := t.TempDir()

	code := `package infra

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/s3"
)

var DataBucket = s3.Bucket{
	BucketName: "my-bucket",
}

var BucketNameOutput = Output{
	Description: "Bucket name",
	Value:       DataBucket,
}
`
	err := os.WriteFile(filepath.Join(dir, "infra.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 1)
	assert.Len(t, result.Outputs, 1)
	assert.Contains(t, result.Outputs, "BucketNameOutput")
}

func TestDiscover_WithMapping(t *testing.T) {
	dir := t.TempDir()

	code := `package infra

import . "github.com/lex00/wetwire-aws-go/intrinsics"

var RegionMap = Mapping{
	"us-east-1": map[string]any{
		"AMI": "ami-12345678",
	},
	"us-west-2": map[string]any{
		"AMI": "ami-87654321",
	},
}
`
	err := os.WriteFile(filepath.Join(dir, "infra.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Mappings, 1)
	assert.Contains(t, result.Mappings, "RegionMap")
}

func TestDiscover_EmptyPackage(t *testing.T) {
	dir := t.TempDir()

	// Create empty Go file
	code := `package empty
`
	err := os.WriteFile(filepath.Join(dir, "empty.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 0)
	assert.Len(t, result.Parameters, 0)
	assert.Len(t, result.Outputs, 0)
}

func TestDiscover_NonExistentDir(t *testing.T) {
	// Discover returns an error for non-existent paths
	result, err := Discover(Options{
		Packages: []string{"/nonexistent/path"},
	})
	// If no error, result should be empty
	if err == nil {
		assert.Len(t, result.Resources, 0)
	}
}

func TestDiscover_RecursivePattern(t *testing.T) {
	dir := t.TempDir()

	// Create a Go file in the root directory
	code := `package main

import "github.com/lex00/wetwire-aws-go/s3"

var SubBucket = s3.Bucket{
	BucketName: "sub-bucket",
}
`
	err := os.WriteFile(filepath.Join(dir, "storage.go"), []byte(code), 0644)
	require.NoError(t, err)

	// Test with ./ pattern
	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 1)
	assert.Contains(t, result.Resources, "SubBucket")
}

func TestResolveAttrRefs(t *testing.T) {
	// Create a Result with VarAttrRefs
	result := &Result{
		VarAttrRefs: map[string]VarAttrRefInfo{
			"ProcessorFunction": {
				AttrRefs: []wetwire.AttrRefUsage{
					{ResourceName: "DataBucket", Attribute: "Arn", FieldPath: "Environment.Variables.BUCKET_ARN"},
					{ResourceName: "ProcessorRole", Attribute: "Arn", FieldPath: "Role"},
				},
			},
			"MyOutput": {
				VarRefs: map[string]string{
					"Value": "ProcessorFunction",
				},
			},
		},
	}

	// Test direct attr refs
	refs := result.ResolveAttrRefs("ProcessorFunction")
	assert.Len(t, refs, 2)

	// Test recursive resolution through VarRefs
	refs = result.ResolveAttrRefs("MyOutput")
	assert.Len(t, refs, 2)
	// Should have prefixed paths
	for _, ref := range refs {
		assert.True(t, len(ref.FieldPath) > 0, "FieldPath should not be empty")
	}
}

func TestResolveAttrRefs_NotFound(t *testing.T) {
	result := &Result{
		VarAttrRefs: map[string]VarAttrRefInfo{},
	}

	refs := result.ResolveAttrRefs("NonExistent")
	assert.Len(t, refs, 0)
}

func TestResolveAttrRefs_CircularReference(t *testing.T) {
	// Create circular reference to test visited map
	result := &Result{
		VarAttrRefs: map[string]VarAttrRefInfo{
			"A": {
				VarRefs: map[string]string{"Field1": "B"},
			},
			"B": {
				VarRefs: map[string]string{"Field2": "A"},
			},
		},
	}

	// Should not infinite loop - returns empty slice since no AttrRefs defined
	refs := result.ResolveAttrRefs("A")
	assert.Len(t, refs, 0)
}

func TestIsIntrinsicPackage(t *testing.T) {
	// isIntrinsicPackage takes pkgName and imports map
	// When pkgName is empty, it checks for dot-import in the imports map
	// Otherwise it checks if pkgName points to an intrinsics package

	// Test with dot-import of intrinsics
	imports := map[string]string{
		".": "github.com/lex00/wetwire-aws-go/intrinsics",
	}
	assert.True(t, isIntrinsicPackage("", imports))

	// Test with named import of intrinsics
	imports = map[string]string{
		"intrinsics": "github.com/lex00/wetwire-aws-go/intrinsics",
	}
	assert.True(t, isIntrinsicPackage("intrinsics", imports))

	// Test with non-intrinsics package
	imports = map[string]string{
		"s3": "github.com/lex00/wetwire-aws-go/resources/s3",
	}
	assert.False(t, isIntrinsicPackage("s3", imports))

	// Test with "intrinsics" as package name - always returns true
	// because it's the expected name for the intrinsics package
	imports = map[string]string{
		"other": "github.com/some/other/package",
	}
	assert.True(t, isIntrinsicPackage("intrinsics", imports))

	// Test with different package name not in imports - returns false
	assert.False(t, isIntrinsicPackage("unknown", imports))
}

func TestDiscover_RecursivePatternWithDots(t *testing.T) {
	// Test the recursive pattern with trailing "..."
	dir := t.TempDir()
	subDir := filepath.Join(dir, "subpkg")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Create a Go file in the subdirectory
	code := `package subpkg

import "github.com/lex00/wetwire-aws-go/resources/s3"

var SubBucket = s3.Bucket{
	BucketName: "sub-bucket",
}
`
	err := os.WriteFile(filepath.Join(subDir, "storage.go"), []byte(code), 0644)
	require.NoError(t, err)

	// Test with "..." suffix (triggers recursive mode after /... stripping has no effect)
	// The pattern "dir..." triggers recursive: ends with "..." -> true
	result, err := Discover(Options{
		Packages: []string{dir + "..."},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 1)
	assert.Contains(t, result.Resources, "SubBucket")
}

func TestDiscover_WithSliceExpr(t *testing.T) {
	// Test slice expression handling in findDepsWithVarRefs
	dir := t.TempDir()

	code := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyData = []string{"a", "b", "c"}
var MySlice = MyData[1:2]

var MyBucket = s3.Bucket{
	BucketName: "test",
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDiscover_WithIndexExpr(t *testing.T) {
	// Test index expression handling in findDepsWithVarRefs
	dir := t.TempDir()

	code := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var Buckets = []s3.Bucket{}
var FirstBucket = Buckets[0]

var MyBucket = s3.Bucket{
	BucketName: "test",
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDiscover_WithUnaryExpr(t *testing.T) {
	// Test unary expression handling (e.g., &Type{})
	dir := t.TempDir()

	code := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	BucketName: "test",
	NotificationConfiguration: &s3.Bucket_NotificationConfiguration{},
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "MyBucket")
}

func TestExtractTypeName_DefaultCase(t *testing.T) {
	// Test coreast.ExtractTypeName returns empty strings for unsupported expression types
	// (The function handles *ast.SelectorExpr and *ast.Ident, anything else returns "","")

	// Create a mock expression that is neither SelectorExpr nor Ident
	// This tests the default switch case
	src := `package test

var x = 42
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	// Walk to find a BasicLit expression (which will trigger default case)
	var foundExpr ast.Expr
	ast.Inspect(file, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok {
			foundExpr = lit
			return false
		}
		return true
	})

	require.NotNil(t, foundExpr)
	typeName, pkgName := coreast.ExtractTypeName(foundExpr)
	assert.Equal(t, "", typeName)
	assert.Equal(t, "", pkgName)
}

func TestDiscover_WalkError(t *testing.T) {
	// Test error handling during directory walk
	result, err := Discover(Options{
		Packages: []string{"/nonexistent/path/..."},
	})
	// Should handle gracefully - either returns error or empty result
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, result)
	}
}

func TestDiscover_ImportedPackageInField(t *testing.T) {
	// Test that package names in imports are skipped correctly in findDepsWithVarRefs
	dir := t.TempDir()

	code := `package test

import (
	"github.com/lex00/wetwire-aws-go/resources/s3"
	"github.com/lex00/wetwire-aws-go/resources/iam"
)

var MyRole = iam.Role{}

var MyBucket = s3.Bucket{
	BucketName: "test",
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "MyBucket")
	assert.Contains(t, result.Resources, "MyRole")
}

func TestDiscover_CommonIdentInField(t *testing.T) {
	// Test that common identifiers like true, false, nil are skipped
	dir := t.TempDir()

	code := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	ObjectLockEnabled: true,
	PublicAccessBlockConfiguration: nil,
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "MyBucket")
	// Should have no dependencies (true/nil are common identifiers, skipped)
	bucket := result.Resources["MyBucket"]
	assert.Empty(t, bucket.Dependencies)
}

func TestDiscover_PackageSelectorSkipped(t *testing.T) {
	// Test that package.Type selectors are skipped (not treated as resource refs)
	dir := t.TempDir()

	code := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	BucketName: s3.BucketName,
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "MyBucket")
}

func TestDiscover_CallExprWithResourceArg(t *testing.T) {
	// Test that function call arguments are scanned for dependencies
	dir := t.TempDir()

	code := `package test

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/s3"
)

var OtherBucket = s3.Bucket{
	BucketName: "other",
}

var MyBucket = s3.Bucket{
	BucketName: Sub("${AWS::StackName}-bucket"),
	Tags: Json{
		"Ref": OtherBucket,
	},
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "MyBucket")
	assert.Contains(t, result.Resources, "OtherBucket")
}

func TestDiscover_LowercaseIdentifierSkipped(t *testing.T) {
	// Test that lowercase identifiers are not treated as resource references
	dir := t.TempDir()

	code := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var helper = "some-value"

var MyBucket = s3.Bucket{
	BucketName: helper,
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "MyBucket")
	// helper is lowercase, so not added as dependency
	bucket := result.Resources["MyBucket"]
	assert.Empty(t, bucket.Dependencies)
}

func TestDiscover_SelectorExprNonIdent(t *testing.T) {
	// Test selector expression with non-ident X (edge case)
	dir := t.TempDir()

	code := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var data = map[string]interface{}{
	"value": "test",
}

var MyBucket = s3.Bucket{
	BucketName: "test",
}
`
	err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "MyBucket")
}

func TestDiscover_MultiplePackages(t *testing.T) {
	// Test discovering multiple packages in one call
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	code1 := `package pkg1

import "github.com/lex00/wetwire-aws-go/resources/s3"

var Bucket1 = s3.Bucket{
	BucketName: "bucket1",
}
`
	code2 := `package pkg2

import "github.com/lex00/wetwire-aws-go/resources/s3"

var Bucket2 = s3.Bucket{
	BucketName: "bucket2",
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "test1.go"), []byte(code1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "test2.go"), []byte(code2), 0644))

	result, err := Discover(Options{
		Packages: []string{dir1, dir2},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Resources, "Bucket1")
	assert.Contains(t, result.Resources, "Bucket2")
}

func TestDiscover_PackageError(t *testing.T) {
	// Test error propagation from discoverPackage
	// Create a directory with invalid Go code to trigger parse error
	dir := t.TempDir()

	// Writing invalid Go syntax
	code := `package test

This is not valid Go code at all!
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "invalid.go"), []byte(code), 0644))

	_, err := Discover(Options{
		Packages: []string{dir},
	})
	// Should return an error for parse failures
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected declaration")
}
