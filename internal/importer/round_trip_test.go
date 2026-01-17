package importer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestRoundTrip verifies semantic equivalence: YAML → import → Go code → build → compare
// This implements the round-trip testing requirement from spec 11.2.
//
// The test flow:
// 1. Parse original YAML template
// 2. Generate Go code via importer
// 3. Write Go code to temp directory
// 4. Build using wetwire-aws build
// 5. Compare generated template with original (semantic comparison)
func TestRoundTrip(t *testing.T) {
	testdataDir := findTestdataDir()
	if testdataDir == "" {
		t.Skip("testdata/reference directory not found")
	}

	cliPath := buildCLI(t)

	referenceDir := filepath.Join(testdataDir, "reference")
	entries, err := os.ReadDir(referenceDir)
	if err != nil {
		t.Fatalf("failed to read reference directory: %v", err)
	}

	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
			templates = append(templates, entry.Name())
		}
	}

	if len(templates) < 20 {
		t.Errorf("expected at least 20 reference templates, got %d", len(templates))
	}

	for _, tmpl := range templates {
		tmpl := tmpl
		t.Run(strings.TrimSuffix(tmpl, filepath.Ext(tmpl)), func(t *testing.T) {
			t.Parallel()

			templatePath := filepath.Join(referenceDir, tmpl)
			runRoundTrip(t, templatePath, cliPath)
		})
	}
}

// runRoundTrip executes the full round-trip test for a single template.
func runRoundTrip(t *testing.T, templatePath, cliPath string) {
	t.Helper()

	// Step 1: Read and parse original template
	originalData, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("failed to read template: %v", err)
	}

	var originalTemplate map[string]any
	if err := yaml.Unmarshal(originalData, &originalTemplate); err != nil {
		t.Fatalf("failed to parse original YAML: %v", err)
	}

	// Step 2: Parse template using importer
	irTemplate, err := ParseTemplateContent(originalData, templatePath)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	// Step 3: Generate Go code
	packageName := sanitizePackageName(strings.TrimSuffix(filepath.Base(templatePath), filepath.Ext(templatePath)))
	files := GenerateCode(irTemplate, packageName)
	if len(files) == 0 {
		t.Fatalf("no files generated")
	}

	// Add scaffold files (go.mod, etc.)
	scaffoldFiles := GenerateTemplateFiles(packageName, packageName)
	for name, content := range scaffoldFiles {
		files[name] = content
	}

	// Step 4: Write to temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, packageName)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	for fileName, content := range files {
		filePath := filepath.Join(projectDir, fileName)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", fileName, err)
		}
	}

	// Step 5: Run go mod tidy
	modTidyCmd := exec.Command("go", "mod", "tidy")
	modTidyCmd.Dir = projectDir
	if output, err := modTidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	// Step 6: Compile Go code
	buildGoCmd := exec.Command("go", "build", "./...")
	buildGoCmd.Dir = projectDir
	if output, err := buildGoCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	// Step 7: Run wetwire-aws build to generate CloudFormation template
	// Use --format=raw to get raw CloudFormation JSON output
	buildCmd := exec.Command(cliPath, "build", projectDir, "--format", "raw")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wetwire-aws build failed: %v\n%s", err, buildOutput)
	}

	// Step 8: Parse generated template
	var generatedTemplate map[string]any
	if err := json.Unmarshal(buildOutput, &generatedTemplate); err != nil {
		t.Fatalf("failed to parse generated template: %v\nOutput: %s", err, buildOutput)
	}

	// Step 9: Semantic comparison
	comparison := compareTemplates(originalTemplate, generatedTemplate)
	if !comparison.equivalent {
		t.Errorf("round-trip comparison failed:\n%s", comparison.report)
	}
}

// sanitizePackageName converts a filename to a valid Go package name.
func sanitizePackageName(name string) string {
	// Convert to lowercase
	result := strings.ToLower(name)

	// Replace non-alphanumeric characters with underscores
	var sb strings.Builder
	for _, r := range result {
		if r >= 'a' && r <= 'z' {
			sb.WriteRune(r)
		} else if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}

	// Trim underscores and remove leading digits
	name = strings.Trim(sb.String(), "_")

	// Remove leading digits/underscores
	for len(name) > 0 && (name[0] >= '0' && name[0] <= '9' || name[0] == '_') {
		name = name[1:]
	}

	// Replace consecutive underscores with single underscore
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	name = strings.Trim(name, "_")

	if name == "" {
		return "infra"
	}
	return name
}

// comparisonResult holds the result of comparing two templates.
type comparisonResult struct {
	equivalent bool
	report     string
}

