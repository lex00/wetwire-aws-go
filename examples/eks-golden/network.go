// Package eks_golden provides a production-ready EKS cluster example.
//
// This example demonstrates best practices for deploying an EKS cluster
// with proper networking, security, and observability configurations.
package eks_golden

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ec2"
)

// VPC is the main VPC for the EKS cluster.
var VPC = ec2.VPC{
	CidrBlock:          "10.0.0.0/16",
	EnableDnsHostnames: true,
	EnableDnsSupport:   true,
	InstanceTenancy:    "default",
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-vpc"}},
		Tag{Key: "kubernetes.io/cluster/${AWS::StackName}", Value: "shared"},
	},
}

// InternetGateway provides internet access for public subnets.
var InternetGateway = ec2.InternetGateway{
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-igw"}},
	},
}

// VPCGatewayAttachment attaches the internet gateway to the VPC.
var VPCGatewayAttachment = ec2.VPCGatewayAttachment{
	InternetGatewayId: InternetGateway,
	VpcId:             VPC,
}

// Public Subnets (for load balancers, NAT gateways)

// PublicSubnetA is the first public subnet.
var PublicSubnetA = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.1.0/24",
	AvailabilityZone:    Select{Index: 0, List: GetAZs{}},
	MapPublicIpOnLaunch: true,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-public-a"}},
		Tag{Key: "kubernetes.io/cluster/${AWS::StackName}", Value: "shared"},
		Tag{Key: "kubernetes.io/role/elb", Value: "1"},
	},
}

// PublicSubnetB is the second public subnet.
var PublicSubnetB = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.2.0/24",
	AvailabilityZone:    Select{Index: 1, List: GetAZs{}},
	MapPublicIpOnLaunch: true,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-public-b"}},
		Tag{Key: "kubernetes.io/cluster/${AWS::StackName}", Value: "shared"},
		Tag{Key: "kubernetes.io/role/elb", Value: "1"},
	},
}

// PublicSubnetC is the third public subnet.
var PublicSubnetC = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.3.0/24",
	AvailabilityZone:    Select{Index: 2, List: GetAZs{}},
	MapPublicIpOnLaunch: true,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-public-c"}},
		Tag{Key: "kubernetes.io/cluster/${AWS::StackName}", Value: "shared"},
		Tag{Key: "kubernetes.io/role/elb", Value: "1"},
	},
}

// Private Subnets (for worker nodes)

// PrivateSubnetA is the first private subnet for worker nodes.
var PrivateSubnetA = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.10.0/24",
	AvailabilityZone:    Select{Index: 0, List: GetAZs{}},
	MapPublicIpOnLaunch: false,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-private-a"}},
		Tag{Key: "kubernetes.io/cluster/${AWS::StackName}", Value: "shared"},
		Tag{Key: "kubernetes.io/role/internal-elb", Value: "1"},
	},
}

// PrivateSubnetB is the second private subnet for worker nodes.
var PrivateSubnetB = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.11.0/24",
	AvailabilityZone:    Select{Index: 1, List: GetAZs{}},
	MapPublicIpOnLaunch: false,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-private-b"}},
		Tag{Key: "kubernetes.io/cluster/${AWS::StackName}", Value: "shared"},
		Tag{Key: "kubernetes.io/role/internal-elb", Value: "1"},
	},
}

// PrivateSubnetC is the third private subnet for worker nodes.
var PrivateSubnetC = ec2.Subnet{
	VpcId:               VPC,
	CidrBlock:           "10.0.12.0/24",
	AvailabilityZone:    Select{Index: 2, List: GetAZs{}},
	MapPublicIpOnLaunch: false,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-private-c"}},
		Tag{Key: "kubernetes.io/cluster/${AWS::StackName}", Value: "shared"},
		Tag{Key: "kubernetes.io/role/internal-elb", Value: "1"},
	},
}

// NAT Gateways for private subnet internet access

// NATGatewayEIPA is the elastic IP for the first NAT gateway.
var NATGatewayEIPA = ec2.EIP{
	Domain: "vpc",
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-nat-eip-a"}},
	},
}

// NATGatewayA provides internet access for private subnet A.
var NATGatewayA = ec2.NatGateway{
	AllocationId: NATGatewayEIPA.AllocationId,
	SubnetId:     PublicSubnetA,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-nat-a"}},
	},
}

// Route Tables

// PublicRouteTable is the route table for public subnets.
var PublicRouteTable = ec2.RouteTable{
	VpcId: VPC,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-public-rt"}},
	},
}

// PublicRoute routes internet traffic through the internet gateway.
var PublicRoute = ec2.Route{
	RouteTableId:         PublicRouteTable,
	DestinationCidrBlock: "0.0.0.0/0",
	GatewayId:            InternetGateway,
}

// PublicSubnetARouteTableAssociation associates public subnet A with the public route table.
var PublicSubnetARouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PublicSubnetA,
	RouteTableId: PublicRouteTable,
}

// PublicSubnetBRouteTableAssociation associates public subnet B with the public route table.
var PublicSubnetBRouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PublicSubnetB,
	RouteTableId: PublicRouteTable,
}

// PublicSubnetCRouteTableAssociation associates public subnet C with the public route table.
var PublicSubnetCRouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PublicSubnetC,
	RouteTableId: PublicRouteTable,
}

// PrivateRouteTable is the route table for private subnets.
var PrivateRouteTable = ec2.RouteTable{
	VpcId: VPC,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-private-rt"}},
	},
}

// PrivateRoute routes internet traffic through the NAT gateway.
var PrivateRoute = ec2.Route{
	RouteTableId:         PrivateRouteTable,
	DestinationCidrBlock: "0.0.0.0/0",
	NatGatewayId:         NATGatewayA,
}

// PrivateSubnetARouteTableAssociation associates private subnet A with the private route table.
var PrivateSubnetARouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PrivateSubnetA,
	RouteTableId: PrivateRouteTable,
}

// PrivateSubnetBRouteTableAssociation associates private subnet B with the private route table.
var PrivateSubnetBRouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PrivateSubnetB,
	RouteTableId: PrivateRouteTable,
}

// PrivateSubnetCRouteTableAssociation associates private subnet C with the private route table.
var PrivateSubnetCRouteTableAssociation = ec2.SubnetRouteTableAssociation{
	SubnetId:     PrivateSubnetC,
	RouteTableId: PrivateRouteTable,
}
