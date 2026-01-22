package main

// SAMSpec defines the AWS Serverless Application Model (SAM) resource types.
// SAM resources are not part of the CloudFormation spec, so we define them statically.
// Reference: https://github.com/aws/serverless-application-model
var SAMSpec = &Service{
	Name:     "serverless",
	CFPrefix: "AWS::Serverless",
	Resources: map[string]ParsedResource{
		"Function":     samFunction,
		"Api":          samApi,
		"HttpApi":      samHttpApi,
		"SimpleTable":  samSimpleTable,
		"LayerVersion": samLayerVersion,
		"StateMachine": samStateMachine,
		"Application":  samApplication,
		"Connector":    samConnector,
		"GraphQLApi":   samGraphQLApi,
	},
	PropertyTypes: map[string]ParsedPropertyType{
		// Function property types
		"Function_Environment":          samFunctionEnvironment,
		"Function_EventSource":          samFunctionEventSource,
		"Function_VpcConfig":            samFunctionVpcConfig,
		"Function_DeadLetterQueue":      samFunctionDeadLetterQueue,
		"Function_DeploymentPreference": samFunctionDeploymentPreference,
		"Function_S3Location":           samFunctionS3Location,
		// Api property types
		"Api_Auth":              samApiAuth,
		"Api_CorsConfiguration": samApiCorsConfiguration,
		// HttpApi property types
		"HttpApi_CorsConfiguration": samHttpApiCorsConfiguration,
		// SimpleTable property types
		"SimpleTable_PrimaryKey": samSimpleTablePrimaryKey,
		// StateMachine property types
		"StateMachine_S3Location": samStateMachineS3Location,
	},
}

// AWS::Serverless::Function
var samFunction = ParsedResource{
	Name:          "Function",
	CFType:        "AWS::Serverless::Function",
	Documentation: "Creates a Lambda function, IAM execution role, and event source mappings.",
	Properties: map[string]ParsedProperty{
		"Handler": {
			Name:          "Handler",
			GoType:        "any",
			Required:      true,
			Documentation: "Function handler. Required for Zip package type.",
		},
		"Runtime": {
			Name:          "Runtime",
			GoType:        "any",
			Required:      true,
			Documentation: "Runtime environment. Required for Zip package type.",
		},
		"CodeUri": {
			Name:          "CodeUri",
			GoType:        "any",
			Required:      true,
			Documentation: "S3 URI, local path, or S3Location object.",
		},
		"FunctionName": {
			Name:          "FunctionName",
			GoType:        "any",
			Documentation: "Name of the Lambda function.",
		},
		"Description": {
			Name:          "Description",
			GoType:        "any",
			Documentation: "Description of the function.",
		},
		"MemorySize": {
			Name:          "MemorySize",
			GoType:        "any",
			Documentation: "Memory size in MB (128-10240).",
		},
		"Timeout": {
			Name:          "Timeout",
			GoType:        "any",
			Documentation: "Function timeout in seconds (1-900).",
		},
		"Role": {
			Name:          "Role",
			GoType:        "any",
			Documentation: "ARN of IAM role. If not specified, one is created.",
		},
		"Policies": {
			Name:          "Policies",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "IAM policies to attach to the function role.",
		},
		"Environment": {
			Name:          "Environment",
			GoType:        "Function_Environment",
			IsPointer:     true,
			Documentation: "Environment variables.",
		},
		"Events": {
			Name:          "Events",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Event sources that trigger the function.",
		},
		"VpcConfig": {
			Name:          "VpcConfig",
			GoType:        "Function_VpcConfig",
			IsPointer:     true,
			Documentation: "VPC configuration.",
		},
		"Architectures": {
			Name:          "Architectures",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Instruction set architecture (x86_64 or arm64).",
		},
		"Layers": {
			Name:          "Layers",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "List of layer ARNs.",
		},
		"Tracing": {
			Name:          "Tracing",
			GoType:        "any",
			Documentation: "X-Ray tracing mode (Active or PassThrough).",
		},
		"Tags": {
			Name:          "Tags",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Tags to apply to the function.",
		},
		"DeadLetterQueue": {
			Name:          "DeadLetterQueue",
			GoType:        "Function_DeadLetterQueue",
			IsPointer:     true,
			Documentation: "Dead letter queue configuration.",
		},
		"DeploymentPreference": {
			Name:          "DeploymentPreference",
			GoType:        "Function_DeploymentPreference",
			IsPointer:     true,
			Documentation: "Deployment preference for gradual deployments.",
		},
		"ReservedConcurrentExecutions": {
			Name:          "ReservedConcurrentExecutions",
			GoType:        "any",
			Documentation: "Reserved concurrent executions.",
		},
		"AutoPublishAlias": {
			Name:          "AutoPublishAlias",
			GoType:        "any",
			Documentation: "Alias name for automatic publishing.",
		},
		"PackageType": {
			Name:          "PackageType",
			GoType:        "any",
			Documentation: "Package type (Zip or Image).",
		},
		"ImageUri": {
			Name:          "ImageUri",
			GoType:        "any",
			Documentation: "URI of container image (for Image package type).",
		},
		"ImageConfig": {
			Name:          "ImageConfig",
			GoType:        "any",
			Documentation: "Container image configuration.",
		},
		"EphemeralStorage": {
			Name:          "EphemeralStorage",
			GoType:        "any",
			Documentation: "Ephemeral storage size (512-10240 MB).",
		},
		"SnapStart": {
			Name:          "SnapStart",
			GoType:        "any",
			Documentation: "SnapStart configuration.",
		},
		"FunctionUrlConfig": {
			Name:          "FunctionUrlConfig",
			GoType:        "any",
			Documentation: "Function URL configuration.",
		},
		"InlineCode": {
			Name:          "InlineCode",
			GoType:        "any",
			Documentation: "Inline code for the function (alternative to CodeUri).",
		},
	},
	Attributes: map[string]ParsedAttribute{
		"Arn": {Name: "Arn", GoType: "any"},
	},
}

