module cognito_stack

go 1.23

require github.com/lex00/wetwire/go/wetwire-aws v0.1.0

require github.com/lex00/cloudformation-schema-go v0.7.0 // indirect

// For local development, uncomment and adjust the path:
replace github.com/lex00/wetwire/go/wetwire-aws => ../../..
