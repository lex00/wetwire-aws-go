package linter

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for WAW015-WAW018 reference and type rules

func TestAvoidExplicitRef(t *testing.T) {
	src := `package test

import . "github.com/lex00/wetwire-aws-go/intrinsics"

var BucketRef = Ref{"MyBucket"}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := AvoidExplicitRef{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Equal(t, "WAW015", issues[0].RuleID)
		assert.Contains(t, issues[0].Message, "Ref{}")
	}
}

func TestAvoidExplicitRef_DirectReference(t *testing.T) {
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyBucket = s3.Bucket{BucketName: "bucket"}
var OtherBucket = s3.Bucket{BucketName: MyBucket.Arn}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := AvoidExplicitRef{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 0)
}

func TestAvoidExplicitGetAtt(t *testing.T) {
	src := `package test

import . "github.com/lex00/wetwire-aws-go/intrinsics"

var RoleArn = GetAtt{"MyRole", "Arn"}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := AvoidExplicitGetAtt{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Equal(t, "WAW016", issues[0].RuleID)
		assert.Contains(t, issues[0].Message, "GetAtt{}")
	}
}

func TestAvoidExplicitGetAtt_FieldAccess(t *testing.T) {
	src := `package test

import (
	"github.com/lex00/wetwire-aws-go/resources/iam"
	"github.com/lex00/wetwire-aws-go/resources/lambda_"
)

var MyRole = iam.Role{RoleName: "role"}
var MyFunction = lambda_.Function{Role: MyRole.Arn}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := AvoidExplicitGetAtt{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 0)
}

func TestAvoidPointerAssignment(t *testing.T) {
	// Test pointer assignments are detected
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyConfig = &s3.Bucket_VersioningConfiguration{
	Status: "Enabled",
}

var MyEncryption = &s3.Bucket_BucketEncryption{
	ServerSideEncryptionConfiguration: nil,
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := AvoidPointerAssignment{}
	issues := rule.Check(file, fset)

	// Should detect 2 pointer assignments
	assert.Len(t, issues, 2)
	if len(issues) > 0 {
		assert.Equal(t, "WAW017", issues[0].RuleID)
		assert.Contains(t, issues[0].Message, "MyConfig")
		assert.Contains(t, issues[0].Message, "Bucket_VersioningConfiguration")
		assert.Equal(t, "error", issues[0].Severity)
	}
	if len(issues) > 1 {
		assert.Contains(t, issues[1].Message, "MyEncryption")
	}
}

func TestAvoidPointerAssignment_ValidValueType(t *testing.T) {
	// Test that value types don't trigger
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/s3"

var MyConfig = s3.Bucket_VersioningConfiguration{
	Status: "Enabled",
}

var MyBucket = s3.Bucket{
	BucketName: "my-bucket",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := AvoidPointerAssignment{}
	issues := rule.Check(file, fset)

	// Should not detect any issues - using value types
	assert.Len(t, issues, 0)
}

func TestPreferJsonType(t *testing.T) {
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/lambda_"

var MyFunction = lambda_.Function{
	Environment: lambda_.Function_Environment{
		Variables: map[string]any{
			"TABLE_NAME": "my-table",
		},
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := PreferJsonType{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Equal(t, "WAW018", issues[0].RuleID)
		assert.Contains(t, issues[0].Message, "Json{}")
	}
}

func TestPreferJsonType_WithJson(t *testing.T) {
	src := `package test

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/lambda_"
)

var MyFunction = lambda_.Function{
	Environment: lambda_.Function_Environment{
		Variables: Json{
			"TABLE_NAME": "my-table",
		},
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := PreferJsonType{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 0)
}
