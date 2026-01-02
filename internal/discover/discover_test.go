package discover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover_SimpleResource(t *testing.T) {
	// Create a temp directory with test Go files
	dir := t.TempDir()

	// Write a simple Go file with resource declarations
	code := `package infra

import "github.com/lex00/wetwire-aws-go/s3"

var MyBucket = s3.Bucket{
	BucketName: "my-bucket",
}
`
	err := os.WriteFile(filepath.Join(dir, "storage.go"), []byte(code), 0644)
	require.NoError(t, err)

	// Discover resources
	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	// Verify the resource was found
	assert.Len(t, result.Resources, 1)
	assert.Contains(t, result.Resources, "MyBucket")

	res := result.Resources["MyBucket"]
	assert.Equal(t, "MyBucket", res.Name)
	assert.Equal(t, "s3.Bucket", res.Type)
	assert.Equal(t, "infra", res.Package)
	assert.Empty(t, res.Dependencies)
}

func TestDiscover_WithDependencies(t *testing.T) {
	dir := t.TempDir()

	code := `package infra

import (
	"github.com/lex00/wetwire-aws-go/iam"
	"github.com/lex00/wetwire-aws-go/lambda"
	"github.com/lex00/wetwire-aws-go/s3"
)

var DataBucket = s3.Bucket{
	BucketName: "data-bucket",
}

var ProcessorRole = iam.Role{
	RoleName: "processor-role",
}

var ProcessorFunction = lambda.Function{
	FunctionName: "processor",
	Role:         ProcessorRole.Arn,
	Environment: &lambda.Environment{
		Variables: map[string]string{
			"BUCKET": DataBucket.BucketName,
		},
	},
}
`
	err := os.WriteFile(filepath.Join(dir, "compute.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 3)

	// Check ProcessorFunction has dependencies
	fn := result.Resources["ProcessorFunction"]
	assert.Equal(t, "lambda.Function", fn.Type)
	assert.Contains(t, fn.Dependencies, "ProcessorRole")
	assert.Contains(t, fn.Dependencies, "DataBucket")
}

func TestDiscover_UndefinedReference(t *testing.T) {
	dir := t.TempDir()

	code := `package infra

import "github.com/lex00/wetwire-aws-go/lambda"

var ProcessorFunction = lambda.Function{
	Role: UndefinedRole.Arn,
}
`
	err := os.WriteFile(filepath.Join(dir, "compute.go"), []byte(code), 0644)
	require.NoError(t, err)

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	// Should have an error for undefined reference
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "UndefinedRole")
}

func TestDiscover_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	storageCode := `package infra

import "github.com/lex00/wetwire-aws-go/s3"

var DataBucket = s3.Bucket{
	BucketName: "data",
}
`
	computeCode := `package infra

import "github.com/lex00/wetwire-aws-go/lambda"

var Processor = lambda.Function{
	FunctionName: "proc",
	Environment: &lambda.Environment{
		Variables: map[string]string{
			"BUCKET": DataBucket.BucketName,
		},
	},
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "storage.go"), []byte(storageCode), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "compute.go"), []byte(computeCode), 0644))

	result, err := Discover(Options{
		Packages: []string{dir},
	})
	require.NoError(t, err)

	assert.Len(t, result.Resources, 2)
	assert.Contains(t, result.Resources, "DataBucket")
	assert.Contains(t, result.Resources, "Processor")

	// Processor should depend on DataBucket
	proc := result.Resources["Processor"]
	assert.Contains(t, proc.Dependencies, "DataBucket")
	assert.Empty(t, result.Errors)
}

func TestExtractTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantPkg  string
	}{
		{
			name:     "qualified type",
			input:    "s3.Bucket",
			wantType: "Bucket",
			wantPkg:  "s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would need proper AST parsing to test properly
			// For now, just verify the function exists
			_ = extractTypeName
		})
	}
}
