// Command wetwire-aws-mcp is an MCP server that exposes wetwire-aws tools.
//
// This server implements the Model Context Protocol (MCP) using infrastructure
// from github.com/lex00/wetwire-core-go/mcp and provides the following tools:
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
//	wetwire-aws-mcp  # Runs on stdio transport
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lex00/wetwire-core-go/mcp"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/linter"
	"github.com/lex00/wetwire-aws-go/internal/runner"
	"github.com/lex00/wetwire-aws-go/internal/template"
)

func main() {
	server := mcp.NewServer(mcp.Config{
		Name:    "wetwire-aws",
		Version: "1.0.0",
	})

	// Register standard wetwire tools using core infrastructure
	server.RegisterToolWithSchema("wetwire_init", "Initialize a new wetwire-aws project with example code", handleInit, initSchema)
	server.RegisterToolWithSchema("wetwire_build", "Generate CloudFormation output from wetwire declarations", handleBuild, buildSchema)
	server.RegisterToolWithSchema("wetwire_lint", "Check code quality and style (domain lint rules)", handleLint, lintSchema)
	server.RegisterToolWithSchema("wetwire_validate", "Validate generated output using external validator", handleValidate, validateSchema)
	server.RegisterToolWithSchema("wetwire_import", "Convert existing CloudFormation configs to wetwire code", handleImport, importSchema)
	server.RegisterToolWithSchema("wetwire_list", "List discovered resources", handleList, listSchema)
	server.RegisterToolWithSchema("wetwire_graph", "Visualize resource dependencies (DOT/Mermaid)", handleGraph, graphSchema)

	// Run on stdio transport
	if err := server.Start(context.Background()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Tool input schemas (aligned with wetwire-core-go/mcp standard schemas)
var initSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"name": map[string]any{
			"type":        "string",
			"description": "Project name",
		},
		"path": map[string]any{
			"type":        "string",
			"description": "Output directory (default: current directory)",
		},
	},
}

var buildSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"package": map[string]any{
			"type":        "string",
			"description": "Package path to discover resources from",
		},
		"output": map[string]any{
			"type":        "string",
			"description": "Output directory for generated files",
		},
		"format": map[string]any{
			"type":        "string",
			"enum":        []string{"yaml", "json"},
			"description": "Output format (default: yaml)",
		},
		"dry_run": map[string]any{
			"type":        "boolean",
			"description": "Return content without writing files",
		},
	},
}

var lintSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"package": map[string]any{
			"type":        "string",
			"description": "Package path to lint",
		},
		"fix": map[string]any{
			"type":        "boolean",
			"description": "Automatically fix fixable issues",
		},
		"format": map[string]any{
			"type":        "string",
			"enum":        []string{"text", "json"},
			"description": "Output format (default: text)",
		},
	},
}

var validateSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "Path to file or directory to validate",
		},
		"format": map[string]any{
			"type":        "string",
			"enum":        []string{"text", "json"},
			"description": "Output format (default: text)",
		},
	},
}

var importSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"files": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "Files to import",
		},
		"output": map[string]any{
			"type":        "string",
			"description": "Output directory for generated code",
		},
		"single_file": map[string]any{
			"type":        "boolean",
			"description": "Generate all code in a single file",
		},
	},
}

var listSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"package": map[string]any{
			"type":        "string",
			"description": "Package path to discover from",
		},
		"format": map[string]any{
			"type":        "string",
			"enum":        []string{"table", "json"},
			"description": "Output format (default: table)",
		},
	},
}

var graphSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"package": map[string]any{
			"type":        "string",
			"description": "Package path to analyze",
		},
		"format": map[string]any{
			"type":        "string",
			"enum":        []string{"dot", "mermaid"},
			"description": "Output format (default: mermaid)",
		},
		"output": map[string]any{
			"type":        "string",
			"description": "Output file path",
		},
	},
}

