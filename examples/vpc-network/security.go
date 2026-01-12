// Package vpc_network demonstrates idiomatic wetwire patterns for VPC networking.
//
// This file contains Security Groups.
package vpc_network

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ec2"
)

// ----------------------------------------------------------------------------
// Web Tier Security Group
// ----------------------------------------------------------------------------

// WebHTTPSIngress allows HTTPS traffic from the internet.
var WebHTTPSIngress = ec2.SecurityGroup_Ingress{
	Description: "Allow HTTPS from internet",
	IpProtocol:  "tcp",
	FromPort:    443,
	ToPort:      443,
	CidrIp:      "0.0.0.0/0",
}

// WebHTTPIngress allows HTTP traffic from the internet (for redirects).
var WebHTTPIngress = ec2.SecurityGroup_Ingress{
	Description: "Allow HTTP from internet",
	IpProtocol:  "tcp",
	FromPort:    80,
	ToPort:      80,
	CidrIp:      "0.0.0.0/0",
}

// WebEgressAll allows all outbound traffic.
var WebEgressAll = ec2.SecurityGroup_Egress{
	Description: "Allow all outbound",
	IpProtocol:  "-1",
	CidrIp:      "0.0.0.0/0",
}

// WebSecurityGroup allows HTTP/HTTPS traffic from the internet.
// Note: Direct reference to VPC - no Ref() needed!
var WebSecurityGroup = ec2.SecurityGroup{
	GroupDescription:     "Security group for web tier - allows HTTP/HTTPS",
	VpcId:                VPC,
	SecurityGroupIngress: []any{WebHTTPSIngress, WebHTTPIngress},
	SecurityGroupEgress:  []any{WebEgressAll},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-web-sg"}},
	},
}

// ----------------------------------------------------------------------------
// App Tier Security Group
// ----------------------------------------------------------------------------

// AppFromWebIngress allows traffic from the web security group.
// Note: Direct reference to WebSecurityGroup.GroupId - no GetAtt() needed!
var AppFromWebIngress = ec2.SecurityGroup_Ingress{
	Description:           "Allow traffic from web tier",
	IpProtocol:            "tcp",
	FromPort:              8080,
	ToPort:                8080,
	SourceSecurityGroupId: WebSecurityGroup.GroupId,
}

// AppEgressAll allows all outbound traffic.
var AppEgressAll = ec2.SecurityGroup_Egress{
	Description: "Allow all outbound",
	IpProtocol:  "-1",
	CidrIp:      "0.0.0.0/0",
}

// AppSecurityGroup allows traffic only from the web tier.
var AppSecurityGroup = ec2.SecurityGroup{
	GroupDescription:     "Security group for app tier - allows traffic from web tier",
	VpcId:                VPC,
	SecurityGroupIngress: []any{AppFromWebIngress},
	SecurityGroupEgress:  []any{AppEgressAll},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-app-sg"}},
	},
}

// ----------------------------------------------------------------------------
// Database Tier Security Group
// ----------------------------------------------------------------------------

// DBFromAppIngress allows PostgreSQL traffic from the app security group.
// Note: Direct reference to AppSecurityGroup.GroupId - no GetAtt() needed!
var DBFromAppIngress = ec2.SecurityGroup_Ingress{
	Description:           "Allow PostgreSQL from app tier",
	IpProtocol:            "tcp",
	FromPort:              5432,
	ToPort:                5432,
	SourceSecurityGroupId: AppSecurityGroup.GroupId,
}

// DBSecurityGroup allows database traffic only from the app tier.
var DBSecurityGroup = ec2.SecurityGroup{
	GroupDescription:     "Security group for database tier - allows traffic from app tier",
	VpcId:                VPC,
	SecurityGroupIngress: []any{DBFromAppIngress},
	// No egress rules - databases typically don't initiate outbound connections
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-db-sg"}},
	},
}