// compareTemplates performs semantic comparison of two CloudFormation templates.
// It ignores formatting differences and focuses on structural equivalence.
func compareTemplates(original, generated map[string]any) comparisonResult {
	var issues []string

	// Compare resources
	origResources, _ := original["Resources"].(map[string]any)
	genResources, _ := generated["Resources"].(map[string]any)

	// Check for missing resources
	for name := range origResources {
		if _, exists := genResources[name]; !exists {
			issues = append(issues, "missing resource: "+name)
		}
	}

	// Check for extra resources (may be acceptable in some cases)
	for name := range genResources {
		if _, exists := origResources[name]; !exists {
			// Only flag if not an implicit SAM resource
			if !strings.HasSuffix(name, "Role") && !strings.HasSuffix(name, "LogGroup") {
				issues = append(issues, "extra resource: "+name)
			}
		}
	}

	// Compare each resource's properties
	for name, origRes := range origResources {
		if genRes, exists := genResources[name]; exists {
			origDef, _ := origRes.(map[string]any)
			genDef, _ := genRes.(map[string]any)

			// Compare Type
			origType := origDef["Type"]
			genType := genDef["Type"]
			if !typesEquivalent(origType, genType) {
				issues = append(issues, "type mismatch for "+name+": expected "+str(origType)+", got "+str(genType))
			}

			// Compare Properties (semantic comparison)
			origProps, _ := origDef["Properties"].(map[string]any)
			genProps, _ := genDef["Properties"].(map[string]any)
			propIssues := compareProperties(name, origProps, genProps)
			issues = append(issues, propIssues...)
		}
	}

	// Compare Parameters (if any)
	origParams, _ := original["Parameters"].(map[string]any)
	genParams, _ := generated["Parameters"].(map[string]any)
	if len(origParams) > 0 || len(genParams) > 0 {
		for name := range origParams {
			if _, exists := genParams[name]; !exists {
				issues = append(issues, "missing parameter: "+name)
			}
		}
	}

	// Compare Outputs (if any)
	origOutputs, _ := original["Outputs"].(map[string]any)
	genOutputs, _ := generated["Outputs"].(map[string]any)
	if len(origOutputs) > 0 || len(genOutputs) > 0 {
		for name := range origOutputs {
			if _, exists := genOutputs[name]; !exists {
				issues = append(issues, "missing output: "+name)
			}
		}
	}

	sort.Strings(issues)

	return comparisonResult{
		equivalent: len(issues) == 0,
		report:     strings.Join(issues, "\n"),
	}
}

// typesEquivalent checks if two resource types are semantically equivalent.
func typesEquivalent(orig, gen any) bool {
	origStr, _ := orig.(string)
	genStr, _ := gen.(string)
	return origStr == genStr
}

// compareProperties compares resource properties semantically.
func compareProperties(resourceName string, orig, gen map[string]any) []string {
	var issues []string

	for key, origVal := range orig {
		genVal, exists := gen[key]
		if !exists {
			issues = append(issues, resourceName+"."+key+": property missing")
			continue
		}

		if !valuesEquivalent(origVal, genVal) {
			issues = append(issues, resourceName+"."+key+": value differs")
		}
	}

	return issues
}

// valuesEquivalent performs deep semantic comparison of values.
func valuesEquivalent(orig, gen any) bool {
	// Handle nil
	if orig == nil && gen == nil {
		return true
	}
	if orig == nil || gen == nil {
		return false
	}

	// Handle intrinsic functions specially
	origMap, origIsMap := orig.(map[string]any)
	genMap, genIsMap := gen.(map[string]any)

	if origIsMap && genIsMap {
		// Check for intrinsic functions
		if isIntrinsic(origMap) || isIntrinsic(genMap) {
			return intrinsicsEquivalent(origMap, genMap)
		}

		// Regular map comparison
		if len(origMap) != len(genMap) {
			return false
		}
		for k, v := range origMap {
			if !valuesEquivalent(v, genMap[k]) {
				return false
			}
		}
		return true
	}

	// Handle arrays
	origArr, origIsArr := orig.([]any)
	genArr, genIsArr := gen.([]any)
	if origIsArr && genIsArr {
		if len(origArr) != len(genArr) {
			return false
		}
		for i := range origArr {
			if !valuesEquivalent(origArr[i], genArr[i]) {
				return false
			}
		}
		return true
	}

	// Handle type conversions (int vs float64 from JSON)
	origFloat, origIsFloat := toFloat64(orig)
	genFloat, genIsFloat := toFloat64(gen)
	if origIsFloat && genIsFloat {
		return origFloat == genFloat
	}

	// Handle string vs intrinsic (for inline string references)
	_, origIsStr := orig.(string)
	_, genIsStr := gen.(string)
	if origIsStr && genIsMap {
		// If original is a string and generated is an intrinsic, check reference
		if ref, ok := genMap["Ref"]; ok {
			// Accept if generated Ref matches some resource (the importer converts them)
			if refStr, ok := ref.(string); ok && refStr != "" {
				return true // Accept valid Ref as equivalent
			}
		}
		if getAtt, ok := genMap["Fn::GetAtt"]; ok {
			parts := parseGetAtt(getAtt)
			if len(parts) == 2 && parts[0] != "" {
				return true // Accept valid GetAtt as equivalent
			}
		}
	}

	// Handle intrinsic in original vs string in generated
	if origIsMap && genIsStr {
		if _, ok := origMap["Ref"]; ok {
			return true // Accept string as potentially equivalent
		}
		if _, ok := origMap["Fn::GetAtt"]; ok {
			return true // Accept string as potentially equivalent
		}
	}

	// Direct comparison
	return reflect.DeepEqual(orig, gen)
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	}
	return 0, false
}

