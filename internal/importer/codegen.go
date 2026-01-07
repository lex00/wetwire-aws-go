package importer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/lex00/cloudformation-schema-go/enums"

	"github.com/lex00/wetwire-aws-go/resources"
)

// serviceCategories maps AWS service names to file categories.
// This matches the Python implementation in linter/splitting.py.
var serviceCategories = map[string]string{
	// Compute
	"EC2":         "compute",
	"Lambda":      "compute",
	"ECS":         "compute",
	"EKS":         "compute",
	"Batch":       "compute",
	"AutoScaling": "compute",
	// Storage
	"S3":  "storage",
	"EFS": "storage",
	"FSx": "storage",
	// Database
	"RDS":         "database",
	"DynamoDB":    "database",
	"ElastiCache": "database",
	"Neptune":     "database",
	"DocumentDB":  "database",
	"Redshift":    "database",
	// Networking
	"ElasticLoadBalancing":   "network",
	"ElasticLoadBalancingV2": "network",
	"Route53":                "network",
	"CloudFront":             "network",
	"APIGateway":             "network",
	"ApiGatewayV2":           "network",
	// Security/IAM
	"IAM":            "security",
	"Cognito":        "security",
	"SecretsManager": "security",
	"KMS":            "security",
	"WAF":            "security",
	"WAFv2":          "security",
	"ACM":            "security",
	"SSM":            "security",
	// Messaging/Integration
	"SNS":           "messaging",
	"SQS":           "messaging",
	"EventBridge":   "messaging",
	"Events":        "messaging",
	"StepFunctions": "messaging",
	// Monitoring/Logging
	"CloudWatch":      "monitoring",
	"Logs":            "monitoring",
	"CloudTrail":      "monitoring",
	"KinesisFirehose": "monitoring",
	// CI/CD
	"CodeBuild":    "cicd",
	"CodePipeline": "cicd",
	"CodeCommit":   "cicd",
	"CodeDeploy":   "cicd",
	// Infrastructure
	"CloudFormation": "infra",
	"Config":         "infra",
	"ServiceCatalog": "infra",
}

// ec2NetworkKeywords are keywords that identify EC2 resources as network-related.
var ec2NetworkKeywords = []string{
	"VPC", "Subnet", "Route", "Gateway", "Network", "Interface",
	"Security", "Acl", "VPN", "Transit", "Peering", "EIP",
	"Customer", "DHCP", "Carrier", "Insights", "FlowLog",
	"Association", "Attachment", "Prefix", "Traffic", "Egress",
	"Ingress", "LocalGateway", "Verified", "Endpoint",
}

// ec2ComputeKeywords are keywords that identify EC2 resources as compute-related.
var ec2ComputeKeywords = []string{
	"Instance", "Fleet", "Host", "KeyPair", "Capacity", "Volume",
	"Placement", "IPAM", "Snapshot", "Enclave", "LaunchTemplate",
	"SpotFleet", "Image", "AMI",
}

// isEC2NetworkType checks if an EC2 resource type is network-related.
func isEC2NetworkType(typeName string) bool {
	// Special case: Endpoint types are always network
	if strings.Contains(typeName, "Endpoint") {
		return true
	}

	// Exclude compute keywords first (these take precedence)
	for _, kw := range ec2ComputeKeywords {
		if strings.Contains(typeName, kw) {
			return false
		}
	}

	// Include network keywords
	for _, kw := range ec2NetworkKeywords {
		if strings.Contains(typeName, kw) {
			return true
		}
	}

	return false
}

// categorizeResourceType returns the category for a CloudFormation resource type.
// Maps AWS resource types to category names for file organization.
func categorizeResourceType(resourceType string) string {
	// Parse resource type: "AWS::EC2::VPC" → service="EC2", typeName="VPC"
	parts := strings.Split(resourceType, "::")
	if len(parts) != 3 || parts[0] != "AWS" {
		return "main"
	}

	service := parts[1]
	typeName := parts[2]

	// Special case: EC2 VPC/networking resources go to network, not compute
	if service == "EC2" && isEC2NetworkType(typeName) {
		return "network"
	}

	if category, ok := serviceCategories[service]; ok {
		return category
	}

	return "main"
}

// pseudoParameterConstants maps pseudo-parameter strings that appear as literal values
// to their Go constant equivalents. This handles edge cases where pseudo-parameters
// appear outside of Ref context.
var pseudoParameterConstants = map[string]string{
	"AWS::NoValue":          "AWS_NO_VALUE",
	"AWS::Region":           "AWS_REGION",
	"AWS::AccountId":        "AWS_ACCOUNT_ID",
	"AWS::StackName":        "AWS_STACK_NAME",
	"AWS::StackId":          "AWS_STACK_ID",
	"AWS::Partition":        "AWS_PARTITION",
	"AWS::URLSuffix":        "AWS_URL_SUFFIX",
	"AWS::NotificationARNs": "AWS_NOTIFICATION_ARNS",
}

