package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/optimizer"
)

// validCategories lists all valid optimization categories.
var validCategories = map[string]bool{
	"all":         true,
	"security":    true,
	"cost":        true,
	"performance": true,
	"reliability": true,
}

// isValidCategory checks if a category is valid.
func isValidCategory(category string) bool {
	return validCategories[category]
}

// newOptimizeCmd creates the "optimize" subcommand for suggesting improvements.
func newOptimizeCmd() *cobra.Command {
	var (
		outputFormat string
		category     string
	)

	cmd := &cobra.Command{
		Use:   "optimize [packages...]",
		Short: "Suggest CloudFormation optimizations",
		Long: `Optimize analyzes CloudFormation resources and suggests improvements
for security, cost, performance, and reliability.

Categories:
    security     - Security best practices (encryption, access control, etc.)
    cost         - Cost optimization suggestions (right-sizing, lifecycle rules, etc.)
    performance  - Performance improvements (memory, caching, etc.)
    reliability  - Reliability enhancements (backups, multi-AZ, auto-scaling, etc.)

Examples:
    wetwire-aws optimize ./infra/...
    wetwire-aws optimize ./infra/... --category security
    wetwire-aws optimize ./infra/... -f json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isValidCategory(category) {
				return fmt.Errorf("invalid category: %s (valid: all, security, cost, performance, reliability)", category)
			}
			return runOptimize(args, outputFormat, category)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format: text or json")
	cmd.Flags().StringVarP(&category, "category", "c", "all", "Category: all, security, cost, performance, or reliability")

	return cmd
}

// runOptimize analyzes packages and suggests optimizations.
func runOptimize(packages []string, format, category string) error {
	// Discover resources
	discoverResult, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("optimize failed: %w", err)
	}

	// Run optimizer
	optResult, err := optimizer.Optimize(discoverResult, optimizer.Options{
		Category: category,
	})
	if err != nil {
		return fmt.Errorf("optimize failed: %w", err)
	}

	result := wetwire.OptimizeResult{
		Success:       true,
		Suggestions:   optResult.Suggestions,
		ResourceCount: len(discoverResult.Resources),
		Summary:       optResult.Summary,
	}

	return outputOptimizeResult(result, format)
}

func outputOptimizeResult(result wetwire.OptimizeResult, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		if len(result.Suggestions) == 0 {
			fmt.Printf("Analyzed %d resources. No optimization suggestions.\n", result.ResourceCount)
			return nil
		}

		fmt.Printf("Analyzed %d resources. Found %d suggestions:\n\n", result.ResourceCount, result.Summary.Total)

		// Group by category
		byCat := map[string][]wetwire.OptimizeSuggestion{}
		for _, s := range result.Suggestions {
			byCat[s.Category] = append(byCat[s.Category], s)
		}

		categoryOrder := []string{"security", "cost", "performance", "reliability"}
		for _, cat := range categoryOrder {
			suggestions := byCat[cat]
			if len(suggestions) == 0 {
				continue
			}

			fmt.Printf("=== %s (%d) ===\n", capitalize(cat), len(suggestions))
			for _, s := range suggestions {
				fmt.Printf("\n[%s] %s\n", s.Severity, s.Title)
				fmt.Printf("  Resource: %s\n", s.Resource)
				if s.File != "" {
					fmt.Printf("  Location: %s:%d\n", s.File, s.Line)
				}
				fmt.Printf("  %s\n", s.Description)
				fmt.Printf("  Suggestion: %s\n", s.Suggestion)
			}
			fmt.Println()
		}

		fmt.Printf("Summary: %d security, %d cost, %d performance, %d reliability\n",
			result.Summary.Security, result.Summary.Cost,
			result.Summary.Performance, result.Summary.Reliability)

	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	if result.Summary.Total > 0 {
		os.Exit(0) // Suggestions found is not an error
	}

	return nil
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
