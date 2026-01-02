// Package linter provides lint rules for wetwire-aws Go code.
//
// This package analyzes Go source files to detect patterns that can be improved.
// Each rule provides clear messages and suggestions.
//
// Rules:
//
//	WAW001: Use pseudo-parameter constants instead of hardcoded strings
//	WAW002: Use intrinsic types instead of raw map[string]any
//	WAW003: Detect duplicate resource variable names
//	WAW004: Split large files with too many resources
//	WAW005: Extract inline property types to separate var declarations
//	WAW006: Use typed policy document structs instead of inline versions
//	WAW007: Use typed slices instead of []any{map[string]any{...}}
//	WAW008: Use named var declarations instead of inline struct literals (block style)
//	WAW009: Use typed structs instead of map[string]any in resource fields
//	WAW010: Flatten inline typed struct literals to named var declarations
//	WAW011: Validate enum property values against allowed values
//	WAW012: Use typed enum constants instead of raw strings
package linter

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// Issue represents a detected lint issue with fix information.
type Issue struct {
	RuleID     string // The rule identifier (e.g., "WAW001")
	Message    string // Human-readable description of the issue
	Suggestion string // The suggested replacement code
	File       string // File path where the issue was found
	Line       int    // Line number (1-indexed)
	Column     int    // Column number (0-indexed)
	Severity   string // "error" or "warning"
}

// Rule is the interface for lint rules.
type Rule interface {
	ID() string
	Description() string
	Check(file *ast.File, fset *token.FileSet) []Issue
}

// HardcodedPseudoParameter detects hardcoded AWS pseudo-parameter strings.
//
// Detects: "AWS::Region", "AWS::AccountId", "AWS::StackName"
// Suggests: intrinsics.AWS_REGION, intrinsics.AWS_ACCOUNT_ID, etc.
type HardcodedPseudoParameter struct{}

func (r HardcodedPseudoParameter) ID() string { return "WAW001" }
func (r HardcodedPseudoParameter) Description() string {
	return "Use pseudo-parameter constants instead of hardcoded strings"
}

var pseudoParams = map[string]string{
	"AWS::Region":           "AWS_REGION",
	"AWS::AccountId":        "AWS_ACCOUNT_ID",
	"AWS::StackName":        "AWS_STACK_NAME",
	"AWS::StackId":          "AWS_STACK_ID",
	"AWS::Partition":        "AWS_PARTITION",
	"AWS::URLSuffix":        "AWS_URL_SUFFIX",
	"AWS::NoValue":          "AWS_NO_VALUE",
	"AWS::NotificationARNs": "AWS_NOTIFICATION_ARNS",
}

func (r HardcodedPseudoParameter) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		// Remove quotes from string value
		value := strings.Trim(lit.Value, `"`)

		if constant, found := pseudoParams[value]; found {
			pos := fset.Position(lit.Pos())
			issues = append(issues, Issue{
				RuleID:     r.ID(),
				Message:    "Use " + constant + " instead of \"" + value + "\"",
				Suggestion: constant,
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   "warning",
			})
		}

		return true
	})

	return issues
}

// MapShouldBeIntrinsic detects map[string]any patterns that should use intrinsic types.
//
// Detects: map[string]any{"Ref": "..."}, map[string]any{"Fn::Sub": "..."}
// Suggests: intrinsics.Ref{...}, intrinsics.Sub{...}
type MapShouldBeIntrinsic struct{}

func (r MapShouldBeIntrinsic) ID() string { return "WAW002" }
func (r MapShouldBeIntrinsic) Description() string {
	return "Use intrinsic types instead of raw map[string]any"
}

var intrinsicKeys = map[string]string{
	"Ref":             "Ref",
	"Fn::Sub":         "Sub",
	"Fn::Join":        "Join",
	"Fn::Select":      "Select",
	"Fn::GetAZs":      "GetAZs",
	"Fn::GetAtt":      "GetAtt",
	"Fn::If":          "If",
	"Fn::Equals":      "Equals",
	"Fn::And":         "And",
	"Fn::Or":          "Or",
	"Fn::Not":         "Not",
	"Fn::Base64":      "Base64",
	"Fn::Split":       "Split",
	"Fn::ImportValue": "ImportValue",
	"Fn::FindInMap":   "FindInMap",
	"Fn::Cidr":        "Cidr",
}

func (r MapShouldBeIntrinsic) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if it's a map[string]any
		if !isMapStringAny(comp.Type) {
			return true
		}

		// Check if it has a single key-value pair with an intrinsic key
		if len(comp.Elts) != 1 {
			return true
		}

		kv, ok := comp.Elts[0].(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		keyLit, ok := kv.Key.(*ast.BasicLit)
		if !ok || keyLit.Kind != token.STRING {
			return true
		}

		keyValue := strings.Trim(keyLit.Value, `"`)
		if typeName, found := intrinsicKeys[keyValue]; found {
			pos := fset.Position(comp.Pos())
			issues = append(issues, Issue{
				RuleID:     r.ID(),
				Message:    "Use intrinsics." + typeName + "{...} instead of map[string]any{\"" + keyValue + "\": ...}",
				Suggestion: typeName + "{...}",
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   "warning",
			})
		}

		return true
	})

	return issues
}

