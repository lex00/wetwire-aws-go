package main

import (
	"testing"
)

func TestNewOptimizeCmd(t *testing.T) {
	cmd := newOptimizeCmd()

	if cmd.Use != "optimize [packages...]" {
		t.Errorf("Use = %q, want 'optimize [packages...]'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Check flags exist
	if cmd.Flags().Lookup("format") == nil {
		t.Error("missing --format flag")
	}

	if cmd.Flags().Lookup("category") == nil {
		t.Error("missing --category flag")
	}
}

func TestOptimizeCategories(t *testing.T) {
	// Verify all expected categories are supported
	expected := []string{"all", "security", "cost", "performance", "reliability"}

	for _, cat := range expected {
		if !isValidCategory(cat) {
			t.Errorf("category %q should be valid", cat)
		}
	}

	// Invalid category
	if isValidCategory("invalid") {
		t.Error("'invalid' should not be a valid category")
	}
}
