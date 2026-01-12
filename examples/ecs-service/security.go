// Package ecs_service demonstrates idiomatic wetwire patterns for ECS.
//
// This file contains IAM resources: task execution role and task role.
package ecs_service

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/iam"
)

// ----------------------------------------------------------------------------
// ECS Task Execution Role
// ----------------------------------------------------------------------------

// ECSAssumeRoleStatement allows ECS tasks to assume this role.
var ECSAssumeRoleStatement = PolicyStatement{
	Effect:    "Allow",
	Principal: ServicePrincipal{"ecs-tasks.amazonaws.com"},
	Action:    "sts:AssumeRole",
}

// ECSAssumeRolePolicy is the trust policy for ECS roles.
var ECSAssumeRolePolicy = PolicyDocument{
	Version:   "2012-10-17",
	Statement: []any{ECSAssumeRoleStatement},
}

// TaskExecutionRole is the IAM role that ECS uses to pull images and write logs.
// This is distinct from the task role (which the application uses).
var TaskExecutionRole = iam.Role{
	RoleName:                 Sub{String: "${AWS::StackName}-task-exec-role"},
	AssumeRolePolicyDocument: ECSAssumeRolePolicy,
	ManagedPolicyArns: []any{
		"arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy",
	},
}

// ----------------------------------------------------------------------------
// ECS Task Role (for application permissions)
// ----------------------------------------------------------------------------

// TaskS3AccessStatement allows the application to access S3.
var TaskS3AccessStatement = PolicyStatement{
	Effect: "Allow",
	Action: []any{
		"s3:GetObject",
		"s3:ListBucket",
	},
	Resource: []any{
		Sub{String: "arn:aws:s3:::${AWS::StackName}-data"},
		Sub{String: "arn:aws:s3:::${AWS::StackName}-data/*"},
	},
}

// TaskS3Policy is the inline policy for S3 access.
var TaskS3Policy = iam.Role_Policy{
	PolicyName: "s3-access",
	PolicyDocument: PolicyDocument{
		Version:   "2012-10-17",
		Statement: []any{TaskS3AccessStatement},
	},
}

// TaskRole is the IAM role that the application container uses.
// This role grants the application permissions to access AWS services.
var TaskRole = iam.Role{
	RoleName:                 Sub{String: "${AWS::StackName}-task-role"},
	AssumeRolePolicyDocument: ECSAssumeRolePolicy,
	Policies:                 []any{TaskS3Policy},
}
