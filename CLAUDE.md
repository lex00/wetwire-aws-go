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
├── resources/         # Generated resource types (263 services)
│   ├── s3/           # S3 resources (Bucket, AccessPoint, etc.)
│   ├── lambda/       # Lambda resources (Function, etc.)
│   ├── iam/          # IAM resources (Role, Policy, etc.)
│   └── serverless/   # SAM resources (Function, Api, etc.)
├── intrinsics/       # Ref, GetAtt, Sub, Join, etc.
├── internal/
│   ├── discover/     # AST-based resource discovery
│   ├── template/     # CF template builder with topo sort
│   ├── linter/       # 16 lint rules (WAW001-WAW018)
│   └── importer/     # YAML to Go conversion
└── cmd/wetwire-aws/  # CLI application
```

## Lint Rules (WAW001-WAW018)

Key rules enforcing declarative patterns:

- **WAW001**: Use pseudo-parameter constants (`AWS_REGION` not `"AWS::Region"`)
- **WAW002**: Use intrinsic types (`Ref{}` not `map[string]any`)
- **WAW005**: Extract inline property types to separate vars
- **WAW015-16**: Avoid explicit `Ref{}` and `GetAtt{}` — use direct references
- **WAW017**: Avoid pointer assignments
- **WAW018**: Use `Json{}` type instead of `map[string]any{}`

## Key Principles

1. **Flat variables** — Extract all nested structs into named variables
2. **No pointers** — Never use `&` or `*` in declarations
3. **Direct references** — Variables reference each other by name
4. **Struct literals only** — No function calls in declarations

## Build

```bash
wetwire-aws build ./infra > template.json
```

## Project Structure

```
my-stack/
├── go.mod
├── network.go         # VPC, subnets, security groups
├── compute.go         # EC2, Lambda, containers
├── storage.go         # S3, EFS, databases
└── security.go        # IAM roles, policies
```
