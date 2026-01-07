package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestExamplesBuild verifies that a curated set of complex imported examples
// compile successfully and produce valid CloudFormation output.
//
// This test catches regressions in the importer's code generation by:
// 1. Building the Go code (go build)
// 2. Running wetwire-aws build to generate CF output
// 3. Validating the CF output for empty refs and resource counts
//
// If this test fails, it means the importer is generating invalid Go code
// or producing broken CloudFormation templates.
func TestExamplesBuild(t *testing.T) {
	// Skip if not in the repo (e.g., when package is used as a dependency)
	examplesDir := findExamplesDir()
	if examplesDir == "" {
		t.Skip("examples directory not found, skipping integration test")
	}

	// Build the wetwire-aws CLI for validation
	cliPath := buildCLI(t)

	// Curated list of complex templates that exercise various importer features:
	// - cloudfront: CloudFront distribution with S3, complex nested properties
	// - dynamodb_table: DynamoDB with secondary indexes
	// - ec2instancewithsecuritygroupsample: EC2 with security groups, mappings
	// - cognito: Cognito user pool with various configurations
	// - load_balancer: ELB with listeners, health checks
	// - rest_api: API Gateway with authorizers
	// - iotanalytics: IoT Analytics pipeline with channels, datastores
	// - cloudformation_codebuild_template: CodeBuild with complex build specs
	// - eip_with_association: EC2 with EIP association
	// - compliant_bucket: S3 bucket with encryption, versioning, policies
	// - lambdasample: Lambda with parameters in Sub strings
	// - neptune: Neptune DB with many tag parameters
	// - datetimenow: SAM template with serverless.Function
	complexExamples := []string{
		"cloudfront",
		"dynamodb_table",
		"ec2instancewithsecuritygroupsample",
		"cognito",
		"load_balancer",
		"rest_api",
		"iotanalytics",
		"cloudformation_codebuild_template",
		"eip_with_association",
		"compliant_bucket",
		"lambdasample",
		"neptune",
		"datetimenow",
	}

	for _, example := range complexExamples {
		example := example // capture range variable
		t.Run(example, func(t *testing.T) {
			t.Parallel()

			examplePath := filepath.Join(examplesDir, example)

			// Check if example exists
			if _, err := os.Stat(examplePath); os.IsNotExist(err) {
				t.Skipf("example %s not found, skipping", example)
			}

			// Check if go.mod exists (valid Go module)
			goModPath := filepath.Join(examplePath, "go.mod")
			if _, err := os.Stat(goModPath); os.IsNotExist(err) {
				t.Skipf("example %s has no go.mod, skipping", example)
			}

			// Step 1: Try to build the example with go build
			cmd := exec.Command("go", "build", "./...")
			cmd.Dir = examplePath

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("example %s failed to build:\n%s", example, string(output))
			}

			// Step 2: Run wetwire-aws build to generate CF output
			buildCmd := exec.Command(cliPath, "build", examplePath)
			buildOutput, err := buildCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("wetwire-aws build failed for %s: %v\n%s", example, err, string(buildOutput))
			}

			// Step 3: Parse and validate the output
			var result BuildResult
			if err := json.Unmarshal(buildOutput, &result); err != nil {
				t.Fatalf("failed to parse build output for %s: %v\n%s", example, err, string(buildOutput))
			}

			if !result.Success {
				t.Fatalf("build failed for %s: %v", example, result.Errors)
			}

			// Step 4: Check for empty refs in the template
			// We need to scan the raw JSON map, not the struct
			var rawResult map[string]any
			if err := json.Unmarshal(buildOutput, &rawResult); err != nil {
				t.Fatalf("failed to parse raw output for %s: %v", example, err)
			}
			template, ok := rawResult["template"].(map[string]any)
			if !ok {
				t.Fatalf("template not found in output for %s", example)
			}
			emptyRefs := findEmptyRefs(template, "template")
			if len(emptyRefs) > 0 {
				t.Errorf("found %d empty refs in %s:\n%s", len(emptyRefs), example, strings.Join(emptyRefs, "\n"))
			}

			// Step 5: Validate resource count
			if len(result.Resources) == 0 && len(result.Template.Resources) > 0 {
				t.Errorf("resources list empty but template has %d resources", len(result.Template.Resources))
			}
		})
	}
}

// BuildResult matches the JSON output from wetwire-aws build
type BuildResult struct {
	Success   bool       `json:"success"`
	Template  CFTemplate `json:"template"`
	Resources []string   `json:"resources"`
	Errors    []string   `json:"errors"`
}

// CFTemplate represents the CloudFormation template structure
type CFTemplate struct {
	AWSTemplateFormatVersion string                    `json:"AWSTemplateFormatVersion"`
	Parameters               map[string]any            `json:"Parameters"`
	Conditions               map[string]any            `json:"Conditions"`
	Resources                map[string]any            `json:"Resources"`
	Outputs                  map[string]any            `json:"Outputs"`
}

// findEmptyRefs recursively searches for empty ref patterns in the JSON structure.
// Returns a list of paths where empty refs were found.
func findEmptyRefs(v any, path string) []string {
	var results []string

	switch val := v.(type) {
	case map[string]any:
		// Check for {"Ref": ""} pattern
		if ref, ok := val["Ref"]; ok {
			if refStr, isStr := ref.(string); isStr && refStr == "" {
				results = append(results, fmt.Sprintf("%s: empty Ref", path))
			}
		}

		// Check for {"Fn::GetAtt": ["", ...]} pattern
		if getAtt, ok := val["Fn::GetAtt"]; ok {
			if arr, isArr := getAtt.([]any); isArr && len(arr) > 0 {
				if first, isStr := arr[0].(string); isStr && first == "" {
					results = append(results, fmt.Sprintf("%s: empty GetAtt resource", path))
				}
			}
		}

		// Recurse into all map values
		for key, value := range val {
			subPath := key
			if path != "" {
				subPath = path + "." + key
			}
			results = append(results, findEmptyRefs(value, subPath)...)
		}

	case []any:
		// Recurse into array elements
		for i, item := range val {
			subPath := fmt.Sprintf("%s[%d]", path, i)
			results = append(results, findEmptyRefs(item, subPath)...)
		}
	}

	return results
}

// buildCLI builds the wetwire-aws CLI and returns the path to the binary.
func buildCLI(t *testing.T) string {
	t.Helper()

	// Find the project root
	projectRoot := findProjectRoot()
	if projectRoot == "" {
		t.Fatal("could not find project root")
	}

	// Build CLI to a temp location
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "wetwire-aws")

	cmd := exec.Command("go", "build", "-o", cliPath, "./cmd/wetwire-aws")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build CLI: %v\n%s", err, string(output))
	}

	return cliPath
}

// findProjectRoot locates the project root (directory containing go.mod).
func findProjectRoot() string {
	candidates := []string{
		"../..",
		"../../..",
		".",
	}

	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		goModPath := filepath.Join(absPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Verify it's the right go.mod (contains wetwire-aws-go)
			content, err := os.ReadFile(goModPath)
			if err == nil && strings.Contains(string(content), "wetwire-aws-go") {
				return absPath
			}
		}
	}

	return ""
}

// findExamplesDir locates the examples directory relative to the test file.
func findExamplesDir() string {
	// Try relative paths from common test execution locations
	candidates := []string{
		"../../examples/aws-cloudformation-templates",
		"../../../examples/aws-cloudformation-templates",
		"examples/aws-cloudformation-templates",
	}

	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath
		}
	}

	return ""
}
