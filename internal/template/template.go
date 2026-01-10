// Package template provides CloudFormation template building from discovered resources.
package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// VarAttrRefInfo mirrors discover.VarAttrRefInfo for AttrRef resolution
type VarAttrRefInfo struct {
	AttrRefs []wetwire.AttrRefUsage
	VarRefs  map[string]string
}

// Builder constructs CloudFormation templates from discovered resources.
type Builder struct {
	resources   map[string]wetwire.DiscoveredResource
	parameters  map[string]wetwire.DiscoveredParameter
	outputs     map[string]wetwire.DiscoveredOutput
	mappings    map[string]wetwire.DiscoveredMapping
	conditions  map[string]wetwire.DiscoveredCondition
	values      map[string]any                // Actual struct values for serialization
	varAttrRefs map[string]VarAttrRefInfo     // For recursive AttrRef resolution
}

// NewBuilder creates a template builder from discovered resources.
func NewBuilder(resources map[string]wetwire.DiscoveredResource) *Builder {
	return &Builder{
		resources:   resources,
		parameters:  make(map[string]wetwire.DiscoveredParameter),
		outputs:     make(map[string]wetwire.DiscoveredOutput),
		mappings:    make(map[string]wetwire.DiscoveredMapping),
		conditions:  make(map[string]wetwire.DiscoveredCondition),
		values:      make(map[string]any),
		varAttrRefs: make(map[string]VarAttrRefInfo),
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
		resources:   resources,
		parameters:  parameters,
		outputs:     outputs,
		mappings:    mappings,
		conditions:  conditions,
		values:      make(map[string]any),
		varAttrRefs: make(map[string]VarAttrRefInfo),
	}
}

// SetVarAttrRefs sets the variable AttrRef info for recursive resolution.
func (b *Builder) SetVarAttrRefs(varAttrRefs map[string]VarAttrRefInfo) {
	b.varAttrRefs = varAttrRefs
}

// resolveAllAttrRefs collects all AttrRefUsages reachable from a variable
// by following all dependencies transitively.
func (b *Builder) resolveAllAttrRefs(varName string) []wetwire.AttrRefUsage {
	visited := make(map[string]bool)
	return b.resolveAllAttrRefsRecursive(varName, visited)
}

