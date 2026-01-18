package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// AvoidExplicitRef detects explicit Ref{} struct literals.
// Prefer direct variable references for resources or Param() for parameters.
//
// Example:
//
//	// Bad - explicit Ref{}
//	Bucket: Ref{"MyBucket"},
//	VpcId: Ref{"VpcIdParam"},
//
//	// Good - direct reference for resources
//	Bucket: MyBucket,
//
//	// Good - Param() helper for parameters
//	VpcId: VpcIdParam,  // where VpcIdParam = Param("VpcIdParam")
type AvoidExplicitRef struct{}

func (r AvoidExplicitRef) ID() string { return "WAW015" }
func (r AvoidExplicitRef) Description() string {
	return "Avoid explicit Ref{} - use direct variable references or Param()"
}

func (r AvoidExplicitRef) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if this is a Ref{...} struct literal
		ident, ok := comp.Type.(*ast.Ident)
		if !ok || ident.Name != "Ref" {
			return true
		}

		// Get the reference name if available
		refName := ""
		if len(comp.Elts) > 0 {
			if lit, ok := comp.Elts[0].(*ast.BasicLit); ok {
				refName = strings.Trim(lit.Value, `"`)
			}
		}

		pos := fset.Position(comp.Pos())
		msg := "Avoid Ref{} - use direct variable reference or Param() helper"
		suggestion := "Use direct variable reference for resources, Param() for parameters"
		if refName != "" {
			suggestion = fmt.Sprintf("For resources: use %s directly. For parameters: var %s = Param(\"%s\")", refName, refName, refName)
		}

		issues = append(issues, Issue{
			Rule:       r.ID(),
			Message:    msg,
			Suggestion: suggestion,
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   SeverityWarning,
		})

		return true
	})

	return issues
}

// AvoidExplicitGetAtt detects explicit GetAtt{} struct literals.
// Prefer resource.Attr field access for GetAtt functionality.
//
// Example:
//
//	// Bad - explicit GetAtt{}
//	Role: GetAtt{"MyRole", "Arn"},
//
//	// Good - field access
//	Role: MyRole.Arn,
type AvoidExplicitGetAtt struct{}

func (r AvoidExplicitGetAtt) ID() string { return "WAW016" }
func (r AvoidExplicitGetAtt) Description() string {
	return "Avoid explicit GetAtt{} - use resource.Attr field access"
}

func (r AvoidExplicitGetAtt) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if this is a GetAtt{...} struct literal
		ident, ok := comp.Type.(*ast.Ident)
		if !ok || ident.Name != "GetAtt" {
			return true
		}

		// Get the resource and attribute names if available
		resourceName := ""
		attrName := ""
		if len(comp.Elts) >= 2 {
			if lit, ok := comp.Elts[0].(*ast.BasicLit); ok {
				resourceName = strings.Trim(lit.Value, `"`)
			}
			if lit, ok := comp.Elts[1].(*ast.BasicLit); ok {
				attrName = strings.Trim(lit.Value, `"`)
			}
		}

		pos := fset.Position(comp.Pos())
		msg := "Avoid GetAtt{} - use resource.Attr field access"
		suggestion := "Use Resource.Attr field access instead"
		if resourceName != "" && attrName != "" {
			suggestion = fmt.Sprintf("Use %s.%s instead", resourceName, attrName)
		}

		issues = append(issues, Issue{
			Rule:       r.ID(),
			Message:    msg,
			Suggestion: suggestion,
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   SeverityWarning,
		})

		return true
	})

	return issues
}

// AvoidPointerAssignment detects pointer assignments in top-level var declarations.
// The AST-based value extraction expects struct literals, not pointers.
//
// Example:
//
//	// Bad - pointer assignment
//	var MyConfig = &s3.Bucket_VersioningConfiguration{
//	    Status: "Enabled",
//	}
//
//	// Good - value assignment
//	var MyConfig = s3.Bucket_VersioningConfiguration{
//	    Status: "Enabled",
//	}
type AvoidPointerAssignment struct{}

func (r AvoidPointerAssignment) ID() string { return "WAW017" }
func (r AvoidPointerAssignment) Description() string {
	return "Avoid pointer assignments (&Type{}) - use value types"
}

func (r AvoidPointerAssignment) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

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

			for i, value := range valueSpec.Values {
				// Check if value is a unary expression with & operator
				unary, ok := value.(*ast.UnaryExpr)
				if !ok || unary.Op != token.AND {
					continue
				}

				// Check if operand is a composite literal (struct)
				comp, ok := unary.X.(*ast.CompositeLit)
				if !ok {
					continue
				}

				// Get the type name for the message
				typeName := "struct"
				switch t := comp.Type.(type) {
				case *ast.SelectorExpr:
					typeName = t.Sel.Name
				case *ast.Ident:
					typeName = t.Name
				}

				// Get variable name
				varName := "_"
				if i < len(valueSpec.Names) {
					varName = valueSpec.Names[i].Name
				}

				pos := fset.Position(unary.Pos())
				issues = append(issues, Issue{
					Rule:       r.ID(),
					Message:    fmt.Sprintf("Avoid pointer assignment for %s - use value type instead of &%s{}", varName, typeName),
					Suggestion: fmt.Sprintf("var %s = %s{...} (remove &)", varName, typeName),
					File:       pos.Filename,
					Line:       pos.Line,
					Column:     pos.Column,
					Severity:   SeverityError,
				})
			}
		}
	}

	return issues
}

