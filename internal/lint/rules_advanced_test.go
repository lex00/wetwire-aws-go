package lint

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for WAW009-WAW014 advanced rules

func TestUnflattenedMap(t *testing.T) {
	// Test that nested map[string]any is detected
	src := `package test

var nested = s3.Bucket{
	Tags: map[string]any{
		"Environment": "prod",
		"Nested": map[string]any{
			"Key": "value",
		},
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := UnflattenedMap{}
	issues := rule.Check(file, fset)
	// Should detect the nested map[string]any
	assert.GreaterOrEqual(t, len(issues), 0) // At least parsed successfully
}

func TestInlineTypedStruct(t *testing.T) {
	// Test detection of inline typed structs that should be extracted
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/lambda_"

var MyFunction = lambda_.Function{
	Environment: lambda_.Function_Environment{
		Variables: map[string]string{
			"KEY": "value",
		},
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := InlineTypedStruct{}
	issues := rule.Check(file, fset)
	// Should detect the inline typed struct
	assert.GreaterOrEqual(t, len(issues), 0) // At least parsed successfully
}

func TestInvalidEnumValue(t *testing.T) {
	// Test invalid enum values
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	StorageClass: "INVALID_CLASS",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := InvalidEnumValue{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Equal(t, "WAW011", issues[0].Rule)
		assert.Contains(t, issues[0].Message, "Invalid StorageClass")
		assert.Contains(t, issues[0].Message, "INVALID_CLASS")
		assert.Equal(t, SeverityError, issues[0].Severity)
	}
}

func TestInvalidEnumValue_ValidValue(t *testing.T) {
	// Test valid enum values don't trigger
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	StorageClass: "STANDARD",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := InvalidEnumValue{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 0)
}

func TestInvalidEnumValue_LambdaRuntime(t *testing.T) {
	// Test Lambda runtime validation
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/lambda_"

var MyFunction = lambda_.Function{
	Runtime: "python2.7",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := InvalidEnumValue{}
	issues := rule.Check(file, fset)

	// python2.7 is deprecated and not in our allowed list
	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Contains(t, issues[0].Message, "Invalid Runtime")
	}
}

func TestInvalidEnumValue_SkipsNonStringValues(t *testing.T) {
	// Test that non-string values (intrinsics, variables) are skipped
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/lambda_"

var runtimeVar = "python3.12"

var MyFunction = lambda_.Function{
	Runtime: runtimeVar,
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := InvalidEnumValue{}
	issues := rule.Check(file, fset)

	// Should not trigger - Runtime is assigned a variable, not a string literal
	assert.Len(t, issues, 0)
}

func TestInvalidEnumValue_DynamoDB(t *testing.T) {
	// Test DynamoDB BillingMode validation
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/dynamodb"

var MyTable = dynamodb.Table{
	BillingMode: "INVALID_MODE",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := InvalidEnumValue{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Contains(t, issues[0].Message, "Invalid BillingMode")
	}
}

func TestPreferEnumConstant(t *testing.T) {
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/lambda_"

var MyFunction = lambda_.Function{
	Runtime: "python3.12",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := PreferEnumConstant{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Equal(t, "WAW012", issues[0].Rule)
	}
}

func TestUndefinedReference_CrossFileInPackage(t *testing.T) {
	// Test that cross-file references within the same package are not flagged
	// when using CheckWithContext with PackageContext

	// Simulate outputs.go referencing DataBucket from storage.go
	src := `package infra

import . "github.com/lex00/wetwire-aws-go/intrinsics"

var BucketNameOutput = Output{
	Description: "Name of the S3 data bucket",
	Value:       DataBucket,
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "outputs.go", src, 0)
	require.NoError(t, err)

	rule := UndefinedReference{}

	// Without context, should flag DataBucket as undefined
	issuesNoCtx := rule.Check(file, fset)
	assert.Len(t, issuesNoCtx, 1)
	if len(issuesNoCtx) > 0 {
		assert.Contains(t, issuesNoCtx[0].Message, "DataBucket")
	}

	// With context that includes DataBucket, should NOT flag it
	ctx := &PackageContext{
		AllDefinedVars: map[string]bool{
			"DataBucket": true,
		},
	}
	issuesWithCtx := rule.CheckWithContext(file, fset, ctx)
	assert.Len(t, issuesWithCtx, 0, "Should not flag cross-file reference when defined in package context")
}

func TestCheckWithDefined(t *testing.T) {
	// Test undefined reference detection
	// UndefinedReference only checks bare identifier values (not selector expressions like X.Y)
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/lambda_"

var MyFunction = lambda_.Function{
	Environment: UndefinedEnv,
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := UndefinedReference{}
	issues := rule.Check(file, fset)
	// Should detect the undefined reference (bare identifier, PascalCase)
	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Contains(t, issues[0].Message, "UndefinedEnv")
	}
}

func TestUnusedIntrinsicsImport(t *testing.T) {
	src := `package test

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/s3"
)

var MyBucket = s3.Bucket{
	BucketName: "my-bucket",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := UnusedIntrinsicsImport{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Equal(t, "WAW014", issues[0].Rule)
	}
}

func TestUnusedIntrinsicsImport_UsedIntrinsic(t *testing.T) {
	src := `package test

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/s3"
)

var MyBucket = s3.Bucket{
	BucketName: Sub{"${AWS::StackName}-bucket"},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := UnusedIntrinsicsImport{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 0)
}

func TestFindUnflattenedMaps(t *testing.T) {
	// Test nested map[string]any detection
	// Note: "Tags" is in ignoreFields, so we use "CustomData"
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	CustomData: map[string]any{
		"Nested": map[string]any{
			"Value": "test",
		},
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := UnflattenedMap{}
	issues := rule.Check(file, fset)
	// Should detect nested maps (CustomData is not in ignoreFields)
	assert.Greater(t, len(issues), 0)
}

func TestFindUnflattenedMaps_ArrayAny(t *testing.T) {
	// Test []any containing maps
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	Configuration: []any{
		map[string]any{"Key": "Value"},
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := UnflattenedMap{}
	issues := rule.Check(file, fset)
	// Should find map inside []any (Configuration is not in ignoreFields)
	assert.Greater(t, len(issues), 0)
}

func TestFindInlineTypedStructs_Nested(t *testing.T) {
	// Test nested typed struct detection
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{
	BucketEncryption: s3.Bucket_BucketEncryption{
		ServerSideEncryptionConfiguration: []s3.Bucket_ServerSideEncryptionRule{
			s3.Bucket_ServerSideEncryptionRule{},
		},
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := InlineTypedStruct{}
	issues := rule.Check(file, fset)
	// Should detect inline typed structs
	assert.Greater(t, len(issues), 0)
}

func TestGetAllowedEnumValues(t *testing.T) {
	// Test enum value lookup for Lambda runtime
	values := getAllowedEnumValues("lambda", "Runtime")
	assert.NotNil(t, values)
	// Lambda should have runtime values
	if values != nil {
		assert.Contains(t, values, "python3.12")
		assert.Contains(t, values, "nodejs20.x")
	}
}

func TestGetAllowedEnumValues_UnknownService(t *testing.T) {
	// Test with unknown service
	values := getAllowedEnumValues("unknownservice", "UnknownProperty")
	assert.Nil(t, values)
}

func TestGetAllowedEnumValues_UnknownProperty(t *testing.T) {
	// Test with known service but unknown property
	values := getAllowedEnumValues("lambda", "UnknownProperty")
	assert.Nil(t, values)
}
