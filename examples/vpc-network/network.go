// Package vpc_network demonstrates idiomatic wetwire patterns for VPC networking.
//
// This example shows:
// - Flat block-style declarations (named top-level vars)
// - Everything wrapped (direct references, no Ref()/GetAtt() calls)
// - Extracted inline configs (no nested struct literals)
// - Clear comments showing network topology
//
// Network Topology:
//
//	VPC (10.0.0.0/16)
//	|
//	+-- Public Subnet AZ-a (10.0.0.0/24)
//	|   +-- NAT Gateway -> Private Subnet routing
//	|
//	+-- Public Subnet AZ-b (10.0.1.0/24)
//	|
//	+-- Private Subnet AZ-a (10.0.10.0/24)
//	|
//	+-- Private Subnet AZ-b (10.0.11.0/24)
package vpc_network

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ec2"
)

// ----------------------------------------------------------------------------
// VPC
// ----------------------------------------------------------------------------

// VPC is the main Virtual Private Cloud with DNS support enabled.
var VPC = ec2.VPC{
	CidrBlock:          "10.0.0.0/16",
	EnableDnsHostnames: true,
	EnableDnsSupport:   true,
	InstanceTenancy:    "default",
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-vpc"}},
	},
}

// ----------------------------------------------------------------------------
// Internet Gateway
// ----------------------------------------------------------------------------

// InternetGateway provides internet access for public subnets.
var InternetGateway = ec2.InternetGateway{
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-igw"}},
	},
}

// VPCGatewayAttachment attaches the Internet Gateway to the VPC.
// Note: Direct references to InternetGateway and VPC - no Ref() needed!
var VPCGatewayAttachment = ec2.VPCGatewayAttachment{
	InternetGatewayId: InternetGateway,
	VpcId:             VPC,
}

// ----------------------------------------------------------------------------
// Public Subnets
// ----------------------------------------------------------------------------

// PublicSubnetA is a public subnet in the first availability zone.
// Note: Direct reference to VPC - no Ref() needed!
var PublicSubnetA = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.0.0/24",
	AvailabilityZone:    Select{Index: 0, List: GetAZs{}},
	MapPublicIpOnLaunch: true,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-public-a"}},
	},
}

// PublicSubnetB is a public subnet in the second availability zone.
var PublicSubnetB = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.1.0/24",
	AvailabilityZone:    Select{Index: 1, List: GetAZs{}},
	MapPublicIpOnLaunch: true,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-public-b"}},
	},
}

// ----------------------------------------------------------------------------
// Private Subnets
// ----------------------------------------------------------------------------

// PrivateSubnetA is a private subnet in the first availability zone.
var PrivateSubnetA = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.10.0/24",
	AvailabilityZone:    Select{Index: 0, List: GetAZs{}},
	MapPublicIpOnLaunch: false,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-private-a"}},
	},
}

// PrivateSubnetB is a private subnet in the second availability zone.
var PrivateSubnetB = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.11.0/24",
	AvailabilityZone:    Select{Index: 1, List: GetAZs{}},
	MapPublicIpOnLaunch: false,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-private-b"}},
	},
}

// ----------------------------------------------------------------------------
// NAT Gateway (for private subnet internet access)
// ----------------------------------------------------------------------------

// NATGatewayEIP is the Elastic IP for the NAT Gateway.
var NATGatewayEIP = ec2.EIP{
	Domain: "vpc",
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-nat-eip"}},
	},
}

// NATGateway provides outbound internet access for private subnets.
// Note: Direct references to NATGatewayEIP.AllocationId and PublicSubnetA - no GetAtt()/Ref() needed!
var NATGateway = ec2.NatGateway{
	AllocationId: NATGatewayEIP.AllocationId,
	SubnetId:     PublicSubnetA,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-nat"}},
	},
}

// ----------------------------------------------------------------------------
// Route Tables - Public
// ----------------------------------------------------------------------------

// PublicRouteTable is the route table for public subnets.
var PublicRouteTable = ec2.RouteTable{
	VpcId: VPC,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-public-rt"}},
	},
}

// PublicRoute routes internet traffic through the Internet Gateway.
// Note: Direct references to PublicRouteTable and InternetGateway - no Ref() needed!
var PublicRoute = ec2.Route{
	RouteTableId:         PublicRouteTable,
	DestinationCidrBlock: "0.0.0.0/0",
	GatewayId:            InternetGateway,
}

// PublicSubnetARouteTableAssociation associates PublicSubnetA with the public route table.
var PublicSubnetARouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PublicSubnetA,
	RouteTableId: PublicRouteTable,
}

// PublicSubnetBRouteTableAssociation associates PublicSubnetB with the public route table.
var PublicSubnetBRouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PublicSubnetB,
	RouteTableId: PublicRouteTable,
}

// ----------------------------------------------------------------------------
// Route Tables - Private
// ----------------------------------------------------------------------------

// PrivateRouteTable is the route table for private subnets.
var PrivateRouteTable = ec2.RouteTable{
	VpcId: VPC,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-private-rt"}},
	},
}

// PrivateRoute routes internet traffic through the NAT Gateway.
// Note: Direct reference to NATGateway - no Ref() needed!
var PrivateRoute = ec2.Route{
	RouteTableId:         PrivateRouteTable,
	DestinationCidrBlock: "0.0.0.0/0",
	NatGatewayId:         NATGateway,
}

// PrivateSubnetARouteTableAssociation associates PrivateSubnetA with the private route table.
var PrivateSubnetARouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PrivateSubnetA,
	RouteTableId: PrivateRouteTable,
}

// PrivateSubnetBRouteTableAssociation associates PrivateSubnetB with the private route table.
var PrivateSubnetBRouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PrivateSubnetB,
	RouteTableId: PrivateRouteTable,
}
