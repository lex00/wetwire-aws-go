# Changelog

All notable changes to wetwire-aws (Go) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Test: Split `internal/lint/rules_test.go` (1,265 lines) into 5 focused files (#205)
  - `rules_core_test.go` (323 lines) - WAW001-WAW008 core rules tests
  - `rules_advanced_test.go` (399 lines) - WAW009-WAW014 advanced rules tests
  - `rules_references_test.go` (205 lines) - WAW015-WAW018 reference/type rules tests
  - `rules_secrets_test.go` (249 lines) - WAW019 secret detection + helper tests
  - `rules_lint_test.go` (133 lines) - LintFile, LintPackage integration tests
  - All tests pass, no file exceeds 600 lines
- Linter: Renamed `internal/linter` to `internal/lint` for consistency (#208)
  - Package name changed from `linter` to `lint`
  - Updated all imports and documentation references
- Discover: Migrated to use `wetwire-core-go/ast` package for shared AST utilities (#207)
  - Uses `coreast.ExtractTypeName` instead of local implementation
  - Uses `coreast.IsBuiltinIdent` for Go builtin checks
  - Removes 13 lines of duplicated code
- Linter: Migrated to use `wetwire-core-go/lint` package for shared lint types (#206)
  - Uses type aliases for `Issue` and `Severity` from core lint package
  - Maintains backward compatibility with existing code
  - Reduces duplication across domain repos
- Deps: Updated wetwire-core-go to v1.16.0 for lint and ast packages (#206, #207)
- MCP: Migrated to `domain.BuildMCPServer()` for auto-generated MCP server (#194)
  - Reduced `cmd/wetwire-aws/mcp.go` from 641 lines to 60 lines (~90% reduction)
  - Removed all manual tool registration and handler functions
  - Now uses `domain.BuildMCPServer()` from wetwire-core-go v1.13.0
  - MCP tools automatically generated from Domain interface implementation
  - All 7 tools (init, build, lint, validate, import, list, graph) work seamlessly
- Deps: Updated wetwire-core-go to v1.13.0 for BuildMCPServer support (#194)
- MCP: Refactored MCP server to use domain interface for automatic tool generation (#185)
  - Replaced ~750 lines of manual MCP handlers with domain-based approach
  - MCP tools now automatically generated from `domain.Domain` interface
  - Uses `RegisterStandardTools` from wetwire-core-go for consistent tool registration
  - All 7 MCP tools (init, build, lint, validate, import, list, graph) now bridge directly to domain implementations
- Deps: Updated wetwire-core-go to v1.5.4 for Kiro provider cwd fix (#181)
  - Ensures Kiro agent runs in correct working directory
  - Added test to verify WorkDir is set in NewConfig()

### Added

- CI: Codecov integration for coverage reporting (#163)
  - Test step now generates coverage profile with `go test -coverprofile=coverage.out -covermode=atomic`
  - Coverage uploaded to Codecov after tests
  - Added codecov badge to README.md

## [1.11.0] - 2026-01-10

### Added

- CLI: `diff` command for semantic comparison of CloudFormation templates (#146)
  - Compares two templates and shows added, removed, and modified resources
  - Supports JSON and YAML input formats
  - Text and JSON output formats
  - `--ignore-order` flag for array element order-insensitive comparison
- CLI: `watch` command for auto-rebuild on source file changes (#146)
  - Monitors source directories for `.go` file changes
  - Runs lint on each change, builds if lint passes
  - `--lint-only` flag to skip build step
  - `--debounce` flag to control rapid change handling (default: 500ms)
  - Skips hidden directories, vendor, and generated `_wetwire_gen.go` files
- Docs: `docs/LINT_RULES.md` - comprehensive documentation for all WAW lint rules (#148)
  - Rule index table with descriptions, severities, and auto-fix support
  - Detailed examples (bad/good) for each rule
  - Contributing guide for new rules
- Docs: `examples/README.md` - attribution for imported AWS example templates (#147)
  - Attribution to aws-cloudformation-templates and aws-sam-cli-app-templates repos
  - Apache 2.0 license notice
  - Template counts and purpose documentation
- Lint: WAW019 secret pattern detection rule (#142)
  - Detects AWS access keys (AKIA..., ASIA...)
  - Detects private key headers (-----BEGIN ... PRIVATE KEY-----)
  - Detects Stripe, GitHub, Slack API tokens
  - Flags hardcoded values in sensitive fields (password, secret, token, etc.)
  - Skips placeholders and common safe patterns
- Test: Benchmark tests for template builder (#136)
  - Build benchmarks for 10-200 resources
  - JSON/YAML serialization benchmarks
  - Topological sort benchmarks with dependency chains
- Test: Runner package integration tests (#136)
  - Coverage improved from 10.4% to 68.7%
  - Integration tests with real Go packages for value extraction
  - Tests for ExtractAll with parameters, outputs, and mappings
  - Edge case tests for go.mod parsing
- Test: Linter package tests (#136)
  - Coverage improved from 46.2% to 83.1%
  - Tests for LintFile, LintPackage, lintRecursive
  - Tests for getRules with filtering and MaxResources options
  - Tests for InlineMapInSlice, UnusedIntrinsicsImport, AvoidExplicitRef, AvoidExplicitGetAtt, PreferJsonType, PreferEnumConstant rules
- Test: Discover package tests (#136)
  - Coverage improved from 52.3% to 82.2%
  - Tests for Parameters, Outputs, Mappings discovery
  - Tests for ResolveAttrRefs with recursive resolution
  - Tests for isCommonIdent and isIntrinsicPackage helpers
- Docs: `docs/INTRINSICS.md` - comprehensive guide to intrinsic functions (#136)
  - Pseudo-parameters reference table
  - All intrinsic function types with examples
  - IAM policy types documentation
  - Best practices and CloudFormation mapping
- Test: Round-trip testing per spec 11.2 (#139)
  - 20 reference YAML templates in `internal/importer/testdata/reference/`
  - Semantic comparison for CloudFormation templates
  - Tests flow: YAML -> import -> Go code -> build -> compare
  - CI job for round-trip test visibility
- Feature: 5-dimension scoring system per spec 6.4 (#138)
  - `TestScore` struct with Completeness, LintQuality, CodeQuality, OutputValidity, QuestionEfficiency
  - Grade scale: Failure (0-5), Partial (6-9), Success (10-12), Excellent (13-15)
  - `internal/scoring` package with scoring calculation logic
  - `TestResult` and `TestSummary` types in contracts.go
- Feature: Add write_file and read_file tools to MCP server per spec 6.2 (#137)
  - `wetwire_write_file` tool for writing source files with parent directory creation
  - `wetwire_read_file` tool for reading file contents
  - Both tools available in embedded and standalone MCP servers
- Docs: New documentation files ported from Python implementation
  - `CONTRIBUTING.md` - Contributing guidelines (root level)
  - `docs/ADOPTION.md` - Migration strategies, escape hatches, team onboarding
  - `docs/CODEGEN.md` - Resource type generation pipeline
  - `docs/DEVELOPERS.md` - Development setup and workflow
  - `docs/EXAMPLES.md` - Imported template catalog (126 CF + 56 SAM)
  - `docs/INTERNALS.md` - AST discovery, template generation, linter architecture
  - `docs/VERSIONING.md` - Version management and release process
- Docs: Expanded FAQ.md troubleshooting section with 9 additional common errors
- Test: Provider flag tests for `design` and `test` commands (#157)

### Changed

- Kiro: Renamed agent from `wetwire-runner` to `wetwire-aws-runner` for domain-specific naming (#157)
- Kiro: MCP server now uses standalone `wetwire-aws-mcp` binary instead of embedded `--mcp-server` flag (#157)
- CLI: Updated `design` and `test` commands help text to reference new agent name (#157)
- Docs: Updated AWS-KIRO-CLI.md with new agent name and MCP configuration (#157)
- Kiro: Removed local `internal/kiro/personas.go`, use core `personas` package for validation (#158)
- CLI: `test` command now uses `personas.Names()` from wetwire-core-go for persona list (#158)
- Deps: Updated wetwire-core-go from v1.0.1 to v1.2.0 (#159)
- Refactor: Split `codegen.go` (2993 lines) into 5 files (#130)
  - `codegen.go` (910 lines) - main entry points
  - `codegen_values.go` (1090 lines) - value conversion
  - `codegen_intrinsics.go` (394 lines) - intrinsic handling
  - `codegen_helpers.go` (408 lines) - constants and utilities
  - `codegen_policy.go` (222 lines) - policy document handling
- Refactor: Split `rules.go` (1892 lines) into 2 files (#130)
  - `rules.go` (1010 lines) - core types + WAW001-WAW010
  - `rules_extra.go` (891 lines) - WAW011-WAW018
- Docs: Added package documentation to importer, runner, graph packages
- Docs: Added function documentation to cmd/ handlers
- Docs: Streamlined README.md (255 → 105 lines) to match Python version
  - Added prominent AI-Assisted Design section
  - Moved detailed sections to dedicated docs

### Removed

- Internal: Removed unused `internal/serialize` package (#159)
  - Serialization logic is domain-specific and handled in `internal/runner`
  - Core `serialize` package available for future use

### Fixed

- Docs: Fixed code examples to use flat variables instead of pointers (SAM.md, README.md)
- Docs: Updated outdated counts (264 services, 18 lint rules, v1.10.0)

## [1.10.0] - 2026-01-09

### Added

- Build: Auto-generate `go.mod` when not found (#124)
  - `wetwire-aws build .` now works in directories without `go.mod`
  - User's Go files are copied to a temp directory with synthetic module
  - No files created in user's directory
- SAM: Added missing resource fields (#125, #127)
  - `serverless.SimpleTable.Arn` attribute accessor
  - `serverless.HttpApi.Domain`, `serverless.HttpApi.Description`
  - `serverless.Api.Domain`, `serverless.Api.Description`
  - `serverless.Api_Auth.InvokeRole`
  - `serverless.Function_DeploymentPreference.TriggerConfigurations`

### Fixed

- Import: List-type parameters now wrapped in `[]any{}` (#128)
  - `CommaDelimitedList` and `List<*>` parameters wrapped when used in struct fields
  - Fixes "cannot use Parameter as []any" build errors
- Import: Maps wrapped in `[]any{}` for array fields (#128)
  - SAM `Policies` field and similar now accept inline policy maps
- Import: Initialization cycles detected and broken (#128)
  - Circular variable references now use explicit `GetAtt{}` to break cycles
  - SAM import build success rate: 72% → 84% (48/57 templates)

## [1.9.0] - 2026-01-09

### Added

- `graph` command for DOT/Mermaid dependency visualization (#121)
  - Generate Graphviz DOT format: `wetwire-aws graph ./infra`
  - Generate Mermaid format: `wetwire-aws graph ./infra -f mermaid`
  - Include parameters: `wetwire-aws graph ./infra -p`
  - Cluster by service: `wetwire-aws graph ./infra -c`
  - Blue edges for GetAtt, solid for Ref dependencies

### Fixed

- Import: SAM implicit resources (like auto-generated IAM roles) now use explicit `GetAtt{}` (#115)
  - Detects `AWS::Serverless::Function` and adds `{Name}Role` to known implicit resources
  - Unknown Ref/GetAtt targets now use explicit intrinsic forms to avoid undefined variables

## [1.7.0] - 2026-01-09

### Added

- Embedded MCP server in main binary for simplified design mode (#106)
  - `wetwire-aws design --mcp-server` hidden flag runs embedded MCP server
  - MCP config now includes `cwd` field for correct path resolution
  - No longer requires separate `wetwire-aws-mcp` binary
  - Kiro tools now work correctly with `./infra/...` paths

## [1.6.2] - 2026-01-08

### Fixed

- CLI: Add `--version` flag (in addition to `version` subcommand)

## [1.6.1] - 2026-01-08

### Added

- Kiro TestRunner: PTY handling via `script` command (#104)
  - kiro-cli requires TTY even with `--no-interactive`
  - Supports both macOS and Linux `script` syntax
  - Enables reliable automated testing
- Kiro agent prompt: ASCII state machine diagram for lint-fix loop
  - Visual workflow enforcement matching Python implementation

## [1.6.0] - 2026-01-08

### Added

- Kiro CLI provider for design mode (#101)
  - New MCP server (`wetwire-aws-mcp`) exposes `wetwire_init`, `wetwire_lint`, `wetwire_build` tools
  - `wetwire-aws design --provider kiro` launches Kiro CLI with wetwire-runner agent
  - Auto-installs agent config to `~/.kiro/agents/wetwire-runner.json`
  - Auto-installs project MCP config to `.kiro/mcp.json`
- Init command now creates `infra/params.go` for Parameters, Mappings, Conditions
- Init command now creates `infra/outputs.go` for Outputs
- Offline builds support (#100)
  - Detects `vendor/` directory and uses in-module `_wetwire_runner` subdirectory
  - Runs with `-mod=vendor` flag for fully offline builds
  - No network access required when dependencies are vendored

### Fixed

- Runner: Import path calculation for subpackages in vendor mode
  - Now correctly calculates full import path (e.g., `mymodule/infra`) instead of just module path
- Discovery: Skip blank identifier (`_`) variables
  - Prevents errors when packages use `var _ = Type{}` placeholders

## [1.4.2] - 2026-01-07

### Fixed

- Builder: Empty `Fn::GetAtt` for AttrRefs inside intrinsic functions (`Join`, `Sub`, `If`, etc.) now correctly resolved (#92)
  - Root cause: Path mismatch between discovery (Go field names) and serialization (CF intrinsic keys)
  - Added `intrinsicFieldNames` mapping to translate CF array positions to Go field names
- Discovery: Added all 263 AWS services to `knownResourcePackages` (#90)
  - Previously only 21 services were recognized, causing resources from ApplicationAutoScaling, CloudWatch, ElasticLoadBalancingV2, AutoScaling, and 240+ other services to be silently skipped
  - ECS schedule example now discovers 24 resources (was 15)

## [1.4.1] - 2026-01-07

### Fixed

- Codegen: `GetAZs{Region: AWS_REGION}` type mismatch - Region field expects string, not Ref type. Now generates `GetAZs{}` for `!GetAZs !Ref "AWS::Region"` patterns
- Codegen: Unused intrinsics import removed from all generated files (not just mappings)
- Codegen: Wrap intrinsics (`GetAZs{}`, `Split{}`, `If{}`, Parameter refs) in `[]any{}` for list-type fields like `SecurityGroupIds`, `SubnetIds`
- Codegen: `Select{Index: "0"}` generates string instead of int - now correctly generates `Select{Index: 0}`
- Codegen: Variable names colliding with intrinsics types (e.g., `Transform`, `Output`) now get `Resource` suffix
- Codegen: Nested GetAtt attributes (e.g., `!GetAtt MyDB.Endpoint.Address`) now generate `GetAtt{MyDB, "Endpoint.Address"}` instead of invalid field access
- Codegen: Unknown resource types (e.g., `Custom::*`) now generate `any` placeholder variables instead of comments, allowing outputs to reference them without undefined variable errors
- Codegen: `Fn::Transform` now generates `Transform{Name: "...", Parameters: {...}}` with proper struct fields instead of raw value passthrough
- Codegen: `ResourceType` field in nested property types no longer incorrectly transformed to `ResourceTypeProp` (only top-level resources have the conflicting method)
- Codegen: Duplicate array element variable names now get `_2`, `_3` suffixes to avoid redeclaration errors
- Codegen: Lowercase resource names are now capitalized to ensure variables are exported (e.g., `myBucket` → `MyBucket`)
- Codegen: `Tag{}` type now correctly triggers intrinsics import (fixes undefined `Tag` errors)
- Codegen: `!Sub ${Resource.Attr}` patterns now generate `Resource.Attr` field access instead of undefined `ResourceAttr` variable
- Codegen: Nested property types inside `If{}` intrinsics now use correct type context (e.g., `Association_S3OutputLocation` instead of parent type)
- Codegen: `SubWithMap` Variables field now generates `Json{}` instead of incorrectly typed struct
- Codegen: Digit-prefixed variable names now use `N` prefix instead of `_` to keep variables exported (e.g., `2RouteTable` → `N2RouteTable`)
- Registry: `wafv2.WebACL_Rule.Statement` now correctly maps to `WebACL_Statement` instead of `RuleGroup_Statement`

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

[unreleased]: https://github.com/lex00/wetwire-aws-go/compare/v1.11.0...HEAD
[1.11.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.10.0...v1.11.0
[1.10.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.8.2...v1.9.0
[1.7.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.6.2...v1.7.0
[1.6.2]: https://github.com/lex00/wetwire-aws-go/compare/v1.6.1...v1.6.2
[1.6.1]: https://github.com/lex00/wetwire-aws-go/compare/v1.6.0...v1.6.1
[1.6.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.4.2...v1.6.0
[1.4.2]: https://github.com/lex00/wetwire-aws-go/compare/v1.4.1...v1.4.2
[1.2.3]: https://github.com/lex00/wetwire-aws-go/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/lex00/wetwire-aws-go/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/lex00/wetwire-aws-go/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/lex00/wetwire-aws-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v1.0.0
[0.4.1]: https://github.com/lex00/wetwire-aws-go/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v0.4.0
[0.1.0]: https://github.com/lex00/wetwire-aws-go/releases/tag/v0.1.0
