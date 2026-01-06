# wetwire-aws (Go)

[![CI](https://github.com/lex00/wetwire-aws-go/actions/workflows/ci.yml/badge.svg)](https://github.com/lex00/wetwire-aws-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/lex00/wetwire-aws-go.svg)](https://pkg.go.dev/github.com/lex00/wetwire-aws-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/lex00/wetwire-aws-go)](https://goreportcard.com/report/github.com/lex00/wetwire-aws-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/lex00/wetwire-aws-go.svg)](https://github.com/lex00/wetwire-aws-go/releases)

Generate CloudFormation templates from Go resource declarations using a declarative, type-safe syntax.

## Status

**v1.3.0 - SAM Support Complete**

- All CLI commands implemented (build, validate, list, lint, init, import, design, test)
- **184 AWS services** with typed enum constants
- **9 SAM resource types** (Function, Api, HttpApi, SimpleTable, LayerVersion, StateMachine, Application, Connector, GraphQLApi)
- 254/254 AWS sample templates import successfully (100% success rate)

See [CHANGELOG.md](CHANGELOG.md) for release details.

## Quick Start

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/s3"
    "github.com/lex00/wetwire-aws-go/resources/iam"
    "github.com/lex00/wetwire-aws-go/resources/lambda"
    . "github.com/lex00/wetwire-aws-go/intrinsics"
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
        "BUCKET": DataBucket,  // Direct resource reference
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

### SAM (Serverless) Resources

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/serverless"
    "github.com/lex00/wetwire-aws-go/resources/dynamodb"
)

// SAM Function with environment variables
var HelloFunction = serverless.Function{
    Handler:    "bootstrap",
    Runtime:    "provided.al2",
    CodeUri:    "./hello/",
    MemorySize: 128,
    Timeout:    30,
    Environment: &serverless.Function_Environment{
        Variables: map[string]any{
            "TABLE_NAME": DataTable.TableName,
        },
    },
}

// DynamoDB table
var DataTable = dynamodb.Table{
    TableName: "my-data-table",
}
```

SAM templates automatically include the `Transform: AWS::Serverless-2016-10-31` header.

## Installation

```bash
go install github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `build` | Generate CloudFormation template from Go source |
| `validate` | Validate resources and references |
| `list` | List discovered resources |
| `lint` | Check for issues (16 rules, --fix support) |
| `init` | Initialize new project |
| `import` | Import CloudFormation template to Go code |
| `design` | AI-assisted infrastructure design (requires wetwire-core-go) |
| `test` | Automated persona-based testing (requires wetwire-core-go) |

## Implementation Status

### What's Working

- **Intrinsic Functions**: All CloudFormation intrinsics (Ref, GetAtt, Sub, Join, etc.)
- **Pseudo-Parameters**: AWS_REGION, AWS_ACCOUNT_ID, AWS_STACK_NAME, etc.
- **AST Discovery**: Parse Go source to find resource declarations
- **Value Extraction**: Extract property values from compiled Go code
- **Template Builder**: Build CF template with topological ordering
- **Cycle Detection**: Detect circular dependencies
- **JSON/YAML Output**: Serialize to CF template format
- **Linter**: 16 rules (WAW001-WAW016) with auto-fix support
- **Code Generator**: Generate Go types from CloudFormation spec
- **SAM Support**: AWS Serverless Application Model (9 resource types)

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
│   ├── import.go          # import command
│   ├── design.go          # design command (AI-assisted)
│   └── test.go            # test command (persona testing)
├── internal/
│   ├── discover/          # AST-based resource discovery
│   ├── importer/          # CloudFormation template importer
│   │   ├── ir.go          # Intermediate representation types
│   │   ├── parser.go      # YAML/JSON template parser
│   │   └── codegen.go     # Go code generator
│   ├── linter/            # Lint rules (WAW001-WAW016)
│   ├── serialize/         # JSON/YAML serialization
│   ├── template/          # Template builder with topo sort
│   └── validation/        # cfn-lint-go integration
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
- [SAM Guide](docs/SAM.md) - Serverless Application Model
- [CLI Reference](docs/CLI.md)
- [Import Workflow](docs/IMPORT_WORKFLOW.md)
- [FAQ](docs/FAQ.md)

## Dependencies

The `design` and `test` commands require:
- [wetwire-core-go](https://github.com/lex00/wetwire-core-go) - AI orchestration and personas
- [cfn-lint-go](https://github.com/lex00/cfn-lint-go) - CloudFormation template validation
- `ANTHROPIC_API_KEY` environment variable

## Related Packages

- [wetwire-core-go](https://github.com/lex00/wetwire-core-go) - AI agent orchestration
- [cfn-lint-go](https://github.com/lex00/cfn-lint-go) - CloudFormation linter (Go port)
- [cloudformation-schema-go](https://github.com/lex00/cloudformation-schema-go) - CF resource types

## License

MIT - See [LICENSE](LICENSE) for details. See [NOTICE](NOTICE) for Apache 2.0 acknowledgements covering AWS contributions.