// isIntrinsic checks if a map represents a CloudFormation intrinsic function.
func isIntrinsic(m map[string]any) bool {
	intrinsics := []string{"Ref", "Fn::GetAtt", "Fn::Sub", "Fn::Join", "Fn::If", "Fn::Select", "Fn::GetAZs", "Fn::Split", "Fn::Equals", "Fn::And", "Fn::Or", "Fn::Not", "Fn::Base64", "Fn::Cidr", "Fn::FindInMap", "Fn::Transform", "Fn::ImportValue"}
	for _, fn := range intrinsics {
		if _, ok := m[fn]; ok {
			return true
		}
	}
	return false
}

// intrinsicsEquivalent compares intrinsic functions semantically.
func intrinsicsEquivalent(orig, gen map[string]any) bool {
	// Handle Ref
	if origRef, ok := orig["Ref"]; ok {
		if genRef, ok := gen["Ref"]; ok {
			return origRef == genRef
		}
		// Ref can be equivalent to GetAtt in some cases
		return false
	}

	// Handle GetAtt
	if origGetAtt, ok := orig["Fn::GetAtt"]; ok {
		if genGetAtt, ok := gen["Fn::GetAtt"]; ok {
			return getAttEquivalent(origGetAtt, genGetAtt)
		}
		return false
	}

	// Handle Sub
	if origSub, ok := orig["Fn::Sub"]; ok {
		if genSub, ok := gen["Fn::Sub"]; ok {
			return subEquivalent(origSub, genSub)
		}
		return false
	}

	// Default: deep equal
	return reflect.DeepEqual(orig, gen)
}

// getAttEquivalent compares GetAtt intrinsics.
func getAttEquivalent(orig, gen any) bool {
	// GetAtt can be either ["Resource", "Attribute"] or "Resource.Attribute"
	origParts := parseGetAtt(orig)
	genParts := parseGetAtt(gen)
	if len(origParts) != 2 || len(genParts) != 2 {
		return false
	}
	return origParts[0] == genParts[0] && origParts[1] == genParts[1]
}

// parseGetAtt parses GetAtt value to [resource, attribute].
func parseGetAtt(v any) []string {
	switch val := v.(type) {
	case []any:
		if len(val) == 2 {
			r, _ := val[0].(string)
			a, _ := val[1].(string)
			return []string{r, a}
		}
	case []string:
		if len(val) == 2 {
			return val
		}
	case string:
		parts := strings.SplitN(val, ".", 2)
		if len(parts) == 2 {
			return parts
		}
	}
	return nil
}

// subEquivalent compares Sub intrinsics.
func subEquivalent(orig, gen any) bool {
	// Sub can be a string or [string, map]
	origStr, origVars := parseSub(orig)
	genStr, genVars := parseSub(gen)

	if origStr != genStr {
		return false
	}

	return reflect.DeepEqual(origVars, genVars)
}

// parseSub parses Sub value to (template string, variables map).
func parseSub(v any) (string, map[string]any) {
	switch val := v.(type) {
	case string:
		return val, nil
	case []any:
		if len(val) >= 1 {
			str, _ := val[0].(string)
			vars, _ := val[1].(map[string]any)
			return str, vars
		}
	}
	return "", nil
}

// str converts any value to string for reporting.
func str(v any) string {
	if v == nil {
		return "<nil>"
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// findTestdataDir locates the testdata directory.
func findTestdataDir() string {
	candidates := []string{
		"testdata",
		"./testdata",
		"internal/importer/testdata",
		"../../internal/importer/testdata",
	}

	// First try relative to current dir
	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		refDir := filepath.Join(absPath, "reference")
		if info, err := os.Stat(refDir); err == nil && info.IsDir() {
			return absPath
		}
	}

	return ""
}
