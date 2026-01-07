// Package template provides CloudFormation template building from discovered resources.
package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// Builder constructs CloudFormation templates from discovered resources.
type Builder struct {
	resources  map[string]wetwire.DiscoveredResource
	parameters map[string]wetwire.DiscoveredParameter
	outputs    map[string]wetwire.DiscoveredOutput
	mappings   map[string]wetwire.DiscoveredMapping
	conditions map[string]wetwire.DiscoveredCondition
	values     map[string]any // Actual struct values for serialization
}

// NewBuilder creates a template builder from discovered resources.
func NewBuilder(resources map[string]wetwire.DiscoveredResource) *Builder {
	return &Builder{
		resources:  resources,
		parameters: make(map[string]wetwire.DiscoveredParameter),
		outputs:    make(map[string]wetwire.DiscoveredOutput),
		mappings:   make(map[string]wetwire.DiscoveredMapping),
		conditions: make(map[string]wetwire.DiscoveredCondition),
		values:     make(map[string]any),
	}
}

// NewBuilderFull creates a template builder from all discovered components.
func NewBuilderFull(
	resources map[string]wetwire.DiscoveredResource,
	parameters map[string]wetwire.DiscoveredParameter,
	outputs map[string]wetwire.DiscoveredOutput,
	mappings map[string]wetwire.DiscoveredMapping,
	conditions map[string]wetwire.DiscoveredCondition,
) *Builder {
	return &Builder{
		resources:  resources,
		parameters: parameters,
		outputs:    outputs,
		mappings:   mappings,
		conditions: conditions,
		values:     make(map[string]any),
	}
}

// SetValue associates a resource value with its logical name.
// This is called by the CLI after loading the resource values.
func (b *Builder) SetValue(name string, value any) {
	b.values[name] = value
}

// Build constructs the CloudFormation template.
func (b *Builder) Build() (*wetwire.Template, error) {
	// Get resources in dependency order
	order, err := b.topologicalSort()
	if err != nil {
		return nil, err
	}

	template := &wetwire.Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Resources:                make(map[string]wetwire.ResourceDef),
	}

	// Build Parameters section
	if len(b.parameters) > 0 {
		template.Parameters = make(map[string]wetwire.Parameter)
		for name := range b.parameters {
			if val, ok := b.values[name]; ok {
				param := b.serializeParameter(name, val)
				template.Parameters[name] = param
			}
		}
	}

	// Build Mappings section
	if len(b.mappings) > 0 {
		template.Mappings = make(map[string]any)
		for name := range b.mappings {
			if val, ok := b.values[name]; ok {
				template.Mappings[name] = val
			}
		}
	}

	// Build Conditions section
	if len(b.conditions) > 0 {
		template.Conditions = make(map[string]any)
		for name := range b.conditions {
			if val, ok := b.values[name]; ok {
				template.Conditions[name] = val
			}
		}
	}

	// Track if any SAM resources are present
	hasSAMResources := false

	for _, name := range order {
		res := b.resources[name]
		value := b.values[name]

		resourceType := cfResourceType(res.Type)
		if resourceType == "" {
			return nil, fmt.Errorf("unknown resource type: %s", res.Type)
		}

		// Check if this is a SAM resource
		if isSAMResourceType(res.Type) {
			hasSAMResources = true
		}

		// Serialize the resource value to properties
		props, err := b.serializeResource(name, value, res)
		if err != nil {
			return nil, fmt.Errorf("serializing %s: %w", name, err)
		}

		template.Resources[name] = wetwire.ResourceDef{
			Type:       resourceType,
			Properties: props,
		}
	}

	// Build Outputs section
	if len(b.outputs) > 0 {
		template.Outputs = make(map[string]wetwire.Output)
		for name := range b.outputs {
			if val, ok := b.values[name]; ok {
				output := b.serializeOutput(name, val)
				template.Outputs[name] = output
			}
		}
	}

	// Set SAM Transform header if any SAM resources are present
	if hasSAMResources {
		template.Transform = "AWS::Serverless-2016-10-31"
	}

	return template, nil
}