// cfServiceToEnumService maps CloudFormation service names (as used in currentResource)
// to botocore service names used in the enums package.
var cfServiceToEnumService = map[string]string{
	"lambda":                 "lambda",
	"ec2":                    "ec2",
	"ecs":                    "ecs",
	"s3":                     "s3",
	"dynamodb":               "dynamodb",
	"apigateway":             "apigateway",
	"elasticloadbalancingv2": "elbv2",
	"logs":                   "logs",
	"acm":                    "acm",
	"events":                 "events",
}

// listTypeProperties are CloudFormation properties that expect list/array types ([]any in Go).
// When assigning intrinsics like GetAZs{} to these, we need to wrap them in []any{}.
var listTypeProperties = map[string]bool{
	// Common list properties
	"AvailabilityZones":                   true,
	"SubnetIds":                           true,
	"Subnets":                             true,
	"SecurityGroupIds":                    true,
	"SecurityGroups":                      true,
	"VpcSecurityGroupIds":                 true,
	"VPCSecurityGroups":                   true,
	"VPCZoneIdentifier":                   true,
	"LoadBalancerNames":                   true,
	"TargetGroupArns":                     true,
	"TargetGroupARNs":                     true,
	"NotificationConfigurations":          true,
	"NotificationArns":                    true,
	"TopicARN":                            true,
	"Regions":                             true,
	"AdditionalPrimaryNodeSecurityGroups": true,
	"AdditionalCoreNodeSecurityGroups":    true,
	"AdditionalMasterSecurityGroups":      true,
	"AdditionalSlaveSecurityGroups":       true,
	"StackSetRegions":                     true,
}

// reservedVarNames are names that collide with types exported from the intrinsics package
// (which is dot-imported). When a resource has one of these logical IDs, we append "Resource".
var reservedVarNames = map[string]bool{
	"Transform": true, // intrinsics.Transform
	"Output":    true, // intrinsics.Output
	"Condition": true, // intrinsics.Condition
	"Tag":       true, // intrinsics.Tag
	"Parameter": true, // intrinsics.Parameter
	"Mapping":   true, // intrinsics.Mapping
	"Ref":       true, // intrinsics.Ref
	"GetAtt":    true, // intrinsics.GetAtt
	"Sub":       true, // intrinsics.Sub
	"Join":      true, // intrinsics.Join
	"Select":    true, // intrinsics.Select
	"If":        true, // intrinsics.If
	"And":       true, // intrinsics.And
	"Or":        true, // intrinsics.Or
	"Not":       true, // intrinsics.Not
	"Equals":    true, // intrinsics.Equals
	"GetAZs":    true, // intrinsics.GetAZs
	"Split":     true, // intrinsics.Split
	"Cidr":      true, // intrinsics.Cidr
	"FindInMap": true, // intrinsics.FindInMap
}

// sanitizeVarName returns a safe Go variable name that doesn't collide with intrinsics types.
// Also ensures the variable is exported (starts with uppercase) for cross-package access.
func sanitizeVarName(name string) string {
	// Capitalize first letter to ensure the variable is exported
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	if reservedVarNames[name] {
		return name + "Resource"
	}
	return name
}

// isListTypeProperty checks if a property expects a list type ([]any).
func isListTypeProperty(propName string) bool {
	return listTypeProperties[propName]
}

// intrinsicNeedsArrayWrapping checks if an intrinsic needs to be wrapped in []any{}
// when assigned to a list-type field. In Go, intrinsic types (structs) can't be
// directly assigned to []any fields, so we wrap them.
func intrinsicNeedsArrayWrapping(intrinsic *IRIntrinsic) bool {
	switch intrinsic.Type {
	case IntrinsicGetAZs, IntrinsicSplit, IntrinsicIf, IntrinsicRef:
		// GetAZs, Split return lists
		// If may return lists (when branches are lists)
		// Ref to a Parameter that's a list type needs wrapping
		return true
	default:
		return false
	}
}

