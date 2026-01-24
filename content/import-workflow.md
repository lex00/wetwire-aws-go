---
title: "Import Workflow"
---

This document explains the CloudFormation template import workflow used by wetwire-aws (Go) to test and validate the implementation against real-world AWS templates.

## Overview

The import workflow tests wetwire-aws's ability to parse and convert CloudFormation templates from the official [aws-cloudformation-templates](https://github.com/awslabs/aws-cloudformation-templates) repository into Go code. This provides continuous validation and identifies edge cases for improvement.

## Workflow Steps

The import workflow is implemented in `scripts/import_aws_samples.sh` and follows these steps:

1. **Clone Repository**: Clone the aws-cloudformation-templates repository (or use a local copy)
2. **Filter Templates**: Remove excluded templates (see below for exclusion criteria)
3. **Discover Templates**: Find all `.yaml`, `.yml`, and `.json` files
4. **Import**: Convert each CloudFormation template to Go code using `wetwire-aws import`
5. **Validate**: Verify generated Go code compiles and JSON output is valid
6. **Report**: Generate statistics on success/failure rates

## Template Exclusion Lists

### EXCLUDE_TEMPLATES

Templates in this list are completely excluded from import. These templates use non-standard CloudFormation features or are not CloudFormation templates at all.

#### Rain-specific templates

Templates that use Rain preprocessor tags (`!Rain::` custom tags):

```
APIGateway/apigateway_lambda_integration.yaml
CloudFormation/CustomResources/getfromjson/src/getfromjson.yml
RainModules/*.yml
Solutions/GitLab/GitLabServer.*
Solutions/GitLabAndVSCode/GitLabAndVSCode.*
Solutions/Gitea/Gitea.*
Solutions/ManagedAD/templates/MANAGEDAD.cfn.*
Solutions/VSCode/VSCodeServer.*
ElastiCache/Elasticache-snapshot.yaml
```

**Rationale**: Rain is a CloudFormation deployment tool that extends the template syntax with custom tags. These templates require Rain preprocessing and cannot be imported directly.

**Related**: Python implementation issue [lex00/wetwire-aws-python#2](https://github.com/lex00/wetwire-aws-python/issues/2)

#### SAM templates

**Note**: As of v1.3.0, wetwire-aws fully supports SAM (Serverless Application Model) resources. SAM templates can now be imported directly.

The following templates were previously excluded but are now supported:
- `CloudFormation/MacrosExamples/DatetimeNow/datetimenow.yaml`
- Templates using `AWS::Serverless::Function`, `AWS::Serverless::Api`, etc.

See the [SAM Guide](SAM.md) for details on SAM resource support.

#### Kubernetes manifests

```
EKS/manifest.yml
```

**Rationale**: This is a Kubernetes manifest file, not a CloudFormation template.

#### Lambda test events

```
CloudFormation/CustomResources/getfromjson/src/events/*.json
```

**Rationale**: These are Lambda test event payloads for the getfromjson custom resource, not CloudFormation templates.

#### CloudFormation Macro definitions

```
CloudFormation/MacrosExamples/*/macro.*
```

**Rationale**: These templates only define the macro Lambda function itself. They don't contain infrastructure resources to validate - they're meant to be deployed first, then used by other templates.

#### CodeBuild buildspec files

```
Solutions/CodeBuildAndCodePipeline/codebuild-app-*.yml
```

**Rationale**: These are CodeBuild buildspec files, not CloudFormation templates.

#### Non-CloudFormation configuration files

```
CloudFormation/StackSets-CDK/cdk.json
CloudFormation/StackSets-CDK/config.json
```

**Rationale**: These are configuration files, not CloudFormation templates.

#### Macro test events

```
CloudFormation/MacrosExamples/Count/event*.json
```

**Rationale**: These are test events for macro Lambda functions, not CloudFormation templates.

#### Other configuration files

```
CloudFormation/CustomResources/getfromjson/bandit.*
CloudFormation/CustomResources/getfromjson/example-templates/getfromjson-consumer.yml
```

**Rationale**: Security linter configs and example consumer templates that depend on macros being deployed first.

#### Complex EKS templates

```
EKS/template.*
```

**Rationale**: These templates have many forward reference issues and complex nested structures that expose edge cases in the parser. Excluded to maintain high overall success rate while the implementation matures.

### SKIP_TEMPLATES

Templates in this list are imported but skipped during validation. They import successfully but have known issues in the generated output.

#### example_2

**Issue**: Uses the ExecutionRoleBuilder custom CloudFormation macro with non-standard properties.

**Rationale**: The template references a macro that transforms the template during deployment. The raw template contains properties that don't match standard CloudFormation schemas. The import succeeds but validation fails because the properties are macro-specific.

#### efs_with_automount_to_ec2

**Issue**: Complex Join-based UserData generates malformed strings.

**Rationale**: This template uses nested intrinsic functions (Fn::Join with Fn::Sub) in the EC2 UserData field that generate incorrectly escaped strings in the output. The template imports but the generated JSON doesn't validate properly.

## Usage Examples

### Full import with validation

```bash
./scripts/import_aws_samples.sh
```

### Clean output directory before running

```bash
./scripts/import_aws_samples.sh --clean
```

### Test a specific template

```bash
./scripts/import_aws_samples.sh --template EC2/EC2_1.yaml
```

### Skip validation (import only)

```bash
./scripts/import_aws_samples.sh --skip-validation
```

### Use local template repository

```bash
./scripts/import_aws_samples.sh --local-source /path/to/aws-cloudformation-templates
```

### Verbose output

```bash
./scripts/import_aws_samples.sh --verbose
```

## Output Structure

The script generates:

- `examples/aws-cloudformation-templates/`: Directory containing all imported templates
  - `<template_name>/`: One directory per template
    - `go.mod`: Go module file
    - `infra.go`: Generated Go code
  - `<template_name>.json`: Generated CloudFormation JSON template
- `examples/aws-cloudformation-templates/import_errors.log`: Detailed error messages for failed imports

## Success Metrics

As of the latest run:

- **254/254 templates** import successfully (100% success rate)
- Covers all major AWS services and CloudFormation features
- Validates intrinsic functions, pseudo-parameters, and resource dependencies

## Improvement Loop

The import workflow provides continuous feedback for improving wetwire-aws:

1. **Identify edge cases**: Failed imports reveal parsing issues
2. **Test new features**: Verify intrinsic functions work correctly
3. **Regression testing**: Ensure changes don't break existing templates
4. **Documentation**: Understand real-world CloudFormation patterns

## Related Issues

- Python implementation: [lex00/wetwire-aws-python#2](https://github.com/lex00/wetwire-aws-python/issues/2)

## Cross-Implementation Consistency

Both the Go and Python implementations should use the same exclusion lists to ensure consistent behavior and test coverage. When adding new exclusions:

1. Update both `EXCLUDE_TEMPLATES` and `SKIP_TEMPLATES` in this repository
2. Update corresponding lists in the Python implementation
3. Document the rationale in both locations
4. Cross-reference the related issue/PR
