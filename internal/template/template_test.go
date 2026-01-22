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
		"Handler":    "bootstrap",
		"Runtime":    "provided.al2",
		"CodeUri":    "./hello-world/",
		"MemorySize": 128,
		"Timeout":    5,
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
		goType  string
		cfnType string
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

func TestCfResourceType_EC2(t *testing.T) {
	tests := []struct {
		goType  string
		cfnType string
	}{
		{"ec2.InternetGateway", "AWS::EC2::InternetGateway"},
		{"ec2.EIP", "AWS::EC2::EIP"},
		{"ec2.RouteTable", "AWS::EC2::RouteTable"},
		{"ec2.VPC", "AWS::EC2::VPC"},
		{"ec2.Subnet", "AWS::EC2::Subnet"},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			result := cfResourceType(tt.goType)
			assert.Equal(t, tt.cfnType, result)
		})
	}
}

func TestBuilder_Build_OutputWithAttrRefInJoin(t *testing.T) {
	// Test that AttrRef inside Join intrinsic is resolved correctly
	// This tests the fix for issue #92 - empty GetAtt in intrinsic functions
	resources := map[string]wetwire.DiscoveredResource{
		"MyALB": {
			Name:    "MyALB",
			Type:    "elasticloadbalancingv2.LoadBalancer",
			Package: "infra",
			File:    "network.go",
			Line:    5,
		},
	}

	outputs := map[string]wetwire.DiscoveredOutput{
		"ALBUrl": {
			Name: "ALBUrl",
			File: "outputs.go",
			Line: 10,
			AttrRefUsages: []wetwire.AttrRefUsage{
				{
					ResourceName: "MyALB",
					Attribute:    "DNSName",
					FieldPath:    "Value.Values", // Go field path
				},
			},
		},
	}

	builder := NewBuilderFull(resources, nil, outputs, nil, nil)
	builder.SetValue("MyALB", map[string]any{
		"Name": "my-alb",
	})
	// Simulate the serialized output value - Join becomes Fn::Join with array
	// The AttrRef field access (MyALB.DNSName) serializes to empty GetAtt
	builder.SetValue("ALBUrl", map[string]any{
		"Value": map[string]any{
			"Fn::Join": []any{
				"",
				[]any{
					"https://",
					map[string]any{
						"Fn::GetAtt": []any{"", ""}, // Empty - needs to be resolved
					},
				},
			},
		},
		"Description": "ALB URL",
	})

	template, err := builder.Build()
	require.NoError(t, err)

	// Verify the output exists
	require.Contains(t, template.Outputs, "ALBUrl")
	output := template.Outputs["ALBUrl"]

	// The Value should be a Join with the GetAtt properly resolved
	value, ok := output.Value.(map[string]any)
	require.True(t, ok, "Value should be a map")

	fnJoin, ok := value["Fn::Join"].([]any)
	require.True(t, ok, "Should have Fn::Join")
	require.Len(t, fnJoin, 2, "Fn::Join should have delimiter and values")

	values, ok := fnJoin[1].([]any)
	require.True(t, ok, "Second element should be values array")
	require.Len(t, values, 2, "Values should have 2 elements")

	// The second element should be the resolved GetAtt
	getAttMap, ok := values[1].(map[string]any)
	require.True(t, ok, "Second value should be a GetAtt map")

	getAtt, ok := getAttMap["Fn::GetAtt"].([]string)
	require.True(t, ok, "Should have Fn::GetAtt with string array")
	assert.Equal(t, "MyALB", getAtt[0], "GetAtt resource should be resolved")
	assert.Equal(t, "DNSName", getAtt[1], "GetAtt attribute should be resolved")
}

func TestBuilder_TransformValueWithPath_IntrinsicFieldMapping(t *testing.T) {
	// Unit test for the intrinsic field name mapping
	builder := NewBuilder(nil)

	attrRefsByPath := map[string]wetwire.AttrRefUsage{
		"Value.Values": {
			ResourceName: "MyResource",
			Attribute:    "Arn",
			FieldPath:    "Value.Values",
		},
	}

	// Simulate a serialized Join with empty GetAtt
	input := map[string]any{
		"Fn::Join": []any{
			"-",
			[]any{
				"prefix",
				map[string]any{
					"Fn::GetAtt": []any{"", ""},
				},
			},
		},
	}

	result := builder.transformValueWithPath(input, "Value", attrRefsByPath)

	// Verify the GetAtt was resolved
	resultMap := result.(map[string]any)
	fnJoin := resultMap["Fn::Join"].([]any)
	values := fnJoin[1].([]any)
	getAttMap := values[1].(map[string]any)
	getAtt := getAttMap["Fn::GetAtt"].([]string)

	assert.Equal(t, "MyResource", getAtt[0])
	assert.Equal(t, "Arn", getAtt[1])
}

