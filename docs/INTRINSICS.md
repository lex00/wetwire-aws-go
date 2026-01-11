# Intrinsics Guide

wetwire-aws-go provides type-safe CloudFormation intrinsic functions through the `intrinsics` package. This guide covers all supported intrinsic types and their usage patterns.

## Quick Start

```go
import (
    . "github.com/lex00/wetwire-aws-go/intrinsics"
    "github.com/lex00/wetwire-aws-go/resources/s3"
)

var MyBucket = s3.Bucket{
    BucketName: Sub("${AWS::StackName}-data-bucket"),
    Tags: []Tag{
        {Key: "Environment", Value: "Production"},
    },
}
```

## Pseudo-Parameters

CloudFormation pseudo-parameters as type-safe constants:

| Constant | CloudFormation | Description |
|----------|----------------|-------------|
| `AWS_REGION` | `AWS::Region` | Deployment region |
| `AWS_ACCOUNT_ID` | `AWS::AccountId` | AWS account ID |
| `AWS_STACK_NAME` | `AWS::StackName` | Stack name |
| `AWS_STACK_ID` | `AWS::StackId` | Stack ID |
| `AWS_PARTITION` | `AWS::Partition` | AWS partition (aws, aws-cn, aws-us-gov) |
| `AWS_URL_SUFFIX` | `AWS::URLSuffix` | Domain suffix (amazonaws.com) |
| `AWS_NO_VALUE` | `AWS::NoValue` | Remove property from resource |
| `AWS_NOTIFICATION_ARNS` | `AWS::NotificationARNs` | Stack notification ARNs |

### Usage

```go
var MyBucket = s3.Bucket{
    BucketName: Sub("${" + AWS_STACK_NAME + "}-bucket-${" + AWS_REGION + "}"),
}
```

## Reference Functions

### Ref

Reference a resource or parameter by logical name:

```go
// Direct syntax (preferred for resources)
var MyFunction = lambda.Function{
    Role: MyRole.Arn,  // Automatic GetAtt to Arn
}

// For parameters
var VpcIdParam = Parameter{Type: "AWS::EC2::VPC::Id"}
VpcId: VpcIdParam,  // Automatic Ref
```

### GetAtt

Get a resource attribute. Use field access syntax:

```go
var MyFunction = lambda.Function{
    Role: MyRole.Arn,  // Equivalent to GetAtt{"MyRole", "Arn"}
}

// Multiple attributes
var MyEndpoint = Sub("https://${" + MyApi.ApiId + "}.execute-api." + AWS_REGION + ".amazonaws.com")
```

## String Functions

### Sub (Fn::Sub)

String substitution with variables:

```go
// Simple substitution
Sub("${AWS::StackName}-bucket")

// With embedded variable references
Sub("arn:aws:s3:::${DataBucket}/*")

// Complex multi-line
Sub(`#!/bin/bash
export BUCKET=${DataBucket}
export REGION=${AWS::Region}
./start.sh`)
```

### SubWithMap (Fn::Sub with mapping)

String substitution with explicit variable mapping:

```go
SubWithMap{
    String: "${Protocol}://${Domain}:${Port}",
    Variables: Json{
        "Protocol": "https",
        "Domain":   MyDomain,
        "Port":     "443",
    },
}
```

### Join (Fn::Join)

Join array elements with delimiter:

```go
// Join with comma
Join{Delimiter: ",", Values: []any{"a", "b", "c"}}
// → "a,b,c"

// Join ARNs
Join{
    Delimiter: "",
    Values: []any{
        "arn:aws:s3:::",
        MyBucket,
        "/*",
    },
}
```

### Split (Fn::Split)

Split string into array:

```go
Split{Delimiter: ",", String: "a,b,c"}
// → ["a", "b", "c"]
```

### Base64 (Fn::Base64)

Encode string as base64:

```go
Base64{Sub("#!/bin/bash\necho ${AWS::StackName}")}
```

## Selection Functions

### Select (Fn::Select)

Select element from array by index:

```go
Select{Index: 0, Values: GetAZs{AWS_REGION}}
// → First availability zone in region
```

### GetAZs (Fn::GetAZs)

Get availability zones for region:

```go
GetAZs{AWS_REGION}
// → ["us-east-1a", "us-east-1b", ...]

GetAZs{""}  // Current region
```

### FindInMap (Fn::FindInMap)

Look up value in mappings:

```go
FindInMap{
    MapName:   "RegionMap",
    TopKey:    AWS_REGION,
    SecondKey: "AMI",
}
```

## Condition Functions

### Equals (Fn::Equals)

Compare two values:

```go
Equals{ValueA: MyParam, ValueB: "production"}
```

### And (Fn::And)

