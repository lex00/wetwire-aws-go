# Developer Guide

Comprehensive guide for developers working on wetwire-aws-go.

## Table of Contents

- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Running Tests](#running-tests)
- [Code Generation](#code-generation)
- [Contributing](#contributing)
- [Dependencies](#dependencies)

---

## Development Setup

### Prerequisites

- **Go 1.21+** (required)
- **git** (version control)

### Clone and Setup

```bash
# Clone repository
git clone https://github.com/lex00/wetwire-aws-go.git
cd wetwire-aws-go

# Download dependencies
go mod download

# Build CLI
go build -o wetwire-aws ./cmd/wetwire-aws

# Verify installation
./wetwire-aws version
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test -v ./internal/linter/...
```

---

## Project Structure

```
wetwire-aws-go/
├── cmd/wetwire-aws/           # CLI application
│   ├── main.go                # Entry point, command registration
│   ├── build.go               # build command
│   ├── validate.go            # validate command
│   ├── list.go                # list command
│   ├── graph.go               # graph command
│   ├── lint.go                # lint command
│   ├── init.go                # init command
│   ├── import.go              # import command
│   ├── design.go              # design command (AI-assisted)
│   ├── test.go                # test command (persona testing)
│   ├── mcp.go                 # MCP server for Kiro integration
│   └── version.go             # version handling
│
├── internal/
│   ├── discover/              # AST-based resource discovery
│   │   └── discover.go        # Parse Go source for var declarations
│   ├── template/              # CloudFormation template builder
│   │   └── template.go        # Build CF template with topo sort
│   ├── linter/                # Lint rules (WAW001-WAW018)
│   │   ├── rules.go           # Core types + WAW001-WAW010
│   │   └── rules_extra.go     # WAW011-WAW018
│   ├── importer/              # CloudFormation template importer
│   │   ├── ir.go              # Intermediate representation
│   │   ├── parser.go          # YAML/JSON template parser
│   │   ├── codegen.go         # Go code generator
│   │   └── ...                # Supporting files
│   ├── runner/                # Go code execution for value extraction
│   ├── graph/                 # DOT/Mermaid graph generation
│   ├── kiro/                  # Kiro CLI integration
│   ├── serialize/             # JSON/YAML serialization
│   └── validation/            # cfn-lint-go integration
│
├── intrinsics/
│   ├── intrinsics.go          # Ref, GetAtt, Sub, Join, etc.
│   └── pseudo.go              # AWS pseudo-parameters
│
├── codegen/                   # Code generator for resource types
│   ├── main.go                # Entry point
│   ├── parse.go               # Parse CloudFormation spec
│   ├── generate.go            # Generate Go files
│   └── sam_spec.go            # Static SAM definitions
│
├── resources/                 # Generated AWS resource types (264 services)
│   ├── s3/                    # S3 resources
│   ├── lambda/                # Lambda resources
│   ├── iam/                   # IAM resources
│   ├── serverless/            # SAM resources
│   └── ...
│
├── contracts.go               # Core types (Resource, AttrRef, Template)
├── contracts_test.go          # Tests for core types
├── go.mod                     # Module definition
├── go.sum                     # Dependency checksums
├── scripts/
│   ├── ci.sh                  # Local CI script
│   ├── import_aws_samples.sh  # Test against AWS samples
│   └── import_sam_samples.sh  # Test against SAM samples
└── docs/                      # Documentation
```

---

## Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v ./internal/linter/... -run TestWAW001

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### CI Script

The `scripts/ci.sh` script runs the full test suite:

```bash
./scripts/ci.sh
```

This runs:
- `go build ./...`
- `go test ./...`
- `go vet ./...`

---

## Code Generation

The generator produces Go modules for each AWS service.

### Running the Generator

```bash
# Full regeneration (from repo root)
go run ./codegen

# Generate specific service
go run ./codegen --service s3

# Dry run (preview only)
go run ./codegen --dry-run
```

### What Gets Generated

For each service, the generator produces:
- Resource struct with `ResourceType()` method
- Property type structs for nested properties
- Attribute fields for `Fn::GetAtt` references

**Example (resources/s3/bucket.go):**

```go
package s3

import wetwire "github.com/lex00/wetwire-aws-go"

type Bucket struct {
    // Attributes for Fn::GetAtt
    Arn        wetwire.AttrRef `json:"-"`
    DomainName wetwire.AttrRef `json:"-"`

    // Properties
    BucketName       any `json:"BucketName,omitempty"`
    BucketEncryption any `json:"BucketEncryption,omitempty"`
}

func (r Bucket) ResourceType() string {
    return "AWS::S3::Bucket"
}
```

---

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for:
- Code style guidelines
- Commit message format
- Pull request process
- Adding new lint rules
- Adding new CLI commands

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | YAML parsing/generation |
| `github.com/stretchr/testify` | Test assertions |
| `github.com/lex00/cloudformation-schema-go` | CloudFormation spec |
| `github.com/lex00/cfn-lint-go` | Template validation |
| `github.com/lex00/wetwire-core-go` | AI orchestration (design/test) |
| `github.com/modelcontextprotocol/go-sdk` | MCP server |

---

## See Also

- [Quick Start](QUICK_START.md) - Getting started
- [CLI Reference](CLI.md) - CLI commands
- [Internals](INTERNALS.md) - Architecture details
- [Versioning](VERSIONING.md) - Version management