func TestSetVarAttrRefs(t *testing.T) {
	builder := NewBuilder(nil)

	refs := map[string]VarAttrRefInfo{
		"MyResource": {
			AttrRefs: []wetwire.AttrRefUsage{
				{ResourceName: "OtherResource", Attribute: "Arn", FieldPath: "Role"},
			},
		},
	}

	builder.SetVarAttrRefs(refs)

	// The internal map should be set
	assert.Len(t, builder.varAttrRefs, 1)
	assert.Contains(t, builder.varAttrRefs, "MyResource")
}

func TestResolveAllAttrRefs(t *testing.T) {
	builder := NewBuilder(map[string]wetwire.DiscoveredResource{
		"Function": {Name: "Function", Dependencies: []string{"Role"}},
		"Role":     {Name: "Role"},
	})

	builder.SetVarAttrRefs(map[string]VarAttrRefInfo{
		"Function": {
			AttrRefs: []wetwire.AttrRefUsage{
				{ResourceName: "Role", Attribute: "Arn", FieldPath: "Role"},
			},
		},
		"Role": {
			AttrRefs: []wetwire.AttrRefUsage{
				{ResourceName: "Policy", Attribute: "PolicyArn", FieldPath: "Policies"},
			},
		},
	})

	refs := builder.resolveAllAttrRefs("Function")

	// Should find AttrRefs from Function and from its dependency Role
	assert.Len(t, refs, 2)
}

func TestResolveAllAttrRefs_VarRefs(t *testing.T) {
	builder := NewBuilder(nil)

	builder.SetVarAttrRefs(map[string]VarAttrRefInfo{
		"Output": {
			VarRefs: map[string]string{"Value": "Helper"},
		},
		"Helper": {
			AttrRefs: []wetwire.AttrRefUsage{
				{ResourceName: "Bucket", Attribute: "Arn", FieldPath: "Arn"},
			},
		},
	})

	refs := builder.resolveAllAttrRefs("Output")

	// Should follow VarRefs and find the nested AttrRef
	assert.Len(t, refs, 1)
	assert.Equal(t, "Bucket", refs[0].ResourceName)
}

func TestResolveAllAttrRefs_CircularDeps(t *testing.T) {
	builder := NewBuilder(map[string]wetwire.DiscoveredResource{
		"A": {Name: "A", Dependencies: []string{"B"}},
		"B": {Name: "B", Dependencies: []string{"A"}},
	})

	// Should not infinite loop
	refs := builder.resolveAllAttrRefs("A")
	assert.Empty(t, refs)
}

func TestBuilder_Build_WithParameters(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {Name: "MyBucket", Type: "s3.Bucket"},
	}
	parameters := map[string]wetwire.DiscoveredParameter{
		"Environment": {Name: "Environment"},
	}

	builder := NewBuilderFull(resources, parameters, nil, nil, nil)
	builder.SetValue("MyBucket", map[string]any{"BucketName": "test"})
	builder.SetValue("Environment", map[string]any{
		"Type":        "String",
		"Default":     "dev",
		"Description": "Deployment environment",
	})

	tmpl, err := builder.Build()
	require.NoError(t, err)

	assert.Len(t, tmpl.Parameters, 1)
	param := tmpl.Parameters["Environment"]
	assert.Equal(t, "String", param.Type)
	assert.Equal(t, "dev", param.Default)
	assert.Equal(t, "Deployment environment", param.Description)
}

func TestBuilder_Build_WithParameterConstraints(t *testing.T) {
	parameters := map[string]wetwire.DiscoveredParameter{
		"Port": {Name: "Port"},
	}

	builder := NewBuilderFull(nil, parameters, nil, nil, nil)
	builder.SetValue("Port", map[string]any{
		"Type":                  "Number",
		"MinValue":              float64(1),
		"MaxValue":              float64(65535),
		"AllowedValues":         []any{80, 443, 8080},
		"AllowedPattern":        "\\d+",
		"ConstraintDescription": "Must be a valid port number",
		"NoEcho":                true,
		"MinLength":             float64(1),
		"MaxLength":             float64(5),
	})

	tmpl, err := builder.Build()
	require.NoError(t, err)

	param := tmpl.Parameters["Port"]
	assert.Equal(t, "Number", param.Type)
	assert.NotNil(t, param.MinValue)
	assert.NotNil(t, param.MaxValue)
	assert.Len(t, param.AllowedValues, 3)
	assert.Equal(t, "\\d+", param.AllowedPattern)
	assert.Equal(t, "Must be a valid port number", param.ConstraintDescription)
	assert.True(t, param.NoEcho)
	assert.NotNil(t, param.MinLength)
	assert.NotNil(t, param.MaxLength)
}

