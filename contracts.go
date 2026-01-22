// Package wetwire_aws provides Go types for AWS CloudFormation resources.
//
// This package enables declarative infrastructure-as-code using native Go syntax:
//
//	var MyBucket = s3.Bucket{
//	    BucketName: "my-bucket",
//	}
//
//	var MyFunction = lambda.Function{
//	    FunctionName: "processor",
//	    Role:         MyRole.Arn,  // GetAtt reference
//	}
//
// The wetwire-aws CLI discovers these declarations via AST parsing and generates
// CloudFormation templates.
package wetwire_aws

import (
	"encoding/json"
)

// Resource represents a CloudFormation resource.
// All generated resource types (s3.Bucket, iam.Role, etc.) implement this interface.
type Resource interface {
	// ResourceType returns the CloudFormation type (e.g., "AWS::S3::Bucket")
	ResourceType() string
}

// AttrRef represents a GetAtt reference to a resource attribute.
// Generated resource types have AttrRef fields for each supported attribute.
//
// Example:
//
//	var MyRole = iam.Role{...}
//	var MyFunction = lambda.Function{
//	    Role: MyRole.Arn,  // MyRole.Arn is an AttrRef
//	}
//
// When serialized to CloudFormation JSON, AttrRef becomes:
//
//	{"Fn::GetAtt": ["MyRole", "Arn"]}
type AttrRef struct {
	// Resource is the logical name of the referenced resource
	Resource string
	// Attribute is the attribute name (e.g., "Arn", "DomainName")
	Attribute string
}

// MarshalJSON serializes AttrRef to CloudFormation GetAtt syntax.
func (a AttrRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string][]string{
		"Fn::GetAtt": {a.Resource, a.Attribute},
	})
}

// IsZero returns true if the AttrRef has not been populated.
func (a AttrRef) IsZero() bool {
	return a.Resource == "" && a.Attribute == ""
}

// DiscoveredResource represents a resource found by AST parsing.
// The CLI builds a map of these from user source files.
type DiscoveredResource struct {
	// Name is the variable name (becomes CloudFormation logical ID)
	Name string
	// Type is the Go type (e.g., "s3.Bucket", "iam.Role")
	Type string
	// Package is the full package path containing the declaration
	Package string
	// File is the source file path
	File string
	// Line is the line number of the declaration
	Line int
	// Dependencies are logical names of referenced resources
	Dependencies []string
	// AttrRefUsages tracks Resource.Attr patterns used in this resource's properties
	AttrRefUsages []AttrRefUsage
}

// AttrRefUsage tracks a Resource.Attr field access pattern for GetAtt resolution.
type AttrRefUsage struct {
	// ResourceName is the referenced resource (e.g., "LambdaRole")
	ResourceName string
	// Attribute is the attribute name (e.g., "Arn")
	Attribute string
	// FieldPath is where this usage appears in the resource (e.g., "Role")
	FieldPath string
}

// DiscoveredParameter represents a parameter found by AST parsing.
type DiscoveredParameter struct {
	// Name is the variable name (becomes CloudFormation parameter name)
	Name string
	// File is the source file path
	File string
	// Line is the line number of the declaration
	Line int
}

// DiscoveredOutput represents an output found by AST parsing.
type DiscoveredOutput struct {
	// Name is the variable name (becomes CloudFormation output name)
	Name string
	// File is the source file path
	File string
	// Line is the line number of the declaration
	Line int
	// AttrRefUsages tracks Resource.Attr patterns used in this output's properties
	AttrRefUsages []AttrRefUsage
}

// DiscoveredMapping represents a mapping found by AST parsing.
type DiscoveredMapping struct {
	// Name is the variable name (becomes CloudFormation mapping name)
	Name string
	// File is the source file path
	File string
	// Line is the line number of the declaration
	Line int
}

// DiscoveredCondition represents a condition found by AST parsing.
type DiscoveredCondition struct {
	// Name is the variable name (becomes CloudFormation condition name)
	Name string
	// Type is the condition type (e.g., "Equals", "And", "Or", "Not")
	Type string
	// File is the source file path
	File string
	// Line is the line number of the declaration
	Line int
}

// Template represents a CloudFormation template.
type Template struct {
	AWSTemplateFormatVersion string                 `json:"AWSTemplateFormatVersion" yaml:"AWSTemplateFormatVersion"`
	Transform                string                 `json:"Transform,omitempty" yaml:"Transform,omitempty"`
	Description              string                 `json:"Description,omitempty" yaml:"Description,omitempty"`
	Parameters               map[string]Parameter   `json:"Parameters,omitempty" yaml:"Parameters,omitempty"`
	Mappings                 map[string]any         `json:"Mappings,omitempty" yaml:"Mappings,omitempty"`
	Conditions               map[string]any         `json:"Conditions,omitempty" yaml:"Conditions,omitempty"`
	Resources                map[string]ResourceDef `json:"Resources" yaml:"Resources"`
	Outputs                  map[string]Output      `json:"Outputs,omitempty" yaml:"Outputs,omitempty"`
}

// ResourceDef is a single resource in the CloudFormation template.
type ResourceDef struct {
	Type       string         `json:"Type" yaml:"Type"`
	Properties map[string]any `json:"Properties,omitempty" yaml:"Properties,omitempty"`
	DependsOn  []string       `json:"DependsOn,omitempty" yaml:"DependsOn,omitempty"`
}

