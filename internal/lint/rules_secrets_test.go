package lint

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for WAW019 secret pattern detection and related helper functions

func TestSecretPattern_AWSAccessKey(t *testing.T) {
	// Test AWS access key detection
	src := `package test

import "github.com/lex00/wetwire-aws-go/resources/lambda_"

var MyFunction = lambda_.Function{
	Environment: map[string]any{
		"AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
	},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := SecretPattern{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Equal(t, "WAW019", issues[0].Rule)
		assert.Contains(t, issues[0].Message, "AWS access key")
		assert.Equal(t, SeverityError, issues[0].Severity)
	}
}

func TestSecretPattern_PrivateKey(t *testing.T) {
	// Test private key header detection
	src := `package test

var PrivateKey = "-----BEGIN RSA PRIVATE KEY-----"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := SecretPattern{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 1)
	if len(issues) > 0 {
		assert.Contains(t, issues[0].Message, "private key")
	}
}

func TestSecretPattern_GenericPassword(t *testing.T) {
	// Test generic password/secret patterns
	src := `package test

var Config = map[string]any{
	"Password": "mysecretpassword123",
	"ApiKey": "sk_live_abcdef1234567890",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := SecretPattern{}
	issues := rule.Check(file, fset)

	// Should detect multiple secrets
	assert.GreaterOrEqual(t, len(issues), 1)
}

func TestSecretPattern_NoFalsePositives(t *testing.T) {
	// Test that common safe patterns don't trigger
	src := `package test

var BucketName = "my-bucket-name"
var Region = "us-east-1"
var StackName = "my-stack"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := SecretPattern{}
	issues := rule.Check(file, fset)

	assert.Len(t, issues, 0)
}

func TestIsSafeString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Safe patterns
		{"arn:aws:s3:::my-bucket", true},
		{"${AWS::StackName}", true},
		{"AWS::Region", true},
		{"https://example.com", true},
		{"http://localhost", true},
		{"s3://my-bucket/key", true},
		{"ecs-tasks.amazonaws.com", true},
		{"lambda.amazonaws.com", true},
		{"logs.us-east-1.amazonaws.com", true},
		{"events.amazonaws.com", true},
		{"ecr.us-east-1.amazonaws.com", true},
		{"api.amazonaws.com", true},

		// Unsafe patterns
		{"AKIAIOSFODNN7EXAMPLE", false},
		{"some-random-string", false},
		{"my-bucket-name", false},
		{"password123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isSafeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHighEntropy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// High entropy: needs >= 3 char types AND length >= 32
		{"4 char types", "AbCdEfGh123456789!@#$%^&*()AbCdEf", true},  // 34 chars, 4 types (upper, lower, digit, special)
		{"3 char types", "test_key_1234567890abcdefghijklmnop", true}, // 38 chars, 3 types (lower, digit, special)

		// Low entropy: fails one or both criteria
		{"short string", "abc123", false},                // Too short (6 chars)
		{"only 2 types uppercase+digit", "AKIAIOSFODNN7EXAMPLE12345678901234", false}, // 34 chars but only 2 types
		{"only 2 types lower+digit", "abcdefghijklmnopqrstuvwxyz1234567890", false}, // 36 chars, only 2 types
		{"too short 8 chars", "password", false},         // Too short
		{"21 chars 2 types", "my-simple-bucket-name", false}, // 21 chars, only 2 types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHighEntropy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPlaceholder(t *testing.T) {
	// isPlaceholder lowercases the input and checks for substrings:
	// "changeme", "placeholder", "example", "your-", "my-", "todo", "fixme",
	// "<", ">", "xxx", "dummy", "test"
	tests := []struct {
		input    string
		expected bool
	}{
		// Matches placeholder patterns
		{"PLACEHOLDER", true},          // contains "placeholder"
		{"TODO", true},                 // contains "todo"
		{"FIXME", true},                // contains "fixme"
		{"your-bucket-name", true},     // contains "your-"
		{"my-access-key", true},        // contains "my-"
		{"example-value", true},        // contains "example"
		{"changeme", true},             // contains "changeme"
		{"<your-value>", true},         // contains "<"
		{"my-production-bucket", true}, // contains "my-"
		{"XXX_REPLACE_ME", true},       // contains "xxx"
		{"dummy-value", true},          // contains "dummy"
		{"test-bucket", true},          // contains "test"

		// Does NOT match (underscore not same as hyphen, etc.)
		{"CHANGE_ME", false},       // "change_me" != "changeme"
		{"YOUR_VALUE_HERE", false}, // "your_value_here" != "your-"
		{"real-config-value", false},
		{"us-east-1", false},
		{"production-bucket", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isPlaceholder(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := itoa(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMapStringAny(t *testing.T) {
	// Test that map[string]any with intrinsic key is detected
	src := `package test

var ref = map[string]any{"Ref": "MyBucket"}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	rule := MapShouldBeIntrinsic{}
	issues := rule.Check(file, fset)
	assert.Len(t, issues, 1, "Should detect map[string]any with Ref key")
}

func TestIsMapStringAny_EdgeCases(t *testing.T) {
	// Test that non-map types return false
	src := `package test

var notAMap = "just a string"
var aSlice = []int{1, 2, 3}
var anInt = 42
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	// The isMapStringAny function isn't exported, but we can test through UnflattenedMap
	rule := UnflattenedMap{}
	issues := rule.Check(file, fset)
	// No issues for non-resource code
	assert.Len(t, issues, 0)
}
