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
	"serverless":     "AWS::Serverless",
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
	// Parameters maps logical name to discovered parameter
	Parameters map[string]wetwire.DiscoveredParameter
	// Outputs maps logical name to discovered output
	Outputs map[string]wetwire.DiscoveredOutput
	// Mappings maps logical name to discovered mapping
	Mappings map[string]wetwire.DiscoveredMapping
	// Conditions maps logical name to discovered condition
	Conditions map[string]wetwire.DiscoveredCondition
	// AllVars tracks all package-level var declarations (including non-resources)
	// Used to avoid false positives when checking dependencies
	AllVars map[string]bool
	// VarAttrRefs tracks AttrRefUsages for all variables (including property types)
	// Key is variable name, value includes AttrRefs and referenced var names with field paths
	VarAttrRefs map[string]VarAttrRefInfo
	// Errors encountered during parsing
	Errors []error
}

// VarAttrRefInfo tracks AttrRef usages and variable references for a single variable
type VarAttrRefInfo struct {
	AttrRefs []wetwire.AttrRefUsage
	// VarRefs maps field path to referenced variable name
	VarRefs map[string]string
}

// Discover scans Go packages for CloudFormation resource declarations.
func Discover(opts Options) (*Result, error) {
	result := &Result{
		Resources:   make(map[string]wetwire.DiscoveredResource),
		Parameters:  make(map[string]wetwire.DiscoveredParameter),
		Outputs:     make(map[string]wetwire.DiscoveredOutput),
		Mappings:    make(map[string]wetwire.DiscoveredMapping),
		Conditions:  make(map[string]wetwire.DiscoveredCondition),
		AllVars:     make(map[string]bool),
		VarAttrRefs: make(map[string]VarAttrRefInfo),
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

			pos := fset.Position(valueSpec.Pos())

			// Check for intrinsic types (Parameter, Output, Mapping, Condition types)
			if isIntrinsicPackage(pkgName, imports) || pkgName == "" {
				switch typeName {
				case "Parameter":
					result.Parameters[name] = wetwire.DiscoveredParameter{
						Name: name,
						File: filename,
						Line: pos.Line,
					}
					continue
				case "Output":
					// Extract AttrRef usages from output fields
					_, attrRefs := extractDependencies(compLit, imports)
					result.Outputs[name] = wetwire.DiscoveredOutput{
						Name:          name,
						File:          filename,
						Line:          pos.Line,
						AttrRefUsages: attrRefs,
					}
					continue
				case "Mapping":
					result.Mappings[name] = wetwire.DiscoveredMapping{
						Name: name,
						File: filename,
						Line: pos.Line,
					}
					continue
				case "Equals", "And", "Or", "Not":
					result.Conditions[name] = wetwire.DiscoveredCondition{
						Name: name,
						Type: typeName,
						File: filename,
						Line: pos.Line,
					}
					continue
				}
			}

			// Extract dependencies, AttrRef usages, and var refs from field values
			// Do this for ALL composite literals, not just resource packages
			deps, attrRefs, varRefs := extractDependenciesWithVarRefs(compLit, imports)

			// Track AttrRefs for all variables (including property types and intrinsics)
			result.VarAttrRefs[name] = VarAttrRefInfo{
				AttrRefs: attrRefs,
				VarRefs:  varRefs,
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

			result.Resources[name] = wetwire.DiscoveredResource{
				Name:          name,
				Type:          fmt.Sprintf("%s.%s", pkgName, typeName),
				Package:       file.Name.Name,
				File:          filename,
				Line:          pos.Line,
				Dependencies:  deps,
				AttrRefUsages: attrRefs,
			}
		}
	}
}

// isIntrinsicPackage checks if the package is the intrinsics package.
func isIntrinsicPackage(pkgName string, imports map[string]string) bool {
	if pkgName == "" {
		// Dot-imported, check if intrinsics is dot-imported
		for alias, path := range imports {
			if alias == "." && strings.HasSuffix(path, "/intrinsics") {
				return true
			}
		}
		return false
	}
	if pkgName == "intrinsics" {
		return true
	}
	// Check if the package alias points to the intrinsics package
	if path, ok := imports[pkgName]; ok {
		return strings.HasSuffix(path, "/intrinsics")
	}
	return false
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
//
// Returns both dependencies and AttrRef usages for GetAtt resolution.
func extractDependencies(lit *ast.CompositeLit, imports map[string]string) ([]string, []wetwire.AttrRefUsage) {
	deps, attrRefs, _ := extractDependenciesWithVarRefs(lit, imports)
	return deps, attrRefs
}

// extractDependenciesWithVarRefs is like extractDependencies but also returns variable references
// with their field paths, for recursive AttrRef resolution.
func extractDependenciesWithVarRefs(lit *ast.CompositeLit, imports map[string]string) ([]string, []wetwire.AttrRefUsage, map[string]string) {
	var deps []string
	var attrRefs []wetwire.AttrRefUsage
	varRefs := make(map[string]string) // field path -> var name
	seen := make(map[string]bool)

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		// Get the field name for the path
		fieldName := ""
		if ident, ok := kv.Key.(*ast.Ident); ok {
			fieldName = ident.Name
		}

		// Recursively find dependencies in the value
		findDepsWithVarRefs(kv.Value, &deps, &attrRefs, varRefs, seen, imports, fieldName)
	}

	return deps, attrRefs, varRefs
}

