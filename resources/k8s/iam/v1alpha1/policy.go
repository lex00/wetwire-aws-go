package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Policy represents an ACK IAM Policy resource.
// +kubebuilder:object:root=true
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicySpec   `json:"spec,omitempty"`
	Status PolicyStatus `json:"status,omitempty"`
}

// PolicySpec defines the desired state of an IAM Policy.
type PolicySpec struct {
	// Name is the name of the policy.
	Name string `json:"name"`

	// Description is a description of the policy.
	Description *string `json:"description,omitempty"`

	// Path is the path to the policy.
	Path *string `json:"path,omitempty"`

	// PolicyDocument is the JSON policy document.
	PolicyDocument *string `json:"policyDocument,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags []*Tag `json:"tags,omitempty"`
}

// PolicyStatus defines the observed state of an IAM Policy.
type PolicyStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// PolicyID is the stable and unique ID of the policy.
	PolicyID *string `json:"policyID,omitempty"`

	// AttachmentCount is the number of entities the policy is attached to.
	AttachmentCount *int64 `json:"attachmentCount,omitempty"`

	// CreateDate is the date and time the policy was created.
	CreateDate *metav1.Time `json:"createDate,omitempty"`

	// DefaultVersionID is the identifier for the current default version.
	DefaultVersionID *string `json:"defaultVersionID,omitempty"`

	// IsAttachable indicates whether the policy can be attached to entities.
	IsAttachable *bool `json:"isAttachable,omitempty"`

	// PermissionsBoundaryUsageCount is the number of entities that use this
	// policy as a permissions boundary.
	PermissionsBoundaryUsageCount *int64 `json:"permissionsBoundaryUsageCount,omitempty"`

	// UpdateDate is the date and time the policy was last updated.
	UpdateDate *metav1.Time `json:"updateDate,omitempty"`
}
