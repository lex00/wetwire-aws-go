package domain

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/importer"
	"github.com/lex00/wetwire-aws-go/internal/lint"
	"github.com/lex00/wetwire-aws-go/internal/runner"
	"github.com/lex00/wetwire-aws-go/internal/schema"
	"github.com/lex00/wetwire-aws-go/internal/template"
	coredomain "github.com/lex00/wetwire-core-go/domain"
)

// AwsDomain implements the Domain interface for AWS CloudFormation.
type AwsDomain struct{}

// Compile-time check that AwsDomain implements Domain and all optional interfaces
var (
	_ coredomain.Domain         = (*AwsDomain)(nil)
	_ coredomain.ImporterDomain = (*AwsDomain)(nil)
	_ coredomain.ListerDomain   = (*AwsDomain)(nil)
	_ coredomain.GrapherDomain  = (*AwsDomain)(nil)
)

// Name returns "aws"
func (d *AwsDomain) Name() string {
	return "aws"
}

// Version returns the current version
func (d *AwsDomain) Version() string {
	return Version
}

// Builder returns the AWS CloudFormation builder implementation
func (d *AwsDomain) Builder() coredomain.Builder {
	return &awsBuilder{}
}

// Linter returns the AWS linter implementation
func (d *AwsDomain) Linter() coredomain.Linter {
	return &awsLinter{}
}

// Initializer returns the AWS project initializer implementation
func (d *AwsDomain) Initializer() coredomain.Initializer {
	return &awsInitializer{}
}

// Validator returns the AWS validator implementation
func (d *AwsDomain) Validator() coredomain.Validator {
	return &awsValidator{}
}

// Importer returns the AWS CloudFormation importer implementation
func (d *AwsDomain) Importer() coredomain.Importer {
	return &awsImporter{}
}

// Lister returns the AWS resource lister implementation
func (d *AwsDomain) Lister() coredomain.Lister {
	return &awsLister{}
}

// Grapher returns the AWS dependency grapher implementation
func (d *AwsDomain) Grapher() coredomain.Grapher {
	return &awsGrapher{}
}

// awsBuilder implements domain.Builder for AWS
type awsBuilder struct{}

