package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wetwire "github.com/lex00/wetwire-aws-go"
)

func TestBuilder_Build_SimpleResource(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {
			Name:    "MyBucket",
			Type:    "s3.Bucket",
			Package: "infra",
			File:    "storage.go",
			Line:    5,
		},
	}

	builder := NewBuilder(resources)

	// Simulate the resource value
	builder.SetValue("MyBucket", map[string]any{
		"BucketName": "my-bucket",
	})

	template, err := builder.Build()
	require.NoError(t, err)

	assert.Equal(t, "2010-09-09", template.AWSTemplateFormatVersion)
	assert.Len(t, template.Resources, 1)

	bucket := template.Resources["MyBucket"]
	assert.Equal(t, "AWS::S3::Bucket", bucket.Type)
	assert.Equal(t, "my-bucket", bucket.Properties["BucketName"])
}

func TestBuilder_Build_WithDependencies(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"DataBucket": {
			Name:    "DataBucket",
			Type:    "s3.Bucket",
			Package: "infra",
			File:    "storage.go",
			Line:    5,
		},
		"ProcessorRole": {
			Name:    "ProcessorRole",
			Type:    "iam.Role",
			Package: "infra",
			File:    "iam.go",
			Line:    10,
		},
		"ProcessorFunction": {
			Name:         "ProcessorFunction",
			Type:         "lambda.Function",
			Package:      "infra",
			File:         "compute.go",
			Line:         15,
			Dependencies: []string{"ProcessorRole", "DataBucket"},
		},
	}

	builder := NewBuilder(resources)
	builder.SetValue("DataBucket", map[string]any{
		"BucketName": "data-bucket",
	})
	builder.SetValue("ProcessorRole", map[string]any{
		"RoleName": "processor-role",
	})
	builder.SetValue("ProcessorFunction", map[string]any{
		"FunctionName": "processor",
		"Role": map[string][]string{
			"Fn::GetAtt": {"ProcessorRole", "Arn"},
		},
	})

	template, err := builder.Build()
	require.NoError(t, err)

	assert.Len(t, template.Resources, 3)

	// Verify GetAtt is preserved
	fn := template.Resources["ProcessorFunction"]
	role := fn.Properties["Role"].(map[string]any)
	assert.Contains(t, role, "Fn::GetAtt")
}

func TestBuilder_TopologicalSort(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"C": {Name: "C", Type: "s3.Bucket", Dependencies: []string{"B"}},
		"B": {Name: "B", Type: "s3.Bucket", Dependencies: []string{"A"}},
		"A": {Name: "A", Type: "s3.Bucket"},
	}

	builder := NewBuilder(resources)
	builder.SetValue("A", map[string]any{})
	builder.SetValue("B", map[string]any{})
	builder.SetValue("C", map[string]any{})

	order, err := builder.topologicalSort()
	require.NoError(t, err)

	// A should come before B, B before C
	aIdx := indexOf(order, "A")
	bIdx := indexOf(order, "B")
	cIdx := indexOf(order, "C")

	assert.Less(t, aIdx, bIdx)
	assert.Less(t, bIdx, cIdx)
}

func TestBuilder_DetectCycle(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"A": {Name: "A", Type: "s3.Bucket", File: "a.go", Line: 1, Dependencies: []string{"B"}},
		"B": {Name: "B", Type: "s3.Bucket", File: "b.go", Line: 2, Dependencies: []string{"C"}},
		"C": {Name: "C", Type: "s3.Bucket", File: "c.go", Line: 3, Dependencies: []string{"A"}},
	}

	builder := NewBuilder(resources)

	_, err := builder.topologicalSort()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestToJSON(t *testing.T) {
	template := &wetwire.Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Resources: map[string]wetwire.ResourceDef{
			"MyBucket": {
				Type: "AWS::S3::Bucket",
				Properties: map[string]any{
					"BucketName": "test-bucket",
				},
			},
		},
	}

	data, err := ToJSON(template)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "2010-09-09", parsed["AWSTemplateFormatVersion"])
	resources := parsed["Resources"].(map[string]any)
	bucket := resources["MyBucket"].(map[string]any)
	assert.Equal(t, "AWS::S3::Bucket", bucket["Type"])
}

