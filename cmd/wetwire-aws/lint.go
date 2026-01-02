package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/linter"
)

func newLintCmd() *cobra.Command {
	var (
		outputFormat string
		fix          bool
	)

	cmd := &cobra.Command{
		Use:   "lint [packages...]",
		Short: "Check Go packages for issues",
		Long: `Lint checks Go packages containing CloudFormation resources for common issues.

Rules:
    WAW001: Use pseudo-parameter constants instead of hardcoded strings
    WAW002: Use intrinsic types instead of raw map[string]any
    WAW003: Detect duplicate resource variable names
    WAW004: Split large files with too many resources
    WAW005: Use struct types instead of inline map[string]any
    WAW006: Use constant for IAM policy version

Examples:
    wetwire-aws lint ./infra/...
    wetwire-aws lint ./infra/... --fix`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLint(args, outputFormat, fix)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format: text or json")
	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically fix issues where possible")

	return cmd
}

func runLint(packages []string, format string, fix bool) error {
	var issues []wetwire.LintIssue

	// Discover resources (validates references)
	discoverResult, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("lint failed: %w", err)
	}

	// Convert discovery errors to lint issues
	for _, e := range discoverResult.Errors {
		issues = append(issues, wetwire.LintIssue{
			Severity: "error",
			Message:  e.Error(),
			Rule:     "undefined-reference",
		})
	}

	// Run lint rules on each package
	for _, pkg := range packages {
		lintResult, err := linter.LintPackage(pkg, linter.Options{})
		if err != nil {
			// Log but continue on errors
			fmt.Fprintf(os.Stderr, "Warning: failed to lint %s: %v\n", pkg, err)
			continue
		}

		for _, issue := range lintResult.Issues {
			issues = append(issues, wetwire.LintIssue{
				Severity: issue.Severity,
				Message:  issue.Message,
				Rule:     issue.RuleID,
				File:     issue.File,
				Line:     issue.Line,
				Column:   issue.Column,
			})
		}
	}

	result := wetwire.LintResult{
		Success: len(issues) == 0,
		Issues:  issues,
	}

	return outputLintResult(result, format)
}

func outputLintResult(result wetwire.LintResult, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		if result.Success {
			fmt.Println("No issues found.")
			return nil
		}

		for _, issue := range result.Issues {
			severity := issue.Severity
			if issue.File != "" {
				fmt.Printf("%s:%d:%d: %s: %s [%s]\n",
					issue.File, issue.Line, issue.Column,
					severity, issue.Message, issue.Rule)
			} else {
				fmt.Printf("%s: %s [%s]\n", severity, issue.Message, issue.Rule)
			}
		}

	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	if !result.Success {
		os.Exit(2) // Exit code 2 for issues found
	}

	return nil
}
