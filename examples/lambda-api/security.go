// Package lambda_api demonstrates idiomatic wetwire patterns for Lambda and API Gateway.
//
// This file contains IAM resources: roles and policies.
package lambda_api

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/iam"
)

// ----------------------------------------------------------------------------
// Lambda Execution Role
// ----------------------------------------------------------------------------

// LambdaAssumeRoleStatement allows Lambda service to assume this role.
var LambdaAssumeRoleStatement = PolicyStatement{
	Effect:    "Allow",
	Principal: ServicePrincipal{"lambda.amazonaws.com"},
	Action:    "sts:AssumeRole",
}

// LambdaAssumeRolePolicy is the trust policy for the Lambda execution role.
var LambdaAssumeRolePolicy = PolicyDocument{
	Version:   "2012-10-17",
	Statement: []any{LambdaAssumeRoleStatement},
}

// LambdaLogsStatement allows Lambda to write CloudWatch logs.
var LambdaLogsStatement = PolicyStatement{
	Effect: "Allow",
	Action: []any{
		"logs:CreateLogGroup",
		"logs:CreateLogStream",
		"logs:PutLogEvents",
	},
	Resource: "arn:aws:logs:*:*:*",
}

// LambdaLogsPolicy is the inline policy for CloudWatch logging.
var LambdaLogsPolicy = iam.Role_Policy{
	PolicyName: "lambda-logs",
	PolicyDocument: PolicyDocument{
		Version:   "2012-10-17",
		Statement: []any{LambdaLogsStatement},
	},
}

// LambdaExecutionRole is the IAM role assumed by Lambda functions.
// It includes permissions to write CloudWatch logs.
var LambdaExecutionRole = iam.Role{
	RoleName:                 Sub{String: "${AWS::StackName}-lambda-role"},
	AssumeRolePolicyDocument: LambdaAssumeRolePolicy,
	Policies:                 []any{LambdaLogsPolicy},
}
