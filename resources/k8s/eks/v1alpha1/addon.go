package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Addon represents an ACK EKS Addon resource.
// +kubebuilder:object:root=true
type Addon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonSpec   `json:"spec,omitempty"`
	Status AddonStatus `json:"status,omitempty"`
}

// AddonSpec defines the desired state of an EKS Addon.
type AddonSpec struct {
	// Name is the name of the add-on.
	Name string `json:"name"`

	// ClusterName is the name of the cluster.
	ClusterName *string `json:"clusterName,omitempty"`

	// ClusterRef is a reference to a Cluster resource.
	ClusterRef *AWSResourceReferenceWrapper `json:"clusterRef,omitempty"`

	// AddonVersion is the version of the add-on.
	AddonVersion *string `json:"addonVersion,omitempty"`

	// ServiceAccountRoleARN is the Amazon Resource Name (ARN) of an existing
	// IAM role to bind to the add-on's service account.
	ServiceAccountRoleARN *string `json:"serviceAccountRoleARN,omitempty"`

	// ServiceAccountRoleRef is a reference to an IAM Role resource.
	ServiceAccountRoleRef *AWSResourceReferenceWrapper `json:"serviceAccountRoleRef,omitempty"`

	// ResolveConflicts specifies how to resolve conflicts during installation.
	// Valid values: NONE, OVERWRITE, PRESERVE.
	ResolveConflicts *string `json:"resolveConflicts,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags map[string]*string `json:"tags,omitempty"`

	// ConfigurationValues is a JSON string for add-on configuration.
	ConfigurationValues *string `json:"configurationValues,omitempty"`
}

// AddonStatus defines the observed state of an EKS Addon.
type AddonStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// AddonARN is the Amazon Resource Name of the add-on.
	AddonARN *string `json:"addonARN,omitempty"`

	// Status is the current status of the add-on.
	Status *string `json:"status,omitempty"`

	// Health describes the health of the add-on.
	Health *AddonHealth `json:"health,omitempty"`

	// CreatedAt is the date and time the add-on was created.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// ModifiedAt is the date and time the add-on was last modified.
	ModifiedAt *metav1.Time `json:"modifiedAt,omitempty"`
}

// AddonHealth describes the health of an add-on.
type AddonHealth struct {
	// Issues is a list of health issues.
	Issues []*AddonIssue `json:"issues,omitempty"`
}

// AddonIssue describes an add-on health issue.
type AddonIssue struct {
	// Code is the error code.
	Code *string `json:"code,omitempty"`

	// Message is the error message.
	Message *string `json:"message,omitempty"`

	// ResourceIDs are the affected resource IDs.
	ResourceIDs []*string `json:"resourceIDs,omitempty"`
}
