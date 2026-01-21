package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Nodegroup represents an ACK EKS Nodegroup resource.
// +kubebuilder:object:root=true
type Nodegroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodegroupSpec   `json:"spec,omitempty"`
	Status NodegroupStatus `json:"status,omitempty"`
}

// NodegroupSpec defines the desired state of an EKS Nodegroup.
type NodegroupSpec struct {
	// Name is the unique name for the managed node group.
	Name string `json:"name"`

	// ClusterName is the name of the cluster to create the node group in.
	ClusterName *string `json:"clusterName,omitempty"`

	// ClusterRef is a reference to a Cluster resource.
	ClusterRef *AWSResourceReferenceWrapper `json:"clusterRef,omitempty"`

	// NodeRole is the Amazon Resource Name (ARN) of the IAM role to associate.
	NodeRole *string `json:"nodeRole,omitempty"`

	// NodeRoleRef is a reference to an IAM Role resource.
	NodeRoleRef *AWSResourceReferenceWrapper `json:"nodeRoleRef,omitempty"`

	// Subnets are the subnets to use for the Auto Scaling group.
	Subnets []*string `json:"subnets,omitempty"`

	// SubnetRefs are references to Subnet resources.
	SubnetRefs []*AWSResourceReferenceWrapper `json:"subnetRefs,omitempty"`

	// ScalingConfig specifies the scaling configuration.
	ScalingConfig *NodegroupScalingConfig `json:"scalingConfig,omitempty"`

	// InstanceTypes specifies the instance types for a node group.
	InstanceTypes []*string `json:"instanceTypes,omitempty"`

	// AMIType specifies the AMI type for the node group.
	AMIType *string `json:"amiType,omitempty"`

	// CapacityType specifies the capacity type (ON_DEMAND or SPOT).
	CapacityType *string `json:"capacityType,omitempty"`

	// DiskSize specifies the root device disk size in GiB.
	DiskSize *int64 `json:"diskSize,omitempty"`

	// Labels are key-value pairs to apply to nodes.
	Labels map[string]*string `json:"labels,omitempty"`

	// Taints are Kubernetes taints to apply to nodes.
	Taints []*Taint `json:"taints,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags map[string]*string `json:"tags,omitempty"`

	// LaunchTemplate specifies a custom launch template.
	LaunchTemplate *LaunchTemplateSpecification `json:"launchTemplate,omitempty"`

	// UpdateConfig specifies the update configuration.
	UpdateConfig *NodegroupUpdateConfig `json:"updateConfig,omitempty"`

	// ReleaseVersion specifies the AMI version.
	ReleaseVersion *string `json:"releaseVersion,omitempty"`

	// Version specifies the Kubernetes version.
	Version *string `json:"version,omitempty"`
}

// NodegroupStatus defines the observed state of an EKS Nodegroup.
type NodegroupStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// NodegroupARN is the Amazon Resource Name of the node group.
	NodegroupARN *string `json:"nodegroupARN,omitempty"`

	// Status is the current status of the node group.
	Status *string `json:"status,omitempty"`

	// Health describes the health of the node group.
	Health *NodegroupHealth `json:"health,omitempty"`

	// Resources describes the resources associated with the node group.
	Resources *NodegroupResources `json:"resources,omitempty"`
}

// NodegroupScalingConfig specifies the scaling configuration.
type NodegroupScalingConfig struct {
	// MinSize is the minimum number of nodes.
	MinSize *int64 `json:"minSize,omitempty"`

	// MaxSize is the maximum number of nodes.
	MaxSize *int64 `json:"maxSize,omitempty"`

	// DesiredSize is the current number of nodes.
	DesiredSize *int64 `json:"desiredSize,omitempty"`
}

// Taint represents a Kubernetes taint.
type Taint struct {
	// Key is the taint key.
	Key *string `json:"key,omitempty"`

	// Value is the taint value.
	Value *string `json:"value,omitempty"`

	// Effect is the taint effect (NO_SCHEDULE, NO_EXECUTE, PREFER_NO_SCHEDULE).
	Effect *string `json:"effect,omitempty"`
}

// LaunchTemplateSpecification specifies a launch template.
type LaunchTemplateSpecification struct {
	// ID is the ID of the launch template.
	ID *string `json:"id,omitempty"`

	// Name is the name of the launch template.
	Name *string `json:"name,omitempty"`

	// Version is the version number of the launch template.
	Version *string `json:"version,omitempty"`
}

// NodegroupUpdateConfig specifies the update configuration.
type NodegroupUpdateConfig struct {
	// MaxUnavailable is the maximum number of unavailable nodes during update.
	MaxUnavailable *int64 `json:"maxUnavailable,omitempty"`

	// MaxUnavailablePercentage is the maximum percentage of unavailable nodes.
	MaxUnavailablePercentage *int64 `json:"maxUnavailablePercentage,omitempty"`
}

// NodegroupHealth describes the health of the node group.
type NodegroupHealth struct {
	// Issues is a list of health issues.
	Issues []*Issue `json:"issues,omitempty"`
}

// Issue describes a health issue.
type Issue struct {
	// Code is the error code.
	Code *string `json:"code,omitempty"`

	// Message is the error message.
	Message *string `json:"message,omitempty"`

	// ResourceIDs are the affected resource IDs.
	ResourceIDs []*string `json:"resourceIDs,omitempty"`
}

// NodegroupResources describes the resources associated with the node group.
type NodegroupResources struct {
	// AutoScalingGroups are the Auto Scaling groups.
	AutoScalingGroups []*AutoScalingGroup `json:"autoScalingGroups,omitempty"`

	// RemoteAccessSecurityGroup is the security group for remote access.
	RemoteAccessSecurityGroup *string `json:"remoteAccessSecurityGroup,omitempty"`
}

// AutoScalingGroup describes an Auto Scaling group.
type AutoScalingGroup struct {
	// Name is the name of the Auto Scaling group.
	Name *string `json:"name,omitempty"`
}
