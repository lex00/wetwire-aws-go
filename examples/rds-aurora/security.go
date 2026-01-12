// Package rds_aurora demonstrates idiomatic wetwire patterns for RDS Aurora.
//
// This file contains the security group for the database.
package rds_aurora

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ec2"
)

// ----------------------------------------------------------------------------
// Database Security Group
// ----------------------------------------------------------------------------

// DBPostgreSQLIngress allows PostgreSQL traffic on port 5432.
// In production, limit SourceSecurityGroupId to your app security group.
var DBPostgreSQLIngress = ec2.SecurityGroup_Ingress{
	Description: "Allow PostgreSQL from app tier",
	IpProtocol:  "tcp",
	FromPort:    5432,
	ToPort:      5432,
	CidrIp:      "10.0.0.0/16", // In production, use SourceSecurityGroupId instead
}

// DBSecurityGroup restricts access to the Aurora cluster.
// In production, reference an actual VPC resource.
var DBSecurityGroup = ec2.SecurityGroup{
	GroupDescription:     "Security group for Aurora PostgreSQL cluster",
	VpcId:                Param("VpcId"), // In production, reference actual VPC resource
	SecurityGroupIngress: []any{DBPostgreSQLIngress},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-aurora-sg"}},
	},
}
