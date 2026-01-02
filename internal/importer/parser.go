package importer

import (
	"path/filepath"
	"strings"

	"github.com/lex00/cloudformation-schema-go/template"
)

// ParseTemplate parses a CloudFormation template file into IR.
// Supports both YAML and JSON formats.
func ParseTemplate(path string) (*IRTemplate, error) {
	tmpl, err := template.ParseTemplate(path)
	if err != nil {
		return nil, err
	}
	return convertTemplate(tmpl), nil
}

// ParseTemplateContent parses CloudFormation template content into IR.
func ParseTemplateContent(content []byte, sourceName string) (*IRTemplate, error) {
	tmpl, err := template.ParseTemplateContent(content, sourceName)
	if err != nil {
		return nil, err
	}
	return convertTemplate(tmpl), nil
}

// convertTemplate converts a shared template.Template to the local IRTemplate.
// This adds GoName fields to properties for code generation.
func convertTemplate(tmpl *template.Template) *IRTemplate {
	ir := NewIRTemplate()
	ir.Description = tmpl.Description
	ir.AWSTemplateFormatVersion = tmpl.AWSTemplateFormatVersion
	ir.SourceFile = tmpl.SourceFile

	// Convert parameters
	for logicalID, param := range tmpl.Parameters {
		ir.Parameters[logicalID] = convertParameter(param)
	}

	// Convert mappings
	for logicalID, mapping := range tmpl.Mappings {
		ir.Mappings[logicalID] = convertMapping(mapping)
	}

	// Convert conditions
	for logicalID, condition := range tmpl.Conditions {
		ir.Conditions[logicalID] = convertCondition(condition)
	}

	// Convert resources
	for logicalID, resource := range tmpl.Resources {
		ir.Resources[logicalID] = convertResource(resource)
	}

	// Convert outputs
	for logicalID, output := range tmpl.Outputs {
		ir.Outputs[logicalID] = convertOutput(output)
	}

	// Copy reference graph
	for k, v := range tmpl.ReferenceGraph {
		ir.ReferenceGraph[k] = v
	}

	return ir
}

func convertParameter(param *template.Parameter) *IRParameter {
	return &IRParameter{
		LogicalID:             param.LogicalID,
		Type:                  param.Type,
		Description:           param.Description,
		Default:               param.Default,
		AllowedValues:         param.AllowedValues,
		AllowedPattern:        param.AllowedPattern,
		MinLength:             param.MinLength,
		MaxLength:             param.MaxLength,
		MinValue:              param.MinValue,
		MaxValue:              param.MaxValue,
		ConstraintDescription: param.ConstraintDescription,
		NoEcho:                param.NoEcho,
	}
}

func convertMapping(mapping *template.Mapping) *IRMapping {
	return &IRMapping{
		LogicalID: mapping.LogicalID,
		MapData:   mapping.MapData,
	}
}

func convertCondition(condition *template.Condition) *IRCondition {
	return &IRCondition{
		LogicalID:  condition.LogicalID,
		Expression: condition.Expression,
	}
}

func convertResource(resource *template.Resource) *IRResource {
	ir := &IRResource{
		LogicalID:           resource.LogicalID,
		ResourceType:        resource.ResourceType,
		Properties:          make(map[string]*IRProperty),
		DependsOn:           resource.DependsOn,
		Condition:           resource.Condition,
		DeletionPolicy:      resource.DeletionPolicy,
		UpdateReplacePolicy: resource.UpdateReplacePolicy,
		Metadata:            resource.Metadata,
	}

	// Convert properties, adding GoName
	for name, prop := range resource.Properties {
		ir.Properties[name] = &IRProperty{
			DomainName: prop.Name,
			GoName:     transformGoFieldName(prop.Name),
			Value:      prop.Value,
		}
	}

	return ir
}

func convertOutput(output *template.Output) *IROutput {
	return &IROutput{
		LogicalID:   output.LogicalID,
		Value:       output.Value,
		Description: output.Description,
		ExportName:  output.ExportName,
		Condition:   output.Condition,
	}
}

// transformGoFieldName handles Go keyword conflicts in field names.
// CloudFormation uses PascalCase (Type), Go keywords are lowercase (type).
// If the lowercase version is a keyword, append underscore (Type_).
func transformGoFieldName(name string) string {
	// Check if the lowercase version is a Go keyword
	lower := strings.ToLower(name)
	if isGoKeyword(lower) {
		return name + "_"
	}
	// Also handle ResourceType which conflicts with a method
	if name == "ResourceType" {
		return "ResourceTypeProp"
	}
	return name
}

// goKeywords lists Go keywords that require field name transformation.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

func isGoKeyword(s string) bool {
	return goKeywords[s]
}

// reservedPackageNames are resource package names that would conflict with imports.
// If a template filename matches one of these, we add "_stack" suffix.
var reservedPackageNames = map[string]bool{
	"s3": true, "ec2": true, "iam": true, "lambda": true, "dynamodb": true,
	"sqs": true, "sns": true, "rds": true, "ecs": true, "eks": true,
	"cloudfront": true, "cloudwatch": true, "route53": true, "apigateway": true,
	"elasticloadbalancingv2": true, "elasticloadbalancing": true,
	"kms": true, "secretsmanager": true, "ssm": true, "logs": true,
	"events": true, "kinesis": true, "firehose": true, "glue": true,
	"athena": true, "redshift": true, "elasticsearch": true, "opensearchservice": true,
	"cognito": true, "waf": true, "wafv2": true, "acm": true,
	"cloudformation": true, "config": true, "guardduty": true, "inspector": true,
	"macie": true, "securityhub": true, "stepfunctions": true, "appsync": true,
	"amplify": true, "codecommit": true, "codebuild": true, "codepipeline": true,
	"codedeploy": true, "codestar": true, "ecr": true, "batch": true,
	"sagemaker": true, "iot": true, "greengrass": true, "mediaconvert": true,
	"intrinsics": true, // Also reserved
}

// DerivePackageName creates a valid Go package name from a file path.
func DerivePackageName(path string) string {
	base := filepath.Base(path)
	// Remove extension
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	// Sanitize for Go package name
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	// Remove leading non-letter characters, but keep underscores at start
	for len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name[1:]
	}
	if name == "" {
		name = "imported"
	}
	// Avoid conflicts with resource package names
	if reservedPackageNames[name] {
		name = name + "_stack"
	}
	return name
}
