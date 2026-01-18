package importer

import (
	"fmt"
	"strings"
)

// generateParams generates parameter declarations and returns code + imports.
func generateParams(ctx *codegenContext) (string, map[string]bool) {
	imports := make(map[string]bool)
	var sections []string

	for _, logicalID := range sortedKeys(ctx.template.Parameters) {
		if !ctx.usedParameters[logicalID] {
			continue
		}
		param := ctx.template.Parameters[logicalID]
		sections = append(sections, generateParameter(ctx, param))
		imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
	}

	return strings.Join(sections, "\n\n"), imports
}

// generateOutputs generates output declarations and returns code + imports.
func generateOutputs(ctx *codegenContext) (string, map[string]bool) {
	imports := make(map[string]bool)

	var sections []string
	for _, logicalID := range sortedKeys(ctx.template.Outputs) {
		output := ctx.template.Outputs[logicalID]
		sections = append(sections, generateOutput(ctx, output))
	}

	if len(sections) == 0 {
		return "", nil
	}

	// Output type requires intrinsics import
	imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true

	return strings.Join(sections, "\n\n"), imports
}

// prescanAllForParams scans all expressions (conditions, resources, outputs) for
// parameter references and marks them as used. This must be called before
// generateParams to ensure all referenced parameters are included.
func prescanAllForParams(ctx *codegenContext) {
	// Scan conditions
	for _, condition := range ctx.template.Conditions {
		scanExprForParams(ctx, condition.Expression)
	}

	// Scan all resources
	for _, resource := range ctx.template.Resources {
		for _, prop := range resource.Properties {
			scanExprForParams(ctx, prop.Value)
		}
	}

	// Scan outputs
	for _, output := range ctx.template.Outputs {
		scanExprForParams(ctx, output.Value)
		if output.Condition != "" {
			// Condition name itself isn't a param, but scan anyway
			scanExprForParams(ctx, output.Condition)
		}
	}
}

// scanExprForParams recursively scans an expression for parameter references.
// Handles Ref intrinsics and parameter names embedded in Sub strings.
func scanExprForParams(ctx *codegenContext, expr any) {
	switch v := expr.(type) {
	case *IRIntrinsic:
		switch v.Type {
		case IntrinsicRef:
			target := fmt.Sprintf("%v", v.Args)
			if _, ok := ctx.template.Parameters[target]; ok {
				ctx.usedParameters[target] = true
			}
		case IntrinsicSub:
			// Extract parameter names from Sub string
			scanSubStringForParams(ctx, v.Args)
		}
		// Recurse into intrinsic args
		scanExprForParams(ctx, v.Args)
	case []any:
		for _, elem := range v {
			scanExprForParams(ctx, elem)
		}
	case map[string]any:
		for _, val := range v {
			scanExprForParams(ctx, val)
		}
	case string:
		// Check if string contains ${ParamName} references (shouldn't happen outside Sub, but be safe)
		scanSubStringForParams(ctx, v)
	}
}

// scanSubStringForParams extracts parameter references from a Sub template string.
// Sub strings can contain ${ParamName} or ${AWS::PseudoParam} references.
func scanSubStringForParams(ctx *codegenContext, args any) {
	var template string

	switch v := args.(type) {
	case string:
		template = v
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				template = s
			}
		}
	default:
		return
	}

	// Extract all ${...} references from the template string
	// Pattern: ${VarName} where VarName doesn't start with AWS::
	for i := 0; i < len(template); i++ {
		if i+1 < len(template) && template[i] == '$' && template[i+1] == '{' {
			// Find closing brace
			end := strings.Index(template[i:], "}")
			if end == -1 {
				break
			}
			ref := template[i+2 : i+end]

			// Skip pseudo-parameters and attribute references
			if !strings.HasPrefix(ref, "AWS::") && !strings.Contains(ref, ".") {
				// Check if this is a known parameter
				if _, ok := ctx.template.Parameters[ref]; ok {
					ctx.usedParameters[ref] = true
				}
			}
			i = i + end
		}
	}
}

