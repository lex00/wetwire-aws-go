package importer

import (
	"fmt"
	"strings"
)

func mapToIntrinsic(m map[string]any) *IRIntrinsic {
	if len(m) != 1 {
		return nil
	}

	for k, v := range m {
		var intrinsicType IntrinsicType
		switch k {
		case "Ref":
			intrinsicType = IntrinsicRef
		case "Fn::GetAtt":
			intrinsicType = IntrinsicGetAtt
		case "Fn::Sub":
			intrinsicType = IntrinsicSub
		case "Fn::Join":
			intrinsicType = IntrinsicJoin
		case "Fn::Select":
			intrinsicType = IntrinsicSelect
		case "Fn::GetAZs":
			intrinsicType = IntrinsicGetAZs
		case "Fn::If":
			intrinsicType = IntrinsicIf
		case "Fn::Equals":
			intrinsicType = IntrinsicEquals
		case "Fn::And":
			intrinsicType = IntrinsicAnd
		case "Fn::Or":
			intrinsicType = IntrinsicOr
		case "Fn::Not":
			intrinsicType = IntrinsicNot
		case "Fn::Base64":
			intrinsicType = IntrinsicBase64
		case "Fn::FindInMap":
			intrinsicType = IntrinsicFindInMap
		case "Fn::Cidr":
			intrinsicType = IntrinsicCidr
		case "Fn::ImportValue":
			intrinsicType = IntrinsicImportValue
		case "Fn::Split":
			intrinsicType = IntrinsicSplit
		case "Fn::Transform":
			intrinsicType = IntrinsicTransform
		case "Condition":
			intrinsicType = IntrinsicCondition
		default:
			return nil
		}
		return &IRIntrinsic{Type: intrinsicType, Args: v}
	}
	return nil
}

// simplifySubString analyzes a Sub template string and returns the simplest representation:
// - If it's just "${VarName}" with no other text, returns the variable directly
// - If it's just "${AWS::Region}" etc., returns the pseudo-parameter constant
// - Otherwise returns Sub{...} with positional syntax
func simplifySubString(ctx *codegenContext, s string) string {
	// Pattern for single variable reference: ${VarName}
	// Match strings that are ONLY a single ${...} with no other text
	if len(s) > 3 && s[0] == '$' && s[1] == '{' && s[len(s)-1] == '}' {
		inner := s[2 : len(s)-1]
		// Check no nested ${} or additional ${}
		if !strings.Contains(inner, "${") && !strings.Contains(inner, "}") {
			// It's a single reference like ${VarName} or ${AWS::Region}
			if strings.HasPrefix(inner, "AWS::") {
				// Pseudo-parameter: return constant
				return pseudoParameterToGo(ctx, inner)
			}
			// Check for GetAtt pattern: ${Resource.Attribute}
			// In Sub templates, ${Resource.Attr} is shorthand for !GetAtt Resource.Attr
			if parts := strings.SplitN(inner, ".", 2); len(parts) == 2 {
				logicalID, attr := parts[0], parts[1]
				// Check if the first part is a known resource
				if _, ok := ctx.template.Resources[logicalID]; ok {
					// Generate field access pattern: Resource.Attr
					return fmt.Sprintf("%s.%s", sanitizeVarName(logicalID), attr)
				}
			}
			// Regular variable reference
			// Check if it's a known resource or parameter
			if _, ok := ctx.template.Resources[inner]; ok {
				return SanitizeGoName(inner)
			}
			if _, ok := ctx.template.Parameters[inner]; ok {
				return SanitizeGoName(inner)
			}
			// Unknown reference - still emit as variable (cross-file)
			return SanitizeGoName(inner)
		}
	}
	// Not a simple reference - use Sub{} with keyed syntax (to satisfy go vet)
	return fmt.Sprintf("Sub{String: %q}", s)
}

