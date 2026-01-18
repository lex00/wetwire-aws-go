package domain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildOpts_Fields tests that BuildOpts fields are correctly defined
func TestBuildOpts_Fields(t *testing.T) {
	opts := BuildOpts{
		Format: "json",
		Type:   "s3.Bucket",
		Output: "/path/to/output.json",
		DryRun: true,
	}

	assert.Equal(t, "json", opts.Format)
	assert.Equal(t, "s3.Bucket", opts.Type)
	assert.Equal(t, "/path/to/output.json", opts.Output)
	assert.True(t, opts.DryRun)
}

// TestLintOpts_Fields tests that LintOpts fields are correctly defined
func TestLintOpts_Fields(t *testing.T) {
	opts := LintOpts{
		Format:  "text",
		Fix:     true,
		Disable: []string{"WAW001", "WAW002"},
	}

	assert.Equal(t, "text", opts.Format)
	assert.True(t, opts.Fix)
	assert.Equal(t, []string{"WAW001", "WAW002"}, opts.Disable)
}

func TestAwsLinter_Lint_Disable(t *testing.T) {
	// Create a temp directory with a Go file that has lint issues
	tmpDir := t.TempDir()

	// This code has a hardcoded pseudo-parameter (WAW001)
	code := `package main

import "github.com/lex00/wetwire-aws-go/resources/s3"

var TestBucket = s3.Bucket{
	BucketName: "AWS::Region",
}
`
	filePath := filepath.Join(tmpDir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

	linter := &awsLinter{}
	ctx := &Context{}

	// Test without disabling any rules - should find WAW001
	result, err := linter.Lint(ctx, tmpDir, LintOpts{})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have lint issues
	foundWAW001 := false
	for _, e := range result.Errors {
		if e.Code == "WAW001" {
			foundWAW001 = true
			break
		}
	}
	assert.True(t, foundWAW001, "Should find WAW001 issue when not disabled")

	// Test with WAW001 disabled - should NOT find WAW001
	result, err = linter.Lint(ctx, tmpDir, LintOpts{
		Disable: []string{"WAW001"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should not have WAW001 issues
	for _, e := range result.Errors {
		assert.NotEqual(t, "WAW001", e.Code, "WAW001 should be disabled")
	}
}

func TestAwsLinter_Lint_Fix(t *testing.T) {
	// Create a temp directory with a Go file that has lint issues
	tmpDir := t.TempDir()

	code := `package main

import "github.com/lex00/wetwire-aws-go/resources/s3"

var TestBucket = s3.Bucket{
	BucketName: "AWS::Region",
}
`
	filePath := filepath.Join(tmpDir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

	linter := &awsLinter{}
	ctx := &Context{}

	// Test with Fix=true - should include "auto-fix" in message
	result, err := linter.Lint(ctx, tmpDir, LintOpts{
		Fix: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// If there are issues, the message should mention auto-fix
	if len(result.Errors) > 0 {
		assert.Contains(t, result.Message, "auto-fix")
	}
}
