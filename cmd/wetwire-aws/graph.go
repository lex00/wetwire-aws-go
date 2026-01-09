package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/graph"
)

func newGraphCmd() *cobra.Command {
	var (
		outputFormat      string
		includeParameters bool
		clusterByType     bool
	)

	cmd := &cobra.Command{
		Use:   "graph [packages...]",
		Short: "Generate DOT graph of resource dependencies",
		Long: `Generate a DOT or Mermaid format graph showing resource dependencies.

The output can be rendered with Graphviz:
    wetwire-aws graph ./infra | dot -Tpng -o deps.png

Or used in GitHub markdown (Mermaid format):
    wetwire-aws graph ./infra -f mermaid

Examples:
    wetwire-aws graph ./infra
    wetwire-aws graph ./infra -p              # include parameters
    wetwire-aws graph ./infra -c              # cluster by service
    wetwire-aws graph ./infra -f mermaid      # mermaid format`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGraph(args, outputFormat, includeParameters, clusterByType)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "dot", "Output format: dot or mermaid")
	cmd.Flags().BoolVarP(&includeParameters, "include-parameters", "p", false, "Include parameter nodes in the graph")
	cmd.Flags().BoolVarP(&clusterByType, "cluster", "c", false, "Cluster resources by AWS service type")

	return cmd
}

func runGraph(packages []string, format string, includeParams bool, cluster bool) error {
	// Discover resources
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if len(result.Resources) == 0 {
		return fmt.Errorf("no resources found")
	}

	// Convert format string to graph.Format
	var graphFormat graph.Format
	switch format {
	case "dot":
		graphFormat = graph.FormatDOT
	case "mermaid":
		graphFormat = graph.FormatMermaid
	default:
		return fmt.Errorf("unknown format: %s (use 'dot' or 'mermaid')", format)
	}

	// Generate graph
	gen := &graph.Generator{
		Format:            graphFormat,
		IncludeParameters: includeParams,
		ClusterByType:     cluster,
	}

	return gen.Generate(result.Resources, result.Parameters, os.Stdout)
}
