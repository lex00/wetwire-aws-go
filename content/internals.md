---
title: "Internals"
---
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./wetwire-dark.svg">
  <img src="./wetwire-light.svg" width="100" height="67">
</picture>

This document covers the internal architecture of wetwire-aws-go for contributors and maintainers.

**Contents:**
- [AST Discovery](#ast-discovery) - How resource discovery works
- [Template Generation](#template-generation) - How templates are built
- [Reference Resolution](#reference-resolution) - Ref and GetAtt handling
- [Value Runner](#value-runner) - How values are extracted
- [Importer](#importer) - CloudFormation to Go conversion
- [Linter Architecture](#linter-architecture) - How lint rules work

---

## AST Discovery

wetwire-aws uses Go's `go/ast` package to discover CloudFormation resource declarations without executing user code.

### How It Works

When you define a resource as a package-level variable:

```go
var MyBucket = s3.Bucket{
    BucketName: "my-bucket",
}
```

The discovery phase:
1. Parses Go source files using `go/parser`
2. Walks the AST looking for `var` declarations
3. Identifies composite literals with types from resource packages
4. Extracts metadata: name, type, file, line, dependencies

### Discovery API

```go
import "github.com/lex00/wetwire-aws-go/internal/discover"

opts := discover.Options{
    Packages: []string{"./infra/..."},
    Verbose:  false,
}

result, err := discover.Discover(opts)

// Access discovered resources
for name, res := range result.Resources {
    fmt.Printf("%s: %s at %s:%d\n", name, res.Type, res.File, res.Line)
}

// Access parameters, outputs, mappings, conditions
for name := range result.Parameters { ... }
for name := range result.Outputs { ... }
```

### What Gets Discovered

| Type | Example | Discovered As |
|------|---------|---------------|
| Resource | `var MyBucket = s3.Bucket{...}` | Resource |
| Parameter | `var Env = Parameter{...}` | Parameter |
| Output | `var BucketArn = Output{...}` | Output |
| Mapping | `var RegionAMI = Mapping{...}` | Mapping |
| Condition | `var IsProd = Equals{...}` | Condition |

### Dependency Extraction

The discovery phase also extracts dependencies by analyzing field values:

```go
var MyFunction = lambda.Function{
    Role: MyRole.Arn,      // Dependency on MyRole (GetAtt)
    Environment: EnvVars,  // Dependency on EnvVars variable
}
```

Dependencies are tracked for:
- Direct variable references (`MyRole`)
- Attribute access (`MyRole.Arn`)
- Nested composite literals

---

## Template Generation

The `template.Builder` constructs CloudFormation templates from discovered resources.

### Build Process

```go
import "github.com/lex00/wetwire-aws-go/internal/template"

// Create builder from discovered resources
builder := template.NewBuilderFull(
    result.Resources,
    result.Parameters,
    result.Outputs,
    result.Mappings,
    result.Conditions,
)

// Set actual values (from runner)
for name, value := range values {
    builder.SetValue(name, value)
}

// Build template
tmpl, err := builder.Build()
```

### Topological Sorting

Resources are ordered so dependencies come before dependents using Kahn's algorithm:

```go
// Given:
// - VPC (no deps)
// - Subnet (depends on VPC)
// - Instance (depends on Subnet)

// After topological sort:
// 1. VPC
// 2. Subnet
// 3. Instance
```

The `topologicalSort()` method:
1. Finds resources with no unsatisfied dependencies
2. Adds them to the result
3. Repeats until all resources are placed
4. Detects circular dependencies and reports them

---

## Value Runner

To get actual property values, the runner compiles and executes user code. This is necessary because AST-based discovery captures structure but not runtime values.

### How It Works

1. **Generate temporary Go program** - Creates `main.go` that imports user's package
2. **Run `go mod tidy`** - Resolves dependencies
3. **Execute and capture JSON** - Runs program, parses JSON output

```go
import "github.com/lex00/wetwire-aws-go/internal/runner"

// Extract values for all discovered components
result, err := runner.ExtractAll(
    pkgPath,
    resources,   // map[string]DiscoveredResource
    parameters,  // map[string]DiscoveredParameter
    outputs,     // map[string]DiscoveredOutput
    mappings,    // map[string]DiscoveredMapping
    conditions,  // map[string]DiscoveredCondition
)

// Result contains organized values
for name, props := range result.Resources {
    fmt.Printf("%s: %v\n", name, props["BucketName"])
}
```

### Vendor Mode

When a `vendor/` directory exists, the runner uses in-module execution for offline builds:

```go
// With vendor directory:
// - Creates _wetwire_runner/ subdir in module
// - Uses -mod=vendor flag
// - No network access needed

// Without vendor directory:
// - Creates temp directory
// - Runs go mod tidy to fetch deps
// - Uses replace directive for local module
```

### Generated Runner Template

The runner generates code like this:

```go
package main

import (
    "encoding/json"
    "fmt"
    pkg "user/module/path"
)

func main() {
    result := make(map[string]map[string]any)

    // For each discovered variable
    value := serializeValue(pkg.MyBucket)
    result["MyBucket"] = value

    output, _ := json.Marshal(result)
    fmt.Println(string(output))
}
```

---

## Importer

The importer converts existing CloudFormation YAML/JSON to Go code.

### Import Process

```go
import "github.com/lex00/wetwire-aws-go/internal/importer"

// Parse CloudFormation template
opts := importer.Options{
    PackageName: "infra",
    UsePointers: false,
    WithComments: true,
}

generated, err := importer.Import("template.yaml", opts)
```

### Code Generation

The generator creates idiomatic Go code:

```yaml
# Input: CloudFormation YAML
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub "${AWS::StackName}-data"
      Tags:
        - Key: Environment
          Value: !Ref Environment
```

```go
// Output: Go code
package infra

import (
    . "github.com/lex00/wetwire-aws-go/intrinsics"
    "github.com/lex00/wetwire-aws-go/resources/s3"
)

var MyBucket = s3.Bucket{
    BucketName: Sub{"${AWS::StackName}-data"},
    Tags: []Tag{
        {Key: "Environment", Value: Environment},
    },
}
```

### Intrinsic Function Mapping

| CloudFormation | Generated Go |
|----------------|--------------|
| `!Ref X` | `X` (direct reference) |
| `!GetAtt X.Attr` | `X.Attr` (field access) |
| `!Sub "..."` | `Sub{"..."}` |
| `!Join [",", [...]]` | `Join{",", []any{...}}` |
| `!If [cond, a, b]` | `If{"cond", a, b}` |
| `Fn::Equals` | `Equals{a, b}` |

---

## Reference Resolution

### Ref Resolution

Direct variable references become CloudFormation `Ref`:

```go
// Source
var MySecurityGroup = ec2.SecurityGroup{
    VpcId: MyVPC,  // Reference to another resource
}

// Generated CloudFormation
{
  "Type": "AWS::EC2::SecurityGroup",
  "Properties": {
    "VpcId": {"Ref": "MyVPC"}
  }
}
```

### GetAtt Resolution

Attribute access becomes CloudFormation `Fn::GetAtt`:

```go
// Source
var MyFunction = lambda.Function{
    Role: MyRole.Arn,  // Attribute access
}

// Generated CloudFormation
{
  "Type": "AWS::Lambda::Function",
  "Properties": {
    "Role": {"Fn::GetAtt": ["MyRole", "Arn"]}
  }
}
```

### AttrRef Tracking

The discovery phase tracks `AttrRefUsage` for each field access:

```go
type AttrRefUsage struct {
    ResourceName string  // "MyRole"
    Attribute    string  // "Arn"
    FieldPath    string  // "Role"
}
```

During serialization, empty GetAtt references are fixed using this tracked information.

### Recursive Resolution

For nested structures, AttrRefs are resolved recursively:

```go
var EnvVars = lambda.Function_Environment{
    Variables: Json{
        "ROLE_ARN": MyRole.Arn,  // AttrRef tracked here
    },
}

var MyFunction = lambda.Function{
    Environment: EnvVars,  // Variable reference
}
```

The builder follows variable references and collects all AttrRefs.

---

## Linter Architecture

The linter checks Go source for style issues and potential problems.

### Rule Structure

Each rule has:
- **ID**: `WAW001` through `WAW018`
- **Severity**: error, warning, or info
- **Check function**: Analyzes AST nodes
- **Fix function** (optional): Generates code fixes

```go
type Rule struct {
    ID       string
    Name     string
    Severity string
    Check    func(*ast.File, *token.FileSet) []Issue
    Fix      func(*Issue, []byte) []byte
}
```

### Current Rules

| ID | Description |
|----|-------------|
| WAW001 | Use pseudo-parameter constants (`AWS_REGION` not `"AWS::Region"`) |
| WAW002 | Use intrinsic types instead of `map[string]any` |
| WAW003 | Detect duplicate resource variable names |
| WAW004 | Flag files with too many resources (>20) |
| WAW005 | Extract inline property types to named variables |
| WAW006 | Use typed policy document structs |
| WAW007 | Use typed slices instead of `[]any` |
| WAW008 | Use named var declarations |
| WAW009 | Use typed structs instead of `map[string]any` |
| WAW010 | Flatten inline typed struct literals |
| WAW011 | Validate enum property values |
| WAW012 | Use typed enum constants |
| WAW013 | Undefined reference |
| WAW014 | Unused intrinsics import |
| WAW015 | Avoid explicit `Ref{}` — use direct references |
| WAW016 | Avoid explicit `GetAtt{}` — use `.Attr` field access |
| WAW017 | Avoid pointer assignments |
| WAW018 | Use `Json{}` instead of `map[string]any{}` |

### Running the Linter

```go
import "github.com/lex00/wetwire-aws-go/internal/lint"

issues, err := lint.Lint(packages, lint.Options{
    Fix: false,
})

for _, issue := range issues {
    fmt.Printf("%s:%d: [%s] %s\n",
        issue.File, issue.Line, issue.Rule, issue.Message)
}
```

### Auto-Fix

Many rules support automatic fixing:

```bash
wetwire-aws lint --fix ./infra/...
```

The fix function receives the original source and returns modified source.

---

## Files Reference

| File | Purpose |
|------|---------|
| `contracts.go` | Core types (Resource, AttrRef, Template, etc.) |
| `internal/discover/discover.go` | AST-based resource discovery |
| `internal/template/template.go` | Template builder with topo sort |
| `internal/runner/runner.go` | Value extraction via compilation |
| `internal/lint/rules.go` | Lint rules WAW001-WAW010 |
| `internal/lint/rules_extra.go` | Lint rules WAW011-WAW018 |
| `internal/importer/parser.go` | CloudFormation YAML/JSON parser |
| `internal/importer/codegen.go` | Go code generator |
| `intrinsics/intrinsics.go` | Intrinsic function types |
| `intrinsics/pseudo.go` | AWS pseudo-parameters |

---

## See Also

- [Developer Guide](DEVELOPERS.md) - Development workflow
- [Code Generation](CODEGEN.md) - Resource type generation
- [CLI Reference](CLI.md) - CLI commands