// AWS::Serverless::Api
var samApi = ParsedResource{
	Name:          "Api",
	CFType:        "AWS::Serverless::Api",
	Documentation: "Creates an API Gateway REST API.",
	Properties: map[string]ParsedProperty{
		"StageName": {
			Name:          "StageName",
			GoType:        "any",
			Required:      true,
			Documentation: "Name of the stage.",
		},
		"DefinitionBody": {
			Name:          "DefinitionBody",
			GoType:        "any",
			Documentation: "OpenAPI specification inline.",
		},
		"DefinitionUri": {
			Name:          "DefinitionUri",
			GoType:        "any",
			Documentation: "S3 URI or local path to OpenAPI spec.",
		},
		"Name": {
			Name:          "Name",
			GoType:        "any",
			Documentation: "Name of the API.",
		},
		"Auth": {
			Name:          "Auth",
			GoType:        "Api_Auth",
			IsPointer:     true,
			Documentation: "Authentication configuration.",
		},
		"Cors": {
			Name:          "Cors",
			GoType:        "any",
			Documentation: "CORS configuration (string or CorsConfiguration).",
		},
		"EndpointConfiguration": {
			Name:          "EndpointConfiguration",
			GoType:        "any",
			Documentation: "Endpoint type (REGIONAL, EDGE, PRIVATE).",
		},
		"TracingEnabled": {
			Name:          "TracingEnabled",
			GoType:        "any",
			Documentation: "Enable X-Ray tracing.",
		},
		"CacheClusterEnabled": {
			Name:          "CacheClusterEnabled",
			GoType:        "any",
			Documentation: "Enable API caching.",
		},
		"CacheClusterSize": {
			Name:          "CacheClusterSize",
			GoType:        "any",
			Documentation: "Cache cluster size.",
		},
		"Variables": {
			Name:          "Variables",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Stage variables.",
		},
		"Tags": {
			Name:          "Tags",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Tags for the API.",
		},
		"AccessLogSetting": {
			Name:          "AccessLogSetting",
			GoType:        "any",
			Documentation: "Access logging configuration.",
		},
		"CanarySetting": {
			Name:          "CanarySetting",
			GoType:        "any",
			Documentation: "Canary deployment configuration.",
		},
		"MethodSettings": {
			Name:          "MethodSettings",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Method-level settings.",
		},
		"BinaryMediaTypes": {
			Name:          "BinaryMediaTypes",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Binary media types.",
		},
		"MinimumCompressionSize": {
			Name:          "MinimumCompressionSize",
			GoType:        "any",
			Documentation: "Minimum response size for compression.",
		},
		"OpenApiVersion": {
			Name:          "OpenApiVersion",
			GoType:        "any",
			Documentation: "OpenAPI version (2.0 or 3.0).",
		},
		"GatewayResponses": {
			Name:          "GatewayResponses",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Gateway response configurations.",
		},
	},
	Attributes: map[string]ParsedAttribute{
		"RootResourceId": {Name: "RootResourceId", GoType: "any"},
	},
}