func TestBuilder_Build_ParameterNotMap(t *testing.T) {
	parameters := map[string]wetwire.DiscoveredParameter{
		"BadParam": {Name: "BadParam"},
	}

	builder := NewBuilderFull(nil, parameters, nil, nil, nil)
	builder.SetValue("BadParam", "not a map")

	tmpl, err := builder.Build()
	require.NoError(t, err)

	// Should get default String type
	param := tmpl.Parameters["BadParam"]
	assert.Equal(t, "String", param.Type)
}

func TestBuilder_Build_WithMappings(t *testing.T) {
	mappings := map[string]wetwire.DiscoveredMapping{
		"RegionMap": {Name: "RegionMap"},
	}

	builder := NewBuilderFull(nil, nil, nil, mappings, nil)
	builder.SetValue("RegionMap", map[string]any{
		"us-east-1": map[string]any{"AMI": "ami-12345678"},
		"us-west-2": map[string]any{"AMI": "ami-87654321"},
	})

	tmpl, err := builder.Build()
	require.NoError(t, err)

	assert.Len(t, tmpl.Mappings, 1)
	regionMap := tmpl.Mappings["RegionMap"].(map[string]any)
	assert.Contains(t, regionMap, "us-east-1")
}

func TestBuilder_Build_WithConditions(t *testing.T) {
	conditions := map[string]wetwire.DiscoveredCondition{
		"IsProd": {Name: "IsProd"},
	}

	builder := NewBuilderFull(nil, nil, nil, nil, conditions)
	builder.SetValue("IsProd", map[string]any{
		"Fn::Equals": []any{"${Environment}", "prod"},
	})

	tmpl, err := builder.Build()
	require.NoError(t, err)

	assert.Len(t, tmpl.Conditions, 1)
	assert.Contains(t, tmpl.Conditions, "IsProd")
}

func TestBuilder_Build_WithOutputExport(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {Name: "MyBucket", Type: "s3.Bucket"},
	}
	outputs := map[string]wetwire.DiscoveredOutput{
		"BucketName": {Name: "BucketName"},
	}

	builder := NewBuilderFull(resources, nil, outputs, nil, nil)
	builder.SetValue("MyBucket", map[string]any{"BucketName": "test"})
	builder.SetValue("BucketName", map[string]any{
		"Value":       map[string]any{"Ref": "MyBucket"},
		"Description": "The bucket name",
		"Export":      map[string]any{"Name": "MyStack-BucketName"},
	})

	tmpl, err := builder.Build()
	require.NoError(t, err)

	output := tmpl.Outputs["BucketName"]
	assert.Equal(t, "The bucket name", output.Description)
	assert.NotNil(t, output.Export)
	assert.Equal(t, "MyStack-BucketName", output.Export.Name)
}

func TestBuilder_Build_WithOutputExportName(t *testing.T) {
	outputs := map[string]wetwire.DiscoveredOutput{
		"BucketArn": {Name: "BucketArn"},
	}

	builder := NewBuilderFull(nil, nil, outputs, nil, nil)
	builder.SetValue("BucketArn", map[string]any{
		"Value":      "arn:aws:s3:::my-bucket",
		"ExportName": "MyStack-BucketArn",
	})

	tmpl, err := builder.Build()
	require.NoError(t, err)

	output := tmpl.Outputs["BucketArn"]
	assert.NotNil(t, output.Export)
	assert.Equal(t, "MyStack-BucketArn", output.Export.Name)
}

