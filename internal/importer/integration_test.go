package importer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestExamplesBuild verifies that a curated set of complex imported examples
// compile successfully. This catches regressions in the importer's code generation.
//
// If this test fails, it means the importer is generating invalid Go code.
// Check the specific failing example to diagnose the issue.
func TestExamplesBuild(t *testing.T) {
	// Skip if not in the repo (e.g., when package is used as a dependency)
	examplesDir := findExamplesDir()
	if examplesDir == "" {
		t.Skip("examples directory not found, skipping integration test")
	}

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
	//
	// NOTE: neptune excluded - has many undefined parameter references that
	// would require extensive template fixes.
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

			// Try to build the example
			cmd := exec.Command("go", "build", "./...")
			cmd.Dir = examplePath

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("example %s failed to build:\n%s", example, string(output))
			}
		})
	}
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
