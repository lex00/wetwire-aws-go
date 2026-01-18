// Package expected provides API Gateway resources for the API scenario.
package expected

import (
	"github.com/lex00/wetwire-aws-go/resources/apigatewayv2"
)

// LambdaApi is an HTTP API that routes requests to the Lambda function.
// It creates a publicly accessible endpoint with automatic Lambda integration.
var LambdaApi = apigatewayv2.Api{
	Name:         "lambda-api",
	ProtocolType: "HTTP",
	Description:  "HTTP API for Lambda-backed REST endpoint",
	Target:       ApiProcessor.Arn,
}