// serializeParameter converts a Parameter value to the template format.
func (b *Builder) serializeParameter(name string, value any) wetwire.Parameter {
	// The value is already serialized as a map from the runner
	valMap, ok := value.(map[string]any)
	if !ok {
		return wetwire.Parameter{Type: "String"}
	}

	param := wetwire.Parameter{}

	if t, ok := valMap["Type"].(string); ok {
		param.Type = t
	} else {
		param.Type = "String" // Default
	}
	if desc, ok := valMap["Description"].(string); ok {
		param.Description = desc
	}
	if def, ok := valMap["Default"]; ok {
		param.Default = def
	}
	if vals, ok := valMap["AllowedValues"].([]any); ok {
		param.AllowedValues = vals
	}
	if pattern, ok := valMap["AllowedPattern"].(string); ok {
		param.AllowedPattern = pattern
	}
	if desc, ok := valMap["ConstraintDescription"].(string); ok {
		param.ConstraintDescription = desc
	}
	if v, ok := valMap["MinLength"].(float64); ok {
		i := int(v)
		param.MinLength = &i
	}
	if v, ok := valMap["MaxLength"].(float64); ok {
		i := int(v)
		param.MaxLength = &i
	}
	if v, ok := valMap["MinValue"].(float64); ok {
		param.MinValue = &v
	}
	if v, ok := valMap["MaxValue"].(float64); ok {
		param.MaxValue = &v
	}
	if v, ok := valMap["NoEcho"].(bool); ok {
		param.NoEcho = v
	}

	return param
}

// serializeOutput converts an Output value to the template format.
func (b *Builder) serializeOutput(name string, value any) wetwire.Output {
	valMap, ok := value.(map[string]any)
	if !ok {
		return wetwire.Output{}
	}

	output := wetwire.Output{}

	if desc, ok := valMap["Description"].(string); ok {
		output.Description = desc
	}
	if val, ok := valMap["Value"]; ok {
		output.Value = val
	}
	if exp, ok := valMap["Export"].(map[string]any); ok {
		if expName, ok := exp["Name"].(string); ok {
			output.Export = &struct {
				Name string `json:"Name"`
			}{Name: expName}
		}
	}
	// Handle ExportName field (alternative format)
	if expName, ok := valMap["ExportName"]; ok {
		output.Export = &struct {
			Name string `json:"Name"`
		}{Name: fmt.Sprintf("%v", expName)}
	}

	return output
}

// serializeResource converts a Go struct to CloudFormation properties.
func (b *Builder) serializeResource(name string, value any, res wetwire.DiscoveredResource) (map[string]any, error) {
	// First, convert to JSON to normalize the structure
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var props map[string]any
	if err := json.Unmarshal(data, &props); err != nil {
		return nil, err
	}

	// Transform any resource references
	props = b.transformRefs(name, props, res)

	return props, nil
}

// transformRefs converts resource references to CloudFormation intrinsics.
func (b *Builder) transformRefs(name string, props map[string]any, res wetwire.DiscoveredResource) map[string]any {
	result := make(map[string]any)

	for key, value := range props {
		result[key] = b.transformValue(value)
	}

	return result
}

func (b *Builder) transformValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		// Check if this is already an intrinsic function
		if _, ok := v["Ref"]; ok {
			return v
		}
		if _, ok := v["Fn::GetAtt"]; ok {
			return v
		}
		if _, ok := v["Fn::Sub"]; ok {
			return v
		}

		// Recursively transform map values
		result := make(map[string]any)
		for key, val := range v {
			result[key] = b.transformValue(val)
		}
		return result

	case []any:
		// Recursively transform slice elements
		result := make([]any, len(v))
		for i, elem := range v {
			result[i] = b.transformValue(elem)
		}
		return result

	default:
		return value
	}
}

// topologicalSort returns resources in dependency order.
func (b *Builder) topologicalSort() ([]string, error) {
	// Build adjacency list
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for name := range b.resources {
		graph[name] = nil
		inDegree[name] = 0
	}

	for name, res := range b.resources {
		for _, dep := range res.Dependencies {
			if _, exists := b.resources[dep]; exists {
				graph[dep] = append(graph[dep], name)
				inDegree[name]++
			}
		}
	}

	// Kahn's algorithm
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Deterministic order

	var result []string
	for len(queue) > 0 {
		// Pop from queue
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Process neighbors
		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue) // Keep sorted for determinism
			}
		}
	}

	// Check for cycles
	if len(result) != len(b.resources) {
		return nil, b.detectCycle()
	}

	return result, nil
}