// isMapStringAny checks if an expression is map[string]any
func isMapStringAny(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}

	// Check key is string
	keyIdent, ok := mapType.Key.(*ast.Ident)
	if !ok || keyIdent.Name != "string" {
		return false
	}

	// Check value is any (or interface{})
	switch v := mapType.Value.(type) {
	case *ast.Ident:
		return v.Name == "any"
	case *ast.InterfaceType:
		return len(v.Methods.List) == 0 // Empty interface
	}

	return false
}

// DuplicateResource detects duplicate resource variable names in a file.
type DuplicateResource struct{}

func (r DuplicateResource) ID() string { return "WAW003" }
func (r DuplicateResource) Description() string {
	return "Detect duplicate resource variable names"
}

func (r DuplicateResource) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	// Track variable names and their positions
	varLocations := make(map[string][]token.Position)

	// Find all top-level var declarations with resource types
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Check if this looks like a resource (has resource module type)
			if !isResourceDeclaration(valueSpec) {
				continue
			}

			for _, name := range valueSpec.Names {
				pos := fset.Position(name.Pos())
				varLocations[name.Name] = append(varLocations[name.Name], pos)
			}
		}
	}

	// Report duplicates
	for name, locations := range varLocations {
		if len(locations) > 1 {
			// Report all locations after the first
			for _, loc := range locations[1:] {
				issues = append(issues, Issue{
					RuleID:     r.ID(),
					Message:    "Duplicate resource variable '" + name + "' (first defined at line " + string(rune(locations[0].Line+'0')) + ")",
					Suggestion: "// DUPLICATE: var " + name,
					File:       loc.Filename,
					Line:       loc.Line,
					Column:     loc.Column,
					Severity:   "error",
				})
			}
		}
	}

	return issues
}

// isResourceDeclaration checks if a value spec is a resource declaration
func isResourceDeclaration(spec *ast.ValueSpec) bool {
	if len(spec.Values) == 0 {
		return false
	}

	// Check if the value is a composite literal with a selector type (e.g., ec2.VPC{})
	for _, value := range spec.Values {
		comp, ok := value.(*ast.CompositeLit)
		if !ok {
			continue
		}

		// Check for selector expression (e.g., ec2.VPC)
		sel, ok := comp.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		// Check if package name looks like a resource module
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			continue
		}

		// Resource modules are typically lowercase service names
		if isResourceModule(pkgIdent.Name) {
			return true
		}
	}

	return false
}

// isResourceModule checks if a package name looks like a resource module
func isResourceModule(name string) bool {
	// Common AWS service module names
	resourceModules := map[string]bool{
		"s3": true, "ec2": true, "iam": true, "lambda_": true, "rds": true,
		"dynamodb": true, "sqs": true, "sns": true, "cloudwatch": true,
		"logs": true, "events": true, "apigateway": true, "route53": true,
		"cloudfront": true, "ecs": true, "eks": true, "elasticache": true,
		"kms": true, "secretsmanager": true, "ssm": true, "stepfunctions": true,
		"cognito": true, "kinesis": true, "firehose": true, "glue": true,
		"athena": true, "redshift": true, "emr": true, "batch": true,
		"codebuild": true, "codepipeline": true, "codecommit": true,
		"codedeploy": true, "waf": true, "wafv2": true, "acm": true,
		"amplify": true, "appconfig": true, "appsync": true, "backup": true,
		"budgets": true, "chatbot": true, "cloudformation": true,
		"cloudtrail": true, "config": true, "elasticloadbalancingv2": true,
	}

	return resourceModules[name]
}

// FileTooLarge detects files with too many resources.
type FileTooLarge struct {
	MaxResources int
}

func (r FileTooLarge) ID() string { return "WAW004" }
func (r FileTooLarge) Description() string {
	return "Split large files into smaller ones"
}

func (r FileTooLarge) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	maxResources := r.MaxResources
	if maxResources == 0 {
		maxResources = 15 // Default
	}

	// Count resource declarations
	count := 0
	var resourceNames []string

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			if isResourceDeclaration(valueSpec) {
				for _, name := range valueSpec.Names {
					count++
					if len(resourceNames) < 5 {
						resourceNames = append(resourceNames, name.Name)
					}
				}
			}
		}
	}

	if count > maxResources {
		pos := fset.Position(file.Pos())
		message := "File has " + itoa(count) + " resources (max " + itoa(maxResources) + "). Consider splitting by category: storage.go, compute.go, network.go, security.go"
		issues = append(issues, Issue{
			RuleID:     r.ID(),
			Message:    message,
			Suggestion: "// Split " + itoa(count) + " resources into multiple files",
			File:       pos.Filename,
			Line:       1,
			Column:     0,
			Severity:   "warning",
		})
	}

	return issues
}

