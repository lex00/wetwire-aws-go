package wetwire_aws

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttrRef_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		ref      AttrRef
		expected string
	}{
		{
			name:     "role arn",
			ref:      AttrRef{Resource: "MyRole", Attribute: "Arn"},
			expected: `{"Fn::GetAtt":["MyRole","Arn"]}`,
		},
		{
			name:     "bucket domain name",
			ref:      AttrRef{Resource: "DataBucket", Attribute: "DomainName"},
			expected: `{"Fn::GetAtt":["DataBucket","DomainName"]}`,
		},
		{
			name:     "function arn",
			ref:      AttrRef{Resource: "ProcessorFunction", Attribute: "Arn"},
			expected: `{"Fn::GetAtt":["ProcessorFunction","Arn"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.ref)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestAttrRef_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		ref      AttrRef
		expected bool
	}{
		{
			name:     "empty",
			ref:      AttrRef{},
			expected: true,
		},
		{
			name:     "with resource",
			ref:      AttrRef{Resource: "MyRole"},
			expected: false,
		},
		{
			name:     "with attribute",
			ref:      AttrRef{Attribute: "Arn"},
			expected: false,
		},
		{
			name:     "fully populated",
			ref:      AttrRef{Resource: "MyRole", Attribute: "Arn"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ref.IsZero())
		})
	}
}

func TestDiscoveredResource_Fields(t *testing.T) {
	resource := DiscoveredResource{
		Name:         "MyBucket",
		Type:         "s3.Bucket",
		Package:      "myapp/infra",
		File:         "storage.go",
		Line:         15,
		Dependencies: []string{"ProcessorRole"},
	}

	assert.Equal(t, "MyBucket", resource.Name)
	assert.Equal(t, "s3.Bucket", resource.Type)
	assert.Equal(t, "myapp/infra", resource.Package)
	assert.Equal(t, "storage.go", resource.File)
	assert.Equal(t, 15, resource.Line)
	assert.Equal(t, []string{"ProcessorRole"}, resource.Dependencies)
}

func TestTemplate_JSON(t *testing.T) {
	template := Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Description:              "Test template",
		Resources: map[string]ResourceDef{
			"MyBucket": {
				Type: "AWS::S3::Bucket",
				Properties: map[string]any{
					"BucketName": "test-bucket",
				},
			},
		},
		Parameters: map[string]Parameter{
			"Environment": {
				Type:          "String",
				Description:   "Deployment environment",
				Default:       "dev",
				AllowedValues: []string{"dev", "staging", "prod"},
			},
		},
		Outputs: map[string]Output{
			"BucketArn": {
				Description: "The bucket ARN",
				Value:       map[string][]string{"Fn::GetAtt": {"MyBucket", "Arn"}},
			},
		},
	}

	data, err := json.Marshal(template)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "2010-09-09", parsed["AWSTemplateFormatVersion"])
	assert.Equal(t, "Test template", parsed["Description"])

	resources := parsed["Resources"].(map[string]any)
	bucket := resources["MyBucket"].(map[string]any)
	assert.Equal(t, "AWS::S3::Bucket", bucket["Type"])

	params := parsed["Parameters"].(map[string]any)
	env := params["Environment"].(map[string]any)
	assert.Equal(t, "String", env["Type"])

	outputs := parsed["Outputs"].(map[string]any)
	bucketArn := outputs["BucketArn"].(map[string]any)
	assert.Equal(t, "The bucket ARN", bucketArn["Description"])
}

func TestResourceDef_DependsOn(t *testing.T) {
	resource := ResourceDef{
		Type: "AWS::Lambda::Function",
		Properties: map[string]any{
			"FunctionName": "processor",
		},
		DependsOn: []string{"MyRole", "MyBucket"},
	}

	data, err := json.Marshal(resource)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "AWS::Lambda::Function", parsed["Type"])
	dependsOn := parsed["DependsOn"].([]any)
	assert.Len(t, dependsOn, 2)
	assert.Equal(t, "MyRole", dependsOn[0])
	assert.Equal(t, "MyBucket", dependsOn[1])
}

func TestBuildResult_Success(t *testing.T) {
	result := BuildResult{
		Success: true,
		Template: Template{
			AWSTemplateFormatVersion: "2010-09-09",
			Resources: map[string]ResourceDef{
				"MyBucket": {
					Type: "AWS::S3::Bucket",
				},
			},
		},
		Resources: []string{"MyBucket"},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.True(t, parsed["success"].(bool))
	resources := parsed["resources"].([]any)
	assert.Equal(t, "MyBucket", resources[0])
}

func TestBuildResult_Error(t *testing.T) {
	result := BuildResult{
		Success: false,
		Errors:  []string{"undefined resource: MissingBucket", "parse error at line 15"},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.False(t, parsed["success"].(bool))
	errors := parsed["errors"].([]any)
	assert.Len(t, errors, 2)
}

func TestLintResult(t *testing.T) {
	result := LintResult{
		Success: false,
		Issues: []LintIssue{
			{
				File:     "storage.go",
				Line:     15,
				Column:   10,
				Severity: "warning",
				Message:  "Use pseudo-parameter constant instead of string",
				Rule:     "WAW002",
				Fixable:  true,
			},
			{
				File:     "compute.go",
				Line:     23,
				Column:   5,
				Severity: "error",
				Message:  "undefined resource reference: MissingRole",
				Rule:     "WAW100",
				Fixable:  false,
			},
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.False(t, parsed["success"].(bool))
	issues := parsed["issues"].([]any)
	assert.Len(t, issues, 2)

	issue1 := issues[0].(map[string]any)
	assert.Equal(t, "storage.go", issue1["file"])
	assert.Equal(t, "warning", issue1["severity"])
	assert.True(t, issue1["fixable"].(bool))

	issue2 := issues[1].(map[string]any)
	assert.Equal(t, "error", issue2["severity"])
	assert.False(t, issue2["fixable"].(bool))
}

func TestOutput_WithExport(t *testing.T) {
	output := Output{
		Description: "Bucket ARN for cross-stack reference",
		Value:       map[string][]string{"Fn::GetAtt": {"DataBucket", "Arn"}},
		Export: &struct {
			Name string `json:"Name"`
		}{
			Name: "MyStack-BucketArn",
		},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	export := parsed["Export"].(map[string]any)
	assert.Equal(t, "MyStack-BucketArn", export["Name"])
}

func TestParameter_AllTypes(t *testing.T) {
	tests := []struct {
		name  string
		param Parameter
	}{
		{
			name: "string with allowed values",
			param: Parameter{
				Type:          "String",
				Description:   "Environment name",
				Default:       "dev",
				AllowedValues: []string{"dev", "staging", "prod"},
			},
		},
		{
			name: "number",
			param: Parameter{
				Type:        "Number",
				Description: "Instance count",
				Default:     1,
			},
		},
		{
			name: "ssm parameter",
			param: Parameter{
				Type:        "AWS::SSM::Parameter::Value<String>",
				Description: "SSM parameter value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.param)
			require.NoError(t, err)

			var parsed map[string]any
			require.NoError(t, json.Unmarshal(data, &parsed))

			assert.Equal(t, tt.param.Type, parsed["Type"])
		})
	}
}
