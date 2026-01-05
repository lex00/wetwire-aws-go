package main

import (
	"testing"
)

func TestSAMSpec(t *testing.T) {
	// Test that SAM spec has all 9 resource types
	expectedResources := []string{
		"Function",
		"Api",
		"HttpApi",
		"SimpleTable",
		"LayerVersion",
		"StateMachine",
		"Application",
		"Connector",
		"GraphQLApi",
	}

	if SAMSpec == nil {
		t.Fatal("SAMSpec is nil")
	}

	if SAMSpec.Name != "serverless" {
		t.Errorf("expected service name 'serverless', got %q", SAMSpec.Name)
	}

	if SAMSpec.CFPrefix != "AWS::Serverless" {
		t.Errorf("expected CFPrefix 'AWS::Serverless', got %q", SAMSpec.CFPrefix)
	}

	for _, name := range expectedResources {
		if _, ok := SAMSpec.Resources[name]; !ok {
			t.Errorf("missing resource: %s", name)
		}
	}

	if len(SAMSpec.Resources) != 9 {
		t.Errorf("expected 9 resources, got %d", len(SAMSpec.Resources))
	}
}

func TestSAMFunctionResource(t *testing.T) {
	fn, ok := SAMSpec.Resources["Function"]
	if !ok {
		t.Fatal("Function resource not found")
	}

	if fn.CFType != "AWS::Serverless::Function" {
		t.Errorf("expected CFType 'AWS::Serverless::Function', got %q", fn.CFType)
	}

	// Check required properties exist
	requiredProps := []string{"Handler", "Runtime", "CodeUri"}
	for _, prop := range requiredProps {
		if _, ok := fn.Properties[prop]; !ok {
			t.Errorf("missing property: %s", prop)
		}
	}

	// Check important optional properties
	optionalProps := []string{
		"FunctionName", "Description", "MemorySize", "Timeout",
		"Environment", "Events", "Policies", "VpcConfig",
		"Architectures", "Layers", "Tracing",
	}
	for _, prop := range optionalProps {
		if _, ok := fn.Properties[prop]; !ok {
			t.Errorf("missing optional property: %s", prop)
		}
	}
}

func TestSAMApiResource(t *testing.T) {
	api, ok := SAMSpec.Resources["Api"]
	if !ok {
		t.Fatal("Api resource not found")
	}

	if api.CFType != "AWS::Serverless::Api" {
		t.Errorf("expected CFType 'AWS::Serverless::Api', got %q", api.CFType)
	}

	// Check key properties
	props := []string{"StageName", "DefinitionBody", "DefinitionUri"}
	for _, prop := range props {
		if _, ok := api.Properties[prop]; !ok {
			t.Errorf("missing property: %s", prop)
		}
	}
}

func TestSAMPropertyTypes(t *testing.T) {
	// Function should have property types
	expectedPropTypes := []string{
		"Function_Environment",
		"Function_EventSource",
		"Function_VpcConfig",
	}

	for _, name := range expectedPropTypes {
		if _, ok := SAMSpec.PropertyTypes[name]; !ok {
			t.Errorf("missing property type: %s", name)
		}
	}
}
