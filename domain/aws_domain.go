package domain

import (
	"context"
	"fmt"
	"os"

	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/importer"
	"github.com/lex00/wetwire-aws-go/internal/linter"
	"github.com/lex00/wetwire-aws-go/internal/runner"
	"github.com/lex00/wetwire-aws-go/internal/schema"
	"github.com/lex00/wetwire-aws-go/internal/template"
	"github.com/lex00/wetwire-aws-go/version"
	"github.com/lex00/wetwire-core-go/cmd"
)

// AwsDomain implements the Domain interface for AWS CloudFormation.
type AwsDomain struct{}

// Compile-time check that AwsDomain implements Domain and all optional interfaces
var (
	_ Domain            = (*AwsDomain)(nil)
	_ OptionalImporter  = (*AwsDomain)(nil)
	_ OptionalLister    = (*AwsDomain)(nil)
	_ OptionalGrapher   = (*AwsDomain)(nil)
)

// Name returns "aws"
func (d *AwsDomain) Name() string {
	return "aws"
}

// Version returns the current version
func (d *AwsDomain) Version() string {
	return version.Version()
}

// Builder returns the AWS CloudFormation builder implementation
func (d *AwsDomain) Builder() cmd.Builder {
	return &awsBuilder{}
}

// Linter returns the AWS linter implementation
func (d *AwsDomain) Linter() cmd.Linter {
	return &awsLinter{}
}

// Initializer returns the AWS project initializer implementation
func (d *AwsDomain) Initializer() cmd.Initializer {
	return &awsInitializer{}
}

// Validator returns the AWS validator implementation
func (d *AwsDomain) Validator() cmd.Validator {
	return &awsValidator{}
}

// Importer returns the AWS CloudFormation importer implementation
func (d *AwsDomain) Importer() Importer {
	return &awsImporter{}
}

// Lister returns the AWS resource lister implementation
func (d *AwsDomain) Lister() Lister {
	return &awsLister{}
}

// Grapher returns the AWS dependency grapher implementation
func (d *AwsDomain) Grapher() Grapher {
	return &awsGrapher{}
}

// awsBuilder implements cmd.Builder for AWS
type awsBuilder struct{}

func (b *awsBuilder) Build(ctx context.Context, path string, opts cmd.BuildOptions) error {
	packages := []string{path}

	// Discover resources
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Check for discovery errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintln(os.Stderr, e.Error())
		}
		return fmt.Errorf("build failed: discovery errors")
	}

	// Build template
	builder := template.NewBuilderFull(
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)

	// Set VarAttrRefs
	varAttrRefs := make(map[string]template.VarAttrRefInfo)
	for name, info := range result.VarAttrRefs {
		varAttrRefs[name] = template.VarAttrRefInfo{
			AttrRefs: info.AttrRefs,
			VarRefs:  info.VarRefs,
		}
	}
	builder.SetVarAttrRefs(varAttrRefs)

	// Extract all values
	values, err := runner.ExtractAll(
		packages[0],
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)
	if err != nil {
		return fmt.Errorf("extracting values: %w", err)
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
		return fmt.Errorf("building template: %w", err)
	}

	// Output template
	var data []byte
	if opts.Output == "" || opts.Output == "-" {
		data, err = template.ToJSON(tmpl)
		if err != nil {
			return fmt.Errorf("serializing template: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Determine format from file extension
		format := "json"
		if len(opts.Output) > 5 && opts.Output[len(opts.Output)-5:] == ".yaml" {
			format = "yaml"
		} else if len(opts.Output) > 4 && opts.Output[len(opts.Output)-4:] == ".yml" {
			format = "yaml"
		}

		if format == "yaml" {
			data, err = template.ToYAML(tmpl)
		} else {
			data, err = template.ToJSON(tmpl)
		}
		if err != nil {
			return fmt.Errorf("serializing template: %w", err)
		}

		if err := os.WriteFile(opts.Output, data, 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
	}

	return nil
}

// awsLinter implements cmd.Linter for AWS
type awsLinter struct{}

func (l *awsLinter) Lint(ctx context.Context, path string, opts cmd.LintOptions) ([]cmd.Issue, error) {
	result, err := linter.LintPackage(path, linter.Options{})
	if err != nil {
		return nil, fmt.Errorf("linting failed: %w", err)
	}

	// Convert linter issues to cmd.Issue format
	issues := make([]cmd.Issue, 0, len(result.Issues))
	for _, issue := range result.Issues {
		issues = append(issues, cmd.Issue{
			File:     issue.File,
			Line:     issue.Line,
			Column:   issue.Column,
			Severity: issue.Severity,
			Message:  issue.Message,
			Rule:     issue.RuleID,
		})
	}

	// Output issues to stderr
	if !opts.Verbose && len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "%s:%d:%d: %s (%s)\n",
				issue.File, issue.Line, issue.Column, issue.Message, issue.Rule)
		}
	}

	return issues, nil
}