// tryEnumConstant attempts to convert a string value to an enum constant reference.
// Returns empty string if no enum mapping exists or the value is not a valid enum value.
func tryEnumConstant(ctx *codegenContext, value string) string {
	if ctx.currentResource == "" || ctx.currentProperty == "" {
		return ""
	}

	// Map CF service name to enums service name
	enumService := cfServiceToEnumService[strings.ToLower(ctx.currentResource)]
	if enumService == "" {
		return ""
	}

	// Look up the enum for this property
	enumName := enums.GetEnumForProperty(enumService, ctx.currentProperty)
	if enumName == "" {
		return ""
	}

	// Check if the value is valid for this enum
	if !enums.IsValidValue(enumService, enumName, value) {
		return ""
	}

	// Generate the constant name: enums.{Service}{EnumName}{ValueName}
	constName := toEnumConstantName(enumService, enumName, value)
	ctx.imports["github.com/lex00/cloudformation-schema-go/enums"] = true
	return "enums." + constName
}

// toEnumConstantName generates the Go constant name for an enum value.
// Example: ("lambda", "Runtime", "python3.12") -> "LambdaRuntimePython312"
func toEnumConstantName(service, enumName, value string) string {
	// Capitalize service name
	serviceCap := capitalizeService(service)

	// Normalize value: replace all non-alphanumeric with spaces for word splitting
	var normalized strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(r)
		} else {
			normalized.WriteRune(' ')
		}
	}

	// Capitalize each word
	var valuePart strings.Builder
	for _, word := range strings.Fields(normalized.String()) {
		if word != "" {
			valuePart.WriteString(strings.ToUpper(string(word[0])))
			valuePart.WriteString(strings.ToLower(word[1:]))
		}
	}

	return serviceCap + enumName + valuePart.String()
}

// capitalizeService capitalizes service name for Go constant prefix.
func capitalizeService(service string) string {
	switch service {
	case "ec2":
		return "Ec2"
	case "ecs":
		return "Ecs"
	case "rds":
		return "Rds"
	case "s3":
		return "S3"
	case "acm":
		return "Acm"
	case "elbv2":
		return "Elbv2"
	default:
		if service == "" {
			return ""
		}
		return strings.ToUpper(string(service[0])) + service[1:]
	}
}

// policyDocFields lists fields that contain IAM policy documents.
// These are flattened into typed PolicyDocument/PolicyStatement structs.
var policyDocFields = map[string]bool{
	"AssumeRolePolicyDocument": true,
	"PolicyDocument":           true,
	"KeyPolicy":                true,
}

// isPolicyDocumentField checks if a property name is an IAM policy document.
func isPolicyDocumentField(propName string) bool {
	return policyDocFields[propName]
}

// goStringLiteral converts a string to a Go string literal.
// Uses backtick (raw string) syntax for multi-line strings to improve readability.
func goStringLiteral(s string) string {
	// Use backticks for multi-line strings if they don't contain backticks
	if strings.Contains(s, "\n") && !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return fmt.Sprintf("%q", s)
}

// conditionOperators maps IAM condition operator strings to Go constant names.
// These are exported from intrinsics/policy.go via dot import.
var conditionOperators = map[string]string{
	"StringEquals":              "StringEquals",
	"StringNotEquals":           "StringNotEquals",
	"StringEqualsIgnoreCase":    "StringEqualsIgnoreCase",
	"StringNotEqualsIgnoreCase": "StringNotEqualsIgnoreCase",
	"StringLike":                "StringLike",
	"StringNotLike":             "StringNotLike",
	"NumericEquals":             "NumericEquals",
	"NumericNotEquals":          "NumericNotEquals",
	"NumericLessThan":           "NumericLessThan",
	"NumericLessThanEquals":     "NumericLessThanEquals",
	"NumericGreaterThan":        "NumericGreaterThan",
	"NumericGreaterThanEquals":  "NumericGreaterThanEquals",
	"DateEquals":                "DateEquals",
	"DateNotEquals":             "DateNotEquals",
	"DateLessThan":              "DateLessThan",
	"DateLessThanEquals":        "DateLessThanEquals",
	"DateGreaterThan":           "DateGreaterThan",
	"DateGreaterThanEquals":     "DateGreaterThanEquals",
	"Bool":                      "Bool",
	"IpAddress":                 "IpAddress",
	"NotIpAddress":              "NotIpAddress",
	"ArnEquals":                 "ArnEquals",
	"ArnNotEquals":              "ArnNotEquals",
	"ArnLike":                   "ArnLike",
	"ArnNotLike":                "ArnNotLike",
	"Null":                      "Null",
}

