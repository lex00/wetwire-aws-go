# Lint Rules

wetwire-aws-go includes 18 lint rules to enforce best practices and idiomatic patterns for declarative CloudFormation infrastructure-as-code.

## Quick Start

```bash
# Lint your infrastructure code
wetwire-aws lint ./infra/...

# Lint with auto-fix (where supported)
wetwire-aws lint ./infra/... --fix

# Output in JSON format
wetwire-aws lint ./infra/... -f json
```

## Rule Index

| Rule | Description | Severity | Auto-fix |
|------|-------------|----------|----------|
| WAW001 | Use pseudo-parameter constants | warning | - |
| WAW002 | Use intrinsic types | warning | - |
| WAW003 | Detect duplicate resource names | error | - |
| WAW004 | Split large files | warning | - |
| WAW005 | Extract inline property types | warning | - |
| WAW006 | Use policy version constant | info | - |
| WAW007 | Use typed slices | warning | - |
| WAW008 | Use named var declarations (block style) | warning | - |
| WAW009 | Use typed structs (recursive) | warning | - |
| WAW010 | Flatten inline typed structs | warning | - |
| WAW011 | Validate enum values | error | - |
| WAW012 | Use typed enum constants | warning | - |
| WAW013 | Detect undefined references | warning | - |
| WAW014 | Detect unused intrinsics import | error | - |
| WAW015 | Avoid explicit Ref{} | warning | - |
| WAW016 | Avoid explicit GetAtt{} | warning | - |
| WAW017 | Avoid pointer assignments | error | - |
| WAW018 | Use Json{} type alias | warning | - |

## Rule Details

### WAW001: Use Pseudo-Parameter Constants

**Description:** Use pseudo-parameter constants instead of hardcoded strings.

**Severity:** warning

Detects hardcoded AWS pseudo-parameter strings like `"AWS::Region"` and suggests using the typed constants from the intrinsics package.

#### Bad

```go
var MyBucket = s3.Bucket{
    BucketName: Sub("my-bucket-${AWS::Region}"),  // Hardcoded string
}
```

#### Good

```go
var MyBucket = s3.Bucket{
    BucketName: Sub("my-bucket-${" + AWS_REGION + "}"),
}
```

**Supported pseudo-parameters:**
- `AWS::Region` → `AWS_REGION`
- `AWS::AccountId` → `AWS_ACCOUNT_ID`
- `AWS::StackName` → `AWS_STACK_NAME`
- `AWS::StackId` → `AWS_STACK_ID`
- `AWS::Partition` → `AWS_PARTITION`
- `AWS::URLSuffix` → `AWS_URL_SUFFIX`
- `AWS::NoValue` → `AWS_NO_VALUE`
- `AWS::NotificationARNs` → `AWS_NOTIFICATION_ARNS`

---

### WAW002: Use Intrinsic Types

**Description:** Use intrinsic types instead of raw `map[string]any`.

**Severity:** warning

Detects raw map literals with CloudFormation intrinsic function keys and suggests using the typed intrinsic structs.

#### Bad

```go
Environment: map[string]any{
    "Fn::Sub": "arn:aws:s3:::${BucketName}/*",
}
```

#### Good

```go
Environment: Sub("arn:aws:s3:::${BucketName}/*"),
```

**Detected intrinsic patterns:**
- `Ref` → `Ref{}`
- `Fn::Sub` → `Sub{}`
- `Fn::Join` → `Join{}`
- `Fn::GetAtt` → `GetAtt{}`
- `Fn::If` → `If{}`
- `Fn::Equals` → `Equals{}`
- And more...

---

### WAW003: Detect Duplicate Resource Names

**Description:** Detect duplicate resource variable names in a file.

**Severity:** error

CloudFormation logical IDs must be unique. This rule detects when multiple resource variables have the same name.

#### Bad

```go
var MyBucket = s3.Bucket{BucketName: "bucket-a"}
var MyBucket = s3.Bucket{BucketName: "bucket-b"}  // Duplicate!
```

#### Good

```go
var DataBucket = s3.Bucket{BucketName: "bucket-a"}
var LogsBucket = s3.Bucket{BucketName: "bucket-b"}
```

---

### WAW004: Split Large Files

**Description:** Split large files with too many resources.

**Severity:** warning

**Threshold:** 15 resources per file (default)

Files with many resources become hard to navigate. This rule suggests splitting by category.

#### Bad

```go
// infrastructure.go with 30+ resources
var VPC = ec2.VPC{...}
var Subnet1 = ec2.Subnet{...}
// ... many more resources
```

#### Good

