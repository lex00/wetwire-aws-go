package importer

import (
	"fmt"
	"sort"
	"strings"
)

// generateResourcesByIDs generates resource declarations for specific resource IDs.
// Returns code and imports for just those resources.
func generateResourcesByIDs(ctx *codegenContext, resourceIDs []string) (string, map[string]bool) {
	// Save and reset imports for this category
	savedImports := ctx.imports
	ctx.imports = make(map[string]bool)

	var sections []string
	for _, resourceID := range resourceIDs {
		resource := ctx.template.Resources[resourceID]
		sections = append(sections, generateResource(ctx, resource))
	}

	// Capture category imports
	categoryImports := ctx.imports

	// Merge into saved imports (for cross-category reference tracking)
	for imp := range categoryImports {
		savedImports[imp] = true
	}
	ctx.imports = savedImports

	return strings.Join(sections, "\n\n"), categoryImports
}

func generateResource(ctx *codegenContext, resource *IRResource) string {
	var lines []string

	// Resolve resource type to Go module and type
	module, typeName := resolveResourceType(resource.ResourceType)
	if module == "" {
		// Generate placeholder variable for unknown resource types so they can still be referenced
		// This allows outputs and other resources to reference custom resources like Custom::*
		varName := sanitizeVarName(resource.LogicalID)
		return fmt.Sprintf("// %s is a placeholder for unknown resource type: %s\n// This allows references from outputs and other resources to compile.\nvar %s any = nil",
			varName, resource.ResourceType, varName)
	}

	// Add import
	ctx.imports[fmt.Sprintf("github.com/lex00/wetwire-aws-go/resources/%s", module)] = true

	// Set current resource context for typed property generation
	ctx.currentResource = module
	ctx.currentTypeName = typeName
	ctx.currentLogicalID = resource.LogicalID

	// Clear property blocks for this resource
	ctx.propertyBlocks = nil

	// First pass: collect top-level property blocks and resource property values
	// This populates ctx.propertyBlocks with typed property instances
	resourceProps := make(map[string]string) // GoName -> generated value (var reference or literal)
	for _, propName := range sortedKeys(resource.Properties) {
		prop := resource.Properties[propName]
		ctx.currentProperty = propName
		var value string
		if propName == "Tags" {
			value = tagsToBlockStyle(ctx, prop.Value)
		} else {
			// Check if this is a typed property
			value = valueToBlockStyleProperty(ctx, prop.Value, propName, resource.LogicalID)
		}
		resourceProps[prop.GoName] = value
	}

	// Process property blocks to generate their code
	// Blocks may add more blocks when processed, so we iterate until stable
	processedBlocks := make(map[int]string) // order -> generated code
	for {
		foundNew := false
		for i := range ctx.propertyBlocks {
			if _, done := processedBlocks[ctx.propertyBlocks[i].order]; done {
				continue
			}
			foundNew = true
			// Generate this block's code (may add more blocks to ctx.propertyBlocks)
			processedBlocks[ctx.propertyBlocks[i].order] = generatePropertyBlock(ctx, ctx.propertyBlocks[i])
		}
		if !foundNew {
			break
		}
	}

	// Output blocks in reverse order (dependencies first, deepest nesting first)
	// We use order field which increases as blocks are discovered during traversal
	// Later orders mean nested blocks, which should be output first
	sortedOrders := make([]int, 0, len(processedBlocks))
	for order := range processedBlocks {
		sortedOrders = append(sortedOrders, order)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedOrders)))

	for _, order := range sortedOrders {
		lines = append(lines, processedBlocks[order])
		lines = append(lines, "") // blank line between blocks
	}

	varName := sanitizeVarName(resource.LogicalID)

	// Build struct literal for the resource
	lines = append(lines, fmt.Sprintf("var %s = %s.%s{", varName, module, typeName))

	// Properties (in sorted order)
	for _, propName := range sortedKeys(resource.Properties) {
		prop := resource.Properties[propName]
		value := resourceProps[prop.GoName]
		lines = append(lines, fmt.Sprintf("\t%s: %s,", prop.GoName, value))
	}

	lines = append(lines, "}")

	return strings.Join(lines, "\n")
}

// detectSAMImplicitResources identifies resources that SAM auto-generates
// (like IAM roles for Lambda functions) and adds them to unknownResources.
// This allows the importer to generate valid code for outputs that reference
// these implicit resources using explicit Ref{}/GetAtt{} forms.
func detectSAMImplicitResources(ctx *codegenContext) {
	for _, res := range ctx.template.Resources {
		switch res.ResourceType {
		case "AWS::Serverless::Function":
			// SAM auto-generates {FunctionName}Role for Lambda functions
			// unless AutoPublishAlias or Role property is explicitly set
			roleLogicalID := res.LogicalID + "Role"
			ctx.unknownResources[roleLogicalID] = true
		case "AWS::Serverless::Api":
			// SAM may create implicit deployment and stages
			ctx.unknownResources[res.LogicalID+"Deployment"] = true
			ctx.unknownResources[res.LogicalID+"Stage"] = true
		case "AWS::Serverless::HttpApi":
			// Similar implicit resources for HTTP API
			ctx.unknownResources[res.LogicalID+"ApiGatewayDefaultStage"] = true
		}
	}
}

// detectInitializationCycles detects cycles in the resource dependency graph
// and marks GetAtt references that should use explicit GetAtt{} form to break cycles.
// Go doesn't allow initialization cycles in package-level variables, so we need to
// break cycles by using GetAtt{} (string literals) instead of Resource.Attr (variable references).
func detectInitializationCycles(ctx *codegenContext) {
	// Build extended dependency graph including both direct Ref and GetAtt references
	// The ReferenceGraph from parsing already contains these
	deps := ctx.template.ReferenceGraph

	// Find all cycles using DFS
	cycles := findCycles(deps)

	// For each cycle, we need to break it by marking one GetAtt reference as cyclic
	// We prefer to break GetAtt references (attribute access) over Ref (variable references)
	// because GetAtt{"Resource", "Attr"} is a cleaner way to break cycles
	for _, cycle := range cycles {
		// Find a GetAtt edge in the cycle to break
		// For simplicity, break the first edge that we can identify as a GetAtt
		for i := 0; i < len(cycle); i++ {
			source := cycle[i]
			target := cycle[(i+1)%len(cycle)]
			// Mark this as a cyclic reference
			key := source + ":" + target
			ctx.cyclicGetAttRefs[key] = true
		}
	}
}

// findCycles finds all cycles in a directed graph using DFS.
// Returns a list of cycles, where each cycle is a list of node IDs.
func findCycles(graph map[string][]string) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parent := make(map[string]string)

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				parent[neighbor] = node
				dfs(neighbor, path)
			} else if recStack[neighbor] {
				// Found a cycle - extract it
				var cycle []string
				// Find where the cycle starts in the current path
				for i, n := range path {
					if n == neighbor {
						cycle = append(cycle, path[i:]...)
						break
					}
				}
				if len(cycle) > 1 {
					cycles = append(cycles, cycle)
				}
			}
		}

		recStack[node] = false
	}

	// Run DFS from all nodes to find all cycles
	for node := range graph {
		if !visited[node] {
			dfs(node, nil)
		}
	}

	return cycles
}