func TestStripArrayIndices(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple", "Simple"},
		{"Policies[0]", "Policies"},
		{"Policies[0].PolicyDocument", "Policies.PolicyDocument"},
		{"Statement[0].Resource[1]", "Statement.Resource"},
		{"Tags[0].Key", "Tags.Key"},
		{"[0]", ""},
		{"Nested[0][1]", "Nested"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripArrayIndices(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformValueWithPath_PreservesRef(t *testing.T) {
	builder := NewBuilder(nil)

	input := map[string]any{
		"Ref": "MyParameter",
	}

	result := builder.transformValueWithPath(input, "Value", nil)

	resultMap := result.(map[string]any)
	assert.Equal(t, "MyParameter", resultMap["Ref"])
}

func TestTransformValueWithPath_PreservesSub(t *testing.T) {
	builder := NewBuilder(nil)

	input := map[string]any{
		"Fn::Sub": "${AWS::StackName}-bucket",
	}

	result := builder.transformValueWithPath(input, "Value", nil)

	resultMap := result.(map[string]any)
	assert.Equal(t, "${AWS::StackName}-bucket", resultMap["Fn::Sub"])
}

func TestTransformValueWithPath_StrippedPathMatch(t *testing.T) {
	builder := NewBuilder(nil)

	attrRefsByPath := map[string]wetwire.AttrRefUsage{
		"Policies.PolicyDocument.Statement.Resource": {
			ResourceName: "MyBucket",
			Attribute:    "Arn",
			FieldPath:    "Policies.PolicyDocument.Statement.Resource",
		},
	}

	input := map[string]any{
		"Fn::GetAtt": []any{"", ""},
	}

	// Path with array indices that should match after stripping
	result := builder.transformValueWithPath(input, "Policies[0].PolicyDocument.Statement[0].Resource", attrRefsByPath)

	resultMap := result.(map[string]any)
	getAtt := resultMap["Fn::GetAtt"].([]string)
	assert.Equal(t, "MyBucket", getAtt[0])
	assert.Equal(t, "Arn", getAtt[1])
}

func TestTransformValueWithPath_SuffixMatch(t *testing.T) {
	builder := NewBuilder(nil)

	attrRefsByPath := map[string]wetwire.AttrRefUsage{
		"Role": {
			ResourceName: "MyRole",
			Attribute:    "Arn",
			FieldPath:    "Role",
		},
	}

	input := map[string]any{
		"Fn::GetAtt": []any{"", ""},
	}

	// Path that has Role as a suffix
	result := builder.transformValueWithPath(input, "Properties.Role", attrRefsByPath)

	resultMap := result.(map[string]any)
	getAtt := resultMap["Fn::GetAtt"].([]string)
	assert.Equal(t, "MyRole", getAtt[0])
	assert.Equal(t, "Arn", getAtt[1])
}

func TestTransformValueWithPath_RecursiveSlice(t *testing.T) {
	builder := NewBuilder(nil)

	attrRefsByPath := map[string]wetwire.AttrRefUsage{
		"Items.Value": {
			ResourceName: "MyResource",
			Attribute:    "Id",
			FieldPath:    "Items.Value",
		},
	}

	input := []any{
		map[string]any{
			"Name": "first",
			"Value": map[string]any{
				"Fn::GetAtt": []any{"", ""},
			},
		},
	}

	result := builder.transformValueWithPath(input, "Items", attrRefsByPath)

	resultSlice := result.([]any)
	firstItem := resultSlice[0].(map[string]any)
	valueMap := firstItem["Value"].(map[string]any)
	getAtt := valueMap["Fn::GetAtt"].([]string)
	assert.Equal(t, "MyResource", getAtt[0])
	assert.Equal(t, "Id", getAtt[1])
}

func TestBuilder_Build_UnknownResourceType(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyResource": {Name: "MyResource", Type: "unknown.ResourceType"},
	}

	builder := NewBuilder(resources)
	builder.SetValue("MyResource", map[string]any{})

	_, err := builder.Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestCfResourceType_Standard(t *testing.T) {
	tests := []struct {
		goType  string
		cfnType string
	}{
		{"s3.Bucket", "AWS::S3::Bucket"},
		{"lambda.Function", "AWS::Lambda::Function"},
		{"iam.Role", "AWS::IAM::Role"},
		{"dynamodb.Table", "AWS::DynamoDB::Table"},
		{"sns.Topic", "AWS::SNS::Topic"},
		{"sqs.Queue", "AWS::SQS::Queue"},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			result := cfResourceType(tt.goType)
			assert.Equal(t, tt.cfnType, result)
		})
	}
}

func TestGoPackageToCFService(t *testing.T) {
	tests := []struct {
		pkg     string
		service string
	}{
		{"ec2", "EC2"},
		{"s3", "S3"},
		{"lambda", "Lambda"},
		{"dynamodb", "DynamoDB"},
		{"apigateway", "ApiGateway"},
		{"cloudwatch", "CloudWatch"},
		{"elasticloadbalancingv2", "ElasticLoadBalancingV2"},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			result := goPackageToCFService(tt.pkg)
			assert.Equal(t, tt.service, result)
		})
	}
}
