---
title: "FAQ"
---

This FAQ covers questions specific to the Go implementation of wetwire for AWS CloudFormation. For general wetwire questions, see the [central FAQ](https://github.com/lex00/wetwire/blob/main/docs/FAQ.md).

---

## Getting Started

<details>
<summary>How do I install wetwire-aws-go?</summary>

See [README.md](../README.md#installation) for installation instructions.
</details>

<details>
<summary>How do I create a new project?</summary>

```bash
wetwire-aws init my-infrastructure
cd my-infrastructure
```
</details>

<details>
<summary>How do I build a CloudFormation template?</summary>

```bash
wetwire-aws build ./...
```
</details>

---

## Syntax

<details>
<summary>How do I reference another resource?</summary>

Use direct variable references:

```go
var MyFunction = lambda.Function{
    Role: MyRole.Arn,  // GetAtt reference via field
}
```
</details>

<details>
<summary>How do I reference a resource without an attribute?</summary>

Use the variable name directly:

```go
var MySecurityGroup = ec2.SecurityGroup{
    VpcId: MyVPC,  // Ref reference
}
```
</details>

<details>
<summary>Why does the linter flag my Ref{} usage?</summary>

The linter enforces direct references for analyzability. Instead of:

```go
// Flagged by WAW015
Role: intrinsics.Ref{Ref: "MyRole"},
```

Use:

```go
// Preferred
Role: MyRole,
```
</details>

<details>
<summary>Why does the linter flag my GetAtt{} usage?</summary>

Same reason. Instead of:

```go
// Flagged by WAW016
Role: intrinsics.GetAtt{LogicalName: "MyRole", AttributeName: "Arn"},
```

Use:

```go
// Preferred
Role: MyRole.Arn,
```
</details>

---

## Lint Rules

<details>
<summary>What do the WAW rule codes mean?</summary>

| Rule | Description |
|------|-------------|
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
</details>

<details>
<summary>How do I auto-fix lint issues?</summary>

```bash
wetwire-aws lint --fix ./...
```
</details>

<details>
<summary>How does the linter help catch errors?</summary>

The linter provides several layers of error detection:

1. **Type safety**: Catches invalid property names and types at compile time
2. **Reference validation**: WAW013 detects undefined references before deployment
3. **Best practices**: Rules like WAW004 (file too large) and WAW005 (extract nested types) improve maintainability
4. **Consistency**: Enforces patterns that make code easier for AI models to understand and generate

Run `wetwire-aws lint ./...` in CI to catch issues early:

```bash
# CI pipeline example
wetwire-aws lint ./... || exit 1
wetwire-aws build ./... > template.json
```
</details>

<details>
<summary>How do I disable a specific lint rule?</summary>

Currently lint rules cannot be disabled individually. If you have a valid use case, file an issue.
</details>

---

## Import

<details>
<summary>How do I convert an existing CloudFormation template?</summary>

```bash
wetwire-aws import template.yaml -o ./my-infrastructure
```
</details>

<details>
<summary>Can I import existing CloudFormation templates?</summary>

Yes. The `import` command converts YAML or JSON CloudFormation templates to Go code:

```bash
# Import a single template
wetwire-aws import template.yaml -o ./infra

# Import generates:
# - go.mod with dependencies
# - infra.go with resource declarations
```

Import handles:
- All resource types
- Parameters and Outputs
- Mappings and Conditions
- Intrinsic functions (Ref, GetAtt, Sub, Join, If, etc.)

For complex templates, run `wetwire-aws lint --fix ./...` after import to clean up the generated code.
</details>

<details>
<summary>Import produced code that doesn't compile?</summary>

Import is best-effort. Complex templates may need manual cleanup:

1. Run `wetwire-aws lint --fix ./...` to apply automatic fixes
2. Review and manually fix remaining issues
3. Check for unsupported intrinsic functions
</details>

<details>
<summary>What CloudFormation features are supported by import?</summary>

- Resources (all types)
- Parameters
- Outputs
- Mappings
- Conditions
- Most intrinsic functions (Ref, GetAtt, Sub, Join, If, etc.)
</details>

---

## Project Structure

<details>
<summary>What's the recommended project structure?</summary>

Organize resources by logical grouping:

```
my-stack/
├── go.mod
├── network.go         # VPC, subnets, security groups
├── compute.go         # EC2, Lambda, containers
├── storage.go         # S3, EFS, databases
├── security.go        # IAM roles, policies
└── outputs.go         # Stack outputs
```

Guidelines:
- Keep files under 20 resources (WAW004)
- Group related resources together
- Use descriptive variable names that match CloudFormation logical IDs
- Extract nested types to separate variables (flat structure)
</details>

<details>
<summary>How do I handle cross-stack references?</summary>

Use CloudFormation exports and imports:

```go
// Stack A: Export a value
var VpcIdOutput = cloudformation.Output{
    Value:      MyVPC,
    Export:     cloudformation.Export{Name: "shared-vpc-id"},
}

// Stack B: Import the value
var MySecurityGroup = ec2.SecurityGroup{
    VpcId: ImportValue("shared-vpc-id"),
}
```

For complex multi-stack architectures, consider:
- Shared parameters via SSM Parameter Store
- Nested stacks for reusable components
- Stack dependencies via CloudFormation StackSets
</details>

---

## CI/CD Integration

<details>
<summary>How do I integrate wetwire-aws with my CI/CD pipeline?</summary>

Example GitHub Actions workflow:

```yaml
name: Infrastructure
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install wetwire-aws
        run: go install github.com/lex00/wetwire-aws-go@latest

      - name: Lint
        run: wetwire-aws lint ./infra/...

      - name: Build template
        run: wetwire-aws build ./infra/... > template.json

      - name: Validate with AWS
        run: aws cloudformation validate-template --template-body file://template.json

      - name: Diff against deployed stack
        run: |
          aws cloudformation get-template --stack-name my-stack > deployed.json
          wetwire-aws diff deployed.json template.json
```

Key integration points:
- **Lint**: Catch errors before deployment
- **Build**: Generate CloudFormation JSON
- **Validate**: Use AWS CLI to validate syntax
- **Diff**: Compare against deployed infrastructure
</details>

---

## Design Mode

<details>
<summary>How do I use AI-assisted design?</summary>

```bash
export ANTHROPIC_API_KEY=your-key
wetwire-aws design
```
</details>

<details>
<summary>What model does design mode use?</summary>

Claude (via Anthropic API). The specific model is configured in wetwire-core-go.
</details>

<details>
<summary>Can I use design mode without an API key?</summary>

No. Design mode requires the Anthropic API.
</details>

---

## Troubleshooting

<details>
<summary>"cannot find package" errors</summary>

Ensure your `go.mod` has the correct module path and dependencies:

```bash
go mod tidy
```
</details>

<details>
<summary>"undefined: s3" or similar import errors</summary>

Add the missing import statement:

```go
import "github.com/lex00/wetwire-aws-go/resources/s3"
```
</details>

<details>
<summary>Build produces empty template</summary>

Check that:
1. Resources are declared as package-level `var` statements
2. Resources have the correct type (e.g., `s3.Bucket`, not `s3.Bucket{}`)
3. The package path is correct in the build command
</details>

<details>
<summary>Circular dependency detected</summary>

Resources cannot have circular references. Review the dependency graph and break the cycle by:
1. Using parameters instead of direct references
2. Restructuring resources

Use the graph command to visualize dependencies:

```bash
wetwire-aws graph ./infra | dot -Tpng -o deps.png
```
</details>

<details>
<summary>"unknown resource type" error</summary>

The resource type may be misspelled or not supported. Check:
1. Correct package import (e.g., `elasticloadbalancingv2` not `elbv2`)
2. Correct type name (e.g., `Listener` not `LoadBalancerListener`)
</details>

<details>
<summary>Import generates code that doesn't compile</summary>

This is expected for complex templates. Fix with:

```bash
# Apply automatic fixes
wetwire-aws lint --fix ./...

# Then manually fix remaining issues
```

Common import issues:
- Forward references (resource used before declaration)
- Complex nested intrinsics
- Custom resource types
</details>

<details>
<summary>"ANTHROPIC_API_KEY not set" error</summary>

Design and test commands require an API key:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
wetwire-aws design "Create an S3 bucket"
```
</details>

<details>
<summary>Lint reports issues but --fix doesn't help</summary>

Some lint rules are advisory and don't have auto-fixes:
- WAW004 (file too large) - manually split the file
- WAW011 (invalid enum) - use correct enum constant
- WAW013 (undefined reference) - fix the reference manually
</details>

<details>
<summary>Build succeeds but CloudFormation deployment fails</summary>

The template is syntactically valid but may have semantic issues:
1. Check IAM permissions
2. Verify resource names are unique in the region
3. Review AWS CloudFormation error messages
</details>

<details>
<summary>SAM template missing Transform header</summary>

The Transform header is added automatically when SAM resources are detected. Ensure you're using `serverless` package types:

```go
import "github.com/lex00/wetwire-aws-go/resources/serverless"

var MyFunc = serverless.Function{...}  // Triggers SAM Transform
```
</details>

<details>
<summary>MCP server connection errors (Kiro)</summary>

If using `--provider kiro`:

```bash
# Ensure wetwire-aws is in PATH
which wetwire-aws

# Re-authenticate with Kiro
kiro-cli login
```
</details>

---

## See Also

- [Wetwire Specification](https://github.com/lex00/wetwire/blob/main/docs/WETWIRE_SPEC.md)
- [CLI Reference]({{< relref "/cli" >}})
- [Quick Start]({{< relref "/quick-start" >}})
- [Import Workflow]({{< relref "/import-workflow" >}})
