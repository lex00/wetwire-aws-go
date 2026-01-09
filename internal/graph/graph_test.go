package graph

import (
	"strings"
	"testing"

	wetwire "github.com/lex00/wetwire-aws-go"
)

func TestGenerator_Generate_SimpleGraph(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {
			Name: "MyBucket",
			Type: "s3.Bucket",
		},
		"MyFunction": {
			Name:         "MyFunction",
			Type:         "lambda.Function",
			Dependencies: []string{"MyBucket"},
		},
	}

	gen := &Generator{}
	var sb strings.Builder
	err := gen.Generate(resources, nil, &sb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := sb.String()

	// Should be a digraph
	if !strings.Contains(output, "digraph") {
		t.Error("expected digraph declaration")
	}

	// Should have nodes for both resources
	if !strings.Contains(output, "MyBucket") {
		t.Error("expected MyBucket node")
	}
	if !strings.Contains(output, "MyFunction") {
		t.Error("expected MyFunction node")
	}

	// Should have edge from MyFunction to MyBucket
	if !strings.Contains(output, "MyFunction") || !strings.Contains(output, "MyBucket") {
		t.Error("expected edge from MyFunction to MyBucket")
	}
}

func TestGenerator_Generate_WithGetAtt(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyRole": {
			Name: "MyRole",
			Type: "iam.Role",
		},
		"MyFunction": {
			Name:         "MyFunction",
			Type:         "lambda.Function",
			Dependencies: []string{"MyRole"},
			AttrRefUsages: []wetwire.AttrRefUsage{
				{ResourceName: "MyRole", Attribute: "Arn", FieldPath: "Role"},
			},
		},
	}

	gen := &Generator{}
	var sb strings.Builder
	err := gen.Generate(resources, nil, &sb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := sb.String()

	// GetAtt edges should be blue
	if !strings.Contains(output, "blue") {
		t.Error("expected blue color for GetAtt edge")
	}
}

func TestGenerator_Generate_WithParameters(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {
			Name:         "MyBucket",
			Type:         "s3.Bucket",
			Dependencies: []string{"BucketName"},
		},
	}

	parameters := map[string]wetwire.DiscoveredParameter{
		"BucketName": {
			Name: "BucketName",
		},
	}

	gen := &Generator{IncludeParameters: true}
	var sb strings.Builder
	err := gen.Generate(resources, parameters, &sb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := sb.String()

	// Should include parameter node
	if !strings.Contains(output, "BucketName") {
		t.Error("expected BucketName parameter node")
	}

	// Parameter nodes should be ellipse/dashed
	if !strings.Contains(output, "ellipse") {
		t.Error("expected ellipse shape for parameter")
	}
}

func TestGenerator_Generate_ClusterByType(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"Bucket1": {
			Name: "Bucket1",
			Type: "s3.Bucket",
		},
		"Bucket2": {
			Name: "Bucket2",
			Type: "s3.Bucket",
		},
		"MyFunction": {
			Name: "MyFunction",
			Type: "lambda.Function",
		},
	}

	gen := &Generator{ClusterByType: true}
	var sb strings.Builder
	err := gen.Generate(resources, nil, &sb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := sb.String()

	// Should have cluster for S3 (multiple resources)
	if !strings.Contains(output, "cluster_") {
		t.Error("expected cluster subgraph")
	}
}

func TestGenerator_Generate_MermaidFormat(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {
			Name: "MyBucket",
			Type: "s3.Bucket",
		},
		"MyFunction": {
			Name:         "MyFunction",
			Type:         "lambda.Function",
			Dependencies: []string{"MyBucket"},
		},
	}

	gen := &Generator{Format: FormatMermaid}
	var sb strings.Builder
	err := gen.Generate(resources, nil, &sb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := sb.String()

	// Should be mermaid format (flowchart or graph)
	if !strings.Contains(output, "graph") && !strings.Contains(output, "flowchart") {
		t.Errorf("expected mermaid graph/flowchart, got:\n%s", output)
	}

	// Should NOT be DOT format
	if strings.Contains(output, "digraph") {
		t.Error("expected mermaid format, not DOT")
	}
}

func TestGenerator_GenerateString(t *testing.T) {
	resources := map[string]wetwire.DiscoveredResource{
		"MyBucket": {
			Name: "MyBucket",
			Type: "s3.Bucket",
		},
	}

	gen := &Generator{}
	output, err := gen.GenerateString(resources, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "MyBucket") {
		t.Error("expected MyBucket in output")
	}
}
