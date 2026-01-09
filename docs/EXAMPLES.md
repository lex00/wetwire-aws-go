# Examples Reference

The `examples/` directory contains imported AWS templates converted to wetwire-aws Go code. These serve as:

1. **Test artifacts** - Validate the import workflow
2. **Reference implementations** - Real-world CloudFormation patterns
3. **Learning resources** - See how complex templates translate to Go

## Directory Structure

```
examples/
├── aws-cloudformation-templates/   # 126 imported templates
│   ├── amazon_linux/
│   │   ├── go.mod                  # Go module
│   │   └── infra.go                # Generated resources
│   └── ...
│
└── aws-sam-templates/              # 56 imported SAM templates
    ├── sam_python_crud_sample_template_yaml/
    └── ...
```

## CloudFormation Templates (126)

Imported from [aws-cloudformation-templates](https://github.com/awslabs/aws-cloudformation-templates).

### By AWS Service

| Category | Count | Examples |
|----------|-------|----------|
| EC2/Compute | ~15 | `amazon_linux`, `centos`, `ec2_*`, `autoscaling*` |
| VPC/Network | ~17 | `vpc`, `public_vpc`, `private_vpc`, `vpcflowlogs*` |
| CloudFront | ~5 | `cloudfront`, `cloudfront_nocache`, `cloudfront_*` |
| CloudWatch | ~5 | `cloudwatch_dashboard_*`, `cloudwatch_*` |
| Lambda | ~4 | `lambda_*`, `apigateway_lambda*` |
| RDS/Database | ~5 | `rds_*`, `dynamodb_*` |
| S3 | ~5 | `compliant_bucket`, `static_website`, `s3_*` |
| Other | ~70 | Various services |

### Notable Examples

| Example | Description |
|---------|-------------|
| `amazon_linux` | Basic EC2 instance with security group |
| `autoscalingmultiazwithnotifications` | Auto Scaling across AZs with SNS |
| `vpc_single_instance_in_subnet` | VPC with EC2 in public subnet |
| `ecsclusterinvpc` | ECS cluster with VPC networking |
| `rds_aurora_mysql` | Aurora MySQL cluster setup |
| `cloudfront_s3_origin` | CloudFront with S3 origin |

## SAM Templates (56)

Imported from [aws-sam-cli-app-templates](https://github.com/aws/aws-sam-cli-app-templates) and [sessions-with-aws-sam](https://github.com/aws-samples/sessions-with-aws-sam).

### By Pattern

| Pattern | Count | Examples |
|---------|-------|----------|
| Step Functions | ~14 | `*_step_func_*` |
| S3 Integration | ~6 | `*_s3_*` |
| DynamoDB | ~4 | `*_ddb_*`, `*_hello_ddb_*` |
| Observability | ~3 | `*_enhanced_observability_*` |
| CRUD | 1 | `sam_python_crud_sample_*` |
| Other | ~28 | Various patterns |

### Notable Examples

| Example | Description |
|---------|-------------|
| `sam_python_crud_sample_template_yaml` | Complete CRUD API with DynamoDB |
| `sessions_with_aws_sam_api_enhanced_observability_*` | API with X-Ray tracing |
| `aws_sam_cli_app_templates_*_step_func_etl_*` | ETL pipeline with Step Functions |

## Package Structure

Each imported template becomes a Go package:

```
example_name/
├── go.mod      # Go module file
└── infra.go    # All resources in single file
```

## Using Examples

### View the generated code

```bash
cat examples/aws-cloudformation-templates/amazon_linux/infra.go
```

### Build a template

```bash
cd examples/aws-cloudformation-templates/amazon_linux
wetwire-aws build .
```

### List resources

```bash
cd examples/aws-cloudformation-templates/amazon_linux
wetwire-aws list .
```

### Copy as starting point

```bash
cp -r examples/aws-cloudformation-templates/vpc_single_instance_in_subnet ./my-vpc
cd my-vpc
# Edit infra.go
wetwire-aws build .
```

## Import Success Rates

| Repository | Success | Total |
|------------|---------|-------|
| aws-cloudformation-templates | 254/254 | 100% |
| aws-sam-templates | 48/57 | 84% |

See [IMPORT_WORKFLOW.md](IMPORT_WORKFLOW.md) for details on exclusions and the testing process.

## Refreshing Examples

To re-import all examples (after updating the importer):

```bash
# CloudFormation templates
./scripts/import_aws_samples.sh

# SAM templates
./scripts/import_sam_samples.sh
```

## Notes

- Examples are auto-generated and may need manual cleanup for production use
- Some examples use explicit `Ref{}` and `GetAtt{}` (flagged by linter) due to import limitations
- Run `wetwire-aws lint --fix ./...` to apply automatic style fixes

## See Also

- [IMPORT_WORKFLOW.md](IMPORT_WORKFLOW.md) - Import testing documentation
- [SAM.md](SAM.md) - SAM resource guide
- [Quick Start](QUICK_START.md) - Getting started
