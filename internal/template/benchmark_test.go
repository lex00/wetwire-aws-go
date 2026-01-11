package template

import (
	"fmt"
	"testing"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// BenchmarkBuild benchmarks building templates with varying resource counts.
func BenchmarkBuild(b *testing.B) {
	sizes := []int{10, 50, 100, 200}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("resources_%d", size), func(b *testing.B) {
			resources := generateMockResources(size)
			builder := NewBuilder(resources)

			// Set mock values
			for name := range resources {
				builder.SetValue(name, map[string]any{
					"BucketName": name + "-bucket",
				})
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := builder.Build()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkToJSON benchmarks JSON serialization with varying resource counts.
func BenchmarkToJSON(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("resources_%d", size), func(b *testing.B) {
			resources := generateMockResources(size)
			builder := NewBuilder(resources)

			for name := range resources {
				builder.SetValue(name, map[string]any{
					"BucketName": name + "-bucket",
					"Tags": []map[string]string{
						{"Key": "Environment", "Value": "Test"},
						{"Key": "Project", "Value": "Benchmark"},
					},
				})
			}

			tmpl, err := builder.Build()
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ToJSON(tmpl)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkToYAML benchmarks YAML serialization with varying resource counts.
func BenchmarkToYAML(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("resources_%d", size), func(b *testing.B) {
			resources := generateMockResources(size)
			builder := NewBuilder(resources)

			for name := range resources {
				builder.SetValue(name, map[string]any{
					"BucketName": name + "-bucket",
					"Tags": []map[string]string{
						{"Key": "Environment", "Value": "Test"},
					},
				})
			}

			tmpl, err := builder.Build()
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ToYAML(tmpl)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTopologicalSort benchmarks dependency ordering with varying complexity.
func BenchmarkTopologicalSort(b *testing.B) {
	sizes := []int{20, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("resources_%d", size), func(b *testing.B) {
			// Create chain of dependencies
			resources := make(map[string]wetwire.DiscoveredResource)
			for i := 0; i < size; i++ {
				name := fmt.Sprintf("Resource%d", i)
				res := wetwire.DiscoveredResource{
					Name:    name,
					Type:    "s3.Bucket",
					Package: "s3",
				}
				// Create dependency chain
				if i > 0 {
					res.Dependencies = []string{fmt.Sprintf("Resource%d", i-1)}
				}
				resources[name] = res
			}

			builder := NewBuilder(resources)
			for name := range resources {
				builder.SetValue(name, map[string]any{})
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := builder.Build()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// generateMockResources creates a set of mock S3 bucket resources for benchmarking.
func generateMockResources(count int) map[string]wetwire.DiscoveredResource {
	resources := make(map[string]wetwire.DiscoveredResource, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("Bucket%d", i)
		resources[name] = wetwire.DiscoveredResource{
			Name:    name,
			Type:    "s3.Bucket",
			Package: "s3",
		}
	}

	return resources
}