// AWS::Serverless::HttpApi
var samHttpApi = ParsedResource{
	Name:          "HttpApi",
	CFType:        "AWS::Serverless::HttpApi",
	Documentation: "Creates an API Gateway HTTP API (v2).",
	Properties: map[string]ParsedProperty{
		"StageName": {
			Name:          "StageName",
			GoType:        "any",
			Documentation: "Name of the stage (default: $default).",
		},
		"DefinitionBody": {
			Name:          "DefinitionBody",
			GoType:        "any",
			Documentation: "OpenAPI specification inline.",
		},
		"DefinitionUri": {
			Name:          "DefinitionUri",
			GoType:        "any",
			Documentation: "S3 URI or local path to OpenAPI spec.",
		},
		"Name": {
			Name:          "Name",
			GoType:        "any",
			Documentation: "Name of the HTTP API.",
		},
		"CorsConfiguration": {
			Name:          "CorsConfiguration",
			GoType:        "any",
			Documentation: "CORS configuration (bool or CorsConfiguration).",
		},
		"Auth": {
			Name:          "Auth",
			GoType:        "any",
			Documentation: "Authentication configuration.",
		},
		"AccessLogSettings": {
			Name:          "AccessLogSettings",
			GoType:        "any",
			Documentation: "Access logging configuration.",
		},
		"DefaultRouteSettings": {
			Name:          "DefaultRouteSettings",
			GoType:        "any",
			Documentation: "Default route settings.",
		},
		"RouteSettings": {
			Name:          "RouteSettings",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Route-specific settings.",
		},
		"StageVariables": {
			Name:          "StageVariables",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Stage variables.",
		},
		"Tags": {
			Name:          "Tags",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Tags for the HTTP API.",
		},
		"FailOnWarnings": {
			Name:          "FailOnWarnings",
			GoType:        "any",
			Documentation: "Fail if warnings during import.",
		},
		"DisableExecuteApiEndpoint": {
			Name:          "DisableExecuteApiEndpoint",
			GoType:        "any",
			Documentation: "Disable default execute-api endpoint.",
		},
	},
	Attributes: map[string]ParsedAttribute{
		"ApiEndpoint": {Name: "ApiEndpoint", GoType: "any"},
	},
}

// AWS::Serverless::SimpleTable
var samSimpleTable = ParsedResource{
	Name:          "SimpleTable",
	CFType:        "AWS::Serverless::SimpleTable",
	Documentation: "Creates a DynamoDB table with a single primary key.",
	Properties: map[string]ParsedProperty{
		"PrimaryKey": {
			Name:          "PrimaryKey",
			GoType:        "SimpleTable_PrimaryKey",
			IsPointer:     true,
			Documentation: "Primary key configuration.",
		},
		"ProvisionedThroughput": {
			Name:          "ProvisionedThroughput",
			GoType:        "any",
			Documentation: "Provisioned throughput settings.",
		},
		"TableName": {
			Name:          "TableName",
			GoType:        "any",
			Documentation: "Name of the table.",
		},
		"Tags": {
			Name:          "Tags",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Tags for the table.",
		},
		"SSESpecification": {
			Name:          "SSESpecification",
			GoType:        "any",
			Documentation: "Server-side encryption configuration.",
		},
	},
	Attributes: map[string]ParsedAttribute{},
}