// GenerateTemplateFiles generates template/scaffold files for a new project.
// These are optional files that help set up a complete Go project.
func GenerateTemplateFiles(packageName string, modulePath string) map[string]string {
	files := make(map[string]string)

	// go.mod
	if modulePath == "" {
		modulePath = packageName
	}
	files["go.mod"] = fmt.Sprintf(`module %s

go 1.23.0

require (
	github.com/lex00/cloudformation-schema-go v1.0.0
	github.com/lex00/wetwire-aws-go v1.0.0
)

// For local development:
replace github.com/lex00/wetwire-aws-go => ../../..
`, modulePath)

	// cmd/main.go - Entry point placeholder
	// Note: The actual synthesis is done via `wetwire-aws build`
	files["cmd/main.go"] = `package main

import "fmt"

func main() {
	// Build this template using the wetwire-aws CLI:
	//   wetwire-aws build .
	//
	// This generates template.json from the Go resource definitions.
	fmt.Println("Usage: wetwire-aws build .")
}
`

	// .gitignore
	files[".gitignore"] = `# Build output
template.json
template.yaml
*.out

# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
`

	// README.md - Project documentation
	files["README.md"] = fmt.Sprintf(`# %s

CloudFormation infrastructure as Go code, generated by [wetwire-aws](https://github.com/lex00/wetwire-aws-go).

## Prerequisites

- Go 1.22+
- wetwire-aws CLI

## Quick Start

`+"`"+`bash
# Build the CloudFormation template
wetwire-aws build .

# Or run directly
cd cmd && go run main.go > template.json

# Deploy with AWS CLI
aws cloudformation deploy \
  --template-file template.json \
  --stack-name %s \
  --capabilities CAPABILITY_IAM
`+"`"+`

## Project Structure

| File | Description |
|------|-------------|
| `+"`params.go`"+` | CloudFormation parameters and conditions |
| `+"`outputs.go`"+` | Stack outputs |
| `+"`security.go`"+` | IAM roles, policies, KMS keys |
| `+"`network.go`"+` | VPC, subnets, security groups, load balancers |
| `+"`compute.go`"+` | EC2 instances, Lambda functions |
| `+"`storage.go`"+` | S3 buckets, EFS |
| `+"`database.go`"+` | RDS, DynamoDB |
| `+"`main.go`"+` | Other resources |

## Modifying Resources

Resources are defined as Go struct literals:

`+"`"+`go
var MyBucket = s3.Bucket{
    BucketName: Sub{String: "${AppName}-bucket-${Environment}"},
    Tags: []any{MyBucketTagName, MyBucketTagEnv},
}
`+"`"+`

Cross-resource references use direct variable names:

`+"`"+`go
var MyPolicy = s3.BucketPolicy{
    Bucket: MyBucket,  // References the bucket above
}
`+"`"+`

## License

[Add your license here]
`, packageName, packageName)

	// CLAUDE.md - Instructions for Claude Code
	files["CLAUDE.md"] = fmt.Sprintf(`# %s Infrastructure

This package contains CloudFormation resources generated by wetwire-aws.

## Structure

- `+"`params.go`"+` - CloudFormation parameters and conditions
- `+"`outputs.go`"+` - Stack outputs
- `+"`security.go`"+` - IAM roles, policies, KMS keys
- `+"`network.go`"+` - VPC, subnets, security groups, load balancers
- `+"`compute.go`"+` - EC2 instances, Lambda functions
- `+"`storage.go`"+` - S3 buckets, EFS
- `+"`database.go`"+` - RDS, DynamoDB
- `+"`main.go`"+` - Other resources

## Usage

### Build the template
`+"`"+`bash
cd cmd && go run main.go > template.json
`+"`"+`

### Or use wetwire-aws CLI
`+"`"+`bash
wetwire-aws build .
`+"`"+`

## Modifying Resources

Each resource is defined as a Go struct. For example:

`+"`"+`go
var MyBucket = s3.Bucket{
    BucketName: "my-bucket",
}
`+"`"+`

Cross-resource references use direct variable references:

`+"`"+`go
var MyPolicy = s3.BucketPolicy{
    Bucket: MyBucket,  // Reference to another resource
}
`+"`"+`

## Intrinsic Functions

Import from `+"`github.com/lex00/wetwire-aws-go/intrinsics`"+`:

`+"`"+`go
import . "github.com/lex00/wetwire-aws-go/intrinsics"

var MyRole = iam.Role{
    RoleName: Sub{String: "${AppName}-role-${Environment}"},
}
`+"`"+`
`, packageName)

	return files
}

