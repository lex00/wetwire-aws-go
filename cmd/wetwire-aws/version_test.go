package main

import (
	"strings"
	"testing"

	"github.com/lex00/wetwire-aws-go/version"
)

func TestGetVersion(t *testing.T) {
	v := version.Version()

	// Version should not be empty
	if v == "" {
		t.Error("Version() returned empty string")
	}

	// When running tests (not via go install), version should be "dev"
	// or a valid semver when installed via go install @version
	if v != "dev" && !strings.HasPrefix(v, "v") {
		t.Errorf("Version() = %q, want 'dev' or 'vX.Y.Z'", v)
	}
}

func TestGetVersionNotEmpty(t *testing.T) {
	// Ensure the version function always returns something useful
	v := version.Version()
	if len(v) == 0 {
		t.Error("version should not be empty")
	}
}