// Simple int to string conversion
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}

// InlinePropertyType detects inline map[string]any for property types.
type InlinePropertyType struct{}

func (r InlinePropertyType) ID() string { return "WAW005" }
func (r InlinePropertyType) Description() string {
	return "Use struct types instead of inline map[string]any for property types"
}

var propertyTypeSuffixes = []string{
	"_configuration", "_config", "_settings", "_options",
	"_specification", "_specifications", "_data", "_profile",
	"_mappings", "_interfaces", "_parameters", "_properties",
	"_attributes", "_metadata", "_definition", "_template",
}

func (r InlinePropertyType) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for field assignments in struct literals
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Check if the key is an identifier
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}

		fieldName := strings.ToLower(keyIdent.Name)

		// Check if field name suggests a property type
		isPropertyType := false
		for _, suffix := range propertyTypeSuffixes {
			if strings.HasSuffix(fieldName, suffix) {
				isPropertyType = true
				break
			}
		}

		if !isPropertyType {
			return true
		}

		// Check if value is map[string]any
		comp, ok := kv.Value.(*ast.CompositeLit)
		if !ok || !isMapStringAny(comp.Type) {
			return true
		}

		// Only flag if map has more than 1 key (not simple cases)
		if len(comp.Elts) <= 1 {
			return true
		}

		pos := fset.Position(kv.Pos())
		issues = append(issues, Issue{
			RuleID:     r.ID(),
			Message:    "Use a struct type for " + keyIdent.Name + " instead of inline map[string]any",
			Suggestion: "// Define a named type for " + keyIdent.Name,
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   "warning",
		})

		return true
	})

	return issues
}

// HardcodedPolicyVersion detects hardcoded IAM policy versions.
type HardcodedPolicyVersion struct{}

func (r HardcodedPolicyVersion) ID() string { return "WAW006" }
func (r HardcodedPolicyVersion) Description() string {
	return "Use constant for IAM policy version"
}

var policyVersionPattern = regexp.MustCompile(`^20\d{2}-\d{2}-\d{2}$`)

func (r HardcodedPolicyVersion) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for key-value pairs with "Version" key
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Check if key is "Version"
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}

		if keyIdent.Name != "Version" {
			return true
		}

		// Check if value is a string literal matching policy version pattern
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		value := strings.Trim(lit.Value, `"`)
		if policyVersionPattern.MatchString(value) {
			pos := fset.Position(lit.Pos())
			issues = append(issues, Issue{
				RuleID:     r.ID(),
				Message:    "Consider using a constant for policy version \"" + value + "\"",
				Suggestion: "PolicyVersion",
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   "info",
			})
		}

		return true
	})

	return issues
}

// InlineMapInSlice detects []any{map[string]any{...}} patterns that should use typed slices.
// Common in SecurityGroupIngress, BlockDeviceMappings, etc.
type InlineMapInSlice struct{}

func (r InlineMapInSlice) ID() string { return "WAW007" }
func (r InlineMapInSlice) Description() string {
	return "Use typed slices instead of []any{map[string]any{...}}"
}

// Known CloudFormation property arrays that should use typed structs
var knownPropertyArrays = map[string]string{
	"SecurityGroupIngress":  "Use ec2.Ingress struct",
	"SecurityGroupEgress":   "Use ec2.Egress struct",
	"BlockDeviceMappings":   "Use ec2.BlockDeviceMapping struct",
	"Tags":                  "Use Tag struct (already handled)",
	"Statement":             "Use iam.Statement struct",
	"Policies":              "Use iam.Policy struct",
	"Rules":                 "Use typed rule struct",
	"Listeners":             "Use typed listener struct",
	"TargetGroupAttributes": "Use typed attribute struct",
}

func (r InlineMapInSlice) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for field assignments in struct literals
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Check if the key is a known property array field
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}

		suggestion, isKnown := knownPropertyArrays[keyIdent.Name]
		if !isKnown {
			return true
		}

		// Skip Tags since we handle those specially
		if keyIdent.Name == "Tags" {
			return true
		}

		// Check if value is []any{map[string]any{...}}
		comp, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check for []any type
		arrType, ok := comp.Type.(*ast.ArrayType)
		if !ok {
			return true
		}
		elemIdent, ok := arrType.Elt.(*ast.Ident)
		if !ok || elemIdent.Name != "any" {
			return true
		}

		// Check if elements are map[string]any
		hasInlineMaps := false
		for _, elt := range comp.Elts {
			if innerComp, ok := elt.(*ast.CompositeLit); ok {
				if isMapStringAny(innerComp.Type) {
					hasInlineMaps = true
					break
				}
			}
		}

		if hasInlineMaps {
			pos := fset.Position(kv.Pos())
			issues = append(issues, Issue{
				RuleID:     r.ID(),
				Message:    keyIdent.Name + " uses inline map[string]any. " + suggestion,
				Suggestion: "// Refactor to use typed structs",
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   "warning",
			})
		}

		return true
	})

	return issues
}

