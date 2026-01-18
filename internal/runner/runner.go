// Package runner provides runtime execution of Go packages to extract resource values.
//
// The runner generates a temporary Go program that imports the user's infrastructure
// package, executes it, and serializes the discovered resources to JSON. This allows
// wetwire-aws to extract the actual values of CloudFormation resources at build time.
//
// # How It Works
//
// 1. Generate a temporary main.go that imports the target package
// 2. The generated code uses reflection to find all package-level variables
// 3. Variables that implement resource interfaces are serialized to JSON
// 4. The runner executes "go run" on the generated program and captures output
// 5. The JSON output is parsed back into Go structs for template generation
//
// # Template-Based Approach
//
// The runner uses text/template to generate the extraction program. This approach
// allows the extraction logic to run in the user's module context with access to
// their go.mod dependencies, ensuring type compatibility.
//
// # Example
//
//	result, err := runner.Run(runner.Options{
//	    Packages: []string{"./infra"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for name, resource := range result.Resources {
//	    fmt.Printf("%s: %s\n", name, resource.Type)
//	}
package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// runnerTemplate is the Go program template that extracts resources at runtime.
var runnerTemplate = template.Must(template.New("runner").Parse(`// Auto-generated runner for resource extraction
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/lex00/wetwire-aws-go/intrinsics"
	pkg "{{.ImportPath}}"
)

// Resource interface for CloudFormation resources
type Resource interface {
	ResourceType() string
}

// parameterNames maps Parameter signature to logical name
var parameterNames = make(map[string]string)

// resourceSignatures maps resource JSON signature to logical name
var resourceSignatures = make(map[string]string)

func main() {
	// Use reflection to find all exported variables in the package
	pkgValue := reflect.ValueOf(pkg.{{.FirstVar}})
	_ = pkgValue // Force import

	// The resources are discovered via var names passed as arguments
	varNames := os.Args[1:]

	// First pass: collect all Parameter values and build name lookup
	// Also collect resources for Ref generation
	for _, name := range varNames {
		value := getVar(name)
		if value == nil {
			continue
		}
		if param, ok := value.(intrinsics.Parameter); ok {
			// Create a signature from the parameter's exported fields
			sig := paramSignature(param)
			parameterNames[sig] = name
		}
		// Check if it's a resource (has ResourceType method)
		if res, ok := value.(Resource); ok {
			// Create signature from ResourceType + JSON serialization
			sig := resourceSignature(res)
			resourceSignatures[sig] = name
		}
	}

	result := make(map[string]map[string]any)

	for _, name := range varNames {
		// Get the variable by evaluating it
		value := getVar(name)
		if value == nil {
			continue
		}

		var props map[string]any

		// Handle Parameter type specially - use ToDefinition() instead of MarshalJSON
		// because MarshalJSON returns a Ref, but we need the full definition
		if param, ok := value.(intrinsics.Parameter); ok {
			props = param.ToDefinition()
			result[name] = props
			continue
		}

		// For Mapping type, convert directly
		if mapping, ok := value.(intrinsics.Mapping); ok {
			props = make(map[string]any)
			for k, v := range mapping {
				props[k] = v
			}
			result[name] = props
			continue
		}

		// Serialize using custom function that handles Parameter refs
		serialized := serializeValue(reflect.ValueOf(value))
		if m, ok := serialized.(map[string]any); ok {
			props = m
		} else if serialized != nil {
			// Handle other return types (e.g., map[string][]any for intrinsics)
			// by converting through JSON
			data, err := json.Marshal(serialized)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling %s: %v\n", name, err)
				continue
			}
			if err := json.Unmarshal(data, &props); err != nil {
				fmt.Fprintf(os.Stderr, "Error unmarshaling %s: %v\n", name, err)
				continue
			}
		} else {
			// serialized is nil, skip this variable
			continue
		}

		result[name] = props
	}

	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}

// paramSignature creates a unique signature for a Parameter based on its fields
func paramSignature(p intrinsics.Parameter) string {
	// Use JSON encoding of exported fields as signature
	data, _ := json.Marshal(p.ToDefinition())
	return string(data)
}

// resourceSignature creates a unique signature for a Resource
func resourceSignature(r Resource) string {
	// Use ResourceType + JSON of the struct as signature
	data, _ := json.Marshal(r)
	return r.ResourceType() + ":" + string(data)
}

// serializeValue converts a value to JSON-compatible format, handling Parameters specially
// When nested=true, Resources are converted to Refs (for use inside Outputs, etc.)
func serializeValue(v reflect.Value) any {
	return serializeValueNested(v, false)
}

func serializeValueNested(v reflect.Value, nested bool) any {
	if !v.IsValid() {
		return nil
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return serializeValueNested(v.Elem(), nested)
	}

	// Handle interfaces
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return serializeValueNested(v.Elem(), nested)
	}

	// Check if this is a Parameter - convert to Ref with name lookup
	if v.Type().String() == "intrinsics.Parameter" {
		param := v.Interface().(intrinsics.Parameter)
		sig := paramSignature(param)
		if name, found := parameterNames[sig]; found {
			return map[string]any{"Ref": name}
		}
		// Fallback - shouldn't happen if all parameters are discovered
		return map[string]any{"Ref": ""}
	}

	// Check if this is a Resource - convert to Ref with name lookup (only when nested)
	if nested && v.CanInterface() {
		if res, ok := v.Interface().(Resource); ok {
			sig := resourceSignature(res)
			if name, found := resourceSignatures[sig]; found {
				return map[string]any{"Ref": name}
			}
		}
	}

	// Handle intrinsic types that can contain nested Parameters
	// We serialize these manually to ensure Parameters get properly resolved
	if v.CanInterface() {
		iface := v.Interface()

		// Handle intrinsics with nested values that might contain Parameters
		switch val := iface.(type) {
		case intrinsics.Equals:
			return map[string][]any{
				"Fn::Equals": {
					serializeValueNested(reflect.ValueOf(val.Value1), true),
					serializeValueNested(reflect.ValueOf(val.Value2), true),
				},
			}
		case intrinsics.If:
			return map[string][]any{
				"Fn::If": {
					val.Condition,
					serializeValueNested(reflect.ValueOf(val.ValueIfTrue), true),
					serializeValueNested(reflect.ValueOf(val.ValueIfFalse), true),
				},
			}
		case intrinsics.Select:
			return map[string][]any{
				"Fn::Select": {
					val.Index,
					serializeValueNested(reflect.ValueOf(val.List), true),
				},
			}
		case intrinsics.And:
			conditions := make([]any, len(val.Conditions))
			for i, c := range val.Conditions {
				conditions[i] = serializeValueNested(reflect.ValueOf(c), true)
			}
			return map[string][]any{"Fn::And": conditions}
		case intrinsics.Or:
			conditions := make([]any, len(val.Conditions))
			for i, c := range val.Conditions {
				conditions[i] = serializeValueNested(reflect.ValueOf(c), true)
			}
			return map[string][]any{"Fn::Or": conditions}
		case intrinsics.Not:
			return map[string][]any{
				"Fn::Not": {serializeValueNested(reflect.ValueOf(val.Condition), true)},
			}
		case intrinsics.Join:
			values := make([]any, len(val.Values))
			for i, v := range val.Values {
				values[i] = serializeValueNested(reflect.ValueOf(v), true)
			}
			return map[string][]any{
				"Fn::Join": {val.Delimiter, values},
			}
		case intrinsics.SubWithMap:
			vars := make(map[string]any)
			for k, v := range val.Variables {
				vars[k] = serializeValueNested(reflect.ValueOf(v), true)
			}
			return map[string][]any{
				"Fn::Sub": {val.String, vars},
			}
		case intrinsics.Base64:
			return map[string]any{
				"Fn::Base64": serializeValueNested(reflect.ValueOf(val.Value), true),
			}
		case intrinsics.ImportValue:
			return map[string]any{
				"Fn::ImportValue": serializeValueNested(reflect.ValueOf(val.ExportName), true),
			}
		case intrinsics.FindInMap:
			return map[string][]any{
				"Fn::FindInMap": {
					val.MapName,
					serializeValueNested(reflect.ValueOf(val.TopKey), true),
					serializeValueNested(reflect.ValueOf(val.SecondKey), true),
				},
			}
		case intrinsics.Split:
			return map[string][]any{
				"Fn::Split": {
					val.Delimiter,
					serializeValueNested(reflect.ValueOf(val.Source), true),
				},
			}
		case intrinsics.Cidr:
			return map[string][]any{
				"Fn::Cidr": {
					serializeValueNested(reflect.ValueOf(val.IPBlock), true),
					serializeValueNested(reflect.ValueOf(val.Count), true),
					serializeValueNested(reflect.ValueOf(val.CidrBits), true),
				},
			}
		case intrinsics.Tag:
			return map[string]any{
				"Key":   val.Key,
				"Value": serializeValueNested(reflect.ValueOf(val.Value), true),
			}
		case intrinsics.Transform:
			params := make(map[string]any)
			for k, v := range val.Parameters {
				params[k] = serializeValueNested(reflect.ValueOf(v), true)
			}
			return map[string]any{
				"Fn::Transform": map[string]any{
					"Name":       val.Name,
					"Parameters": params,
				},
			}
		}

		// For other MarshalJSON types (Ref, GetAtt, Sub, GetAZs, Condition)
		// that don't contain nested values, use their MarshalJSON directly
		_, isParam := iface.(intrinsics.Parameter)
		_, isRes := iface.(Resource)
		_, isOutput := iface.(intrinsics.Output)
		if !isParam && !isRes && !isOutput {
			if marshaler, ok := iface.(json.Marshaler); ok {
				data, err := marshaler.MarshalJSON()
				if err == nil {
					var result any
					if json.Unmarshal(data, &result) == nil {
						return result
					}
				}
			}
		}
	}

	switch v.Kind() {
	case reflect.Struct:
		result := make(map[string]any)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			// Get JSON tag name
			name := field.Name
			if tag := field.Tag.Get("json"); tag != "" {
				parts := splitFirst(tag, ',')
				if parts != "-" && parts != "" {
					name = parts
				} else if parts == "-" {
					continue
				}
			}
			fieldVal := v.Field(i)
			if isZero(fieldVal) {
				continue
			}
			// All struct fields are nested
			serialized := serializeValueNested(fieldVal, true)
			if serialized != nil {
				result[name] = serialized
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result

	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return nil
		}
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = serializeValueNested(v.Index(i), true)
		}
		return result

	case reflect.Map:
		if v.Len() == 0 {
			return nil
		}
		result := make(map[string]any)
		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key().Interface())
			result[key] = serializeValueNested(iter.Value(), true)
		}
		return result

	case reflect.String:
		s := v.String()
		if s == "" {
			return nil
		}
		return s

	case reflect.Bool:
		return v.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()

	case reflect.Float32, reflect.Float64:
		return v.Float()

	default:
		if v.CanInterface() {
			return v.Interface()
		}
		return nil
	}
}

func splitFirst(s string, sep byte) string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return s[:i]
		}
	}
	return s
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Struct:
		// Check if all fields are zero
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func getVar(name string) any {
	switch name {
{{range .VarNames}}	case "{{.}}":
		return pkg.{{.}}
{{end}}	}
	return nil
}
`))

