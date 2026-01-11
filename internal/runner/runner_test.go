package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	wetwire "github.com/lex00/wetwire-aws-go"
)

func TestHasVendorDir(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// No vendor - should return false
	if hasVendorDir(tmpDir) {
		t.Error("hasVendorDir should return false when no vendor directory")
	}

	// Create vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.Mkdir(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	// With vendor - should return true
	if !hasVendorDir(tmpDir) {
		t.Error("hasVendorDir should return true when vendor directory exists")
	}
}

func TestCreateRunnerSubdir(t *testing.T) {
	// Create temp directory with basic Go structure
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test

go 1.23
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.Mkdir(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create runner subdir
	runnerDir, cleanup, err := createRunnerSubdir(tmpDir)
	if err != nil {
		t.Fatalf("createRunnerSubdir failed: %v", err)
	}
	defer cleanup()

	// Verify runner directory was created
	expected := filepath.Join(tmpDir, "_wetwire_runner")
	if runnerDir != expected {
		t.Errorf("runnerDir = %q, want %q", runnerDir, expected)
	}

	// Verify directory exists
	if _, err := os.Stat(runnerDir); err != nil {
		t.Errorf("runner directory should exist: %v", err)
	}

	// Run cleanup and verify directory is removed
	cleanup()
	if _, err := os.Stat(runnerDir); !os.IsNotExist(err) {
		t.Error("runner directory should be removed after cleanup")
	}
}

func TestFindGoModInfo_NoGoMod(t *testing.T) {
	// Create temp directory without go.mod
	tmpDir := t.TempDir()

	// Create a simple Go file
	goFile := `package infra

import "github.com/lex00/wetwire-aws-go/resources/s3"

var Bucket = s3.Bucket{BucketName: "test"}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	// findGoModInfo should auto-generate module info when no go.mod found
	info, err := findGoModInfo(tmpDir)
	if err != nil {
		t.Fatalf("findGoModInfo should succeed without go.mod: %v", err)
	}

	// Should have a synthetic module path
	if info.ModulePath == "" {
		t.Error("ModulePath should not be empty")
	}

	// Should indicate this is synthetic
	if !info.Synthetic {
		t.Error("Synthetic flag should be true for auto-generated module info")
	}
}

func TestRunnerModeSelection(t *testing.T) {
	tests := []struct {
		name       string
		hasVendor  bool
		wantSubdir bool // true = use _wetwire_runner subdir, false = use temp dir
	}{
		{"no vendor uses temp dir", false, false},
		{"with vendor uses subdir", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			if tt.hasVendor {
				if err := os.Mkdir(filepath.Join(tmpDir, "vendor"), 0755); err != nil {
					t.Fatal(err)
				}
			}

			gotSubdir := shouldUseSubdirRunner(tmpDir)
			if gotSubdir != tt.wantSubdir {
				t.Errorf("shouldUseSubdirRunner = %v, want %v", gotSubdir, tt.wantSubdir)
			}
		})
	}
}

func TestFindGoModInfo_WithGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod with module directive
	goMod := `module example.com/myproject

go 1.23.0

require github.com/lex00/wetwire-aws-go v1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := findGoModInfo(tmpDir)
	if err != nil {
		t.Fatalf("findGoModInfo failed: %v", err)
	}

	if info.ModulePath != "example.com/myproject" {
		t.Errorf("ModulePath = %q, want %q", info.ModulePath, "example.com/myproject")
	}

	if info.GoModDir != tmpDir {
		t.Errorf("GoModDir = %q, want %q", info.GoModDir, tmpDir)
	}

	if info.Synthetic {
		t.Error("Synthetic should be false when go.mod exists")
	}
}

func TestFindGoModInfo_WithReplaceDirectives(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod with replace directives
	goMod := `module example.com/myproject

go 1.23.0

require github.com/lex00/wetwire-aws-go v1.0.0

replace github.com/lex00/wetwire-aws-go => ../wetwire-aws-go
replace github.com/other/package => /absolute/path
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := findGoModInfo(tmpDir)
	if err != nil {
		t.Fatalf("findGoModInfo failed: %v", err)
	}

	if len(info.Replaces) != 2 {
		t.Errorf("got %d replace directives, want 2", len(info.Replaces))
	}
}

func TestFindGoModInfo_WithReplaceBlock(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod with replace block
	goMod := `module example.com/myproject

go 1.23.0

replace (
	github.com/foo/bar => ../bar
	github.com/baz/qux => /path/to/qux
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := findGoModInfo(tmpDir)
	if err != nil {
		t.Fatalf("findGoModInfo failed: %v", err)
	}

	if len(info.Replaces) != 2 {
		t.Errorf("got %d replace directives, want 2", len(info.Replaces))
	}
}

func TestFindGoModInfo_InSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod at root
	goMod := `module example.com/myproject

go 1.23.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "infra", "networking")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Find go.mod from subdirectory
	info, err := findGoModInfo(subDir)
	if err != nil {
		t.Fatalf("findGoModInfo failed: %v", err)
	}

	if info.ModulePath != "example.com/myproject" {
		t.Errorf("ModulePath = %q, want %q", info.ModulePath, "example.com/myproject")
	}

	if info.GoModDir != tmpDir {
		t.Errorf("GoModDir = %q, want %q", info.GoModDir, tmpDir)
	}
}

func TestResolveReplacePath(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		goModDir   string
		wantResult string
	}{
		{
			name:       "relative path",
			line:       "replace github.com/foo/bar => ../bar",
			goModDir:   "/home/user/project",
			wantResult: "replace github.com/foo/bar => /home/user/bar",
		},
		{
			name:       "absolute path unchanged",
			line:       "replace github.com/foo/bar => /absolute/path",
			goModDir:   "/home/user/project",
			wantResult: "replace github.com/foo/bar => /absolute/path",
		},
		{
			name:       "version replace unchanged",
			line:       "replace github.com/foo/bar v1.0.0 => v1.1.0",
			goModDir:   "/home/user/project",
			wantResult: "replace github.com/foo/bar v1.0.0 => v1.1.0",
		},
		{
			name:       "dot relative path",
			line:       "replace github.com/foo/bar => ./vendor/bar",
			goModDir:   "/home/user/project",
			wantResult: "replace github.com/foo/bar => /home/user/project/vendor/bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveReplacePath(tt.line, tt.goModDir)
			if got != tt.wantResult {
				t.Errorf("resolveReplacePath() = %q, want %q", got, tt.wantResult)
			}
		})
	}
}

func TestFindGoBinary(t *testing.T) {
	// This should always find go since we're running tests with go
	goBin := findGoBinary()
	if goBin == "" {
		t.Error("findGoBinary should return a non-empty path")
	}

	// Verify the binary exists
	if _, err := os.Stat(goBin); err != nil && goBin != "go" {
		t.Errorf("go binary not found at %q", goBin)
	}
}

func TestCreateSyntheticGoModInfo(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		wantPath string
	}{
		{
			name:     "normal directory",
			dir:      "/path/to/myproject",
			wantPath: "myproject",
		},
		{
			name:     "empty path",
			dir:      "",
			wantPath: "template",
		},
		{
			name:     "dot path",
			dir:      ".",
			wantPath: "template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := createSyntheticGoModInfo(tt.dir)
			if err != nil {
				t.Fatalf("createSyntheticGoModInfo failed: %v", err)
			}

			if info.ModulePath != tt.wantPath {
				t.Errorf("ModulePath = %q, want %q", info.ModulePath, tt.wantPath)
			}

			if !info.Synthetic {
				t.Error("Synthetic should be true")
			}
		})
	}
}

func TestExtractValues_EmptyResources(t *testing.T) {
	result, err := ExtractValues("./testpkg", nil)
	if err != nil {
		t.Fatalf("ExtractValues failed: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result for empty resources, got %v", result)
	}
}

func TestExtractAll_EmptyInputs(t *testing.T) {
	result, err := ExtractAll("./testpkg", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("ExtractAll failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result.Resources) != 0 {
		t.Errorf("expected empty Resources, got %d", len(result.Resources))
	}

	if len(result.Parameters) != 0 {
		t.Errorf("expected empty Parameters, got %d", len(result.Parameters))
	}
}

func TestRunnerTemplateGeneration(t *testing.T) {
	// Test that the runner template generates valid Go code
	var buf bytes.Buffer

	data := struct {
		ImportPath string
		FirstVar   string
		VarNames   []string
	}{
		ImportPath: "example.com/test/infra",
		FirstVar:   "MyBucket",
		VarNames:   []string{"MyBucket", "MyFunction", "MyRole"},
	}

	err := runnerTemplate.Execute(&buf, data)
	if err != nil {
		t.Fatalf("template execution failed: %v", err)
	}

	output := buf.String()

	// Verify key components are present
	if !contains(output, "package main") {
		t.Error("generated code should have package main")
	}

	if !contains(output, `pkg "example.com/test/infra"`) {
		t.Error("generated code should import the target package")
	}

	if !contains(output, "case \"MyBucket\":") {
		t.Error("generated code should have case for MyBucket")
	}

	if !contains(output, "case \"MyFunction\":") {
		t.Error("generated code should have case for MyFunction")
	}

	if !contains(output, "case \"MyRole\":") {
		t.Error("generated code should have case for MyRole")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Integration tests that run the actual extraction process

func TestExtractValues_WithRealPackage(t *testing.T) {
	// Use the testdata/simple package
	pkgPath := "./testdata/simple"

	// Create mock resources matching the package
	resources := map[string]struct {
		Name    string
		Type    string
		Package string
	}{
		"TestBucket": {Name: "TestBucket", Type: "s3.Bucket", Package: "s3"},
	}

	// Convert to the expected type
	discoveredResources := make(map[string]struct {
		Name         string
		Type         string
		Package      string
		Dependencies []string
	})
	for k, v := range resources {
		discoveredResources[k] = struct {
			Name         string
			Type         string
			Package      string
			Dependencies []string
		}{Name: v.Name, Type: v.Type, Package: v.Package}
	}

	// Use a wrapper to call ExtractValues with proper type
	result, err := extractTestValues(pkgPath, []string{"TestBucket"})
	if err != nil {
		t.Fatalf("ExtractValues failed: %v", err)
	}

	// Verify the bucket was extracted
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	bucketProps, ok := result["TestBucket"]
	if !ok {
		t.Fatal("TestBucket not found in result")
	}

	if bucketProps["BucketName"] != "my-test-bucket" {
		t.Errorf("BucketName = %v, want %q", bucketProps["BucketName"], "my-test-bucket")
	}
}

// extractTestValues is a helper that bypasses the DiscoveredResource type requirement
func extractTestValues(pkgPath string, varNames []string) (map[string]map[string]any, error) {
	// This calls the unexported extractVarValues directly
	return extractVarValues(pkgPath, varNames)
}

func TestExtractAll_WithRealPackage(t *testing.T) {
	pkgPath := "./testdata/simple"

	// Create minimal resource map
	resources := map[string]struct {
		Name         string
		Type         string
		Package      string
		Dependencies []string
	}{
		"TestBucket": {Name: "TestBucket", Type: "s3.Bucket", Package: "s3"},
	}

	// Convert to wetwire.DiscoveredResource
	discoveredResources := make(map[string]wetwire.DiscoveredResource)
	for k, v := range resources {
		discoveredResources[k] = wetwire.DiscoveredResource{
			Name:         v.Name,
			Type:         v.Type,
			Package:      v.Package,
			Dependencies: v.Dependencies,
		}
	}

	result, err := ExtractAll(pkgPath, discoveredResources, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("ExtractAll failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(result.Resources))
	}

	bucketProps, ok := result.Resources["TestBucket"]
	if !ok {
		t.Fatal("TestBucket not found in Resources")
	}

	if bucketProps["BucketName"] != "my-test-bucket" {
		t.Errorf("BucketName = %v, want %q", bucketProps["BucketName"], "my-test-bucket")
	}

	// Verify other maps are initialized but empty
	if result.Parameters == nil {
		t.Error("Parameters should be initialized")
	}
	if result.Outputs == nil {
		t.Error("Outputs should be initialized")
	}
	if result.Mappings == nil {
		t.Error("Mappings should be initialized")
	}
	if result.Conditions == nil {
		t.Error("Conditions should be initialized")
	}
}

func TestFindGoModInfo_NoModuleDirective(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod without module directive
	goMod := `go 1.23.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := findGoModInfo(tmpDir)
	if err == nil {
		t.Error("Expected error for go.mod without module directive")
	}
	if err != nil && !contains(err.Error(), "no module directive") {
		t.Errorf("Expected 'no module directive' error, got: %v", err)
	}
}

func TestExtractVarValues_EmptyVarNames(t *testing.T) {
	result, err := extractVarValues("./testdata/simple", nil)
	if err != nil {
		t.Fatalf("Expected no error for empty varNames, got: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result for empty varNames, got: %v", result)
	}
}

func TestExtractVarValues_EmptySlice(t *testing.T) {
	result, err := extractVarValues("./testdata/simple", []string{})
	if err != nil {
		t.Fatalf("Expected no error for empty slice, got: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result for empty slice, got: %v", result)
	}
}

func TestExtractValues_WithDiscoveredResource(t *testing.T) {
	pkgPath := "./testdata/simple"

	resources := map[string]wetwire.DiscoveredResource{
		"TestBucket": {
			Name:    "TestBucket",
			Type:    "s3.Bucket",
			Package: "s3",
		},
	}

	result, err := ExtractValues(pkgPath, resources)
	if err != nil {
		t.Fatalf("ExtractValues failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	bucketProps, ok := result["TestBucket"]
	if !ok {
		t.Fatal("TestBucket not found in result")
	}

	if bucketProps["BucketName"] != "my-test-bucket" {
		t.Errorf("BucketName = %v, want %q", bucketProps["BucketName"], "my-test-bucket")
	}
}

func TestCreateRunnerSubdir_ExistingDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create vendor directory to trigger subdir mode
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.Mkdir(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module example.com/test

go 1.23
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create existing _wetwire_runner directory
	existingRunner := filepath.Join(tmpDir, "_wetwire_runner")
	if err := os.Mkdir(existingRunner, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in the existing directory
	if err := os.WriteFile(filepath.Join(existingRunner, "stale.go"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	// createRunnerSubdir should remove existing and create fresh
	runnerDir, cleanup, err := createRunnerSubdir(tmpDir)
	if err != nil {
		t.Fatalf("createRunnerSubdir failed: %v", err)
	}
	defer cleanup()

	// Verify stale file was removed
	if _, err := os.Stat(filepath.Join(runnerDir, "stale.go")); !os.IsNotExist(err) {
		t.Error("stale.go should have been removed")
	}

	// Verify directory exists
	if _, err := os.Stat(runnerDir); err != nil {
		t.Errorf("runner directory should exist: %v", err)
	}
}

func TestResolveReplacePath_NoArrow(t *testing.T) {
	// Test line without " => " separator
	line := "replace github.com/foo/bar"
	result := resolveReplacePath(line, "/home/user/project")

	// Should return unchanged
	if result != line {
		t.Errorf("resolveReplacePath() = %q, want %q", result, line)
	}
}

func TestFindGoModInfo_CommentedReplace(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod with commented replace directive
	goMod := `module example.com/myproject

go 1.23.0

// replace github.com/foo/bar => ../bar

replace (
	// This is a comment
	github.com/baz/qux => /path/to/qux
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := findGoModInfo(tmpDir)
	if err != nil {
		t.Fatalf("findGoModInfo failed: %v", err)
	}

	// Should only have 1 replace (the uncommented one in the block)
	if len(info.Replaces) != 1 {
		t.Errorf("got %d replace directives, want 1", len(info.Replaces))
	}
}

func TestExtractAll_WithParametersAndOutputs(t *testing.T) {
	pkgPath := "./testdata/complex"

	resources := map[string]wetwire.DiscoveredResource{
		"DataBucket": {Name: "DataBucket", Type: "s3.Bucket", Package: "s3"},
	}

	parameters := map[string]wetwire.DiscoveredParameter{
		"Environment": {Name: "Environment"},
	}

	outputs := map[string]wetwire.DiscoveredOutput{
		"BucketNameOutput": {Name: "BucketNameOutput"},
	}

	mappings := map[string]wetwire.DiscoveredMapping{
		"RegionMapping": {Name: "RegionMapping"},
	}

	result, err := ExtractAll(pkgPath, resources, parameters, outputs, mappings, nil)
	if err != nil {
		t.Fatalf("ExtractAll failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check resources
	if len(result.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(result.Resources))
	}

	// Check parameters
	if len(result.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(result.Parameters))
	}
	if paramProps, ok := result.Parameters["Environment"]; ok {
		if paramProps["Type"] != "String" {
			t.Errorf("Parameter Type = %v, want String", paramProps["Type"])
		}
		if paramProps["Default"] != "dev" {
			t.Errorf("Parameter Default = %v, want dev", paramProps["Default"])
		}
	} else {
		t.Error("Environment parameter not found")
	}

	// Check outputs
	if len(result.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(result.Outputs))
	}
	if outputProps, ok := result.Outputs["BucketNameOutput"]; ok {
		if outputProps["Description"] != "Name of the data bucket" {
			t.Errorf("Output Description = %v, want 'Name of the data bucket'", outputProps["Description"])
		}
	} else {
		t.Error("BucketNameOutput not found")
	}

	// Check mappings
	if len(result.Mappings) != 1 {
		t.Errorf("Expected 1 mapping, got %d", len(result.Mappings))
	}
	if _, ok := result.Mappings["RegionMapping"]; !ok {
		t.Error("RegionMapping not found")
	}
}
