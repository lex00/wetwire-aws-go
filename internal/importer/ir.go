// Package importer provides CloudFormation template import functionality.
// It parses CloudFormation YAML/JSON templates and generates Go code
// using wetwire-aws patterns.
//
// This package uses the shared template parsing from cloudformation-schema-go.
package importer

import (
	"github.com/lex00/cloudformation-schema-go/template"
)

// Re-export intrinsic types from shared package for backwards compatibility.
// This allows existing code to use importer.IntrinsicRef, importer.IRIntrinsic, etc.
type IntrinsicType = template.IntrinsicType

// IRIntrinsic is an alias for the shared Intrinsic type.
type IRIntrinsic = template.Intrinsic

// Re-export intrinsic constants for backwards compatibility.
const (
	IntrinsicRef         = template.IntrinsicRef
	IntrinsicGetAtt      = template.IntrinsicGetAtt
	IntrinsicSub         = template.IntrinsicSub
	IntrinsicJoin        = template.IntrinsicJoin
	IntrinsicSelect      = template.IntrinsicSelect
	IntrinsicGetAZs      = template.IntrinsicGetAZs
	IntrinsicIf          = template.IntrinsicIf
	IntrinsicEquals      = template.IntrinsicEquals
	IntrinsicAnd         = template.IntrinsicAnd
	IntrinsicOr          = template.IntrinsicOr
	IntrinsicNot         = template.IntrinsicNot
	IntrinsicCondition   = template.IntrinsicCondition
	IntrinsicFindInMap   = template.IntrinsicFindInMap
	IntrinsicBase64      = template.IntrinsicBase64
	IntrinsicCidr        = template.IntrinsicCidr
	IntrinsicImportValue = template.IntrinsicImportValue
	IntrinsicSplit       = template.IntrinsicSplit
	IntrinsicTransform   = template.IntrinsicTransform
	IntrinsicValueOf     = template.IntrinsicValueOf
)

// IRProperty represents a resource property key-value pair.
// This extends the shared Property type with GoName for code generation.
type IRProperty struct {
	DomainName string // Original CloudFormation name (e.g., "BucketName")
	GoName     string // Go field name (e.g., "BucketName" or "Type_" for keywords)
	Value      any    // Parsed value (may contain *IRIntrinsic)
}

// IRParameter represents a CloudFormation parameter.
type IRParameter struct {
	LogicalID             string
	Type                  string
	Description           string
	Default               any
	AllowedValues         []any
	AllowedPattern        string
	MinLength             *int
	MaxLength             *int
	MinValue              *float64
	MaxValue              *float64
	ConstraintDescription string
	NoEcho                bool
}

// IRResource represents a CloudFormation resource.
type IRResource struct {
	LogicalID           string
	ResourceType        string // e.g., "AWS::S3::Bucket"
	Properties          map[string]*IRProperty
	DependsOn           []string
	Condition           string
	DeletionPolicy      string
	UpdateReplacePolicy string
	Metadata            map[string]any
}

// Service returns the AWS service name (e.g., "S3" from "AWS::S3::Bucket").
func (r *IRResource) Service() string {
	parts := splitResourceType(r.ResourceType)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// TypeName returns the resource type name (e.g., "Bucket" from "AWS::S3::Bucket").
func (r *IRResource) TypeName() string {
	parts := splitResourceType(r.ResourceType)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func splitResourceType(rt string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(rt); i++ {
		if rt[i] == ':' && i+1 < len(rt) && rt[i+1] == ':' {
			parts = append(parts, rt[start:i])
			start = i + 2
			i++
		}
	}
	if start < len(rt) {
		parts = append(parts, rt[start:])
	}
	return parts
}

// IROutput represents a CloudFormation output.
type IROutput struct {
	LogicalID   string
	Value       any
	Description string
	ExportName  any // May be string or *IRIntrinsic
	Condition   string
}

// IRMapping represents a CloudFormation mapping table.
type IRMapping struct {
	LogicalID string
	MapData   map[string]map[string]any
}

// IRCondition represents a CloudFormation condition.
type IRCondition struct {
	LogicalID  string
	Expression any // Usually an *IRIntrinsic
}

// IRTemplate represents a complete parsed CloudFormation template.
type IRTemplate struct {
	Description              string
	AWSTemplateFormatVersion string
	Parameters               map[string]*IRParameter
	Mappings                 map[string]*IRMapping
	Conditions               map[string]*IRCondition
	Resources                map[string]*IRResource
	Outputs                  map[string]*IROutput
	SourceFile               string
	ReferenceGraph           map[string][]string // resource -> list of resources it references
}

// NewIRTemplate creates a new empty IR template.
func NewIRTemplate() *IRTemplate {
	return &IRTemplate{
		AWSTemplateFormatVersion: "2010-09-09",
		Parameters:               make(map[string]*IRParameter),
		Mappings:                 make(map[string]*IRMapping),
		Conditions:               make(map[string]*IRCondition),
		Resources:                make(map[string]*IRResource),
		Outputs:                  make(map[string]*IROutput),
		ReferenceGraph:           make(map[string][]string),
	}
}
