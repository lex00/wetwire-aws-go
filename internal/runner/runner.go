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

	pkg "{{.ImportPath}}"
)

// Resource interface for CloudFormation resources
type Resource interface {
	ResourceType() string
}

func main() {
	// Use reflection to find all exported variables in the package
	pkgValue := reflect.ValueOf(pkg.{{.FirstVar}})
	_ = pkgValue // Force import

	// The resources are discovered via var names passed as arguments
	varNames := os.Args[1:]

	result := make(map[string]map[string]any)

	for _, name := range varNames {
		// Get the variable by evaluating it
		value := getVar(name)
		if value == nil {
			continue
		}

		// Serialize to JSON and back to get a map
		data, err := json.Marshal(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling %s: %v\n", name, err)
			continue
		}

		var props map[string]any
		if err := json.Unmarshal(data, &props); err != nil {
			fmt.Fprintf(os.Stderr, "Error unmarshaling %s: %v\n", name, err)
			continue
		}

		result[name] = props
	}

	output, _ := json.Marshal(result)
	fmt.Println(string(output))
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
		return nil, fmt.Errorf("parsing output: %w\noutput: %s", err, stdout.String())
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
