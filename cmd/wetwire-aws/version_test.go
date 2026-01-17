package main

import (
	"strings"
	"testing"

	"github.com/lex00/wetwire-aws-go/domain"
)

func TestGetVersion(t *testing.T) {
	v := domain.Version

	// Version should not be empty
	if v == "" {
		t.Error("Version is empty")
	}

	// When running tests, version should be "dev"
	// or a valid semver when built with ldflags
	if v != "dev" && !strings.HasPrefix(v, "v") {
		t.Errorf("Version = %q, want 'dev' or 'vX.Y.Z'", v)
	}
}

func TestGetVersionNotEmpty(t *testing.T) {
	// Ensure the version is always set to something useful
	v := domain.Version
	if len(v) == 0 {
		t.Error("version should not be empty")
	}
}
