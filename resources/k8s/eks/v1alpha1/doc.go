// Package v1alpha1 contains ACK EKS resource types for Kubernetes-native AWS infrastructure management.
//
// These types enable managing EKS clusters, node groups, and add-ons using
// Kubernetes CRDs via AWS Controllers for Kubernetes (ACK).
//
// Example usage:
//
//	import (
//		eksv1alpha1 "github.com/lex00/wetwire-aws-go/resources/k8s/eks/v1alpha1"
//		metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	)
//
//	var MyCluster = eksv1alpha1.Cluster{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-cluster",
//			Namespace: "ack-system",
//		},
//		Spec: eksv1alpha1.ClusterSpec{
//			Name:    "my-eks-cluster",
//			Version: strPtr("1.29"),
//		},
//	}
package v1alpha1
