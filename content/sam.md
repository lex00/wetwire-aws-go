---
title: "SAM"
---

wetwire-aws fully supports AWS SAM (Serverless Application Model) resources, allowing you to define serverless applications using the same declarative Go syntax as CloudFormation resources.

## Overview

SAM is an extension of CloudFormation that simplifies building serverless applications. wetwire-aws supports all 9 SAM resource types:

| Resource | Description |
|----------|-------------|
| `serverless.Function` | Lambda functions with SAM-specific features |
| `serverless.Api` | API Gateway REST APIs |
| `serverless.HttpApi` | API Gateway HTTP APIs (v2) |
| `serverless.SimpleTable` | DynamoDB tables (simplified) |
| `serverless.LayerVersion` | Lambda layers |
| `serverless.StateMachine` | Step Functions state machines |
| `serverless.Application` | Nested SAM applications |
| `serverless.Connector` | Resource permission connectors |
| `serverless.GraphQLApi` | AppSync GraphQL APIs |

## Quick Start

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/serverless"
)

var HelloFunction = serverless.Function{
    Handler:    "bootstrap",
    Runtime:    "provided.al2",
    CodeUri:    "./hello/",
    MemorySize: 128,
    Timeout:    30,
}
```

Generate the template:

```bash
wetwire-aws build ./infra > template.yaml
```

The output automatically includes the SAM Transform header:

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Resources:
  HelloFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: bootstrap
      Runtime: provided.al2
      CodeUri: ./hello/
      MemorySize: 128
      Timeout: 30
```

---

## SAM Function

The `serverless.Function` resource is the most commonly used SAM resource. It extends Lambda with simplified event source configuration.

### Basic Function

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/serverless"
)

var MyFunction = serverless.Function{
    Handler:      "index.handler",
    Runtime:      "python3.12",
    CodeUri:      "./src/",
    MemorySize:   256,
    Timeout:      30,
    Description:  "My serverless function",
}
```

### Function with Environment Variables

```go
var ProcessorEnv = serverless.Function_Environment{
    Variables: map[string]any{
        "TABLE_NAME":  DataTable.TableName,
        "BUCKET_NAME": DataBucket.BucketName,
        "LOG_LEVEL":   "INFO",
    },
}

var ProcessorFunction = serverless.Function{
    Handler:     "bootstrap",
    Runtime:     "provided.al2",
    CodeUri:     "./processor/",
    Environment: ProcessorEnv,
}
```

### Function with VPC Configuration

```go
var VpcConfig = serverless.Function_VpcConfig{
    SecurityGroupIds: []any{"sg-12345678"},
    SubnetIds:        []any{"subnet-abc123", "subnet-def456"},
}

var VpcFunction = serverless.Function{
    Handler:   "bootstrap",
    Runtime:   "provided.al2",
    CodeUri:   "./handler/",
    VpcConfig: VpcConfig,
}
```

### Function with Inline Code

For simple functions, you can embed code directly:

```go
var DateTimeFunction = serverless.Function{
    Handler:    "index.handler",
    Runtime:    "python3.12",
    InlineCode: `
import json
from datetime import datetime

def handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'timestamp': datetime.now().isoformat()})
    }
`,
}
```

### Function Properties Reference

| Property | Type | Description |
|----------|------|-------------|
| `Handler` | string | Function entry point |
| `Runtime` | string | Runtime environment (e.g., `python3.12`, `provided.al2`) |
| `CodeUri` | string | Path to function code |
| `InlineCode` | string | Inline function code (alternative to CodeUri) |
| `FunctionName` | string | Function name (optional) |
| `Description` | string | Function description |
| `MemorySize` | int | Memory in MB (128-10240) |
| `Timeout` | int | Timeout in seconds (1-900) |
| `Environment` | any | Environment variables |
| `VpcConfig` | any | VPC configuration |
| `Role` | any | IAM role ARN (auto-created if omitted) |
| `Policies` | any | SAM policy templates or managed policy ARNs |
| `Events` | map[string]any | Event sources (API, S3, SQS, etc.) |
| `Layers` | []any | Lambda layer ARNs |
| `Architectures` | []any | CPU architecture (`x86_64` or `arm64`) |
| `Tracing` | any | X-Ray tracing mode |
| `Tags` | map[string]any | Resource tags |

---

## SAM Api

Create REST APIs with API Gateway:

```go
var MyApi = serverless.Api{
    StageName:   "prod",
    Description: "My REST API",
}
```

### API with CORS

```go
var CorsConfig = serverless.Api_CorsConfiguration{
    AllowOrigin:  "'*'",
    AllowMethods: "'GET,POST,PUT,DELETE'",
    AllowHeaders: "'Content-Type,Authorization'",
}

