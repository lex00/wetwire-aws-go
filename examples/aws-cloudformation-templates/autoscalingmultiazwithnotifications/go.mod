module autoscalingmultiazwithnotifications

go 1.23

require (
	github.com/lex00/cloudformation-schema-go v0.7.0
	github.com/lex00/wetwire/go/wetwire-aws v0.1.0
)

// For local development, uncomment and adjust the path:
replace github.com/lex00/wetwire/go/wetwire-aws => ../../..