// GenerateCode generates Go code from a parsed IR template.
// Returns a map of filename to content.
// Files are split by category:
//   - params.go: Parameters + Conditions
//   - outputs.go: Outputs
//   - security.go: IAM, Cognito, KMS, etc.
//   - network.go: VPC, Subnets, ELB, CloudFront, etc.
//   - compute.go: EC2 Instances, Lambda, ECS, etc.
//   - storage.go: S3, EFS, etc.
//   - database.go: RDS, DynamoDB, etc.
//   - messaging.go: SNS, SQS, EventBridge, etc.
//   - monitoring.go: CloudWatch, Logs, etc.
//   - main.go: Mappings + uncategorized resources
func GenerateCode(template *IRTemplate, packageName string) map[string]string {
	ctx := newCodegenContext(template, packageName)

	// Generate multi-file output
	files := make(map[string]string)

	// First pass: categorize resources
	resourcesByCategory := make(map[string][]string)
	for _, resourceID := range ctx.resourceOrder {
		resource := ctx.template.Resources[resourceID]
		category := categorizeResourceType(resource.ResourceType)
		resourcesByCategory[category] = append(resourcesByCategory[category], resourceID)
	}

	// Second pass: generate resources by category to track parameter usage and collect imports
	categoryCode := make(map[string]string)
	categoryImports := make(map[string]map[string]bool)

	for category, resourceIDs := range resourcesByCategory {
		code, imports := generateResourcesByIDs(ctx, resourceIDs)
		categoryCode[category] = code
		categoryImports[category] = imports
	}

	// Pre-scan all expressions for parameter references before generating params
	// This ensures parameters used in conditions, resources, and Sub strings are included
	prescanAllForParams(ctx)

	// Generate params.go if there are used parameters or conditions
	paramsCode, paramsImports := generateParams(ctx)
	conditionsCode := generateConditions(ctx)
	if paramsCode != "" || conditionsCode != "" {
		combined := paramsCode
		if conditionsCode != "" {
			if combined != "" {
				combined += "\n\n"
			}
			combined += conditionsCode
		}
		files["params.go"] = buildFile(ctx.packageName, "Parameters and Conditions", paramsImports, combined)
	}

	// Generate outputs.go if there are outputs
	if outputsCode, outputsImports := generateOutputs(ctx); outputsCode != "" {
		files["outputs.go"] = buildFile(ctx.packageName, "Outputs", outputsImports, outputsCode)
	}

	// Generate category files
	categoryDescriptions := map[string]string{
		"security":   "Security resources: IAM, Cognito, KMS, etc.",
		"network":    "Network resources: VPC, Subnets, Load Balancers, CloudFront, etc.",
		"compute":    "Compute resources: EC2, Lambda, ECS, etc.",
		"storage":    "Storage resources: S3, EFS, etc.",
		"database":   "Database resources: RDS, DynamoDB, etc.",
		"messaging":  "Messaging resources: SNS, SQS, EventBridge, etc.",
		"monitoring": "Monitoring resources: CloudWatch, Logs, etc.",
		"cicd":       "CI/CD resources: CodeBuild, CodePipeline, etc.",
		"infra":      "Infrastructure resources: CloudFormation, Config, etc.",
		"main":       "Main resources",
	}

	for category, code := range categoryCode {
		if code == "" {
			continue
		}

		imports := categoryImports[category]
		description := categoryDescriptions[category]
		if description == "" {
			description = category + " resources"
		}

		// Add mappings to main file only
		if category == "main" {
			if mappingsCode := generateMappings(ctx); mappingsCode != "" {
				code = mappingsCode + "\n\n" + code
			}
		}

		// Remove intrinsics import if the code doesn't actually use any intrinsic types
		// Resource/parameter references resolve to direct variable names, not intrinsic types
		if imports["github.com/lex00/wetwire-aws-go/intrinsics"] && !codeUsesIntrinsics(code) {
			delete(imports, "github.com/lex00/wetwire-aws-go/intrinsics")
		}

		filename := category + ".go"
		files[filename] = buildFile(ctx.packageName, description, imports, code)
	}

	// If there are only mappings and no main resources, still generate main.go
	if _, hasMain := categoryCode["main"]; !hasMain {
		if mappingsCode := generateMappings(ctx); mappingsCode != "" {
			imports := make(map[string]bool)
			if codeUsesIntrinsics(mappingsCode) {
				imports["github.com/lex00/wetwire-aws-go/intrinsics"] = true
			}
			files["main.go"] = buildFile(ctx.packageName, "Mappings", imports, mappingsCode)
		}
	}

	return files
}

