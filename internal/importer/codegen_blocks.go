package importer

import (
	"fmt"
	"sort"
	"strings"
)

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
