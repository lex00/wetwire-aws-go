package importer

import (
	"fmt"
	"strings"
)

// --- Policy Document Flattening ---

// policyDocToBlocks converts a policy document map to typed structs.
// Creates PolicyDocument and PolicyStatement blocks, returns the PolicyDocument var name.
func policyDocToBlocks(ctx *codegenContext, doc map[string]any, parentVarName string, propName string) string {
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true

	// Extract version (default "2012-10-17")
	version := "2012-10-17"
	if v, ok := doc["Version"].(string); ok {
		version = v
	}

	// Extract statements
	statements, _ := doc["Statement"].([]any)
	var statementVarNames []string

	for i, stmt := range statements {
		stmtMap, ok := stmt.(map[string]any)
		if !ok {
			continue
		}

		// Generate var name for this statement
		varName := fmt.Sprintf("%s%sStatement%d", parentVarName, propName, i)

		// Create statement block
		statementToBlock(ctx, stmtMap, varName)
		statementVarNames = append(statementVarNames, varName)
	}

	// Create PolicyDocument block
	policyVarName := parentVarName + propName
	policyProps := map[string]any{
		"Version":   version,
		"Statement": statementVarNames, // Will be handled specially
	}

	ctx.propertyBlocks = append(ctx.propertyBlocks, propertyBlock{
		varName:    policyVarName,
		typeName:   "PolicyDocument",
		properties: policyProps,
		isPointer:  false,
		order:      len(ctx.propertyBlocks),
	})

	return policyVarName
}

// statementToBlock creates a PolicyStatement property block.
func statementToBlock(ctx *codegenContext, stmt map[string]any, varName string) {
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true

	// Determine if this is a Deny statement
	effect, _ := stmt["Effect"].(string)
	typeName := "PolicyStatement"
	if effect == "Deny" {
		typeName = "DenyStatement"
	}

	// Convert the statement properties
	props := make(map[string]any)

	// Copy fields, transforming Principal and Condition
	for k, v := range stmt {
		switch k {
		case "Effect":
			// Skip Effect for DenyStatement (it's implicit)
			if typeName != "DenyStatement" {
				props[k] = v
			}
		case "Principal":
			props[k] = v // Will be transformed during generation
		case "Condition":
			props[k] = v // Will be transformed during generation
		default:
			props[k] = v
		}
	}

	ctx.propertyBlocks = append(ctx.propertyBlocks, propertyBlock{
		varName:    varName,
		typeName:   typeName,
		properties: props,
		isPointer:  false,
		order:      len(ctx.propertyBlocks),
	})
}

// principalToGo converts a Principal value to typed Go code.
// Converts {"Service": [...]} to ServicePrincipal{...}, etc.
func principalToGo(ctx *codegenContext, value any) string {
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true

	// Handle string principal (like "*")
	if s, ok := value.(string); ok {
		if s == "*" {
			return `"*"`
		}
		return fmt.Sprintf("%q", s)
	}

	// Handle map principal
	m, ok := value.(map[string]any)
	if !ok {
		return valueToGo(ctx, value, 0)
	}

	// Check for known principal types
	if service, ok := m["Service"]; ok {
		return principalTypeToGo(ctx, "ServicePrincipal", service)
	}
	if aws, ok := m["AWS"]; ok {
		return principalTypeToGo(ctx, "AWSPrincipal", aws)
	}
	if federated, ok := m["Federated"]; ok {
		return principalTypeToGo(ctx, "FederatedPrincipal", federated)
	}

	// Unknown principal format, fall back to Json
	return jsonMapToGo(ctx, m)
}

// principalTypeToGo converts a principal value to a typed principal.
func principalTypeToGo(ctx *codegenContext, typeName string, value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%s{%q}", typeName, v)
	case []any:
		var items []string
		for _, item := range v {
			items = append(items, valueToGo(ctx, item, 0))
		}
		return fmt.Sprintf("%s{%s}", typeName, strings.Join(items, ", "))
	default:
		return fmt.Sprintf("%s{%s}", typeName, valueToGo(ctx, value, 0))
	}
}

// conditionToGo converts a Condition map to Go code with typed operators.
func conditionToGo(ctx *codegenContext, value any) string {
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true

	m, ok := value.(map[string]any)
	if !ok {
		return valueToGo(ctx, value, 0)
	}

	var items []string
	for _, k := range sortedKeys(m) {
		v := m[k]
		// Use constant name if it's a known operator
		var keyStr string
		if constName, ok := conditionOperators[k]; ok {
			keyStr = constName
		} else {
			keyStr = fmt.Sprintf("%q", k)
		}
		valStr := jsonMapToGo(ctx, v)
		items = append(items, fmt.Sprintf("%s: %s", keyStr, valStr))
	}

	return fmt.Sprintf("Json{%s}", strings.Join(items, ", "))
}

// jsonMapToGo converts a map to Json{...} syntax.
func jsonMapToGo(ctx *codegenContext, value any) string {
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true

	switch v := value.(type) {
	case map[string]any:
		var items []string
		for _, k := range sortedKeys(v) {
			val := v[k]
			items = append(items, fmt.Sprintf("%q: %s", k, jsonValueToGo(ctx, val)))
		}
		if len(items) == 0 {
			return "Json{}"
		}
		return fmt.Sprintf("Json{%s}", strings.Join(items, ", "))
	default:
		return valueToGo(ctx, value, 0)
	}
}

// jsonValueToGo converts a value for use inside Json{}.
func jsonValueToGo(ctx *codegenContext, value any) string {
	switch v := value.(type) {
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
		return goStringLiteral(v)
	case []any:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		var items []string
		for _, item := range v {
			items = append(items, jsonValueToGo(ctx, item))
		}
		return fmt.Sprintf("[]any{%s}", strings.Join(items, ", "))
	case map[string]any:
		return jsonMapToGo(ctx, v)
	default:
		return valueToGo(ctx, value, 0)
	}
}
