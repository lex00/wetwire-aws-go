// Package intrinsics provides CloudFormation intrinsic functions.
//
// This package re-exports the core intrinsic types from cloudformation-schema-go
// and adds IAM policy-specific types.
//
// Core intrinsic functions:
//
//	Ref{"MyBucket"} → {"Ref": "MyBucket"}
//	Sub{"${AWS::Region}-bucket"} → {"Fn::Sub": "${AWS::Region}-bucket"}
//	Join{",", []any{"a", "b"}} → {"Fn::Join": [",", ["a", "b"]]}
//
// Pseudo-parameters:
//
//	AWS_REGION, AWS_ACCOUNT_ID, AWS_STACK_NAME, etc.
package intrinsics

import (
	"encoding/json"

	"github.com/lex00/cloudformation-schema-go/intrinsics"
)

// Re-export core intrinsic types from shared package.
// This provides backwards compatibility for existing code.
type (
	// Ref represents a CloudFormation Ref intrinsic function.
	Ref = intrinsics.Ref

	// GetAtt represents a CloudFormation Fn::GetAtt intrinsic function.
	GetAtt = intrinsics.GetAtt

	// Sub represents a CloudFormation Fn::Sub intrinsic function.
	Sub = intrinsics.Sub

	// SubWithMap is Fn::Sub with a variable map.
	SubWithMap = intrinsics.SubWithMap

	// Join represents a CloudFormation Fn::Join intrinsic function.
	Join = intrinsics.Join

	// Select represents a CloudFormation Fn::Select intrinsic function.
	Select = intrinsics.Select

	// GetAZs represents a CloudFormation Fn::GetAZs intrinsic function.
	GetAZs = intrinsics.GetAZs

	// If represents a CloudFormation Fn::If intrinsic function.
	If = intrinsics.If

	// Equals represents a CloudFormation Fn::Equals condition function.
	Equals = intrinsics.Equals

	// And represents a CloudFormation Fn::And condition function.
	And = intrinsics.And

	// Or represents a CloudFormation Fn::Or condition function.
	Or = intrinsics.Or

	// Not represents a CloudFormation Fn::Not condition function.
	Not = intrinsics.Not

	// Base64 represents a CloudFormation Fn::Base64 intrinsic function.
	Base64 = intrinsics.Base64

	// ImportValue represents a CloudFormation Fn::ImportValue intrinsic function.
	ImportValue = intrinsics.ImportValue

	// FindInMap represents a CloudFormation Fn::FindInMap intrinsic function.
	FindInMap = intrinsics.FindInMap

	// Split represents a CloudFormation Fn::Split intrinsic function.
	Split = intrinsics.Split

	// Cidr represents a CloudFormation Fn::Cidr intrinsic function.
	Cidr = intrinsics.Cidr

	// Condition represents a CloudFormation Condition reference.
	Condition = intrinsics.Condition

	// Tag represents a CloudFormation resource tag.
	Tag = intrinsics.Tag

	// Transform represents a CloudFormation Fn::Transform intrinsic function.
	Transform = intrinsics.Transform

	// Output represents a CloudFormation stack output.
	Output = intrinsics.Output
)

// Param creates a Ref for a CloudFormation parameter.
// Re-exported from shared package.
var Param = intrinsics.Param

// Parameter defines a CloudFormation template parameter with full metadata.
// When used as a value in resource properties, it serializes to {"Ref": "ParameterName"}.
//
// Example:
//
//	var Environment = Parameter{
//	    Type:          "String",
//	    Description:   "Environment name",
//	    Default:       "dev",
//	    AllowedValues: []any{"dev", "staging", "prod"},
//	}
//
//	var MyBucket = s3.Bucket{
//	    BucketName: Environment,  // Serializes to {"Ref": "Environment"}
//	}
type Parameter struct {
	// Type is the CloudFormation parameter type (String, Number, List<Number>, etc.)
	Type string
	// Description is optional documentation for the parameter
	Description string
	// Default is the default value if none is provided
	Default any
	// AllowedValues restricts the parameter to specific values
	AllowedValues []any
	// AllowedPattern is a regex pattern for String type validation
	AllowedPattern string
	// ConstraintDescription explains validation failures
	ConstraintDescription string
	// MinLength is minimum string length (for String type)
	MinLength *int
	// MaxLength is maximum string length (for String type)
	MaxLength *int
	// MinValue is minimum numeric value (for Number type)
	MinValue *float64
	// MaxValue is maximum numeric value (for Number type)
	MaxValue *float64
	// NoEcho masks the parameter value in console/logs
	NoEcho bool

	// name is set during discovery to enable proper Ref serialization
	name string
}

// SetName sets the parameter name for Ref serialization.
// This is called by the template builder after discovery.
func (p *Parameter) SetName(name string) {
	p.name = name
}

// Name returns the parameter name.
func (p Parameter) Name() string {
	return p.name
}

// MarshalJSON serializes Parameter as a CloudFormation Ref when used as a value.
func (p Parameter) MarshalJSON() ([]byte, error) {
	if p.name == "" {
		// Fallback: serialize as empty ref (should not happen in normal use)
		return json.Marshal(map[string]string{"Ref": ""})
	}
	return json.Marshal(map[string]string{"Ref": p.name})
}

// ToDefinition returns the parameter as a map suitable for the Parameters section.
func (p Parameter) ToDefinition() map[string]any {
	def := map[string]any{
		"Type": p.Type,
	}
	if p.Description != "" {
		def["Description"] = p.Description
	}
	if p.Default != nil {
		def["Default"] = p.Default
	}
	if len(p.AllowedValues) > 0 {
		def["AllowedValues"] = p.AllowedValues
	}
	if p.AllowedPattern != "" {
		def["AllowedPattern"] = p.AllowedPattern
	}
	if p.ConstraintDescription != "" {
		def["ConstraintDescription"] = p.ConstraintDescription
	}
	if p.MinLength != nil {
		def["MinLength"] = *p.MinLength
	}
	if p.MaxLength != nil {
		def["MaxLength"] = *p.MaxLength
	}
	if p.MinValue != nil {
		def["MinValue"] = *p.MinValue
	}
	if p.MaxValue != nil {
		def["MaxValue"] = *p.MaxValue
	}
	if p.NoEcho {
		def["NoEcho"] = true
	}
	return def
}

// Mapping represents a CloudFormation Mappings table.
// It maps a top-level key to a second-level key to values.
//
// Example:
//
//	var RegionAMI = Mapping{
//	    "us-east-1": {"AMI": "ami-12345"},
//	    "us-west-2": {"AMI": "ami-67890"},
//	}
type Mapping map[string]map[string]any

// Helper functions for creating pointers to primitive types.
// These are used in generated code for optional parameter fields.

// IntPtr returns a pointer to the given int value.
func IntPtr(i int) *int {
	return &i
}

// Float64Ptr returns a pointer to the given float64 value.
func Float64Ptr(f float64) *float64 {
	return &f
}
