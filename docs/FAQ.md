# wetwire-aws-go FAQ

This FAQ covers questions specific to the Go implementation of wetwire for AWS CloudFormation. For general wetwire questions, see the [central FAQ](https://github.com/lex00/wetwire/blob/main/docs/FAQ.md).

---

## Getting Started

### How do I install wetwire-aws-go?

```bash
go install github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest
```

### How do I create a new project?

```bash
wetwire-aws init my-infrastructure
cd my-infrastructure
```

### How do I build a CloudFormation template?

```bash
wetwire-aws build ./...
```

---

## Syntax

### How do I reference another resource?

Use direct variable references:

```go
var MyFunction = lambda.Function{
    Role: MyRole.Arn,  // GetAtt reference via field
}
```

### How do I reference a resource without an attribute?

Use the variable name directly:

```go
var MySecurityGroup = ec2.SecurityGroup{
    VpcId: MyVPC,  // Ref reference
}
```

### Why does the linter flag my `Ref{}` usage?

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

### Why does the linter flag my `GetAtt{}` usage?

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

---

## Lint Rules

### What do the WAW rule codes mean?

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

### How do I auto-fix lint issues?

```bash
wetwire-aws lint --fix ./...
```

### How do I disable a specific lint rule?

Currently lint rules cannot be disabled individually. If you have a valid use case, file an issue.

---

## Import

### How do I convert an existing CloudFormation template?

```bash
wetwire-aws import template.yaml -o ./my-infrastructure
```

### Import produced code that doesn't compile?

Import is best-effort. Complex templates may need manual cleanup:

1. Run `wetwire-aws lint --fix ./...` to apply automatic fixes
2. Review and manually fix remaining issues
3. Check for unsupported intrinsic functions

### What CloudFormation features are supported by import?

- Resources (all types)
- Parameters
- Outputs
- Mappings
- Conditions
- Most intrinsic functions (Ref, GetAtt, Sub, Join, If, etc.)

---

## Design Mode

### How do I use AI-assisted design?

```bash
export ANTHROPIC_API_KEY=your-key
wetwire-aws design
```

### What model does design mode use?

Claude (via Anthropic API). The specific model is configured in wetwire-core-go.

### Can I use design mode without an API key?

No. Design mode requires the Anthropic API.

---

## Troubleshooting

### "cannot find package" errors

Ensure your `go.mod` has the correct module path and dependencies:

```bash
go mod tidy
```

### Build produces empty template

Check that:
1. Resources are declared as package-level `var` statements
2. Resources have the correct type (e.g., `s3.Bucket`, not `s3.Bucket{}`)
3. The package path is correct in the build command

### Circular dependency detected

Resources cannot have circular references. Review the dependency graph and break the cycle by:
1. Using parameters instead of direct references
2. Restructuring resources

---

## Resources

- [Wetwire Specification](https://github.com/lex00/wetwire/blob/main/docs/WETWIRE_SPEC.md)
- [CLI Documentation](CLI.md)
- [Quick Start](QUICK_START.md)
- [Import Workflow](IMPORT_WORKFLOW.md)
