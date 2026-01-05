package main

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	version := getVersion()

	// Version should not be empty
	if version == "" {
		t.Error("getVersion() returned empty string")
	}

	// When running tests (not via go install), version should be "dev"
	// or a valid semver when installed via go install @version
	if version != "dev" && !strings.HasPrefix(version, "v") {
		t.Errorf("getVersion() = %q, want 'dev' or 'vX.Y.Z'", version)
	}
}

func TestGetVersionNotEmpty(t *testing.T) {
	// Ensure the version function always returns something useful
	version := getVersion()
	if len(version) == 0 {
		t.Error("version should not be empty")
	}
}