// InlineStructLiteral detects anonymous struct literals in typed slices.
// Enforces the block style where each property type instance should be a named var.
// Example:
//
//	// Bad - inline struct literals
//	SecurityGroupIngress: []ec2.SecurityGroup_Ingress{{CidrIp: "0.0.0.0/0", ...}, {...}}
//
//	// Good - named var references
//	SecurityGroupIngress: []ec2.SecurityGroup_Ingress{MyPort443, MyPort80}
type InlineStructLiteral struct{}

func (r InlineStructLiteral) ID() string { return "WAW008" }
func (r InlineStructLiteral) Description() string {
	return "Use named var declarations instead of inline struct literals (block style)"
}

// knownTypedSlices maps property names to their expected typed slice element types.
// These are the properties where we enforce block style.
var knownTypedSlices = map[string]bool{
	"SecurityGroupIngress":  true,
	"SecurityGroupEgress":   true,
	"BlockDeviceMappings":   true,
	"Volumes":               true,
	"Policies":              true,
	"TargetGroupAttributes": true,
	"Actions":               true,
	"Tags":                  true,
}

func (r InlineStructLiteral) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for field assignments in struct literals
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Get the field name
		fieldName := ""
		switch key := kv.Key.(type) {
		case *ast.Ident:
			fieldName = key.Name
		}

		// Check if this is a known property that should use block style
		if !knownTypedSlices[fieldName] {
			return true
		}

		// Check if the value is a composite literal (slice)
		comp, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if it's a slice type (has [...] syntax)
		_, isArray := comp.Type.(*ast.ArrayType)
		if !isArray {
			return true
		}

		// Check each element - if any is a composite literal (not an ident), flag it
		for _, elt := range comp.Elts {
			// If element is a composite literal (anonymous struct), it's inline
			if innerComp, ok := elt.(*ast.CompositeLit); ok {
				pos := fset.Position(innerComp.Pos())
				issues = append(issues, Issue{
					RuleID:     r.ID(),
					Message:    fmt.Sprintf("Use a named var declaration for %s item instead of inline struct literal", fieldName),
					Suggestion: "Extract to: var MyItem = Type{...} and reference by name",
					File:       pos.Filename,
					Line:       pos.Line,
					Column:     pos.Column,
					Severity:   "warning",
				})
			}
		}

		return true
	})

	return issues
}

// UnflattenedMap detects any map[string]any in resource field assignments, recursively.
// This is a comprehensive rule that catches all cases where typed structs should be used,
// including deeply nested maps within slices and other maps.
//
// Example:
//
//	// Bad - unflattened maps at any depth
//	DistributionConfig: map[string]any{
//	    "Origins": []any{
//	        map[string]any{
//	            "CustomOriginConfig": map[string]any{...},  // Nested - also caught
//	        },
//	    },
//	}
//
//	// Good - use typed structs at all levels
//	DistributionConfig: cloudfront.Distribution_DistributionConfig{
//	    Origins: []cloudfront.Distribution_Origin{...},
//	}
type UnflattenedMap struct{}

func (r UnflattenedMap) ID() string { return "WAW009" }
func (r UnflattenedMap) Description() string {
	return "Use typed structs instead of map[string]any in resource fields (recursive)"
}

// Fields to ignore (they legitimately use map[string]any or are handled elsewhere)
var ignoreFields = map[string]bool{
	"Tags":     true, // Handled by Tag type
	"Metadata": true, // Arbitrary metadata
}

func (r UnflattenedMap) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for field assignments in struct literals
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Get the field name
		fieldName := ""
		switch key := kv.Key.(type) {
		case *ast.Ident:
			fieldName = key.Name
		case *ast.BasicLit:
			// Skip string keys in maps (those are the map[string]any keys)
			return true
		}

		// Skip ignored fields
		if ignoreFields[fieldName] {
			return true
		}

		// Recursively find all map[string]any in the value
		foundIssues := findUnflattenedMaps(kv.Value, fieldName, fset, r.ID())
		issues = append(issues, foundIssues...)

		// Return false to prevent double-processing of nested KeyValueExpr
		// (we handle them recursively in findUnflattenedMaps)
		return false
	})

	return issues
}

