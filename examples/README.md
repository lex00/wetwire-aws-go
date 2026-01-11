# Examples

This directory contains CloudFormation and SAM templates imported from official AWS repositories. These templates are used for round-trip testing to verify import → build → compare workflows.

## Imported Templates

| Directory | Source Repository | License |
|-----------|-------------------|---------|
| `aws-cloudformation-templates/` | [aws-cloudformation/aws-cloudformation-templates](https://github.com/aws-cloudformation/aws-cloudformation-templates) | Apache 2.0 |
| `aws-sam-templates/` | [aws/aws-sam-cli-app-templates](https://github.com/aws/aws-sam-cli-app-templates) | Apache 2.0 |

**Import date:** 2026-01-08

## Purpose

These templates serve as reference implementations for:

1. **Round-trip testing** - Verify that templates can be imported to Go and built back to CloudFormation
2. **Import validation** - Test the `wetwire-aws import` command against real-world templates
3. **Pattern discovery** - Learn common CloudFormation patterns used in production

## Template Counts

| Directory | Count |
|-----------|-------|
| CloudFormation templates | 126 |
| SAM templates | 57 |
| **Total** | 183 |

## Running Tests

```bash
# Run round-trip tests on all templates
go test ./internal/importer/... -v

# Import a specific template
wetwire-aws import examples/aws-cloudformation-templates/VPC/vpc-single-instance-in-subnet.yaml
```

## Attribution

All templates in this directory are the property of Amazon Web Services and are used under the Apache 2.0 license. See the source repositories for original authorship and contribution history.

### Apache 2.0 License Notice

```
Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License").
You may not use this file except in compliance with the License.
A copy of the License is located at

    http://www.apache.org/licenses/LICENSE-2.0

or in the "license" file accompanying this file. This file is distributed
on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
express or implied. See the License for the specific language governing
permissions and limitations under the License.
```
