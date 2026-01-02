package intrinsics

import (
	"github.com/lex00/cloudformation-schema-go/intrinsics"
)

// Pseudo-parameters are predefined by CloudFormation and available in every template.
// They return values specific to the current stack.
// Re-exported from shared package.
//
// Usage:
//
//	region := AWS_REGION           // {"Ref": "AWS::Region"}
//	arn := Sub{fmt.Sprintf("arn:aws:s3:::%s/*", AWS_ACCOUNT_ID)}
var (
	// AWS_ACCOUNT_ID returns the AWS account ID of the account in which the stack is created.
	AWS_ACCOUNT_ID = intrinsics.AWS_ACCOUNT_ID

	// AWS_NOTIFICATION_ARNS returns the list of notification ARNs for the current stack.
	AWS_NOTIFICATION_ARNS = intrinsics.AWS_NOTIFICATION_ARNS

	// AWS_NO_VALUE removes the resource property when used with Fn::If.
	AWS_NO_VALUE = intrinsics.AWS_NO_VALUE

	// AWS_PARTITION returns the partition the resource is in (aws, aws-cn, aws-us-gov).
	AWS_PARTITION = intrinsics.AWS_PARTITION

	// AWS_REGION returns the AWS Region in which the stack is created.
	AWS_REGION = intrinsics.AWS_REGION

	// AWS_STACK_ID returns the ID of the stack.
	AWS_STACK_ID = intrinsics.AWS_STACK_ID

	// AWS_STACK_NAME returns the name of the stack.
	AWS_STACK_NAME = intrinsics.AWS_STACK_NAME

	// AWS_URL_SUFFIX returns the suffix for a domain (usually amazonaws.com).
	AWS_URL_SUFFIX = intrinsics.AWS_URL_SUFFIX
)