// AWS::Serverless::LayerVersion
var samLayerVersion = ParsedResource{
	Name:          "LayerVersion",
	CFType:        "AWS::Serverless::LayerVersion",
	Documentation: "Creates a Lambda layer version.",
	Properties: map[string]ParsedProperty{
		"ContentUri": {
			Name:          "ContentUri",
			GoType:        "any",
			Required:      true,
			Documentation: "S3 URI or local path to layer content.",
		},
		"LayerName": {
			Name:          "LayerName",
			GoType:        "any",
			Documentation: "Name of the layer.",
		},
		"Description": {
			Name:          "Description",
			GoType:        "any",
			Documentation: "Description of the layer.",
		},
		"CompatibleRuntimes": {
			Name:          "CompatibleRuntimes",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Compatible runtimes.",
		},
		"CompatibleArchitectures": {
			Name:          "CompatibleArchitectures",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Compatible architectures.",
		},
		"LicenseInfo": {
			Name:          "LicenseInfo",
			GoType:        "any",
			Documentation: "License information.",
		},
		"RetentionPolicy": {
			Name:          "RetentionPolicy",
			GoType:        "any",
			Documentation: "Retention policy (Retain or Delete).",
		},
	},
	Attributes: map[string]ParsedAttribute{
		"Arn":      {Name: "Arn", GoType: "any"},
		"LayerArn": {Name: "LayerArn", GoType: "any"},
	},
}

// AWS::Serverless::StateMachine
var samStateMachine = ParsedResource{
	Name:          "StateMachine",
	CFType:        "AWS::Serverless::StateMachine",
	Documentation: "Creates a Step Functions state machine.",
	Properties: map[string]ParsedProperty{
		"Definition": {
			Name:          "Definition",
			GoType:        "any",
			Documentation: "State machine definition (inline).",
		},
		"DefinitionUri": {
			Name:          "DefinitionUri",
			GoType:        "any",
			Documentation: "S3 URI or local path to definition.",
		},
		"DefinitionSubstitutions": {
			Name:          "DefinitionSubstitutions",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Substitutions for definition.",
		},
		"Name": {
			Name:          "Name",
			GoType:        "any",
			Documentation: "Name of the state machine.",
		},
		"Role": {
			Name:          "Role",
			GoType:        "any",
			Documentation: "IAM role ARN.",
		},
		"Policies": {
			Name:          "Policies",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "IAM policies.",
		},
		"Type": {
			Name:          "Type",
			GoType:        "any",
			Documentation: "State machine type (STANDARD or EXPRESS).",
		},
		"Logging": {
			Name:          "Logging",
			GoType:        "any",
			Documentation: "Logging configuration.",
		},
		"Tracing": {
			Name:          "Tracing",
			GoType:        "any",
			Documentation: "X-Ray tracing configuration.",
		},
		"Events": {
			Name:          "Events",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Event sources.",
		},
		"Tags": {
			Name:          "Tags",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Tags for the state machine.",
		},
		"PermissionsBoundary": {
			Name:          "PermissionsBoundary",
			GoType:        "any",
			Documentation: "Permissions boundary ARN.",
		},
	},
	Attributes: map[string]ParsedAttribute{
		"Arn":  {Name: "Arn", GoType: "any"},
		"Name": {Name: "Name", GoType: "any"},
	},
}