func TestToYAML(t *testing.T) {
	template := &wetwire.Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Resources: map[string]wetwire.ResourceDef{
			"MyBucket": {
				Type: "AWS::S3::Bucket",
				Properties: map[string]any{
					"BucketName": "test-bucket",
				},
			},
		},
	}

	data, err := ToYAML(template)
	require.NoError(t, err)

	// Should be valid YAML
	assert.Contains(t, string(data), "AWSTemplateFormatVersion")
	assert.Contains(t, string(data), "AWS::S3::Bucket")
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func TestBuilder_Build_SAMFunction(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"HelloWorldFunction": {
			Name:    "HelloWorldFunction",
			Type:    "serverless.Function",
			Package: "infra",
			File:    "functions.go",
			Line:    5,
		},
	}

	builder := NewBuilder(resources)
	builder.SetValue("HelloWorldFunction", map[string]any{
		"Handler":     "bootstrap",
		"Runtime":     "provided.al2",
		"CodeUri":     "./hello-world/",
		"MemorySize":  128,
		"Timeout":     5,
	})

	template, err := builder.Build()
	require.NoError(t, err)

	// SAM templates must have Transform header
	assert.Equal(t, "AWS::Serverless-2016-10-31", template.Transform)
	assert.Equal(t, "2010-09-09", template.AWSTemplateFormatVersion)

	// Verify resource type
	fn := template.Resources["HelloWorldFunction"]
	assert.Equal(t, "AWS::Serverless::Function", fn.Type)
	assert.Equal(t, "bootstrap", fn.Properties["Handler"])
}

func TestBuilder_Build_SAMApi(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyApi": {
			Name:    "MyApi",
			Type:    "serverless.Api",
			Package: "infra",
			File:    "api.go",
			Line:    5,
		},
	}

	builder := NewBuilder(resources)
	builder.SetValue("MyApi", map[string]any{
		"StageName": "prod",
	})

	template, err := builder.Build()
	require.NoError(t, err)

	assert.Equal(t, "AWS::Serverless-2016-10-31", template.Transform)
	assert.Equal(t, "AWS::Serverless::Api", template.Resources["MyApi"].Type)
}

func TestBuilder_Build_MixedSAMAndCFN(t *testing.T) {
	// Template with both SAM and CloudFormation resources
	resources := map[string]wetwire.DiscoveredResource{
		"DataBucket": {
			Name:    "DataBucket",
			Type:    "s3.Bucket",
			Package: "infra",
			File:    "storage.go",
			Line:    5,
		},
		"ProcessorFunction": {
			Name:         "ProcessorFunction",
			Type:         "serverless.Function",
			Package:      "infra",
			File:         "functions.go",
			Line:         10,
			Dependencies: []string{"DataBucket"},
		},
	}

	builder := NewBuilder(resources)
	builder.SetValue("DataBucket", map[string]any{
		"BucketName": "data-bucket",
	})
	builder.SetValue("ProcessorFunction", map[string]any{
		"Handler": "bootstrap",
		"Runtime": "provided.al2",
	})

	template, err := builder.Build()
	require.NoError(t, err)

	// Transform should be set because SAM resources are present
	assert.Equal(t, "AWS::Serverless-2016-10-31", template.Transform)

	// Both resources should be present with correct types
	assert.Equal(t, "AWS::S3::Bucket", template.Resources["DataBucket"].Type)
	assert.Equal(t, "AWS::Serverless::Function", template.Resources["ProcessorFunction"].Type)
}

func TestBuilder_Build_NoSAM_NoTransform(t *testing.T) {
	// Template with only CloudFormation resources should NOT have Transform
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {
			Name:    "MyBucket",
			Type:    "s3.Bucket",
			Package: "infra",
			File:    "storage.go",
			Line:    5,
		},
	}

	builder := NewBuilder(resources)
	builder.SetValue("MyBucket", map[string]any{
		"BucketName": "my-bucket",
	})

	template, err := builder.Build()
	require.NoError(t, err)

	// No Transform for non-SAM templates
	assert.Empty(t, template.Transform)
}

func TestCfResourceType_SAM(t *testing.T) {
	tests := []struct {
		goType   string
		cfnType  string
	}{
		{"serverless.Function", "AWS::Serverless::Function"},
		{"serverless.Api", "AWS::Serverless::Api"},
		{"serverless.HttpApi", "AWS::Serverless::HttpApi"},
		{"serverless.SimpleTable", "AWS::Serverless::SimpleTable"},
		{"serverless.LayerVersion", "AWS::Serverless::LayerVersion"},
		{"serverless.StateMachine", "AWS::Serverless::StateMachine"},
		{"serverless.Application", "AWS::Serverless::Application"},
		{"serverless.Connector", "AWS::Serverless::Connector"},
		{"serverless.GraphQLApi", "AWS::Serverless::GraphQLApi"},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			result := cfResourceType(tt.goType)
			assert.Equal(t, tt.cfnType, result)
		})
	}
}
