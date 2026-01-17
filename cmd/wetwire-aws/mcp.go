// Command mcp runs an MCP server that exposes wetwire-aws tools.
//
// This server implements the Model Context Protocol (MCP) using infrastructure
// from github.com/lex00/wetwire-core-go/mcp and automatically generates tools
// from the domain.Domain interface implementation.
//
// Tools are automatically registered based on the domain interface:
//   - wetwire_init: Initialize a new wetwire-aws project
//   - wetwire_build: Generate CloudFormation template from Go packages
//   - wetwire_lint: Lint Go packages for wetwire-aws issues
//   - wetwire_validate: Validate generated CloudFormation templates
//   - wetwire_import: Import existing CloudFormation templates to Go code
//   - wetwire_list: List discovered resources
//   - wetwire_graph: Visualize resource dependencies
//
// Usage:
//
//	wetwire-aws mcp  # Runs on stdio transport
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lex00/wetwire-aws-go/domain"
	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/version"
	coredomain "github.com/lex00/wetwire-core-go/domain"
	coremcp "github.com/lex00/wetwire-core-go/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP server for wetwire-aws tools",
		Long: `Run an MCP (Model Context Protocol) server that exposes wetwire-aws tools.

This command starts an MCP server on stdio transport, automatically providing tools
generated from the domain interface for:
- Initializing projects (wetwire_init)
- Building CloudFormation templates (wetwire_build)
- Linting code (wetwire_lint)
- Validating templates (wetwire_validate)
- Importing templates (wetwire_import)
- Listing resources (wetwire_list)
- Generating dependency graphs (wetwire_graph)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer()
		},
	}
}

func runMCPServer() error {
	server := coremcp.NewServer(coremcp.Config{
		Name:    "wetwire-aws",
		Version: version.Version(),
	})

	// Create domain instance
	awsDomain := &domain.AwsDomain{}

	// Register standard tools from domain interface
	handlers := coremcp.StandardToolHandlers{
		Init:     makeDomainInitHandler(awsDomain),
		Build:    makeDomainBuildHandler(awsDomain),
		Lint:     makeDomainLintHandler(awsDomain),
		Validate: makeDomainValidateHandler(awsDomain),
		Import:   makeDomainImportHandler(awsDomain),
		List:     makeDomainListHandler(awsDomain),
		Graph:    makeDomainGraphHandler(awsDomain),
	}

	// Register standard tools from domain interface
	coremcp.RegisterStandardTools(server, awsDomain.Name(), handlers)

	// Run on stdio transport
	return server.Start(context.Background())
}

// makeDomainInitHandler creates an MCP handler from the domain's Initializer
func makeDomainInitHandler(d *domain.AwsDomain) coremcp.ToolHandler {
	return func(_ context.Context, args map[string]any) (string, error) {
		name, _ := args["name"].(string)
		path, _ := args["path"].(string)

		if path == "" {
			path = "."
		}
		if name == "" {
			name = filepath.Base(path)
			if name == "." {
				cwd, _ := os.Getwd()
				name = filepath.Base(cwd)
			}
		}

		result := MCPInitResult{Path: path}

		// Create project directory
		if path != "." {
			if err := os.MkdirAll(path, 0755); err != nil {
				result.Error = fmt.Sprintf("creating project directory: %v", err)
				return toJSON(result)
			}
		}

		// Create infra subdirectory
		infraDir := filepath.Join(path, "infra")
		if err := os.MkdirAll(infraDir, 0755); err != nil {
			result.Error = fmt.Sprintf("creating infra directory: %v", err)
			return toJSON(result)
		}

		// Write go.mod
		goMod := fmt.Sprintf(`module %s

go 1.23

require github.com/lex00/wetwire-aws-go v1.2.3
`, name)

		goModPath := filepath.Join(path, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil {
			result.Error = fmt.Sprintf("writing go.mod: %v", err)
			return toJSON(result)
		}
		result.Files = append(result.Files, "go.mod")

		// Write main.go
		mainGo := `package main

import (
	"fmt"

	_ "` + name + `/infra" // Register resources
)

func main() {
	fmt.Println("Run: wetwire-aws build ./infra/...")
}
`
		mainGoPath := filepath.Join(path, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(mainGo), 0644); err != nil {
			result.Error = fmt.Sprintf("writing main.go: %v", err)
			return toJSON(result)
		}
		result.Files = append(result.Files, "main.go")

		// Write infra/resources.go
		resourcesGo := `package infra

