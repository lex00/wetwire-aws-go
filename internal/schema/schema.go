// Package schema provides offline CloudFormation schema validation.
// It validates resources against known CloudFormation resource schemas.
package schema

import (
	"fmt"
	"strings"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// Options configures schema validation.
type Options struct {
	// Strict enables strict validation mode
	Strict bool
}

// Result contains schema validation results.
type Result struct {
	Valid    bool
	Errors   []wetwire.SchemaError
	Warnings []wetwire.SchemaError
}

// ValidateTemplate validates a CloudFormation template against known schemas.
func ValidateTemplate(template *wetwire.Template, opts Options) (*Result, error) {
	result := &Result{Valid: true}

	for name, resource := range template.Resources {
		errors, warnings := validateResource(name, resource, opts)
		result.Errors = append(result.Errors, errors...)
		result.Warnings = append(result.Warnings, warnings...)
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return result, nil
}

// validateResource validates a single resource.
func validateResource(name string, resource wetwire.ResourceDef, opts Options) ([]wetwire.SchemaError, []wetwire.SchemaError) {
	var errors, warnings []wetwire.SchemaError

	// Validate resource type format
	if !isValidResourceType(resource.Type) {
		errors = append(errors, wetwire.SchemaError{
			Resource: name,
			Property: "Type",
			Message:  fmt.Sprintf("invalid resource type format: %s", resource.Type),
		})
	}

	// Get schema for resource type
	schema, ok := resourceSchemas[resource.Type]
	if !ok {
		// Unknown resource type - this is a warning, not an error
		// CloudFormation may have new resource types not yet in our schema
		warnings = append(warnings, wetwire.SchemaError{
			Resource: name,
			Property: "Type",
			Message:  fmt.Sprintf("unknown resource type: %s (schema not available for validation)", resource.Type),
		})
		return errors, warnings
	}

	// Validate required properties
	for _, required := range schema.Required {
		if _, exists := resource.Properties[required]; !exists {
			errors = append(errors, wetwire.SchemaError{
				Resource: name,
				Property: required,
				Message:  fmt.Sprintf("missing required property: %s", required),
			})
		}
	}

	// Validate known properties
	for propName, propValue := range resource.Properties {
		propSchema, ok := schema.Properties[propName]
		if !ok {
			if opts.Strict {
				warnings = append(warnings, wetwire.SchemaError{
					Resource: name,
					Property: propName,
					Message:  fmt.Sprintf("unknown property: %s", propName),
				})
			}
			continue
		}

		propErrors := validateProperty(name, propName, propValue, propSchema)
		errors = append(errors, propErrors...)
	}

	return errors, warnings
}

// isValidResourceType checks if a resource type has valid format.
func isValidResourceType(resourceType string) bool {
	// CloudFormation resource types follow pattern: AWS::Service::Resource or Custom::*
	if strings.HasPrefix(resourceType, "Custom::") {
		return true
	}
	parts := strings.Split(resourceType, "::")
	if len(parts) != 3 {
		return false
	}
	return parts[0] == "AWS" || parts[0] == "Alexa"
}

// validateProperty validates a property value against its schema.
func validateProperty(resource, property string, value any, schema PropertySchema) []wetwire.SchemaError {
	var errors []wetwire.SchemaError

	// Check type
	if !isValidType(value, schema.Type) {
		errors = append(errors, wetwire.SchemaError{
			Resource: resource,
			Property: property,
			Message:  fmt.Sprintf("expected type %s", schema.Type),
		})
	}

	// Check allowed values
	if len(schema.AllowedValues) > 0 {
		strVal, ok := value.(string)
		if ok {
			found := false
			for _, allowed := range schema.AllowedValues {
				if strVal == allowed {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, wetwire.SchemaError{
					Resource: resource,
					Property: property,
					Message:  fmt.Sprintf("value %q not in allowed values: %v", strVal, schema.AllowedValues),
				})
			}
		}
	}

	return errors
}

// isValidType checks if a value matches the expected type.
func isValidType(value any, expectedType string) bool {
	// Handle CloudFormation intrinsic functions - they're always valid
	if m, ok := value.(map[string]any); ok {
		for key := range m {
			if strings.HasPrefix(key, "Fn::") || key == "Ref" {
				return true
			}
		}
	}

	switch expectedType {
	case "String":
		_, ok := value.(string)
		return ok
	case "Integer":
		switch value.(type) {
		case int, int32, int64, float64:
			return true
		}
		return false
	case "Boolean":
		_, ok := value.(bool)
		return ok
	case "List":
		_, ok := value.([]any)
		return ok
	case "Map":
		_, ok := value.(map[string]any)
		return ok
	case "Json":
		return true // Accept any value as JSON
	default:
		return true // Unknown type - accept
	}
}

// ResourceSchema defines the schema for a resource type.
type ResourceSchema struct {
	Type       string
	Required   []string
	Properties map[string]PropertySchema
}

// PropertySchema defines the schema for a property.
type PropertySchema struct {
	Type          string
	Required      bool
	AllowedValues []string
}
