package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// generateRegistry creates resources/registry.go with all property type names and mappings.
// This allows the importer to check if a type exists and look up the correct type for a property.
func generateRegistry(services []*Service, outputDir string, dryRun bool) error {
	// PropertyTypes: map of all property type names for existence check.
	// The map key in svc.PropertyTypes is already qualified (e.g., "Bucket_PublicAccessBlockConfiguration").
	var typeEntries []string
	for _, svc := range services {
		for qualifiedKey := range svc.PropertyTypes {
			// Format: "service.Resource_PropertyType"
			fullName := fmt.Sprintf("%s.%s", svc.Name, qualifiedKey)
			typeEntries = append(typeEntries, fullName)
		}
	}
	sort.Strings(typeEntries)

	// PropertyTypeMap: maps property paths to their actual type names
	// Format: "service.Resource.PropertyName" -> "Resource_ActualTypeName"
	// Format: "service.Resource_Type.PropertyName" -> "Resource_ActualTypeName"
	propMap := make(map[string]string)

	for _, svc := range services {
		// Build sorted list of property type keys for deterministic ordering.
		sortedTypeKeys := make([]string, 0, len(svc.PropertyTypes))
		for k := range svc.PropertyTypes {
			sortedTypeKeys = append(sortedTypeKeys, k)
		}
		sort.Strings(sortedTypeKeys)

		// Build qualifiedNames map with deterministic ordering (alphabetically first wins).
		// This matches the logic in generateResourcePropertyTypes.
		qualifiedNames := make(map[string]string)
		for _, qualifiedKey := range sortedTypeKeys {
			propType := svc.PropertyTypes[qualifiedKey]
			if _, exists := qualifiedNames[propType.Name]; !exists {
				qualifiedNames[propType.Name] = qualifiedKey
			}
			qualifiedNames[qualifiedKey] = qualifiedKey
		}

		// resolveTypeName resolves a short type name to its qualified form.
		resolveTypeName := func(shortName string) string {
			if qn, ok := qualifiedNames[shortName]; ok {
				return qn
			}
			return shortName
		}

		// Map resource properties to their types.
		// For resources, we use parent preference (matching generateResource logic).
		for resName, res := range svc.Resources {
			// Build propTypeNames with parent preference for this resource
			// (matching the logic in generateResource at lines 180-196)
			propTypeNames := make(map[string]string)
			for _, qualifiedKey := range sortedTypeKeys {
				pt := svc.PropertyTypes[qualifiedKey]
				// Prefer property types from the same parent resource
				if pt.ParentResource == resName {
					propTypeNames[pt.Name] = qualifiedKey
				} else if _, exists := propTypeNames[pt.Name]; !exists {
					propTypeNames[pt.Name] = qualifiedKey
				}
			}

			resolveForResource := func(shortName string) string {
				if qn, ok := propTypeNames[shortName]; ok {
					return qn
				}
				return shortName
			}

			for propName, prop := range res.Properties {
				// Skip primitives and any
				if prop.GoType == "any" || prop.GoType == "" {
					continue
				}

				var typeName string
				if prop.IsList && prop.ItemType != "" && !isPrimitive(prop.ItemType) {
					// List of property types: []SomeType
					typeName = resolveForResource(prop.ItemType)
				} else if prop.IsMap && prop.ItemType != "" && !isPrimitive(prop.ItemType) {
					// Map of property types: map[string]SomeType
					typeName = resolveForResource(prop.ItemType)
				} else if !prop.IsList && !prop.IsMap && !isPrimitive(prop.GoType) {
					// Direct property type reference
					typeName = resolveForResource(prop.GoType)
				}

				if typeName != "" {
					// Check if this type exists
					fullTypeName := svc.Name + "." + typeName
					exists := false
					for _, e := range typeEntries {
						if e == fullTypeName {
							exists = true
							break
						}
					}
					if exists {
						key := fmt.Sprintf("%s.%s.%s", svc.Name, resName, propName)
						propMap[key] = typeName
					}
				}
			}
		}

		// Map property type properties to their types.
		// The map key is already qualified (e.g., "Bucket_PublicAccessBlockConfiguration").
		for qualifiedKey, pt := range svc.PropertyTypes {
			// qualifiedKey is already the qualified type name
			parentTypeName := qualifiedKey
			for propName, prop := range pt.Properties {
				// Skip primitives and any (but not []any - list properties)
				if prop.GoType == "any" || prop.GoType == "" {
					continue
				}

				var typeName string
				if prop.IsList && prop.ItemType != "" && !isPrimitive(prop.ItemType) {
					// List of property types
					typeName = resolveTypeName(prop.ItemType)
				} else if prop.IsMap && prop.ItemType != "" && !isPrimitive(prop.ItemType) {
					// Map of property types
					typeName = resolveTypeName(prop.ItemType)
				} else if !prop.IsList && !prop.IsMap && !isPrimitive(prop.GoType) {
					// Direct property type reference
					typeName = resolveTypeName(prop.GoType)
				}

				if typeName != "" {
					// Check if this type exists
					fullTypeName := svc.Name + "." + typeName
					exists := false
					for _, e := range typeEntries {
						if e == fullTypeName {
							exists = true
							break
						}
					}
					if exists {
						key := fmt.Sprintf("%s.%s.%s", svc.Name, parentTypeName, propName)
						propMap[key] = typeName
					}
				}
			}
		}
	}

	// Sort property map keys
	propMapKeys := make([]string, 0, len(propMap))
	for k := range propMap {
		propMapKeys = append(propMapKeys, k)
	}
	sort.Strings(propMapKeys)

	// PointerFields: tracks which fields expect pointer types
	// Format: "service.ParentType.PropertyName" -> true if pointer
	pointerFields := make(map[string]bool)

	for _, svc := range services {
		// Check resource properties
		for resName, res := range svc.Resources {
			for propName, prop := range res.Properties {
				if prop.IsPointer {
					key := fmt.Sprintf("%s.%s.%s", svc.Name, resName, propName)
					pointerFields[key] = true
				}
			}
		}

		// Check property type properties
		for qualifiedKey, pt := range svc.PropertyTypes {
			for propName, prop := range pt.Properties {
				if prop.IsPointer {
					key := fmt.Sprintf("%s.%s.%s", svc.Name, qualifiedKey, propName)
					pointerFields[key] = true
				}
			}
		}
	}

	// Sort pointer fields keys
	pointerFieldKeys := make([]string, 0, len(pointerFields))
	for k := range pointerFields {
		pointerFieldKeys = append(pointerFieldKeys, k)
	}
	sort.Strings(pointerFieldKeys)

	// Generate the registry file content
	var buf []byte
	buf = append(buf, "// Code generated by wetwire-aws codegen. DO NOT EDIT.\n\n"...)
	buf = append(buf, "package resources\n\n"...)

	buf = append(buf, "// PropertyTypes lists all generated property type names.\n"...)
	buf = append(buf, "// Used by the importer to check if a type exists.\n"...)
	buf = append(buf, "var PropertyTypes = map[string]bool{\n"...)
	for _, entry := range typeEntries {
		buf = append(buf, fmt.Sprintf("\t%q: true,\n", entry)...)
	}
	buf = append(buf, "}\n\n"...)

	buf = append(buf, "// PropertyTypeMap maps property paths to their actual type names.\n"...)
	buf = append(buf, "// Format: \"service.ParentType.PropertyName\" -> \"ParentResource_ActualTypeName\"\n"...)
	buf = append(buf, "// Used by the importer to look up the correct type for a property.\n"...)
	buf = append(buf, "var PropertyTypeMap = map[string]string{\n"...)
	for _, key := range propMapKeys {
		buf = append(buf, fmt.Sprintf("\t%q: %q,\n", key, propMap[key])...)
	}
	buf = append(buf, "}\n\n"...)

	buf = append(buf, "// PointerFields tracks which property fields expect pointer types.\n"...)
	buf = append(buf, "// Format: \"service.ParentType.PropertyName\" -> true if pointer\n"...)
	buf = append(buf, "// Used by the importer to determine if a block should be a pointer.\n"...)
	buf = append(buf, "var PointerFields = map[string]bool{\n"...)
	for _, key := range pointerFieldKeys {
		buf = append(buf, fmt.Sprintf("\t%q: true,\n", key)...)
	}
	buf = append(buf, "}\n"...)

	// Write the file
	resourcesDir := filepath.Join(outputDir, "resources")
	registryFile := filepath.Join(resourcesDir, "registry.go")

	if dryRun {
		fmt.Printf("Would write: %s (%d property types, %d mappings)\n", registryFile, len(typeEntries), len(propMap))
		return nil
	}

	// Ensure resources directory exists
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		return fmt.Errorf("creating resources directory: %w", err)
	}

	if err := os.WriteFile(registryFile, buf, 0644); err != nil {
		return fmt.Errorf("writing registry file: %w", err)
	}

	fmt.Printf("Generated registry with %d property types, %d mappings: %s\n", len(typeEntries), len(propMap), registryFile)
	return nil
}

// isPrimitive checks if a type is a Go primitive (not a property type reference).
func isPrimitive(t string) bool {
	primitives := map[string]bool{
		"string": true, "int": true, "int64": true, "float64": true,
		"bool": true, "any": true, "": true,
	}
	return primitives[t]
}
