---
title: "Quick Start"
---
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./wetwire-dark.svg">
  <img src="./wetwire-light.svg" width="100" height="67">
</picture>

Get started with `wetwire-aws` in 5 minutes.

## Installation

See [README.md](../README.md#installation) for installation instructions.

## Quick Test (No Setup Required)

You can test `wetwire-aws` without creating a Go module:

```bash
mkdir test && cd test
cat > main.go << 'EOF'
package infra

import "github.com/lex00/wetwire-aws-go/resources/s3"

var Bucket = s3.Bucket{BucketName: "my-bucket"}
EOF

wetwire-aws build .
```

This works because `wetwire-aws` auto-generates a synthetic module when no `go.mod` is found.

---

## Your First Project

For real projects, create a proper Go module:

```
myapp/
├── go.mod
└── infra/
    └── storage.go
```

**infra/storage.go:**
```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/s3"
)

// DataBucket defines an S3 bucket for data storage
var DataBucket = s3.Bucket{
    BucketName: "my-data-bucket",
}
```

**Generate template:**
```bash
wetwire-aws build ./infra > template.json
```

That's it. Resources are discovered via AST parsing when you run `build`.

---

## Adding References

Reference other resources using the `Ref` and `GetAtt` intrinsics:

**infra/storage.go:**
```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/s3"
    "github.com/lex00/wetwire-aws-go/resources/iam"
    "github.com/lex00/wetwire-aws-go/resources/lambda"
    . "github.com/lex00/wetwire-aws-go/intrinsics"
)

// DataBucket is an S3 bucket for data
var DataBucket = s3.Bucket{
    BucketName: "data",
}

// Flat policy statement - extracted from inline slice
var LambdaAssumeRoleStatement = PolicyStatement{
    Effect:    "Allow",
    Principal: ServicePrincipal{"lambda.amazonaws.com"},
    Action:    "sts:AssumeRole",
}

// Flat policy document - references the statement
var LambdaAssumeRolePolicy = PolicyDocument{
    Statement: []any{LambdaAssumeRoleStatement},
}

// ProcessorRole is the IAM role for the Lambda function
var ProcessorRole = iam.Role{
    RoleName:                 "processor",
    AssumeRolePolicyDocument: LambdaAssumeRolePolicy,
}

// Flat environment - extracted from inline
var ProcessorEnv = lambda.Environment{
    Variables: Json{
        "BUCKET_NAME": DataBucket,
    },
}

// ProcessorFunction processes data from the bucket
var ProcessorFunction = lambda.Function{
    FunctionName: "processor",
    Runtime:      lambda.RuntimePython312,
    Handler:      "index.handler",
    Role:         ProcessorRole.Arn,  // GetAtt via field access
    Environment:  ProcessorEnv,
}
```

---

## Using the CLI

```bash
# Generate template from a directory
wetwire-aws build ./infra > template.json

# Generate YAML
wetwire-aws build ./infra --format yaml

# Initialize a new project
wetwire-aws init myapp

# Lint code for issues
wetwire-aws lint ./infra
```

---

## Multi-File Organization

Split resources across files:

```
myapp/
├── go.mod
└── infra/
    ├── storage.go    # S3, EFS
    ├── compute.go    # Lambda, EC2
    ├── network.go    # VPC, Subnets
    └── database.go   # DynamoDB, RDS
```

**storage.go:**
```go
package infra

import "github.com/lex00/wetwire-aws-go/resources/s3"

var DataBucket = s3.Bucket{
    BucketName: "data",
}
```

**compute.go:**
```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/lambda"
    . "github.com/lex00/wetwire-aws-go/intrinsics"
)

// Flat environment variable
var ProcessorEnv = lambda.Environment{
    Variables: Json{
        // Cross-file reference - DataBucket is discovered from storage.go
        "BUCKET_NAME": DataBucket,
    },
}

var ProcessorFunction = lambda.Function{
    FunctionName: "processor",
    Runtime:      lambda.RuntimePython312,
    Handler:      "index.handler",
    Environment:  ProcessorEnv,
}
```

**Generate:**
```bash
wetwire-aws build ./infra
```

---

## Type-Safe Constants

Use generated enum constants for type safety:

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/lambda"
    "github.com/lex00/wetwire-aws-go/resources/dynamodb"
)

var MyFunction = lambda.Function{
    Runtime:       lambda.RuntimePython312,    // Not "python3.12"
    Architectures: []any{lambda.ArchitectureArm64},
}

// Flat key schema - extracted from inline slice
var MyTablePK = dynamodb.KeySchema{
    AttributeName: "pk",
    KeyType:       dynamodb.KeyTypeHash,
}

// Flat attribute definition - extracted from inline slice
var MyTablePKAttr = dynamodb.AttributeDefinition{
    AttributeName: "pk",
    AttributeType: dynamodb.ScalarAttributeTypeS,
}

var MyTable = dynamodb.Table{
    KeySchema:            []any{MyTablePK},
    AttributeDefinitions: []any{MyTablePKAttr},
}
```

---

## Deploy

```bash
# Generate template
wetwire-aws build ./infra > template.json

# Deploy with AWS CLI
aws cloudformation deploy \
  --template-file template.json \
  --stack-name myapp \
  --capabilities CAPABILITY_IAM
```

---

## Serverless (SAM) Resources

For serverless applications, use the `serverless` package:

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/serverless"
    "github.com/lex00/wetwire-aws-go/resources/dynamodb"
)

// SAM Function - simplified Lambda
var HelloFunction = serverless.Function{
    Handler:    "bootstrap",
    Runtime:    "provided.al2",
    CodeUri:    "./hello/",
    MemorySize: 128,
    Timeout:    30,
}

// Environment variables as flat variable
var ProcessorEnv = serverless.Function_Environment{
    Variables: map[string]any{
        "TABLE_NAME": DataTable.TableName,
    },
}

var ProcessorFunction = serverless.Function{
    Handler:     "bootstrap",
    Runtime:     "provided.al2",
    CodeUri:     "./processor/",
    Environment: &ProcessorEnv,
}

// Standard DynamoDB table (can mix SAM and CloudFormation)
var DataTable = dynamodb.Table{
    TableName: "my-data-table",
}
```

SAM templates automatically include the `Transform: AWS::Serverless-2016-10-31` header.

See the full [SAM Guide](SAM.md) for all 9 SAM resource types.

---

## AI-Assisted Design

Let AI help create your AWS infrastructure:

```bash
# No API key required - uses Claude CLI
wetwire-aws design "Create an S3 bucket with versioning and a Lambda function to process uploads"
```

The design command creates Go code following wetwire patterns, runs linting, and builds the final CloudFormation template.

## Next Steps

- See the full [CLI Reference](CLI.md)
- Read the [SAM Guide](SAM.md) for serverless applications
- Explore the generated resource types
