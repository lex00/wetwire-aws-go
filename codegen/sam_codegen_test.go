package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSAMService(t *testing.T) {
	// Create temp directory for output
	tmpDir, err := os.MkdirTemp("", "sam-codegen-test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate SAM service
	stats := &GenerationStats{}
	err = generateService(SAMSpec, tmpDir, false, stats)
	if err != nil {
		t.Fatalf("generating SAM service: %v", err)
	}

	// Verify serverless directory was created
	serverlessDir := filepath.Join(tmpDir, "resources", "serverless")
	if _, err := os.Stat(serverlessDir); os.IsNotExist(err) {
		t.Fatalf("serverless directory not created")
	}

	// Verify resource files were created
	// Note: toSnakeCase converts GraphQLApi to graph_q_l_api
	expectedFiles := []string{
		"function.go",
		"api.go",
		"http_api.go",
		"simple_table.go",
		"layer_version.go",
		"state_machine.go",
		"application.go",
		"connector.go",
		"graph_q_l_api.go",
		"doc.go",
		"types.go",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(serverlessDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file not created: %s", file)
		}
	}

	// Verify stats
	if stats.Resources != 9 {
		t.Errorf("expected 9 resources, got %d", stats.Resources)
	}
}

func TestGenerateSAMFunction(t *testing.T) {
	// Generate Function resource
	code, err := generateResource(SAMSpec, SAMSpec.Resources["Function"])
	if err != nil {
		t.Fatalf("generating Function: %v", err)
	}

	content := string(code)

	// Verify package declaration
	if !contains(content, "package serverless") {
		t.Error("missing package declaration")
	}

	// Verify struct definition
	if !contains(content, "type Function struct") {
		t.Error("missing Function struct")
	}

	// Verify ResourceType method
	if !contains(content, `return "AWS::Serverless::Function"`) {
		t.Error("missing ResourceType method")
	}

	// Verify key properties
	expectedProps := []string{
		"Handler", "Runtime", "CodeUri", "FunctionName",
		"MemorySize", "Timeout", "Environment", "Events",
	}
	for _, prop := range expectedProps {
		if !contains(content, prop) {
			t.Errorf("missing property: %s", prop)
		}
	}
}

func TestGenerateSAMApi(t *testing.T) {
	code, err := generateResource(SAMSpec, SAMSpec.Resources["Api"])
	if err != nil {
		t.Fatalf("generating Api: %v", err)
	}

	content := string(code)

	if !contains(content, "type Api struct") {
		t.Error("missing Api struct")
	}

	if !contains(content, `return "AWS::Serverless::Api"`) {
		t.Error("missing ResourceType method")
	}
}

func TestGenerateSAMPropertyTypes(t *testing.T) {
	// Generate property types for Function
	typeNames := []string{
		"Function_Environment",
		"Function_EventSource",
		"Function_VpcConfig",
	}

	code, err := generateResourcePropertyTypes(SAMSpec, "Function", typeNames)
	if err != nil {
		t.Fatalf("generating property types: %v", err)
	}

	content := string(code)

	// Verify property type structs
	if !contains(content, "type Function_Environment struct") {
		t.Error("missing Function_Environment struct")
	}

	if !contains(content, "type Function_VpcConfig struct") {
		t.Error("missing Function_VpcConfig struct")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
