// Package eks_golden includes sample K8s workloads that would be deployed to the cluster.
//
// Note: These are represented as CloudFormation custom resources or could be
// deployed separately using kubectl/Helm after cluster creation.
// This file demonstrates the K8s manifests that would run on the cluster.
package eks_golden

// SampleDeploymentYAML represents a sample Kubernetes Deployment manifest
// that would be deployed to the cluster. In practice, this would be deployed
// via kubectl, Helm, or GitOps tools like ArgoCD/Flux.
//
// This is included as documentation of what a multi-cloud portable workload looks like.
const SampleDeploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-app
  namespace: default
  labels:
    app: sample-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: sample-app
  template:
    metadata:
      labels:
        app: sample-app
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      nodeSelector:
        nodegroup-type: application
      tolerations:
        - key: "lifecycle"
          operator: "Equal"
          value: "spot"
          effect: "NoSchedule"
      containers:
        - name: app
          image: nginx:1.25
          ports:
            - containerPort: 80
              name: http
            - containerPort: 8080
              name: metrics
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "512Mi"
          readinessProbe:
            httpGet:
              path: /healthz
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: 80
            initialDelaySeconds: 15
            periodSeconds: 20
---
apiVersion: v1
kind: Service
metadata:
  name: sample-app
  namespace: default
  labels:
    app: sample-app
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
spec:
  type: ClusterIP
  selector:
    app: sample-app
  ports:
    - name: http
      port: 80
      targetPort: 80
    - name: metrics
      port: 8080
      targetPort: 8080
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: sample-app
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: sample-app
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
`

// SampleIngressYAML represents a sample ALB Ingress configuration
// that would be deployed to the cluster.
const SampleIngressYAML = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: sample-app
  namespace: default
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /healthz
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:us-east-1:ACCOUNT:certificate/CERT-ID
spec:
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: sample-app
                port:
                  number: 80
`
