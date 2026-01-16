// Package domain provides the Domain interface for automatic CLI generation.
package domain

import (
	"github.com/lex00/wetwire-core-go/cmd"
	"github.com/spf13/cobra"
)

// Domain represents a wetwire domain (e.g., "aws", "honeycomb") that provides
// infrastructure-as-code capabilities. Domains implement this interface to
// enable automatic CLI command generation.
type Domain interface {
	// Name returns the domain name (e.g., "aws", "honeycomb")
	Name() string

	// Version returns the current version of the domain
	Version() string

	// Builder returns the implementation for building infrastructure
	Builder() cmd.Builder

	// Linter returns the implementation for linting source files
	Linter() cmd.Linter

	// Initializer returns the implementation for creating new projects
	Initializer() cmd.Initializer

	// Validator returns the implementation for validating infrastructure
	Validator() cmd.Validator
}

// OptionalImporter provides import functionality (e.g., YAML/JSON to Go)
type OptionalImporter interface {
	Importer() Importer
}

// OptionalLister provides resource listing functionality
type OptionalLister interface {
	Lister() Lister
}

// OptionalGrapher provides dependency graph generation
type OptionalGrapher interface {
	Grapher() Grapher
}

// Importer converts external formats to domain source code
type Importer interface {
	Import(inputPath, outputPath string) error
}

// Lister lists discovered resources in packages
type Lister interface {
	List(packages []string, format string) error
}

// Grapher generates dependency graphs
type Grapher interface {
	Graph(packages []string, format string) error
}

// CreateRootCommand creates a root command with all standard domain commands.
// This allows callers to add additional domain-specific commands before executing.
func CreateRootCommand(d Domain) *cobra.Command {
	description := "Infrastructure as Code for " + d.Name()
	root := cmd.NewRootCommand("wetwire-"+d.Name(), description)

	// Add version command
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			println("wetwire-" + d.Name() + " " + d.Version())
		},
	})

	// Add core commands
	root.AddCommand(cmd.NewBuildCommand(d.Builder()))
	root.AddCommand(cmd.NewLintCommand(d.Linter()))
	root.AddCommand(cmd.NewInitCommand(d.Initializer()))
	root.AddCommand(cmd.NewValidateCommand(d.Validator()))

	// Add optional commands if implemented
	if importer, ok := d.(OptionalImporter); ok {
		root.AddCommand(newImportCommand(importer.Importer()))
	}

	if lister, ok := d.(OptionalLister); ok {
		root.AddCommand(newListCommand(lister.Lister()))
	}

	if grapher, ok := d.(OptionalGrapher); ok {
		root.AddCommand(newGraphCommand(grapher.Grapher()))
	}

	return root
}

// Run creates and executes a CLI for the given domain.
// It automatically generates commands based on the domain's interface implementations.
func Run(d Domain) error {
	root := CreateRootCommand(d)
	return root.Execute()
}

func newImportCommand(importer Importer) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "import [input]",
		Short: "Import external format to domain source code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return importer.Import(args[0], outputPath)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")

	return cmd
}

func newListCommand(lister Lister) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list [packages...]",
		Short: "List discovered resources",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return lister.List(args, format)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json, yaml")

	return cmd
}

func newGraphCommand(grapher Grapher) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "graph [packages...]",
		Short: "Generate dependency graph",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return grapher.Graph(args, format)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "dot", "Output format: dot, mermaid")

	return cmd
}