// detectCycle finds and reports a cycle in the dependency graph.
func (b *Builder) detectCycle() error {
	// Simple cycle detection: find resources with remaining in-degree
	visited := make(map[string]bool)
	path := make(map[string]bool)

	var cycle []string
	var findCycle func(node string) bool
	findCycle = func(node string) bool {
		visited[node] = true
		path[node] = true

		for _, dep := range b.resources[node].Dependencies {
			if _, exists := b.resources[dep]; !exists {
				continue
			}
			if !visited[dep] {
				if findCycle(dep) {
					cycle = append([]string{node}, cycle...)
					return true
				}
			} else if path[dep] {
				cycle = append([]string{dep, node}, cycle...)
				return true
			}
		}

		path[node] = false
		return false
	}

	for name := range b.resources {
		if !visited[name] {
			if findCycle(name) {
				break
			}
		}
	}

	if len(cycle) > 0 {
		// Format cycle for error message
		msg := "circular dependency detected:\n"
		for i, name := range cycle {
			res := b.resources[name]
			msg += fmt.Sprintf("  %s (%s:%d)", name, res.File, res.Line)
			if i < len(cycle)-1 {
				msg += "\n    → "
			}
		}
		return errors.New(msg)
	}

	return errors.New("circular dependency detected")
}

// cfResourceType converts Go type to CloudFormation type.
// e.g., "s3.Bucket" -> "AWS::S3::Bucket"
func cfResourceType(goType string) string {
	typeMap := map[string]string{
		"s3.Bucket":             "AWS::S3::Bucket",
		"s3.BucketPolicy":       "AWS::S3::BucketPolicy",
		"iam.Role":              "AWS::IAM::Role",
		"iam.Policy":            "AWS::IAM::Policy",
		"iam.InstanceProfile":   "AWS::IAM::InstanceProfile",
		"lambda.Function":       "AWS::Lambda::Function",
		"lambda.Permission":     "AWS::Lambda::Permission",
		"ec2.VPC":               "AWS::EC2::VPC",
		"ec2.Subnet":            "AWS::EC2::Subnet",
		"ec2.SecurityGroup":     "AWS::EC2::SecurityGroup",
		"ec2.Instance":          "AWS::EC2::Instance",
		"dynamodb.Table":        "AWS::DynamoDB::Table",
		"sqs.Queue":             "AWS::SQS::Queue",
		"sns.Topic":             "AWS::SNS::Topic",
		"apigateway.RestApi":    "AWS::ApiGateway::RestApi",
		"events.Rule":           "AWS::Events::Rule",
		"logs.LogGroup":         "AWS::Logs::LogGroup",
		"kms.Key":               "AWS::KMS::Key",
		"secretsmanager.Secret": "AWS::SecretsManager::Secret",
		// SAM (Serverless Application Model) resources
		"serverless.Function":     "AWS::Serverless::Function",
		"serverless.Api":          "AWS::Serverless::Api",
		"serverless.HttpApi":      "AWS::Serverless::HttpApi",
		"serverless.SimpleTable":  "AWS::Serverless::SimpleTable",
		"serverless.LayerVersion": "AWS::Serverless::LayerVersion",
		"serverless.StateMachine": "AWS::Serverless::StateMachine",
		"serverless.Application":  "AWS::Serverless::Application",
		"serverless.Connector":    "AWS::Serverless::Connector",
		"serverless.GraphQLApi":   "AWS::Serverless::GraphQLApi",
		// Add more as needed - code generator will maintain this
	}
	return typeMap[goType]
}

// isSAMResourceType returns true if the Go type is a SAM resource.
func isSAMResourceType(goType string) bool {
	return len(goType) > 11 && goType[:11] == "serverless."
}

// ToJSON serializes the template to JSON.
func ToJSON(t *wetwire.Template) ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

// ToYAML serializes the template to YAML.
func ToYAML(t *wetwire.Template) ([]byte, error) {
	return yaml.Marshal(t)
}