func findDeps(expr ast.Expr, deps *[]string, attrRefs *[]wetwire.AttrRefUsage, seen map[string]bool, imports map[string]string, fieldPath string) {
	// Delegate to findDepsWithVarRefs with a nil varRefs map
	findDepsWithVarRefs(expr, deps, attrRefs, nil, seen, imports, fieldPath)
}

func findDepsWithVarRefs(expr ast.Expr, deps *[]string, attrRefs *[]wetwire.AttrRefUsage, varRefs map[string]string, seen map[string]bool, imports map[string]string, fieldPath string) {
	switch v := expr.(type) {
	case *ast.Ident:
		// Could be a reference to another resource or variable
		// Skip if it's a known package or common identifier
		name := v.Name
		if _, isImport := imports[name]; isImport {
			return
		}
		if isCommonIdent(name) {
			return
		}
		// Heuristic: starts with uppercase = likely a resource/var reference
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			if !seen[name] {
				*deps = append(*deps, name)
				seen[name] = true
			}
			// Track this variable reference with its field path
			if varRefs != nil && fieldPath != "" {
				varRefs[fieldPath] = name
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
			// This is likely Resource.Attribute (e.g., LambdaRole.Arn)
			if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
				if !seen[name] {
					*deps = append(*deps, name)
					seen[name] = true
				}
				// Record the AttrRef usage for GetAtt resolution
				*attrRefs = append(*attrRefs, wetwire.AttrRefUsage{
					ResourceName: name,
					Attribute:    v.Sel.Name,
					FieldPath:    fieldPath,
				})
			}
		}

	case *ast.CompositeLit:
		// Nested struct, check its elements
		for _, elt := range v.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				// Build nested field path
				nestedPath := fieldPath
				if ident, ok := kv.Key.(*ast.Ident); ok {
					if fieldPath != "" {
						nestedPath = fieldPath + "." + ident.Name
					} else {
						nestedPath = ident.Name
					}
				}
				findDepsWithVarRefs(kv.Value, deps, attrRefs, varRefs, seen, imports, nestedPath)
			} else {
				findDepsWithVarRefs(elt, deps, attrRefs, varRefs, seen, imports, fieldPath)
			}
		}

	case *ast.UnaryExpr:
		// Handle &Type{...}
		findDepsWithVarRefs(v.X, deps, attrRefs, varRefs, seen, imports, fieldPath)

	case *ast.CallExpr:
		// Handle function calls - check arguments
		for _, arg := range v.Args {
			findDepsWithVarRefs(arg, deps, attrRefs, varRefs, seen, imports, fieldPath)
		}

	case *ast.SliceExpr:
		findDepsWithVarRefs(v.X, deps, attrRefs, varRefs, seen, imports, fieldPath)

	case *ast.IndexExpr:
		findDepsWithVarRefs(v.X, deps, attrRefs, varRefs, seen, imports, fieldPath)
		findDepsWithVarRefs(v.Index, deps, attrRefs, varRefs, seen, imports, fieldPath)
	}
}

// ResolveAttrRefs recursively collects all AttrRefUsages for a variable by following
// its variable references and prefixing paths appropriately.
func (r *Result) ResolveAttrRefs(varName string) []wetwire.AttrRefUsage {
	visited := make(map[string]bool)
	return r.resolveAttrRefsRecursive(varName, "", visited)
}

func (r *Result) resolveAttrRefsRecursive(varName, pathPrefix string, visited map[string]bool) []wetwire.AttrRefUsage {
	if visited[varName] {
		return nil
	}
	visited[varName] = true

	info, ok := r.VarAttrRefs[varName]
	if !ok {
		return nil
	}

	var result []wetwire.AttrRefUsage

	// Add direct AttrRefs with path prefix
	for _, ref := range info.AttrRefs {
		fullPath := ref.FieldPath
		if pathPrefix != "" {
			fullPath = pathPrefix + "." + ref.FieldPath
		}
		result = append(result, wetwire.AttrRefUsage{
			ResourceName: ref.ResourceName,
			Attribute:    ref.Attribute,
			FieldPath:    fullPath,
		})
	}

	// Recursively resolve variable references
	for fieldPath, refVarName := range info.VarRefs {
		fullPath := fieldPath
		if pathPrefix != "" {
			fullPath = pathPrefix + "." + fieldPath
		}
		nested := r.resolveAttrRefsRecursive(refVarName, fullPath, visited)
		result = append(result, nested...)
	}

	return result
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
		"Parameter": true, "Output": true, "Mapping": true,

		// Pseudo-parameter constants (from intrinsics package)
		"AWS_ACCOUNT_ID": true, "AWS_NOTIFICATION_ARNS": true,
		"AWS_NO_VALUE": true, "AWS_PARTITION": true,
		"AWS_REGION": true, "AWS_STACK_ID": true,
		"AWS_STACK_NAME": true, "AWS_URL_SUFFIX": true,
	}
	return common[name]
}
