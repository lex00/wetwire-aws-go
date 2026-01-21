// Package v1alpha1 contains ACK EC2 resource types for Kubernetes-native AWS infrastructure management.
//
// These types enable managing VPCs, Subnets, and Security Groups using
// Kubernetes CRDs via AWS Controllers for Kubernetes (ACK).
//
// Example usage:
//
//	import (
//		ec2v1alpha1 "github.com/lex00/wetwire-aws-go/resources/k8s/ec2/v1alpha1"
//		metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	)
//
//	var MyVPC = ec2v1alpha1.VPC{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-vpc",
//			Namespace: "ack-system",
//		},
//		Spec: ec2v1alpha1.VPCSpec{
//			CIDRBlocks:         []*string{strPtr("10.0.0.0/16")},
//			EnableDNSHostnames: boolPtr(true),
//		},
//	}
package v1alpha1
