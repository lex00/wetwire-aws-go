package importer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/lex00/wetwire-aws-go/resources"
)

// valueToBlockStyleProperty converts a property value to block style.
// Returns either a var reference (for typed properties) or a literal value.
func valueToBlockStyleProperty(ctx *codegenContext, value any, propName string, parentVarName string) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case *IRIntrinsic:
		goCode := intrinsicToGo(ctx, v)
		// Check if this is a Ref to a list-type Parameter - always wrap these
		// because Parameter struct type can't be directly assigned to []any
		if v.Type == IntrinsicRef {
			target := fmt.Sprintf("%v", v.Args)
			if param, ok := ctx.template.Parameters[target]; ok {
				if isListTypeParameter(param) {
					return fmt.Sprintf("[]any{%s}", goCode)
				}
			}
		}
		// If this property expects a list type and the intrinsic needs wrapping,
		// wrap it in []any{} to satisfy Go's type system
		if isListTypeProperty(propName) && intrinsicNeedsArrayWrapping(v) {
			return fmt.Sprintf("[]any{%s}", goCode)
		}
		return goCode

	case bool:
		if v {
			return "true"
		}
		return "false"

	case int, int64:
		return fmt.Sprintf("%d", v)

	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)

	case string:
		// Check for pseudo-parameters that should be constants
		if pseudoConst, ok := pseudoParameterConstants[v]; ok {
			ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
			return pseudoConst
		}
		// Check for enum constants
		if enumConst := tryEnumConstant(ctx, v); enumConst != "" {
			return enumConst
		}
		return goStringLiteral(v)

	case []any:
		if len(v) == 0 {
			return "[]any{}"
		}
		// Check if array elements are maps that should be typed
		if propName != "" && len(v) > 0 {
			if _, isMap := v[0].(map[string]any); isMap {
				elemTypeName := getArrayElementTypeName(ctx, propName)
				if elemTypeName != "" {
					return arrayToBlockStyle(ctx, v, elemTypeName, parentVarName, propName)
				}
			}
		}
		// Fallback: inline array ([]any{} is plain Go, no import needed)
		var items []string
		for _, item := range v {
			items = append(items, valueToBlockStyleProperty(ctx, item, "", parentVarName))
		}
		return fmt.Sprintf("[]any{%s}", strings.Join(items, ", "))

	case map[string]any:
		// Check if this is an intrinsic function map
		if len(v) == 1 {
			for k := range v {
				if k == "Ref" || strings.HasPrefix(k, "Fn::") || k == "Condition" {
					intrinsic := mapToIntrinsic(v)
					if intrinsic != nil {
						return intrinsicToGo(ctx, intrinsic)
					}
				}
			}
		}

		// Check if this is a policy document field
		if isPolicyDocumentField(propName) {
			return policyDocToBlocks(ctx, v, parentVarName, propName)
		}

		// Check if this should be a typed property block
		typeName := getPropertyTypeName(ctx, propName)
		if typeName != "" && allKeysValidIdentifiers(v) {
			// Create a block for this property
			blockVarName := parentVarName + propName
			fullTypeName := fmt.Sprintf("%s.%s", ctx.currentResource, typeName)

			// Check if this field is a pointer BEFORE updating type context
			// (we need to check against the parent type, not the nested type)
			needsPointer := isPointerField(ctx, propName)

			// Save and update type context
			savedTypeName := ctx.currentTypeName
			ctx.currentTypeName = typeName

			ctx.propertyBlocks = append(ctx.propertyBlocks, propertyBlock{
				varName:    blockVarName,
				typeName:   fullTypeName,
				properties: v,
				isPointer:  needsPointer,
				order:      len(ctx.propertyBlocks),
			})

			// Restore type context
			ctx.currentTypeName = savedTypeName

			// Pointer fields are now `any` type, so no & prefix needed
			return blockVarName
		}

		// Fallback: inline map
		var items []string
		for _, k := range sortedKeys(v) {
			val := v[k]
			items = append(items, fmt.Sprintf("\t%q: %s,", k, valueToBlockStyleProperty(ctx, val, k, parentVarName)))
		}
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		var jsonLiteral string
		if len(items) == 0 {
			jsonLiteral = "Json{}"
		} else {
			jsonLiteral = fmt.Sprintf("Json{\n%s\n}", strings.Join(items, "\n"))
		}
		// If this property expects a list type, wrap the map in []any{}
		// This handles cases like SAM Policies where a map can be used instead of a list
		if isListTypeProperty(propName) {
			return fmt.Sprintf("[]any{%s}", jsonLiteral)
		}
		return jsonLiteral
	}

	return fmt.Sprintf("%#v", value)
}

