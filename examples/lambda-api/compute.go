// Package lambda_api demonstrates idiomatic wetwire patterns for Lambda and API Gateway.
//
// This file contains the Lambda function and permissions.
package lambda_api

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/lambda"
)

// ----------------------------------------------------------------------------
// Lambda Function
// ----------------------------------------------------------------------------

// HelloFunctionCode contains the inline Python handler code.
var HelloFunctionCode = lambda.Function_Code{
	ZipFile: `import json

def handler(event, context):
    return {
        'statusCode': 200,
        'headers': {'Content-Type': 'application/json'},
        'body': json.dumps({'message': 'Hello from Lambda!'})
    }
`,
}

// HelloFunction is a simple Lambda that returns a greeting.
// Note: Role uses direct reference to LambdaExecutionRole.Arn - no GetAtt needed!
var HelloFunction = lambda.Function{
	FunctionName: Sub{String: "${AWS::StackName}-hello"},
	Description:  "Returns a hello message via API Gateway",
	Runtime:      "python3.12",
	Handler:      "index.handler",
	Code:         HelloFunctionCode,
	Role:         LambdaExecutionRole.Arn,
	Timeout:      30,
	MemorySize:   128,
}

// ----------------------------------------------------------------------------
// Lambda Permission for API Gateway
// ----------------------------------------------------------------------------

// APIGatewayInvokePermission allows API Gateway to invoke the Lambda function.
// Note: Direct reference to HelloFunction.Arn and RestAPI - no Ref() needed!
var APIGatewayInvokePermission = lambda.Permission{
	FunctionName: HelloFunction.Arn,
	Action:       "lambda:InvokeFunction",
	Principal:    "apigateway.amazonaws.com",
	SourceArn: Join{
		Delimiter: "",
		Values: []any{
			"arn:aws:execute-api:",
			AWS_REGION,
			":",
			AWS_ACCOUNT_ID,
			":",
			RestAPI,
			"/*",
		},
	},
}