func (b *awsBuilder) Build(ctx *Context, path string, opts BuildOpts) (*Result, error) {
	packages := []string{path}

	// Discover resources
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Check for discovery errors
	if len(result.Errors) > 0 {
		errs := make([]Error, 0, len(result.Errors))
		for _, e := range result.Errors {
			errs = append(errs, Error{
				Message: e.Error(),
			})
		}
		return NewErrorResultMultiple("discovery errors", errs), nil
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

	// Serialize template to JSON for the result
	data, err := template.ToJSON(tmpl)
	if err != nil {
		return nil, fmt.Errorf("serializing template: %w", err)
	}

	// If DryRun is enabled, return the template without writing
	if opts.DryRun {
		return NewResultWithData("Build completed (dry run - no files written)", string(data)), nil
	}

	// If Output path is specified, write the template to file
	if opts.Output != "" {
		if err := os.WriteFile(opts.Output, data, 0644); err != nil {
			return nil, fmt.Errorf("writing template to %s: %w", opts.Output, err)
		}
		return NewResultWithData(fmt.Sprintf("Build completed, template written to %s", opts.Output), string(data)), nil
	}

	// Return the JSON as the result data
	return NewResultWithData("Build completed", string(data)), nil
}

// awsLinter implements domain.Linter for AWS
type awsLinter struct{}

func (l *awsLinter) Lint(ctx *Context, path string, opts LintOpts) (*Result, error) {
	// Build lint options from LintOpts
	lintOpts := lint.Options{
		DisabledRules: opts.Disable,
		Fix:           opts.Fix,
	}

	result, err := lint.LintPackage(path, lintOpts)
	if err != nil {
		return nil, fmt.Errorf("linting failed: %w", err)
	}

	// Convert linter issues to domain.Error format
	if len(result.Issues) > 0 {
		errs := make([]Error, 0, len(result.Issues))
		for _, issue := range result.Issues {
			errs = append(errs, Error{
				Path:     issue.File,
				Line:     issue.Line,
				Column:   issue.Column,
				Severity: issue.Severity.String(),
				Message:  issue.Message,
				Code:     issue.Rule,
			})
		}

		// If Fix mode is enabled, add a note about auto-fixing
		if opts.Fix {
			return NewErrorResultMultiple("lint issues found (auto-fix not yet implemented for these issues)", errs), nil
		}
		return NewErrorResultMultiple("lint issues found", errs), nil
	}

	return NewResult("No lint issues found"), nil
}

// awsInitializer implements domain.Initializer for AWS
type awsInitializer struct{}

func (i *awsInitializer) Init(ctx *Context, path string, opts InitOpts) (*Result, error) {
	// Use opts.Path if provided, otherwise fall back to path argument
	targetPath := opts.Path
	if targetPath == "" || targetPath == "." {
		targetPath = path
	}

	// Handle scenario initialization
	if opts.Scenario {
		return i.initScenario(ctx, targetPath, opts)
	}

	// Basic project initialization
	return i.initProject(ctx, targetPath, opts)
}

// initScenario creates a full scenario structure with prompts and expected outputs
func (i *awsInitializer) initScenario(ctx *Context, path string, opts InitOpts) (*Result, error) {
	name := opts.Name
	if name == "" {
		name = filepath.Base(path)
	}

	description := opts.Description
	if description == "" {
		description = "AWS CloudFormation scenario"
	}

	// Use core's scenario scaffolding
	scenario := coredomain.ScaffoldScenario(name, description, "aws")
	created, err := coredomain.WriteScenario(path, scenario)
	if err != nil {
		return nil, fmt.Errorf("write scenario: %w", err)
	}

	// Create AWS-specific expected directories
	expectedDirs := []string{
		filepath.Join(path, "expected", "resources"),
		filepath.Join(path, "expected", "parameters"),
		filepath.Join(path, "expected", "outputs"),
	}
	for _, dir := range expectedDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Create example CloudFormation code in expected/resources/
	exampleResource := `package resources

import (
	"github.com/lex00/wetwire-aws-go/ec2"
	"github.com/lex00/wetwire-aws-go/iam"
)

// WebServerRole is the IAM role for the web server instance
var WebServerRole = iam.Role{
	AssumeRolePolicyDocument: map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "ec2.amazonaws.com",
				},
				"Action": "sts:AssumeRole",
			},
		},
	},
	ManagedPolicyArns: []string{
		"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
	},
}

// WebServerInstance is an EC2 instance for the web server
var WebServerInstance = ec2.Instance{
	InstanceType: "t3.micro",
	ImageId:      "ami-0c55b159cbfafe1f0", // Amazon Linux 2
	IamInstanceProfile: WebServerRole,
	Tags: []map[string]interface{}{
		{
			"Key":   "Name",
			"Value": "WebServer",
		},
	},
}
`
	resourcePath := filepath.Join(path, "expected", "resources", "resources.go")
	if err := os.WriteFile(resourcePath, []byte(exampleResource), 0644); err != nil {
		return nil, fmt.Errorf("write example resource: %w", err)
	}
	created = append(created, "expected/resources/resources.go")

	return NewResultWithData(
		fmt.Sprintf("Created scenario %s with %d files", name, len(created)),
		created,
	), nil
}

// initProject creates a basic project with example resources
func (i *awsInitializer) initProject(ctx *Context, path string, opts InitOpts) (*Result, error) {
	// Create directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	// Create go.mod
	name := opts.Name
	if name == "" {
		name = filepath.Base(path)
	}
	goMod := fmt.Sprintf(`module %s

go 1.23

require github.com/lex00/wetwire-aws-go v0.0.0
`, name)
	goModPath := filepath.Join(path, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil {
		return nil, fmt.Errorf("write go.mod: %w", err)
	}

	// Create example resource file
	exampleContent := `package main

import (
	"github.com/lex00/wetwire-aws-go/s3"
)

// MyBucket is an S3 bucket for storing application data
var MyBucket = s3.Bucket{
	BucketName: "my-app-bucket", // TODO: Change to unique name
	VersioningConfiguration: &s3.VersioningConfiguration{
		Status: "Enabled",
	},
	PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
		BlockPublicAcls:       true,
		BlockPublicPolicy:     true,
		IgnorePublicAcls:      true,
		RestrictPublicBuckets: true,
	},
}
`
	examplePath := filepath.Join(path, "main.go")
	if err := os.WriteFile(examplePath, []byte(exampleContent), 0644); err != nil {
		return nil, fmt.Errorf("write example: %w", err)
	}

	return NewResultWithData(
		fmt.Sprintf("Created %s with example resources", path),
		[]string{"go.mod", "main.go"},
	), nil
}

// awsValidator implements domain.Validator for AWS
type awsValidator struct{}

func (v *awsValidator) Validate(ctx *Context, path string, opts ValidateOpts) (*Result, error) {
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
	validationResult, err := schema.ValidateTemplate(tmpl, schema.Options{})
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Convert validation errors to domain.Error format
	if len(validationResult.Errors) > 0 {
		errs := make([]Error, 0, len(validationResult.Errors))
		for _, verr := range validationResult.Errors {
			errs = append(errs, Error{
				Path:    fmt.Sprintf("%s.%s", verr.Resource, verr.Property),
				Message: verr.Message,
				Code:    verr.Resource,
			})
		}
		return NewErrorResultMultiple("validation errors", errs), nil
	}

	return NewResult("Validation passed"), nil
}

// awsImporter implements domain.Importer for AWS
type awsImporter struct{}

func (i *awsImporter) Import(ctx *Context, source string, opts ImportOpts) (*Result, error) {
	outputPath := opts.Target
	if outputPath == "" {
		outputPath = "imported.go"
	}

	// Parse the CloudFormation template
	irTemplate, err := importer.ParseTemplate(source)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	// Generate Go code
	packageName := importer.DerivePackageName(source)
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
		return nil, fmt.Errorf("no main content generated")
	}

	if err := os.WriteFile(outputPath, []byte(mainContent), 0644); err != nil {
		return nil, fmt.Errorf("writing output: %w", err)
	}

	return NewResult(fmt.Sprintf("Imported template to %s", outputPath)), nil
}

// awsLister implements domain.Lister for AWS
type awsLister struct{}

func (l *awsLister) List(ctx *Context, path string, opts ListOpts) (*Result, error) {
	result, err := discover.Discover(discover.Options{
		Packages: []string{path},
	})
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Build a list of resources
	resources := make([]map[string]string, 0, len(result.Resources))
	for name, res := range result.Resources {
		resources = append(resources, map[string]string{
			"name": name,
			"type": res.Type,
		})
	}

	return NewResultWithData(fmt.Sprintf("Discovered %d resources", len(result.Resources)), resources), nil
}

// awsGrapher implements domain.Grapher for AWS
type awsGrapher struct{}

func (g *awsGrapher) Graph(ctx *Context, path string, opts GraphOpts) (*Result, error) {
	result, err := discover.Discover(discover.Options{
		Packages: []string{path},
	})
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Generate DOT format graph as string
	var graph string
	switch opts.Format {
	case "dot", "":
		graph = "digraph G {\n"
		for name := range result.Resources {
			graph += fmt.Sprintf("  %s;\n", name)
		}
		// TODO: Add edges based on dependencies
		graph += "}"
	case "mermaid":
		return nil, fmt.Errorf("mermaid format not yet implemented")
	default:
		return nil, fmt.Errorf("unknown format: %s", opts.Format)
	}

	return NewResultWithData("Graph generated", graph), nil
}
