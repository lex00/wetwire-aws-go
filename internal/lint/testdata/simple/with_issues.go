package simple

import "github.com/lex00/wetwire-aws-go/resources/s3"

// This file has lint issues for testing

var BadBucket = s3.Bucket{
	BucketName: "bucket-${AWS::Region}",
}

var region = "AWS::Region"
