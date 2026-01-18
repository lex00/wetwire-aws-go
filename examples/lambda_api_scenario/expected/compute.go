// Package expected provides Lambda compute resources for the API scenario.
package expected

import (
	"github.com/lex00/wetwire-aws-go/resources/lambda"
)

// ApiProcessor is the Lambda function that handles API requests.
// It returns a simple JSON response for demonstration purposes.
var ApiProcessor = lambda.Function{
	FunctionName: "api-processor",
	Runtime:      "python3.12",
	Handler:      "index.handler",
	Role:         ExecutionRole.Arn,
	MemorySize:   128,
	Timeout:      30,
	Code: lambda.Function_Code{
		ZipFile: `def handler(event, context):
    return {
        "statusCode": 200,
        "headers": {"Content-Type": "application/json"},
        "body": '{"message": "Hello from Lambda"}'
    }`,
	},
}
