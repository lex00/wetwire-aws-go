// Package v1alpha1 contains ACK EC2 resource types for Kubernetes-native AWS infrastructure management.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VPC represents an ACK EC2 VPC resource.
// +kubebuilder:object:root=true
type VPC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VPCSpec   `json:"spec,omitempty"`
	Status VPCStatus `json:"status,omitempty"`
}

// VPCSpec defines the desired state of a VPC.
type VPCSpec struct {
	// CIDRBlocks are the IPv4 CIDR blocks for the VPC.
	CIDRBlocks []*string `json:"cidrBlocks,omitempty"`

	// EnableDNSHostnames indicates whether instances have DNS hostnames.
	EnableDNSHostnames *bool `json:"enableDNSHostnames,omitempty"`

	// EnableDNSSupport indicates whether DNS resolution is supported.
	EnableDNSSupport *bool `json:"enableDNSSupport,omitempty"`

	// InstanceTenancy is the tenancy option for instances (default, dedicated, host).
	InstanceTenancy *string `json:"instanceTenancy,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags []*Tag `json:"tags,omitempty"`
}

// VPCStatus defines the observed state of a VPC.
type VPCStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// VPCID is the ID of the VPC.
	VPCID *string `json:"vpcID,omitempty"`

	// CIDRBlockAssociationSet describes the CIDR block associations.
	CIDRBlockAssociationSet []*VPCCIDRBlockAssociation `json:"cidrBlockAssociationSet,omitempty"`

	// DHCPOptionsID is the ID of the DHCP options set.
	DHCPOptionsID *string `json:"dhcpOptionsID,omitempty"`

	// IsDefault indicates whether this is the default VPC.
	IsDefault *bool `json:"isDefault,omitempty"`

	// OwnerID is the ID of the AWS account that owns the VPC.
	OwnerID *string `json:"ownerID,omitempty"`

	// State is the current state of the VPC.
	State *string `json:"state,omitempty"`
}

// VPCCIDRBlockAssociation describes a CIDR block association.
type VPCCIDRBlockAssociation struct {
	// AssociationID is the association ID.
	AssociationID *string `json:"associationID,omitempty"`

	// CIDRBlock is the IPv4 CIDR block.
	CIDRBlock *string `json:"cidrBlock,omitempty"`

	// CIDRBlockState describes the state of the CIDR block.
	CIDRBlockState *VPCCIDRBlockState `json:"cidrBlockState,omitempty"`
}

// VPCCIDRBlockState describes the state of a CIDR block.
type VPCCIDRBlockState struct {
	// State is the state of the CIDR block.
	State *string `json:"state,omitempty"`

	// StatusMessage is a message about the status.
	StatusMessage *string `json:"statusMessage,omitempty"`
}

// Tag represents an AWS tag.
type Tag struct {
	// Key is the tag key.
	Key *string `json:"key,omitempty"`

	// Value is the tag value.
	Value *string `json:"value,omitempty"`
}

// ACKResourceMetadata contains ACK-specific metadata.
type ACKResourceMetadata struct {
	// ARN is the Amazon Resource Name.
	ARN *string `json:"arn,omitempty"`

	// OwnerAccountID is the AWS account ID of the resource owner.
	OwnerAccountID *string `json:"ownerAccountID,omitempty"`

	// Region is the AWS region.
	Region *string `json:"region,omitempty"`
}

// Condition represents a condition.
type Condition struct {
	// Type is the type of condition.
	Type *string `json:"type,omitempty"`

	// Status is the status of the condition.
	Status *string `json:"status,omitempty"`

	// LastTransitionTime is when the condition last transitioned.
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Message is a human-readable message.
	Message *string `json:"message,omitempty"`

	// Reason is a brief reason for the condition.
	Reason *string `json:"reason,omitempty"`
}

// AWSResourceReferenceWrapper wraps an AWS resource reference.
type AWSResourceReferenceWrapper struct {
	// From contains the reference information.
	From *AWSResourceReference `json:"from,omitempty"`
}

// AWSResourceReference references an AWS resource.
type AWSResourceReference struct {
	// Name is the name of the resource.
	Name *string `json:"name,omitempty"`
}
