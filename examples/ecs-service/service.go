// Package ecs_service demonstrates idiomatic wetwire patterns for ECS.
//
// This file contains the ECS service.
package ecs_service

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ecs"
)

// ----------------------------------------------------------------------------
// Service Configuration
// ----------------------------------------------------------------------------

// ServiceNetworkConfig configures VPC networking for the service.
// In production, you would reference actual subnet and security group resources.
var ServiceNetworkConfig = ecs.Service_AwsVpcConfiguration{
	AssignPublicIp: "ENABLED",
	Subnets: []any{
		Param("SubnetId"), // In production, reference actual subnet resources
	},
	SecurityGroups: []any{
		Param("SecurityGroupId"), // In production, reference actual security group resources
	},
}

// ServiceNetwork wraps the VPC configuration.
var ServiceNetwork = ecs.Service_NetworkConfiguration{
	AwsvpcConfiguration: ServiceNetworkConfig,
}

// DeploymentCircuitBreaker enables rollback on deployment failure.
var DeploymentCircuitBreaker = ecs.Service_DeploymentCircuitBreaker{
	Enable:   true,
	Rollback: true,
}

// DeploymentConfig configures rolling deployments with circuit breaker.
var DeploymentConfig = ecs.Service_DeploymentConfiguration{
	MaximumPercent:           200,
	MinimumHealthyPercent:    100,
	DeploymentCircuitBreaker: DeploymentCircuitBreaker,
}

// ----------------------------------------------------------------------------
// ECS Service
// ----------------------------------------------------------------------------

// AppService runs the application as a Fargate service.
// Note: Direct references to Cluster and TaskDefinition - no Ref() needed!
var AppService = ecs.Service{
	ServiceName:             Sub{String: "${AWS::StackName}-service"},
	Cluster:                 Cluster,
	TaskDefinition:          TaskDefinition,
	DesiredCount:            2,
	LaunchType:              "FARGATE",
	NetworkConfiguration:    ServiceNetwork,
	DeploymentConfiguration: DeploymentConfig,
	EnableECSManagedTags:    true,
	PropagateTags:           "TASK_DEFINITION",
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-service"}},
	},
}
