# Changelog

All notable changes to wetwire-aws (Go) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-01-05

### Added

- Integration test for importer: validates 12 complex AWS templates compile successfully
- WAW013 linter rule: detects undefined references in generated code
- WAW014 linter rule: detects unused intrinsics imports
- WAW015 linter rule: detects explicit Ref{} (prefer direct variable refs or Param())
- WAW016 linter rule: detects explicit GetAtt{} (prefer Resource.Attr field access)
- CLI documentation for design, test, validate, list commands

### Fixed

- Importer: never generates Ref{} or GetAtt{} patterns (always uses direct refs and field access)
- Importer: variable names with hyphens now sanitized (e.g., `Port-1ICMP` → `PortNeg1ICMP`)
- Importer: unknown resource types skipped with comment instead of broken imports
- Importer: intrinsics import only added when intrinsic types are actually used
- Importer: pre-scan conditions for parameter references (fixes missing param generation)
- Importer: runs `go mod tidy` after generating scaffold files
- Fixed go.mod replace directive path in generated examples

## [1.0.0] - 2026-01-03

### Changed

- Updated cloudformation-schema-go from v0.7.0 to v1.0.0
- Updated cfn-lint-go from v0.7.2 to v1.0.0
- Updated wetwire-core-go from v0.1.0 to v1.0.0
- Updated spf13/cobra from v1.8.1 to v1.10.2

### Fixed

- Fixed unchecked error return values in internal/runner/runner.go (lint)

## [0.4.1] - 2026-01-02

### Fixed

- Module path alignment: changed from `github.com/lex00/wetwire-aws` to
  `github.com/lex00/wetwire-aws-go` to match actual repository location
- This enables proper pkg.go.dev indexing and `go install` from the correct path

### Changed

- Updated all import statements (1,268 occurrences) to use new module path
- Updated documentation across wetwire-aws, wetwire-agent, and research docs
- Marked all features as complete in ImplementationChecklist.md

## [0.4.0] - 2026-01-02

### Added

- Typed enum constants for 184 AWS services via cloudformation-schema-go v0.7.0:
  - Full coverage: Lambda, S3, EC2, DynamoDB, ECS, RDS, IAM, CloudWatch, SQS, SNS, and 174 more
  - 10,014 enum types with 45,318 total values
  - Enables type-safe property values (e.g., `lambda.RuntimePython312` instead of `"python3.12"`)
- Category-based file splitting for imported templates:
  - `security.go` - IAM roles, policies, KMS keys
  - `network.go` - VPC, subnets, security groups, load balancers
  - `compute.go` - EC2 instances, Lambda functions
  - `storage.go` - S3 buckets, EFS
  - `database.go` - RDS, DynamoDB
  - `messaging.go` - SQS, SNS, EventBridge
  - `monitoring.go` - CloudWatch, logs
  - `cicd.go` - CodePipeline, CodeBuild
  - `infra.go` - CloudFormation, SSM
- Scaffold file generation by default (`--no-scaffold` to opt out):
  - `go.mod` with replace directive for local development
  - `cmd/main.go` placeholder
  - `README.md` with project documentation
  - `CLAUDE.md` with AI assistant instructions
  - `.gitignore` for build artifacts
- IAM policy document flattening with typed helpers:
  - `PolicyDocument`, `PolicyStatement` types
  - `ServicePrincipal`, `AWSPrincipal`, `FederatedPrincipal` helpers
  - Condition operator constants (StringEquals, ArnLike, etc.)
- Typed property struct generation instead of `map[string]any`
- `Param()` and `Output()` types for cleaner generated code
- WAW009 linter rule to detect unflattened `map[string]any` (recursive)
- PascalCase for PropertyType submodule directories

### Fixed

- Parser stack overflow on GetAZs and ValueOf intrinsics with cycle detection
- Package name collisions in importer (reserved names get `_stack` suffix)
- Simplified `cmd/main.go` scaffold to avoid import issues

## [0.1.0] - 2024-12-26

### Added

- Initial Go implementation of wetwire-aws
- `wetwire-aws import` command to convert CloudFormation YAML/JSON to Go
- `wetwire-aws build` command to synthesize templates
- `wetwire-aws lint` command with auto-fix support
- Intrinsic function types: Ref, GetAtt, Sub, Join, Select, If, etc.
- Pseudo-parameter constants: AWS_REGION, AWS_ACCOUNT_ID, AWS_STACK_NAME
- Resource codegen from CloudFormation specification
- Block-style code generation with typed property types
- 254/254 AWS sample templates import successfully (100% success rate)

[unreleased]: https://github.com/lex00/wetwire-aws-go/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v1.0.0
[0.4.1]: https://github.com/lex00/wetwire-aws-go/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v0.4.0
[0.1.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v0.1.0
