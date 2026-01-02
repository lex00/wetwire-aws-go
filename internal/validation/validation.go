// Package validation provides functions to run wetwire-aws and cfn-lint validation.
//
// This package validates infrastructure code:
//   - wetwire-aws lint: Check code style and patterns (shells out to CLI)
//   - wetwire-aws build: Generate CloudFormation templates (shells out to CLI)
//   - cfn-lint-go: Validate CloudFormation templates (library dependency)
package validation

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lex00/cfn-lint-go/pkg/lint"
)

// LintResult contains the result of running wetwire-aws lint.
type LintResult struct {
	Passed bool     `json:"passed"`
	Issues []string `json:"issues"`
	Output string   `json:"output"`
}

// BuildResult contains the result of running wetwire-aws build.
type BuildResult struct {
	Success      bool   `json:"success"`
	Template     string `json:"template"`
	TemplatePath string `json:"template_path,omitempty"`
	Error        string `json:"error,omitempty"`
}

// CfnLintIssue represents a single cfn-lint issue.
type CfnLintIssue struct {
	Rule     CfnLintRule `json:"Rule"`
	Location CfnLintLoc  `json:"Location"`
	Level    string      `json:"Level"` // "Error", "Warning", "Informational"
	Message  string      `json:"Message"`
}

// CfnLintRule contains rule metadata.
type CfnLintRule struct {
	ID          string `json:"Id"`
	Description string `json:"Description"`
	ShortDesc   string `json:"ShortDescription"`
	Source      string `json:"Source"`
}

// CfnLintLoc contains location information.
type CfnLintLoc struct {
	Start    CfnLintPos `json:"Start"`
	End      CfnLintPos `json:"End"`
	Path     []any      `json:"Path"`
	Filename string     `json:"Filename"`
}

// CfnLintPos represents a position in a file.
type CfnLintPos struct {
	LineNumber   int `json:"LineNumber"`
	ColumnNumber int `json:"ColumnNumber"`
}

// CfnLintResult contains the result of running cfn-lint.
type CfnLintResult struct {
	Passed        bool     `json:"passed"`
	Errors        []string `json:"errors"`
	Warnings      []string `json:"warnings"`
	Informational []string `json:"informational"`
	RawOutput     string   `json:"raw_output,omitempty"`
}

// TotalIssues returns the total number of issues found.
func (r CfnLintResult) TotalIssues() int {
	return len(r.Errors) + len(r.Warnings) + len(r.Informational)
}

// ValidationResult contains all validation results for a package.
type ValidationResult struct {
	LintResult    *LintResult    `json:"lint_result"`
	BuildResult   *BuildResult   `json:"build_result"`
	CfnLintResult *CfnLintResult `json:"cfn_lint_result"`
}

// RunLint runs wetwire-aws lint on the given directory.
func RunLint(packageDir string) (*LintResult, error) {
	cmd := exec.Command("wetwire-aws", "lint", packageDir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	result := &LintResult{
		Passed: err == nil,
		Output: output,
		Issues: []string{},
	}

	// Parse issues from output (lines containing ": WAW")
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ": WAW") {
			result.Issues = append(result.Issues, line)
		}
	}

	// If no exit error but has issues, still mark as not passed
	if len(result.Issues) > 0 {
		result.Passed = false
	}

	return result, nil
}

// RunBuild runs wetwire-aws build on the given package and returns the template.
func RunBuild(packageDir string) (*BuildResult, error) {
	// Determine module name from directory name
	moduleName := filepath.Base(packageDir)

	// Build command with PYTHONPATH set to parent directory (for Python packages)
	// For Go packages, this might need adjustment
	cmd := exec.Command("wetwire-aws", "build", "-m", moduleName, "-f", "yaml")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", filepath.Dir(packageDir)))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		return &BuildResult{
			Success: false,
			Error:   stderr.String(),
		}, nil
	}

	return &BuildResult{
		Success:  true,
		Template: stdout.String(),
	}, nil
}

