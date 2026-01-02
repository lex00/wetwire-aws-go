package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
)

func newListCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list [packages...]",
		Short: "List discovered resources",
		Long: `List discovers and displays all CloudFormation resources in the specified packages.

Examples:
    wetwire-aws list ./infra/...
    wetwire-aws list ./infra/... --format json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(args, outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format: text or json")

	return cmd
}

func runList(packages []string, format string) error {
	// Discover resources
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Build list result
	listResult := wetwire.ListResult{
		Resources: make([]wetwire.ListResource, 0, len(result.Resources)),
	}

	for name, res := range result.Resources {
		listResult.Resources = append(listResult.Resources, wetwire.ListResource{
			Name: name,
			Type: res.Type,
			File: res.File,
			Line: res.Line,
		})
	}

	// Sort by name for consistent output
	sort.Slice(listResult.Resources, func(i, j int) bool {
		return listResult.Resources[i].Name < listResult.Resources[j].Name
	})

	return outputListResult(listResult, format)
}

func outputListResult(result wetwire.ListResult, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		if len(result.Resources) == 0 {
			fmt.Println("No resources found.")
			return nil
		}

		fmt.Printf("Registered resources (%d):\n\n", len(result.Resources))
		for _, res := range result.Resources {
			fmt.Printf("  %s: %s\n", res.Name, res.Type)
		}

	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	return nil
}