// generatePropertyBlock generates a var declaration for a property type block.
func generatePropertyBlock(ctx *codegenContext, block propertyBlock) string {
	var lines []string

	// Always use value types (no & prefix) for AST extraction compatibility
	// The consuming code will take addresses as needed
	lines = append(lines, fmt.Sprintf("var %s = %s{", block.varName, block.typeName))

	// Add intrinsics import if this is a Tag type
	if block.typeName == "Tag" {
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
	}

	// Set type context from block type name (e.g., "s3.Bucket_BucketEncryption" -> "Bucket_BucketEncryption")
	// This is needed for nested property type resolution
	savedTypeName := ctx.currentTypeName
	if idx := strings.LastIndex(block.typeName, "."); idx >= 0 {
		ctx.currentTypeName = block.typeName[idx+1:]
	}

	// Check if this is a policy-related type
	isPolicyType := block.typeName == "PolicyDocument" || block.typeName == "PolicyStatement" || block.typeName == "DenyStatement"

	// Sort property keys for deterministic output
	keys := make([]string, 0, len(block.properties))
	for k := range block.properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := block.properties[k]
		var fieldVal string

		// Special handling for policy types
		if isPolicyType {
			switch k {
			case "Principal":
				fieldVal = principalToGo(ctx, v)
			case "Condition":
				fieldVal = conditionToGo(ctx, v)
			case "Statement":
				// Statement is a list of var references (strings)
				if varNames, ok := v.([]string); ok {
					fieldVal = fmt.Sprintf("[]any{%s}", strings.Join(varNames, ", "))
				} else {
					fieldVal = valueToGoForBlock(ctx, v, k, block.varName)
				}
			default:
				fieldVal = valueToGoForBlock(ctx, v, k, block.varName)
			}
		} else {
			// Process the value, which may create nested property blocks
			fieldVal = valueToGoForBlock(ctx, v, k, block.varName)
		}

		// Transform field name for Go keyword conflicts
		// Use type-aware transformation to handle ResourceType correctly for nested types
		goFieldName := transformGoFieldNameForType(k, ctx.currentTypeName)
		lines = append(lines, fmt.Sprintf("\t%s: %s,", goFieldName, fieldVal))
	}

	// Restore type context
	ctx.currentTypeName = savedTypeName

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

