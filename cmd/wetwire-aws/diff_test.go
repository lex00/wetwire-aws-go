package main

import (
	"testing"
)

func TestNewDiffCmd(t *testing.T) {
	cmd := newDiffCmd()

	if cmd.Use != "diff <template1> <template2>" {
		t.Errorf("Use = %q, want 'diff <template1> <template2>'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Check flags exist
	if cmd.Flags().Lookup("format") == nil {
		t.Error("missing --format flag")
	}

	if cmd.Flags().Lookup("ignore-order") == nil {
		t.Error("missing --ignore-order flag")
	}
}
