package eks_k8s

// This file demonstrates how to integrate wetwire-observability-go monitoring
// modules with the EKS K8s example.
//
// The monitoring configuration imports from wetwire-observability-go/monitoring/eks
// which provides EKS-specific scrape configs, alerts, recording rules, and dashboards.

/*
Usage with wetwire-observability-go:

import (
	"github.com/lex00/wetwire-observability-go/monitoring/eks"
	"github.com/lex00/wetwire-observability-go/prometheus"
)

// PrometheusConfig creates a complete Prometheus configuration for EKS monitoring.
var PrometheusConfig = prometheus.Config{
	Global: prometheus.GlobalConfig{
		ScrapeInterval:     15 * prometheus.Second,
		EvaluationInterval: 15 * prometheus.Second,
		ExternalLabels: map[string]string{
			"cluster": "eks-k8s-cluster",
			"region":  "us-east-1",
		},
	},
	// Use all EKS-specific scrape configs
	ScrapeConfigs: eks.AllScrapeConfigs(),
	// Use EKS remote write helper for AWS Managed Prometheus
	RemoteWrite: []*prometheus.RemoteWriteConfig{
		eks.AWSManagedPrometheusRemoteWrite("ws-abc123", "us-east-1"),
	},
}

// AlertingRules combines base K8s alerts with EKS-specific alerts.
var AlertingRules = eks.AllAlerts()

// RecordingRules provides pre-computed metrics for efficient querying.
var RecordingRules = eks.AllRecordingRules()

// ClusterDashboard provides a comprehensive EKS cluster overview.
var ClusterDashboard = eks.EKSClusterDashboard

// ALBDashboard provides AWS Application Load Balancer monitoring.
var ALBDashboard = eks.EKSALBDashboard

// NetworkingDashboard provides VPC CNI and IRSA monitoring.
var NetworkingDashboard = eks.EKSNetworkingDashboard

// The monitoring/eks package provides:
//
// Scrape Configs:
// - Base K8s scrapes (nodes, pods, services, kube-state-metrics, cadvisor)
// - AWS Managed Prometheus endpoint
// - CloudWatch Agent scrape
// - ALB Controller metrics
// - EBS CSI Driver metrics
// - VPC CNI plugin metrics
// - ADOT Collector metrics
// - CloudWatch Exporter metrics
//
// Alerts:
// - Base K8s alerts (node, pod, cluster health)
// - Node Group scaling alerts
// - ALB health and error rate alerts
// - IRSA credential errors
// - VPC CNI IP exhaustion
// - EBS volume attachment failures
// - API Server high latency
// - Fargate scheduling failures
//
// PromQL Expressions:
// - Node group utilization metrics
// - ALB request rates and latencies
// - Container Insights pod/node metrics
// - Control plane metrics (API server, etcd)
// - Fargate pod metrics
//
// Dashboards:
// - EKS Cluster Overview
// - ALB Performance
// - VPC CNI/IRSA Networking
// - Fargate Pods
// - Control Plane Health
*/