// ExtractValues runs a generated Go program to extract resource values.
// Deprecated: Use ExtractAll instead which supports Parameters, Outputs, etc.
func ExtractValues(pkgPath string, resources map[string]wetwire.DiscoveredResource) (map[string]map[string]any, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	// Collect var names from resources
	varNames := make([]string, 0, len(resources))
	for name := range resources {
		varNames = append(varNames, name)
	}

	// Delegate to the common extraction function
	return extractVarValues(pkgPath, varNames)
}

// ExtractedValues contains all extracted values organized by type.
type ExtractedValues struct {
	Resources  map[string]map[string]any
	Parameters map[string]map[string]any
	Outputs    map[string]map[string]any
	Mappings   map[string]any
	Conditions map[string]any
}

// ExtractAll extracts values for all discovered components.
func ExtractAll(pkgPath string,
	resources map[string]wetwire.DiscoveredResource,
	parameters map[string]wetwire.DiscoveredParameter,
	outputs map[string]wetwire.DiscoveredOutput,
	mappings map[string]wetwire.DiscoveredMapping,
	conditions map[string]wetwire.DiscoveredCondition,
) (*ExtractedValues, error) {
	// Collect all variable names
	varNames := make([]string, 0)
	for name := range resources {
		varNames = append(varNames, name)
	}
	for name := range parameters {
		varNames = append(varNames, name)
	}
	for name := range outputs {
		varNames = append(varNames, name)
	}
	for name := range mappings {
		varNames = append(varNames, name)
	}
	for name := range conditions {
		varNames = append(varNames, name)
	}

	if len(varNames) == 0 {
		return &ExtractedValues{
			Resources:  make(map[string]map[string]any),
			Parameters: make(map[string]map[string]any),
			Outputs:    make(map[string]map[string]any),
			Mappings:   make(map[string]any),
			Conditions: make(map[string]any),
		}, nil
	}

	// Extract all values using the generic extractor
	allValues, err := extractVarValues(pkgPath, varNames)
	if err != nil {
		return nil, err
	}

	// Organize by type
	result := &ExtractedValues{
		Resources:  make(map[string]map[string]any),
		Parameters: make(map[string]map[string]any),
		Outputs:    make(map[string]map[string]any),
		Mappings:   make(map[string]any),
		Conditions: make(map[string]any),
	}

	for name := range resources {
		if val, ok := allValues[name]; ok {
			result.Resources[name] = val
		}
	}
	for name := range parameters {
		if val, ok := allValues[name]; ok {
			result.Parameters[name] = val
		}
	}
	for name := range outputs {
		if val, ok := allValues[name]; ok {
			result.Outputs[name] = val
		}
	}
	for name := range mappings {
		if val, ok := allValues[name]; ok {
			result.Mappings[name] = val
		}
	}
	for name := range conditions {
		if val, ok := allValues[name]; ok {
			result.Conditions[name] = val
		}
	}

	return result, nil
}

