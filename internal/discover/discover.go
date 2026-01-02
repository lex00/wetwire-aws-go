// Package discover provides AST-based discovery of CloudFormation resource declarations.
//
// It parses Go source files looking for package-level variable declarations
// of the form:
//
//	var MyBucket = s3.Bucket{...}
//
// and extracts resource metadata including dependencies on other resources.
package discover

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// knownResourcePackages maps package names to CloudFormation service prefixes.
// This is populated by the code generator.
var knownResourcePackages = map[string]string{
	"s3":             "AWS::S3",
	"iam":            "AWS::IAM",
	"ec2":            "AWS::EC2",
	"lambda":         "AWS::Lambda",
	"dynamodb":       "AWS::DynamoDB",
	"sqs":            "AWS::SQS",
	"sns":            "AWS::SNS",
	"apigateway":     "AWS::ApiGateway",
	"rds":            "AWS::RDS",
	"ecs":            "AWS::ECS",
	"eks":            "AWS::EKS",
	"cloudfront":     "AWS::CloudFront",
	"route53":        "AWS::Route53",
	"kinesis":        "AWS::Kinesis",
	"events":         "AWS::Events",
	"logs":           "AWS::Logs",
	"kms":            "AWS::KMS",
	"secretsmanager": "AWS::SecretsManager",
	"ssm":            "AWS::SSM",
	"elasticache":    "AWS::ElastiCache",
	// Add more as needed - code generator will maintain this
}

// Options configures the discovery process.
type Options struct {
	// Packages to scan (e.g., "./infra/...")
	Packages []string
	// Verbose enables debug output
	Verbose bool
}

// Result contains all discovered resources and any errors.
type Result struct {
	// Resources maps logical name to discovered resource
	Resources map[string]wetwire.DiscoveredResource
	// AllVars tracks all package-level var declarations (including non-resources)
	// Used to avoid false positives when checking dependencies
	AllVars map[string]bool
	// Errors encountered during parsing
	Errors []error
}

// Discover scans Go packages for CloudFormation resource declarations.
func Discover(opts Options) (*Result, error) {
	result := &Result{
		Resources: make(map[string]wetwire.DiscoveredResource),
		AllVars:   make(map[string]bool),
	}

	for _, pkg := range opts.Packages {
		if err := discoverPackage(pkg, result, opts); err != nil {
			return nil, fmt.Errorf("discovering %s: %w", pkg, err)
		}
	}

	// Validate dependencies - only flag truly undefined references
	// Skip vars that are defined locally (including property type blocks)
	for name, res := range result.Resources {
		for _, dep := range res.Dependencies {
			// Skip if it's a known resource
			if _, ok := result.Resources[dep]; ok {
				continue
			}
			// Skip if it's a local var declaration (e.g., Tag blocks, property types)
			if result.AllVars[dep] {
				continue
			}
			result.Errors = append(result.Errors, fmt.Errorf(
				"%s:%d: %s references undefined resource %q",
				res.File, res.Line, name, dep,
			))
		}
	}

	return result, nil
}

func discoverPackage(pattern string, result *Result, opts Options) error {
	// Handle ./... pattern
	pattern = strings.TrimSuffix(pattern, "/...")
	recursive := strings.HasSuffix(pattern, "...")
	if recursive {
		pattern = strings.TrimSuffix(pattern, "...")
	}

	// Get absolute path
	absPath, err := filepath.Abs(pattern)
	if err != nil {
		return err
	}

	if recursive {
		return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return discoverDir(path, result, opts)
			}
			return nil
		})
	}

	return discoverDir(absPath, result, opts)
}

func discoverDir(dir string, result *Result, opts Options) error {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		// Directory might not contain Go files
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no Go files") {
			return nil
		}
		return err
	}

	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			discoverFile(fset, filename, file, result, opts)
		}
	}

	return nil
}

