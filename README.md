<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 67 45" width="67" height="45" align="right">
  <style>
    text {
      font-family: 'Consolas', 'DejaVu Sans Mono', 'Courier New', monospace;
      font-size: 7px;
      fill: #1f2328;
    }
    @media (prefers-color-scheme: dark) {
      text { fill: #e6edf3; }
    }
  </style>
  <text x="4" y="9">&#160;&#160;╭↯↯↯↯╮</text>
  <text x="4" y="18">&#160;┌┴────┴┐</text>
  <text x="4" y="27">&#160;│︵&#160;&#160;︵&#160;│</text>
  <text x="4" y="36">&#160;│◎&#160;⬭&#160;◎&#160;│</text>
  <text x="4" y="45">&#160;└──────┘</text>
</svg>

# wetwire-aws (Go)

[![CI](https://github.com/lex00/wetwire-aws-go/actions/workflows/ci.yml/badge.svg)](https://github.com/lex00/wetwire-aws-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/lex00/wetwire-aws-go/branch/main/graph/badge.svg)](https://codecov.io/gh/lex00/wetwire-aws-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/lex00/wetwire-aws-go.svg)](https://pkg.go.dev/github.com/lex00/wetwire-aws-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/lex00/wetwire-aws-go)](https://goreportcard.com/report/github.com/lex00/wetwire-aws-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/lex00/wetwire-aws-go.svg)](https://github.com/lex00/wetwire-aws-go/releases)

AWS CloudFormation synthesis using Go struct literals.

## Installation

```bash
go install github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest
```

## Quick Example

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/s3"
    "github.com/lex00/wetwire-aws-go/resources/iam"
    "github.com/lex00/wetwire-aws-go/resources/lambda"
    . "github.com/lex00/wetwire-aws-go/intrinsics"
)

var MyBucket = s3.Bucket{
    BucketName: "my-data",
}

var MyRole = iam.Role{
    RoleName: "processor-role",
}

var MyFunction = lambda.Function{
    FunctionName: "processor",
    Runtime:      lambda.RuntimePython312,
    Role:         MyRole.Arn,  // Type-safe GetAtt reference
}
```

```bash
wetwire-aws build ./infra > template.json
```

## Serverless (SAM) Support

Build serverless applications with type-safe SAM resources:

```go
package infra

import "github.com/lex00/wetwire-aws-go/resources/serverless"

var ProcessorFunction = serverless.Function{
    Handler:    "bootstrap",
    Runtime:    "provided.al2",
    CodeUri:    "./src",
    MemorySize: 128,
    Timeout:    30,
}
```

All 9 SAM resource types supported: `Function`, `Api`, `HttpApi`, `SimpleTable`, `LayerVersion`, `StateMachine`, `Application`, `Connector`, `GraphQLApi`.

## AI-Assisted Design

Create infrastructure interactively with AI:

```bash
# No API key required - uses Claude CLI
wetwire-aws design "Create an encrypted S3 bucket"

# Automated testing with personas
wetwire-aws test --persona beginner "Create a Lambda function"
```

Uses [Claude CLI](https://claude.ai/download) by default (no API key required). Falls back to Anthropic API if Claude CLI is not installed. See [CLI Reference](docs/CLI.md#design) for details.

## Documentation

**Getting Started:**
- [Quick Start](docs/QUICK_START.md) - 5-minute tutorial
- [FAQ](docs/FAQ.md) - Common questions

**Reference:**
- [CLI Reference](docs/CLI.md) - All commands
- [SAM Guide](docs/SAM.md) - Serverless resources
- [Lint Rules](docs/LINT_RULES.md) - WAW rule reference

**Advanced:**
- [Internals](docs/INTERNALS.md) - Architecture and extension points
- [Adoption Guide](docs/ADOPTION.md) - Team migration strategies
- [Examples](docs/EXAMPLES.md) - Imported template catalog

## Development

```bash
git clone https://github.com/lex00/wetwire-aws-go.git
cd wetwire-aws-go
go mod download
go test ./...           # Run tests
./scripts/ci.sh         # Full CI checks
```

See [Developer Guide](docs/DEVELOPERS.md) and [Contributing](CONTRIBUTING.md) for details.

## License

MIT - See [LICENSE](LICENSE) for details. Third-party attributions in [NOTICE](NOTICE).