// generateConditions generates condition declarations.
func generateConditions(ctx *codegenContext) string {
	var sections []string
	for _, logicalID := range sortedKeys(ctx.template.Conditions) {
		condition := ctx.template.Conditions[logicalID]
		sections = append(sections, generateCondition(ctx, condition))
	}
	return strings.Join(sections, "\n\n")
}

// generateMappings generates mapping declarations.
func generateMappings(ctx *codegenContext) string {
	var sections []string
	for _, logicalID := range sortedKeys(ctx.template.Mappings) {
		mapping := ctx.template.Mappings[logicalID]
		sections = append(sections, generateMapping(ctx, mapping))
	}
	return strings.Join(sections, "\n\n")
}

func generateParameter(ctx *codegenContext, param *IRParameter) string {
	var lines []string

	// Capitalize parameter name to ensure it's exported
	varName := sanitizeVarName(param.LogicalID)
	if param.Description != "" {
		// Wrap long descriptions to avoid multi-line comment issues
		desc := wrapComment(param.Description, 80)
		lines = append(lines, fmt.Sprintf("// %s - %s", varName, desc))
	}

	// Generate full Parameter{} struct with all metadata
	ctx.imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
	lines = append(lines, fmt.Sprintf("var %s = Parameter{", varName))

	// Type is required
	paramType := param.Type
	if paramType == "" {
		paramType = "String"
	}
	lines = append(lines, fmt.Sprintf("\tType: %q,", paramType))

	if param.Description != "" {
		lines = append(lines, fmt.Sprintf("\tDescription: %q,", param.Description))
	}
	if param.Default != nil {
		defaultVal := valueToGo(ctx, param.Default, 1)
		lines = append(lines, fmt.Sprintf("\tDefault: %s,", defaultVal))
	}
	if len(param.AllowedValues) > 0 {
		var vals []string
		for _, v := range param.AllowedValues {
			vals = append(vals, valueToGo(ctx, v, 0))
		}
		lines = append(lines, fmt.Sprintf("\tAllowedValues: []any{%s},", strings.Join(vals, ", ")))
	}
	if param.AllowedPattern != "" {
		lines = append(lines, fmt.Sprintf("\tAllowedPattern: %q,", param.AllowedPattern))
	}
	if param.ConstraintDescription != "" {
		lines = append(lines, fmt.Sprintf("\tConstraintDescription: %q,", param.ConstraintDescription))
	}
	if param.MinLength != nil {
		lines = append(lines, fmt.Sprintf("\tMinLength: IntPtr(%d),", *param.MinLength))
	}
	if param.MaxLength != nil {
		lines = append(lines, fmt.Sprintf("\tMaxLength: IntPtr(%d),", *param.MaxLength))
	}
	if param.MinValue != nil {
		lines = append(lines, fmt.Sprintf("\tMinValue: Float64Ptr(%g),", *param.MinValue))
	}
	if param.MaxValue != nil {
		lines = append(lines, fmt.Sprintf("\tMaxValue: Float64Ptr(%g),", *param.MaxValue))
	}
	if param.NoEcho {
		lines = append(lines, "\tNoEcho: true,")
	}

	lines = append(lines, "}")

	return strings.Join(lines, "\n")
}

// wrapComment truncates or wraps a comment to fit on a single line.
func wrapComment(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// Collapse multiple spaces
	s = strings.Join(strings.Fields(s), " ")
	// Truncate if too long
	if len(s) > maxLen {
		s = s[:maxLen-3] + "..."
	}
	return s
}

func generateMapping(ctx *codegenContext, mapping *IRMapping) string {
	varName := mapping.LogicalID + "Mapping"
	value := valueToGo(ctx, mapping.MapData, 0)
	return fmt.Sprintf("var %s = %s", varName, value)
}

func generateCondition(ctx *codegenContext, condition *IRCondition) string {
	varName := SanitizeGoName(condition.LogicalID) + "Condition"
	value := valueToGo(ctx, condition.Expression, 0)
	return fmt.Sprintf("var %s = %s", varName, value)
}
