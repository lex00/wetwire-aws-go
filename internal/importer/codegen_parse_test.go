package importer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplate_FromFile(t *testing.T) {
	// Test ParseTemplate function (reads from file)
	tmpDir := t.TempDir()
	yaml := `
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-bucket
`
	filePath := tmpDir + "/template.yaml"
	require.NoError(t, os.WriteFile(filePath, []byte(yaml), 0644))

	ir, err := ParseTemplate(filePath)
	require.NoError(t, err)
	assert.NotNil(t, ir)
	assert.Contains(t, ir.Resources, "MyBucket")
}

func TestParseTemplate_InvalidFile(t *testing.T) {
	// Test ParseTemplate with nonexistent file
	_, err := ParseTemplate("/nonexistent/file.yaml")
	assert.Error(t, err)
}
