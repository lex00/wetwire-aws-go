package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lex00/cfn-lint-go/pkg/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCfnLintResult_TotalIssues(t *testing.T) {
	tests := []struct {
		name     string
		result   CfnLintResult
		expected int
	}{
		{
			name:     "empty result",
			result:   CfnLintResult{},
			expected: 0,
		},
		{
			name: "errors only",
			result: CfnLintResult{
				Errors: []string{"error1", "error2"},
			},
			expected: 2,
		},
		{
			name: "warnings only",
			result: CfnLintResult{
				Warnings: []string{"warning1"},
			},
			expected: 1,
		},
		{
			name: "mixed issues",
			result: CfnLintResult{
				Errors:        []string{"error1"},
				Warnings:      []string{"warning1", "warning2"},
				Informational: []string{"info1"},
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.TotalIssues())
		})
	}
}

func TestFormatMatch(t *testing.T) {
	tests := []struct {
		name     string
		match    lint.Match
		expected string
	}{
		{
			name: "simple match",
			match: lint.Match{
				Rule:    lint.MatchRule{ID: "E1234"},
				Message: "Something is wrong",
			},
			expected: "E1234: Something is wrong",
		},
		{
			name: "match with path",
			match: lint.Match{
				Rule:    lint.MatchRule{ID: "W5678"},
				Message: "Warning message",
				Location: lint.MatchLocation{
					Path: []any{"Resources", "MyBucket", "Properties"},
				},
			},
			expected: "W5678: Warning message (at Resources/MyBucket/Properties)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMatch(tt.match)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunCfnLint_FileNotFound(t *testing.T) {
	result, err := RunCfnLint("/nonexistent/template.yaml")
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Template file not found")
}

func TestRunCfnLint_ValidTemplate(t *testing.T) {
	// Create a valid CloudFormation template
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "template.yaml")

	validTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Description: Test template
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test-bucket
`
	err := os.WriteFile(templatePath, []byte(validTemplate), 0644)
	require.NoError(t, err)

	// Now uses cfn-lint-go library - no external binary needed
	result, err := RunCfnLint(templatePath)
	require.NoError(t, err)
	// Result should parse successfully (whether or not there are warnings)
	assert.NotNil(t, result)
}

func TestLintResult_Struct(t *testing.T) {
	result := LintResult{
		Passed: true,
		Issues: []string{},
		Output: "No issues found",
	}

	assert.True(t, result.Passed)
	assert.Empty(t, result.Issues)
}

func TestBuildResult_Struct(t *testing.T) {
	result := BuildResult{
		Success:      true,
		Template:     "AWSTemplateFormatVersion: '2010-09-09'",
		TemplatePath: "/tmp/template.yaml",
	}

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Template)
	assert.NotEmpty(t, result.TemplatePath)
}

func TestValidationResult_Struct(t *testing.T) {
	result := ValidationResult{
		LintResult: &LintResult{
			Passed: true,
		},
		BuildResult: &BuildResult{
			Success: true,
		},
		CfnLintResult: &CfnLintResult{
			Passed: true,
		},
	}

	assert.True(t, result.LintResult.Passed)
	assert.True(t, result.BuildResult.Success)
	assert.True(t, result.CfnLintResult.Passed)
}

func TestLintMatch_Struct(t *testing.T) {
	// Test that we can create and use lint.Match structs from cfn-lint-go
	match := lint.Match{
		Rule: lint.MatchRule{
			ID:          "E1234",
			Description: "Test rule",
		},
		Location: lint.MatchLocation{
			Start:    lint.MatchPosition{LineNumber: 1, ColumnNumber: 1},
			End:      lint.MatchPosition{LineNumber: 1, ColumnNumber: 10},
			Path:     []any{"Resources", "MyBucket"},
			Filename: "template.yaml",
		},
		Level:   "Error",
		Message: "Test error message",
	}

	assert.Equal(t, "E1234", match.Rule.ID)
	assert.Equal(t, "Error", match.Level)
	assert.Equal(t, "Test error message", match.Message)
	assert.Equal(t, 1, match.Location.Start.LineNumber)
}