// valueToGoForBlock converts values for block generation, creating nested blocks as needed.
// Returns either a literal value or a reference to another block variable.
func valueToGoForBlock(ctx *codegenContext, value any, propName string, parentVarName string) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case *IRIntrinsic:
		// If we have a property name, update the type context before processing
		// the intrinsic so that nested values use the correct type.
		// For example, S3Location inside an If should use Association_S3OutputLocation,
		// not the parent Association_InstanceAssociationOutputLocation.
		var goCode string
		if propName != "" {
			typeName := getPropertyTypeName(ctx, propName)
			if typeName != "" {
				savedTypeName := ctx.currentTypeName
				ctx.currentTypeName = typeName
				goCode = intrinsicToGo(ctx, v)
				ctx.currentTypeName = savedTypeName
			} else {
				goCode = intrinsicToGo(ctx, v)
			}
		} else {
			goCode = intrinsicToGo(ctx, v)
		}
		// Check if this is a Ref to a list-type Parameter - always wrap these
		// because Parameter struct type can't be directly assigned to []any
		if v.Type == IntrinsicRef {
			target := fmt.Sprintf("%v", v.Args)
			if param, ok := ctx.template.Parameters[target]; ok {
				if isListTypeParameter(param) {
					return fmt.Sprintf("[]any{%s}", goCode)
				}
			}
		}
		// If this property expects a list type and the intrinsic needs wrapping,
		// wrap it in []any{} to satisfy Go's type system
		if isListTypeProperty(propName) && intrinsicNeedsArrayWrapping(v) {
			return fmt.Sprintf("[]any{%s}", goCode)
		}
		return goCode

	case bool:
		if v {
			return "true"
		}
		return "false"

	case int, int64:
		return fmt.Sprintf("%d", v)

	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)

	case string:
		// Check for pseudo-parameters that should be constants
		if pseudoConst, ok := pseudoParameterConstants[v]; ok {
			ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
			return pseudoConst
		}
		// Check for enum constants
		if enumConst := tryEnumConstant(ctx, v); enumConst != "" {
			return enumConst
		}
		return goStringLiteral(v)

	case []any:
		if len(v) == 0 {
			return "[]any{}"
		}
		// Check if array elements are maps that should be typed
		if propName != "" && len(v) > 0 {
			if _, isMap := v[0].(map[string]any); isMap {
				elemTypeName := getArrayElementTypeName(ctx, propName)
				if elemTypeName != "" {
					return arrayToBlockStyle(ctx, v, elemTypeName, parentVarName, propName)
				}
			}
		}
		// Fallback: inline array ([]any{} is plain Go, no import needed)
		var items []string
		for _, item := range v {
			items = append(items, valueToGoForBlock(ctx, item, "", parentVarName))
		}
		return fmt.Sprintf("[]any{%s}", strings.Join(items, ", "))

	case map[string]any:
		// Check if this is an intrinsic function map
		if len(v) == 1 {
			for k := range v {
				if k == "Ref" || strings.HasPrefix(k, "Fn::") || k == "Condition" {
					intrinsic := mapToIntrinsic(v)
					if intrinsic != nil {
						// If we have a property name, update the type context before processing
						// the intrinsic so that nested values use the correct type.
						// For example, S3Location inside an If should use Association_S3OutputLocation,
						// not the parent Association_InstanceAssociationOutputLocation.
						if propName != "" {
							typeName := getPropertyTypeName(ctx, propName)
							if typeName != "" {
								savedTypeName := ctx.currentTypeName
								ctx.currentTypeName = typeName
								result := intrinsicToGo(ctx, intrinsic)
								ctx.currentTypeName = savedTypeName
								return result
							}
						}
						return intrinsicToGo(ctx, intrinsic)
					}
				}
			}
		}

		// Check if this is a policy document field
		if isPolicyDocumentField(propName) {
			return policyDocToBlocks(ctx, v, parentVarName, propName)
		}

		// Check if this should be a nested property type block
		typeName := getPropertyTypeName(ctx, propName)
		if typeName != "" && allKeysValidIdentifiers(v) {
			// Create a nested block
			nestedVarName := parentVarName + propName
			fullTypeName := fmt.Sprintf("%s.%s", ctx.currentResource, typeName)

			// Check if this field is a pointer BEFORE updating type context
			// (we need to check against the parent type, not the nested type)
			needsPointer := isPointerField(ctx, propName)

			// Save and update type context for nested properties
			savedTypeName := ctx.currentTypeName
			ctx.currentTypeName = typeName

			ctx.propertyBlocks = append(ctx.propertyBlocks, propertyBlock{
				varName:    nestedVarName,
				typeName:   fullTypeName,
				properties: v,
				isPointer:  needsPointer,
				order:      len(ctx.propertyBlocks),
			})

			// Restore type context
			ctx.currentTypeName = savedTypeName

			// Pointer fields are now `any` type, so no & prefix needed
			return nestedVarName
		}

		// Fallback: inline map
		var items []string
		for _, k := range sortedKeys(v) {
			val := v[k]
			items = append(items, fmt.Sprintf("%q: %s", k, valueToGoForBlock(ctx, val, k, parentVarName)))
		}
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if len(items) == 0 {
			return "Json{}"
		}
		return fmt.Sprintf("Json{%s}", strings.Join(items, ", "))
	}

	return fmt.Sprintf("%#v", value)
}

