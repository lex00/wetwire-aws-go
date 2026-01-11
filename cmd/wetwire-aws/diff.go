package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/differ"
)

// newDiffCmd creates the "diff" subcommand for comparing templates.
func newDiffCmd() *cobra.Command {
	var (
		outputFormat string
		ignoreOrder  bool
	)

	cmd := &cobra.Command{
		Use:   "diff <template1> <template2>",
		Short: "Compare two CloudFormation templates",
		Long: `Diff performs a semantic comparison of two CloudFormation templates,
showing added, removed, and modified resources.

The comparison is semantic, not textual - it compares the structure and values
of resources rather than the raw text.

Examples:
    wetwire-aws diff old.json new.json
    wetwire-aws diff old.yaml new.yaml -f json
    wetwire-aws diff template1.json template2.json --ignore-order`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(args[0], args[1], outputFormat, ignoreOrder)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format: text or json")
	cmd.Flags().BoolVar(&ignoreOrder, "ignore-order", false, "Ignore array element order in comparisons")

	return cmd
}

// runDiff compares two templates and outputs differences.
func runDiff(file1, file2, format string, ignoreOrder bool) error {
	diffResult, err := differ.CompareFiles(file1, file2, differ.Options{
		IgnoreOrder: ignoreOrder,
	})
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	result := wetwire.DiffResult{
		Success: true,
		Diff:    diffResult.Diff,
		Summary: diffResult.Summary,
	}

	return outputDiffResult(result, format, file1, file2)
}

func outputDiffResult(result wetwire.DiffResult, format, file1, file2 string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		if result.Summary.Total == 0 {
			fmt.Printf("No differences between %s and %s\n", file1, file2)
			return nil
		}

		fmt.Printf("Comparing %s vs %s\n\n", file1, file2)

		if len(result.Diff.Added) > 0 {
			fmt.Println("=== Added Resources ===")
			for _, entry := range result.Diff.Added {
				fmt.Printf("  + %s (%s)\n", entry.Resource, entry.Type)
			}
			fmt.Println()
		}

		if len(result.Diff.Removed) > 0 {
			fmt.Println("=== Removed Resources ===")
			for _, entry := range result.Diff.Removed {
				fmt.Printf("  - %s (%s)\n", entry.Resource, entry.Type)
			}
			fmt.Println()
		}

		if len(result.Diff.Modified) > 0 {
			fmt.Println("=== Modified Resources ===")
			for _, entry := range result.Diff.Modified {
				fmt.Printf("  ~ %s (%s)\n", entry.Resource, entry.Type)
				for _, change := range entry.Changes {
					fmt.Printf("      %s\n", change)
				}
			}
			fmt.Println()
		}

		fmt.Printf("Summary: %d added, %d removed, %d modified\n",
			result.Summary.Added, result.Summary.Removed, result.Summary.Modified)

	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	if result.Summary.Total > 0 {
		os.Exit(1) // Exit code 1 indicates differences found
	}

	return nil
}
