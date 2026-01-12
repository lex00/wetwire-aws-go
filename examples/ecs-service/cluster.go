// Package ecs_service demonstrates idiomatic wetwire patterns for ECS.
//
// This file contains the ECS cluster and related configuration.
package ecs_service

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ecs"
	"github.com/lex00/wetwire-aws-go/resources/logs"
)

// ----------------------------------------------------------------------------
// CloudWatch Log Group
// ----------------------------------------------------------------------------

// AppLogGroup stores container logs with 30-day retention.
var AppLogGroup = logs.LogGroup{
	LogGroupName:    Sub{String: "/ecs/${AWS::StackName}"},
	RetentionInDays: 30,
}

// ----------------------------------------------------------------------------
// ECS Cluster
// ----------------------------------------------------------------------------

// ContainerInsightsSetting enables CloudWatch Container Insights.
var ContainerInsightsSetting = ecs.Cluster_ClusterSettings{
	Name:  "containerInsights",
	Value: "enabled",
}

// Cluster is the ECS cluster that runs our services.
var Cluster = ecs.Cluster{
	ClusterName:     Sub{String: "${AWS::StackName}-cluster"},
	ClusterSettings: []any{ContainerInsightsSetting},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-cluster"}},
	},
}
