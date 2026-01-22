// Package lint provides lint rules for wetwire-aws Go code.
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
//	WAW015: Avoid explicit Ref{} - use direct variable references or Param()
//	WAW016: Avoid explicit GetAtt{} - use resource.Attr field access
//	WAW017: Avoid pointer assignments (&Type{}) - use value types
//	WAW018: Use Json{} instead of map[string]any{} for cleaner syntax
package lint

import (
	"go/ast"
	"go/token"
	"strings"
)

// Note: Rule interface is imported from corelint via type alias in linter.go.
// Issue and Severity types are also imported from corelint.

// PackageContext holds information about all files in a package.
// This is used by package-aware rules that need cross-file visibility.
type PackageContext struct {
	// AllDefinedVars contains all package-level variable names across all files
	AllDefinedVars map[string]bool
}

// PackageAwareRule is an optional interface for rules that need package-level context.
// Rules implementing this interface will receive cross-file information.
type PackageAwareRule interface {
	Rule
	CheckWithContext(file *ast.File, fset *token.FileSet, ctx *PackageContext) []Issue
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
				Rule:       r.ID(),
				Message:    "Use " + constant + " instead of \"" + value + "\"",
				Suggestion: constant,
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
				Rule:       r.ID(),
				Message:    "Use intrinsics." + typeName + "{...} instead of map[string]any{\"" + keyValue + "\": ...}",
				Suggestion: typeName + "{...}",
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
					Rule:       r.ID(),
					Message:    "Duplicate resource variable '" + name + "' (first defined at line " + string(rune(locations[0].Line+'0')) + ")",
					Suggestion: "// DUPLICATE: var " + name,
					File:       loc.Filename,
					Line:       loc.Line,
					Column:     loc.Column,
					Severity:   SeverityError,
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
			Rule:       r.ID(),
			Message:    message,
			Suggestion: "// Split " + itoa(count) + " resources into multiple files",
			File:       pos.Filename,
			Line:       1,
			Column:     0,
			Severity:   SeverityWarning,
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
			Rule:       r.ID(),
			Message:    "Use a struct type for " + keyIdent.Name + " instead of inline map[string]any",
			Suggestion: "// Define a named type for " + keyIdent.Name,
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   SeverityWarning,
		})

		return true
	})

	return issues
}
