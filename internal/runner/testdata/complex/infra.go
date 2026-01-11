package complex

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/s3"
)

var Environment = Parameter{
	Type:        "String",
	Default:     "dev",
	Description: "Deployment environment",
}

var DataBucket = s3.Bucket{
	BucketName: Sub{"${AWS::StackName}-data-${Environment}"},
}

var BucketNameOutput = Output{
	Description: "Name of the data bucket",
	Value:       DataBucket,
}

var RegionMapping = Mapping{
	"us-east-1": map[string]any{
		"AMI": "ami-12345678",
	},
	"us-west-2": map[string]any{
		"AMI": "ami-87654321",
	},
}