// PreferJsonType detects map[string]any{} literals and suggests using Json{} instead.
// The Json type is cleaner and provides better readability.
//
// Example:
//
//	// Bad - verbose map syntax
//	CustomOriginConfig: map[string]any{
//	    "HTTPPort": 80,
//	    "HTTPSPort": 443,
//	}
//
//	// Good - use Json type alias
//	CustomOriginConfig: Json{
//	    "HTTPPort": 80,
//	    "HTTPSPort": 443,
//	}
type PreferJsonType struct{}

func (r PreferJsonType) ID() string { return "WAW018" }
func (r PreferJsonType) Description() string {
	return "Use Json{} instead of map[string]any{} for cleaner syntax"
}

func (r PreferJsonType) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if this is map[string]any
		if !isMapStringAny(comp.Type) {
			return true
		}

		// Skip maps that are intrinsic functions (those are handled by WAW002)
		// Check if it has a single key-value pair with an intrinsic key
		if len(comp.Elts) == 1 {
			if kv, ok := comp.Elts[0].(*ast.KeyValueExpr); ok {
				if keyLit, ok := kv.Key.(*ast.BasicLit); ok && keyLit.Kind == token.STRING {
					keyValue := strings.Trim(keyLit.Value, `"`)
					if _, isIntrinsic := intrinsicKeys[keyValue]; isIntrinsic {
						return true // Skip intrinsic patterns, handled by WAW002
					}
				}
			}
		}

		pos := fset.Position(comp.Pos())
		issues = append(issues, Issue{
			Rule:       r.ID(),
			Message:    "Use Json{} instead of map[string]any{} for cleaner syntax",
			Suggestion: "Json{...}",
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   SeverityWarning,
		})

		return true
	})

	return issues
}

// SecretPattern detects hardcoded secrets, API keys, passwords, and tokens.
// This rule helps prevent accidental exposure of sensitive credentials.
//
// Detected patterns:
//   - AWS access keys (AKIA...)
//   - AWS secret keys (long base64-like strings with access key context)
//   - Private key headers (-----BEGIN ... PRIVATE KEY-----)
//   - Generic API keys (sk_live_, sk_test_, api_key, etc.)
//   - Passwords in field names (password, secret, token with literal values)
//
// Example:
//
//	// Bad - hardcoded AWS credentials
//	Environment: Json{
//	    "AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
//	    "AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
//	}
//
//	// Good - use AWS Secrets Manager or Parameter Store
//	Environment: Json{
//	    "DB_SECRET_ARN": MyDbSecret.Arn,
//	}
type SecretPattern struct{}

func (r SecretPattern) ID() string { return "WAW019" }
func (r SecretPattern) Description() string {
	return "Detect hardcoded secrets, API keys, and sensitive credentials"
}

// secretPatterns defines patterns to detect hardcoded secrets
type secretPatternDef struct {
	name    string
	pattern *regexp.Regexp
}