// awsInitializer implements cmd.Initializer for AWS
type awsInitializer struct{}

func (i *awsInitializer) Init(ctx context.Context, name string, opts cmd.InitOptions) error {
	// Use the existing init logic from init.go
	// For now, this is a placeholder that would need the actual implementation
	return fmt.Errorf("init not yet implemented via domain interface")
}

// awsValidator implements cmd.Validator for AWS
type awsValidator struct{}

func (v *awsValidator) Validate(ctx context.Context, path string, opts cmd.ValidateOptions) ([]cmd.ValidationError, error) {
	packages := []string{path}

	// First build the template
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Build template
	builder := template.NewBuilderFull(
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)

	// Set VarAttrRefs
	varAttrRefs := make(map[string]template.VarAttrRefInfo)
	for name, info := range result.VarAttrRefs {
		varAttrRefs[name] = template.VarAttrRefInfo{
			AttrRefs: info.AttrRefs,
			VarRefs:  info.VarRefs,
		}
	}
	builder.SetVarAttrRefs(varAttrRefs)

	// Extract all values
	values, err := runner.ExtractAll(
		packages[0],
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)
	if err != nil {
		return nil, fmt.Errorf("extracting values: %w", err)
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
		return nil, fmt.Errorf("building template: %w", err)
	}

	// Validate the template
	validationResult, err := schema.ValidateTemplate(tmpl, schema.Options{
		Strict: opts.Strict,
	})
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Convert validation errors to cmd.ValidationError format
	errors := make([]cmd.ValidationError, 0, len(validationResult.Errors))
	for _, verr := range validationResult.Errors {
		errors = append(errors, cmd.ValidationError{
			Path:    fmt.Sprintf("%s.%s", verr.Resource, verr.Property),
			Message: verr.Message,
			Code:    verr.Resource,
		})
	}

	return errors, nil
}

// awsImporter implements Importer for AWS
type awsImporter struct{}

func (i *awsImporter) Import(inputPath, outputPath string) error {
	if outputPath == "" {
		outputPath = "imported.go"
	}

	// Parse the CloudFormation template
	irTemplate, err := importer.ParseTemplate(inputPath)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Generate Go code
	packageName := importer.DerivePackageName(inputPath)
	files := importer.GenerateCode(irTemplate, packageName)

	// Write the main file
	var mainContent string
	for filename, content := range files {
		if filename == "main.go" || filename == packageName+".go" {
			mainContent = content
			break
		}
	}

	if mainContent == "" {
		return fmt.Errorf("no main content generated")
	}

	if err := os.WriteFile(outputPath, []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("Imported template to %s\n", outputPath)
	return nil
}

// awsLister implements Lister for AWS
type awsLister struct{}

func (l *awsLister) List(packages []string, format string) error {
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	switch format {
	case "text":
		fmt.Printf("Discovered %d resources:\n", len(result.Resources))
		for name, res := range result.Resources {
			fmt.Printf("  - %s (%s)\n", name, res.Type)
		}
	case "json", "yaml":
		// TODO: Implement JSON/YAML output
		return fmt.Errorf("format %s not yet implemented", format)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	return nil
}

// awsGrapher implements Grapher for AWS
type awsGrapher struct{}

func (g *awsGrapher) Graph(packages []string, format string) error {
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	switch format {
	case "dot":
		// Generate DOT format graph
		fmt.Println("digraph G {")
		for name := range result.Resources {
			fmt.Printf("  %s;\n", name)
		}
		// TODO: Add edges based on dependencies
		fmt.Println("}")
	case "mermaid":
		return fmt.Errorf("mermaid format not yet implemented")
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	return nil
}