// codeUsesIntrinsics checks if generated code actually uses intrinsic types.
// Resource/parameter references resolve to direct variable names, not intrinsic types,
// so we need to check if any intrinsic struct types are present in the code.
func codeUsesIntrinsics(code string) bool {
	intrinsicTypes := []string{
		"Sub{", "SubWithMap{", "Ref{", "GetAtt{", "Join{", "Select{", "GetAZs{",
		"If{", "Equals{", "And{", "Or{", "Not{", "FindInMap{", "Split{", "Cidr{",
		"Condition{", "ImportValue{", "Transform{", "Json{", "Parameter{", "Output{",
		"AWS_REGION", "AWS_ACCOUNT_ID", "AWS_STACK_NAME", "AWS_STACK_ID",
		"AWS_PARTITION", "AWS_URL_SUFFIX", "AWS_NO_VALUE", "AWS_NOTIFICATION_ARNS",
		"PolicyDocument{", "PolicyStatement{", "DenyStatement{", "AllowStatement{",
		"ServicePrincipal{", "AWSPrincipal{", "FederatedPrincipal{", "Tag{",
	}
	for _, t := range intrinsicTypes {
		if strings.Contains(code, t) {
			return true
		}
	}
	return false
}

// buildFile constructs a complete Go source file.
func buildFile(packageName, description string, imports map[string]bool, code string) string {
	var sections []string

	// Package header
	header := fmt.Sprintf("// Package %s contains CloudFormation resources.\n", packageName)
	if description != "" {
		header += fmt.Sprintf("// %s\n", description)
	}
	header += "//\n// Generated by wetwire-aws import.\n"
	header += fmt.Sprintf("package %s", packageName)
	sections = append(sections, header)

	// Imports
	if len(imports) > 0 {
		var importLines []string
		sortedImports := sortedKeys(imports)
		for _, imp := range sortedImports {
			if imp == "github.com/lex00/wetwire-aws-go/intrinsics" {
				importLines = append(importLines, fmt.Sprintf("\t. %q", imp))
			} else {
				importLines = append(importLines, fmt.Sprintf("\t%q", imp))
			}
		}
		sections = append(sections, fmt.Sprintf("import (\n%s\n)", strings.Join(importLines, "\n")))
	}

	// Code
	sections = append(sections, code)

	return strings.Join(sections, "\n\n") + "\n"
}

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

// codegenContext holds state during code generation.
type codegenContext struct {
	template         *IRTemplate
	packageName      string
	imports          map[string]bool // import path -> true
	resourceOrder    []string        // topologically sorted resource IDs
	currentResource  string          // current resource module being generated (e.g., "ec2", "cloudfront")
	currentTypeName  string          // current resource type name (e.g., "Distribution", "VPC")
	currentProperty  string          // current property being generated (e.g., "SecurityGroupIngress")
	currentLogicalID string          // current resource's logical ID (e.g., "SecurityGroup")

	// Block-style property type declarations
	// Each property type instance becomes its own var declaration
	propertyBlocks []propertyBlock // collected during resource traversal
	blockNameCount map[string]int  // for generating unique names

	// Track which parameters are directly referenced via Ref
	usedParameters map[string]bool

	// Track unknown resource types (placeholders) - GetAtt must use explicit GetAtt{} for these
	unknownResources map[string]bool
}

// propertyBlock represents a top-level var declaration for a property type instance.
type propertyBlock struct {
	varName    string         // e.g., "LoggingBucketBucketEncryption"
	typeName   string         // e.g., "s3.Bucket_BucketEncryption"
	properties map[string]any // property key-value pairs for generation
	isPointer  bool           // whether this should be a pointer type (&Type{})
	order      int            // insertion order for stable sorting
}

func newCodegenContext(template *IRTemplate, packageName string) *codegenContext {
	ctx := &codegenContext{
		template:         template,
		packageName:      packageName,
		imports:          make(map[string]bool),
		blockNameCount:   make(map[string]int),
		usedParameters:   make(map[string]bool),
		unknownResources: make(map[string]bool),
	}

	// Topologically sort resources
	ctx.resourceOrder = topologicalSort(template)

	// Pre-scan for unknown resource types so GetAtt can use explicit GetAtt{} for them
	for _, res := range template.Resources {
		module, _ := resolveResourceType(res.ResourceType)
		if module == "" {
			ctx.unknownResources[res.LogicalID] = true
		}
	}

	return ctx
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

// valueToBlockStyleProperty converts a property value to block style.
// Returns either a var reference (for typed properties) or a literal value.
func valueToBlockStyleProperty(ctx *codegenContext, value any, propName string, parentVarName string) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case *IRIntrinsic:
		goCode := intrinsicToGo(ctx, v)
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
		if len(items) == 0 {
			return "Json{}"
		}
		return fmt.Sprintf("Json{\n%s\n}", strings.Join(items, "\n"))
	}

	return fmt.Sprintf("%#v", value)
}

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

