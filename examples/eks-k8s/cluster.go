package eks_k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eksv1alpha1 "github.com/lex00/wetwire-aws-go/resources/k8s/eks/v1alpha1"
	iamv1alpha1 "github.com/lex00/wetwire-aws-go/resources/k8s/iam/v1alpha1"
)

// Helper variables for pointer fields
var (
	clusterName       = "eks-k8s-cluster"
	kubernetesVersion = "1.29"

	// Scaling configuration
	systemMinSize     int64 = 2
	systemMaxSize     int64 = 4
	systemDesiredSize int64 = 2
	appMinSize        int64 = 2
	appMaxSize        int64 = 10
	appDesiredSize    int64 = 3
	spotMinSize       int64 = 0
	spotMaxSize       int64 = 20
	spotDesiredSize   int64 = 0

	// Node configuration
	amiType       = "AL2_x86_64"
	onDemand      = "ON_DEMAND"
	spot          = "SPOT"
	diskSize      int64 = 100

	// Logging
	loggingEnabled = true

	// Endpoint access
	privateAccess = true
	publicAccess  = true
)

// ClusterRole is the IAM role for the EKS cluster.
var ClusterRole = iamv1alpha1.Role{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "iam.services.k8s.aws/v1alpha1",
		Kind:       "Role",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-cluster-role",
		Namespace: "ack-system",
	},
	Spec: iamv1alpha1.RoleSpec{
		Name:        "eks-k8s-cluster-role",
		Description: strPtr("IAM role for EKS cluster control plane"),
		AssumeRolePolicyDocument: strPtr(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"Service": "eks.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}
			]
		}`),
		Policies: []*string{
			strPtr("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
		},
		Tags: []*iamv1alpha1.Tag{
			{Key: strPtr("Environment"), Value: strPtr("production")},
			{Key: strPtr("ManagedBy"), Value: strPtr("wetwire-ack")},
		},
	},
}

// NodeRole is the IAM role for EKS worker nodes.
var NodeRole = iamv1alpha1.Role{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "iam.services.k8s.aws/v1alpha1",
		Kind:       "Role",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-node-role",
		Namespace: "ack-system",
	},
	Spec: iamv1alpha1.RoleSpec{
		Name:        "eks-k8s-node-role",
		Description: strPtr("IAM role for EKS worker nodes"),
		AssumeRolePolicyDocument: strPtr(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"Service": "ec2.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}
			]
		}`),
		Policies: []*string{
			strPtr("arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"),
			strPtr("arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"),
			strPtr("arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"),
			strPtr("arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"),
		},
		Tags: []*iamv1alpha1.Tag{
			{Key: strPtr("Environment"), Value: strPtr("production")},
			{Key: strPtr("ManagedBy"), Value: strPtr("wetwire-ack")},
		},
	},
}

// Cluster is the main EKS cluster resource, managed via ACK.
var Cluster = eksv1alpha1.Cluster{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Cluster",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-cluster",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.ClusterSpec{
		Name:    clusterName,
		Version: &kubernetesVersion,
		RoleRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(ClusterRole.ObjectMeta.Name),
			},
		},
		ResourcesVPCConfig: &eksv1alpha1.VPCConfigRequest{
			SubnetRefs: []*eksv1alpha1.AWSResourceReferenceWrapper{
				{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetA.ObjectMeta.Name)}},
				{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetB.ObjectMeta.Name)}},
				{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetC.ObjectMeta.Name)}},
			},
			SecurityGroupRefs: []*eksv1alpha1.AWSResourceReferenceWrapper{
				{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(ControlPlaneSecurityGroup.ObjectMeta.Name)}},
			},
			EndpointPrivateAccess: &privateAccess,
			EndpointPublicAccess:  &publicAccess,
		},
		Logging: &eksv1alpha1.Logging{
			ClusterLogging: []*eksv1alpha1.LogSetup{
				{
					Enabled: &loggingEnabled,
					Types: []*string{
						strPtr("api"),
						strPtr("audit"),
						strPtr("authenticator"),
						strPtr("controllerManager"),
						strPtr("scheduler"),
					},
				},
			},
		},
		Tags: map[string]*string{
			"Environment": strPtr("production"),
			"ManagedBy":   strPtr("wetwire-ack"),
		},
	},
}

// SystemNodegroup runs critical system workloads like CoreDNS.
var SystemNodegroup = eksv1alpha1.Nodegroup{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Nodegroup",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-system-ng",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.NodegroupSpec{
		Name: "system",
		ClusterRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(Cluster.ObjectMeta.Name),
			},
		},
		NodeRoleRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(NodeRole.ObjectMeta.Name),
			},
		},
		SubnetRefs: []*eksv1alpha1.AWSResourceReferenceWrapper{
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetA.ObjectMeta.Name)}},
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetB.ObjectMeta.Name)}},
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetC.ObjectMeta.Name)}},
		},
		ScalingConfig: &eksv1alpha1.NodegroupScalingConfig{
			MinSize:     &systemMinSize,
			MaxSize:     &systemMaxSize,
			DesiredSize: &systemDesiredSize,
		},
		InstanceTypes: []*string{strPtr("t3.medium")},
		AMIType:       &amiType,
		CapacityType:  &onDemand,
		DiskSize:      &diskSize,
		Labels: map[string]*string{
			"nodegroup-type": strPtr("system"),
		},
		Taints: []*eksv1alpha1.Taint{
			{
				Key:    strPtr("CriticalAddonsOnly"),
				Value:  strPtr("true"),
				Effect: strPtr("NO_SCHEDULE"),
			},
		},
		Tags: map[string]*string{
			"Environment": strPtr("production"),
			"NodePool":    strPtr("system"),
		},
	},
}