// findUnflattenedMaps recursively searches for map[string]any patterns in an expression.
// It tracks the path through the structure for meaningful error messages.
func findUnflattenedMaps(expr ast.Expr, path string, fset *token.FileSet, ruleID string) []Issue {
	var issues []Issue

	comp, ok := expr.(*ast.CompositeLit)
	if !ok {
		return issues
	}

	// Check if this is map[string]any
	if isMapStringAny(comp.Type) {
		pos := fset.Position(comp.Pos())
		issues = append(issues, Issue{
			RuleID:     ruleID,
			Message:    fmt.Sprintf("%s: use typed struct instead of map[string]any", path),
			Suggestion: fmt.Sprintf("// Use the appropriate property type struct for %s", path),
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   "warning",
		})

		// Recursively check map values
		for _, elt := range comp.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				// Get the key name for the path
				keyName := "?"
				if lit, ok := kv.Key.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					keyName = strings.Trim(lit.Value, `"`)
				}
				newPath := path + "." + keyName
				issues = append(issues, findUnflattenedMaps(kv.Value, newPath, fset, ruleID)...)
			}
		}
		return issues
	}

	// Check if this is []any containing elements
	if arrType, ok := comp.Type.(*ast.ArrayType); ok {
		if elemIdent, ok := arrType.Elt.(*ast.Ident); ok && elemIdent.Name == "any" {
			// Check each element recursively
			for i, elt := range comp.Elts {
				newPath := fmt.Sprintf("%s[%d]", path, i)
				issues = append(issues, findUnflattenedMaps(elt, newPath, fset, ruleID)...)
			}
			return issues
		}
	}

	// For typed struct literals, check their field values
	for _, elt := range comp.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			fieldName := ""
			if ident, ok := kv.Key.(*ast.Ident); ok {
				fieldName = ident.Name
			}
			if fieldName != "" && !ignoreFields[fieldName] {
				newPath := path + "." + fieldName
				issues = append(issues, findUnflattenedMaps(kv.Value, newPath, fset, ruleID)...)
			}
		}
	}

	return issues
}

// InlineTypedStruct detects typed property type struct literals that should be
// extracted to separate named variable declarations (block style).
//
// This enforces the pattern where each property type instance is a separate
// top-level var declaration, matching the Python wetwire-aws pattern:
//
//	// Bad - inline typed struct literals
//	var LoggingBucket = s3.Bucket{
//	    BucketEncryption: &s3.Bucket_BucketEncryption{
//	        ServerSideEncryptionConfiguration: []s3.Bucket_ServerSideEncryptionRule{
//	            s3.Bucket_ServerSideEncryptionRule{...},
//	        },
//	    },
//	}
//
//	// Good - flattened to named var declarations (block style)
//	var LoggingBucketSSEByDefault = &s3.Bucket_ServerSideEncryptionByDefault{...}
//	var LoggingBucketSSERule = s3.Bucket_ServerSideEncryptionRule{
//	    ServerSideEncryptionByDefault: LoggingBucketSSEByDefault,
//	}
//	var LoggingBucketEncryption = &s3.Bucket_BucketEncryption{
//	    ServerSideEncryptionConfiguration: []s3.Bucket_ServerSideEncryptionRule{LoggingBucketSSERule},
//	}
//	var LoggingBucket = s3.Bucket{
//	    BucketEncryption: LoggingBucketEncryption,
//	}
type InlineTypedStruct struct{}

func (r InlineTypedStruct) ID() string { return "WAW010" }
func (r InlineTypedStruct) Description() string {
	return "Flatten inline typed struct literals to named var declarations (block style)"
}

func (r InlineTypedStruct) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	// Track nesting depth - we only flag structs that are nested (depth > 0)
	// Top-level var declarations are fine
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for top-level var declarations
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			return true
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, value := range valueSpec.Values {
				// Find inline typed structs within this top-level var
				foundIssues := findInlineTypedStructs(value, fset, r.ID(), 0)
				issues = append(issues, foundIssues...)
			}
		}

		return true
	})

	return issues
}

// findInlineTypedStructs recursively finds typed property type struct literals
// that are nested (depth > 0) and should be flattened.
func findInlineTypedStructs(expr ast.Expr, fset *token.FileSet, ruleID string, depth int) []Issue {
	var issues []Issue

	switch e := expr.(type) {
	case *ast.CompositeLit:
		// Check if this is a typed property type struct (contains "_" in type name)
		isPropertyType := false
		typeName := ""

		if sel, ok := e.Type.(*ast.SelectorExpr); ok {
			typeName = sel.Sel.Name
			// Property types contain underscore: Bucket_BucketEncryption
			if strings.Contains(typeName, "_") {
				isPropertyType = true
			}
		}

		// Check if this is a slice type (array)
		isSlice := false
		if _, ok := e.Type.(*ast.ArrayType); ok {
			isSlice = true
		}

		// Flag if this is a nested property type struct (depth > 0)
		if isPropertyType && depth > 0 {
			pos := fset.Position(e.Pos())
			issues = append(issues, Issue{
				RuleID:     ruleID,
				Message:    fmt.Sprintf("Flatten inline %s to a named var declaration", typeName),
				Suggestion: fmt.Sprintf("var My%s = ...", typeName),
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   "warning",
			})
		}

		// Recurse into elements
		for _, elt := range e.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				// Struct field: key-value pair
				issues = append(issues, findInlineTypedStructs(kv.Value, fset, ruleID, depth+1)...)
			} else if isSlice {
				// Array element: direct element (not key-value)
				issues = append(issues, findInlineTypedStructs(elt, fset, ruleID, depth+1)...)
			}
		}

	case *ast.UnaryExpr:
		// Handle pointer expressions like &s3.Bucket_BucketEncryption{...}
		if e.Op == token.AND {
			issues = append(issues, findInlineTypedStructs(e.X, fset, ruleID, depth)...)
		}

	case *ast.CallExpr:
		// Recurse into function call arguments
		for _, arg := range e.Args {
			issues = append(issues, findInlineTypedStructs(arg, fset, ruleID, depth+1)...)
		}
	}

	return issues
}