// RunBuildAndSave runs wetwire-aws build and saves the template to a file.
func RunBuildAndSave(packageDir, outputDir string) (*BuildResult, error) {
	result, err := RunBuild(packageDir)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return result, nil
	}

	// Save template to file
	moduleName := filepath.Base(packageDir)
	templatePath := filepath.Join(outputDir, moduleName+".yaml")

	if err := os.WriteFile(templatePath, []byte(result.Template), 0644); err != nil {
		return nil, fmt.Errorf("writing template: %w", err)
	}

	result.TemplatePath = templatePath
	return result, nil
}

// RunCfnLint runs cfn-lint-go on the given template file.
// This uses cfn-lint-go as a library dependency for guaranteed version control.
func RunCfnLint(templatePath string) (*CfnLintResult, error) {
	// Check if file exists
	if _, err := os.Stat(templatePath); err != nil {
		return &CfnLintResult{
			Passed: false,
			Errors: []string{fmt.Sprintf("Template file not found: %s", templatePath)},
		}, nil
	}

	// Create linter and run
	linter := lint.New(lint.Options{})
	matches, err := linter.LintFile(templatePath)
	if err != nil {
		return &CfnLintResult{
			Passed: false,
			Errors: []string{fmt.Sprintf("Linter error: %v", err)},
		}, nil
	}

	result := &CfnLintResult{
		Errors:        []string{},
		Warnings:      []string{},
		Informational: []string{},
	}

	// No issues found
	if len(matches) == 0 {
		result.Passed = true
		return result, nil
	}

	// Categorize issues by level
	for _, match := range matches {
		formatted := formatMatch(match)

		switch match.Level {
		case "Error":
			result.Errors = append(result.Errors, formatted)
		case "Warning":
			result.Warnings = append(result.Warnings, formatted)
		default:
			result.Informational = append(result.Informational, formatted)
		}
	}

	// Passed if no errors (warnings are acceptable)
	result.Passed = len(result.Errors) == 0

	return result, nil
}

// formatMatch formats a cfn-lint-go match for display.
func formatMatch(match lint.Match) string {
	// Format path if available
	pathStr := ""
	if len(match.Location.Path) > 0 {
		parts := make([]string, len(match.Location.Path))
		for i, p := range match.Location.Path {
			parts[i] = fmt.Sprintf("%v", p)
		}
		pathStr = strings.Join(parts, "/")
	}

	if pathStr != "" {
		return fmt.Sprintf("%s: %s (at %s)", match.Rule.ID, match.Message, pathStr)
	}
	return fmt.Sprintf("%s: %s", match.Rule.ID, match.Message)
}

// ValidatePackage runs the full validation pipeline on a package.
func ValidatePackage(packageDir, outputDir string) (*ValidationResult, error) {
	result := &ValidationResult{}

	// Step 1: Run lint
	lintResult, err := RunLint(packageDir)
	if err != nil {
		return nil, fmt.Errorf("running lint: %w", err)
	}
	result.LintResult = lintResult

	// Step 2: Run build (even if lint fails, to get as much feedback as possible)
	buildResult, err := RunBuildAndSave(packageDir, outputDir)
	if err != nil {
		return nil, fmt.Errorf("running build: %w", err)
	}
	result.BuildResult = buildResult

	// Step 3: Run cfn-lint (only if build succeeded)
	if buildResult.Success && buildResult.TemplatePath != "" {
		cfnResult, err := RunCfnLint(buildResult.TemplatePath)
		if err != nil {
			return nil, fmt.Errorf("running cfn-lint: %w", err)
		}
		result.CfnLintResult = cfnResult
	} else {
		result.CfnLintResult = &CfnLintResult{
			Passed: false,
			Errors: []string{"Build failed - no template to validate"},
		}
	}

	return result, nil
}