func discoverFile(fset *token.FileSet, filename string, file *ast.File, result *Result, opts Options) {
	// Build import map: alias -> package path
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			// Use last component of path
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}
		imports[name] = path
	}

	// Find package-level var declarations
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}

			name := valueSpec.Names[0].Name
			value := valueSpec.Values[0]

			// Track ALL var declarations to avoid false positive undefined references
			result.AllVars[name] = true

			// Check if it's a composite literal (Type{...})
			compLit, ok := value.(*ast.CompositeLit)
			if !ok {
				continue
			}

			// Get the type
			typeName, pkgName := extractTypeName(compLit.Type)
			if typeName == "" {
				continue
			}

			// Check if this is a known resource package
			if _, known := knownResourcePackages[pkgName]; !known {
				continue
			}

			// Skip property types (e.g., Bucket_ServerSideEncryptionRule)
			// These contain "_" and are nested types, not CloudFormation resources
			if strings.Contains(typeName, "_") {
				continue
			}

			// Extract dependencies from field values
			deps := extractDependencies(compLit, imports)

			pos := fset.Position(valueSpec.Pos())
			result.Resources[name] = wetwire.DiscoveredResource{
				Name:         name,
				Type:         fmt.Sprintf("%s.%s", pkgName, typeName),
				Package:      file.Name.Name,
				File:         filename,
				Line:         pos.Line,
				Dependencies: deps,
			}
		}
	}
}

// extractTypeName extracts the type name and package from a type expression.
// For s3.Bucket, returns ("Bucket", "s3").
func extractTypeName(expr ast.Expr) (typeName, pkgName string) {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		// pkg.Type
		if ident, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name, ident.Name
		}
	case *ast.Ident:
		// Just Type (unqualified)
		return t.Name, ""
	}
	return "", ""
}

// extractDependencies finds references to other resources in composite literal fields.
// It looks for patterns like:
//   - OtherResource (identifier reference)
//   - OtherResource.Arn (selector for AttrRef)
func extractDependencies(lit *ast.CompositeLit, imports map[string]string) []string {
	var deps []string
	seen := make(map[string]bool)

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		// Recursively find dependencies in the value
		findDeps(kv.Value, &deps, seen, imports)
	}

	return deps
}

func findDeps(expr ast.Expr, deps *[]string, seen map[string]bool, imports map[string]string) {
	switch v := expr.(type) {
	case *ast.Ident:
		// Could be a reference to another resource
		// Skip if it's a known package or common identifier
		name := v.Name
		if _, isImport := imports[name]; isImport {
			return
		}
		if isCommonIdent(name) {
			return
		}
		// Heuristic: starts with uppercase = likely a resource reference
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			if !seen[name] {
				*deps = append(*deps, name)
				seen[name] = true
			}
		}

	case *ast.SelectorExpr:
		// Could be Resource.Attr or pkg.Type
		if ident, ok := v.X.(*ast.Ident); ok {
			name := ident.Name
			// Skip package selectors
			if _, isImport := imports[name]; isImport {
				return
			}
			// This is likely Resource.Attribute
			if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
				if !seen[name] {
					*deps = append(*deps, name)
					seen[name] = true
				}
			}
		}

	case *ast.CompositeLit:
		// Nested struct, check its elements
		for _, elt := range v.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				findDeps(kv.Value, deps, seen, imports)
			} else {
				findDeps(elt, deps, seen, imports)
			}
		}

	case *ast.UnaryExpr:
		// Handle &Type{...}
		findDeps(v.X, deps, seen, imports)

	case *ast.CallExpr:
		// Handle function calls - check arguments
		for _, arg := range v.Args {
			findDeps(arg, deps, seen, imports)
		}

	case *ast.SliceExpr:
		findDeps(v.X, deps, seen, imports)

	case *ast.IndexExpr:
		findDeps(v.X, deps, seen, imports)
		findDeps(v.Index, deps, seen, imports)
	}
}

// isCommonIdent returns true for identifiers that are likely not resource names.
func isCommonIdent(name string) bool {
	common := map[string]bool{
		// Go built-ins
		"true": true, "false": true, "nil": true,
		"string": true, "int": true, "bool": true, "float64": true,
		"any": true, "error": true,

		// Intrinsic function types (from intrinsics package)
		"Ref": true, "Sub": true, "Join": true, "GetAtt": true,
		"Select": true, "Split": true, "If": true, "Equals": true,
		"And": true, "Or": true, "Not": true, "Condition": true,
		"FindInMap": true, "Base64": true, "Cidr": true, "GetAZs": true,
		"ImportValue": true, "Transform": true, "Json": true,

		// Pseudo-parameter constants (from intrinsics package)
		"AWS_ACCOUNT_ID": true, "AWS_NOTIFICATION_ARNS": true,
		"AWS_NO_VALUE": true, "AWS_PARTITION": true,
		"AWS_REGION": true, "AWS_STACK_ID": true,
		"AWS_STACK_NAME": true, "AWS_URL_SUFFIX": true,
	}
	return common[name]
}
