package importer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/lex00/cloudformation-schema-go/enums"
)

var serviceCategories = map[string]string{
	// Compute
	"EC2":                    "compute",
	"Lambda":                 "compute",
	"ECS":                    "compute",
	"EKS":                    "compute",
	"Batch":                  "compute",
	"AutoScaling":            "compute",
	"ApplicationAutoScaling": "compute",
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
	// Cognito list properties
	"CallbackURLs": true,
	"LogoutURLs":   true,
	// SAM list properties
	"Policies": true,
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

// isListTypeParameter checks if a CloudFormation parameter is a list type.
// List-type parameters include CommaDelimitedList and List<*> types.
func isListTypeParameter(param *IRParameter) bool {
	if param == nil {
		return false
	}
	t := param.Type
	// CommaDelimitedList is a list type
	if t == "CommaDelimitedList" {
		return true
	}
	// List<*> types (e.g., List<Number>, List<String>)
	if strings.HasPrefix(t, "List<") {
		return true
	}
	// AWS::SSM::Parameter::Value<List<*>> types
	if strings.Contains(t, "<List<") {
		return true
	}
	return false
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
