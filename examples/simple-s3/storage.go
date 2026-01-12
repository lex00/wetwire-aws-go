// Package simple_s3 demonstrates idiomatic wetwire patterns for S3.
//
// This example shows:
// - Flat block-style declarations (named top-level vars)
// - Everything wrapped (direct references, no Ref()/GetAtt() calls)
// - Extracted inline configs (no nested struct literals)
// - Clear comments
package simple_s3

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/s3"
)

// ----------------------------------------------------------------------------
// S3 Bucket Configuration
// ----------------------------------------------------------------------------

// BucketEncryptionDefault configures AES256 server-side encryption.
var BucketEncryptionDefault = s3.Bucket_ServerSideEncryptionByDefault{
	SSEAlgorithm: "AES256",
}

// BucketEncryptionRule wraps the default encryption configuration.
var BucketEncryptionRule = s3.Bucket_ServerSideEncryptionRule{
	ServerSideEncryptionByDefault: BucketEncryptionDefault,
}

// BucketEncryption defines the encryption configuration for the bucket.
var BucketEncryption = s3.Bucket_BucketEncryption{
	ServerSideEncryptionConfiguration: []any{BucketEncryptionRule},
}

// BucketVersioning enables versioning on the bucket.
var BucketVersioning = s3.Bucket_VersioningConfiguration{
	Status: "Enabled",
}

// BucketPublicAccessBlock blocks all public access to the bucket.
var BucketPublicAccessBlock = s3.Bucket_PublicAccessBlockConfiguration{
	BlockPublicAcls:       true,
	BlockPublicPolicy:     true,
	IgnorePublicAcls:      true,
	RestrictPublicBuckets: true,
}

// DataBucket is the main S3 bucket for storing data.
// It has encryption, versioning, and public access blocked.
var DataBucket = s3.Bucket{
	BucketName:                     Sub{String: "data-${AWS::AccountId}-${AWS::Region}"},
	BucketEncryption:               BucketEncryption,
	VersioningConfiguration:        BucketVersioning,
	PublicAccessBlockConfiguration: BucketPublicAccessBlock,
}

// ----------------------------------------------------------------------------
// S3 Bucket Policy
// ----------------------------------------------------------------------------

// DenyInsecureTransportStatement denies requests that don't use HTTPS.
var DenyInsecureTransportStatement = DenyStatement{
	Effect:    "Deny",
	Action:    "s3:*",
	Principal: AWSPrincipal{"*"},
	Resource: []any{
		DataBucket.Arn,
		Join{Delimiter: "", Values: []any{DataBucket.Arn, "/*"}},
	},
	Condition: Json{
		Bool: Json{"aws:SecureTransport": "false"},
	},
}

// DataBucketPolicyDocument defines the policy document for the bucket.
// It requires HTTPS for all requests.
var DataBucketPolicyDocument = PolicyDocument{
	Version:   "2012-10-17",
	Statement: []any{DenyInsecureTransportStatement},
}

// DataBucketPolicy attaches the policy to the DataBucket.
// Note: Direct reference to DataBucket - no Ref() needed!
var DataBucketPolicy = s3.BucketPolicy{
	Bucket:         DataBucket,
	PolicyDocument: DataBucketPolicyDocument,
}