// extractVarValues extracts values for a list of variable names.
func extractVarValues(pkgPath string, varNames []string) (map[string]map[string]any, error) {
	if len(varNames) == 0 {
		return nil, nil
	}

	// Get absolute package path
	absPath, err := filepath.Abs(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path: %w", err)
	}

	// Read go.mod to find the module path and replace directives
	modInfo, err := findGoModInfo(absPath)
	if err != nil {
		return nil, fmt.Errorf("finding module info: %w", err)
	}

	// Get the first var name (needed to force the import)
	firstVar := varNames[0]

	// Find Go executable
	goBin := findGoBinary()

	// Decide which mode to use based on vendor directory presence
	useVendorMode := shouldUseSubdirRunner(modInfo.GoModDir)

	var runnerDir string
	var cleanup func()
	var goRunArgs []string
	var workDir string

	// Calculate import path - depends on mode
	var importPath string

	if useVendorMode {
		// Vendor mode: create _wetwire_runner subdir in module directory
		runnerDir, cleanup, err = createRunnerSubdir(modInfo.GoModDir)
		if err != nil {
			return nil, err
		}
		defer cleanup()

		// Use -mod=vendor for offline builds
		goRunArgs = []string{"run", "-mod=vendor", "./_wetwire_runner"}
		workDir = modInfo.GoModDir

		// Calculate import path - includes subpackage path if not at module root
		importPath = modInfo.ModulePath
		if relPath, err := filepath.Rel(modInfo.GoModDir, absPath); err == nil && relPath != "." {
			importPath = modInfo.ModulePath + "/" + filepath.ToSlash(relPath)
		}
	} else if modInfo.Synthetic {
		// Synthetic mode: no go.mod found, create self-contained runner
		runnerDir, err = os.MkdirTemp("", "wetwire-runner-*")
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
		defer func() { _ = os.RemoveAll(runnerDir) }()

		// Use -mod=mod to allow automatic dependency resolution
		goRunArgs = []string{"run", "-mod=mod", "main.go"}
		workDir = runnerDir

		// Copy user's Go files to a subdirectory
		userPkgDir := filepath.Join(runnerDir, "userpkg")
		if err := os.MkdirAll(userPkgDir, 0755); err != nil {
			return nil, fmt.Errorf("creating userpkg dir: %w", err)
		}

		files, err := os.ReadDir(absPath)
		if err != nil {
			return nil, fmt.Errorf("reading source dir: %w", err)
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") {
				continue
			}
			srcPath := filepath.Join(absPath, f.Name())
			dstPath := filepath.Join(userPkgDir, f.Name())
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", f.Name(), err)
			}
			if err := os.WriteFile(dstPath, content, 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", f.Name(), err)
			}
		}

		// Import path is within the runner module
		importPath = "runner/userpkg"

		// Create go.mod that directly includes the user's package
		goModPath := filepath.Join(runnerDir, "go.mod")
		goModContent := `module runner

go 1.23.0

require github.com/lex00/wetwire-aws-go v1.9.0
`
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			return nil, fmt.Errorf("writing go.mod: %w", err)
		}

		// Run go mod tidy to resolve dependencies
		tidyCmd := exec.Command(goBin, "mod", "tidy")
		tidyCmd.Dir = runnerDir
		if output, err := tidyCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go mod tidy failed: %w\n%s", err, output)
		}

		// Download all dependencies to populate go.sum
		downloadCmd := exec.Command(goBin, "mod", "download")
		downloadCmd.Dir = runnerDir
		if output, err := downloadCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go mod download failed: %w\n%s", err, output)
		}
	} else {
		// Normal mode: create temp directory
		runnerDir, err = os.MkdirTemp("", "wetwire-runner-*")
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
		defer func() { _ = os.RemoveAll(runnerDir) }()

		goRunArgs = []string{"run", "main.go"}
		workDir = runnerDir

		// Calculate import path - includes subpackage path if not at module root
		importPath = modInfo.ModulePath
		if relPath, err := filepath.Rel(modInfo.GoModDir, absPath); err == nil && relPath != "." {
			importPath = modInfo.ModulePath + "/" + filepath.ToSlash(relPath)
		}
	}

	// Generate runner main.go
	runnerPath := filepath.Join(runnerDir, "main.go")
	f, err := os.Create(runnerPath)
	if err != nil {
		return nil, fmt.Errorf("creating runner file: %w", err)
	}

	data := struct {
		ImportPath string
		FirstVar   string
		VarNames   []string
	}{
		ImportPath: importPath,
		FirstVar:   firstVar,
		VarNames:   varNames,
	}

	if err := runnerTemplate.Execute(f, data); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("executing template: %w", err)
	}
	_ = f.Close()

	// Create go.mod for normal mode (not vendor, not synthetic)
	if !useVendorMode && !modInfo.Synthetic {
		// In normal mode, we need go.mod in the temp directory
		workDir = runnerDir

		// Create go.mod for the runner
		goModPath := filepath.Join(runnerDir, "go.mod")

		// Build replace directives - point to module root (where go.mod is), not package path
		var replaceDirectives strings.Builder
		replaceDirectives.WriteString(fmt.Sprintf("replace %s => %s\n", modInfo.ModulePath, modInfo.GoModDir))

		// Add any replace directives from the target package's go.mod
		for _, repl := range modInfo.Replaces {
			resolved := resolveReplacePath(repl, modInfo.GoModDir)
			replaceDirectives.WriteString(resolved + "\n")
		}

		goModContent := fmt.Sprintf(`module runner

go 1.23.0

require %s v0.0.0

%s`, modInfo.ModulePath, replaceDirectives.String())
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			return nil, fmt.Errorf("writing go.mod: %w", err)
		}

		// Run go mod tidy (only needed in normal mode)
		tidyCmd := exec.Command(goBin, "mod", "tidy")
		tidyCmd.Dir = runnerDir
		if output, err := tidyCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go mod tidy failed: %w\n%s", err, output)
		}
	}

	// Run the program with var names as arguments
	args := append(goRunArgs, varNames...)
	runCmd := exec.Command(goBin, args...)
	runCmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr

	if err := runCmd.Run(); err != nil {
		return nil, fmt.Errorf("running extractor: %w\n%s", err, stderr.String())
	}

	// Parse the output
	var result map[string]map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing output: %w\noutput: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	return result, nil
}