var secretPatterns = []secretPatternDef{
	// AWS Access Key ID (starts with AKIA, ABIA, ACCA, or ASIA)
	{"AWS access key", regexp.MustCompile(`^(A3T[A-Z0-9]|AKIA|ABIA|ACCA|ASIA)[A-Z0-9]{16}$`)},

	// AWS Secret Access Key (40 character base64-like string)
	{"AWS secret key", regexp.MustCompile(`^[A-Za-z0-9/+=]{40}$`)},

	// Private key headers
	{"private key", regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`)},
	{"private key", regexp.MustCompile(`-----BEGIN\s+EC\s+PRIVATE\s+KEY-----`)},
	{"private key", regexp.MustCompile(`-----BEGIN\s+DSA\s+PRIVATE\s+KEY-----`)},
	{"private key", regexp.MustCompile(`-----BEGIN\s+OPENSSH\s+PRIVATE\s+KEY-----`)},

	// Stripe API keys
	{"Stripe API key", regexp.MustCompile(`^sk_(live|test)_[a-zA-Z0-9]{24,}$`)},
	{"Stripe API key", regexp.MustCompile(`^pk_(live|test)_[a-zA-Z0-9]{24,}$`)},

	// GitHub tokens
	{"GitHub token", regexp.MustCompile(`^gh[pousr]_[A-Za-z0-9_]{36,}$`)},
	{"GitHub token", regexp.MustCompile(`^github_pat_[A-Za-z0-9_]{22,}$`)},

	// Slack tokens
	{"Slack token", regexp.MustCompile(`^xox[baprs]-[0-9]{10,}-[0-9]{10,}-[a-zA-Z0-9]{24,}$`)},

	// Generic API key pattern (high entropy strings after "key" fields)
	{"API key", regexp.MustCompile(`^[A-Za-z0-9_\-]{32,}$`)},
}

// sensitiveFieldNames are field names that commonly hold secrets
var sensitiveFieldNames = map[string]bool{
	"password":          true,
	"secret":            true,
	"api_key":           true,
	"apikey":            true,
	"access_key":        true,
	"accesskey":         true,
	"private_key":       true,
	"privatekey":        true,
	"secret_key":        true,
	"secretkey":         true,
	"token":             true,
	"auth_token":        true,
	"authtoken":         true,
	"bearer_token":      true,
	"bearertoken":       true,
	"credentials":       true,
	"connection_string": true,
}

func (r SecretPattern) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		// Check string literals
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		value := strings.Trim(lit.Value, `"`+"`")

		// Skip empty or very short strings
		if len(value) < 10 {
			return true
		}

		// Check against secret patterns
		for _, sp := range secretPatterns {
			if sp.pattern.MatchString(value) {
				// Skip AWS secret key pattern for common safe strings
				if sp.name == "AWS secret key" && isSafeString(value) {
					continue
				}
				// Skip generic API key pattern unless it looks high-entropy
				if sp.name == "API key" && !isHighEntropy(value) {
					continue
				}

				pos := fset.Position(lit.Pos())
				issues = append(issues, Issue{
					Rule:       r.ID(),
					Message:    fmt.Sprintf("Potential %s detected - avoid hardcoding secrets", sp.name),
					Suggestion: "Use AWS Secrets Manager, Parameter Store, or environment variables",
					File:       pos.Filename,
					Line:       pos.Line,
					Column:     pos.Column,
					Severity:   SeverityError,
				})
				return true // Only report first match per string
			}
		}

		return true
	})

	// Check for sensitive field names with literal string values
	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Check if key is a sensitive field name
		var keyName string
		switch key := kv.Key.(type) {
		case *ast.Ident:
			keyName = strings.ToLower(key.Name)
		case *ast.BasicLit:
			if key.Kind == token.STRING {
				keyName = strings.ToLower(strings.Trim(key.Value, `"`))
			}
		}

		if !sensitiveFieldNames[keyName] {
			return true
		}

		// Check if value is a string literal (not a reference or intrinsic)
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true // Variable reference is ok
		}

		value := strings.Trim(lit.Value, `"`)

		// Skip empty, short, or placeholder values
		if len(value) < 8 {
			return true
		}
		if isPlaceholder(value) {
			return true
		}

		pos := fset.Position(lit.Pos())
		issues = append(issues, Issue{
			Rule:       r.ID(),
			Message:    fmt.Sprintf("Hardcoded value in sensitive field '%s' - avoid storing secrets in code", keyName),
			Suggestion: "Use AWS Secrets Manager, Parameter Store, or environment variables",
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   SeverityError,
		})

		return true
	})

	return issues
}

// isSafeString checks if a string is likely safe (not a secret)
func isSafeString(s string) bool {
	// Common safe patterns
	safePatterns := []string{
		"arn:aws:",       // ARNs
		"${",             // Intrinsic substitutions
		"AWS::",          // Pseudo-parameters
		"http://",        // URLs
		"https://",       // URLs
		"s3://",          // S3 URIs
		"ecs-tasks",      // Common service values
		"lambda",         // Common service values
		"logs.",          // CloudWatch patterns
		"events.",        // EventBridge patterns
		"ecr.",           // ECR patterns
		".amazonaws.com", // AWS endpoints
	}

	for _, pattern := range safePatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}

	return false
}

// isHighEntropy checks if a string has high entropy (likely a secret)
func isHighEntropy(s string) bool {
	// Simple entropy check: count unique character types
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	// High entropy if has at least 3 character types and length >= 32
	count := 0
	if hasLower {
		count++
	}
	if hasUpper {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}

	return count >= 3 && len(s) >= 32
}

// isPlaceholder checks if a string looks like a placeholder
func isPlaceholder(s string) bool {
	s = strings.ToLower(s)
	placeholders := []string{
		"changeme",
		"placeholder",
		"example",
		"your-",
		"my-",
		"todo",
		"fixme",
		"<",
		">",
		"xxx",
		"dummy",
		"test",
	}

	for _, p := range placeholders {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
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
		UndefinedReference{},
		UnusedIntrinsicsImport{},
		AvoidExplicitRef{},
		AvoidExplicitGetAtt{},
		AvoidPointerAssignment{},
		PreferJsonType{},
		SecretPattern{},
	}
}
