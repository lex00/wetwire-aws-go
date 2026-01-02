package main

import (
	"sort"
	"strings"

	"github.com/lex00/cloudformation-schema-go/spec"
)

// Service represents a group of resources for one AWS service.
type Service struct {
	Name          string                        // e.g., "s3"
	CFPrefix      string                        // e.g., "AWS::S3"
	Resources     map[string]ParsedResource     // ResourceName -> Definition
	PropertyTypes map[string]ParsedPropertyType // PropertyTypeName -> Definition
}

// ParsedPropertyType is a parsed property type (nested struct).
type ParsedPropertyType struct {
	Name           string // e.g., "Ingress"
	CFType         string // e.g., "AWS::EC2::SecurityGroup.Ingress"
	ParentResource string // e.g., "SecurityGroup" (the resource this belongs to)
	Documentation  string
	Properties     map[string]ParsedProperty // The properties of this type
}

// ParsedResource is a parsed resource type.
type ParsedResource struct {
	Name          string // e.g., "Bucket"
	CFType        string // e.g., "AWS::S3::Bucket"
	Documentation string
	Properties    map[string]ParsedProperty
	Attributes    map[string]ParsedAttribute
}

// ParsedProperty is a parsed property.
type ParsedProperty struct {
	Name          string
	GoType        string
	CFType        string
	Documentation string
	Required      bool
	IsPointer     bool
	IsList        bool
	IsMap         bool
	ItemType      string
}

// ParsedAttribute is a parsed resource attribute (for GetAtt).
type ParsedAttribute struct {
	Name   string
	GoType string
}

// parseSpec organizes the CloudFormation spec by service.
func parseSpec(cfnSpec *spec.Spec, filterService string) []*Service {
	services := make(map[string]*Service)

	// Parse resource types
	for cfType, resDef := range cfnSpec.ResourceTypes {
		// Parse AWS::S3::Bucket -> service=s3, name=Bucket
		parts := strings.Split(cfType, "::")
		if len(parts) != 3 || parts[0] != "AWS" {
			continue
		}

		serviceName := strings.ToLower(parts[1])
		resourceName := parts[2]

		// Filter by service if specified
		if filterService != "" && serviceName != filterService {
			continue
		}

		// Get or create service
		svc, ok := services[serviceName]
		if !ok {
			svc = &Service{
				Name:          serviceName,
				CFPrefix:      "AWS::" + parts[1],
				Resources:     make(map[string]ParsedResource),
				PropertyTypes: make(map[string]ParsedPropertyType),
			}
			services[serviceName] = svc
		}

		// Parse resource
		resource := ParsedResource{
			Name:          resourceName,
			CFType:        cfType,
			Documentation: resDef.Documentation,
			Properties:    make(map[string]ParsedProperty),
			Attributes:    make(map[string]ParsedAttribute),
		}

		// Parse properties
		for propName, propDef := range resDef.Properties {
			resource.Properties[propName] = parseProperty(propName, propDef)
		}

		// Parse attributes
		for attrName, attrDef := range resDef.Attributes {
			resource.Attributes[attrName] = ParsedAttribute{
				Name:   attrName,
				GoType: primitiveToGo(attrDef.PrimitiveType),
			}
		}

		svc.Resources[resourceName] = resource
	}

	// Parse property types and add to services
	for cfType, propTypeDef := range cfnSpec.PropertyTypes {
		// Parse AWS::S3::Bucket.VersioningConfiguration
		parts := strings.Split(cfType, "::")
		if len(parts) != 3 || parts[0] != "AWS" {
			continue
		}

		// The property type name is after the dot
		dotParts := strings.SplitN(parts[2], ".", 2)
		if len(dotParts) != 2 {
			continue
		}

		serviceName := strings.ToLower(parts[1])
		parentResource := dotParts[0]
		propTypeName := dotParts[1]

		if filterService != "" && serviceName != filterService {
			continue
		}

		svc, ok := services[serviceName]
		if !ok {
			continue
		}

		// Parse properties for this property type
		properties := make(map[string]ParsedProperty)
		for propName, propDef := range propTypeDef.Properties {
			properties[propName] = parseProperty(propName, propDef)
		}

		// Parse as a full nested struct with properties
		parsed := ParsedPropertyType{
			Name:           propTypeName,
			CFType:         cfType,
			ParentResource: parentResource,
			Documentation:  propTypeDef.Documentation,
			Properties:     properties,
		}

		// Use qualified key to avoid collisions when multiple resources have
		// property types with the same name (e.g., PublicAccessBlockConfiguration)
		qualifiedKey := parentResource + "_" + propTypeName
		svc.PropertyTypes[qualifiedKey] = parsed
	}

	// Convert to sorted slice
	result := make([]*Service, 0, len(services))
	for _, svc := range services {
		result = append(result, svc)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// parseProperty converts a CloudFormation property definition to our parsed format.
func parseProperty(name string, def spec.Property) ParsedProperty {
	prop := ParsedProperty{
		Name:          name,
		Documentation: def.Documentation,
		Required:      def.Required,
	}

	// Determine Go type
	if def.PrimitiveType != "" {
		prop.GoType = primitiveToGo(def.PrimitiveType)
		prop.CFType = def.PrimitiveType
	} else if def.Type == "List" {
		prop.IsList = true
		if def.PrimitiveItemType != "" {
			prop.ItemType = primitiveToGo(def.PrimitiveItemType)
		} else if def.ItemType != "" {
			prop.ItemType = def.ItemType
		} else {
			prop.ItemType = "any"
		}
		prop.GoType = "[]" + prop.ItemType
	} else if def.Type == "Map" {
		prop.IsMap = true
		if def.PrimitiveItemType != "" {
			prop.ItemType = primitiveToGo(def.PrimitiveItemType)
		} else if def.ItemType != "" {
			prop.ItemType = def.ItemType
		} else {
			prop.ItemType = "any"
		}
		prop.GoType = "map[string]" + prop.ItemType
	} else if def.Type != "" {
		// Reference to a property type
		prop.GoType = def.Type
		prop.CFType = def.Type
	} else {
		prop.GoType = "any"
	}

	// Non-required fields should be pointers (except slices/maps/any which are nil-able)
	// Since all primitives are now `any`, only property type references need pointers
	if !def.Required && !prop.IsList && !prop.IsMap && prop.GoType != "any" && def.Type != "" {
		prop.IsPointer = true
	}

	return prop
}

// primitiveToGo converts CloudFormation primitive types to Go types.
// All primitive types map to `any` to allow both literal values and intrinsic
// functions (Ref, GetAtt, Sub, etc.) to be assigned to any property field.
// This matches the Python implementation where every field uses Union types.
func primitiveToGo(cfType string) string {
	// All types are `any` to accept intrinsic functions
	return "any"
}
