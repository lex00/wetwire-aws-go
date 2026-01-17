package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCode_WithIntrinsics(t *testing.T) {
	content := []byte(`
Resources:
  MyVPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: "10.0.0.0/16"

  MySubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref MyVPC
      CidrBlock: "10.0.1.0/24"
      AvailabilityZone: !Select [0, !GetAZs ""]
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "vpc")
	// VPC and Subnet go to network.go (EC2 network resources)
	code := files["network.go"]

	// Check imports - uses dot import for intrinsics
	assert.Contains(t, code, `"github.com/lex00/wetwire-aws-go/resources/ec2"`)
	assert.Contains(t, code, `. "github.com/lex00/wetwire-aws-go/intrinsics"`)

	// Check VPC
	assert.Contains(t, code, "var MyVPC = ec2.VPC{")

	// Check Subnet with Ref - bare resource name pattern
	assert.Contains(t, code, "var MySubnet = ec2.Subnet{")
	// VpcId should reference MyVPC directly (no-parens pattern)
	assert.Contains(t, code, "VpcId: MyVPC,")
}

func TestGenerateCode_WithPseudoParameters(t *testing.T) {
	content := []byte(`
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub "${AWS::StackName}-bucket-${AWS::Region}"
`)

	ir, err := ParseTemplateContent(content, "test.yaml")
	require.NoError(t, err)

	files := GenerateCode(ir, "stack")
	// S3 bucket goes to storage.go
	code := files["storage.go"]

	// Check intrinsics import (dot import)
	assert.Contains(t, code, `. "github.com/lex00/wetwire-aws-go/intrinsics"`)

	// Check Sub usage with keyed syntax (no intrinsics. prefix with dot import)
	assert.Contains(t, code, `Sub{String: "${AWS::StackName}-bucket-${AWS::Region}"}`)
}

func TestMapToIntrinsic_Ref(t *testing.T) {
	m := map[string]any{"Ref": "MyResource"}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicRef, result.Type)
	assert.Equal(t, "MyResource", result.Args)
}

func TestMapToIntrinsic_GetAtt(t *testing.T) {
	m := map[string]any{"Fn::GetAtt": []string{"MyResource", "Arn"}}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicGetAtt, result.Type)
}

func TestMapToIntrinsic_Sub(t *testing.T) {
	m := map[string]any{"Fn::Sub": "${AWS::StackName}-bucket"}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicSub, result.Type)
}

func TestMapToIntrinsic_Join(t *testing.T) {
	m := map[string]any{"Fn::Join": []any{"-", []any{"a", "b", "c"}}}
	result := mapToIntrinsic(m)

	require.NotNil(t, result)
	assert.Equal(t, IntrinsicJoin, result.Type)
}

func TestMapToIntrinsic_AllTypes(t *testing.T) {
	tests := []struct {
		key      string
		expected IntrinsicType
	}{
		{"Ref", IntrinsicRef},
		{"Fn::GetAtt", IntrinsicGetAtt},
		{"Fn::Sub", IntrinsicSub},
		{"Fn::Join", IntrinsicJoin},
		{"Fn::Select", IntrinsicSelect},
		{"Fn::GetAZs", IntrinsicGetAZs},
		{"Fn::If", IntrinsicIf},
		{"Fn::Equals", IntrinsicEquals},
		{"Fn::And", IntrinsicAnd},
		{"Fn::Or", IntrinsicOr},
		{"Fn::Not", IntrinsicNot},
		{"Fn::Base64", IntrinsicBase64},
		{"Fn::FindInMap", IntrinsicFindInMap},
		{"Fn::Cidr", IntrinsicCidr},
		{"Fn::ImportValue", IntrinsicImportValue},
		{"Fn::Split", IntrinsicSplit},
		{"Fn::Transform", IntrinsicTransform},
		{"Condition", IntrinsicCondition},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			m := map[string]any{tt.key: "value"}
			result := mapToIntrinsic(m)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result.Type)
		})
	}
}

func TestMapToIntrinsic_EmptyMap(t *testing.T) {
	m := map[string]any{}
	result := mapToIntrinsic(m)
	assert.Nil(t, result)
}

func TestMapToIntrinsic_MultipleKeys(t *testing.T) {
	m := map[string]any{"Ref": "A", "Fn::Sub": "B"}
	result := mapToIntrinsic(m)
	assert.Nil(t, result)
}

func TestMapToIntrinsic_UnknownKey(t *testing.T) {
	m := map[string]any{"Unknown::Function": "value"}
	result := mapToIntrinsic(m)
	assert.Nil(t, result)
}

func TestPseudoParameterToGo(t *testing.T) {
	ctx := &codegenContext{
		template: NewIRTemplate(),
		imports:  make(map[string]bool),
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"AWS::Region", "AWS_REGION"},
		{"AWS::AccountId", "AWS_ACCOUNT_ID"},
		{"AWS::StackName", "AWS_STACK_NAME"},
		{"AWS::StackId", "AWS_STACK_ID"},
		{"AWS::Partition", "AWS_PARTITION"},
		{"AWS::URLSuffix", "AWS_URL_SUFFIX"},
		{"AWS::NoValue", "AWS_NO_VALUE"},
		{"AWS::NotificationARNs", "AWS_NOTIFICATION_ARNS"},
		{"UnknownPseudo", "UnknownPseudo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pseudoParameterToGo(ctx, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