import (
	// Common AWS services - add/remove as needed
	"github.com/lex00/wetwire-aws-go/resources/s3"
	"github.com/lex00/wetwire-aws-go/resources/iam"
	"github.com/lex00/wetwire-aws-go/resources/ec2"
	"github.com/lex00/wetwire-aws-go/resources/lambda"
	"github.com/lex00/wetwire-aws-go/resources/dynamodb"
	"github.com/lex00/wetwire-aws-go/resources/sqs"
	"github.com/lex00/wetwire-aws-go/resources/sns"
	"github.com/lex00/wetwire-aws-go/resources/apigateway"
)

// Define your infrastructure resources below

// Example bucket - uncomment and modify:
// var MyBucket = s3.Bucket{
//     BucketName: "my-bucket",
// }

// Placeholder imports to prevent unused import errors
// Remove these as you add real resources
var _ = s3.Bucket{}
var _ = iam.Role{}
var _ = ec2.VPC{}
var _ = lambda.Function{}
var _ = dynamodb.Table{}
var _ = sqs.Queue{}
var _ = sns.Topic{}
var _ = apigateway.RestApi{}
`
		resourcesGoPath := filepath.Join(infraDir, "resources.go")
		if err := os.WriteFile(resourcesGoPath, []byte(resourcesGo), 0644); err != nil {
			result.Error = fmt.Sprintf("writing resources.go: %v", err)
			return toJSON(result)
		}
		result.Files = append(result.Files, "infra/resources.go")

		// Write infra/params.go for Parameters, Mappings, and Conditions
		paramsGo := `package infra

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
)

// Parameters - define CloudFormation parameters here
// Example:
// var Environment = Parameter{
//     Type:          "String",
//     Description:   "Environment name",
//     Default:       "dev",
//     AllowedValues: []any{"dev", "staging", "prod"},
// }

// Mappings - define CloudFormation mappings here
// Example:
// var RegionConfig = Mapping{
//     "us-east-1": {"AMI": "ami-12345678"},
//     "us-west-2": {"AMI": "ami-87654321"},
// }

// Conditions - define CloudFormation conditions here
// Example:
// var IsProd = Equals{Ref{"Environment"}, "prod"}

// Placeholder to prevent unused import error
var _ = Parameter{}
`
		paramsGoPath := filepath.Join(infraDir, "params.go")
		if err := os.WriteFile(paramsGoPath, []byte(paramsGo), 0644); err != nil {
			result.Error = fmt.Sprintf("writing params.go: %v", err)
			return toJSON(result)
		}
		result.Files = append(result.Files, "infra/params.go")

		// Write infra/outputs.go for CloudFormation Outputs
		outputsGo := `package infra

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
)

// Outputs - define CloudFormation outputs here
// Example:
// var BucketNameOutput = Output{
//     Description: "Name of the S3 bucket",
//     Value:       MyBucket,  // Direct reference to resource
//     Export: &struct{ Name string }{
//         Name: Sub("${AWS::StackName}-BucketName"),
//     },
// }

// Placeholder to prevent unused import error
var _ = Output{}
`
		outputsGoPath := filepath.Join(infraDir, "outputs.go")
		if err := os.WriteFile(outputsGoPath, []byte(outputsGo), 0644); err != nil {
			result.Error = fmt.Sprintf("writing outputs.go: %v", err)
			return toJSON(result)
		}
		result.Files = append(result.Files, "infra/outputs.go")

		// Write .gitignore
		gitignore := `# Build output
