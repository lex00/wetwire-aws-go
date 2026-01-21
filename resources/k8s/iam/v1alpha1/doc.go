// Package v1alpha1 contains ACK IAM resource types for Kubernetes-native AWS infrastructure management.
//
// These types enable managing IAM Roles and Policies using
// Kubernetes CRDs via AWS Controllers for Kubernetes (ACK).
//
// Example usage:
//
//	import (
//		iamv1alpha1 "github.com/lex00/wetwire-aws-go/resources/k8s/iam/v1alpha1"
//		metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	)
//
//	var MyRole = iamv1alpha1.Role{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-role",
//			Namespace: "ack-system",
//		},
//		Spec: iamv1alpha1.RoleSpec{
//			Name:                     "my-eks-role",
//			AssumeRolePolicyDocument: strPtr(`{"Version":"2012-10-17"...}`),
//		},
//	}
package v1alpha1
