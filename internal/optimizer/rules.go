package optimizer

import (
	"strings"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// s3BucketRules contains optimization rules for S3 buckets.
var s3BucketRules = []Rule{
	{
		ID:          "OPT-S3-001",
		Category:    "security",
		Title:       "S3 bucket should have encryption enabled",
		Description: "Server-side encryption protects data at rest",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			// This is a static analysis suggestion - actual property checking
			// would require runtime extraction
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "security",
				Severity:    "high",
				Title:       "Consider enabling S3 bucket encryption",
				Description: "S3 buckets should have server-side encryption enabled to protect data at rest.",
				Suggestion:  "Add BucketEncryption with SSE-S3 or SSE-KMS configuration.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-S3-002",
		Category:    "security",
		Title:       "S3 bucket should block public access",
		Description: "Public access can lead to data exposure",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "security",
				Severity:    "high",
				Title:       "Consider blocking public access",
				Description: "S3 buckets should have PublicAccessBlockConfiguration to prevent accidental public exposure.",
				Suggestion:  "Add PublicAccessBlockConfiguration with BlockPublicAcls, BlockPublicPolicy, IgnorePublicAcls, and RestrictPublicBuckets set to true.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-S3-003",
		Category:    "reliability",
		Title:       "S3 bucket should have versioning enabled",
		Description: "Versioning protects against accidental deletion",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "reliability",
				Severity:    "medium",
				Title:       "Consider enabling versioning",
				Description: "S3 bucket versioning protects against accidental deletion and allows recovery of previous versions.",
				Suggestion:  "Add VersioningConfiguration with Status set to 'Enabled'.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-S3-004",
		Category:    "cost",
		Title:       "S3 bucket should have lifecycle rules",
		Description: "Lifecycle rules help manage storage costs",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "cost",
				Severity:    "low",
				Title:       "Consider adding lifecycle rules",
				Description: "Lifecycle rules can automatically transition objects to cheaper storage classes or delete old objects.",
				Suggestion:  "Add LifecycleConfiguration with appropriate transition and expiration rules.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
}

// lambdaFunctionRules contains optimization rules for Lambda functions.
var lambdaFunctionRules = []Rule{
	{
		ID:          "OPT-LAM-001",
		Category:    "performance",
		Title:       "Lambda function memory should be optimized",
		Description: "Memory allocation affects both performance and cost",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "performance",
				Severity:    "medium",
				Title:       "Review Lambda memory configuration",
				Description: "Lambda memory allocation affects CPU allocation and execution speed. Consider using AWS Lambda Power Tuning to find the optimal memory setting.",
				Suggestion:  "Use AWS Lambda Power Tuning tool to find the optimal memory configuration for your workload.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-LAM-002",
		Category:    "reliability",
		Title:       "Lambda function should have error handling",
		Description: "Dead letter queues help handle failed invocations",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "reliability",
				Severity:    "medium",
				Title:       "Consider adding a dead letter queue",
				Description: "A dead letter queue (DLQ) captures failed async invocations for later analysis or retry.",
				Suggestion:  "Add DeadLetterConfig pointing to an SQS queue or SNS topic.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-LAM-003",
		Category:    "cost",
		Title:       "Lambda function timeout should be appropriate",
		Description: "Excessive timeouts can increase costs",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "cost",
				Severity:    "low",
				Title:       "Review Lambda timeout setting",
				Description: "Ensure the timeout is set appropriately for your function's expected execution time. Long timeouts with errors can be costly.",
				Suggestion:  "Set Timeout to match your function's expected execution time plus a reasonable buffer.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
}

// iamRules contains optimization rules for IAM resources.
var iamRules = []Rule{
	{
		ID:          "OPT-IAM-001",
		Category:    "security",
		Title:       "IAM role should use least privilege",
		Description: "Overly permissive policies increase security risk",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			if res.Type == "iam.Role" || res.Type == "iam.Policy" {
				return &wetwire.OptimizeSuggestion{
					Resource:    res.Name,
					Category:    "security",
					Severity:    "high",
					Title:       "Review IAM permissions for least privilege",
					Description: "Ensure IAM policies grant only the minimum permissions required. Avoid using * in Resource or Action.",
					Suggestion:  "Replace wildcard (*) permissions with specific resource ARNs and actions.",
					File:        res.File,
					Line:        res.Line,
				}
			}
			return nil
		},
	},
}

// ec2InstanceRules contains optimization rules for EC2 instances.
var ec2InstanceRules = []Rule{
	{
		ID:          "OPT-EC2-001",
		Category:    "cost",
		Title:       "EC2 instance type should be appropriate",
		Description: "Right-sizing instances reduces costs",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "cost",
				Severity:    "medium",
				Title:       "Review EC2 instance type sizing",
				Description: "Ensure the instance type matches your workload requirements. Consider using AWS Compute Optimizer for recommendations.",
				Suggestion:  "Use AWS Compute Optimizer to analyze utilization and get right-sizing recommendations.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-EC2-002",
		Category:    "reliability",
		Title:       "EC2 instance should use auto scaling",
		Description: "Auto scaling improves availability and cost efficiency",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "reliability",
				Severity:    "medium",
				Title:       "Consider using Auto Scaling",
				Description: "Individual EC2 instances are single points of failure. Auto Scaling groups provide high availability and automatic replacement of unhealthy instances.",
				Suggestion:  "Replace standalone EC2 instances with Auto Scaling groups for production workloads.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-EC2-003",
		Category:    "security",
		Title:       "EC2 instance should have IMDSv2 enabled",
		Description: "IMDSv2 provides better metadata security",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "security",
				Severity:    "high",
				Title:       "Enable IMDSv2 for EC2 instances",
				Description: "Instance Metadata Service Version 2 (IMDSv2) provides defense in depth against SSRF attacks.",
				Suggestion:  "Set MetadataOptions.HttpTokens to 'required' to enforce IMDSv2.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
}

// rdsInstanceRules contains optimization rules for RDS instances.
var rdsInstanceRules = []Rule{
	{
		ID:          "OPT-RDS-001",
		Category:    "reliability",
		Title:       "RDS instance should have Multi-AZ enabled",
		Description: "Multi-AZ provides high availability",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "reliability",
				Severity:    "high",
				Title:       "Consider enabling Multi-AZ deployment",
				Description: "Multi-AZ deployments provide automatic failover to a standby instance in another Availability Zone.",
				Suggestion:  "Set MultiAZ to true for production databases.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-RDS-002",
		Category:    "reliability",
		Title:       "RDS instance should have automated backups",
		Description: "Automated backups enable point-in-time recovery",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "reliability",
				Severity:    "high",
				Title:       "Enable automated backups",
				Description: "Automated backups with point-in-time recovery protect against data loss.",
				Suggestion:  "Set BackupRetentionPeriod to at least 7 days.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-RDS-003",
		Category:    "security",
		Title:       "RDS instance should have encryption enabled",
		Description: "Storage encryption protects data at rest",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "security",
				Severity:    "high",
				Title:       "Enable storage encryption",
				Description: "RDS storage encryption protects data at rest using AES-256 encryption.",
				Suggestion:  "Set StorageEncrypted to true.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
}

// dynamoDBTableRules contains optimization rules for DynamoDB tables.
var dynamoDBTableRules = []Rule{
	{
		ID:          "OPT-DDB-001",
		Category:    "cost",
		Title:       "DynamoDB table should use appropriate capacity mode",
		Description: "On-demand vs provisioned capacity affects costs",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "cost",
				Severity:    "medium",
				Title:       "Review DynamoDB capacity mode",
				Description: "On-demand mode is cost-effective for unpredictable workloads. Provisioned with auto-scaling is better for predictable traffic.",
				Suggestion:  "Use on-demand for variable workloads, or provisioned with auto-scaling for steady traffic patterns.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
	{
		ID:          "OPT-DDB-002",
		Category:    "reliability",
		Title:       "DynamoDB table should have point-in-time recovery",
		Description: "PITR enables recovery from accidental data loss",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			return &wetwire.OptimizeSuggestion{
				Resource:    res.Name,
				Category:    "reliability",
				Severity:    "medium",
				Title:       "Enable point-in-time recovery",
				Description: "Point-in-time recovery (PITR) provides continuous backups of your DynamoDB table data.",
				Suggestion:  "Set PointInTimeRecoverySpecification.PointInTimeRecoveryEnabled to true.",
				File:        res.File,
				Line:        res.Line,
			}
		},
	},
}

// genericRules apply to all resources.
var genericRules = []Rule{
	{
		ID:          "OPT-GEN-001",
		Category:    "reliability",
		Title:       "Resource should have DeletionPolicy",
		Description: "DeletionPolicy controls what happens when resource is deleted",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			// Only suggest for stateful resources
			statefulTypes := []string{
				"s3.Bucket", "rds.DBInstance", "dynamodb.Table",
				"ec2.Volume", "efs.FileSystem", "elasticache.CacheCluster",
			}
			for _, t := range statefulTypes {
				if strings.HasPrefix(res.Type, strings.Split(t, ".")[0]) {
					return &wetwire.OptimizeSuggestion{
						Resource:    res.Name,
						Category:    "reliability",
						Severity:    "low",
						Title:       "Consider adding DeletionPolicy",
						Description: "DeletionPolicy controls behavior when the resource is deleted from the stack. Use 'Retain' or 'Snapshot' for stateful resources.",
						Suggestion:  "Add DeletionPolicy: Retain or DeletionPolicy: Snapshot to protect data.",
						File:        res.File,
						Line:        res.Line,
					}
				}
			}
			return nil
		},
	},
	{
		ID:          "OPT-GEN-002",
		Category:    "reliability",
		Title:       "Resource should have UpdateReplacePolicy",
		Description: "UpdateReplacePolicy controls behavior during replacements",
		Check: func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion {
			// Only suggest for stateful resources
			statefulTypes := []string{
				"s3.Bucket", "rds.DBInstance", "dynamodb.Table",
			}
			for _, t := range statefulTypes {
				if strings.HasPrefix(res.Type, strings.Split(t, ".")[0]) {
					return &wetwire.OptimizeSuggestion{
						Resource:    res.Name,
						Category:    "reliability",
						Severity:    "low",
						Title:       "Consider adding UpdateReplacePolicy",
						Description: "UpdateReplacePolicy controls behavior when updates require resource replacement.",
						Suggestion:  "Add UpdateReplacePolicy: Retain or UpdateReplacePolicy: Snapshot.",
						File:        res.File,
						Line:        res.Line,
					}
				}
			}
			return nil
		},
	},
}