// InvalidEnumValue detects invalid enum property values.
// Uses cloudformation-schema-go/enums to validate values against known enums.
//
// Example:
//
//	// Bad - invalid enum value
//	StorageClass: "INVALID_CLASS",
//
//	// Good - valid enum value
//	StorageClass: "STANDARD",
//	// Or use the constant
//	StorageClass: enums.S3StorageClassSTANDARD,
type InvalidEnumValue struct{}

func (r InvalidEnumValue) ID() string { return "WAW011" }
func (r InvalidEnumValue) Description() string {
	return "Validate enum property values against allowed values"
}

// enumFieldInfo holds the service and enum name for a field.
type enumFieldInfo struct {
	service  string
	enumName string
}

// enumFields maps CloudFormation property names to their enum info.
// This allows the linter to validate string literals assigned to these fields.
var enumFields = map[string]enumFieldInfo{
	// S3 enums
	"StorageClass": {"s3", "StorageClass"},

	// EC2 enums
	"InstanceType": {"ec2", "InstanceType"},

	// Lambda enums
	"Runtime":     {"lambda", "Runtime"},
	"PackageType": {"lambda", "PackageType"},

	// RDS enums
	"Engine":        {"rds", "Engine"},
	"EngineVersion": {"rds", "EngineVersion"},

	// DynamoDB enums
	"BillingMode":    {"dynamodb", "BillingMode"},
	"StreamViewType": {"dynamodb", "StreamViewType"},
	"TableClass":     {"dynamodb", "TableClass"},

	// ECS enums
	"LaunchType":         {"ecs", "LaunchType"},
	"NetworkMode":        {"ecs", "NetworkMode"},
	"SchedulingStrategy": {"ecs", "SchedulingStrategy"},
}

func (r InvalidEnumValue) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	// Check if the enums package is available
	hasEnumsImport := false
	for _, imp := range file.Imports {
		if imp.Path != nil && strings.Contains(imp.Path.Value, "cloudformation-schema-go/enums") {
			hasEnumsImport = true
			break
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Get field name
		fieldIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}

		// Check if this is a known enum field
		enumInfo, ok := enumFields[fieldIdent.Name]
		if !ok {
			return true
		}

		// Check if value is a string literal
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			// Skip non-string values (could be intrinsics, selectors, etc.)
			return true
		}

		value := strings.Trim(lit.Value, `"`)

		// Use the enums package to validate
		if !isValidEnumValue(enumInfo.service, enumInfo.enumName, value) {
			pos := fset.Position(lit.Pos())
			allowed := getAllowedEnumValues(enumInfo.service, enumInfo.enumName)

			// Limit displayed values if there are too many
			displayAllowed := allowed
			if len(allowed) > 5 {
				displayAllowed = append(allowed[:5], "...")
			}

			suggestion := fmt.Sprintf("Use one of: %s", strings.Join(displayAllowed, ", "))
			if !hasEnumsImport {
				suggestion += " (or import enums package and use constants)"
			}

			issues = append(issues, Issue{
				RuleID:     r.ID(),
				Message:    fmt.Sprintf("Invalid %s value: %q", fieldIdent.Name, value),
				Suggestion: suggestion,
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   "error",
			})
		}

		return true
	})

	return issues
}

// isValidEnumValue checks if a value is valid for the given enum.
// This is a local implementation to avoid import cycles with the enums package.
func isValidEnumValue(service, enumName, value string) bool {
	allowed := getAllowedEnumValues(service, enumName)
	for _, v := range allowed {
		if v == value {
			return true
		}
	}
	return false
}