// isPointerField checks if a property field expects a pointer type.
// Uses the PointerFields registry generated by codegen.
func isPointerField(ctx *codegenContext, propName string) bool {
	if propName == "" || ctx.currentTypeName == "" {
		return false
	}
	key := ctx.currentResource + "." + ctx.currentTypeName + "." + propName
	return resources.PointerFields[key]
}

// getPropertyTypeName returns the typed struct name for a property, if known.
// CloudFormation property types are always flat: {ResourceType}_{PropertyTypeName}
// e.g., Distribution_DistributionConfig, Distribution_DefaultCacheBehavior, Distribution_Cookies
// Returns empty string if the property should use map[string]any.
func getPropertyTypeName(ctx *codegenContext, propName string) string {
	if propName == "" || ctx.currentTypeName == "" {
		return ""
	}

	// Skip known fields that should remain as map[string]any or are handled specially
	skipFields := map[string]bool{
		"Tags":     true,
		"Metadata": true,
	}
	if skipFields[propName] {
		return ""
	}

	// First, check PropertyTypeMap for the exact mapping.
	// Format: "service.ResourceType.PropertyName" -> "ResourceType_ActualTypeName"
	// This handles cases where the property name differs from the type name.
	key := ctx.currentResource + "." + ctx.currentTypeName + "." + propName
	if typeName, ok := resources.PropertyTypeMap[key]; ok {
		return typeName
	}

	// CloudFormation property types are FLAT - they use the base resource type, not nested type.
	// e.g., Distribution_DistributionConfig has property Logging with type Distribution_Logging
	// NOT Distribution_DistributionConfig_Logging.
	// Extract base resource type from current type name.
	baseResourceType := ctx.currentTypeName
	if idx := strings.Index(ctx.currentTypeName, "_"); idx > 0 {
		baseResourceType = ctx.currentTypeName[:idx]
	}

	// Try flat pattern first: BaseResourceType_PropName
	flatTypeName := baseResourceType + "_" + propName
	fullName := ctx.currentResource + "." + flatTypeName
	if resources.PropertyTypes[fullName] {
		return flatTypeName
	}

	// Fallback: Try nested pattern (currentTypeName_propName) for rare cases
	nestedTypeName := ctx.currentTypeName + "_" + propName
	fullName = ctx.currentResource + "." + nestedTypeName
	if resources.PropertyTypes[fullName] {
		return nestedTypeName
	}

	// Type doesn't exist, fall back to map[string]any
	return ""
}

// getArrayElementTypeName returns the typed struct name for array elements.
// CloudFormation uses singular names for element types: Origins -> Origin
func getArrayElementTypeName(ctx *codegenContext, propName string) string {
	if propName == "" || ctx.currentTypeName == "" {
		return ""
	}

	// Skip known fields that should remain as []any
	skipFields := map[string]bool{
		"Tags": true,
	}
	if skipFields[propName] {
		return ""
	}

	// First, check PropertyTypeMap for the exact mapping.
	// Format: "service.ResourceType.PropertyName" -> "ResourceType_ActualTypeName"
	// This handles array properties where the type name differs from singular property name.
	// e.g., "s3.Bucket.AnalyticsConfigurations" -> "Bucket_AnalyticsConfiguration"
	key := ctx.currentResource + "." + ctx.currentTypeName + "." + propName
	if typeName, ok := resources.PropertyTypeMap[key]; ok {
		return typeName
	}

	singular := singularize(propName)

	// CloudFormation property types are FLAT - they use the base resource type, not nested type.
	// e.g., Distribution_DistributionConfig has property Origins with element type Distribution_Origin
	// NOT Distribution_DistributionConfig_Origin.
	// Extract base resource type from current type name.
	baseResourceType := ctx.currentTypeName
	if idx := strings.Index(ctx.currentTypeName, "_"); idx > 0 {
		baseResourceType = ctx.currentTypeName[:idx]
	}

	// Try flat pattern first: BaseResourceType_SingularPropName
	flatTypeName := baseResourceType + "_" + singular
	fullName := ctx.currentResource + "." + flatTypeName
	if resources.PropertyTypes[fullName] {
		return flatTypeName
	}

	// Fallback: Try nested pattern (currentTypeName_singular) for rare cases
	nestedTypeName := ctx.currentTypeName + "_" + singular
	fullName = ctx.currentResource + "." + nestedTypeName
	if resources.PropertyTypes[fullName] {
		return nestedTypeName
	}

	// Type doesn't exist, fall back to []any
	return ""
}

