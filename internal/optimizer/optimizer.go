// Package optimizer provides CloudFormation optimization suggestions.
// It analyzes resources for security, cost, performance, and reliability improvements.
package optimizer

import (
	wetwire "github.com/lex00/wetwire-aws-go"
	"github.com/lex00/wetwire-aws-go/internal/discover"
)

// Options configures the optimizer.
type Options struct {
	// Category filters suggestions: "all", "security", "cost", "performance", "reliability"
	Category string
}

// Result contains optimization suggestions.
type Result struct {
	Suggestions []wetwire.OptimizeSuggestion
	Summary     wetwire.OptimizeSummary
}

// Optimize analyzes discovered resources and returns optimization suggestions.
func Optimize(discoverResult *discover.Result, opts Options) (*Result, error) {
	result := &Result{}

	// Apply all rules to each resource
	for name, res := range discoverResult.Resources {
		// Ensure the resource name is set
		if res.Name == "" {
			res.Name = name
		}
		suggestions := analyzeResource(res, opts.Category)
		result.Suggestions = append(result.Suggestions, suggestions...)
	}

	// Calculate summary
	result.Summary = calculateSummary(result.Suggestions)

	return result, nil
}

// analyzeResource applies optimization rules to a single resource.
func analyzeResource(res wetwire.DiscoveredResource, category string) []wetwire.OptimizeSuggestion {
	var suggestions []wetwire.OptimizeSuggestion

	// Run rules based on resource type
	rules := getRulesForType(res.Type)
	for _, rule := range rules {
		if category != "all" && rule.Category != category {
			continue
		}
		if suggestion := rule.Check(res); suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	return suggestions
}

// calculateSummary tallies suggestions by category.
func calculateSummary(suggestions []wetwire.OptimizeSuggestion) wetwire.OptimizeSummary {
	summary := wetwire.OptimizeSummary{}
	for _, s := range suggestions {
		switch s.Category {
		case "security":
			summary.Security++
		case "cost":
			summary.Cost++
		case "performance":
			summary.Performance++
		case "reliability":
			summary.Reliability++
		}
		summary.Total++
	}
	return summary
}

// Rule represents an optimization rule.
type Rule struct {
	ID          string
	Category    string
	Title       string
	Description string
	Check       func(res wetwire.DiscoveredResource) *wetwire.OptimizeSuggestion
}

// getRulesForType returns applicable rules for a resource type.
func getRulesForType(resourceType string) []Rule {
	var rules []Rule

	// Add type-specific rules
	switch resourceType {
	case "s3.Bucket":
		rules = append(rules, s3BucketRules...)
	case "lambda.Function":
		rules = append(rules, lambdaFunctionRules...)
	case "iam.Role", "iam.Policy":
		rules = append(rules, iamRules...)
	case "ec2.Instance":
		rules = append(rules, ec2InstanceRules...)
	case "rds.DBInstance":
		rules = append(rules, rdsInstanceRules...)
	case "dynamodb.Table":
		rules = append(rules, dynamoDBTableRules...)
	}

	// Add generic rules applicable to all resources
	rules = append(rules, genericRules...)

	return rules
}
