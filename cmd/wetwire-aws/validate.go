package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
)

// newValidateCmd creates the "validate" subcommand for checking resource validity.
func newValidateCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "validate [packages...]",
		Short: "Validate resources and references",
		Long: `Validate discovers CloudFormation resources and checks for issues.

Checks performed:
  - Reference validity: All resource references point to defined resources
  - Dependency graph: Validates resource dependencies exist

Examples:
    wetwire-aws validate ./infra/...
    wetwire-aws validate ./infra/... --format json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args, outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format: text or json")

	return cmd
}

// runValidate validates resources and checks for undefined references.
func runValidate(packages []string, format string) error {
	// Discover resources (this also validates references)
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Build validation result
	validateResult := wetwire.ValidateResult{
		Success:   len(result.Errors) == 0,
		Resources: len(result.Resources),
	}

	for _, e := range result.Errors {
		validateResult.Errors = append(validateResult.Errors, e.Error())
	}

	return outputValidateResult(validateResult, format)
}

func outputValidateResult(result wetwire.ValidateResult, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		if result.Success {
			fmt.Printf("Validation passed: %d resources OK\n", result.Resources)
			return nil
		}

		fmt.Println("Validation FAILED:")
		for _, errMsg := range result.Errors {
			fmt.Printf("  ERROR: %s\n", errMsg)
		}
		for _, warnMsg := range result.Warnings {
			fmt.Printf("  WARNING: %s\n", warnMsg)
		}

	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	if !result.Success {
		os.Exit(1)
	}

	return nil
}