template.json
template.yaml

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`
		gitignorePath := filepath.Join(path, ".gitignore")
		if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
			result.Error = fmt.Sprintf("writing .gitignore: %v", err)
			return toJSON(result)
		}
		result.Files = append(result.Files, ".gitignore")

		result.Success = true
		return toJSON(result)
	}
}

// makeDomainBuildHandler creates an MCP handler from the domain's Builder
func makeDomainBuildHandler(d *domain.AwsDomain) coremcp.ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		pkg, _ := args["package"].(string)
		format, _ := args["format"].(string)

		result := MCPBuildResult{}

		if pkg == "" {
			pkg = "./..."
		}

		if format == "" {
			format = "json"
		}

		// Use the domain's Builder implementation
		builder := d.Builder()
		domainCtx := coredomain.NewContext(ctx, pkg)
		opts := coredomain.BuildOpts{
			Format: format,
		}

		// Build the template
		buildResult, err := builder.Build(domainCtx, pkg, opts)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			return toJSON(result)
		}

		// Check if build was successful
		if !buildResult.Success {
			for _, e := range buildResult.Errors {
				result.Errors = append(result.Errors, e.Message)
			}
			return toJSON(result)
		}

		// Get the template from the result data
		if template, ok := buildResult.Data.(string); ok {
			result.Template = template
		}

		// Get resource list
		discoverResult, err := discover.Discover(discover.Options{
			Packages: []string{pkg},
		})
		if err == nil {
			for name := range discoverResult.Resources {
				result.Resources = append(result.Resources, name)
			}
		}

		result.Success = true
		return toJSON(result)
	}
}

// makeDomainLintHandler creates an MCP handler from the domain's Linter
func makeDomainLintHandler(d *domain.AwsDomain) coremcp.ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		pkg, _ := args["package"].(string)

		result := MCPLintResult{}

		if pkg == "" {
			pkg = "./..."
		}

		// Use the domain's Linter implementation
		linter := d.Linter()
		domainCtx := coredomain.NewContext(ctx, pkg)
		lintResult, err := linter.Lint(domainCtx, pkg, coredomain.LintOpts{})
		if err != nil {
			result.Issues = append(result.Issues, MCPLintIssue{
				Severity: "error",
				Message:  fmt.Sprintf("lint failed: %v", err),
				RuleID:   "internal",
			})
			return toJSON(result)
		}

		// Convert errors to MCP format
		for _, issue := range lintResult.Errors {
			result.Issues = append(result.Issues, MCPLintIssue{
				Severity: issue.Severity,
				Message:  issue.Message,
				RuleID:   issue.Code,
				File:     issue.Path,
				Line:     issue.Line,
				Column:   issue.Column,
			})
		}

		result.Success = lintResult.Success
		return toJSON(result)
	}
}

// makeDomainValidateHandler creates an MCP handler from the domain's Validator
func makeDomainValidateHandler(d *domain.AwsDomain) coremcp.ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		path, _ := args["path"].(string)

		result := MCPValidateResult{}

		if path == "" {
			result.Error = "path is required"
			return toJSON(result)
		}

		// Check if file exists
		if _, err := os.Stat(path); err != nil {
			result.Error = fmt.Sprintf("file not found: %s", path)
			return toJSON(result)
		}

		// Use the domain's Validator implementation
		validator := d.Validator()
		domainCtx := coredomain.NewContext(ctx, path)
		validateResult, err := validator.Validate(domainCtx, path, coredomain.ValidateOpts{})
		if err != nil {
			result.Error = fmt.Sprintf("validation failed: %v", err)
			return toJSON(result)
		}

		result.Success = validateResult.Success
		if !validateResult.Success {
			result.Error = fmt.Sprintf("found %d validation errors", len(validateResult.Errors))
		} else {
			result.Message = validateResult.Message
		}

		return toJSON(result)
	}
}

// makeDomainImportHandler creates an MCP handler from the domain's Importer
func makeDomainImportHandler(d *domain.AwsDomain) coremcp.ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		files, _ := args["files"].([]any)
		output, _ := args["output"].(string)

		result := MCPImportResult{}

		if len(files) == 0 {
			result.Error = "files is required"
			return toJSON(result)
		}

		if output == "" {
			output = "."
		}

		// Use the domain's Importer implementation
		importer := d.Importer()

		for _, f := range files {
			filePath, ok := f.(string)
			if !ok {
				continue
			}

			// Generate output file name
			outFile := filepath.Join(output, filepath.Base(filePath)+".go")

			domainCtx := coredomain.NewContext(ctx, output)
			importResult, err := importer.Import(domainCtx, filePath, coredomain.ImportOpts{
				Target: outFile,
			})
			if err != nil {
				result.Error = fmt.Sprintf("importing %s: %v", filePath, err)
				return toJSON(result)
			}
			if !importResult.Success {
				result.Error = fmt.Sprintf("importing %s: %s", filePath, importResult.Message)
				return toJSON(result)
			}
			result.Files = append(result.Files, outFile)
		}

		result.Success = true
		return toJSON(result)
	}
}

// makeDomainListHandler creates an MCP handler from the domain's Lister
func makeDomainListHandler(d *domain.AwsDomain) coremcp.ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		pkg, _ := args["package"].(string)

		result := MCPListResult{}

		if pkg == "" {
			pkg = "./..."
		}

		// Use the domain's Lister implementation
		lister := d.Lister()
		domainCtx := coredomain.NewContext(ctx, pkg)
		listResult, err := lister.List(domainCtx, pkg, coredomain.ListOpts{})
		if err != nil {
			result.Error = fmt.Sprintf("list failed: %v", err)
			return toJSON(result)
		}

		// Extract resources from result data
		if resources, ok := listResult.Data.([]map[string]string); ok {
			for _, res := range resources {
				result.Resources = append(result.Resources, MCPResourceInfo{
					Name: res["name"],
					Type: res["type"],
				})
			}
		} else {
			// Fallback to discover for detailed info
			discoverResult, err := discover.Discover(discover.Options{
				Packages: []string{pkg},
			})
			if err != nil {
				result.Error = fmt.Sprintf("discovery failed: %v", err)
				return toJSON(result)
			}

			for name, info := range discoverResult.Resources {
				result.Resources = append(result.Resources, MCPResourceInfo{
					Name: name,
					Type: info.Type,
					File: info.File,
				})
			}
		}

		result.Success = true
		return toJSON(result)
	}
}

// makeDomainGraphHandler creates an MCP handler from the domain's Grapher
func makeDomainGraphHandler(d *domain.AwsDomain) coremcp.ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		pkg, _ := args["package"].(string)
		format, _ := args["format"].(string)

		result := MCPGraphResult{}

		if pkg == "" {
			pkg = "./..."
		}

		if format == "" {
			format = "dot"
		}

		// Use the domain's Grapher implementation
		grapher := d.Grapher()
		domainCtx := coredomain.NewContext(ctx, pkg)
		graphResult, err := grapher.Graph(domainCtx, pkg, coredomain.GraphOpts{
			Format: format,
		})
		if err != nil {
			result.Error = fmt.Sprintf("graph failed: %v", err)
			return toJSON(result)
		}

		if !graphResult.Success {
			result.Error = graphResult.Message
			return toJSON(result)
		}

		// Get the graph from the result data
		if graph, ok := graphResult.Data.(string); ok {
			result.Graph = graph
		}

		result.Success = true
		return toJSON(result)
	}
}

// Result types

// MCPInitResult is the result of the wetwire_init tool.
type MCPInitResult struct {
	Success bool     `json:"success"`
	Path    string   `json:"path"`
	Files   []string `json:"files"`
	Error   string   `json:"error,omitempty"`
}

// MCPBuildResult is the result of the wetwire_build tool.
type MCPBuildResult struct {
	Success   bool     `json:"success"`
	Template  string   `json:"template,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// MCPLintResult is the result of the wetwire_lint tool.
type MCPLintResult struct {
	Success bool           `json:"success"`
	Issues  []MCPLintIssue `json:"issues,omitempty"`
}

// MCPLintIssue represents a single lint issue.
type MCPLintIssue struct {
	RuleID   string `json:"rule_id"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`
}

// MCPValidateResult is the result of the wetwire_validate tool.
type MCPValidateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// MCPImportResult is the result of the wetwire_import tool.
type MCPImportResult struct {
	Success bool     `json:"success"`
	Files   []string `json:"files,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// MCPListResult is the result of the wetwire_list tool.
type MCPListResult struct {
	Success   bool              `json:"success"`
	Resources []MCPResourceInfo `json:"resources,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// MCPResourceInfo describes a discovered resource.
type MCPResourceInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
	File string `json:"file"`
}

// MCPGraphResult is the result of the wetwire_graph tool.
type MCPGraphResult struct {
	Success bool   `json:"success"`
	Graph   string `json:"graph,omitempty"`
	Error   string `json:"error,omitempty"`
}

// toJSON converts a value to a JSON string.
func toJSON(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling result: %w", err)
	}
	return string(data), nil
}
