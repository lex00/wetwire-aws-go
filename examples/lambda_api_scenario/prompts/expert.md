Create CloudFormation template for Lambda API:

- Lambda: Python 3.12, inline handler, 128MB/30s
- API Gateway v2 HTTP API with Lambda integration
- IAM role: lambda.amazonaws.com trust, AWSLambdaBasicExecutionRole
- Lambda permission for API Gateway invocation
- Outputs: API endpoint URL, function ARN

Single YAML file. No documentation.
