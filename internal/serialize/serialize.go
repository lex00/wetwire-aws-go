// Package serialize provides CloudFormation-specific serialization utilities.
package serialize

import (
	"encoding/json"
	"reflect"
	"strings"
	"unicode"
)

// Resource serializes a Go struct to CloudFormation resource properties.
// It handles:
// - PascalCase field names (BucketName, not bucket_name)
// - Omitting nil/zero values
// - Nested structs
// - AttrRef fields (converts to Fn::GetAtt)
func Resource(v any) (map[string]any, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil
	}

	result := make(map[string]any)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the JSON tag or use field name
		name := getFieldName(field)
		if name == "-" {
			continue
		}

		// Skip zero values unless explicitly required
		if isZeroValue(fieldVal) {
			continue
		}

		// Serialize the field value
		serialized, err := serializeValue(fieldVal)
		if err != nil {
			return nil, err
		}

		if serialized != nil {
			result[name] = serialized
		}
	}

	return result, nil
}

// getFieldName returns the JSON field name for a struct field.
func getFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		return field.Name
	}
	return name
}

// isZeroValue returns true if the value is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		// Check if it has an IsZero method
		if v.CanInterface() {
			if zeroer, ok := v.Interface().(interface{ IsZero() bool }); ok {
				return zeroer.IsZero()
			}
		}
		return false
	default:
		return false
	}
}

// serializeValue converts a reflect.Value to a JSON-compatible value.
func serializeValue(v reflect.Value) (any, error) {
	if !v.IsValid() {
		return nil, nil
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		return serializeValue(v.Elem())
	}

	// Handle interfaces
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, nil
		}
		return serializeValue(v.Elem())
	}

	// Check if the value implements json.Marshaler
	if v.CanInterface() {
		if marshaler, ok := v.Interface().(json.Marshaler); ok {
			data, err := marshaler.MarshalJSON()
			if err != nil {
				return nil, err
			}
			var result any
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	switch v.Kind() {
	case reflect.Struct:
		return Resource(v.Interface())

	case reflect.Slice:
		if v.Len() == 0 {
			return nil, nil
		}
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem, err := serializeValue(v.Index(i))
			if err != nil {
				return nil, err
			}
			result[i] = elem
		}
		return result, nil

	case reflect.Map:
		if v.Len() == 0 {
			return nil, nil
		}
		result := make(map[string]any)
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			val, err := serializeValue(iter.Value())
			if err != nil {
				return nil, err
			}
			result[key] = val
		}
		return result, nil

	case reflect.String:
		return v.String(), nil

	case reflect.Bool:
		return v.Bool(), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint(), nil

	case reflect.Float32, reflect.Float64:
		return v.Float(), nil

	default:
		// Fall back to JSON marshaling
		data, err := json.Marshal(v.Interface())
		if err != nil {
			return nil, err
		}
		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
}

// ToPascalCase converts snake_case to PascalCase.
// e.g., "bucket_name" -> "BucketName"
func ToPascalCase(s string) string {
	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToSnakeCase converts PascalCase to snake_case.
// e.g., "BucketName" -> "bucket_name"
func ToSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