// Parameter is a CloudFormation template parameter for output serialization.
type Parameter struct {
	Type                  string   `json:"Type"`
	Description           string   `json:"Description,omitempty"`
	Default               any      `json:"Default,omitempty"`
	AllowedValues         []any    `json:"AllowedValues,omitempty"`
	AllowedPattern        string   `json:"AllowedPattern,omitempty"`
	ConstraintDescription string   `json:"ConstraintDescription,omitempty"`
	MinLength             *int     `json:"MinLength,omitempty"`
	MaxLength             *int     `json:"MaxLength,omitempty"`
	MinValue              *float64 `json:"MinValue,omitempty"`
	MaxValue              *float64 `json:"MaxValue,omitempty"`
	NoEcho                bool     `json:"NoEcho,omitempty"`
}

// Output is a CloudFormation template output.
type Output struct {
	Description string `json:"Description,omitempty"`
	Value       any    `json:"Value"`
	Export      *struct {
		Name string `json:"Name"`
	} `json:"Export,omitempty"`
}

// BuildResult is the JSON output from `wetwire-aws build`.
type BuildResult struct {
	Success   bool     `json:"success"`
	Template  Template `json:"template,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// LintResult is the JSON output from `wetwire-aws lint`.
type LintResult struct {
	Success bool        `json:"success"`
	Issues  []LintIssue `json:"issues,omitempty"`
}

// LintIssue is a single linting issue.
type LintIssue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"` // "error", "warning", "info"
	Message  string `json:"message"`
	Rule     string `json:"rule"`
	Fixable  bool   `json:"fixable"`
}

// ValidateResult is the JSON output from `wetwire-aws validate`.
type ValidateResult struct {
	Success   bool     `json:"success"`
	Resources int      `json:"resources"`
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

// ListResult is the JSON output from `wetwire-aws list`.
type ListResult struct {
	Resources []ListResource `json:"resources"`
}

// ListResource is a single resource in the list output.
type ListResource struct {
	Name string `json:"name"`
	Type string `json:"type"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// OptimizeResult is the JSON output from `wetwire-aws optimize`.
type OptimizeResult struct {
	Success       bool                 `json:"success"`
	Suggestions   []OptimizeSuggestion `json:"suggestions,omitempty"`
	ResourceCount int                  `json:"resource_count"`
	Summary       OptimizeSummary      `json:"summary"`
}

// OptimizeSuggestion is a single optimization suggestion.
type OptimizeSuggestion struct {
	Resource    string `json:"resource"`
	Category    string `json:"category"` // "security", "cost", "performance", "reliability"
	Severity    string `json:"severity"` // "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
}

// OptimizeSummary provides counts by category.
type OptimizeSummary struct {
	Security    int `json:"security"`
	Cost        int `json:"cost"`
	Performance int `json:"performance"`
	Reliability int `json:"reliability"`
	Total       int `json:"total"`
}

// DiffResult is the JSON output from `wetwire-aws diff`.
type DiffResult struct {
	Success bool         `json:"success"`
	Diff    TemplateDiff `json:"diff"`
	Summary DiffSummary  `json:"summary"`
}

// TemplateDiff represents the differences between two templates.
type TemplateDiff struct {
	Added    []DiffEntry `json:"added,omitempty"`
	Removed  []DiffEntry `json:"removed,omitempty"`
	Modified []DiffEntry `json:"modified,omitempty"`
}

// DiffEntry represents a single difference.
type DiffEntry struct {
	Resource string   `json:"resource"`
	Type     string   `json:"type"`
	Changes  []string `json:"changes,omitempty"`
}

// DiffSummary provides counts of changes.
type DiffSummary struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Modified int `json:"modified"`
	Total    int `json:"total"`
}

// SchemaResult is the JSON output from `wetwire-aws schema`.
type SchemaResult struct {
	Success   bool          `json:"success"`
	Valid     bool          `json:"valid"`
	Errors    []SchemaError `json:"errors,omitempty"`
	Warnings  []SchemaError `json:"warnings,omitempty"`
	Resources int           `json:"resources"`
}

// SchemaError represents a schema validation error.
type SchemaError struct {
	Resource string `json:"resource"`
	Property string `json:"property"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// TestScore represents the 5-dimension scoring system per spec 6.4.
// Each dimension is scored 0-3, total possible score is 15.
type TestScore struct {
	// Completeness: Were all required resources generated? (0-3)
	Completeness int `json:"completeness"`
	// LintQuality: How many lint cycles needed? (0-3, fewer = better)
	LintQuality int `json:"lint_quality"`
	// CodeQuality: Does code follow idiomatic patterns? (0-3)
	CodeQuality int `json:"code_quality"`
	// OutputValidity: Does generated output validate? (0-3)
	OutputValidity int `json:"output_validity"`
	// QuestionEfficiency: Appropriate number of clarifying questions? (0-3)
	QuestionEfficiency int `json:"question_efficiency"`
	// Total: Sum of all dimensions (0-15)
	Total int `json:"total"`
	// Grade: Letter grade based on total score
	Grade string `json:"grade"`
}

// TestResult is the JSON output from `wetwire-aws test`.
type TestResult struct {
	Success        bool           `json:"success"`
	Persona        string         `json:"persona"`
	Score          TestScore      `json:"score"`
	Duration       string         `json:"duration"`
	LintCycles     int            `json:"lint_cycles"`
	BuildPassed    bool           `json:"build_passed"`
	ValidatePassed bool           `json:"validate_passed"`
	FilesCreated   []string       `json:"files_created,omitempty"`
	Errors         []string       `json:"errors,omitempty"`
	Details        map[string]any `json:"details,omitempty"`
}

// TestSummary aggregates results from multiple test runs.
type TestSummary struct {
	TotalRuns    int          `json:"total_runs"`
	PassedRuns   int          `json:"passed_runs"`
	FailedRuns   int          `json:"failed_runs"`
	AverageScore float64      `json:"average_score"`
	Results      []TestResult `json:"results"`
}