```
infra/
├── network.go     # VPC, Subnets, Security Groups
├── compute.go     # EC2, Lambda, ECS
├── storage.go     # S3, EFS, RDS
└── security.go    # IAM Roles, Policies
```

---

### WAW005: Extract Inline Property Types

**Description:** Use struct types instead of inline `map[string]any` for property types.

**Severity:** warning

Complex configuration fields should use typed structs for IDE autocomplete and validation.

#### Bad

```go
var MyBucket = s3.Bucket{
    VersioningConfiguration: map[string]any{
        "Status": "Enabled",
    },
}
```

#### Good

```go
var MyBucketVersioning = s3.Bucket_VersioningConfiguration{
    Status: "Enabled",
}

var MyBucket = s3.Bucket{
    VersioningConfiguration: MyBucketVersioning,
}
```

---

### WAW006: Use Policy Version Constant

**Description:** Consider using a constant for IAM policy version.

**Severity:** info

Detects hardcoded policy version strings like `"2012-10-17"`.

#### Bad

```go
PolicyDocument: Json{
    "Version": "2012-10-17",
    "Statement": []any{...},
}
```

#### Good

```go
PolicyDocument: PolicyDocument{
    Statement: []PolicyStatement{...},
}
```

---

### WAW007: Use Typed Slices

**Description:** Use typed slices instead of `[]any{map[string]any{...}}`.

**Severity:** warning

Property arrays like `SecurityGroupIngress` should use typed slice elements.

#### Bad

```go
SecurityGroupIngress: []any{
    map[string]any{
        "IpProtocol": "tcp",
        "FromPort":   443,
        "ToPort":     443,
        "CidrIp":     "0.0.0.0/0",
    },
}
```

#### Good

```go
var HTTPSIngress = ec2.SecurityGroup_Ingress{
    IpProtocol: "tcp",
    FromPort:   443,
    ToPort:     443,
    CidrIp:     "0.0.0.0/0",
}

SecurityGroupIngress: []ec2.SecurityGroup_Ingress{HTTPSIngress}
```

---

### WAW008: Use Named Var Declarations (Block Style)

**Description:** Use named var declarations instead of inline struct literals.

**Severity:** warning

Enforces the block style pattern where each property type instance is a separate named variable.

#### Bad

```go
SecurityGroupIngress: []ec2.SecurityGroup_Ingress{
    {CidrIp: "0.0.0.0/0", ...},  // Inline struct literal
    {CidrIp: "10.0.0.0/8", ...},
}
```

#### Good

```go
var PublicIngress = ec2.SecurityGroup_Ingress{CidrIp: "0.0.0.0/0", ...}
var PrivateIngress = ec2.SecurityGroup_Ingress{CidrIp: "10.0.0.0/8", ...}

SecurityGroupIngress: []ec2.SecurityGroup_Ingress{PublicIngress, PrivateIngress}
```

---

### WAW009: Use Typed Structs (Recursive)

**Description:** Use typed structs instead of `map[string]any` in resource fields.

**Severity:** warning

Recursively detects `map[string]any` patterns at any nesting depth.

#### Bad

```go
DistributionConfig: map[string]any{
    "Origins": []any{
        map[string]any{
            "DomainName": "example.com",
            "CustomOriginConfig": map[string]any{...},  // Nested map
        },
    },
}
```

#### Good

```go
var MyOrigin = cloudfront.Distribution_Origin{
    DomainName:         "example.com",
    CustomOriginConfig: MyOriginConfig,
}

DistributionConfig: cloudfront.Distribution_DistributionConfig{
    Origins: []cloudfront.Distribution_Origin{MyOrigin},
}
```

---

### WAW010: Flatten Inline Typed Structs

**Description:** Flatten inline typed struct literals to named var declarations.

**Severity:** warning

Even typed property structs should be extracted to separate variables for readability.

#### Bad

```go
var MyBucket = s3.Bucket{
    BucketEncryption: s3.Bucket_BucketEncryption{  // Inline typed struct
        ServerSideEncryptionConfiguration: []s3.Bucket_ServerSideEncryptionRule{
            s3.Bucket_ServerSideEncryptionRule{...},  // Nested inline
        },
    },
}
```

#### Good

```go
var MyBucketSSERule = s3.Bucket_ServerSideEncryptionRule{...}
var MyBucketEncryption = s3.Bucket_BucketEncryption{
    ServerSideEncryptionConfiguration: []s3.Bucket_ServerSideEncryptionRule{MyBucketSSERule},
}

var MyBucket = s3.Bucket{
    BucketEncryption: MyBucketEncryption,
}
```

---

### WAW011: Validate Enum Values

**Description:** Validate enum property values against allowed values.

**Severity:** error

