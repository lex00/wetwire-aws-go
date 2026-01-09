// Package graph generates DOT and Mermaid format dependency graphs from discovered resources.
package graph

import (
	"io"
	"strings"

	"github.com/emicklei/dot"
	wetwire "github.com/lex00/wetwire-aws-go"
)

// Format specifies the output format for the graph.
type Format string

const (
	// FormatDOT outputs Graphviz DOT format.
	FormatDOT Format = "dot"
	// FormatMermaid outputs Mermaid format for GitHub/markdown rendering.
	FormatMermaid Format = "mermaid"
)

// Generator creates dependency graphs from discovered resources.
type Generator struct {
	// IncludeParameters includes parameter references in the graph.
	IncludeParameters bool

	// Format specifies the output format (dot or mermaid). Defaults to dot.
	Format Format

	// ClusterByType groups resources by AWS service type.
	ClusterByType bool
}

// Generate creates a dependency graph and writes it to w.
func (g *Generator) Generate(resources map[string]wetwire.DiscoveredResource, parameters map[string]wetwire.DiscoveredParameter, w io.Writer) error {
	graph := g.buildGraph(resources, parameters)

	format := g.Format
	if format == "" {
		format = FormatDOT
	}

	var output string
	if format == FormatMermaid {
		output = dot.MermaidGraph(graph, dot.MermaidTopToBottom)
	} else {
		output = graph.String()
	}

	_, err := w.Write([]byte(output))
	return err
}

// GenerateString is a convenience method that returns the graph as a string.
func (g *Generator) GenerateString(resources map[string]wetwire.DiscoveredResource, parameters map[string]wetwire.DiscoveredParameter) (string, error) {
	var sb strings.Builder
	if err := g.Generate(resources, parameters, &sb); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// buildGraph creates the dot.Graph structure from discovered resources.
func (g *Generator) buildGraph(resources map[string]wetwire.DiscoveredResource, parameters map[string]wetwire.DiscoveredParameter) *dot.Graph {
	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "TB")

	// Set default node style
	graph.NodeInitializer(func(n dot.Node) {
		n.Attr("shape", "box")
		n.Attr("fontname", "Arial")
	})

	// Set default edge style
	graph.EdgeInitializer(func(e dot.Edge) {
		e.Attr("fontname", "Arial")
		e.Attr("fontsize", "10")
	})

	// Build set of GetAtt references for edge styling
	getAttRefs := g.buildGetAttSet(resources)

	if g.ClusterByType {
		g.addClusteredNodes(graph, resources)
	} else {
		g.addNodes(graph, resources)
	}

	// Add parameter nodes if requested
	if g.IncludeParameters && parameters != nil {
		for name := range parameters {
			n := graph.Node(name)
			n.Attr("shape", "ellipse")
			n.Attr("style", "dashed")
			n.Label(name)
		}
	}

	// Add edges
	for name, res := range resources {
		for _, dep := range res.Dependencies {
			// Skip if dependency is a parameter and we're not including parameters
			if _, isParam := parameters[dep]; isParam && !g.IncludeParameters {
				continue
			}
			// Skip if dependency doesn't exist as resource or parameter
			_, isResource := resources[dep]
			_, isParam := parameters[dep]
			if !isResource && !isParam {
				continue
			}

			from := graph.Node(name)
			to := graph.Node(dep)
			e := graph.Edge(from, to)

			// Style based on reference type
			key := name + "->" + dep
			if getAttRefs[key] {
				e.Attr("color", "blue")
			}
		}
	}

	return graph
}

// buildGetAttSet creates a set of edges that are GetAtt references.
func (g *Generator) buildGetAttSet(resources map[string]wetwire.DiscoveredResource) map[string]bool {
	getAttRefs := make(map[string]bool)
	for name, res := range resources {
		for _, usage := range res.AttrRefUsages {
			key := name + "->" + usage.ResourceName
			getAttRefs[key] = true
		}
	}
	return getAttRefs
}

// addNodes adds resource nodes without clustering.
func (g *Generator) addNodes(graph *dot.Graph, resources map[string]wetwire.DiscoveredResource) {
	for name, res := range resources {
		n := graph.Node(name)
		cfType := goTypeToCFType(res.Type)
		n.Label(name + "\\n[" + cfType + "]")
	}
}

// addClusteredNodes adds resource nodes grouped by AWS service type.
func (g *Generator) addClusteredNodes(graph *dot.Graph, resources map[string]wetwire.DiscoveredResource) {
	// Group resources by service
	serviceResources := make(map[string][]string)
	resourceTypes := make(map[string]string)

	for name, res := range resources {
		service := extractService(res.Type)
		serviceResources[service] = append(serviceResources[service], name)
		resourceTypes[name] = res.Type
	}

	// Create clusters for each service with multiple resources
	for service, resNames := range serviceResources {
		if len(resNames) > 1 {
			cluster := graph.Subgraph("cluster_"+service, dot.ClusterOption{})
			cluster.Attr("label", service)
			cluster.Attr("style", "rounded")
			cluster.Attr("bgcolor", "lightyellow")

			for _, name := range resNames {
				n := cluster.Node(name)
				cfType := goTypeToCFType(resourceTypes[name])
				n.Label(name + "\\n[" + cfType + "]")
			}
		} else {
			// Single resource, no cluster needed
			for _, name := range resNames {
				n := graph.Node(name)
				cfType := goTypeToCFType(resourceTypes[name])
				n.Label(name + "\\n[" + cfType + "]")
			}
		}
	}
}

// extractService extracts the AWS service name from a Go type.
// e.g., "s3.Bucket" -> "S3"
func extractService(goType string) string {
	parts := strings.Split(goType, ".")
	if len(parts) >= 1 {
		return strings.ToUpper(parts[0])
	}
	return "Other"
}

// goTypeToCFType converts a Go type to CloudFormation type format.
// e.g., "s3.Bucket" -> "AWS::S3::Bucket"
func goTypeToCFType(goType string) string {
	parts := strings.Split(goType, ".")
	if len(parts) == 2 {
		service := strings.ToUpper(parts[0][:1]) + parts[0][1:]
		return "AWS::" + service + "::" + parts[1]
	}
	return goType
}
