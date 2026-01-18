package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

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
				Rule:       r.ID(),
				Message:    "Consider using a constant for policy version \"" + value + "\"",
				Suggestion: "PolicyVersion",
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   SeverityInfo,
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
				Rule:       r.ID(),
				Message:    keyIdent.Name + " uses inline map[string]any. " + suggestion,
				Suggestion: "// Refactor to use typed structs",
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   SeverityWarning,
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
					Rule:       r.ID(),
					Message:    fmt.Sprintf("Use a named var declaration for %s item instead of inline struct literal", fieldName),
					Suggestion: "Extract to: var MyItem = Type{...} and reference by name",
					File:       pos.Filename,
					Line:       pos.Line,
					Column:     pos.Column,
					Severity:   SeverityWarning,
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
			Rule:       ruleID,
			Message:    fmt.Sprintf("%s: use typed struct instead of map[string]any", path),
			Suggestion: fmt.Sprintf("// Use the appropriate property type struct for %s", path),
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   SeverityWarning,
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
				Rule:       ruleID,
				Message:    fmt.Sprintf("Flatten inline %s to a named var declaration", typeName),
				Suggestion: fmt.Sprintf("var My%s = ...", typeName),
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   SeverityWarning,
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
