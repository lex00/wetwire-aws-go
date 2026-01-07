// Package runner provides runtime execution of Go packages to extract resource values.
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
func ExtractValues(pkgPath string, resources map[string]wetwire.DiscoveredResource) (map[string]map[string]any, error) {
	if len(resources) == 0 {
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

	// Calculate import path
	importPath := modInfo.ModulePath

	// Get the first var name (needed to force the import)
	var firstVar string
	varNames := make([]string, 0, len(resources))
	for name := range resources {
		varNames = append(varNames, name)
		if firstVar == "" {
			firstVar = name
		}
	}

	// Create temp directory for runner
	tmpDir, err := os.MkdirTemp("", "wetwire-runner-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate runner main.go
	runnerPath := filepath.Join(tmpDir, "main.go")
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

	// Create go.mod for the runner
	goModPath := filepath.Join(tmpDir, "go.mod")

	// Build replace directives - start with the user's package
	var replaceDirectives strings.Builder
	replaceDirectives.WriteString(fmt.Sprintf("replace %s => %s\n", modInfo.ModulePath, absPath))

	// Add any replace directives from the target package's go.mod
	// (e.g., for local development dependencies like github.com/lex00/wetwire-aws-go)
	for _, repl := range modInfo.Replaces {
		// Resolve relative paths to absolute
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

	// Find Go executable - check common paths if not in PATH
	goBin := findGoBinary()

	// Run go mod tidy
	tidyCmd := exec.Command(goBin, "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %w\n%s", err, output)
	}

	// Run the program with var names as arguments
	args := append([]string{"run", "main.go"}, varNames...)
	runCmd := exec.Command(goBin, args...)
	runCmd.Dir = tmpDir

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

// findGoBinary locates the Go executable.
// It first checks PATH, then common installation locations.
func findGoBinary() string {
	// Check if go is in PATH
	if path, err := exec.LookPath("go"); err == nil {
		return path
	}

	// Check common locations
	commonPaths := []string{
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
		"/usr/bin/go",
		"/usr/local/bin/go",
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fall back to "go" and let exec fail with a clearer message
	return "go"
}

// goModInfo contains parsed go.mod information.
type goModInfo struct {
	ModulePath string
	GoModDir   string
	Replaces   []string // replace directive lines
}

// findGoModInfo reads go.mod to find the module path and replace directives.
func findGoModInfo(dir string) (*goModInfo, error) {
	// Walk up to find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err == nil {
			info := &goModInfo{GoModDir: dir}

			// Parse go.mod
			lines := strings.Split(string(data), "\n")
			inReplaceBlock := false

			for _, line := range lines {
				trimmed := strings.TrimSpace(line)

				// Look for module directive
				if strings.HasPrefix(trimmed, "module ") {
					info.ModulePath = strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
				}

				// Handle replace block: replace ( ... )
				if trimmed == "replace (" {
					inReplaceBlock = true
					continue
				}
				if inReplaceBlock {
					if trimmed == ")" {
						inReplaceBlock = false
						continue
					}
					// Inside replace block, each line is a replace directive
					if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
						info.Replaces = append(info.Replaces, "replace "+trimmed)
					}
					continue
				}

				// Handle single-line replace directives
				if strings.HasPrefix(trimmed, "replace ") && !strings.HasPrefix(trimmed, "replace (") {
					info.Replaces = append(info.Replaces, trimmed)
				}
			}

			if info.ModulePath == "" {
				return nil, fmt.Errorf("no module directive in go.mod")
			}
			return info, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no go.mod found")
		}
		dir = parent
	}
}

// resolveReplacePath converts a relative replace path to absolute.
func resolveReplacePath(replaceLine, goModDir string) string {
	// Parse: replace old => new
	parts := strings.Split(replaceLine, " => ")
	if len(parts) != 2 {
		return replaceLine
	}

	oldPart := parts[0]
	newPart := strings.TrimSpace(parts[1])

	// If newPart is a relative path, make it absolute
	if strings.HasPrefix(newPart, ".") || strings.HasPrefix(newPart, "/") {
		if !filepath.IsAbs(newPart) {
			absPath := filepath.Join(goModDir, newPart)
			return oldPart + " => " + absPath
		}
	}
	return replaceLine
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

	// Calculate import path
	importPath := modInfo.ModulePath

	// Get the first var name (needed to force the import)
	firstVar := varNames[0]

	// Create temp directory for runner
	tmpDir, err := os.MkdirTemp("", "wetwire-runner-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate runner main.go
	runnerPath := filepath.Join(tmpDir, "main.go")
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

	// Create go.mod for the runner
	goModPath := filepath.Join(tmpDir, "go.mod")

	// Build replace directives - start with the user's package
	var replaceDirectives strings.Builder
	replaceDirectives.WriteString(fmt.Sprintf("replace %s => %s\n", modInfo.ModulePath, absPath))

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

	// Find Go executable
	goBin := findGoBinary()

	// Run go mod tidy
	tidyCmd := exec.Command(goBin, "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %w\n%s", err, output)
	}

	// Run the program with var names as arguments
	args := append([]string{"run", "main.go"}, varNames...)
	runCmd := exec.Command(goBin, args...)
	runCmd.Dir = tmpDir

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