// AppNodegroup runs general application workloads.
var AppNodegroup = eksv1alpha1.Nodegroup{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Nodegroup",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-app-ng",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.NodegroupSpec{
		Name: "application",
		ClusterRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(Cluster.ObjectMeta.Name),
			},
		},
		NodeRoleRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(NodeRole.ObjectMeta.Name),
			},
		},
		SubnetRefs: []*eksv1alpha1.AWSResourceReferenceWrapper{
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetA.ObjectMeta.Name)}},
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetB.ObjectMeta.Name)}},
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetC.ObjectMeta.Name)}},
		},
		ScalingConfig: &eksv1alpha1.NodegroupScalingConfig{
			MinSize:     &appMinSize,
			MaxSize:     &appMaxSize,
			DesiredSize: &appDesiredSize,
		},
		InstanceTypes: []*string{strPtr("t3.large"), strPtr("t3.xlarge")},
		AMIType:       &amiType,
		CapacityType:  &onDemand,
		DiskSize:      &diskSize,
		Labels: map[string]*string{
			"nodegroup-type": strPtr("application"),
		},
		Tags: map[string]*string{
			"Environment": strPtr("production"),
			"NodePool":    strPtr("application"),
		},
	},
}

// SpotNodegroup runs fault-tolerant workloads on spot instances.
var SpotNodegroup = eksv1alpha1.Nodegroup{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Nodegroup",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-spot-ng",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.NodegroupSpec{
		Name: "spot",
		ClusterRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(Cluster.ObjectMeta.Name),
			},
		},
		NodeRoleRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(NodeRole.ObjectMeta.Name),
			},
		},
		SubnetRefs: []*eksv1alpha1.AWSResourceReferenceWrapper{
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetA.ObjectMeta.Name)}},
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetB.ObjectMeta.Name)}},
			{From: &eksv1alpha1.AWSResourceReference{Name: strPtr(PrivateSubnetC.ObjectMeta.Name)}},
		},
		ScalingConfig: &eksv1alpha1.NodegroupScalingConfig{
			MinSize:     &spotMinSize,
			MaxSize:     &spotMaxSize,
			DesiredSize: &spotDesiredSize,
		},
		InstanceTypes: []*string{
			strPtr("t3.large"),
			strPtr("t3.xlarge"),
			strPtr("t3a.large"),
			strPtr("t3a.xlarge"),
		},
		AMIType:      &amiType,
		CapacityType: &spot,
		DiskSize:     &diskSize,
		Labels: map[string]*string{
			"nodegroup-type": strPtr("spot"),
			"lifecycle":      strPtr("spot"),
		},
		Taints: []*eksv1alpha1.Taint{
			{
				Key:    strPtr("lifecycle"),
				Value:  strPtr("spot"),
				Effect: strPtr("NO_SCHEDULE"),
			},
		},
		Tags: map[string]*string{
			"Environment": strPtr("production"),
			"NodePool":    strPtr("spot"),
		},
	},
}

// EKS Add-ons

// VpcCniAddon installs the Amazon VPC CNI plugin.
var VpcCniAddon = eksv1alpha1.Addon{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Addon",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-vpc-cni",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.AddonSpec{
		Name: "vpc-cni",
		ClusterRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(Cluster.ObjectMeta.Name),
			},
		},
		AddonVersion:     strPtr("v1.16.0-eksbuild.1"),
		ResolveConflicts: strPtr("OVERWRITE"),
	},
}

// CoreDnsAddon installs CoreDNS.
var CoreDnsAddon = eksv1alpha1.Addon{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Addon",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-coredns",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.AddonSpec{
		Name: "coredns",
		ClusterRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(Cluster.ObjectMeta.Name),
			},
		},
		AddonVersion:     strPtr("v1.11.1-eksbuild.4"),
		ResolveConflicts: strPtr("OVERWRITE"),
	},
}

// KubeProxyAddon installs kube-proxy.
var KubeProxyAddon = eksv1alpha1.Addon{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Addon",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-kube-proxy",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.AddonSpec{
		Name: "kube-proxy",
		ClusterRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(Cluster.ObjectMeta.Name),
			},
		},
		AddonVersion:     strPtr("v1.29.0-eksbuild.1"),
		ResolveConflicts: strPtr("OVERWRITE"),
	},
}

// EbsCsiAddon installs the Amazon EBS CSI driver.
var EbsCsiAddon = eksv1alpha1.Addon{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "eks.services.k8s.aws/v1alpha1",
		Kind:       "Addon",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "eks-k8s-ebs-csi",
		Namespace: "ack-system",
	},
	Spec: eksv1alpha1.AddonSpec{
		Name: "aws-ebs-csi-driver",
		ClusterRef: &eksv1alpha1.AWSResourceReferenceWrapper{
			From: &eksv1alpha1.AWSResourceReference{
				Name: strPtr(Cluster.ObjectMeta.Name),
			},
		},
		AddonVersion:     strPtr("v1.27.0-eksbuild.1"),
		ResolveConflicts: strPtr("OVERWRITE"),
	},
}
