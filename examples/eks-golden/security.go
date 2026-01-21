package eks_golden

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/ec2"
	"github.com/lex00/wetwire-aws-go/resources/iam"
)

// Security Groups

// ControlPlaneSecurityGroup is the security group for the EKS control plane.
var ControlPlaneSecurityGroup = ec2.SecurityGroup{
	GroupDescription: "Security group for EKS control plane communication",
	VpcId:            VPC,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-control-plane-sg"}},
	},
}

// NodeSecurityGroup is the security group for EKS worker nodes.
var NodeSecurityGroup = ec2.SecurityGroup{
	GroupDescription: "Security group for EKS worker nodes",
	VpcId:            VPC,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-node-sg"}},
	},
}

// NodeSecurityGroupIngress allows nodes to communicate with each other.
var NodeSecurityGroupIngress = ec2.SecurityGroupIngress{
	GroupId:               NodeSecurityGroup,
	SourceSecurityGroupId: NodeSecurityGroup,
	IpProtocol:            "-1",
	Description:           "Allow node to node communication",
}

// ControlPlaneToNodeIngress allows control plane to communicate with nodes.
var ControlPlaneToNodeIngress = ec2.SecurityGroupIngress{
	GroupId:               NodeSecurityGroup,
	SourceSecurityGroupId: ControlPlaneSecurityGroup,
	IpProtocol:            "tcp",
	FromPort:              1025,
	ToPort:                65535,
	Description:           "Allow control plane to communicate with worker nodes",
}

// ControlPlaneToNodeKubeletIngress allows control plane to communicate with kubelet.
var ControlPlaneToNodeKubeletIngress = ec2.SecurityGroupIngress{
	GroupId:               NodeSecurityGroup,
	SourceSecurityGroupId: ControlPlaneSecurityGroup,
	IpProtocol:            "tcp",
	FromPort:              443,
	ToPort:                443,
	Description:           "Allow control plane to communicate with kubelet",
}

// NodeToControlPlaneIngress allows nodes to communicate with control plane.
var NodeToControlPlaneIngress = ec2.SecurityGroupIngress{
	GroupId:               ControlPlaneSecurityGroup,
	SourceSecurityGroupId: NodeSecurityGroup,
	IpProtocol:            "tcp",
	FromPort:              443,
	ToPort:                443,
	Description:           "Allow worker nodes to communicate with cluster API server",
}

// IAM Roles

// EKS Cluster Role

// ClusterAssumeRoleStatement allows EKS service to assume the cluster role.
var ClusterAssumeRoleStatement = PolicyStatement{
	Effect:    "Allow",
	Principal: ServicePrincipal{"eks.amazonaws.com"},
	Action:    "sts:AssumeRole",
}

// ClusterAssumeRolePolicy is the assume role policy document for the cluster role.
var ClusterAssumeRolePolicy = PolicyDocument{
	Version:   "2012-10-17",
	Statement: []any{ClusterAssumeRoleStatement},
}

// ClusterRole is the IAM role for the EKS cluster.
var ClusterRole = iam.Role{
	RoleName:                 Sub{String: "${AWS::StackName}-cluster-role"},
	AssumeRolePolicyDocument: ClusterAssumeRolePolicy,
	ManagedPolicyArns: []any{
		"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
	},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-cluster-role"}},
	},
}

// EKS Node Role

// NodeAssumeRoleStatement allows EC2 service to assume the node role.
var NodeAssumeRoleStatement = PolicyStatement{
	Effect:    "Allow",
	Principal: ServicePrincipal{"ec2.amazonaws.com"},
	Action:    "sts:AssumeRole",
}

// NodeAssumeRolePolicy is the assume role policy document for the node role.
var NodeAssumeRolePolicy = PolicyDocument{
	Version:   "2012-10-17",
	Statement: []any{NodeAssumeRoleStatement},
}

// NodeRole is the IAM role for EKS worker nodes.
var NodeRole = iam.Role{
	RoleName:                 Sub{String: "${AWS::StackName}-node-role"},
	AssumeRolePolicyDocument: NodeAssumeRolePolicy,
	ManagedPolicyArns: []any{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
	},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-node-role"}},
	},
}

// IRSA (IAM Roles for Service Accounts) OIDC Provider would be created after cluster
// as it requires the cluster's OIDC issuer URL
