package main

// SAM Property Types

var samFunctionEnvironment = ParsedPropertyType{
	Name:           "Environment",
	CFType:         "AWS::Serverless::Function.Environment",
	ParentResource: "Function",
	Documentation:  "Environment variable configuration.",
	Properties: map[string]ParsedProperty{
		"Variables": {
			Name:          "Variables",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Environment variables.",
		},
	},
}

var samFunctionEventSource = ParsedPropertyType{
	Name:           "EventSource",
	CFType:         "AWS::Serverless::Function.EventSource",
	ParentResource: "Function",
	Documentation:  "Event source configuration.",
	Properties: map[string]ParsedProperty{
		"Type": {
			Name:          "Type",
			GoType:        "any",
			Required:      true,
			Documentation: "Event type (Api, Schedule, S3, SNS, SQS, etc.).",
		},
		"Properties": {
			Name:          "Properties",
			GoType:        "any",
			Documentation: "Event-specific properties.",
		},
	},
}

var samFunctionVpcConfig = ParsedPropertyType{
	Name:           "VpcConfig",
	CFType:         "AWS::Serverless::Function.VpcConfig",
	ParentResource: "Function",
	Documentation:  "VPC configuration.",
	Properties: map[string]ParsedProperty{
		"SecurityGroupIds": {
			Name:          "SecurityGroupIds",
			GoType:        "[]any",
			IsList:        true,
			Required:      true,
			Documentation: "Security group IDs.",
		},
		"SubnetIds": {
			Name:          "SubnetIds",
			GoType:        "[]any",
			IsList:        true,
			Required:      true,
			Documentation: "Subnet IDs.",
		},
	},
}

var samFunctionDeadLetterQueue = ParsedPropertyType{
	Name:           "DeadLetterQueue",
	CFType:         "AWS::Serverless::Function.DeadLetterQueue",
	ParentResource: "Function",
	Documentation:  "Dead letter queue configuration.",
	Properties: map[string]ParsedProperty{
		"Type": {
			Name:          "Type",
			GoType:        "any",
			Required:      true,
			Documentation: "Queue type (SQS or SNS).",
		},
		"TargetArn": {
			Name:          "TargetArn",
			GoType:        "any",
			Required:      true,
			Documentation: "Target queue/topic ARN.",
		},
	},
}

var samFunctionDeploymentPreference = ParsedPropertyType{
	Name:           "DeploymentPreference",
	CFType:         "AWS::Serverless::Function.DeploymentPreference",
	ParentResource: "Function",
	Documentation:  "Gradual deployment configuration.",
	Properties: map[string]ParsedProperty{
		"Type": {
			Name:          "Type",
			GoType:        "any",
			Required:      true,
			Documentation: "Deployment type (Canary10Percent5Minutes, Linear10PercentEvery1Minute, AllAtOnce).",
		},
		"Enabled": {
			Name:          "Enabled",
			GoType:        "any",
			Documentation: "Enable deployment preference.",
		},
		"Alarms": {
			Name:          "Alarms",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "CloudWatch alarm ARNs.",
		},
		"Hooks": {
			Name:          "Hooks",
			GoType:        "any",
			Documentation: "Pre/post traffic hooks.",
		},
	},
}

var samFunctionS3Location = ParsedPropertyType{
	Name:           "S3Location",
	CFType:         "AWS::Serverless::Function.S3Location",
	ParentResource: "Function",
	Documentation:  "S3 location for code.",
	Properties: map[string]ParsedProperty{
		"Bucket": {
			Name:          "Bucket",
			GoType:        "any",
			Required:      true,
			Documentation: "S3 bucket name.",
		},
		"Key": {
			Name:          "Key",
			GoType:        "any",
			Required:      true,
			Documentation: "S3 object key.",
		},
		"Version": {
			Name:          "Version",
			GoType:        "any",
			Documentation: "S3 object version.",
		},
	},
}

