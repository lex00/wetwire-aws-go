// Command wetwire-aws generates CloudFormation templates from Go resource declarations.
//
// Usage:
//
//	wetwire-aws build ./infra/...     Generate CloudFormation template
//	wetwire-aws lint ./infra/...      Check for issues
//	wetwire-aws validate ./infra/...  Validate resources and references
//	wetwire-aws list ./infra/...      List discovered resources
//	wetwire-aws graph ./infra/...     Generate DOT dependency graph
//	wetwire-aws init myproject        Create new project
//	wetwire-aws import template.yaml  Import CloudFormation template to Go
//	wetwire-aws version               Show version
package main

import (
	"fmt"
	"os"

	"github.com/lex00/wetwire-aws-go/domain"
)

func main() {
	if err := domain.Run(&domain.AwsDomain{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