// arrayToBlockStyle converts an array of maps to block style with separate var declarations.
func arrayToBlockStyle(ctx *codegenContext, arr []any, elemTypeName string, parentVarName string, propName string) string {
	var varNames []string

	// Save and update type context for array elements
	savedTypeName := ctx.currentTypeName
	ctx.currentTypeName = elemTypeName

	for i, item := range arr {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Generate unique var name for this element
		varName := generateArrayElementVarName(ctx, parentVarName, propName, itemMap, i)
		fullTypeName := fmt.Sprintf("%s.%s", ctx.currentResource, elemTypeName)

		ctx.propertyBlocks = append(ctx.propertyBlocks, propertyBlock{
			varName:    varName,
			typeName:   fullTypeName,
			properties: itemMap,
			isPointer:  false, // Array elements are values, not pointers
			order:      len(ctx.propertyBlocks),
		})

		varNames = append(varNames, varName)
	}

	// Restore type context
	ctx.currentTypeName = savedTypeName

	if len(varNames) == 0 {
		return fmt.Sprintf("[]%s.%s{}", ctx.currentResource, elemTypeName)
	}

	return fmt.Sprintf("[]any{%s}", strings.Join(varNames, ", "))
}

// generateArrayElementVarName generates a unique var name for an array element.
func generateArrayElementVarName(ctx *codegenContext, parentVarName string, propName string, props map[string]any, index int) string {
	// Try to find a distinguishing value
	var suffix string

	// For various types, look for identifying fields
	for _, key := range []string{"Id", "Name", "Key", "Type", "DeviceName", "PolicyName", "Status"} {
		if val, ok := props[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				suffix = cleanForVarName(s)
				break
			}
		}
	}

	// For security group rules, use port info
	if suffix == "" {
		if fromPort, ok := props["FromPort"]; ok {
			suffix = fmt.Sprintf("Port%s", cleanForVarName(fmt.Sprintf("%v", fromPort)))
			if proto, ok := props["IpProtocol"].(string); ok && proto != "tcp" {
				suffix += strings.ToUpper(cleanForVarName(proto))
			}
		}
	}

	// Fallback to index
	if suffix == "" {
		suffix = fmt.Sprintf("%d", index+1)
	}

	// Use singular form for array element names
	singularProp := singularize(propName)
	baseName := parentVarName + singularProp + suffix

	// Deduplicate: if name was already used, append an index
	ctx.blockNameCount[baseName]++
	count := ctx.blockNameCount[baseName]
	if count > 1 {
		return fmt.Sprintf("%s_%d", baseName, count)
	}
	return baseName
}

// cleanForVarName cleans a string value for use in a Go variable name.
func cleanForVarName(s string) string {
	// Handle negative numbers (e.g., "-1" -> "Neg1") before removing hyphens
	s = strings.ReplaceAll(s, "-", "Neg")

	// Remove other special chars
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ":", "")

	// If starts with a digit, prefix with N
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "N" + s
	}

	// Capitalize first letter if lowercase
	if len(s) > 0 && s[0] >= 'a' && s[0] <= 'z' {
		s = strings.ToUpper(s[:1]) + s[1:]
	}

	// Limit length
	if len(s) > 20 {
		s = s[:20]
	}

	return s
}

// tagsToBlockStyle converts tags to block style with separate var declarations.
// Tags field is []any in generated resources, so we use []any{Tag{}, Tag{}, ...}
func tagsToBlockStyle(ctx *codegenContext, value any) string {
	tags, ok := value.([]any)
	if !ok || len(tags) == 0 {
		return "[]any{}"
	}

	// Only add intrinsics import when we have tags to generate
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true

	var varNames []string
	for _, tag := range tags {
		tagMap, ok := tag.(map[string]any)
		if !ok {
			continue
		}

		key, hasKey := tagMap["Key"]
		val, hasValue := tagMap["Value"]
		if !hasKey || !hasValue {
			continue
		}

		// Generate var name from tag key
		keyStr, ok := key.(string)
		if !ok {
			continue
		}
		varName := ctx.currentLogicalID + "Tag" + cleanForVarName(keyStr)

		// Add to property blocks
		ctx.propertyBlocks = append(ctx.propertyBlocks, propertyBlock{
			varName:    varName,
			typeName:   "Tag",
			properties: map[string]any{"Key": key, "Value": val},
			isPointer:  false, // Tags are values
			order:      len(ctx.propertyBlocks),
		})

		varNames = append(varNames, varName)
	}

	if len(varNames) == 0 {
		return "[]any{}"
	}

	return fmt.Sprintf("[]any{%s}", strings.Join(varNames, ", "))
}

