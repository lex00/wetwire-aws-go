# Changelog

All notable changes to wetwire-aws (Go) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Codegen: `GetAZs{Region: AWS_REGION}` type mismatch - Region field expects string, not Ref type. Now generates `GetAZs{}` for `!GetAZs !Ref "AWS::Region"` patterns
- Codegen: Unused intrinsics import removed from all generated files (not just mappings)
- Codegen: Wrap intrinsics (`GetAZs{}`, `Split{}`, `If{}`, Parameter refs) in `[]any{}` for list-type fields like `SecurityGroupIds`, `SubnetIds`
- Codegen: `Select{Index: "0"}` generates string instead of int - now correctly generates `Select{Index: 0}`
- Codegen: Variable names colliding with intrinsics types (e.g., `Transform`, `Output`) now get `Resource` suffix
- Codegen: Nested GetAtt attributes (e.g., `!GetAtt MyDB.Endpoint.Address`) now generate `GetAtt{MyDB, "Endpoint.Address"}` instead of invalid field access
- Codegen: Unknown resource types (e.g., `Custom::*`) now generate `any` placeholder variables instead of comments, allowing outputs to reference them without undefined variable errors

### Changed

- Resource types: Pointer fields (`*Type`) changed to `any` to allow intrinsics like `If{}`, `Sub{}` etc.
  - Previously `LoggingConfiguration: &VPCFlowLogsBucketLoggingConfig` - incompatible with `If{}`
  - Now `LoggingConfiguration: VPCFlowLogsBucketLoggingConfig` or `LoggingConfiguration: If{...}`
  - Fixes: vpcflowlogss3, directory_ad_clients, rds_mysql_with_read_replica templates

## [1.4.0] - 2026-01-06

### Added

- **Full template component support**: Complete round-trip support for all CloudFormation template sections
  - `Parameter{}` type with full metadata (Type, Default, AllowedValues, MinLength, MaxLength, etc.) that serializes to `{"Ref": "name"}` when used as property values
  - `Mapping` type as `map[string]map[string]any` for CloudFormation Mappings
  - Discovery of Parameters, Outputs, Mappings, and Conditions in Go packages
  - Template builder outputs complete CloudFormation templates with all sections
  - Import codegen generates full `Parameter{}` structs instead of `Param()` calls
- Helper functions `IntPtr()` and `Float64Ptr()` for parameter constraint fields
- Dynamic CloudFormation type resolution for all 263 services (replaces hardcoded map)
- Round-trip build validation in import script

## [1.3.1] - 2026-01-05

### Added

- **Alexa::ASK support**: Codegen now accepts `Alexa` prefix in addition to `AWS`
  - `ask.Skill` - Alexa Skills Kit skill resource
  - Property types: AuthenticationConfiguration, Overrides, SkillPackage
- Service count increased from 262 to **263** (262 AWS + 1 Alexa)
- Achieves full parity with wetwire-aws-python

## [1.3.0] - 2026-01-05

### Added

- **AWS Serverless Application Model (SAM) support**: All 9 SAM resource types now fully supported
  - `serverless.Function` - Lambda functions with SAM-specific features
  - `serverless.Api` - API Gateway REST APIs
  - `serverless.HttpApi` - API Gateway HTTP APIs
  - `serverless.SimpleTable` - DynamoDB tables
  - `serverless.LayerVersion` - Lambda layers
  - `serverless.StateMachine` - Step Functions state machines
  - `serverless.Application` - Nested applications
  - `serverless.Connector` - Resource permissions
  - `serverless.GraphQLApi` - AppSync GraphQL APIs
- SAM property types: Function_Environment, Function_VpcConfig, Api_CorsConfiguration, etc.
- Auto-detection of SAM resources sets `Transform: AWS::Serverless-2016-10-31` header
- SAM templates can now be imported with `wetwire-aws import`
- Comprehensive tests for SAM resource serialization and template building
- `scripts/import_sam_samples.sh` for testing against official AWS SAM repositories

## [1.2.3] - 2026-01-05

### Changed

- Importer: use backtick strings for multi-line content (ZipFile, InlineCode) for better readability

## [1.2.2] - 2026-01-05

### Fixed

- Importer: neptune template now imports successfully (all 28 parameters detected)

## [1.2.1] - 2026-01-05

### Fixed

- Importer: detect parameters referenced in `!Sub` template strings (e.g., `!Sub ${LambdaHandlerPath}`)

## [1.2.0] - 2026-01-05

### Added

- **Flattened complex property types**: Origins, Rules, and other nested arrays now generate typed block variables instead of inline `Json{}` maps
- `Json{}` type alias for cleaner inline map syntax (replaces `map[string]any{}`)
- WAW017 linter rule: detects pointer assignments (`&Type{}`) - prefer value types
- WAW018 linter rule: detects `map[string]any{}` usage - prefer `Json{}`
- Version info embedded via `runtime/debug.ReadBuildInfo()` for `go install @version`

### Changed

- Array syntax standardized to `[]any{}` everywhere (removed `List()` and `Any()` helpers)
- Type lookup now uses flat CloudFormation pattern (`Distribution_Origin` not `Distribution_DistributionConfig_Origin`)
- PropertyTypeMap generation fixed for array element types

### Fixed

- Importer: complex nested types like CloudFront Origins now flatten to typed structs with IDE autocomplete support
- Importer: S3 ReplicationConfiguration.Rules correctly maps to `Bucket_ReplicationRule`
- Importer: removed pointer assignments from generated code (value types only)
- Documentation: updated QUICK_START examples to use `[]any{}` syntax

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

[unreleased]: https://github.com/lex00/wetwire-aws-go/compare/v1.2.3...HEAD
[1.2.3]: https://github.com/lex00/wetwire-aws-go/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/lex00/wetwire-aws-go/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/lex00/wetwire-aws-go/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v1.0.0
[0.4.1]: https://github.com/lex00/wetwire-aws-go/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v0.4.0
[0.1.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v0.1.0
