// Command wetwire-aws-mcp is an MCP server that exposes wetwire-aws tools.
//
// This server implements the Model Context Protocol (MCP) and provides
// the following tools:
//   - wetwire_init: Initialize a new wetwire-aws project
//   - wetwire_lint: Lint Go packages for wetwire-aws issues
//   - wetwire_build: Generate CloudFormation template from Go packages
//
// Usage:
//
//	wetwire-aws-mcp  # Runs on stdio transport
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/linter"
	"github.com/lex00/wetwire-aws-go/internal/runner"
	"github.com/lex00/wetwire-aws-go/internal/template"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "wetwire-aws",
		Version: "1.0.0",
	}, nil)

	// Register tools
	registerInitTool(server)
	registerLintTool(server)
	registerBuildTool(server)

	// Run on stdio transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// InitArgs are the arguments for the wetwire_init tool.
type InitArgs struct {
	Path       string `json:"path" jsonschema:"Path to create the project at"`
	ModuleName string `json:"module_name,omitempty" jsonschema:"Go module name (defaults to path basename)"`
}

// InitResult is the result of the wetwire_init tool.
type InitResult struct {
	Success bool     `json:"success"`
	Path    string   `json:"path"`
	Files   []string `json:"files"`
	Error   string   `json:"error,omitempty"`
}

func registerInitTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "wetwire_init",
		Description: "Initialize a new wetwire-aws project with Go module and example resources",
	}, handleInit)
}

func handleInit(_ context.Context, _ *mcp.CallToolRequest, args InitArgs) (*mcp.CallToolResult, any, error) {
	result := InitResult{Path: args.Path}

	if args.Path == "" {
		result.Error = "path is required"
		return toolResult(result)
	}

	// Create project directory
	if err := os.MkdirAll(args.Path, 0755); err != nil {
		result.Error = fmt.Sprintf("creating project directory: %v", err)
		return toolResult(result)
	}

	// Create infra subdirectory
	infraDir := filepath.Join(args.Path, "infra")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		result.Error = fmt.Sprintf("creating infra directory: %v", err)
		return toolResult(result)
	}

	// Get the module name
	moduleName := args.ModuleName
	if moduleName == "" {
		moduleName = filepath.Base(args.Path)
	}

	// Write go.mod
	goMod := fmt.Sprintf(`module %s

go 1.23

require github.com/lex00/wetwire-aws-go v1.2.3
`, moduleName)

	goModPath := filepath.Join(args.Path, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil {
		result.Error = fmt.Sprintf("writing go.mod: %v", err)
		return toolResult(result)
	}
	result.Files = append(result.Files, "go.mod")

	// Write main.go
	mainGo := `package main

import (
	"fmt"

	_ "` + moduleName + `/infra" // Register resources
)

func main() {
	fmt.Println("Run: wetwire-aws build ./infra/...")
}
`
	mainGoPath := filepath.Join(args.Path, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGo), 0644); err != nil {
		result.Error = fmt.Sprintf("writing main.go: %v", err)
		return toolResult(result)
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
		return toolResult(result)
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
		return toolResult(result)
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
		return toolResult(result)
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
	gitignorePath := filepath.Join(args.Path, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
		result.Error = fmt.Sprintf("writing .gitignore: %v", err)
		return toolResult(result)
	}
	result.Files = append(result.Files, ".gitignore")

	result.Success = true
	return toolResult(result)
}

// LintArgs are the arguments for the wetwire_lint tool.
type LintArgs struct {
	Path string `json:"path" jsonschema:"Path to the Go package(s) to lint (e.g. ./infra/...)"`
}

func registerLintTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "wetwire_lint",
		Description: "Lint Go packages for wetwire-aws style issues (WAW001-WAW018 rules)",
	}, handleLint)
}

