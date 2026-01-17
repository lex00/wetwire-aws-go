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
//	wetwire-aws design "prompt"       AI-assisted infrastructure design
//	wetwire-aws test "prompt"         Run persona-based testing
//	wetwire-aws diff old.json new.json Compare two templates
//	wetwire-aws watch ./infra/...     Auto-rebuild on file changes
//	wetwire-aws optimize ./infra/...  Suggest CloudFormation optimizations
//	wetwire-aws mcp                   Run MCP server
//	wetwire-aws version               Show version
package main

import (
	"fmt"
	"os"

	"github.com/lex00/wetwire-aws-go/domain"
)

// Version information set via ldflags
var version = "dev"

func main() {
	// Set domain version from ldflags
	domain.Version = version

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Create the domain instance and get root command with standard tools
	d := &domain.AwsDomain{}
	root := domain.CreateRootCommand(d)

	// Add AWS-specific commands
	root.AddCommand(newDesignCmd())
	root.AddCommand(newTestCmd())
	root.AddCommand(newOptimizeCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newWatchCmd())
	root.AddCommand(newMCPCmd())

	return root.Execute()
}