// getAllowedEnumValues returns the allowed values for an enum.
// This is embedded here to avoid runtime dependency on the enums package.
var enumAllowedValues = map[string]map[string][]string{
	"s3": {
		"StorageClass": {
			"STANDARD", "REDUCED_REDUNDANCY", "STANDARD_IA", "ONEZONE_IA",
			"INTELLIGENT_TIERING", "GLACIER", "DEEP_ARCHIVE", "OUTPOSTS",
			"GLACIER_IR", "SNOW", "EXPRESS_ONEZONE",
		},
	},
	"ec2": {
		"InstanceType": {
			// Common instance types (subset for linting purposes)
			"t2.micro", "t2.small", "t2.medium", "t2.large", "t2.xlarge", "t2.2xlarge",
			"t3.micro", "t3.small", "t3.medium", "t3.large", "t3.xlarge", "t3.2xlarge",
			"t3a.micro", "t3a.small", "t3a.medium", "t3a.large", "t3a.xlarge", "t3a.2xlarge",
			"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge",
			"m6i.large", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge",
			"c5.large", "c5.xlarge", "c5.2xlarge", "c5.4xlarge", "c5.9xlarge",
			"r5.large", "r5.xlarge", "r5.2xlarge", "r5.4xlarge", "r5.8xlarge",
		},
	},
	"lambda": {
		"Runtime": {
			"nodejs18.x", "nodejs20.x", "nodejs22.x",
			"python3.9", "python3.10", "python3.11", "python3.12", "python3.13",
			"java11", "java17", "java21",
			"dotnet6", "dotnet8",
			"ruby3.2", "ruby3.3",
			"provided", "provided.al2", "provided.al2023",
		},
		"PackageType":  {"Zip", "Image"},
		"Architecture": {"x86_64", "arm64"},
	},
	"rds": {
		"Engine": {
			"mysql", "mariadb", "postgres", "oracle-ee", "oracle-se2",
			"sqlserver-ee", "sqlserver-se", "sqlserver-ex", "sqlserver-web",
			"aurora", "aurora-mysql", "aurora-postgresql",
		},
	},
	"dynamodb": {
		"BillingMode":    {"PROVISIONED", "PAY_PER_REQUEST"},
		"StreamViewType": {"KEYS_ONLY", "NEW_IMAGE", "OLD_IMAGE", "NEW_AND_OLD_IMAGES"},
		"TableClass":     {"STANDARD", "STANDARD_INFREQUENT_ACCESS"},
	},
	"ecs": {
		"LaunchType":         {"EC2", "FARGATE", "EXTERNAL"},
		"NetworkMode":        {"bridge", "host", "awsvpc", "none"},
		"SchedulingStrategy": {"REPLICA", "DAEMON"},
	},
}

func getAllowedEnumValues(service, enumName string) []string {
	if svc, ok := enumAllowedValues[service]; ok {
		if vals, ok := svc[enumName]; ok {
			return vals
		}
	}
	return nil
}

// PreferEnumConstant detects raw string literals used for enum properties
// and suggests using typed enum constants instead.
//
// Example:
//
//	// Bad - raw string literal
//	Runtime: "python3.12",
//	StorageClass: "STANDARD",
//
//	// Good - typed enum constants
//	Runtime: enums.LambdaRuntimePython312,
//	StorageClass: enums.S3StorageClassStandard,
type PreferEnumConstant struct{}

func (r PreferEnumConstant) ID() string { return "WAW012" }
func (r PreferEnumConstant) Description() string {
	return "Use typed enum constants instead of raw strings"
}

