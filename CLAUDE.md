# wetwire-aws (Go)

Generate CloudFormation templates from Go resource declarations.

## Syntax Principles

All resources are Go struct literals. No function calls, no pointers, no registration.

### Resource Declaration

Resources are declared as package-level variables:

```go
var DataBucket = s3.Bucket{
    BucketName: "my-data-bucket",
}
```

### Direct References

Reference other resources directly by variable name:

```go
var ProcessorFunction = lambda.Function{
    Role: ProcessorRole.Arn,  // GetAtt via field access
    Environment: ProcessorEnv,
}
```

### Nested Types

Extract nested configurations to separate variables:

```go
var ProcessorEnv = lambda.Environment{
    Variables: Json{
        "BUCKET": DataBucket,  // Direct reference
    },
}
```

### Dot-Import for Intrinsics

The `intrinsics` package is dot-imported for clean syntax:

```go
import (
    . "github.com/lex00/wetwire-aws-go/intrinsics"
    "github.com/lex00/wetwire-aws-go/resources/s3"
)

var MyBucket = s3.Bucket{
    BucketName: Sub("${AWS::StackName}-data"),  // intrinsic via dot-import
}
```

**intrinsics provides:**
- `Json` — Type alias for `map[string]any`
- `Ref`, `GetAtt`, `Sub`, `Join`, `If`, `Equals` — CF intrinsics
- `AWS_REGION`, `AWS_STACK_NAME`, `AWS_ACCOUNT_ID` — Pseudo-parameters

## Package Structure

```
wetwire-aws-go/
├── resources/         # Generated resource types (264 services)
│   ├── s3/           # S3 resources (Bucket, AccessPoint, etc.)
│   ├── lambda/       # Lambda resources (Function, etc.)
│   ├── iam/          # IAM resources (Role, Policy, etc.)
│   ├── serverless/   # SAM resources (Function, Api, etc.)
│   └── k8s/          # K8s CRD types (via ACK) for kubectl deployments
│       ├── ec2/      # VPC, Subnet, SecurityGroup CRDs
│       ├── eks/      # Cluster, NodeGroup, Addon CRDs
│       └── iam/      # Role, Policy CRDs
├── intrinsics/       # Ref, GetAtt, Sub, Join, etc.
├── domain/           # Domain interface implementation
│   ├── domain.go     # Domain interface and CreateRootCommand
│   └── aws_domain.go # AWS implementation (AwsDomain)
├── internal/
│   ├── discover/     # AST-based resource discovery
│   ├── template/     # CF template builder with topo sort
│   ├── linter/       # 18 lint rules (WAW001-WAW018)
│   ├── importer/     # YAML to Go conversion
│   └── kiro/         # Kiro CLI integration
└── cmd/wetwire-aws/  # CLI application
    ├── main.go       # CLI entry point using domain.CreateRootCommand
    ├── mcp.go        # MCP server for Claude integration
    ├── design.go     # AI-assisted design command
    ├── test.go       # Persona-based testing command
    ├── optimize.go   # Optimization suggestions command
    ├── diff.go       # Template diff command
    └── watch.go      # File watching command
```

## Architecture

wetwire-aws implements the `domain.Domain` interface from wetwire-core-go, enabling automatic CLI command generation and MCP tool registration.

### Domain Pattern

The `domain` package provides:
- **Domain interface**: Core methods (Builder, Linter, Initializer, Validator)
- **Optional interfaces**: OptionalImporter, OptionalLister, OptionalGrapher
- **CreateRootCommand**: Generates CLI with all standard commands
- **Run**: Creates and executes CLI (for simple use cases)

### AWS-Specific Commands

Beyond the domain interface, wetwire-aws adds AWS-specific commands:
- **design**: AI-assisted infrastructure design (Anthropic/Kiro)
- **test**: Automated persona-based testing
- **optimize**: CloudFormation optimization suggestions
- **diff**: Semantic template comparison
- **watch**: Auto-rebuild on file changes
- **mcp**: MCP server for Claude Code integration

## Lint Rules

Uses the `WAW` prefix (Wetwire AWS). See [LINT_RULES.md](docs/LINT_RULES.md) for the complete rule reference (WAW001-WAW019).

## Key Principles

1. **Flat variables** — Extract all nested structs into named variables
2. **No pointers** — Never use `&` or `*` in declarations
3. **Direct references** — Variables reference each other by name
4. **Struct literals only** — No function calls in declarations

## Build

```bash
wetwire-aws build ./infra > template.json
```

## K8s-Native Deployments (ACK)

The `resources/k8s/` directory contains AWS Controllers for Kubernetes (ACK) types for deploying AWS resources via `kubectl apply`. This provides a Kubernetes-native alternative to CloudFormation.

### Using ACK Types

```go
import (
    eksv1 "github.com/lex00/wetwire-aws-go/resources/k8s/eks/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var Cluster = eksv1.Cluster{
    TypeMeta: metav1.TypeMeta{
        APIVersion: "eks.services.k8s.aws/v1alpha1",
        Kind:       "Cluster",
    },
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-cluster",
        Namespace: "ack-system",
    },
    Spec: eksv1.ClusterSpec{
        Name:    "my-cluster",
        Version: "1.28",
    },
}
```

### When to Use ACK vs CloudFormation

| Approach | Use When |
|----------|----------|
| **CloudFormation** (`resources/eks/`) | Traditional IaC, AWS-native tooling, existing CF pipelines |
| **ACK** (`resources/k8s/eks/`) | GitOps workflows, Kubernetes-centric teams, unified K8s API |

See `examples/eks-golden/` for CloudFormation approach and `examples/eks-k8s/` for ACK approach.

## Project Structure

```
my-stack/
├── go.mod
├── network.go         # VPC, subnets, security groups
├── compute.go         # EC2, Lambda, containers
├── storage.go         # S3, EFS, databases
└── security.go        # IAM roles, policies
```
