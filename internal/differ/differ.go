// Package differ provides semantic comparison of CloudFormation templates.
package differ

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"gopkg.in/yaml.v3"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// Options configures the differ.
type Options struct {
	// IgnoreOrder ignores array element order in comparisons
	IgnoreOrder bool
}

// Result contains the difference between two templates.
type Result struct {
	Diff    wetwire.TemplateDiff
	Summary wetwire.DiffSummary
}

// Compare compares two CloudFormation templates and returns differences.
func Compare(template1, template2 *wetwire.Template, opts Options) (*Result, error) {
	result := &Result{}

	// Build resource maps
	res1 := template1.Resources
	res2 := template2.Resources

	// Find added resources (in template2 but not in template1)
	for name, def := range res2 {
		if _, exists := res1[name]; !exists {
			result.Diff.Added = append(result.Diff.Added, wetwire.DiffEntry{
				Resource: name,
				Type:     def.Type,
			})
		}
	}

	// Find removed resources (in template1 but not in template2)
	for name, def := range res1 {
		if _, exists := res2[name]; !exists {
			result.Diff.Removed = append(result.Diff.Removed, wetwire.DiffEntry{
				Resource: name,
				Type:     def.Type,
			})
		}
	}

	// Find modified resources
	for name, def1 := range res1 {
		if def2, exists := res2[name]; exists {
			changes := compareResources(name, def1, def2, opts)
			if len(changes) > 0 {
				result.Diff.Modified = append(result.Diff.Modified, wetwire.DiffEntry{
					Resource: name,
					Type:     def1.Type,
					Changes:  changes,
				})
			}
		}
	}

	// Sort entries for consistent output
	sortEntries(result.Diff.Added)
	sortEntries(result.Diff.Removed)
	sortEntries(result.Diff.Modified)

	// Calculate summary
	result.Summary = wetwire.DiffSummary{
		Added:    len(result.Diff.Added),
		Removed:  len(result.Diff.Removed),
		Modified: len(result.Diff.Modified),
	}
	result.Summary.Total = result.Summary.Added + result.Summary.Removed + result.Summary.Modified

	return result, nil
}

// CompareFiles compares two template files.
func CompareFiles(file1, file2 string, opts Options) (*Result, error) {
	t1, err := LoadTemplate(file1)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", file1, err)
	}

	t2, err := LoadTemplate(file2)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", file2, err)
	}

	return Compare(t1, t2, opts)
}

// LoadTemplate loads a CloudFormation template from a file.
func LoadTemplate(path string) (*wetwire.Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var template wetwire.Template

	// Try JSON first
	if err := json.Unmarshal(data, &template); err != nil {
		// Try YAML
		if err := yaml.Unmarshal(data, &template); err != nil {
			return nil, fmt.Errorf("failed to parse as JSON or YAML: %w", err)
		}
	}

	return &template, nil
}

// compareResources compares two resource definitions and returns changes.
func compareResources(name string, def1, def2 wetwire.ResourceDef, opts Options) []string {
	var changes []string

	// Compare type
	if def1.Type != def2.Type {
		changes = append(changes, fmt.Sprintf("Type changed: %s → %s", def1.Type, def2.Type))
	}

	// Compare properties
	propChanges := compareProperties("", def1.Properties, def2.Properties, opts)
	changes = append(changes, propChanges...)

	// Compare DependsOn
	if !equalStringSlices(def1.DependsOn, def2.DependsOn) {
		changes = append(changes, "DependsOn changed")
	}

	return changes
}

// compareProperties recursively compares property maps.
func compareProperties(prefix string, props1, props2 map[string]any, opts Options) []string {
	var changes []string

	// Find added/modified properties
	for key, val2 := range props2 {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		if val1, exists := props1[key]; exists {
			if !deepEqual(val1, val2, opts) {
				changes = append(changes, fmt.Sprintf("%s modified", path))
			}
		} else {
			changes = append(changes, fmt.Sprintf("%s added", path))
		}
	}

	// Find removed properties
	for key := range props1 {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		if _, exists := props2[key]; !exists {
			changes = append(changes, fmt.Sprintf("%s removed", path))
		}
	}

	sort.Strings(changes)
	return changes
}

// deepEqual compares two values deeply, optionally ignoring order.
func deepEqual(a, b any, opts Options) bool {
	if opts.IgnoreOrder {
		// Normalize slices for comparison
		a = normalizeValue(a)
		b = normalizeValue(b)
	}
	return reflect.DeepEqual(a, b)
}

// normalizeValue normalizes a value for comparison.
func normalizeValue(v any) any {
	switch val := v.(type) {
	case []any:
		// Sort string slices for comparison
		result := make([]any, len(val))
		copy(result, val)
		return result
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = normalizeValue(v)
		}
		return result
	default:
		return v
	}
}

// equalStringSlices compares two string slices for equality.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sortEntries sorts diff entries by resource name.
func sortEntries(entries []wetwire.DiffEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Resource < entries[j].Resource
	})
}