func handleLint(_ context.Context, _ *mcp.CallToolRequest, args LintArgs) (*mcp.CallToolResult, any, error) {
	result := wetwire.LintResult{}

	if args.Path == "" {
		result.Issues = append(result.Issues, wetwire.LintIssue{
			Severity: "error",
			Message:  "path is required",
			Rule:     "internal",
		})
		return toolResult(result)
	}

	packages := []string{args.Path}

	// Discover resources (validates references)
	discoverResult, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		result.Issues = append(result.Issues, wetwire.LintIssue{
			Severity: "error",
			Message:  fmt.Sprintf("discovery failed: %v", err),
			Rule:     "internal",
		})
		return toolResult(result)
	}

	// Convert discovery errors to lint issues
	for _, e := range discoverResult.Errors {
		result.Issues = append(result.Issues, wetwire.LintIssue{
			Severity: "error",
			Message:  e.Error(),
			Rule:     "undefined-reference",
		})
	}

	// Run lint rules on each package
	for _, pkg := range packages {
		lintResult, err := linter.LintPackage(pkg, linter.Options{})
		if err != nil {
			result.Issues = append(result.Issues, wetwire.LintIssue{
				Severity: "warning",
				Message:  fmt.Sprintf("failed to lint %s: %v", pkg, err),
				Rule:     "internal",
			})
			continue
		}

		for _, issue := range lintResult.Issues {
			result.Issues = append(result.Issues, wetwire.LintIssue{
				Severity: issue.Severity,
				Message:  issue.Message,
				Rule:     issue.RuleID,
				File:     issue.File,
				Line:     issue.Line,
				Column:   issue.Column,
			})
		}
	}

	result.Success = len(result.Issues) == 0
	return toolResult(result)
}

// BuildArgs are the arguments for the wetwire_build tool.
type BuildArgs struct {
	Path   string `json:"path" jsonschema:"Path to the Go package(s) to build (e.g. ./infra/...)"`
	Format string `json:"format,omitempty" jsonschema:"Output format: json or yaml (default: json)"`
}

// BuildResult is the result of the wetwire_build tool.
type BuildResult struct {
	Success   bool     `json:"success"`
	Template  string   `json:"template,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

func registerBuildTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "wetwire_build",
		Description: "Generate CloudFormation template from Go packages containing wetwire-aws resources",
	}, handleBuild)
}

func handleBuild(_ context.Context, _ *mcp.CallToolRequest, args BuildArgs) (*mcp.CallToolResult, any, error) {
	result := BuildResult{}

	if args.Path == "" {
		result.Errors = append(result.Errors, "path is required")
		return toolResult(result)
	}

	format := args.Format
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "yaml" {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid format: %s (use json or yaml)", format))
		return toolResult(result)
	}

	packages := []string{args.Path}

	// Discover resources and other template components
	discoverResult, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("discovery failed: %v", err))
		return toolResult(result)
	}

	// Check for discovery errors
	if len(discoverResult.Errors) > 0 {
		for _, e := range discoverResult.Errors {
			result.Errors = append(result.Errors, e.Error())
		}
		return toolResult(result)
	}

	// Build template with all discovered components
	builder := template.NewBuilderFull(
		discoverResult.Resources,
		discoverResult.Parameters,
		discoverResult.Outputs,
		discoverResult.Mappings,
		discoverResult.Conditions,
	)

	// Set VarAttrRefs for recursive AttrRef resolution
	varAttrRefs := make(map[string]template.VarAttrRefInfo)
	for name, info := range discoverResult.VarAttrRefs {
		varAttrRefs[name] = template.VarAttrRefInfo{
			AttrRefs: info.AttrRefs,
			VarRefs:  info.VarRefs,
		}
	}
	builder.SetVarAttrRefs(varAttrRefs)

	// Extract all values by running a generated Go program
	values, err := runner.ExtractAll(
		packages[0],
		discoverResult.Resources,
		discoverResult.Parameters,
		discoverResult.Outputs,
		discoverResult.Mappings,
		discoverResult.Conditions,
	)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("extracting values: %v", err))
		return toolResult(result)
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
		result.Errors = append(result.Errors, err.Error())
		return toolResult(result)
	}

	// Serialize template
	var data []byte
	switch format {
	case "json":
		data, err = template.ToJSON(tmpl)
	case "yaml":
		data, err = template.ToYAML(tmpl)
	}
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("serializing template: %v", err))
		return toolResult(result)
	}

	// Build success result
	for name := range discoverResult.Resources {
		result.Resources = append(result.Resources, name)
	}

	result.Success = true
	result.Template = string(data)
	return toolResult(result)
}

// toolResult creates an MCP CallToolResult from any JSON-serializable value.
func toolResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}
