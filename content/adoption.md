---
title: "Adoption"
---
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./wetwire-dark.svg">
  <img src="./wetwire-light.svg" width="100" height="67">
</picture>

Practical guidance for teams adopting wetwire-aws alongside existing infrastructure.

---

## Migration Strategies

### Side-by-Side Adoption

You don't need to migrate everything at once. wetwire-aws generates standard CloudFormation templates that deploy with the same CLI you already use.

**Coexistence patterns:**

| Existing Tool | Integration Approach |
|---------------|---------------------|
| Raw CloudFormation | Keep existing templates as-is and gradually add new stacks in Go |
| CDK | Both generate CloudFormation. Deploy CDK stacks and wetwire stacks independently |
| Terraform | Separate state domains. Terraform manages its resources; CloudFormation manages yours |

### Incremental Migration Path

**Week 1: Proof of concept**
- Pick a small, isolated stack (dev environment, internal tool)
- Write it in wetwire-aws
- Verify the generated CloudFormation output
- Deploy to a test environment

**Week 2-4: Build confidence**
- Convert 2-3 more stacks
- Establish team patterns (file organization, naming conventions)
- Set up CI/CD for the new Go stacks

**Ongoing: New stacks in Go**
- All new infrastructure uses wetwire-aws
- Migrate legacy stacks opportunistically (when you're touching them anyway)

### What NOT to Migrate

Some stacks are better left alone:
- **Stable production stacks** that never change
- **Stacks managed by other teams** (coordinate first)
- **CDK stacks with heavy L2/L3 usage** (the abstractions don't translate)

Migration should reduce maintenance burden, not create it.

---

## Escape Hatches

When you hit an edge case the library doesn't handle cleanly.

### Raw CloudFormation Passthrough

For properties not yet typed, use `map[string]any`:

```go
var MyResource = someservice.Resource{
    // Typed properties
    Name: "my-resource",
    // Raw passthrough for untyped/new properties
    SomeNewProperty: map[string]any{
        "Key": "Value",
        "Nested": map[string]any{"Deep": true},
    },
}
```

The serializer passes maps through unchanged.

### Untyped Resources

If a resource type isn't in the library yet (new AWS service, custom resource), you can define it manually:

```go
package infra

// CustomResource for a service not yet in the library
type CustomResource struct {
    PropertyOne string
    PropertyTwo int
}

func (r CustomResource) ResourceType() string {
    return "AWS::NewService::Resource"
}

var MyCustom = CustomResource{
    PropertyOne: "value",
    PropertyTwo: 42,
}
```

This gives you type safety for your properties while using a resource type the library doesn't know about.

### Inline CloudFormation JSON

For complex intrinsic function combinations, use `Json`:

```go
var MyFunction = lambda.Function{
    Environment: lambda.Function_Environment{
        Variables: Json{
            "COMPLEX_VALUE": map[string]any{
                "Fn::Join": []any{"-", []any{
                    map[string]any{"Ref": "AWS::StackName"},
                    map[string]any{"Fn::Select": []any{0, map[string]any{
                        "Fn::Split": []any{",", map[string]any{"Ref": "SomeParam"}},
                    }}},
                }},
            },
        },
    },
}
```

Raw CloudFormation intrinsics pass through. Use this sparingly—if you're doing this often, something's wrong.

### When to Use Escape Hatches

| Situation | Approach |
|-----------|----------|
| New AWS resource type | Custom resource struct |
| New property on existing resource | Raw map passthrough |
| Complex Fn::If/Fn::Join nesting | Inline CloudFormation JSON |
| One-off weird requirement | Whatever works, with a comment |

### When to File an Issue

If you're using escape hatches for:
- Common resource types
- Standard properties
- Patterns other teams would need

...file an issue. The library should handle it.

---

## Team Onboarding

A playbook for getting your team productive in the first week.

### Day 1: Environment Setup

```bash
# Clone your stack repo
git clone <repo>
cd <repo>

# Install wetwire-aws CLI
go install github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest

# Verify it works
wetwire-aws list ./infra/... && echo "OK"
```

**What to check:**
- Go 1.21+ installed
- wetwire-aws CLI available in PATH
- AWS credentials configured (for deployment)

### Day 1-2: Read the Code

Start with a resource file:

```go
package infra

import "github.com/lex00/wetwire-aws-go/resources/s3"

var DataBucket = s3.Bucket{
    BucketName: "my-data",
}
```

That's the pattern. Every resource file looks like this.

### Day 2-3: Make a Small Change

Find something low-risk:
- Add a tag to an existing resource
- Change a property value
- Add a new output

```go
// Before
var MyBucket = s3.Bucket{
    BucketName: "data",
}

// After
var MyBucket = s3.Bucket{
    BucketName: "data",
    Tags: []s3.Tag{
        {Key: "Environment", Value: "dev"},
    },
}
```

Run it, diff the output, deploy to dev.

### Day 3-4: Add a New Resource

Create a new file in the package:

```go
// monitoring.go
package infra

import "github.com/lex00/wetwire-aws-go/resources/sns"

var AlertTopic = sns.Topic{
    TopicName: "alerts",
}
```

Resources auto-register when discovered via AST parsing.

### Day 5: Review the Patterns

By now you've seen:
- Struct literals for resources (e.g., `s3.Bucket{...}`)
- Direct variable references (`MyBucket`, `MyRole.Arn`)
- Flat variables for nested types (extract inline structs)
- Package-level var declarations

That's 90% of what you need.

### Common Gotchas

| Problem | Solution |
|---------|----------|
| "undefined: s3" | Add import for the resource package |
| "Resource not in template" | Ensure it's a package-level var declaration |
| "Wrong property name" | Use IDE autocomplete, or check the AWS docs |
| Build errors after import | Run `wetwire-aws lint --fix ./...` to apply fixes |

### Team Conventions to Establish

Decide these early:
- **File organization**: By service (network.go, compute.go) or by feature?
- **Naming**: PascalCase for variables, matching CloudFormation logical IDs
- **Flat variables**: Always extract nested types to separate vars

Document in your repo's README.

### Resources

- [Quick Start](QUICK_START.md) - 5-minute intro
- [CLI Reference](CLI.md) - Build and validate commands
- [Internals](INTERNALS.md) - How AST discovery works

---

## See Also

- [FAQ](FAQ.md) - Common questions and answers