// handleInit initializes a new wetwire-aws project.
func handleInit(_ context.Context, args map[string]any) (string, error) {
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

	result := InitResult{Path: path}

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

// handleBuild generates CloudFormation template from Go packages.
func handleBuild(_ context.Context, args map[string]any) (string, error) {
	pkg, _ := args["package"].(string)
	output, _ := args["output"].(string)
	format, _ := args["format"].(string)
	dryRun, _ := args["dry_run"].(bool)

	result := BuildResult{}

	if pkg == "" {
		pkg = "./..."
	}

	if format == "" {
		format = "yaml"
	}
	if format != "json" && format != "yaml" {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid format: %s (use json or yaml)", format))
		return toJSON(result)
	}

	packages := []string{pkg}

	// Discover resources and other template components
	discoverResult, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("discovery failed: %v", err))
		return toJSON(result)
	}

	// Check for discovery errors
	if len(discoverResult.Errors) > 0 {
		for _, e := range discoverResult.Errors {
			result.Errors = append(result.Errors, e.Error())
		}
		return toJSON(result)
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
		return toJSON(result)
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
		return toJSON(result)
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
		return toJSON(result)
	}

	// Build success result
	for name := range discoverResult.Resources {
		result.Resources = append(result.Resources, name)
	}

	if dryRun {
		result.Template = string(data)
	} else if output != "" {
		// Write to file
		if err := os.WriteFile(output, data, 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("writing output: %v", err))
			return toJSON(result)
		}
		result.Template = fmt.Sprintf("Written to %s", output)
	} else {
		result.Template = string(data)
	}

	result.Success = true
	return toJSON(result)
}

// handleLint lints Go packages for wetwire-aws style issues.
func handleLint(_ context.Context, args map[string]any) (string, error) {
	pkg, _ := args["package"].(string)

	result := wetwire.LintResult{}

	if pkg == "" {
		pkg = "./..."
	}

	packages := []string{pkg}

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
		return toJSON(result)
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
	for _, p := range packages {
		lintResult, err := linter.LintPackage(p, linter.Options{})
		if err != nil {
			result.Issues = append(result.Issues, wetwire.LintIssue{
				Severity: "warning",
				Message:  fmt.Sprintf("failed to lint %s: %v", p, err),
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
	return toJSON(result)
}

// handleValidate validates generated CloudFormation templates.
func handleValidate(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)

	result := ValidateResult{}

	if path == "" {
		result.Error = "path is required"
		return toJSON(result)
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		result.Error = fmt.Sprintf("file not found: %s", path)
		return toJSON(result)
	}

	// TODO: Integrate with cfn-lint or AWS CloudFormation validation
	result.Success = true
	result.Message = "Validation not yet implemented - template file exists"
	return toJSON(result)
}

// handleImport imports existing CloudFormation templates to Go code.
func handleImport(_ context.Context, args map[string]any) (string, error) {
	files, _ := args["files"].([]any)
	output, _ := args["output"].(string)

	result := ImportResult{}

	if len(files) == 0 {
		result.Error = "files is required"
		return toJSON(result)
	}

	if output == "" {
		output = "."
	}

	// TODO: Implement CloudFormation to Go import
	result.Success = false
	result.Error = "import not yet implemented"
	return toJSON(result)
}

// handleList lists discovered resources.
func handleList(_ context.Context, args map[string]any) (string, error) {
	pkg, _ := args["package"].(string)

	result := ListResult{}

	if pkg == "" {
		pkg = "./..."
	}

	packages := []string{pkg}

	// Discover resources
	discoverResult, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		result.Error = fmt.Sprintf("discovery failed: %v", err)
		return toJSON(result)
	}

	for name, info := range discoverResult.Resources {
		result.Resources = append(result.Resources, ResourceInfo{
			Name: name,
			Type: info.Type,
			File: info.File,
		})
	}

	result.Success = true
	return toJSON(result)
}

// handleGraph visualizes resource dependencies.
func handleGraph(_ context.Context, args map[string]any) (string, error) {
	pkg, _ := args["package"].(string)
	format, _ := args["format"].(string)

	result := GraphResult{}

	if pkg == "" {
		pkg = "./..."
	}

	if format == "" {
		format = "mermaid"
	}

	// TODO: Implement dependency graph generation
	result.Success = false
	result.Error = "graph not yet implemented"
	return toJSON(result)
}

// Result types

// InitResult is the result of the wetwire_init tool.
type InitResult struct {
	Success bool     `json:"success"`
	Path    string   `json:"path"`
	Files   []string `json:"files"`
	Error   string   `json:"error,omitempty"`
}

// BuildResult is the result of the wetwire_build tool.
type BuildResult struct {
	Success   bool     `json:"success"`
	Template  string   `json:"template,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// ValidateResult is the result of the wetwire_validate tool.
type ValidateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ImportResult is the result of the wetwire_import tool.
type ImportResult struct {
	Success bool     `json:"success"`
	Files   []string `json:"files,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// ListResult is the result of the wetwire_list tool.
type ListResult struct {
	Success   bool           `json:"success"`
	Resources []ResourceInfo `json:"resources,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// ResourceInfo describes a discovered resource.
type ResourceInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
	File string `json:"file"`
}

// GraphResult is the result of the wetwire_graph tool.
type GraphResult struct {
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