// intrinsicToGo converts an IRIntrinsic to Go source code.
// Uses function call syntax for cleaner generated code:
//
//	Sub{...} with positional syntax for template strings
//	Select(0, GetAZs()) instead of intrinsics.Select{Index: 0, List: intrinsics.GetAZs{}}
func intrinsicToGo(ctx *codegenContext, intrinsic *IRIntrinsic) string {
	// Note: We only add intrinsics import when we actually emit an intrinsic type.
	// Ref/GetAtt to known resources/parameters use bare identifiers, no import needed.

	switch intrinsic.Type {
	case IntrinsicRef:
		target := fmt.Sprintf("%v", intrinsic.Args)
		// Check if it's a pseudo-parameter
		if strings.HasPrefix(target, "AWS::") {
			return pseudoParameterToGo(ctx, target)
		}
		// Check if it's a known resource - use sanitized name (no-parens pattern)
		if _, ok := ctx.template.Resources[target]; ok {
			return sanitizeVarName(target)
		}
		// Check if it's a parameter - use sanitized name and track usage
		if _, ok := ctx.template.Parameters[target]; ok {
			ctx.usedParameters[target] = true
			return sanitizeVarName(target)
		}
		// Check if it's a known SAM implicit resource - use explicit Ref{}
		if ctx.unknownResources[target] {
			ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
			return fmt.Sprintf("Ref{%q}", target)
		}
		// Unknown reference - use explicit Ref{} form to avoid undefined variable errors
		// This ensures the code compiles even if the reference target doesn't exist
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		return fmt.Sprintf("Ref{%q}", target)

	case IntrinsicGetAtt:
		var logicalID, attr string
		switch args := intrinsic.Args.(type) {
		case []string:
			if len(args) >= 2 {
				logicalID = args[0]
				attr = args[1]
			}
		case []any:
			if len(args) >= 2 {
				logicalID = fmt.Sprintf("%v", args[0])
				attr = fmt.Sprintf("%v", args[1])
			}
		}
		// Check for nested attributes (e.g., "Endpoint.Address"), unknown resources,
		// or resources not in the template (SAM implicit resources, etc.)
		// These can't use field access pattern:
		// - Nested attributes: AttrRef doesn't have sub-fields
		// - Unknown resources: placeholder is `any` type with no fields
		// - Missing resources: would cause undefined variable error
		// - Cyclic references: would cause Go initialization cycle
		_, isKnownResource := ctx.template.Resources[logicalID]
		cycleKey := ctx.currentLogicalID + ":" + logicalID
		isCyclicRef := ctx.cyclicGetAttRefs[cycleKey]
		if strings.Contains(attr, ".") || ctx.unknownResources[logicalID] || !isKnownResource || isCyclicRef {
			ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
			// Use string literal for logical ID since GetAtt.LogicalName expects string
			return fmt.Sprintf("GetAtt{%q, %q}", logicalID, attr)
		}
		// Use attribute access pattern - Resource.Attr
		// This avoids generating GetAtt{} which violates style guidelines
		return fmt.Sprintf("%s.%s", sanitizeVarName(logicalID), attr)

	case IntrinsicSub:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		switch args := intrinsic.Args.(type) {
		case string:
			return simplifySubString(ctx, args)
		case []any:
			if len(args) >= 2 {
				template := fmt.Sprintf("%v", args[0])
				// Clear type context for Variables - it should always be Json{}, not a struct type
				savedTypeName := ctx.currentTypeName
				ctx.currentTypeName = ""
				vars := valueToGo(ctx, args[1], 0)
				ctx.currentTypeName = savedTypeName
				return fmt.Sprintf("SubWithMap{String: %q, Variables: %s}", template, vars)
			} else if len(args) == 1 {
				template := fmt.Sprintf("%v", args[0])
				return simplifySubString(ctx, template)
			}
		}
		return `Sub{String: ""}`

	case IntrinsicJoin:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok && len(args) >= 2 {
			delimiter := valueToGo(ctx, args[0], 0)
			// Check if the second argument is an intrinsic that needs wrapping
			// Join.Values expects []any, but intrinsics like Ref resolve to bare var names
			var values string
			if valIntrinsic, ok := args[1].(*IRIntrinsic); ok {
				// The intrinsic resolves to a variable name, wrap in []any{}
				innerVal := intrinsicToGo(ctx, valIntrinsic)
				values = fmt.Sprintf("[]any{%s}", innerVal)
			} else {
				values = valueToGo(ctx, args[1], 0)
			}
			return fmt.Sprintf("Join{Delimiter: %s, Values: %s}", delimiter, values)
		}
		return `Join{Delimiter: "", Values: nil}`

	case IntrinsicSelect:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok && len(args) >= 2 {
			// Convert index to int (may come as string "0" or float64 0)
			var indexInt int
			switch idx := args[0].(type) {
			case float64:
				indexInt = int(idx)
			case int:
				indexInt = idx
			case string:
				_, _ = fmt.Sscanf(idx, "%d", &indexInt)
			}
			list := valueToGo(ctx, args[1], 0)
			return fmt.Sprintf("Select{Index: %d, List: %s}", indexInt, list)
		}
		return "Select{Index: 0, List: nil}"

	case IntrinsicGetAZs:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if intrinsic.Args == nil || intrinsic.Args == "" {
			return "GetAZs{}"
		}
		// Special case: GetAZs with !Ref "AWS::Region" should use empty string
		// GetAZs.Region is a string field, not any, so we can't use AWS_REGION (Ref type)
		// Empty string in GetAZs means "current region" which is the same as AWS::Region
		if nested, ok := intrinsic.Args.(*IRIntrinsic); ok {
			if nested.Type == IntrinsicRef {
				if refName, ok := nested.Args.(string); ok && refName == "AWS::Region" {
					return "GetAZs{}"
				}
			}
		}
		// For literal string regions, use them directly
		if regionStr, ok := intrinsic.Args.(string); ok {
			return fmt.Sprintf("GetAZs{Region: %q}", regionStr)
		}
		// Fallback for other cases - use empty string (safest)
		return "GetAZs{}"

	case IntrinsicIf:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok && len(args) >= 3 {
			condName := fmt.Sprintf("%v", args[0])
			trueVal := valueToGo(ctx, args[1], 0)
			falseVal := valueToGo(ctx, args[2], 0)
			return fmt.Sprintf("If{%q, %s, %s}", condName, trueVal, falseVal)
		}
		return `If{"", nil, nil}`

	case IntrinsicEquals:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok && len(args) >= 2 {
			val1 := valueToGo(ctx, args[0], 0)
			val2 := valueToGo(ctx, args[1], 0)
			return fmt.Sprintf("Equals{%s, %s}", val1, val2)
		}
		return "Equals{nil, nil}"

	case IntrinsicAnd:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok {
			values := valueToGo(ctx, args, 0)
			return fmt.Sprintf("And{%s}", values)
		}
		return "And{nil}"

	case IntrinsicOr:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok {
			values := valueToGo(ctx, args, 0)
			return fmt.Sprintf("Or{%s}", values)
		}
		return "Or{nil}"

	case IntrinsicNot:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		condition := valueToGo(ctx, intrinsic.Args, 0)
		return fmt.Sprintf("Not{%s}", condition)

	case IntrinsicCondition:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		condName := fmt.Sprintf("%v", intrinsic.Args)
		return fmt.Sprintf("Condition{%q}", condName)

	case IntrinsicFindInMap:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok && len(args) >= 3 {
			mapName := fmt.Sprintf("%v", args[0])
			topKey := valueToGo(ctx, args[1], 0)
			secondKey := valueToGo(ctx, args[2], 0)
			return fmt.Sprintf("FindInMap{%q, %s, %s}", mapName, topKey, secondKey)
		}
		return `FindInMap{"", nil, nil}`

	case IntrinsicBase64:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		value := valueToGo(ctx, intrinsic.Args, 0)
		return fmt.Sprintf("Base64{%s}", value)

	case IntrinsicCidr:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok && len(args) >= 3 {
			ipBlock := valueToGo(ctx, args[0], 0)
			count := valueToGo(ctx, args[1], 0)
			cidrBits := valueToGo(ctx, args[2], 0)
			return fmt.Sprintf("Cidr{%s, %s, %s}", ipBlock, count, cidrBits)
		}
		return "Cidr{nil, nil, nil}"

	case IntrinsicImportValue:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		value := valueToGo(ctx, intrinsic.Args, 0)
		return fmt.Sprintf("ImportValue{%s}", value)

	case IntrinsicSplit:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		if args, ok := intrinsic.Args.([]any); ok && len(args) >= 2 {
			delimiter := valueToGo(ctx, args[0], 0)
			source := valueToGo(ctx, args[1], 0)
			return fmt.Sprintf("Split{%s, %s}", delimiter, source)
		}
		return `Split{"", nil}`

	case IntrinsicTransform:
		ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
		// Transform expects Name and Parameters fields
		// The args can be either a map or a list containing a map
		var transformMap map[string]any
		if args, ok := intrinsic.Args.(map[string]any); ok {
			transformMap = args
		} else if args, ok := intrinsic.Args.([]any); ok && len(args) > 0 {
			// Handle list format: [{Name: ..., Parameters: ...}]
			if firstArg, ok := args[0].(map[string]any); ok {
				transformMap = firstArg
			}
		}
		if transformMap != nil {
			name := ""
			if n, ok := transformMap["Name"].(string); ok {
				name = n
			}
			params := "nil"
			if p, ok := transformMap["Parameters"]; ok {
				params = valueToGo(ctx, p, 0)
			}
			return fmt.Sprintf("Transform{Name: %q, Parameters: %s}", name, params)
		}
		// Fallback for unexpected format
		value := valueToGo(ctx, intrinsic.Args, 0)
		return fmt.Sprintf("Transform{Name: \"\", Parameters: %s}", value)
	}

	return fmt.Sprintf("/* unknown intrinsic: %s */nil", intrinsic.Type)
}

// pseudoParameterToGo converts an AWS pseudo-parameter to Go.
// Uses dot import, so no intrinsics. prefix needed.
func pseudoParameterToGo(ctx *codegenContext, name string) string {
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
	switch name {
	case "AWS::Region":
		return "AWS_REGION"
	case "AWS::AccountId":
		return "AWS_ACCOUNT_ID"
	case "AWS::StackName":
		return "AWS_STACK_NAME"
	case "AWS::StackId":
		return "AWS_STACK_ID"
	case "AWS::Partition":
		return "AWS_PARTITION"
	case "AWS::URLSuffix":
		return "AWS_URL_SUFFIX"
	case "AWS::NoValue":
		return "AWS_NO_VALUE"
	case "AWS::NotificationARNs":
		return "AWS_NOTIFICATION_ARNS"
	default:
		// Unknown pseudo-parameter - use bare name (likely a parameter or resource)
		// This avoids generating Ref{} which violates style guidelines
		return name
	}
}

// resolveResourceType converts a CloudFormation resource type to Go module and type name.
// e.g., "AWS::S3::Bucket" -> ("s3", "Bucket")
