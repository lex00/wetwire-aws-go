Create Lambda API infrastructure. Required files:

**expected/compute.go:**
- Lambda function: Python 3.12, inline handler, 128MB/30s, reference IAM role

**expected/api.go:**
- ApiGatewayV2 HTTP API with Lambda target integration

**expected/security.go:**
- IAM role: lambda.amazonaws.com trust policy, AWSLambdaBasicExecutionRole managed policy

Follow wetwire-aws-go patterns: direct references, dot-import intrinsics, Json type for policy docs.
