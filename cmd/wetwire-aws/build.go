package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/runner"
	"github.com/lex00/wetwire-aws-go/internal/template"
)

func newBuildCmd() *cobra.Command {
	var (
		outputFormat string
		outputFile   string
	)

	cmd := &cobra.Command{
		Use:   "build [packages...]",
		Short: "Generate CloudFormation template from Go packages",
		Long: `Build discovers CloudFormation resources in Go packages and generates a template.

Examples:
    wetwire-aws build ./infra/...
    wetwire-aws build ./infra/... -o template.json
    wetwire-aws build ./infra/... --format yaml`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(args, outputFormat, outputFile)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "Output format: json or yaml")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")

	return cmd
}

func runBuild(packages []string, format, outputFile string) error {
	// Discover resources
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Check for discovery errors
	if len(result.Errors) > 0 {
		buildResult := wetwire.BuildResult{
			Success: false,
			Errors:  make([]string, len(result.Errors)),
		}
		for i, e := range result.Errors {
			buildResult.Errors[i] = e.Error()
		}
		return outputResult(buildResult, format, outputFile)
	}

	// Build template
	builder := template.NewBuilder(result.Resources)

	// Extract actual resource values by running a generated Go program
	values, err := runner.ExtractValues(packages[0], result.Resources)
	if err != nil {
		buildResult := wetwire.BuildResult{
			Success: false,
			Errors:  []string{fmt.Sprintf("extracting values: %v", err)},
		}
		return outputResult(buildResult, format, outputFile)
	}

	// Set the extracted values
	for name, props := range values {
		builder.SetValue(name, props)
	}

	tmpl, err := builder.Build()
	if err != nil {
		buildResult := wetwire.BuildResult{
			Success: false,
			Errors:  []string{err.Error()},
		}
		return outputResult(buildResult, format, outputFile)
	}

	// Build success result
	resourceNames := make([]string, 0, len(result.Resources))
	for name := range result.Resources {
		resourceNames = append(resourceNames, name)
	}

	buildResult := wetwire.BuildResult{
		Success:   true,
		Template:  *tmpl,
		Resources: resourceNames,
	}

	return outputResult(buildResult, format, outputFile)
}

func outputResult(result wetwire.BuildResult, format, outputFile string) error {
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(result, "", "  ")
	case "yaml":
		// For YAML, just output the template directly
		if result.Success {
			data, err = template.ToYAML(&result.Template)
		} else {
			data, err = json.MarshalIndent(result, "", "  ")
		}
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	if err != nil {
		return err
	}

	if outputFile == "" {
		fmt.Println(string(data))
		return nil
	}

	return os.WriteFile(outputFile, data, 0644)
}
