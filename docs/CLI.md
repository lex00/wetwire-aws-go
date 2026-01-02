# CLI Reference

The `wetwire-aws` command provides tools for generating and validating CloudFormation templates from Go code.

## Quick Reference

| Command | Description |
|---------|-------------|
| `wetwire-aws build` | Generate CloudFormation template from Go source |
| `wetwire-aws lint` | Lint code for issues |
| `wetwire-aws init` | Initialize a new project |

```bash
wetwire-aws --help     # Show help
```

---

## build

Generate CloudFormation template from Go source files.

```bash
# Generate JSON to stdout
wetwire-aws build ./infra > template.json

# Generate YAML format
wetwire-aws build ./infra --format yaml > template.yaml

# With description
wetwire-aws build ./infra --description "My Application Stack"
```

### Options

| Option | Description |
|--------|-------------|
| `PATH` | Directory containing Go source files |
| `--format, -f {json,yaml}` | Output format (default: json) |
| `--description, -d TEXT` | Template description |

### How It Works

1. Parses Go source files using `go/ast`
2. Discovers `var X = Type{...}` resource declarations
3. Extracts resource dependencies from intrinsic references
4. Orders resources topologically by dependencies
5. Generates CloudFormation JSON or YAML

### Output Modes

**JSON (default):**
```json
{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "DataBucket": {
      "Type": "AWS::S3::Bucket",
      "Properties": { "BucketName": "my-data" }
    }
  }
}
```

**YAML:**
```yaml
AWSTemplateFormatVersion: '2010-09-09'
Resources:
  DataBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-data
```

---

## lint

Lint wetwire-aws code for issues.

```bash
# Lint a directory
wetwire-aws lint ./infra

# Lint a single file
wetwire-aws lint ./infra/storage.go
```

### Options

| Option | Description |
|--------|-------------|
| `PATH` | File or directory to lint |

### What It Checks

1. **Resource discovery**: Validates resources can be parsed from source
2. **Reference validity**: Checks that referenced resources exist
3. **Type correctness**: Validates resource types are valid CloudFormation types

### Output Examples

**Linting passed:**
```
Linting passed: 5 resources OK
```

**Issues found:**
```
./infra/storage.go:15: undefined resource reference: MissingBucket
./infra/compute.go:23: unknown resource type: AWS::Invalid::Type
```

---

## init

Initialize a new wetwire-aws project.

```bash
# Create a new project
wetwire-aws init -o myapp/
```

### Options

| Option | Description |
|--------|-------------|
| `-o, --output DIR` | Output directory (required) |

### Generated Structure

```
myapp/
├── go.mod
├── main.go
└── infra/
    └── storage.go
```

**main.go:**
```go
package main

import (
    "fmt"
    "myapp/infra"

    "github.com/lex00/wetwire/go/wetwire-aws/internal/template"
)

func main() {
    t := template.New()
    t.Description = "My Application"
    // Add resources from infra package
    fmt.Println(t.ToJSON())
}
```

**infra/storage.go:**
```go
package infra

import "github.com/lex00/wetwire/go/wetwire-aws/resources/s3"

var DataBucket = s3.Bucket{
    BucketName: "my-data-bucket",
}
```

---

## Typical Workflow

### Development

```bash
# Lint before generating
wetwire-aws lint ./infra

# Generate template
wetwire-aws build ./infra > template.json

# Preview YAML format
wetwire-aws build ./infra --format yaml
```

### CI/CD

```bash
#!/bin/bash
# ci.sh

# Lint first
wetwire-aws lint ./infra || exit 1

# Generate template
wetwire-aws build ./infra > template.json

# Deploy with AWS CLI
aws cloudformation deploy \
  --template-file template.json \
  --stack-name myapp \
  --capabilities CAPABILITY_IAM
```

---

## Intrinsic Functions

All CloudFormation intrinsic functions are supported:

| Function | Go API |
|----------|--------|
| Ref | `Ref{"MyResource"}` |
| GetAtt | `GetAtt{"MyResource", "Arn"}` |
| Sub | `Sub{String: "${AWS::StackName}-bucket"}` |
| SubWithMap | `SubWithMap{String: "...", Variables: Json{...}}` |
| Join | `Join{Delimiter: ",", Values: []any{"a", "b"}}` |
| If | `If{Condition: "IsProd", IfTrue: val1, IfFalse: val2}` |
| Equals | `Equals{Left: value1, Right: value2}` |
| And/Or/Not | `And{Conditions: []any{...}}`, `Or{...}`, `Not{...}` |
| FindInMap | `FindInMap{MapName: "...", TopKey: "...", SecondKey: "..."}` |
| Select | `Select{Index: 0, List: GetAZs{}}` |
| Split | `Split{Delimiter: ",", String: "a,b,c"}` |
| Base64 | `Base64{Value: "Hello"}` |
| Cidr | `Cidr{IpBlock: "10.0.0.0/16", Count: 256, CidrBits: 8}` |
| GetAZs | `GetAZs{Region: "us-east-1"}` or `GetAZs{}` |
| ImportValue | `ImportValue{Name: "ExportedValue"}` |

**Note:** Use dot import for cleaner syntax: `import . "github.com/lex00/wetwire/go/wetwire-aws/intrinsics"`

---

## Pseudo-Parameters

Built-in CloudFormation pseudo-parameters:

```go
import "github.com/lex00/wetwire/go/wetwire-aws/intrinsics"

// Available pseudo-parameters
intrinsics.AWS_REGION        // {"Ref": "AWS::Region"}
intrinsics.AWS_ACCOUNT_ID    // {"Ref": "AWS::AccountId"}
intrinsics.AWS_STACK_NAME    // {"Ref": "AWS::StackName"}
intrinsics.AWS_STACK_ID      // {"Ref": "AWS::StackId"}
intrinsics.AWS_PARTITION     // {"Ref": "AWS::Partition"}
intrinsics.AWS_URL_SUFFIX    // {"Ref": "AWS::URLSuffix"}
intrinsics.AWS_NO_VALUE      // {"Ref": "AWS::NoValue"}
```

Usage:
```go
import . "github.com/lex00/wetwire/go/wetwire-aws/intrinsics"

var MyBucket = s3.Bucket{
    BucketName: Sub{String: "${AWS::StackName}-data"},
}
```

---

## See Also

- [Quick Start](QUICK_START.md) - Create your first project
- [Intrinsic Functions](INTRINSICS.md) - Full intrinsics reference
