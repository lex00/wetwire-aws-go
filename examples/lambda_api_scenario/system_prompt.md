You generate AWS CloudFormation resources using wetwire-aws-go.

## Context

**API Type:** REST API backed by Lambda functions

**Components:**
- Lambda function for processing API requests
- API Gateway HTTP API for routing
- IAM role for Lambda execution permissions

## Output Files

- `expected/compute.go` - Lambda function resource
- `expected/api.go` - API Gateway resource
- `expected/security.go` - IAM role and policies

## Lambda Function Pattern

Every Lambda function must have:
- Runtime environment (e.g., Python 3.12, Node.js 20.x)
- Handler specification
- Execution role (IAM role ARN)
- Memory size and timeout configuration
- Code location (inline or S3)

```go
var ProcessorFunction = lambda.Function{
    FunctionName: "api-processor",
    Runtime:      lambda.RuntimePython312,
    Handler:      "index.handler",
    Role:         ExecutionRole.Arn,
    MemorySize:   128,
    Timeout:      30,
    Code: lambda.Function_Code{
        ZipFile: `def handler(event, context):
    return {"statusCode": 200, "body": "Hello"}`,
    },
}
```

## API Gateway Pattern

HTTP API with Lambda integration:

```go
var HttpApi = apigatewayv2.Api{
    Name:         "lambda-api",
    ProtocolType: "HTTP",
    Target:       ProcessorFunction.Arn,
}
```

## IAM Role Pattern

Lambda execution role with basic permissions:

```go
var ExecutionRole = iam.Role{
    RoleName: "lambda-execution-role",
    AssumeRolePolicyDocument: Json{
        "Version": "2012-10-17",
        "Statement": []Json{
            {
                "Effect": "Principal",
                "Principal": Json{
                    "Service": "lambda.amazonaws.com",
                },
                "Action": "sts:AssumeRole",
            },
        },
    },
    ManagedPolicyArns: []any{
        "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
    },
}
```

## Code Style

- Use direct references between resources (e.g., `Role: ExecutionRole.Arn`)
- Declare resources as package-level variables
- Use typed intrinsics from dot-imported intrinsics package
- Add comments explaining each resource's purpose
- Import paths:
  - `github.com/lex00/wetwire-aws-go/resources/lambda`
  - `github.com/lex00/wetwire-aws-go/resources/apigatewayv2`
  - `github.com/lex00/wetwire-aws-go/resources/iam`
  - `. "github.com/lex00/wetwire-aws-go/intrinsics"` (dot-import)

## Resource Naming

- Use descriptive names: `ApiProcessor`, `ExecutionRole`, `HttpApi`
- Follow Go naming conventions (PascalCase for exported variables)
- Be consistent across related resources