func generateOutput(ctx *codegenContext, output *IROutput) string {
	var lines []string

	varName := output.LogicalID + "Output"

	if output.Description != "" {
		lines = append(lines, fmt.Sprintf("// %s - %s", varName, output.Description))
	}

	// Use the Output type from intrinsics
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
	lines = append(lines, fmt.Sprintf("var %s = Output{", varName))

	value := valueToGo(ctx, output.Value, 1)
	lines = append(lines, fmt.Sprintf("\tValue:       %s,", value))

	if output.Description != "" {
		lines = append(lines, fmt.Sprintf("\tDescription: %q,", output.Description))
	}
	if output.ExportName != nil {
		exportValue := valueToGo(ctx, output.ExportName, 1)
		lines = append(lines, fmt.Sprintf("\tExportName:  %s,", exportValue))
	}
	if output.Condition != "" {
		lines = append(lines, fmt.Sprintf("\tCondition:   %q,", output.Condition))
	}

	lines = append(lines, "}")

	return strings.Join(lines, "\n")
}

// valueToGo converts an IR value to Go source code.
func valueToGo(ctx *codegenContext, value any, indent int) string {
	return valueToGoWithProperty(ctx, value, indent, "")
}

// valueToGoWithProperty converts an IR value to Go source code, with property context.
// The propName parameter indicates the property name if this value is a field in a struct,
// which allows us to determine the typed struct name for nested property types.
func valueToGoWithProperty(ctx *codegenContext, value any, indent int, propName string) string {
	indentStr := strings.Repeat("\t", indent)
	nextIndent := strings.Repeat("\t", indent+1)

	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case *IRIntrinsic:
		// If we have a property name, update the type context before processing
		// the intrinsic so that nested values use the correct type.
		// For example, S3Location inside an If should use Association_S3OutputLocation,
		// not the parent Association_InstanceAssociationOutputLocation.
		if propName != "" {
			typeName := getPropertyTypeName(ctx, propName)
			if typeName != "" {
				savedTypeName := ctx.currentTypeName
				ctx.currentTypeName = typeName
				result := intrinsicToGo(ctx, v)
				ctx.currentTypeName = savedTypeName
				return result
			}
		}
		return intrinsicToGo(ctx, v)

	case bool:
		if v {
			return "true"
		}
		return "false"

	case int:
		return fmt.Sprintf("%d", v)

	case int64:
		return fmt.Sprintf("%d", v)

	case float64:
		// Check if it's a whole number
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)

	case string:
		// Check for pseudo-parameters that should be constants
		if pseudoConst, ok := pseudoParameterConstants[v]; ok {
			ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
			return pseudoConst
		}
		// Check for enum constants
		if enumConst := tryEnumConstant(ctx, v); enumConst != "" {
			return enumConst
		}
		return goStringLiteral(v)

	case []any:
		if len(v) == 0 {
			return "[]any{}"
		}
		// Check if this is an array of objects that should use typed slice
		if propName != "" && len(v) > 0 {
			if _, isMap := v[0].(map[string]any); isMap {
				// Determine the element type name (singular form for arrays)
				elemTypeName := getArrayElementTypeName(ctx, propName)
				if elemTypeName != "" {
					// Save current type context and switch to element type for nested properties
					savedTypeName := ctx.currentTypeName
					ctx.currentTypeName = elemTypeName

					var items []string
					for _, item := range v {
						// Pass empty propName - the element IS the type, not a nested property
						items = append(items, nextIndent+valueToGoWithProperty(ctx, item, indent+1, "")+",")
					}

					// Restore type context
					ctx.currentTypeName = savedTypeName
					return fmt.Sprintf("[]%s.%s{\n%s\n%s}", ctx.currentResource, elemTypeName, strings.Join(items, "\n"), indentStr)
				}
			}
		}
		var items []string
		for _, item := range v {
			items = append(items, nextIndent+valueToGoWithProperty(ctx, item, indent+1, "")+",")
		}
		return fmt.Sprintf("[]any{\n%s\n%s}", strings.Join(items, "\n"), indentStr)

	case map[string]any:
		if len(v) == 0 {
			ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
			return "Json{}"
		}
		// Check if this is an intrinsic function map (single key starting with "Ref" or "Fn::")
		if len(v) == 1 {
			for k := range v {
				if k == "Ref" || strings.HasPrefix(k, "Fn::") || k == "Condition" {
					// Convert to IRIntrinsic and use intrinsicToGo
					intrinsic := mapToIntrinsic(v)
					if intrinsic != nil {
						return intrinsicToGo(ctx, intrinsic)
					}
				}
			}
		}

		// Try to use a typed struct based on property context
		// But only if all keys are valid Go identifiers
		typeName := getPropertyTypeName(ctx, propName)
		if typeName != "" && allKeysValidIdentifiers(v) {
			// Save current type context and switch to nested type
			savedTypeName := ctx.currentTypeName
			ctx.currentTypeName = typeName

			var items []string
			for _, k := range sortedKeys(v) {
				val := v[k]
				items = append(items, fmt.Sprintf("%s%s: %s,", nextIndent, k, valueToGoWithProperty(ctx, val, indent+1, k)))
			}

			// Restore type context
			ctx.currentTypeName = savedTypeName
			// Property type fields are now `any` type, so no & prefix needed
			return fmt.Sprintf("%s.%s{\n%s\n%s}", ctx.currentResource, typeName, strings.Join(items, "\n"), indentStr)
		}

		// Check if we're at an array element level (propName is empty but currentTypeName is a property type)
		// This happens when processing elements of a typed slice like []Bucket_Rule
		if propName == "" && strings.Contains(ctx.currentTypeName, "_") && allKeysValidIdentifiers(v) {
			var items []string
			for _, k := range sortedKeys(v) {
				val := v[k]
				items = append(items, fmt.Sprintf("%s%s: %s,", nextIndent, k, valueToGoWithProperty(ctx, val, indent+1, k)))
			}
			return fmt.Sprintf("%s.%s{\n%s\n%s}", ctx.currentResource, ctx.currentTypeName, strings.Join(items, "\n"), indentStr)
		}

		// Fallback to Json{}
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		var items []string
		for _, k := range sortedKeys(v) {
			val := v[k]
			items = append(items, fmt.Sprintf("%s%q: %s,", nextIndent, k, valueToGoWithProperty(ctx, val, indent+1, k)))
		}
		return fmt.Sprintf("Json{\n%s\n%s}", strings.Join(items, "\n"), indentStr)
	}

	return fmt.Sprintf("%#v", value)
}

