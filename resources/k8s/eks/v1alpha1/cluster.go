// Package v1alpha1 contains ACK EKS resource types for Kubernetes-native AWS infrastructure management.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cluster represents an ACK EKS Cluster resource.
// +kubebuilder:object:root=true
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterSpec defines the desired state of an EKS Cluster.
type ClusterSpec struct {
	// Name is the unique name for the cluster.
	Name string `json:"name"`

	// Version is the Kubernetes version for the cluster.
	Version *string `json:"version,omitempty"`

	// RoleARN is the Amazon Resource Name (ARN) of the IAM role that provides
	// permissions for the Kubernetes control plane.
	RoleARN *string `json:"roleARN,omitempty"`

	// RoleRef is a reference to an IAM Role resource.
	RoleRef *AWSResourceReferenceWrapper `json:"roleRef,omitempty"`

	// ResourcesVPCConfig defines the VPC configuration for the cluster.
	ResourcesVPCConfig *VPCConfigRequest `json:"resourcesVPCConfig,omitempty"`

	// Logging enables control plane logging.
	Logging *Logging `json:"logging,omitempty"`

	// EncryptionConfig enables envelope encryption for Kubernetes secrets.
	EncryptionConfig []*EncryptionConfig `json:"encryptionConfig,omitempty"`

	// Tags are key-value pairs to categorize resources.
	Tags map[string]*string `json:"tags,omitempty"`
}

// ClusterStatus defines the observed state of an EKS Cluster.
type ClusterStatus struct {
	// ACKResourceMetadata contains ACK-specific metadata.
	ACKResourceMetadata *ACKResourceMetadata `json:"ackResourceMetadata,omitempty"`

	// Conditions represent the latest available observations.
	Conditions []*Condition `json:"conditions,omitempty"`

	// ARN is the Amazon Resource Name of the cluster.
	ARN *string `json:"arn,omitempty"`

	// CertificateAuthority contains certificate data.
	CertificateAuthority *Certificate `json:"certificateAuthority,omitempty"`

	// Endpoint is the endpoint for the Kubernetes API server.
	Endpoint *string `json:"endpoint,omitempty"`

	// Identity contains the identity provider information.
	Identity *Identity `json:"identity,omitempty"`

	// PlatformVersion is the platform version of the cluster.
	PlatformVersion *string `json:"platformVersion,omitempty"`

	// Status is the current status of the cluster.
	Status *string `json:"status,omitempty"`
}

// VPCConfigRequest defines the VPC configuration request.
type VPCConfigRequest struct {
	// SubnetIDs are the IDs of subnets to use for the cluster.
	SubnetIDs []*string `json:"subnetIDs,omitempty"`

	// SubnetRefs are references to Subnet resources.
	SubnetRefs []*AWSResourceReferenceWrapper `json:"subnetRefs,omitempty"`

	// SecurityGroupIDs are the IDs of security groups to use.
	SecurityGroupIDs []*string `json:"securityGroupIDs,omitempty"`

	// SecurityGroupRefs are references to SecurityGroup resources.
	SecurityGroupRefs []*AWSResourceReferenceWrapper `json:"securityGroupRefs,omitempty"`

	// EndpointPrivateAccess enables private access to the cluster endpoint.
	EndpointPrivateAccess *bool `json:"endpointPrivateAccess,omitempty"`

	// EndpointPublicAccess enables public access to the cluster endpoint.
	EndpointPublicAccess *bool `json:"endpointPublicAccess,omitempty"`

	// PublicAccessCIDRs are the CIDR blocks allowed public access.
	PublicAccessCIDRs []*string `json:"publicAccessCidrs,omitempty"`
}

// Logging defines control plane logging configuration.
type Logging struct {
	// ClusterLogging defines the cluster logging configuration.
	ClusterLogging []*LogSetup `json:"clusterLogging,omitempty"`
}

// LogSetup defines log types and enablement.
type LogSetup struct {
	// Enabled indicates whether the logs are enabled.
	Enabled *bool `json:"enabled,omitempty"`

	// Types are the available log types.
	Types []*string `json:"types,omitempty"`
}

// EncryptionConfig defines encryption configuration.
type EncryptionConfig struct {
	// Provider defines the encryption provider.
	Provider *Provider `json:"provider,omitempty"`

	// Resources are the resources to encrypt.
	Resources []*string `json:"resources,omitempty"`
}

// Provider defines the encryption provider.
type Provider struct {
	// KeyARN is the ARN of the KMS key.
	KeyARN *string `json:"keyARN,omitempty"`
}

// Certificate contains certificate data.
type Certificate struct {
	// Data is the base64-encoded certificate data.
	Data *string `json:"data,omitempty"`
}

// Identity contains identity provider information.
type Identity struct {
	// OIDC contains OIDC identity provider information.
	OIDC *OIDC `json:"oidc,omitempty"`
}

// OIDC contains OIDC identity provider information.
type OIDC struct {
	// Issuer is the issuer URL.
	Issuer *string `json:"issuer,omitempty"`
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
