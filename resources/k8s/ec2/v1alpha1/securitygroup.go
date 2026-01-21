package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityGroup represents an ACK EC2 SecurityGroup resource.
// +kubebuilder:object:root=true
type SecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityGroupSpec   `json:"spec,omitempty"`
	Status SecurityGroupStatus `json:"status,omitempty"`
}

// SecurityGroupSpec defines the desired state of a SecurityGroup.
type SecurityGroupSpec struct {
	// Description is the description for the security group.
	Description *string `json:"description,omitempty"`

	// Name is the name of the security group.
	Name *string `json:"name,omitempty"`

	// VPCID is the ID of the VPC.
	VPCID *string `json:"vpcID,omitempty"`

	// VPCRef is a reference to a VPC resource.
	VPCRef *AWSResourceReferenceWrapper `json:"vpcRef,omitempty"`

	// IngressRules are the inbound rules.
	IngressRules []*IPPermission `json:"ingressRules,omitempty"`

	// EgressRules are the outbound rules.
	EgressRules []*IPPermission `json:"egressRules,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags []*Tag `json:"tags,omitempty"`
}

// SecurityGroupStatus defines the observed state of a SecurityGroup.
type SecurityGroupStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// ID is the ID of the security group.
	ID *string `json:"id,omitempty"`

	// OwnerID is the ID of the AWS account that owns the security group.
	OwnerID *string `json:"ownerID,omitempty"`
}

// IPPermission describes an IP permission (security group rule).
type IPPermission struct {
	// FromPort is the start of the port range.
	FromPort *int64 `json:"fromPort,omitempty"`

	// ToPort is the end of the port range.
	ToPort *int64 `json:"toPort,omitempty"`

	// IPProtocol is the IP protocol name or number.
	IPProtocol *string `json:"ipProtocol,omitempty"`

	// IPRanges are the IPv4 ranges.
	IPRanges []*IPRange `json:"ipRanges,omitempty"`

	// IPv6Ranges are the IPv6 ranges.
	IPv6Ranges []*IPv6Range `json:"ipv6Ranges,omitempty"`

	// PrefixListIDs are the prefix list IDs.
	PrefixListIDs []*PrefixListID `json:"prefixListIDs,omitempty"`

	// UserIDGroupPairs are the security group and AWS account ID pairs.
	UserIDGroupPairs []*UserIDGroupPair `json:"userIDGroupPairs,omitempty"`
}

// IPRange describes an IPv4 address range.
type IPRange struct {
	// CIDRip is the IPv4 CIDR range.
	CIDRIP *string `json:"cidrIP,omitempty"`

	// Description is a description for the rule.
	Description *string `json:"description,omitempty"`
}

// IPv6Range describes an IPv6 address range.
type IPv6Range struct {
	// CIDRIPv6 is the IPv6 CIDR range.
	CIDRIPv6 *string `json:"cidrIPv6,omitempty"`

	// Description is a description for the rule.
	Description *string `json:"description,omitempty"`
}

// PrefixListID describes a prefix list ID.
type PrefixListID struct {
	// Description is a description for the rule.
	Description *string `json:"description,omitempty"`

	// PrefixListID is the ID of the prefix list.
	PrefixListID *string `json:"prefixListID,omitempty"`
}

// UserIDGroupPair describes a security group and AWS account ID pair.
type UserIDGroupPair struct {
	// Description is a description for the rule.
	Description *string `json:"description,omitempty"`

	// GroupID is the ID of the security group.
	GroupID *string `json:"groupID,omitempty"`

	// GroupName is the name of the security group.
	GroupName *string `json:"groupName,omitempty"`

	// UserID is the ID of an AWS account.
	UserID *string `json:"userID,omitempty"`

	// VPCID is the ID of the VPC.
	VPCID *string `json:"vpcID,omitempty"`

	// VPCPeeringConnectionID is the ID of the VPC peering connection.
	VPCPeeringConnectionID *string `json:"vpcPeeringConnectionID,omitempty"`
}