// singularize converts a plural property name to singular for element types.
// e.g., Origins -> Origin, CacheBehaviors -> CacheBehavior
func singularize(name string) string {
	// Handle common CloudFormation patterns
	if strings.HasSuffix(name, "ies") {
		// e.g., Policies -> Policy
		return name[:len(name)-3] + "y"
	}
	if strings.HasSuffix(name, "sses") {
		// e.g., Addresses -> Address (but keep one 's')
		return name[:len(name)-2]
	}
	if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
		// e.g., Origins -> Origin, but keep Addresses as Address
		return name[:len(name)-1]
	}
	return name
}

// allKeysValidIdentifiers checks if all keys in a map are valid Go identifiers.
// Returns false if any key contains special characters like ':' or starts with a number.
func allKeysValidIdentifiers(m map[string]any) bool {
	for k := range m {
		if !isValidGoIdentifier(k) {
			return false
		}
	}
	return true
}

// isValidGoIdentifier checks if a string is a valid Go identifier.
func isValidGoIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}
	// Also check for Go keywords
	return !isGoKeyword(s)
}

// mapToIntrinsic converts a map with an intrinsic key to an IRIntrinsic.
// Returns nil if the map is not a recognized intrinsic.
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
		// Unknown reference - use sanitized variable name (let Go compiler catch undefined)
		// This avoids generating Ref{} which violates style guidelines
		return sanitizeVarName(target)

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
		// Check for nested attributes (e.g., "Endpoint.Address") or unknown resources
		// These can't use field access pattern:
		// - Nested attributes: AttrRef doesn't have sub-fields
		// - Unknown resources: placeholder is `any` type with no fields
		if strings.Contains(attr, ".") || ctx.unknownResources[logicalID] {
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
func resolveResourceType(cfType string) (module, typeName string) {
	parts := strings.Split(cfType, "::")
	if len(parts) != 3 || parts[0] != "AWS" {
		return "", ""
	}

	service := parts[1]
	resource := parts[2]

	// Map service name to Go module name
	module = strings.ToLower(service)

	typeName = resource

	return module, typeName
}

// topologicalSort returns resources in dependency order (dependencies first).
func topologicalSort(template *IRTemplate) []string {
	// Build dependency graph: node -> list of nodes it depends on
	deps := make(map[string][]string)
	for id := range template.Resources {
		deps[id] = nil
	}
	for source, targets := range template.ReferenceGraph {
		if _, ok := template.Resources[source]; !ok {
			continue
		}
		for _, target := range targets {
			if _, ok := template.Resources[target]; ok {
				// source depends on target
				deps[source] = append(deps[source], target)
			}
		}
	}

	// Kahn's algorithm - compute in-degree (nodes that depend on this one)
	inDegree := make(map[string]int)
	for id := range template.Resources {
		inDegree[id] = 0
	}
	// For each dependency edge, increment the in-degree of the dependent
	for id, idDeps := range deps {
		inDegree[id] = len(idDeps)
	}

	// Start with nodes that have no dependencies (in-degree 0)
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue) // Stable order

	var result []string
	processed := make(map[string]bool)

	for len(queue) > 0 {
		// Take from front
		node := queue[0]
		queue = queue[1:]

		if processed[node] {
			continue
		}
		processed[node] = true
		result = append(result, node)

		// Find nodes that depend on this node
		for id, idDeps := range deps {
			if processed[id] {
				continue
			}
			for _, dep := range idDeps {
				if dep == node {
					inDegree[id]--
					if inDegree[id] == 0 {
						queue = append(queue, id)
					}
					break
				}
			}
		}
		sort.Strings(queue)
	}

	// Handle cycles by adding remaining nodes
	for id := range template.Resources {
		if !processed[id] {
			result = append(result, id)
		}
	}

	return result
}

// sortedKeys returns sorted keys from a map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ToSnakeCase converts PascalCase to snake_case.
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// ToPascalCase converts snake_case to PascalCase.
func ToPascalCase(s string) string {
	words := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(string(word[0])))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

// SanitizeGoName ensures a name is a valid Go identifier.
// Also capitalizes the first letter to ensure the variable is exported.
func SanitizeGoName(name string) string {
	// Remove invalid characters
	var result strings.Builder
	for i, r := range name {
		if i == 0 {
			if unicode.IsLetter(r) || r == '_' {
				// Capitalize first letter for export
				result.WriteRune(unicode.ToUpper(r))
			} else {
				result.WriteRune('_')
			}
		} else {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				result.WriteRune(r)
			}
		}
	}

	s := result.String()
	if s == "" {
		return "_"
	}

	// Check for Go keywords
	if isGoKeyword(s) {
		return s + "_"
	}

	return s
}

// goKeywords and isGoKeyword are defined in parser.go

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
