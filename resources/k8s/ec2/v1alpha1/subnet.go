package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Subnet represents an ACK EC2 Subnet resource.
// +kubebuilder:object:root=true
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec,omitempty"`
	Status SubnetStatus `json:"status,omitempty"`
}

// SubnetSpec defines the desired state of a Subnet.
type SubnetSpec struct {
	// AvailabilityZone is the Availability Zone for the subnet.
	AvailabilityZone *string `json:"availabilityZone,omitempty"`

	// AvailabilityZoneID is the AZ ID for the subnet.
	AvailabilityZoneID *string `json:"availabilityZoneID,omitempty"`

	// CIDRBlock is the IPv4 CIDR block for the subnet.
	CIDRBlock *string `json:"cidrBlock,omitempty"`

	// VPCID is the ID of the VPC.
	VPCID *string `json:"vpcID,omitempty"`

	// VPCRef is a reference to a VPC resource.
	VPCRef *AWSResourceReferenceWrapper `json:"vpcRef,omitempty"`

	// MapPublicIPOnLaunch indicates whether instances receive public IPs.
	MapPublicIPOnLaunch *bool `json:"mapPublicIPOnLaunch,omitempty"`

	// AssignIPv6AddressOnCreation indicates whether IPv6 addresses are assigned.
	AssignIPv6AddressOnCreation *bool `json:"assignIPv6AddressOnCreation,omitempty"`

	// IPv6CIDRBlock is the IPv6 CIDR block for the subnet.
	IPv6CIDRBlock *string `json:"ipv6CIDRBlock,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags []*Tag `json:"tags,omitempty"`
}

// SubnetStatus defines the observed state of a Subnet.
type SubnetStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// SubnetID is the ID of the subnet.
	SubnetID *string `json:"subnetID,omitempty"`

	// AvailableIPAddressCount is the number of available IPs.
	AvailableIPAddressCount *int64 `json:"availableIPAddressCount,omitempty"`

	// DefaultForAZ indicates whether this is the default subnet for the AZ.
	DefaultForAZ *bool `json:"defaultForAZ,omitempty"`

	// OwnerID is the ID of the AWS account that owns the subnet.
	OwnerID *string `json:"ownerID,omitempty"`

	// State is the current state of the subnet.
	State *string `json:"state,omitempty"`
}
