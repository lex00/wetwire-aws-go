package serialize

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestBucket struct {
	BucketName  string            `json:"BucketName,omitempty"`
	Tags        []Tag             `json:"Tags,omitempty"`
	Versioning  *TestVersioning   `json:"VersioningConfiguration,omitempty"`
	Environment map[string]string `json:"Environment,omitempty"`
}

type Tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type TestVersioning struct {
	Status string `json:"Status"`
}

func TestResource_SimpleStruct(t *testing.T) {
	bucket := TestBucket{
		BucketName: "my-bucket",
	}

	props, err := Resource(bucket)
	require.NoError(t, err)

	assert.Equal(t, "my-bucket", props["BucketName"])
	assert.NotContains(t, props, "Tags")       // Empty slice should be omitted
	assert.NotContains(t, props, "Versioning") // Nil pointer should be omitted
}

func TestResource_WithNestedStruct(t *testing.T) {
	bucket := TestBucket{
		BucketName: "my-bucket",
		Versioning: &TestVersioning{
			Status: "Enabled",
		},
	}

	props, err := Resource(bucket)
	require.NoError(t, err)

	assert.Equal(t, "my-bucket", props["BucketName"])

	versioning := props["VersioningConfiguration"].(map[string]any)
	assert.Equal(t, "Enabled", versioning["Status"])
}

func TestResource_WithSlice(t *testing.T) {
	bucket := TestBucket{
		BucketName: "my-bucket",
		Tags: []Tag{
			{Key: "Environment", Value: "prod"},
			{Key: "Team", Value: "platform"},
		},
	}

	props, err := Resource(bucket)
	require.NoError(t, err)

	tags := props["Tags"].([]any)
	assert.Len(t, tags, 2)

	tag0 := tags[0].(map[string]any)
	assert.Equal(t, "Environment", tag0["Key"])
	assert.Equal(t, "prod", tag0["Value"])
}

func TestResource_WithMap(t *testing.T) {
	bucket := TestBucket{
		BucketName: "my-bucket",
		Environment: map[string]string{
			"BUCKET_NAME": "my-bucket",
			"REGION":      "us-east-1",
		},
	}

	props, err := Resource(bucket)
	require.NoError(t, err)

	env := props["Environment"].(map[string]any)
	assert.Equal(t, "my-bucket", env["BUCKET_NAME"])
	assert.Equal(t, "us-east-1", env["REGION"])
}

func TestResource_OmitsZeroValues(t *testing.T) {
	bucket := TestBucket{
		BucketName: "", // Empty string
		Tags:       nil,
		Versioning: nil,
	}

	props, err := Resource(bucket)
	require.NoError(t, err)

	// All zero values should be omitted
	assert.Empty(t, props)
}

func TestResource_WithPointer(t *testing.T) {
	bucket := &TestBucket{
		BucketName: "my-bucket",
	}

	props, err := Resource(bucket)
	require.NoError(t, err)

	assert.Equal(t, "my-bucket", props["BucketName"])
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bucket_name", "BucketName"},
		{"vpc_id", "VpcId"},
		{"already_pascal", "AlreadyPascal"},
		{"simple", "Simple"},
		{"multiple_word_name", "MultipleWordName"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"BucketName", "bucket_name"},
		{"VpcId", "vpc_id"},
		{"Simple", "simple"},
		{"MultipleWordName", "multiple_word_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
