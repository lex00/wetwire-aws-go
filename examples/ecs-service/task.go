// Package ecs_service demonstrates idiomatic wetwire patterns for ECS.
//
// This file contains the ECS task definition.
package ecs_service

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ecs"
)

// ----------------------------------------------------------------------------
// Task Definition
// ----------------------------------------------------------------------------

// AppPortMapping exposes port 8080 from the container.
var AppPortMapping = ecs.TaskDefinition_PortMapping{
	ContainerPort: 8080,
	Protocol:      "tcp",
}

// AppLogConfiguration sends container logs to CloudWatch.
// Note: Direct reference to AppLogGroup - no Ref() needed!
var AppLogConfiguration = ecs.TaskDefinition_LogConfiguration{
	LogDriver: "awslogs",
	Options: map[string]any{
		"awslogs-group":         AppLogGroup,
		"awslogs-region":        AWS_REGION,
		"awslogs-stream-prefix": "app",
	},
}

// AppContainerDefinition defines the main application container.
var AppContainerDefinition = ecs.TaskDefinition_ContainerDefinition{
	Name:             "app",
	Image:            Sub{String: "${AWS::AccountId}.dkr.ecr.${AWS::Region}.amazonaws.com/myapp:latest"},
	Essential:        true,
	PortMappings:     []any{AppPortMapping},
	LogConfiguration: AppLogConfiguration,
	Environment: []any{
		ecs.TaskDefinition_KeyValuePair{Name: "PORT", Value: "8080"},
		ecs.TaskDefinition_KeyValuePair{Name: "ENV", Value: "production"},
	},
}

// TaskDefinition defines the Fargate task.
// Note: Direct references to TaskExecutionRole.Arn and TaskRole.Arn - no GetAtt() needed!
var TaskDefinition = ecs.TaskDefinition{
	Family:                  Sub{String: "${AWS::StackName}-task"},
	NetworkMode:             "awsvpc",
	RequiresCompatibilities: []any{"FARGATE"},
	Cpu:                     "256",
	Memory:                  "512",
	ExecutionRoleArn:        TaskExecutionRole.Arn,
	TaskRoleArn:             TaskRole.Arn,
	ContainerDefinitions:    []any{AppContainerDefinition},
}