var MyApi = serverless.Api{
    StageName: "v1",
    Cors:      CorsConfig,
}
```

---

## SAM HttpApi

HTTP APIs are simpler and cheaper than REST APIs:

```go
var MyHttpApi = serverless.HttpApi{
    StageName: "v1",
}
```

---

## SAM SimpleTable

Simplified DynamoDB table definition:

```go
var PrimaryKey = serverless.SimpleTable_PrimaryKey{
    Name:  "id",
    Type_: "String",
}

var UsersTable = serverless.SimpleTable{
    TableName:  "users",
    PrimaryKey: PrimaryKey,
}
```

---

## SAM LayerVersion

Create Lambda layers:

```go
var CommonLayer = serverless.LayerVersion{
    LayerName:          "common-dependencies",
    ContentUri:         "./layers/common/",
    CompatibleRuntimes: []any{"python3.11", "python3.12"},
    Description:        "Common Python dependencies",
}

// Reference in a function
var MyFunction = serverless.Function{
    Handler: "index.handler",
    Runtime: "python3.12",
    CodeUri: "./src/",
    Layers:  []any{CommonLayer.Arn},
}
```

---

## SAM StateMachine

Define Step Functions state machines:

```go
var OrderWorkflow = serverless.StateMachine{
    Name: "order-processing",
    Type_: "STANDARD",
    DefinitionUri: "./statemachine/order.asl.json",
}
```

---

## SAM Application

Deploy nested SAM applications:

```go
var SharedInfra = serverless.Application{
    Location: "arn:aws:serverlessrepo:us-east-1:123456789:applications/SharedInfra",
}
```

---

## SAM Connector

Simplify resource permissions:

```go
var FunctionToTableConnector = serverless.Connector{
    Source: map[string]any{
        "Id": MyFunction,
    },
    Destination: map[string]any{
        "Id": DataTable,
    },
    Permissions: []any{"Read", "Write"},
}
```

---

## Mixing SAM and CloudFormation

You can freely mix SAM resources with standard CloudFormation resources:

```go
package infra

import (
    "github.com/lex00/wetwire-aws-go/resources/serverless"
    "github.com/lex00/wetwire-aws-go/resources/s3"
    "github.com/lex00/wetwire-aws-go/resources/dynamodb"
)

// Standard CloudFormation S3 bucket
var DataBucket = s3.Bucket{
    BucketName: "my-data-bucket",
}

// Standard CloudFormation DynamoDB table
var DataTable = dynamodb.Table{
    TableName: "my-data-table",
    // ... full DynamoDB configuration
}

// SAM Function environment referencing CloudFormation resources
var ProcessorEnv = serverless.Function_Environment{
    Variables: map[string]any{
        "BUCKET_NAME": DataBucket.BucketName,
        "TABLE_NAME":  DataTable.TableName,
    },
}

// SAM Function referencing CloudFormation resources
var ProcessorFunction = serverless.Function{
    Handler:     "bootstrap",
    Runtime:     "provided.al2",
    CodeUri:     "./processor/",
    Environment: ProcessorEnv,
}
```

The generated template will include the SAM Transform header because SAM resources are detected:

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Resources:
  DataBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-data-bucket
  DataTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: my-data-table
  ProcessorFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: bootstrap
      Runtime: provided.al2
      CodeUri: ./processor/
      Environment:
        Variables:
          BUCKET_NAME: !Ref DataBucket
          TABLE_NAME: !Ref DataTable
```

---

## Importing SAM Templates

You can import existing SAM templates into Go code:

```bash
wetwire-aws import template.yaml -o ./infra
```

This generates Go code using the `serverless` package for SAM resources.

---

## Deploying SAM Applications

SAM templates require the SAM CLI or CloudFormation with package/deploy:

### Using SAM CLI

```bash
# Build (packages Lambda code)
sam build

# Deploy
sam deploy --guided
```

### Using AWS CLI

```bash
# Package (upload Lambda code to S3)
aws cloudformation package \
  --template-file template.yaml \
  --s3-bucket my-deployment-bucket \
  --output-template-file packaged.yaml

# Deploy
aws cloudformation deploy \
  --template-file packaged.yaml \
  --stack-name my-app \
  --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND
```

---

## Testing SAM Templates

wetwire-aws includes a script to test against official AWS SAM template repositories:

```bash
./scripts/import_sam_samples.sh
```

This clones and tests templates from:
- [aws-sam-cli-app-templates](https://github.com/aws/aws-sam-cli-app-templates)
- [sessions-with-aws-sam](https://github.com/aws-samples/sessions-with-aws-sam)
- [sam-python-crud-sample](https://github.com/aws-samples/sam-python-crud-sample)

---

## See Also

- [Quick Start](QUICK_START.md) - Getting started with wetwire-aws
- [CLI Reference](CLI.md) - Command reference
- [AWS SAM Documentation](https://docs.aws.amazon.com/serverless-application-model/)
