// Package eks_k8s provides a production-ready EKS cluster example using ACK (AWS Controllers for Kubernetes).
//
// This example demonstrates the KRM-style (Kubernetes Resource Model) approach
// to AWS infrastructure, where AWS resources are managed as Kubernetes CRDs.
package eks_k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ec2v1alpha1 "github.com/lex00/wetwire-aws-go/resources/k8s/ec2/v1alpha1"
)

// Helper variables for pointer fields
var (
	region        = "us-east-1"
	vpcCIDR       = "10.0.0.0/16"
	publicCIDRA   = "10.0.1.0/24"
	publicCIDRB   = "10.0.2.0/24"
	publicCIDRC   = "10.0.3.0/24"
	privateCIDRA  = "10.0.10.0/24"
	privateCIDRB  = "10.0.11.0/24"
	privateCIDRC  = "10.0.12.0/24"
	azA           = "us-east-1a"
	azB           = "us-east-1b"
	azC           = "us-east-1c"
	instanceTenancy = "default"
	dnsHostnames  = true
	dnsSupport    = true
	publicIP      = true
	privateIP     = false
)

// VPC is the virtual network for the EKS cluster, managed via ACK.
var VPC = ec2v1alpha1.VPC{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "VPC",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-vpc",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.VPCSpec{
		CIDRBlocks:         []*string{&vpcCIDR},
		EnableDNSHostnames: &dnsHostnames,
		EnableDNSSupport:   &dnsSupport,
		InstanceTenancy:    &instanceTenancy,
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-vpc")},
			{Key: strPtr("Environment"), Value: strPtr("production")},
			{Key: strPtr("ManagedBy"), Value: strPtr("wetwire-ack")},
		},
	},
}

// Public Subnets (for load balancers)

// PublicSubnetA is the first public subnet.
var PublicSubnetA = ec2v1alpha1.Subnet{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "Subnet",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-public-a",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.SubnetSpec{
		CIDRBlock:           &publicCIDRA,
		AvailabilityZone:    &azA,
		MapPublicIPOnLaunch: &publicIP,
		VPCRef: &ec2v1alpha1.AWSResourceReferenceWrapper{
			From: &ec2v1alpha1.AWSResourceReference{
				Name: strPtr(VPC.ObjectMeta.Name),
			},
		},
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-public-a")},
			{Key: strPtr("kubernetes.io/role/elb"), Value: strPtr("1")},
		},
	},
}

// PublicSubnetB is the second public subnet.
var PublicSubnetB = ec2v1alpha1.Subnet{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "Subnet",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-public-b",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.SubnetSpec{
		CIDRBlock:           &publicCIDRB,
		AvailabilityZone:    &azB,
		MapPublicIPOnLaunch: &publicIP,
		VPCRef: &ec2v1alpha1.AWSResourceReferenceWrapper{
			From: &ec2v1alpha1.AWSResourceReference{
				Name: strPtr(VPC.ObjectMeta.Name),
			},
		},
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-public-b")},
			{Key: strPtr("kubernetes.io/role/elb"), Value: strPtr("1")},
		},
	},
}

// PublicSubnetC is the third public subnet.
var PublicSubnetC = ec2v1alpha1.Subnet{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "Subnet",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-public-c",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.SubnetSpec{
		CIDRBlock:           &publicCIDRC,
		AvailabilityZone:    &azC,
		MapPublicIPOnLaunch: &publicIP,
		VPCRef: &ec2v1alpha1.AWSResourceReferenceWrapper{
			From: &ec2v1alpha1.AWSResourceReference{
				Name: strPtr(VPC.ObjectMeta.Name),
			},
		},
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-public-c")},
			{Key: strPtr("kubernetes.io/role/elb"), Value: strPtr("1")},
		},
	},
}

// Private Subnets (for worker nodes)