Catches invalid enum values before deployment.

#### Bad

```go
var MyFunction = lambda.Function{
    Runtime: "python3.99",  // Invalid runtime
}
```

#### Good

```go
var MyFunction = lambda.Function{
    Runtime: "python3.12",
}
```

**Validated enums:**
- Lambda: Runtime, PackageType, Architecture
- S3: StorageClass
- EC2: InstanceType
- RDS: Engine
- DynamoDB: BillingMode, StreamViewType, TableClass
- ECS: LaunchType, NetworkMode, SchedulingStrategy

---

### WAW012: Use Typed Enum Constants

**Description:** Use typed enum constants instead of raw strings.

**Severity:** warning

Suggests using constants from `cloudformation-schema-go/enums` for type safety.

#### Bad

```go
Runtime: "python3.12",
StorageClass: "STANDARD",
```

#### Good

```go
import "github.com/lex00/cloudformation-schema-go/enums"

Runtime: enums.LambdaRuntimePython312,
StorageClass: enums.S3StorageClassStandard,
```

---

### WAW013: Detect Undefined References

**Description:** Potential undefined reference (resource or parameter).

**Severity:** warning

Catches PascalCase identifiers that might be undefined resources or parameters.

#### Bad

```go
var MyFunction = lambda.Function{
    Role: UndefinedRole.Arn,  // Not defined in this package
}
```

#### Good

```go
var MyRole = iam.Role{...}

var MyFunction = lambda.Function{
    Role: MyRole.Arn,  // Defined above
}
```

---

### WAW014: Detect Unused Intrinsics Import

**Description:** Intrinsics package imported but not used.

**Severity:** error

Detects when the intrinsics package is dot-imported but no intrinsic types are used.

#### Bad

```go
import (
    . "github.com/lex00/wetwire-aws-go/intrinsics"  // Unused
)

var MyBucket = s3.Bucket{
    BucketName: "my-bucket",  // No intrinsics used
}
```

#### Good

```go
// Either remove the import or use intrinsics
import (
    . "github.com/lex00/wetwire-aws-go/intrinsics"
)

var MyBucket = s3.Bucket{
    BucketName: Sub("${AWS::StackName}-bucket"),  // Intrinsic used
}
```

---

### WAW015: Avoid Explicit Ref{}

**Description:** Avoid explicit `Ref{}` - use direct variable references or `Param()`.

**Severity:** warning

Direct variable references are cleaner and provide better type checking.

#### Bad

```go
Bucket: Ref{"MyBucket"},
VpcId: Ref{"VpcIdParam"},
```

#### Good

```go
// For resources - direct reference
Bucket: MyBucket,

// For parameters - use the Param type
var VpcIdParam = Parameter{Type: "AWS::EC2::VPC::Id"}
VpcId: VpcIdParam,
```

---

### WAW016: Avoid Explicit GetAtt{}

**Description:** Avoid explicit `GetAtt{}` - use `Resource.Attr` field access.

**Severity:** warning

Field access syntax is cleaner and provides IDE autocomplete.

#### Bad

```go
Role: GetAtt{"MyRole", "Arn"},
```

#### Good

```go
Role: MyRole.Arn,
```

---

### WAW017: Avoid Pointer Assignments

**Description:** Avoid pointer assignments (`&Type{}`) - use value types.

**Severity:** error

The AST-based value extraction expects struct literals, not pointers.

#### Bad

```go
var MyConfig = &s3.Bucket_VersioningConfiguration{
    Status: "Enabled",
}
```

#### Good

```go
var MyConfig = s3.Bucket_VersioningConfiguration{
    Status: "Enabled",
}
```

---

### WAW018: Use Json{} Type Alias

**Description:** Use `Json{}` instead of `map[string]any{}` for cleaner syntax.

**Severity:** warning

The `Json` type alias from intrinsics provides cleaner syntax for inline maps.

#### Bad

```go
Environment: lambda.Function_Environment{
    Variables: map[string]any{
        "TABLE_NAME": TableName,
        "REGION":     AWS_REGION,
    },
}
```

#### Good

```go
Environment: lambda.Function_Environment{
    Variables: Json{
        "TABLE_NAME": TableName,
        "REGION":     AWS_REGION,
    },
}
```

## Disabling Rules

Currently, individual rules cannot be disabled. To skip linting, simply don't run `wetwire-aws lint`.

## Contributing

To add new lint rules:

1. Add the rule struct in `internal/linter/rules.go` or `rules_extra.go`
2. Implement the `Rule` interface: `ID()`, `Description()`, `Check()`
3. Add to `AllRules()` function
4. Add tests in `rules_test.go`
5. Document in this file
