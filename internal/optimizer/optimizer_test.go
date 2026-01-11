package optimizer

import (
	"testing"

	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
)

func TestOptimize(t *testing.T) {
	discoverResult := &discover.Result{
		Resources: map[string]wetwire.DiscoveredResource{
			"DataBucket": {
				Name:    "DataBucket",
				Type:    "s3.Bucket",
				Package: "myproject",
				File:    "storage.go",
				Line:    10,
			},
			"ProcessorRole": {
				Name:    "ProcessorRole",
				Type:    "iam.Role",
				Package: "myproject",
				File:    "security.go",
				Line:    5,
			},
		},
	}

	result, err := Optimize(discoverResult, Options{Category: "all"})
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}

	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions for S3 bucket and IAM role")
	}

	// Verify we have different categories
	categories := map[string]bool{}
	for _, s := range result.Suggestions {
		categories[s.Category] = true
	}

	if !categories["security"] {
		t.Error("expected security suggestions")
	}
}

func TestOptimizeWithCategoryFilter(t *testing.T) {
	discoverResult := &discover.Result{
		Resources: map[string]wetwire.DiscoveredResource{
			"DataBucket": {
				Name: "DataBucket",
				Type: "s3.Bucket",
				File: "storage.go",
				Line: 10,
			},
		},
	}

	result, err := Optimize(discoverResult, Options{Category: "security"})
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}

	// All suggestions should be security category
	for _, s := range result.Suggestions {
		if s.Category != "security" {
			t.Errorf("expected only security suggestions, got %s", s.Category)
		}
	}
}

func TestOptimizeEmptyResources(t *testing.T) {
	discoverResult := &discover.Result{
		Resources: map[string]wetwire.DiscoveredResource{},
	}

	result, err := Optimize(discoverResult, Options{Category: "all"})
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}

	if len(result.Suggestions) != 0 {
		t.Error("expected no suggestions for empty resources")
	}

	if result.Summary.Total != 0 {
		t.Error("expected zero total in summary")
	}
}

func TestCalculateSummary(t *testing.T) {
	suggestions := []wetwire.OptimizeSuggestion{
		{Category: "security"},
		{Category: "security"},
		{Category: "cost"},
		{Category: "performance"},
		{Category: "reliability"},
		{Category: "reliability"},
	}

	summary := calculateSummary(suggestions)

	if summary.Security != 2 {
		t.Errorf("Security = %d, want 2", summary.Security)
	}
	if summary.Cost != 1 {
		t.Errorf("Cost = %d, want 1", summary.Cost)
	}
	if summary.Performance != 1 {
		t.Errorf("Performance = %d, want 1", summary.Performance)
	}
	if summary.Reliability != 2 {
		t.Errorf("Reliability = %d, want 2", summary.Reliability)
	}
	if summary.Total != 6 {
		t.Errorf("Total = %d, want 6", summary.Total)
	}
}

func TestGetRulesForType(t *testing.T) {
	tests := []struct {
		resourceType string
		wantRules    bool
	}{
		{"s3.Bucket", true},
		{"lambda.Function", true},
		{"iam.Role", true},
		{"iam.Policy", true},
		{"ec2.Instance", true},
		{"rds.DBInstance", true},
		{"dynamodb.Table", true},
		{"unknown.Type", true}, // generic rules still apply
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			rules := getRulesForType(tt.resourceType)
			if tt.wantRules && len(rules) == 0 {
				t.Errorf("getRulesForType(%s) returned no rules", tt.resourceType)
			}
		})
	}
}
