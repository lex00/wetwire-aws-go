// Package domain provides the Domain interface for automatic CLI generation.
package domain

import (
	"github.com/lex00/wetwire-core-go/domain"
	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time.
var Version = "dev"

// Re-export core domain types for convenience
type (
	// Domain is the core interface for wetwire domains.
	Domain = domain.Domain

	// ImporterDomain is an optional interface for domains that support importing.
	ImporterDomain = domain.ImporterDomain

	// ListerDomain is an optional interface for domains that support listing.
	ListerDomain = domain.ListerDomain

	// GrapherDomain is an optional interface for domains that support graphing.
	GrapherDomain = domain.GrapherDomain

	// Builder builds domain resources from source code.
	Builder = domain.Builder

	// Linter validates domain resources according to domain-specific rules.
	Linter = domain.Linter

	// Initializer creates new domain projects with example code.
	Initializer = domain.Initializer

	// Validator validates that generated output conforms to domain specifications.
	Validator = domain.Validator

	// Importer imports external resources or configurations into the domain.
	Importer = domain.Importer

	// Lister discovers and lists domain resources.
	Lister = domain.Lister

	// Grapher visualizes relationships between domain resources.
	Grapher = domain.Grapher

	// Context wraps context.Context with additional domain operation context.
	Context = domain.Context

	// Result represents the outcome of a domain operation.
	Result = domain.Result

	// Error represents a structured error with location and context information.
	Error = domain.Error

	// BuildOpts contains options for the Build operation.
	BuildOpts = domain.BuildOpts

	// LintOpts contains options for the Lint operation.
	LintOpts = domain.LintOpts

	// InitOpts contains options for the Init operation.
	InitOpts = domain.InitOpts

	// ValidateOpts contains options for the Validate operation.
	ValidateOpts = domain.ValidateOpts

	// ImportOpts contains options for the Import operation.
	ImportOpts = domain.ImportOpts

	// ListOpts contains options for the List operation.
	ListOpts = domain.ListOpts

	// GraphOpts contains options for the Graph operation.
	GraphOpts = domain.GraphOpts
)

// Re-export constructors
var (
	NewResult              = domain.NewResult
	NewResultWithData      = domain.NewResultWithData
	NewErrorResult         = domain.NewErrorResult
	NewErrorResultMultiple = domain.NewErrorResultMultiple
	NewContext             = domain.NewContext
	NewContextWithVerbose  = domain.NewContextWithVerbose
)

// CreateRootCommand creates a root command with all standard domain commands.
// This allows callers to add additional domain-specific commands before executing.
func CreateRootCommand(d Domain) *cobra.Command {
	return domain.Run(d)
}

// Run creates and executes a CLI for the given domain.
// It automatically generates commands based on the domain's interface implementations.
func Run(d Domain) error {
	root := CreateRootCommand(d)
	return root.Execute()
}