// AWS::Serverless::Application
var samApplication = ParsedResource{
	Name:          "Application",
	CFType:        "AWS::Serverless::Application",
	Documentation: "Deploys a nested serverless application from SAR or S3.",
	Properties: map[string]ParsedProperty{
		"Location": {
			Name:          "Location",
			GoType:        "any",
			Required:      true,
			Documentation: "SAR application ID or S3 location.",
		},
		"Parameters": {
			Name:          "Parameters",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Parameters to pass to the nested application.",
		},
		"NotificationArns": {
			Name:          "NotificationArns",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "SNS topic ARNs for stack notifications.",
		},
		"Tags": {
			Name:          "Tags",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Tags for the application.",
		},
		"TimeoutInMinutes": {
			Name:          "TimeoutInMinutes",
			GoType:        "any",
			Documentation: "Stack creation timeout.",
		},
	},
	Attributes: map[string]ParsedAttribute{
		"Outputs": {Name: "Outputs", GoType: "any"},
	},
}

// AWS::Serverless::Connector
var samConnector = ParsedResource{
	Name:          "Connector",
	CFType:        "AWS::Serverless::Connector",
	Documentation: "Creates permissions between two resources.",
	Properties: map[string]ParsedProperty{
		"Source": {
			Name:          "Source",
			GoType:        "any",
			Required:      true,
			Documentation: "Source resource reference.",
		},
		"Destination": {
			Name:          "Destination",
			GoType:        "any",
			Required:      true,
			Documentation: "Destination resource reference.",
		},
		"Permissions": {
			Name:          "Permissions",
			GoType:        "[]any",
			IsList:        true,
			Required:      true,
			Documentation: "Permission types (Read, Write).",
		},
	},
	Attributes: map[string]ParsedAttribute{},
}

// AWS::Serverless::GraphQLApi
var samGraphQLApi = ParsedResource{
	Name:          "GraphQLApi",
	CFType:        "AWS::Serverless::GraphQLApi",
	Documentation: "Creates an AWS AppSync GraphQL API.",
	Properties: map[string]ParsedProperty{
		"SchemaUri": {
			Name:          "SchemaUri",
			GoType:        "any",
			Documentation: "S3 URI or local path to GraphQL schema.",
		},
		"SchemaInline": {
			Name:          "SchemaInline",
			GoType:        "any",
			Documentation: "Inline GraphQL schema.",
		},
		"Name": {
			Name:          "Name",
			GoType:        "any",
			Documentation: "Name of the GraphQL API.",
		},
		"Auth": {
			Name:          "Auth",
			GoType:        "any",
			Required:      true,
			Documentation: "Authentication configuration.",
		},
		"DataSources": {
			Name:          "DataSources",
			GoType:        "any",
			Documentation: "Data source configurations.",
		},
		"Functions": {
			Name:          "Functions",
			GoType:        "any",
			Documentation: "AppSync function configurations.",
		},
		"Resolvers": {
			Name:          "Resolvers",
			GoType:        "any",
			Documentation: "Resolver configurations.",
		},
		"Logging": {
			Name:          "Logging",
			GoType:        "any",
			Documentation: "Logging configuration.",
		},
		"XrayEnabled": {
			Name:          "XrayEnabled",
			GoType:        "any",
			Documentation: "Enable X-Ray tracing.",
		},
		"Tags": {
			Name:          "Tags",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Tags for the GraphQL API.",
		},
		"Cache": {
			Name:          "Cache",
			GoType:        "any",
			Documentation: "Caching configuration.",
		},
		"DomainName": {
			Name:          "DomainName",
			GoType:        "any",
			Documentation: "Custom domain configuration.",
		},
	},
	Attributes: map[string]ParsedAttribute{
		"ApiId":       {Name: "ApiId", GoType: "any"},
		"Arn":         {Name: "Arn", GoType: "any"},
		"GraphQLUrl":  {Name: "GraphQLUrl", GoType: "any"},
		"GraphQLDns":  {Name: "GraphQLDns", GoType: "any"},
		"RealtimeUrl": {Name: "RealtimeUrl", GoType: "any"},
		"RealtimeDns": {Name: "RealtimeDns", GoType: "any"},
	},
}
