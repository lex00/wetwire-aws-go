// Package lambda_api demonstrates idiomatic wetwire patterns for Lambda and API Gateway.
//
// This file contains API Gateway resources: REST API, resource, method, and deployment.
package lambda_api

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/apigateway"
)

// ----------------------------------------------------------------------------
// REST API
// ----------------------------------------------------------------------------

// RestAPI is the API Gateway REST API.
var RestAPI = apigateway.RestApi{
	Name:        Sub{String: "${AWS::StackName}-api"},
	Description: "API Gateway for Lambda function",
}

// ----------------------------------------------------------------------------
// API Resource (path)
// ----------------------------------------------------------------------------

// HelloResource creates the /hello path on the REST API.
// Note: Direct references to RestAPI and RestAPI.RootResourceId - no Ref()/GetAtt() needed!
var HelloResource = apigateway.Resource{
	RestApiId: RestAPI,
	ParentId:  RestAPI.RootResourceId,
	PathPart:  "hello",
}

// ----------------------------------------------------------------------------
// API Method Integration
// ----------------------------------------------------------------------------

// HelloIntegrationResponse defines the integration response mapping.
var HelloIntegrationResponse = apigateway.Method_IntegrationResponse{
	StatusCode: "200",
}

// HelloIntegration configures the Lambda proxy integration.
// Note: Direct reference to HelloFunction.Arn in the URI - no Ref() needed!
var HelloIntegration = apigateway.Method_Integration{
	Type_:                 "AWS_PROXY",
	IntegrationHttpMethod: "POST",
	Uri: Join{
		Delimiter: "",
		Values: []any{
			"arn:aws:apigateway:",
			AWS_REGION,
			":lambda:path/2015-03-31/functions/",
			HelloFunction.Arn,
			"/invocations",
		},
	},
	IntegrationResponses: []any{HelloIntegrationResponse},
}

// HelloMethodResponse defines the 200 response for the method.
var HelloMethodResponse = apigateway.Method_MethodResponse{
	StatusCode: "200",
}

// HelloMethod creates a GET method on the /hello resource.
// Note: Direct references to RestAPI and HelloResource - no Ref() needed!
var HelloMethod = apigateway.Method{
	RestApiId:         RestAPI,
	ResourceId:        HelloResource,
	HttpMethod:        "GET",
	AuthorizationType: "NONE",
	Integration:       HelloIntegration,
	MethodResponses:   []any{HelloMethodResponse},
}

// ----------------------------------------------------------------------------
// API Deployment
// ----------------------------------------------------------------------------

// APIDeployment deploys the REST API to a stage.
// Note: Direct reference to RestAPI - no Ref() needed!
var APIDeployment = apigateway.Deployment{
	RestApiId: RestAPI,
	StageName: "prod",
}
