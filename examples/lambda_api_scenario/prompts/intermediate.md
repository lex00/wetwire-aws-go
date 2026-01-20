Create a CloudFormation template for a Lambda-backed HTTP API:

**Lambda Function:**
- Python 3.12 runtime
- Simple handler that returns JSON response
- 128 MB memory, 30 second timeout
- Inline code deployment

**API Gateway:**
- HTTP API (v2)
- Direct Lambda integration
- Public endpoint

**IAM:**
- Lambda execution role with basic CloudWatch logging permissions

Generate a single CloudFormation YAML template with all resources.
