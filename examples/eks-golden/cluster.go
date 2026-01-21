package eks_golden

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/eks"
)

// EKS Cluster

// ClusterLogging enables all control plane logging.
var ClusterLogging = eks.Cluster_Logging{
	ClusterLogging: eks.Cluster_ClusterLogging{
		EnabledTypes: []any{
			eks.Cluster_LoggingTypeConfig{Type_: "api"},
			eks.Cluster_LoggingTypeConfig{Type_: "audit"},
			eks.Cluster_LoggingTypeConfig{Type_: "authenticator"},
			eks.Cluster_LoggingTypeConfig{Type_: "controllerManager"},
			eks.Cluster_LoggingTypeConfig{Type_: "scheduler"},
		},
	},
}

// ClusterVpcConfig defines the VPC configuration for the cluster.
var ClusterVpcConfig = eks.Cluster_ResourcesVpcConfig{
	SubnetIds: []any{
		PrivateSubnetA,
		PrivateSubnetB,
		PrivateSubnetC,
	},
	SecurityGroupIds:      []any{ControlPlaneSecurityGroup},
	EndpointPublicAccess:  true,
	EndpointPrivateAccess: true,
}

// Cluster is the main EKS cluster resource.
var Cluster = eks.Cluster{
	Name:               Sub{String: "${AWS::StackName}"},
	Version:            "1.29",
	RoleArn:            ClusterRole.Arn,
	ResourcesVpcConfig: ClusterVpcConfig,
	Logging:            ClusterLogging,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}"}},
		Tag{Key: "Environment", Value: "production"},
	},
}

// Node Groups

// SystemNodegroupScalingConfig defines scaling for the system node group.
var SystemNodegroupScalingConfig = eks.Nodegroup_ScalingConfig{
	MinSize:     2,
	MaxSize:     4,
	DesiredSize: 2,
}

// SystemNodegroup runs critical system workloads like CoreDNS.
var SystemNodegroup = eks.Nodegroup{
	ClusterName:   Cluster,
	NodegroupName: Sub{String: "${AWS::StackName}-system"},
	NodeRole:      NodeRole.Arn,
	Subnets: []any{
		PrivateSubnetA,
		PrivateSubnetB,
		PrivateSubnetC,
	},
	InstanceTypes: []any{"t3.medium"},
	AmiType:       "AL2_x86_64",
	CapacityType:  "ON_DEMAND",
	ScalingConfig: SystemNodegroupScalingConfig,
	Labels: map[string]any{
		"nodegroup-type": "system",
	},
	Taints: []any{
		eks.Nodegroup_Taint{
			Key:    "CriticalAddonsOnly",
			Value:  "true",
			Effect: "NO_SCHEDULE",
		},
	},
	Tags: map[string]any{
		"Name":        Sub{String: "${AWS::StackName}-system-ng"},
		"Environment": "production",
	},
}

// ApplicationNodegroupScalingConfig defines scaling for the application node group.
var ApplicationNodegroupScalingConfig = eks.Nodegroup_ScalingConfig{
	MinSize:     2,
	MaxSize:     10,
	DesiredSize: 3,
}

// ApplicationNodegroup runs general application workloads.
var ApplicationNodegroup = eks.Nodegroup{
	ClusterName:   Cluster,
	NodegroupName: Sub{String: "${AWS::StackName}-app"},
	NodeRole:      NodeRole.Arn,
	Subnets: []any{
		PrivateSubnetA,
		PrivateSubnetB,
		PrivateSubnetC,
	},
	InstanceTypes: []any{"t3.large", "t3.xlarge"},
	AmiType:       "AL2_x86_64",
	CapacityType:  "ON_DEMAND",
	ScalingConfig: ApplicationNodegroupScalingConfig,
	Labels: map[string]any{
		"nodegroup-type": "application",
	},
	Tags: map[string]any{
		"Name":        Sub{String: "${AWS::StackName}-app-ng"},
		"Environment": "production",
	},
}

// SpotNodegroupScalingConfig defines scaling for the spot node group.
var SpotNodegroupScalingConfig = eks.Nodegroup_ScalingConfig{
	MinSize:     0,
	MaxSize:     20,
	DesiredSize: 3,
}

// SpotNodegroup runs fault-tolerant workloads on spot instances.
var SpotNodegroup = eks.Nodegroup{
	ClusterName:   Cluster,
	NodegroupName: Sub{String: "${AWS::StackName}-spot"},
	NodeRole:      NodeRole.Arn,
	Subnets: []any{
		PrivateSubnetA,
		PrivateSubnetB,
		PrivateSubnetC,
	},
	InstanceTypes: []any{"t3.large", "t3.xlarge", "t3a.large", "t3a.xlarge"},
	AmiType:       "AL2_x86_64",
	CapacityType:  "SPOT",
	ScalingConfig: SpotNodegroupScalingConfig,
	Labels: map[string]any{
		"nodegroup-type": "spot",
		"lifecycle":      "spot",
	},
	Taints: []any{
		eks.Nodegroup_Taint{
			Key:    "lifecycle",
			Value:  "spot",
			Effect: "NO_SCHEDULE",
		},
	},
	Tags: map[string]any{
		"Name":        Sub{String: "${AWS::StackName}-spot-ng"},
		"Environment": "production",
	},
}

// EKS Add-ons

// VpcCniAddon installs the Amazon VPC CNI plugin.
var VpcCniAddon = eks.Addon{
	ClusterName:      Cluster,
	AddonName:        "vpc-cni",
	AddonVersion:     "v1.16.0-eksbuild.1",
	ResolveConflicts: "OVERWRITE",
}

// CoreDnsAddon installs CoreDNS.
var CoreDnsAddon = eks.Addon{
	ClusterName:      Cluster,
	AddonName:        "coredns",
	AddonVersion:     "v1.11.1-eksbuild.4",
	ResolveConflicts: "OVERWRITE",
}

// KubeProxyAddon installs kube-proxy.
var KubeProxyAddon = eks.Addon{
	ClusterName:      Cluster,
	AddonName:        "kube-proxy",
	AddonVersion:     "v1.29.0-eksbuild.1",
	ResolveConflicts: "OVERWRITE",
}

// EbsCsiAddon installs the Amazon EBS CSI driver.
var EbsCsiAddon = eks.Addon{
	ClusterName:      Cluster,
	AddonName:        "aws-ebs-csi-driver",
	AddonVersion:     "v1.27.0-eksbuild.1",
	ResolveConflicts: "OVERWRITE",
}

// PodIdentityAddon installs EKS Pod Identity Agent.
var PodIdentityAddon = eks.Addon{
	ClusterName:      Cluster,
	AddonName:        "eks-pod-identity-agent",
	AddonVersion:     "v1.2.0-eksbuild.1",
	ResolveConflicts: "OVERWRITE",
}
