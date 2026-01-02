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