func (b *Builder) resolveAllAttrRefsRecursive(varName string, visited map[string]bool) []wetwire.AttrRefUsage {
	if visited[varName] {
		return nil
	}
	visited[varName] = true

	var result []wetwire.AttrRefUsage

	// Get AttrRefs from this variable
	if info, ok := b.varAttrRefs[varName]; ok {
		result = append(result, info.AttrRefs...)

		// Follow VarRefs
		for _, refVarName := range info.VarRefs {
			nested := b.resolveAllAttrRefsRecursive(refVarName, visited)
			result = append(result, nested...)
		}
	}

	// Follow dependencies if it's a resource
	if res, ok := b.resources[varName]; ok {
		for _, depName := range res.Dependencies {
			nested := b.resolveAllAttrRefsRecursive(depName, visited)
			result = append(result, nested...)
		}
	}

	return result
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
		for name, discovered := range b.outputs {
			if val, ok := b.values[name]; ok {
				output := b.serializeOutput(name, val, discovered)
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
func (b *Builder) serializeOutput(name string, value any, discovered wetwire.DiscoveredOutput) wetwire.Output {
	valMap, ok := value.(map[string]any)
	if !ok {
		return wetwire.Output{}
	}

	// Build a lookup map from field path to AttrRefUsage
	attrRefsByPath := make(map[string]wetwire.AttrRefUsage)
	for _, usage := range discovered.AttrRefUsages {
		attrRefsByPath[usage.FieldPath] = usage
	}

	output := wetwire.Output{}

	if desc, ok := valMap["Description"].(string); ok {
		output.Description = desc
	}
	if val, ok := valMap["Value"]; ok {
		// Apply AttrRef fix to the Value field
		output.Value = b.transformValueWithPath(val, "Value", attrRefsByPath)
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
// It also fixes empty GetAtt references using AttrRefUsages from discovery.
func (b *Builder) transformRefs(name string, props map[string]any, res wetwire.DiscoveredResource) map[string]any {
	// Build a lookup map from field path to AttrRefUsage
	// Use recursive resolution to get AttrRefs from all transitively reachable variables
	attrRefsByPath := make(map[string]wetwire.AttrRefUsage)
	resolvedAttrRefs := b.resolveAllAttrRefs(name)
	for _, usage := range resolvedAttrRefs {
		attrRefsByPath[usage.FieldPath] = usage
	}

	result := make(map[string]any)
	for key, value := range props {
		result[key] = b.transformValueWithPath(value, key, attrRefsByPath)
	}

	return result
}

// intrinsicFieldNames maps CloudFormation intrinsic function keys to their Go field names.
// This allows path matching to work when AttrRefs are used inside intrinsic functions.
// The slice index corresponds to the position in the intrinsic's array value.
var intrinsicFieldNames = map[string][]string{
	"Fn::Join":        {"Delimiter", "Values"},
	"Fn::Sub":         {"String"},            // SubWithMap has {"String", "Variables"}
	"Fn::Select":      {"Index", "List"},
	"Fn::If":          {"Condition", "TrueValue", "FalseValue"},
	"Fn::GetAZs":      {"Region"},
	"Fn::Split":       {"Delimiter", "String"},
	"Fn::Base64":      {"String"},
	"Fn::Cidr":        {"IpBlock", "Count", "CidrBits"},
	"Fn::ImportValue": {"Name"},
	"Fn::FindInMap":   {"MapName", "TopLevelKey", "SecondLevelKey"},
	"Fn::Transform":   {"Name", "Parameters"},
	"Fn::Equals":      {"Left", "Right"},
	"Fn::And":         {"Conditions"},
	"Fn::Or":          {"Conditions"},
	"Fn::Not":         {"Condition"},
}

// stripArrayIndices removes array indices from a path for fuzzy matching.
// e.g., "Policies[0].PolicyDocument.Statement[1].Resource[0]" -> "Policies.PolicyDocument.Statement.Resource"
func stripArrayIndices(path string) string {
	result := make([]byte, 0, len(path))
	i := 0
	for i < len(path) {
		if path[i] == '[' {
			// Skip until closing bracket
			for i < len(path) && path[i] != ']' {
				i++
			}
			if i < len(path) {
				i++ // skip ']'
			}
		} else {
			result = append(result, path[i])
			i++
		}
	}
	return string(result)
}

func (b *Builder) transformValueWithPath(value any, path string, attrRefsByPath map[string]wetwire.AttrRefUsage) any {
	return b.transformValueWithPathAndContext(value, path, attrRefsByPath, "")
}

// transformValueWithPathAndContext is the internal implementation that tracks intrinsic context.
// intrinsicKey is the current intrinsic function key (e.g., "Fn::Join") if we're inside one.
func (b *Builder) transformValueWithPathAndContext(value any, path string, attrRefsByPath map[string]wetwire.AttrRefUsage, intrinsicKey string) any {
	switch v := value.(type) {
	case map[string]any:
		// Check if this is a GetAtt with empty resource name
		if getAtt, ok := v["Fn::GetAtt"]; ok {
			if arr, isArr := getAtt.([]any); isArr && len(arr) >= 2 {
				resourceName, _ := arr[0].(string)
				if resourceName == "" {
					// Look up the AttrRefUsage for this path
					// First try exact match
					if usage, found := attrRefsByPath[path]; found {
						return map[string]any{
							"Fn::GetAtt": []string{usage.ResourceName, usage.Attribute},
						}
					}
					// Then try matching with stripped array indices
					strippedPath := stripArrayIndices(path)
					if usage, found := attrRefsByPath[strippedPath]; found {
						return map[string]any{
							"Fn::GetAtt": []string{usage.ResourceName, usage.Attribute},
						}
					}
					// Try suffix matching - find any AttrRefUsage whose path is a suffix of strippedPath
					for refPath, usage := range attrRefsByPath {
						if strings.HasSuffix(strippedPath, "."+refPath) || strippedPath == refPath {
							return map[string]any{
								"Fn::GetAtt": []string{usage.ResourceName, usage.Attribute},
							}
						}
					}
				}
			}
			return v
		}

		// Check if this is already an intrinsic function
		if _, ok := v["Ref"]; ok {
			return v
		}
		if _, ok := v["Fn::Sub"]; ok {
			return v
		}

		// Recursively transform map values
		result := make(map[string]any)
		for key, val := range v {
			newPath := path + "." + key
			// If this is an intrinsic function key, pass it down to handle array field mapping
			if _, isIntrinsic := intrinsicFieldNames[key]; isIntrinsic {
				result[key] = b.transformValueWithPathAndContext(val, newPath, attrRefsByPath, key)
			} else {
				result[key] = b.transformValueWithPathAndContext(val, newPath, attrRefsByPath, "")
			}
		}
		return result

	case []any:
		// Recursively transform slice elements
		result := make([]any, len(v))
		for i, elem := range v {
			elemPath := path
			// If we're inside an intrinsic function, use the Go field name for this position
			if intrinsicKey != "" {
				if fieldNames, ok := intrinsicFieldNames[intrinsicKey]; ok && i < len(fieldNames) {
					// Replace the intrinsic key in path with the Go field name
					// e.g., "Value.Fn::Join" + index 1 -> "Value.Values"
					if idx := strings.LastIndex(path, "."+intrinsicKey); idx >= 0 {
						elemPath = path[:idx+1] + fieldNames[i]
					} else if path == intrinsicKey {
						elemPath = fieldNames[i]
					}
				}
			}
			result[i] = b.transformValueWithPathAndContext(elem, elemPath, attrRefsByPath, "")
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
// e.g., "s3.Bucket" -> "AWS::S3::Bucket", "cloudfront.Distribution" -> "AWS::CloudFront::Distribution"
func cfResourceType(goType string) string {
	// Parse the Go type: "package.Type"
	parts := strings.SplitN(goType, ".", 2)
	if len(parts) != 2 {
		return ""
	}

	pkgName := parts[0]
	typeName := parts[1]

	// Map Go package names to CloudFormation service names
	serviceName := goPackageToCFService(pkgName)
	if serviceName == "" {
		return ""
	}

	return "AWS::" + serviceName + "::" + typeName
}

// goPackageToCFService maps Go package names to CloudFormation service names.
// This handles the case transformations needed for proper CloudFormation types.
func goPackageToCFService(pkg string) string {
	// Direct mappings for packages that don't follow simple title-casing
	directMap := map[string]string{
		"accessanalyzer":            "AccessAnalyzer",
		"acmpca":                    "ACMPCA",
		"aiops":                     "AIOps",
		"amazonmq":                  "AmazonMQ",
		"amplify":                   "Amplify",
		"amplifyuibuilder":          "AmplifyUIBuilder",
		"apigateway":                "ApiGateway",
		"apigatewayv2":              "ApiGatewayV2",
		"appconfig":                 "AppConfig",
		"appflow":                   "AppFlow",
		"appintegrations":           "AppIntegrations",
		"applicationautoscaling":    "ApplicationAutoScaling",
		"applicationinsights":       "ApplicationInsights",
		"applicationsignals":        "ApplicationSignals",
		"appmesh":                   "AppMesh",
		"apprunner":                 "AppRunner",
		"appstream":                 "AppStream",
		"appsync":                   "AppSync",
		"apptest":                   "AppTest",
		"aps":                       "APS",
		"arcregionswitch":           "ARCRegionSwitch",
		"arczonalshift":             "ARCZonalShift",
		"ask":                       "ASK",
		"athena":                    "Athena",
		"auditmanager":              "AuditManager",
		"autoscaling":               "AutoScaling",
		"autoscalingplans":          "AutoScalingPlans",
		"b2bi":                      "B2BI",
		"backup":                    "Backup",
		"backupgateway":             "BackupGateway",
		"batch":                     "Batch",
		"bcmdataexports":            "BCMDataExports",
		"bedrock":                   "Bedrock",
		"bedrockagentcore":          "BedrockAgentCore",
		"billing":                   "Billing",
		"billingconductor":          "BillingConductor",
		"budgets":                   "Budgets",
		"cases":                     "Cases",
		"cassandra":                 "Cassandra",
		"ce":                        "CE",
		"certificatemanager":        "CertificateManager",
		"chatbot":                   "Chatbot",
		"cleanrooms":                "CleanRooms",
		"cleanroomsml":              "CleanRoomsML",
		"cloud9":                    "Cloud9",
		"cloudformation":            "CloudFormation",
		"cloudfront":                "CloudFront",
		"cloudtrail":                "CloudTrail",
		"cloudwatch":                "CloudWatch",
		"codeartifact":              "CodeArtifact",
		"codebuild":                 "CodeBuild",
		"codecommit":                "CodeCommit",
		"codeconnections":           "CodeConnections",
		"codedeploy":                "CodeDeploy",
		"codeguruprofiler":          "CodeGuruProfiler",
		"codegurureviewer":          "CodeGuruReviewer",
		"codepipeline":              "CodePipeline",
		"codestar":                  "CodeStar",
		"codestarconnections":       "CodeStarConnections",
		"codestarnotifications":     "CodeStarNotifications",
		"cognito":                   "Cognito",
		"comprehend":                "Comprehend",
		"config":                    "Config",
		"connect":                   "Connect",
		"connectcampaigns":          "ConnectCampaigns",
		"connectcampaignsv2":        "ConnectCampaignsV2",
		"controltower":              "ControlTower",
		"cur":                       "CUR",
		"customerprofiles":          "CustomerProfiles",
		"databrew":                  "DataBrew",
		"datapipeline":              "DataPipeline",
		"datasync":                  "DataSync",
		"datazone":                  "DataZone",
		"dax":                       "DAX",
		"deadline":                  "Deadline",
		"detective":                 "Detective",
		"devopsagent":               "DevOpsAgent",
		"devopsguru":                "DevOpsGuru",
		"directoryservice":          "DirectoryService",
		"dlm":                       "DLM",
		"dms":                       "DMS",
		"docdb":                     "DocDB",
		"docdbelastic":              "DocDBElastic",
		"dsql":                      "DSQL",
		"dynamodb":                  "DynamoDB",
		"ec2":                       "EC2",
		"ecr":                       "ECR",
		"ecs":                       "ECS",
		"efs":                       "EFS",
		"eks":                       "EKS",
		"elasticache":               "ElastiCache",
		"elasticbeanstalk":          "ElasticBeanstalk",
		"elasticloadbalancing":      "ElasticLoadBalancing",
		"elasticloadbalancingv2":    "ElasticLoadBalancingV2",
		"elasticsearch":             "Elasticsearch",
		"emr":                       "EMR",
		"emrcontainers":             "EMRContainers",
		"emrserverless":             "EMRServerless",
		"entityresolution":          "EntityResolution",
		"events":                    "Events",
		"eventschemas":              "EventSchemas",
		"evidently":                 "Evidently",
		"evs":                       "EVS",
		"finspace":                  "FinSpace",
		"fis":                       "FIS",
		"fms":                       "FMS",
		"forecast":                  "Forecast",
		"frauddetector":             "FraudDetector",
		"fsx":                       "FSx",
		"gamelift":                  "GameLift",
		"globalaccelerator":         "GlobalAccelerator",
		"glue":                      "Glue",
		"grafana":                   "Grafana",
		"greengrass":                "Greengrass",
		"greengrassv2":              "GreengrassV2",
		"groundstation":             "GroundStation",
		"guardduty":                 "GuardDuty",
		"healthimaging":             "HealthImaging",
		"healthlake":                "HealthLake",
		"iam":                       "IAM",
		"identitystore":             "IdentityStore",
		"imagebuilder":              "ImageBuilder",
		"inspector":                 "Inspector",
		"inspectorv2":               "InspectorV2",
		"internetmonitor":           "InternetMonitor",
		"invoicing":                 "Invoicing",
		"iot":                       "IoT",
		"iotanalytics":              "IoTAnalytics",
		"iotcoredeviceadvisor":      "IoTCoreDeviceAdvisor",
		"iotevents":                 "IoTEvents",
		"iotfleetwise":              "IoTFleetWise",
		"iotsitewise":               "IoTSiteWise",
		"iotthingsgraph":            "IoTThingsGraph",
		"iottwinmaker":              "IoTTwinMaker",
		"iotwireless":               "IoTWireless",
		"ivs":                       "IVS",
		"ivschat":                   "IVSChat",
		"kafkaconnect":              "KafkaConnect",
		"kendra":                    "Kendra",
		"kendraranking":             "KendraRanking",
		"kinesis":                   "Kinesis",
		"kinesisanalytics":          "KinesisAnalytics",
		"kinesisanalyticsv2":        "KinesisAnalyticsV2",
		"kinesisfirehose":           "KinesisFirehose",
		"kinesisvideo":              "KinesisVideo",
		"kms":                       "KMS",
		"lakeformation":             "LakeFormation",
		"lambda":                    "Lambda",
		"launchwizard":              "LaunchWizard",
		"lex":                       "Lex",
		"licensemanager":            "LicenseManager",
		"lightsail":                 "Lightsail",
		"location":                  "Location",
		"logs":                      "Logs",
		"lookoutequipment":          "LookoutEquipment",
		"lookoutvision":             "LookoutVision",
		"m2":                        "M2",
		"macie":                     "Macie",
		"managedblockchain":         "ManagedBlockchain",
		"mediaconnect":              "MediaConnect",
		"mediaconvert":              "MediaConvert",
		"medialive":                 "MediaLive",
		"mediapackage":              "MediaPackage",
		"mediapackagev2":            "MediaPackageV2",
		"mediastore":                "MediaStore",
		"mediatailor":               "MediaTailor",
		"memorydb":                  "MemoryDB",
		"mpa":                       "MPA",
		"msk":                       "MSK",
		"mwaa":                      "MWAA",
		"neptune":                   "Neptune",
		"neptunegraph":              "NeptuneGraph",
		"networkfirewall":           "NetworkFirewall",
		"networkmanager":            "NetworkManager",
		"notifications":             "Notifications",
		"notificationscontacts":     "NotificationsContacts",
		"oam":                       "Oam",
		"observabilityadmin":        "ObservabilityAdmin",
		"odb":                       "ODB",
		"omics":                     "Omics",
		"opensearchserverless":      "OpenSearchServerless",
		"opensearchservice":         "OpenSearchService",
		"opsworks":                  "OpsWorks",
		"organizations":             "Organizations",
		"osis":                      "OSIS",
		"panorama":                  "Panorama",
		"paymentcryptography":       "PaymentCryptography",
		"pcaconnectorad":            "PCAConnectorAD",
		"pcaconnectorscep":          "PCAConnectorSCEP",
		"pcs":                       "PCS",
		"personalize":               "Personalize",
		"pinpoint":                  "Pinpoint",
		"pinpointemail":             "PinpointEmail",
		"pipes":                     "Pipes",
		"proton":                    "Proton",
		"qbusiness":                 "QBusiness",
		"qldb":                      "QLDB",
		"quicksight":                "QuickSight",
		"ram":                       "RAM",
		"rbin":                      "Rbin",
		"rds":                       "RDS",
		"redshift":                  "Redshift",
		"redshiftserverless":        "RedshiftServerless",
		"refactorspaces":            "RefactorSpaces",
		"rekognition":               "Rekognition",
		"resiliencehub":             "ResilienceHub",
		"resourceexplorer2":         "ResourceExplorer2",
		"resourcegroups":            "ResourceGroups",
		"robomaker":                 "RoboMaker",
		"rolesanywhere":             "RolesAnywhere",
		"route53":                   "Route53",
		"route53profiles":           "Route53Profiles",
		"route53recoverycontrol":    "Route53RecoveryControl",
		"route53recoveryreadiness":  "Route53RecoveryReadiness",
		"route53resolver":           "Route53Resolver",
		"rtbfabric":                 "RTBFabric",
		"rum":                       "RUM",
		"s3":                        "S3",
		"s3express":                 "S3Express",
		"s3objectlambda":            "S3ObjectLambda",
		"s3outposts":                "S3Outposts",
		"s3tables":                  "S3Tables",
		"s3vectors":                 "S3Vectors",
		"sagemaker":                 "SageMaker",
		"scheduler":                 "Scheduler",
		"sdb":                       "SDB",
		"secretsmanager":            "SecretsManager",
		"securityhub":               "SecurityHub",
		"securitylake":              "SecurityLake",
		"serverless":                "Serverless",
		"servicecatalog":            "ServiceCatalog",
		"servicecatalogappregistry": "ServiceCatalogAppRegistry",
		"servicediscovery":          "ServiceDiscovery",
		"ses":                       "SES",
		"shield":                    "Shield",
		"signer":                    "Signer",
		"simspaceweaver":            "SimSpaceWeaver",
		"smsvoice":                  "SMSVoice",
		"sns":                       "SNS",
		"sqs":                       "SQS",
		"ssm":                       "SSM",
		"ssmcontacts":               "SSMContacts",
		"ssmguiconnect":             "SSMGuiConnect",
		"ssmincidents":              "SSMIncidents",
		"ssmquicksetup":             "SSMQuickSetup",
		"sso":                       "SSO",
		"stepfunctions":             "StepFunctions",
		"supportapp":                "SupportApp",
		"synthetics":                "Synthetics",
		"systemsmanagersap":         "SystemsManagerSAP",
		"timestream":                "Timestream",
		"transfer":                  "Transfer",
		"verifiedpermissions":       "VerifiedPermissions",
		"voiceid":                   "VoiceID",
		"vpclattice":                "VpcLattice",
		"waf":                       "WAF",
		"wafregional":               "WAFRegional",
		"wafv2":                     "WAFv2",
		"wisdom":                    "Wisdom",
		"workspaces":                "WorkSpaces",
		"workspacesinstances":       "WorkSpacesInstances",
		"workspacesthinclient":      "WorkSpacesThinClient",
		"workspacesweb":             "WorkSpacesWeb",
		"xray":                      "XRay",
	}

	if service, ok := directMap[pkg]; ok {
		return service
	}
	return ""
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
