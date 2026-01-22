# Simple REST API with Lambda and API Gateway

This CloudFormation template creates a complete REST API using AWS Lambda and API Gateway.

## What's Included

### Resources

1. **Lambda Function** (`ApiProcessorFunction`)
   - Runtime: Python 3.12
   - Handles all API requests
   - Returns JSON responses with request details

2. **API Gateway HTTP API** (`HttpApi`)
   - Modern HTTP API (faster and cheaper than REST API)
   - Automatically routes all requests to Lambda

3. **IAM Role** (`LambdaExecutionRole`)
   - Allows Lambda to write logs to CloudWatch
   - Follows AWS best practices for least privilege

4. **Lambda Permission** (`ApiGatewayInvokePermission`)
   - Allows API Gateway to invoke the Lambda function
   - Critical for the integration to work

## Deployment

### Using AWS CLI

```bash
# Deploy the stack
aws cloudformation create-stack \
  --stack-name my-lambda-api \
  --template-body file://template.yaml \
  --capabilities CAPABILITY_NAMED_IAM

# Wait for completion
aws cloudformation wait stack-create-complete \
  --stack-name my-lambda-api

# Get the API endpoint
aws cloudformation describe-stacks \
  --stack-name my-lambda-api \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiEndpoint`].OutputValue' \
  --output text
```

### Using AWS Console

1. Go to CloudFormation in AWS Console
2. Click "Create stack" → "With new resources"
3. Upload `template.yaml`
4. Enter stack name: `my-lambda-api`
5. Click through and acknowledge IAM resource creation
6. Wait for stack to complete
7. Check Outputs tab for API endpoint

## Testing the API

Once deployed, you'll get an API endpoint URL like:
```
https://abc123xyz.execute-api.us-east-1.amazonaws.com
```

### Test with curl

```bash
# Root path
curl https://YOUR-API-ENDPOINT.execute-api.us-east-1.amazonaws.com/

# With a custom path
curl https://YOUR-API-ENDPOINT.execute-api.us-east-1.amazonaws.com/hello/world

# POST request
curl -X POST https://YOUR-API-ENDPOINT.execute-api.us-east-1.amazonaws.com/api/data
```

### Example Response

```json
{
  "message": "Hello from Lambda!",
  "method": "GET",
  "path": "/hello/world",
  "timestamp": "a1b2c3d4-e5f6-g7h8-i9j0-k1l2m3n4o5p6"
}
```

## How It Works

1. **Request Flow:**
   - User makes HTTP request → API Gateway
   - API Gateway invokes Lambda function
   - Lambda processes request and returns response
   - API Gateway returns response to user

2. **Permissions:**
   - Lambda needs permission to write logs (via IAM role)
   - API Gateway needs permission to invoke Lambda (via Lambda permission)

3. **Routing:**
   - `ANY /{proxy+}` catches all paths and methods
   - `ANY /` handles root path specifically
   - Lambda receives full request details in the event object

## Customizing the Lambda Function

The Lambda function code is embedded in the template. To modify it:

1. Edit the `Code.ZipFile` section in `ApiProcessorFunction`
2. Update the stack: `aws cloudformation update-stack ...`

For more complex functions:
- Upload code to S3 and use `Code.S3Bucket` and `Code.S3Key`
- Or use AWS SAM for better development experience

## Cost Estimation

This setup falls within AWS Free Tier:
- Lambda: 1M requests/month free
- API Gateway: 1M requests/month free (first 12 months)
- CloudWatch Logs: 5GB ingestion free

Beyond free tier:
- Lambda: ~$0.20 per 1M requests
- API Gateway HTTP API: ~$1.00 per 1M requests
- Very cost-effective for small to medium workloads

## Clean Up

To delete all resources:

```bash
aws cloudformation delete-stack --stack-name my-lambda-api
```

## Next Steps

- Add more complex logic to the Lambda function
- Implement different routes for different operations
- Add DynamoDB for data storage
- Set up custom domain name
- Add API authentication (Lambda authorizer or Cognito)
- Enable CORS for web applications

## Troubleshooting

**502 Bad Gateway:**
- Check Lambda execution role permissions
- Check Lambda function logs in CloudWatch

**403 Forbidden:**
- Verify Lambda permission for API Gateway is created
- Check the SourceArn matches your API

**Function errors:**
- Check CloudWatch Logs: `/aws/lambda/api-processor`
- Verify function code syntax
