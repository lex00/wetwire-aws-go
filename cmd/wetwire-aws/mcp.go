// Command mcp runs an MCP server that exposes wetwire-aws tools.
//
// This server implements the Model Context Protocol (MCP) using infrastructure
// from github.com/lex00/wetwire-core-go/domain and automatically generates tools
// from the domain.Domain interface implementation.
//
// Tools are automatically registered based on the domain interface:
//   - wetwire_init: Initialize a new wetwire-aws project
//   - wetwire_build: Generate CloudFormation template from Go packages
//   - wetwire_lint: Lint Go packages for wetwire-aws issues
//   - wetwire_validate: Validate generated CloudFormation templates
//   - wetwire_import: Import existing CloudFormation templates to Go code
//   - wetwire_list: List discovered resources
//   - wetwire_graph: Visualize resource dependencies
//
// Usage:
//
//	wetwire-aws mcp  # Runs on stdio transport
package main

import (
	"context"

	"github.com/lex00/wetwire-aws-go/domain"
	coredomain "github.com/lex00/wetwire-core-go/domain"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP server for wetwire-aws tools",
		Long: `Run an MCP (Model Context Protocol) server that exposes wetwire-aws tools.

This command starts an MCP server on stdio transport, automatically providing tools
generated from the domain interface for:
- Initializing projects (wetwire_init)
- Building CloudFormation templates (wetwire_build)
- Linting code (wetwire_lint)
- Validating templates (wetwire_validate)
- Importing templates (wetwire_import)
- Listing resources (wetwire_list)
- Generating dependency graphs (wetwire_graph)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer()
		},
	}
}

func runMCPServer() error {
	// Create domain instance
	awsDomain := &domain.AwsDomain{}

	// Build MCP server from domain using auto-generation
	server := coredomain.BuildMCPServer(awsDomain)

	// Start the server on stdio transport
	return server.Start(context.Background())
}