// enumConstantMap maps (service, enumName, value) to the constant name.
// This is used to suggest the appropriate constant when a raw string is detected.
var enumConstantMap = map[string]map[string]map[string]string{
	"lambda": {
		"Runtime": {
			"python3.9":       "LambdaRuntimePython39",
			"python3.10":      "LambdaRuntimePython310",
			"python3.11":      "LambdaRuntimePython311",
			"python3.12":      "LambdaRuntimePython312",
			"python3.13":      "LambdaRuntimePython313",
			"nodejs18.x":      "LambdaRuntimeNodejs18X",
			"nodejs20.x":      "LambdaRuntimeNodejs20X",
			"nodejs22.x":      "LambdaRuntimeNodejs22X",
			"java11":          "LambdaRuntimeJava11",
			"java17":          "LambdaRuntimeJava17",
			"java21":          "LambdaRuntimeJava21",
			"dotnet6":         "LambdaRuntimeDotnet6",
			"dotnet8":         "LambdaRuntimeDotnet8",
			"ruby3.2":         "LambdaRuntimeRuby32",
			"ruby3.3":         "LambdaRuntimeRuby33",
			"provided.al2":    "LambdaRuntimeProvidedAl2",
			"provided.al2023": "LambdaRuntimeProvidedAl2023",
		},
		"PackageType": {
			"Zip":   "LambdaPackageTypeZip",
			"Image": "LambdaPackageTypeImage",
		},
		"Architecture": {
			"x86_64": "LambdaArchitectureX8664",
			"arm64":  "LambdaArchitectureArm64",
		},
	},
	"s3": {
		"StorageClass": {
			"STANDARD":            "S3StorageClassStandard",
			"REDUCED_REDUNDANCY":  "S3StorageClassReducedRedundancy",
			"STANDARD_IA":         "S3StorageClassStandardIa",
			"ONEZONE_IA":          "S3StorageClassOnezoneIa",
			"INTELLIGENT_TIERING": "S3StorageClassIntelligentTiering",
			"GLACIER":             "S3StorageClassGlacier",
			"DEEP_ARCHIVE":        "S3StorageClassDeepArchive",
			"GLACIER_IR":          "S3StorageClassGlacierIr",
		},
		"ServerSideEncryption": {
			"AES256":       "S3ServerSideEncryptionAes256",
			"aws:kms":      "S3ServerSideEncryptionAwsKms",
			"aws:kms:dsse": "S3ServerSideEncryptionAwsKmsDsse",
		},
	},
	"ec2": {
		"VolumeType": {
			"gp2":      "Ec2VolumeTypeGp2",
			"gp3":      "Ec2VolumeTypeGp3",
			"io1":      "Ec2VolumeTypeIo1",
			"io2":      "Ec2VolumeTypeIo2",
			"st1":      "Ec2VolumeTypeSt1",
			"sc1":      "Ec2VolumeTypeSc1",
			"standard": "Ec2VolumeTypeStandard",
		},
	},
	"ecs": {
		"LaunchType": {
			"EC2":      "EcsLaunchTypeEc2",
			"FARGATE":  "EcsLaunchTypeFargate",
			"EXTERNAL": "EcsLaunchTypeExternal",
		},
		"NetworkMode": {
			"bridge": "EcsNetworkModeBridge",
			"host":   "EcsNetworkModeHost",
			"awsvpc": "EcsNetworkModeAwsvpc",
			"none":   "EcsNetworkModeNone",
		},
		"SchedulingStrategy": {
			"REPLICA": "EcsSchedulingStrategyReplica",
			"DAEMON":  "EcsSchedulingStrategyDaemon",
		},
	},
	"dynamodb": {
		"BillingMode": {
			"PROVISIONED":     "DynamodbBillingModeProvisioned",
			"PAY_PER_REQUEST": "DynamodbBillingModePayPerRequest",
		},
		"StreamViewType": {
			"KEYS_ONLY":          "DynamodbStreamViewTypeKeysOnly",
			"NEW_IMAGE":          "DynamodbStreamViewTypeNewImage",
			"OLD_IMAGE":          "DynamodbStreamViewTypeOldImage",
			"NEW_AND_OLD_IMAGES": "DynamodbStreamViewTypeNewAndOldImages",
		},
		"TableClass": {
			"STANDARD":                   "DynamodbTableClassStandard",
			"STANDARD_INFREQUENT_ACCESS": "DynamodbTableClassStandardInfrequentAccess",
		},
	},
}

// enumFieldToService maps field names to their service for constant lookup.
var enumFieldToService = map[string]string{
	"Runtime":            "lambda",
	"PackageType":        "lambda",
	"Architectures":      "lambda",
	"StorageClass":       "s3",
	"SSEAlgorithm":       "s3",
	"VolumeType":         "ec2",
	"LaunchType":         "ecs",
	"NetworkMode":        "ecs",
	"SchedulingStrategy": "ecs",
	"BillingMode":        "dynamodb",
	"StreamViewType":     "dynamodb",
	"TableClass":         "dynamodb",
}

// enumFieldToEnumName maps field names to enum names (for cases where they differ).
var enumFieldToEnumName = map[string]string{
	"SSEAlgorithm":  "ServerSideEncryption",
	"Architectures": "Architecture",
}

func (r PreferEnumConstant) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Get field name
		fieldIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}

		// Check if this is a known enum field
		service, ok := enumFieldToService[fieldIdent.Name]
		if !ok {
			return true
		}

		// Determine enum name (may differ from field name)
		enumName := fieldIdent.Name
		if mapped, ok := enumFieldToEnumName[fieldIdent.Name]; ok {
			enumName = mapped
		}

		// Check if value is a string literal
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			// Skip non-string values (could be constants, selectors, etc.)
			return true
		}

		value := strings.Trim(lit.Value, `"`)

		// Look up the constant name
		if serviceEnums, ok := enumConstantMap[service]; ok {
			if enumValues, ok := serviceEnums[enumName]; ok {
				if constName, ok := enumValues[value]; ok {
					pos := fset.Position(lit.Pos())
					issues = append(issues, Issue{
						RuleID:     r.ID(),
						Message:    fmt.Sprintf("Use enums.%s instead of %q", constName, value),
						Suggestion: "enums." + constName,
						File:       pos.Filename,
						Line:       pos.Line,
						Column:     pos.Column,
						Severity:   "warning",
					})
				}
			}
		}

		return true
	})

	return issues
}

// AllRules returns all available lint rules.
func AllRules() []Rule {
	return []Rule{
		HardcodedPseudoParameter{},
		MapShouldBeIntrinsic{},
		DuplicateResource{},
		FileTooLarge{MaxResources: 15},
		InlinePropertyType{},
		HardcodedPolicyVersion{},
		InlineMapInSlice{},
		InlineStructLiteral{},
		UnflattenedMap{},
		InlineTypedStruct{},
		InvalidEnumValue{},
		PreferEnumConstant{},
	}
}
