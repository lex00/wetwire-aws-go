// Package intrinsics provides CloudFormation intrinsic functions.
// This file contains IAM policy document types and helpers.
package intrinsics

import (
	"encoding/json"
)

// Json is a shorthand for map[string]any.
// Used for inline JSON objects like Condition blocks.
//
// Example:
//
//	Condition: Json{
//	    Bool: Json{"aws:SecureTransport": false},
//	}
type Json = map[string]any

// List creates a typed slice from the given items.
// Avoids verbose slice type annotations in struct literals.
//
// Example:
//
//	// Instead of:
//	Actions: []elasticloadbalancingv2.ListenerRule_Action{ActionForward},
//	// Write:
//	Actions: List(ActionForward),
//
//	// Multiple items:
//	Origins: List(Origin1, Origin2, Origin3),
func List[T any](items ...T) []T {
	return items
}

// Any creates a []any slice from the given items.
// Use for fields typed as []any that accept mixed types or intrinsics.
//
// Example:
//
//	// Instead of:
//	SecurityGroupIds: []any{EC2InstanceSG, ALBExternalAccessSG},
//	// Write:
//	SecurityGroupIds: Any(EC2InstanceSG, ALBExternalAccessSG),
func Any(items ...any) []any {
	return items
}

// PolicyDocument represents an IAM policy document.
//
// Example:
//
//	var MyPolicy = PolicyDocument{
//	    Version:   "2012-10-17",
//	    Statement: []any{MyStatement},
//	}
type PolicyDocument struct {
	Version   string `json:"Version,omitempty"`
	Statement []any  `json:"Statement"`
}

// NewPolicyDocument creates a PolicyDocument with the default version.
func NewPolicyDocument() PolicyDocument {
	return PolicyDocument{Version: "2012-10-17"}
}

// PolicyStatement represents an IAM policy statement.
//
// Example:
//
//	var MyStatement = PolicyStatement{
//	    Effect:    "Allow",
//	    Principal: ServicePrincipal{"lambda.amazonaws.com"},
//	    Action:    []any{"sts:AssumeRole"},
//	}
type PolicyStatement struct {
	Sid       string `json:"Sid,omitempty"`
	Effect    string `json:"Effect"`
	Principal any    `json:"Principal,omitempty"`
	Action    any    `json:"Action,omitempty"`
	Resource  any    `json:"Resource,omitempty"`
	Condition Json   `json:"Condition,omitempty"`
}

// DenyStatement is a PolicyStatement with Effect="Deny".
type DenyStatement struct {
	Sid       string `json:"Sid,omitempty"`
	Effect    string `json:"Effect"`
	Principal any    `json:"Principal,omitempty"`
	Action    any    `json:"Action,omitempty"`
	Resource  any    `json:"Resource,omitempty"`
	Condition Json   `json:"Condition,omitempty"`
}

// NewDenyStatement creates a DenyStatement with Effect pre-set.
func NewDenyStatement() DenyStatement {
	return DenyStatement{Effect: "Deny"}
}

// --- Principal Helpers ---

// ServicePrincipal represents a service principal (e.g., lambda.amazonaws.com).
// Serializes to {"Service": ...} format.
//
// Examples:
//
//	ServicePrincipal{"lambda.amazonaws.com"}
//	ServicePrincipal{"ec2.amazonaws.com", "lambda.amazonaws.com"}
type ServicePrincipal []any

// MarshalJSON serializes to {"Service": ...} format.
func (p ServicePrincipal) MarshalJSON() ([]byte, error) {
	if len(p) == 1 {
		return json.Marshal(map[string]any{"Service": p[0]})
	}
	return json.Marshal(map[string]any{"Service": []any(p)})
}

// AWSPrincipal represents an AWS account/role/user principal.
// Serializes to {"AWS": ...} format.
//
// Examples:
//
//	AWSPrincipal{"arn:aws:iam::123456789:root"}
//	AWSPrincipal{Sub{String: "arn:${AWS::Partition}:iam::${AWS::AccountId}:root"}}
//	AWSPrincipal{"*"}
type AWSPrincipal []any

// MarshalJSON serializes to {"AWS": ...} format.
func (p AWSPrincipal) MarshalJSON() ([]byte, error) {
	if len(p) == 1 {
		return json.Marshal(map[string]any{"AWS": p[0]})
	}
	return json.Marshal(map[string]any{"AWS": []any(p)})
}

// FederatedPrincipal represents a federated identity principal.
// Serializes to {"Federated": ...} format.
//
// Example:
//
//	FederatedPrincipal{"cognito-identity.amazonaws.com"}
type FederatedPrincipal []any

// MarshalJSON serializes to {"Federated": ...} format.
func (p FederatedPrincipal) MarshalJSON() ([]byte, error) {
	if len(p) == 1 {
		return json.Marshal(map[string]any{"Federated": p[0]})
	}
	return json.Marshal(map[string]any{"Federated": []any(p)})
}

// AllPrincipal represents the wildcard principal "*".
const AllPrincipal = "*"

// --- IAM Condition Operator Constants ---
// Use these as keys in Condition maps for type safety and typo prevention.
//
// Example:
//
//	Condition: Json{
//	    Bool: Json{"aws:SecureTransport": false},
//	    StringEquals: Json{"s3:x-amz-acl": "bucket-owner-full-control"},
//	}

const (
	// String conditions
	StringEquals              = "StringEquals"
	StringNotEquals           = "StringNotEquals"
	StringEqualsIgnoreCase    = "StringEqualsIgnoreCase"
	StringNotEqualsIgnoreCase = "StringNotEqualsIgnoreCase"
	StringLike                = "StringLike"
	StringNotLike             = "StringNotLike"

	// Numeric conditions
	NumericEquals            = "NumericEquals"
	NumericNotEquals         = "NumericNotEquals"
	NumericLessThan          = "NumericLessThan"
	NumericLessThanEquals    = "NumericLessThanEquals"
	NumericGreaterThan       = "NumericGreaterThan"
	NumericGreaterThanEquals = "NumericGreaterThanEquals"

	// Date conditions
	DateEquals            = "DateEquals"
	DateNotEquals         = "DateNotEquals"
	DateLessThan          = "DateLessThan"
	DateLessThanEquals    = "DateLessThanEquals"
	DateGreaterThan       = "DateGreaterThan"
	DateGreaterThanEquals = "DateGreaterThanEquals"

	// Boolean condition
	Bool = "Bool"

	// IP address conditions
	IpAddress    = "IpAddress"
	NotIpAddress = "NotIpAddress"

	// ARN conditions
	ArnEquals    = "ArnEquals"
	ArnNotEquals = "ArnNotEquals"
	ArnLike      = "ArnLike"
	ArnNotLike   = "ArnNotLike"

	// Null condition
	Null = "Null"
)
