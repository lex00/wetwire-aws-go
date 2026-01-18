// Package expected provides IAM security resources for the API scenario.
package expected

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/iam"
)

// ExecutionRole is the IAM role that grants the Lambda function permission to execute.
// It includes basic CloudWatch Logs permissions for function logging.
var ExecutionRole = iam.Role{
	RoleName:    "lambda-execution-role",
	Description: "Execution role for Lambda API processor",
	AssumeRolePolicyDocument: Json{
		"Version": "2012-10-17",
		"Statement": []Json{
			{
				"Effect": "Allow",
				"Principal": Json{
					"Service": "lambda.amazonaws.com",
				},
				"Action": "sts:AssumeRole",
			},
		},
	},
	ManagedPolicyArns: []any{
		"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
	},
}
