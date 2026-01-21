// Package v1alpha1 contains ACK IAM resource types for Kubernetes-native AWS infrastructure management.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Role represents an ACK IAM Role resource.
// +kubebuilder:object:root=true
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleSpec   `json:"spec,omitempty"`
	Status RoleStatus `json:"status,omitempty"`
}

// RoleSpec defines the desired state of an IAM Role.
type RoleSpec struct {
	// Name is the name of the role.
	Name string `json:"name"`

	// AssumeRolePolicyDocument is the trust relationship policy document.
	AssumeRolePolicyDocument *string `json:"assumeRolePolicyDocument,omitempty"`

	// Description is a description of the role.
	Description *string `json:"description,omitempty"`

	// MaxSessionDuration is the maximum session duration in seconds.
	MaxSessionDuration *int64 `json:"maxSessionDuration,omitempty"`

	// Path is the path to the role.
	Path *string `json:"path,omitempty"`

	// PermissionsBoundary is the ARN of the policy used as the permissions boundary.
	PermissionsBoundary *string `json:"permissionsBoundary,omitempty"`

	// PermissionsBoundaryRef is a reference to a Policy resource.
	PermissionsBoundaryRef *AWSResourceReferenceWrapper `json:"permissionsBoundaryRef,omitempty"`

	// Policies are the inline policies attached to the role.
	Policies []*string `json:"policies,omitempty"`

	// PolicyRefs are references to Policy resources.
	PolicyRefs []*AWSResourceReferenceWrapper `json:"policyRefs,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags []*Tag `json:"tags,omitempty"`
}

// RoleStatus defines the observed state of an IAM Role.
type RoleStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// RoleID is the stable and unique ID of the role.
	RoleID *string `json:"roleID,omitempty"`

	// RoleLastUsed describes when the role was last used.
	RoleLastUsed *RoleLastUsed `json:"roleLastUsed,omitempty"`

	// CreateDate is the date and time the role was created.
	CreateDate *metav1.Time `json:"createDate,omitempty"`
}

// RoleLastUsed describes when a role was last used.
type RoleLastUsed struct {
	// LastUsedDate is the date and time the role was last used.
	LastUsedDate *metav1.Time `json:"lastUsedDate,omitempty"`

	// Region is the region where the role was last used.
	Region *string `json:"region,omitempty"`
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
