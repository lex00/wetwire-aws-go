package main

import (
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
	// Discover resources and other template components
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

	// Build template with all discovered components
	builder := template.NewBuilderFull(
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)

	// Set VarAttrRefs for recursive AttrRef resolution
	varAttrRefs := make(map[string]template.VarAttrRefInfo)
	for name, info := range result.VarAttrRefs {
		varAttrRefs[name] = template.VarAttrRefInfo{
			AttrRefs: info.AttrRefs,
			VarRefs:  info.VarRefs,
		}
	}
	builder.SetVarAttrRefs(varAttrRefs)

	// Extract all values by running a generated Go program
	values, err := runner.ExtractAll(
		packages[0],
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)
	if err != nil {
		buildResult := wetwire.BuildResult{
			Success: false,
			Errors:  []string{fmt.Sprintf("extracting values: %v", err)},
		}
		return outputResult(buildResult, format, outputFile)
	}

	// Set all extracted values
	for name, props := range values.Resources {
		builder.SetValue(name, props)
	}
	for name, props := range values.Parameters {
		builder.SetValue(name, props)
	}
	for name, props := range values.Outputs {
		builder.SetValue(name, props)
	}
	for name, val := range values.Mappings {
		builder.SetValue(name, val)
	}
	for name, val := range values.Conditions {
		builder.SetValue(name, val)
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
	// Handle build failures - output errors to stderr
	if !result.Success {
		for _, e := range result.Errors {
			fmt.Fprintln(os.Stderr, e)
		}
		return fmt.Errorf("build failed")
	}

	// Output raw CloudFormation template (matching Python wetwire-aws behavior)
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = template.ToJSON(&result.Template)
	case "yaml":
		data, err = template.ToYAML(&result.Template)
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