// isPointerField checks if a property field expects a pointer type.
// Uses the PointerFields registry generated by codegen.
func isPointerField(ctx *codegenContext, propName string) bool {
	if propName == "" || ctx.currentTypeName == "" {
		return false
	}
	key := ctx.currentResource + "." + ctx.currentTypeName + "." + propName
	return resources.PointerFields[key]
}

// getPropertyTypeName returns the typed struct name for a property, if known.
// CloudFormation property types are always flat: {ResourceType}_{PropertyTypeName}
// e.g., Distribution_DistributionConfig, Distribution_DefaultCacheBehavior, Distribution_Cookies
// Returns empty string if the property should use map[string]any.
func getPropertyTypeName(ctx *codegenContext, propName string) string {
	if propName == "" || ctx.currentTypeName == "" {
		return ""
	}

	// Skip known fields that should remain as map[string]any or are handled specially
	skipFields := map[string]bool{
		"Tags":     true,
		"Metadata": true,
	}
	if skipFields[propName] {
		return ""
	}

	// First, check PropertyTypeMap for the exact mapping.
	// Format: "service.ResourceType.PropertyName" -> "ResourceType_ActualTypeName"
	// This handles cases where the property name differs from the type name.
	key := ctx.currentResource + "." + ctx.currentTypeName + "." + propName
	if typeName, ok := resources.PropertyTypeMap[key]; ok {
		return typeName
	}

	// CloudFormation property types are FLAT - they use the base resource type, not nested type.
	// e.g., Distribution_DistributionConfig has property Logging with type Distribution_Logging
	// NOT Distribution_DistributionConfig_Logging.
	// Extract base resource type from current type name.
	baseResourceType := ctx.currentTypeName
	if idx := strings.Index(ctx.currentTypeName, "_"); idx > 0 {
		baseResourceType = ctx.currentTypeName[:idx]
	}

	// Try flat pattern first: BaseResourceType_PropName
	flatTypeName := baseResourceType + "_" + propName
	fullName := ctx.currentResource + "." + flatTypeName
	if resources.PropertyTypes[fullName] {
		return flatTypeName
	}

	// Fallback: Try nested pattern (currentTypeName_propName) for rare cases
	nestedTypeName := ctx.currentTypeName + "_" + propName
	fullName = ctx.currentResource + "." + nestedTypeName
	if resources.PropertyTypes[fullName] {
		return nestedTypeName
	}

	// Type doesn't exist, fall back to map[string]any
	return ""
}