Logical AND of conditions:

```go
And{Conditions: []any{
    Condition{"IsProd"},
    Condition{"IsUSEast"},
}}
```

### Or (Fn::Or)

Logical OR of conditions:

```go
Or{Conditions: []any{
    Condition{"IsProd"},
    Condition{"IsStaging"},
}}
```

### Not (Fn::Not)

Logical NOT of condition:

```go
Not{Condition: Condition{"IsDev"}}
```

### If (Fn::If)

Conditional value:

```go
If{
    Condition:  "IsProd",
    TrueValue:  "m5.xlarge",
    FalseValue: "t3.micro",
}
```

## Network Functions

### Cidr (Fn::Cidr)

Generate CIDR blocks:

```go
Cidr{
    IpBlock: MyVPC.CidrBlock,
    Count:   6,
    CidrBits: 8,
}
// → ["10.0.0.0/24", "10.0.1.0/24", ...]
```

## Stack Functions

### ImportValue (Fn::ImportValue)

Import value from another stack:

```go
ImportValue{Sub("${NetworkStack}-VpcId")}
```

### Transform (Fn::Transform)

Apply a macro transform:

```go
Transform{
    Name: "AWS::Include",
    Parameters: Json{
        "Location": "s3://mybucket/mysnippet.yaml",
    },
}
```

## Type Aliases

### Json

Type alias for `map[string]any`, providing cleaner syntax:

```go
// Instead of:
Variables: map[string]any{"KEY": value}

// Use:
Variables: Json{"KEY": value}
```

### Tag

CloudFormation resource tag:

```go
Tags: []Tag{
    {Key: "Environment", Value: "Production"},
    {Key: "Project", Value: Sub("${AWS::StackName}")},
}
```

## Parameters

### Parameter

Define CloudFormation template parameters:

```go
var Environment = Parameter{
    Type:        "String",
    Default:     "development",
    AllowedValues: []string{"development", "staging", "production"},
    Description: "Deployment environment",
}

// Use in resources
var MyBucket = s3.Bucket{
    BucketName: Sub("${AWS::StackName}-${Environment}-data"),
}
```

### Param Helper

Quick parameter reference:

```go
var VpcId = Param("VpcIdParam")  // Creates Ref{"VpcIdParam"}
```

## Outputs

### Output

Define stack outputs:

```go
var BucketNameOutput = Output{
    Description: "Name of the S3 bucket",
    Value:       MyBucket,
    Export: &Output_Export{
        Name: Sub("${AWS::StackName}-BucketName"),
    },
}
```

## IAM Policy Types

### PolicyDocument

Type-safe IAM policy documents:

```go
var LambdaPolicy = PolicyDocument{
    Statement: []PolicyStatement{
        {
            Effect:   "Allow",
            Action:   []string{"s3:GetObject", "s3:PutObject"},
            Resource: Sub("arn:aws:s3:::${DataBucket}/*"),
        },
    },
}
```

### PolicyStatement

Individual policy statement:

```go
PolicyStatement{
    Effect:   "Allow",
    Action:   []string{"logs:*"},
    Resource: "*",
}
```

### Principal Types

```go
// Service principal
ServicePrincipal("lambda.amazonaws.com")

// AWS account principal
AWSPrincipal("arn:aws:iam::123456789012:root")

// All principals (use with caution)
AllPrincipal()

// Federated principal
FederatedPrincipal("cognito-identity.amazonaws.com")
```

## Best Practices

1. **Use Dot Import** - Import intrinsics with `.` for cleaner syntax
2. **Prefer Direct References** - Use `MyRole.Arn` instead of `GetAtt{"MyRole", "Arn"}`
3. **Use Constants** - Use `AWS_REGION` instead of `"AWS::Region"`
4. **Use Json Type** - Use `Json{}` instead of `map[string]any{}`
5. **Extract Parameters** - Define parameters as top-level variables

## CloudFormation Mapping

| Go Type | CloudFormation |
|---------|----------------|
| `Ref{}` | `{"Ref": "..."}` |
| `GetAtt{}` | `{"Fn::GetAtt": [...]}` |
| `Sub{}` | `{"Fn::Sub": "..."}` |
| `Join{}` | `{"Fn::Join": [...]}` |
| `Select{}` | `{"Fn::Select": [...]}` |
| `If{}` | `{"Fn::If": [...]}` |
| `Equals{}` | `{"Fn::Equals": [...]}` |
| `Base64{}` | `{"Fn::Base64": "..."}` |
| `FindInMap{}` | `{"Fn::FindInMap": [...]}` |
| `Cidr{}` | `{"Fn::Cidr": [...]}` |
| `ImportValue{}` | `{"Fn::ImportValue": "..."}` |
