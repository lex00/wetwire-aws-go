# wetwire-aws (Go)

Generate CloudFormation templates from Go resource declarations using a declarative, type-safe syntax.

## Status

**v0.4.0 - All Core Features Complete**

- All CLI commands implemented (build, validate, list, lint, init, import)
- **184 AWS services** with typed enum constants
- 254/254 AWS sample templates import successfully (100% success rate)

See [CHANGELOG.md](CHANGELOG.md) for release details.

## Quick Start

```go
package infra

import (
    "github.com/lex00/wetwire/go/wetwire-aws/resources/s3"
    "github.com/lex00/wetwire/go/wetwire-aws/resources/iam"
    "github.com/lex00/wetwire/go/wetwire-aws/resources/lambda"
    . "github.com/lex00/wetwire/go/wetwire-aws/intrinsics"
)

// Direct type declaration - no wrappers, no registration
var DataBucket = s3.Bucket{
    BucketName: "my-data-bucket",
}

var ProcessorRole = iam.Role{
    RoleName: "processor-role",
}

// Environment extracted to flat variable
var ProcessorEnv = lambda.Environment{
    Variables: Json{
        "BUCKET": Ref{"DataBucket"},
    },
}

var ProcessorFunction = lambda.Function{
    FunctionName: "processor",
    Role:         ProcessorRole.Arn,  // GetAtt via field access
    Environment:  ProcessorEnv,
}
```

Generate template:

```bash
wetwire-aws build ./infra > template.json
```

## Installation

```bash
go install github.com/lex00/wetwire/go/wetwire-aws/cmd/wetwire-aws@latest
```

## CLI Commands

| Command | Status | Description |
|---------|--------|-------------|
| `build` | ✅ Complete | Generate CloudFormation template |
| `validate` | ✅ Complete | Validate resources and references |
| `list` | ✅ Complete | List discovered resources |
| `lint` | ✅ Complete | Check for issues (6 rules, --fix support) |
| `init` | ✅ Complete | Initialize new project |
| `import` | ✅ Complete | Import CF template to Go code |

## Implementation Status

### What's Working

- **Intrinsic Functions**: All CloudFormation intrinsics (Ref, GetAtt, Sub, Join, etc.)
- **Pseudo-Parameters**: AWS_REGION, AWS_ACCOUNT_ID, AWS_STACK_NAME, etc.
- **AST Discovery**: Parse Go source to find resource declarations
- **Value Extraction**: Extract property values from compiled Go code
- **Template Builder**: Build CF template with topological ordering
- **Cycle Detection**: Detect circular dependencies
- **JSON/YAML Output**: Serialize to CF template format
- **Linter**: 6 rules (WAW001-WAW006) with auto-fix support
- **Code Generator**: Generate Go types from CloudFormation spec

### What's Missing

All core CLI commands are now implemented. Potential future enhancements:

| Feature | Priority | Description |
|---------|----------|-------------|
| Additional lint rules | P3 | Port more Python lint rules to Go |

## Package Structure

```
wetwire-aws/
├── cmd/wetwire-aws/       # CLI application
│   ├── main.go            # Entry point
│   ├── build.go           # build command
│   ├── validate.go        # validate command
│   ├── list.go            # list command
│   ├── lint.go            # lint command
│   ├── init.go            # init command
│   └── import.go          # import command
├── internal/
│   ├── discover/          # AST-based resource discovery
│   ├── importer/          # CloudFormation template importer
│   │   ├── ir.go          # Intermediate representation types
│   │   ├── parser.go      # YAML/JSON template parser
│   │   └── codegen.go     # Go code generator
│   ├── serialize/         # JSON/YAML serialization
│   └── template/          # Template builder with topo sort
├── intrinsics/
│   ├── intrinsics.go      # Ref, GetAtt, Sub, Join, etc.
│   └── pseudo.go          # AWS pseudo-parameters
├── codegen/               # Generate Go types from CF spec
│   ├── fetch.go           # Download CF spec
│   ├── parse.go           # Parse spec JSON
│   └── generate.go        # Generate Go files
├── contracts.go           # Core types (Resource, AttrRef, Template)
├── docs/
│   ├── QUICK_START.md
│   └── CLI.md
└── scripts/
    ├── ci.sh              # Local CI script
    └── import_aws_samples.sh  # Test against AWS samples
```

## Development

```bash
# Run tests
go test -v ./...

# Run CI checks
./scripts/ci.sh

# Build CLI
go build -o wetwire-aws ./cmd/wetwire-aws
```

## Documentation

- [Quick Start](docs/QUICK_START.md)
- [CLI Reference](docs/CLI.md)
- [Implementation Checklist](../../docs/research/ImplementationChecklist.md)
- [Go Design Decisions](../../docs/research/Go.md)

## Related Packages

- [wetwire-agent](../wetwire-agent/) - AI agent for infrastructure design
- [wetwire-aws (Python)](../../python/packages/wetwire-aws/) - Python implementation