// PrivateSubnetA is the first private subnet for worker nodes.
var PrivateSubnetA = ec2v1alpha1.Subnet{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "Subnet",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-private-a",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.SubnetSpec{
		CIDRBlock:           &privateCIDRA,
		AvailabilityZone:    &azA,
		MapPublicIPOnLaunch: &privateIP,
		VPCRef: &ec2v1alpha1.AWSResourceReferenceWrapper{
			From: &ec2v1alpha1.AWSResourceReference{
				Name: strPtr(VPC.ObjectMeta.Name),
			},
		},
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-private-a")},
			{Key: strPtr("kubernetes.io/role/internal-elb"), Value: strPtr("1")},
		},
	},
}

// PrivateSubnetB is the second private subnet for worker nodes.
var PrivateSubnetB = ec2v1alpha1.Subnet{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "Subnet",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-private-b",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.SubnetSpec{
		CIDRBlock:           &privateCIDRB,
		AvailabilityZone:    &azB,
		MapPublicIPOnLaunch: &privateIP,
		VPCRef: &ec2v1alpha1.AWSResourceReferenceWrapper{
			From: &ec2v1alpha1.AWSResourceReference{
				Name: strPtr(VPC.ObjectMeta.Name),
			},
		},
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-private-b")},
			{Key: strPtr("kubernetes.io/role/internal-elb"), Value: strPtr("1")},
		},
	},
}

// PrivateSubnetC is the third private subnet for worker nodes.
var PrivateSubnetC = ec2v1alpha1.Subnet{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "Subnet",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-private-c",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.SubnetSpec{
		CIDRBlock:           &privateCIDRC,
		AvailabilityZone:    &azC,
		MapPublicIPOnLaunch: &privateIP,
		VPCRef: &ec2v1alpha1.AWSResourceReferenceWrapper{
			From: &ec2v1alpha1.AWSResourceReference{
				Name: strPtr(VPC.ObjectMeta.Name),
			},
		},
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-private-c")},
			{Key: strPtr("kubernetes.io/role/internal-elb"), Value: strPtr("1")},
		},
	},
}

// ControlPlaneSecurityGroup is the security group for the EKS control plane.
var ControlPlaneSecurityGroup = ec2v1alpha1.SecurityGroup{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "ec2.services.k8s.aws/v1alpha1",
		Kind:       "SecurityGroup",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-control-plane-sg",
		Namespace: "ack-system",
	},
	Spec: ec2v1alpha1.SecurityGroupSpec{
		Name:        strPtr("eks-k8s-control-plane-sg"),
		Description: strPtr("Security group for EKS control plane"),
		VPCRef: &ec2v1alpha1.AWSResourceReferenceWrapper{
			From: &ec2v1alpha1.AWSResourceReference{
				Name: strPtr(VPC.ObjectMeta.Name),
			},
		},
		IngressRules: []*ec2v1alpha1.IPPermission{
			{
				IPProtocol: strPtr("tcp"),
				FromPort:   int64Ptr(443),
				ToPort:     int64Ptr(443),
				IPRanges: []*ec2v1alpha1.IPRange{
					{CIDRIP: &vpcCIDR, Description: strPtr("Allow HTTPS from VPC")},
				},
			},
		},
		EgressRules: []*ec2v1alpha1.IPPermission{
			{
				IPProtocol: strPtr("-1"),
				IPRanges: []*ec2v1alpha1.IPRange{
					{CIDRIP: strPtr("0.0.0.0/0"), Description: strPtr("Allow all outbound")},
				},
			},
		},
		Tags: []*ec2v1alpha1.Tag{
			{Key: strPtr("Name"), Value: strPtr("eks-k8s-control-plane-sg")},
			{Key: strPtr("Environment"), Value: strPtr("production")},
		},
	},
}

// strPtr is a helper to create string pointers.
func strPtr(s string) *string {
	return &s
}

// int64Ptr is a helper to create int64 pointers.
func int64Ptr(i int64) *int64 {
	return &i
}
