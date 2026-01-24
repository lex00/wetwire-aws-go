---
title: "Wetwire AWS"
---

[![Go Reference](https://pkg.go.dev/badge/github.com/lex00/wetwire-aws-go.svg)](https://pkg.go.dev/github.com/lex00/wetwire-aws-go)
[![CI](https://github.com/lex00/wetwire-aws-go/actions/workflows/ci.yml/badge.svg)](https://github.com/lex00/wetwire-aws-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/lex00/wetwire-aws-go/graph/badge.svg)](https://codecov.io/gh/lex00/wetwire-aws-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/lex00/wetwire-aws-go)](https://goreportcard.com/report/github.com/lex00/wetwire-aws-go)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Generate AWS CloudFormation templates from Go structs with AI-assisted design.

## Philosophy

Wetwire uses typed constraints to reduce the model capability required for accurate code generation.

**Core hypothesis:** Typed input + smaller model ≈ Semantic input + larger model

The type system and lint rules act as a force multiplier — cheaper models produce quality output when guided by schema-generated types and iterative lint feedback.

## Documentation

| Document | Description |
|----------|-------------|
| [CLI Reference]({{< relref "/cli" >}}) | Command-line interface |
| [Quick Start]({{< relref "/quick-start" >}}) | Get started in 5 minutes |
| [Examples]({{< relref "/examples" >}}) | Sample CloudFormation projects |
| [FAQ]({{< relref "/faq" >}}) | Frequently asked questions |

## Installation

```bash
go install github.com/lex00/wetwire-aws-go@latest
```

## Quick Example

```go
// Keep this minimal - just show the pattern
var MyBucket = s3.Bucket{
    BucketName: "my-data-bucket",
    Versioning: s3.VersioningEnabled,
}
```
