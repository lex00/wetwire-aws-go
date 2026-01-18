package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for LintFile, LintPackage, and related functions

func TestLintFile_CleanFile(t *testing.T) {
	result, err := LintFile("testdata/simple/clean.go", Options{})
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Len(t, result.Issues, 0)
}

func TestLintFile_WithIssues(t *testing.T) {
	result, err := LintFile("testdata/simple/with_issues.go", Options{})
	require.NoError(t, err)

	assert.False(t, result.Success)
	assert.Greater(t, len(result.Issues), 0)

	// Should detect hardcoded pseudo-parameter
	foundPseudoParam := false
	for _, issue := range result.Issues {
		if issue.Rule == "WAW001" {
			foundPseudoParam = true
			break
		}
	}
	assert.True(t, foundPseudoParam, "Should detect hardcoded pseudo-parameter")
}

func TestLintFile_NonExistentFile(t *testing.T) {
	_, err := LintFile("testdata/nonexistent.go", Options{})
	assert.Error(t, err)
}

func TestLintPackage_Directory(t *testing.T) {
	result, err := LintPackage("testdata/simple", Options{})
	require.NoError(t, err)

	// Should have issues from with_issues.go
	assert.Greater(t, len(result.Issues), 0)
}

func TestLintPackage_NonExistentDir(t *testing.T) {
	_, err := LintPackage("testdata/nonexistent", Options{})
	assert.Error(t, err)
}

func TestLintPackage_RecursivePattern(t *testing.T) {
	result, err := LintPackage("testdata/...", Options{})
	require.NoError(t, err)

	// Should find issues in simple directory
	assert.Greater(t, len(result.Issues), 0)
}

func TestGetRules_AllRules(t *testing.T) {
	rules := getRules(Options{})
	assert.GreaterOrEqual(t, len(rules), 15)
}

func TestGetRules_FilteredRules(t *testing.T) {
	rules := getRules(Options{
		EnabledRules: []string{"WAW001", "WAW002"},
	})
	assert.Len(t, rules, 2)

	ruleIDs := []string{}
	for _, r := range rules {
		ruleIDs = append(ruleIDs, r.ID())
	}
	assert.Contains(t, ruleIDs, "WAW001")
	assert.Contains(t, ruleIDs, "WAW002")
}

func TestGetRules_WithMaxResources(t *testing.T) {
	rules := getRules(Options{
		MaxResources: 25,
	})

	// Find FileTooLarge rule
	var ftl FileTooLarge
	for _, r := range rules {
		if f, ok := r.(FileTooLarge); ok {
			ftl = f
			break
		}
	}

	assert.Equal(t, 25, ftl.MaxResources)
}

func TestLintRecursive_Directory(t *testing.T) {
	// Test linting a directory recursively
	issues, err := LintPackage("./testdata/...", Options{})
	require.NoError(t, err)
	// The testdata directory should exist and have some test files
	assert.NotNil(t, issues)
}

func TestLintFile_WithSubdirectory(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subpkg")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Write a go file in the subdirectory
	code := `package subpkg

import "github.com/lex00/wetwire-aws-go/resources/s3"

var SubBucket = s3.Bucket{
	BucketName: "sub-bucket",
}
`
	filePath := filepath.Join(subDir, "sub.go")
	require.NoError(t, os.WriteFile(filePath, []byte(code), 0644))

	// Run linter on the file
	result, err := LintFile(filePath, Options{})
	require.NoError(t, err)
	// Should successfully lint file in subdirectory
	assert.NotNil(t, result)
}

func TestGetRules_DisabledRules(t *testing.T) {
	// Get all rules first
	allRules := getRules(Options{})
	allRuleCount := len(allRules)

	// Disable WAW001 and WAW002
	rules := getRules(Options{
		DisabledRules: []string{"WAW001", "WAW002"},
	})

	// Should have 2 fewer rules
	assert.Len(t, rules, allRuleCount-2)

	// WAW001 and WAW002 should not be in the returned rules
	ruleIDs := []string{}
	for _, r := range rules {
		ruleIDs = append(ruleIDs, r.ID())
	}
	assert.NotContains(t, ruleIDs, "WAW001")
	assert.NotContains(t, ruleIDs, "WAW002")
}

func TestGetRules_DisabledOverridesEnabled(t *testing.T) {
	// Enable WAW001 and WAW002, but also disable WAW001
	rules := getRules(Options{
		EnabledRules:  []string{"WAW001", "WAW002"},
		DisabledRules: []string{"WAW001"},
	})

	// Should only have WAW002
	assert.Len(t, rules, 1)

	ruleIDs := []string{}
	for _, r := range rules {
		ruleIDs = append(ruleIDs, r.ID())
	}
	assert.NotContains(t, ruleIDs, "WAW001")
	assert.Contains(t, ruleIDs, "WAW002")
}

func TestLintFile_WithDisabledRules(t *testing.T) {
	// Test that disabled rules are not applied
	result, err := LintFile("testdata/simple/with_issues.go", Options{
		DisabledRules: []string{"WAW001"},
	})
	require.NoError(t, err)

	// WAW001 issues should not appear
	for _, issue := range result.Issues {
		assert.NotEqual(t, "WAW001", issue.Rule, "WAW001 issues should be disabled")
	}
}

func TestOptions_FixField(t *testing.T) {
	// Test that Fix field is respected (currently no-op but should not error)
	opts := Options{
		Fix: true,
	}

	rules := getRules(opts)
	assert.NotEmpty(t, rules)
}
