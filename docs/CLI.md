# CLI Reference

The `wetwire-aws` command provides tools for generating and validating CloudFormation templates from Go code.

## Quick Reference

| Command | Description |
|---------|-------------|
| `wetwire-aws build` | Generate CloudFormation template from Go source |
| `wetwire-aws lint` | Lint code for issues |
| `wetwire-aws init` | Initialize a new project |
| `wetwire-aws import` | Import CloudFormation template to Go code |
| `wetwire-aws design` | AI-assisted infrastructure design |
| `wetwire-aws test` | Run automated persona-based testing |
| `wetwire-aws validate` | Validate resources and references |
| `wetwire-aws list` | List discovered resources |

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

    "github.com/lex00/wetwire-aws-go/internal/template"
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

import "github.com/lex00/wetwire-aws-go/resources/s3"

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

All CloudFormation intrinsic functions are supported. **Prefer direct references over explicit intrinsics:**

| Function | Preferred Style | Alternative |
|----------|-----------------|-------------|
| Ref (resource) | `MyBucket` | Direct variable reference |
| Ref (parameter) | `Param("VpcId")` | Helper function |
| GetAtt | `MyBucket.Arn` | Field access on resource |
| Sub | `Sub{String: "${AWS::StackName}-bucket"}` | - |
| SubWithMap | `SubWithMap{String: "...", Variables: Json{...}}` | - |
| Join | `Join{Delimiter: ",", Values: []any{"a", "b"}}` | - |
| If | `If{Condition: "IsProd", IfTrue: val1, IfFalse: val2}` | - |
| Equals | `Equals{Value1: x, Value2: y}` | - |
| And/Or/Not | `And{Conditions: []any{...}}` | - |
| FindInMap | `FindInMap{MapName: "...", TopKey: "...", SecondKey: "..."}` | - |
| Select | `Select{Index: 0, List: GetAZs{}}` | - |
| Split | `Split{Delimiter: ",", Source: "a,b,c"}` | - |
| Base64 | `Base64{Value: "Hello"}` | - |
| Cidr | `Cidr{IPBlock: "10.0.0.0/16", Count: 256, CidrBits: 8}` | - |
| GetAZs | `GetAZs{}` or `GetAZs{Region: "us-east-1"}` | - |
| ImportValue | `ImportValue{ExportName: "Value"}` | - |

**Note:** Use dot import for cleaner syntax: `import . "github.com/lex00/wetwire-aws-go/intrinsics"`

---

## Pseudo-Parameters

Built-in CloudFormation pseudo-parameters:

```go
import "github.com/lex00/wetwire-aws-go/intrinsics"

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
import . "github.com/lex00/wetwire-aws-go/intrinsics"

var MyBucket = s3.Bucket{
    BucketName: Sub{String: "${AWS::StackName}-data"},
}
```

---

## design

AI-assisted infrastructure design. Starts an interactive session to generate infrastructure code.

```bash
# Start design session with a prompt
wetwire-aws design "Create a serverless API with Lambda and API Gateway"

# Specify output directory
wetwire-aws design -o ./myproject "Create an S3 bucket with encryption"
```

### Options

| Option | Description |
|--------|-------------|
| `prompt` | Natural language description of infrastructure |
| `-o, --output` | Output directory (default: current dir) |
| `-l, --max-lint-cycles` | Maximum lint/fix cycles (default: 3) |
| `-s, --stream` | Stream AI responses (default: true) |

### Workflow

1. AI asks clarifying questions about requirements
2. Generates Go code using wetwire-aws patterns
3. Runs linter and auto-fixes issues
4. Builds CloudFormation template
5. Validates with cfn-lint-go

---

## test

Run automated persona-based testing to evaluate code generation quality.

```bash
# Run with default persona
wetwire-aws test "Create an S3 bucket with versioning"

# Use a specific persona
wetwire-aws test --persona beginner "Create a Lambda function"

# Track test scenario
wetwire-aws test --scenario "s3-encryption" "Create encrypted bucket"
```

### Personas

| Persona | Description |
|---------|-------------|
| `beginner` | New to AWS, asks many clarifying questions |
| `intermediate` | Familiar with AWS basics (default) |
| `expert` | Deep AWS knowledge, asks advanced questions |
| `terse` | Gives minimal responses |
| `verbose` | Provides detailed context |

### Options

| Option | Description |
|--------|-------------|
| `prompt` | Infrastructure description to test |
| `-p, --persona` | Persona to use (default: intermediate) |
| `-S, --scenario` | Scenario name for tracking |
| `-o, --output` | Output directory |
| `-l, --max-lint-cycles` | Maximum lint/fix cycles (default: 3) |

---

## validate

Validate resources and check dependencies.

```bash
wetwire-aws validate ./infra/...
wetwire-aws validate ./infra/... --format json
```

### Checks Performed

- **Reference validity**: All resource references point to defined resources
- **Dependency graph**: Validates resource dependencies exist

---

## list

List discovered resources in a package.

```bash
wetwire-aws list ./infra/...
```

---

## CloudFormation Validation

The `design` and `test` commands automatically validate generated templates using **cfn-lint-go**, which checks for:

- Valid resource types and properties
- Correct intrinsic function usage
- Best practices and security recommendations
- AWS CloudFormation specification compliance

---

## See Also

- [Quick Start](QUICK_START.md) - Create your first project
- [Intrinsic Functions](INTRINSICS.md) - Full intrinsics reference
