# Quick Start

Get started with `wetwire-aws` in 5 minutes.

## Installation

```bash
go install github.com/lex00/wetwire/go/wetwire-aws/cmd/wetwire-aws@latest
```

Or add to your project:

```bash
go get github.com/lex00/wetwire/go/wetwire-aws
```

## Your First Project

Create a package for your infrastructure:

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
    "github.com/lex00/wetwire/go/wetwire-aws/resources/s3"
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
    "github.com/lex00/wetwire/go/wetwire-aws/resources/s3"
    "github.com/lex00/wetwire/go/wetwire-aws/resources/iam"
    "github.com/lex00/wetwire/go/wetwire-aws/resources/lambda"
    . "github.com/lex00/wetwire/go/wetwire-aws/intrinsics"
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
        "BUCKET_NAME": Ref{"DataBucket"},
    },
}

// ProcessorFunction processes data from the bucket
var ProcessorFunction = lambda.Function{
    FunctionName: "processor",
    Runtime:      lambda.RuntimePython312,
    Handler:      "index.handler",
    Role:         GetAtt{"ProcessorRole", "Arn"},
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
wetwire-aws init -o myapp/

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

import "github.com/lex00/wetwire/go/wetwire-aws/resources/s3"

var DataBucket = s3.Bucket{
    BucketName: "data",
}
```

**compute.go:**
```go
package infra

import (
    "github.com/lex00/wetwire/go/wetwire-aws/resources/lambda"
    . "github.com/lex00/wetwire/go/wetwire-aws/intrinsics"
)

// Flat environment variable
var ProcessorEnv = lambda.Environment{
    Variables: Json{
        // Cross-file reference - DataBucket is discovered from storage.go
        "BUCKET_NAME": Ref{"DataBucket"},
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
    "github.com/lex00/wetwire/go/wetwire-aws/resources/lambda"
    "github.com/lex00/wetwire/go/wetwire-aws/resources/dynamodb"
)

var MyFunction = lambda.Function{
    Runtime:       lambda.RuntimePython312,    // Not "python3.12"
    Architectures: []string{lambda.ArchitectureArm64},
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
    KeySchema:            []dynamodb.KeySchema{MyTablePK},
    AttributeDefinitions: []dynamodb.AttributeDefinition{MyTablePKAttr},
}
```

---

## Template Building (Programmatic)

For programmatic template building, use the `template` package:

```go
package main

import (
    "fmt"
    "github.com/lex00/wetwire/go/wetwire-aws/internal/template"
    "github.com/lex00/wetwire/go/wetwire-aws/resources/s3"
    . "github.com/lex00/wetwire/go/wetwire-aws/intrinsics"
)

func main() {
    t := template.New()
    t.Description = "My Application Stack"

    // Add resources
    t.AddResource("DataBucket", s3.Bucket{
        BucketName: "my-data",
    })

    // Add parameters
    t.AddParameter("Environment", template.Parameter{
        Type:          "String",
        Default:       "dev",
        AllowedValues: []string{"dev", "staging", "prod"},
    })

    // Add outputs - use typed intrinsics, not raw maps
    t.AddOutput("BucketArn", template.Output{
        Value:       GetAtt{"DataBucket", "Arn"},
        Description: "Data bucket ARN",
    })

    json, _ := t.ToJSON()
    fmt.Println(string(json))
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

## Next Steps

- See the full [CLI Reference](CLI.md)
- Learn about [Intrinsic Functions](INTRINSICS.md)
- Explore the generated resource types