// getArrayElementTypeName returns the typed struct name for array elements.
// CloudFormation uses singular names for element types: Origins -> Origin
func getArrayElementTypeName(ctx *codegenContext, propName string) string {
	if propName == "" || ctx.currentTypeName == "" {
		return ""
	}

	// Skip known fields that should remain as []any
	skipFields := map[string]bool{
		"Tags": true,
	}
	if skipFields[propName] {
		return ""
	}

	// First, check PropertyTypeMap for the exact mapping.
	// Format: "service.ResourceType.PropertyName" -> "ResourceType_ActualTypeName"
	// This handles array properties where the type name differs from singular property name.
	// e.g., "s3.Bucket.AnalyticsConfigurations" -> "Bucket_AnalyticsConfiguration"
	key := ctx.currentResource + "." + ctx.currentTypeName + "." + propName
	if typeName, ok := resources.PropertyTypeMap[key]; ok {
		return typeName
	}

	singular := singularize(propName)

	// CloudFormation property types are FLAT - they use the base resource type, not nested type.
	// e.g., Distribution_DistributionConfig has property Origins with element type Distribution_Origin
	// NOT Distribution_DistributionConfig_Origin.
	// Extract base resource type from current type name.
	baseResourceType := ctx.currentTypeName
	if idx := strings.Index(ctx.currentTypeName, "_"); idx > 0 {
		baseResourceType = ctx.currentTypeName[:idx]
	}

	// Try flat pattern first: BaseResourceType_SingularPropName
	flatTypeName := baseResourceType + "_" + singular
	fullName := ctx.currentResource + "." + flatTypeName
	if resources.PropertyTypes[fullName] {
		return flatTypeName
	}

	// Fallback: Try nested pattern (currentTypeName_singular) for rare cases
	nestedTypeName := ctx.currentTypeName + "_" + singular
	fullName = ctx.currentResource + "." + nestedTypeName
	if resources.PropertyTypes[fullName] {
		return nestedTypeName
	}

	// Type doesn't exist, fall back to []any
	return ""
}

// singularize converts a plural property name to singular for element types.
// e.g., Origins -> Origin, CacheBehaviors -> CacheBehavior
func singularize(name string) string {
	// Handle common CloudFormation patterns
	if strings.HasSuffix(name, "ies") {
		// e.g., Policies -> Policy
		return name[:len(name)-3] + "y"
	}
	if strings.HasSuffix(name, "sses") {
		// e.g., Addresses -> Address (but keep one 's')
		return name[:len(name)-2]
	}
	if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
		// e.g., Origins -> Origin, but keep Addresses as Address
		return name[:len(name)-1]
	}
	return name
}

