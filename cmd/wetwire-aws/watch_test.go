package main

import (
	"testing"
)

func TestNewWatchCmd(t *testing.T) {
	cmd := newWatchCmd()

	if cmd.Use != "watch [packages...]" {
		t.Errorf("Use = %q, want 'watch [packages...]'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Check flags exist
	if cmd.Flags().Lookup("lint-only") == nil {
		t.Error("missing --lint-only flag")
	}

	if cmd.Flags().Lookup("debounce") == nil {
		t.Error("missing --debounce flag")
	}
}

func TestDebounceDefault(t *testing.T) {
	cmd := newWatchCmd()

	flag := cmd.Flags().Lookup("debounce")
	if flag == nil {
		t.Fatal("missing --debounce flag")
	}

	if flag.DefValue != "500ms" {
		t.Errorf("debounce default = %q, want '500ms'", flag.DefValue)
	}
}
