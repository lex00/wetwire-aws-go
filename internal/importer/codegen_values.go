package importer

import (
	"fmt"
	"strings"
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