// allKeysValidIdentifiers checks if all keys in a map are valid Go identifiers.
// Returns false if any key contains special characters like ':' or starts with a number.
func allKeysValidIdentifiers(m map[string]any) bool {
	for k := range m {
		if !isValidGoIdentifier(k) {
			return false
		}
	}
	return true
}

// isValidGoIdentifier checks if a string is a valid Go identifier.
func isValidGoIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}
	// Also check for Go keywords
	return !isGoKeyword(s)
}

// mapToIntrinsic converts a map with an intrinsic key to an IRIntrinsic.
// Returns nil if the map is not a recognized intrinsic.
func resolveResourceType(cfType string) (module, typeName string) {
	parts := strings.Split(cfType, "::")
	if len(parts) != 3 || parts[0] != "AWS" {
		return "", ""
	}

	service := parts[1]
	resource := parts[2]

	// Map service name to Go module name
	module = strings.ToLower(service)

	typeName = resource

	return module, typeName
}

// topologicalSort returns resources in dependency order (dependencies first).
func topologicalSort(template *IRTemplate) []string {
	// Build dependency graph: node -> list of nodes it depends on
	deps := make(map[string][]string)
	for id := range template.Resources {
		deps[id] = nil
	}
	for source, targets := range template.ReferenceGraph {
		if _, ok := template.Resources[source]; !ok {
			continue
		}
		for _, target := range targets {
			if _, ok := template.Resources[target]; ok {
				// source depends on target
				deps[source] = append(deps[source], target)
			}
		}
	}

	// Kahn's algorithm - compute in-degree (nodes that depend on this one)
	inDegree := make(map[string]int)
	for id := range template.Resources {
		inDegree[id] = 0
	}
	// For each dependency edge, increment the in-degree of the dependent
	for id, idDeps := range deps {
		inDegree[id] = len(idDeps)
	}

	// Start with nodes that have no dependencies (in-degree 0)
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue) // Stable order

	var result []string
	processed := make(map[string]bool)

	for len(queue) > 0 {
		// Take from front
		node := queue[0]
		queue = queue[1:]

		if processed[node] {
			continue
		}
		processed[node] = true
		result = append(result, node)

		// Find nodes that depend on this node
		for id, idDeps := range deps {
			if processed[id] {
				continue
			}
			for _, dep := range idDeps {
				if dep == node {
					inDegree[id]--
					if inDegree[id] == 0 {
						queue = append(queue, id)
					}
					break
				}
			}
		}
		sort.Strings(queue)
	}

	// Handle cycles by adding remaining nodes
	for id := range template.Resources {
		if !processed[id] {
			result = append(result, id)
		}
	}

	return result
}

// sortedKeys returns sorted keys from a map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ToSnakeCase converts PascalCase to snake_case.
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// ToPascalCase converts snake_case to PascalCase.
func ToPascalCase(s string) string {
	words := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(string(word[0])))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

// SanitizeGoName ensures a name is a valid Go identifier.
// Also capitalizes the first letter to ensure the variable is exported.
func SanitizeGoName(name string) string {
	// Remove invalid characters
	var result strings.Builder
	for i, r := range name {
		if i == 0 {
			if unicode.IsLetter(r) || r == '_' {
				// Capitalize first letter for export
				result.WriteRune(unicode.ToUpper(r))
			} else if unicode.IsDigit(r) {
				// Names starting with digits need a letter prefix to be valid Go identifiers
				// Use "N" (for Number) instead of "_" to keep the variable exported
				result.WriteRune('N')
				result.WriteRune(r)
			} else {
				result.WriteRune('_')
			}
		} else {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				result.WriteRune(r)
			}
		}
	}

	s := result.String()
	if s == "" {
		return "_"
	}

	// Check for Go keywords
	if isGoKeyword(s) {
		return s + "_"
	}

	return s
}

// goKeywords and isGoKeyword are defined in parser.go