var samApiAuth = ParsedPropertyType{
	Name:           "Auth",
	CFType:         "AWS::Serverless::Api.Auth",
	ParentResource: "Api",
	Documentation:  "API authentication configuration.",
	Properties: map[string]ParsedProperty{
		"DefaultAuthorizer": {
			Name:          "DefaultAuthorizer",
			GoType:        "any",
			Documentation: "Default authorizer name.",
		},
		"Authorizers": {
			Name:          "Authorizers",
			GoType:        "map[string]any",
			IsMap:         true,
			Documentation: "Authorizer definitions.",
		},
		"ApiKeyRequired": {
			Name:          "ApiKeyRequired",
			GoType:        "any",
			Documentation: "Require API key.",
		},
		"UsagePlan": {
			Name:          "UsagePlan",
			GoType:        "any",
			Documentation: "Usage plan configuration.",
		},
	},
}

var samApiCorsConfiguration = ParsedPropertyType{
	Name:           "CorsConfiguration",
	CFType:         "AWS::Serverless::Api.CorsConfiguration",
	ParentResource: "Api",
	Documentation:  "CORS configuration.",
	Properties: map[string]ParsedProperty{
		"AllowOrigin": {
			Name:          "AllowOrigin",
			GoType:        "any",
			Required:      true,
			Documentation: "Allowed origin.",
		},
		"AllowHeaders": {
			Name:          "AllowHeaders",
			GoType:        "any",
			Documentation: "Allowed headers.",
		},
		"AllowMethods": {
			Name:          "AllowMethods",
			GoType:        "any",
			Documentation: "Allowed methods.",
		},
		"AllowCredentials": {
			Name:          "AllowCredentials",
			GoType:        "any",
			Documentation: "Allow credentials.",
		},
		"MaxAge": {
			Name:          "MaxAge",
			GoType:        "any",
			Documentation: "Max age for preflight cache.",
		},
	},
}

var samHttpApiCorsConfiguration = ParsedPropertyType{
	Name:           "CorsConfiguration",
	CFType:         "AWS::Serverless::HttpApi.CorsConfiguration",
	ParentResource: "HttpApi",
	Documentation:  "CORS configuration for HTTP API.",
	Properties: map[string]ParsedProperty{
		"AllowOrigins": {
			Name:          "AllowOrigins",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Allowed origins.",
		},
		"AllowHeaders": {
			Name:          "AllowHeaders",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Allowed headers.",
		},
		"AllowMethods": {
			Name:          "AllowMethods",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Allowed methods.",
		},
		"AllowCredentials": {
			Name:          "AllowCredentials",
			GoType:        "any",
			Documentation: "Allow credentials.",
		},
		"ExposeHeaders": {
			Name:          "ExposeHeaders",
			GoType:        "[]any",
			IsList:        true,
			Documentation: "Exposed headers.",
		},
		"MaxAge": {
			Name:          "MaxAge",
			GoType:        "any",
			Documentation: "Max age for preflight cache.",
		},
	},
}

var samSimpleTablePrimaryKey = ParsedPropertyType{
	Name:           "PrimaryKey",
	CFType:         "AWS::Serverless::SimpleTable.PrimaryKey",
	ParentResource: "SimpleTable",
	Documentation:  "Primary key configuration.",
	Properties: map[string]ParsedProperty{
		"Name": {
			Name:          "Name",
			GoType:        "any",
			Required:      true,
			Documentation: "Attribute name.",
		},
		"Type": {
			Name:          "Type",
			GoType:        "any",
			Required:      true,
			Documentation: "Attribute type (String, Number, Binary).",
		},
	},
}

var samStateMachineS3Location = ParsedPropertyType{
	Name:           "S3Location",
	CFType:         "AWS::Serverless::StateMachine.S3Location",
	ParentResource: "StateMachine",
	Documentation:  "S3 location for state machine definition.",
	Properties: map[string]ParsedProperty{
		"Bucket": {
			Name:          "Bucket",
			GoType:        "any",
			Required:      true,
			Documentation: "S3 bucket name.",
		},
		"Key": {
			Name:          "Key",
			GoType:        "any",
			Required:      true,
			Documentation: "S3 object key.",
		},
		"Version": {
			Name:          "Version",
			GoType:        "any",
			Documentation: "S3 object version.",
		},
	},
}
