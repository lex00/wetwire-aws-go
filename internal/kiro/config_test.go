package kiro

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig_IncludesWorkDir(t *testing.T) {
	// Test that NewConfig includes the WorkDir field
	// This ensures the Kiro agent runs in the correct directory
	// See: https://github.com/lex00/wetwire-core-go/issues/XXX

	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	config := NewConfig()

	// Verify WorkDir is set
	if config.WorkDir == "" {
		t.Fatal("NewConfig should return non-empty WorkDir")
	}

	// Verify WorkDir matches current working directory
	expectedDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Resolve both paths to absolute form for comparison
	absWorkDir, err := filepath.Abs(config.WorkDir)
	if err != nil {
		t.Fatal(err)
	}

	absExpectedDir, err := filepath.Abs(expectedDir)
	if err != nil {
		t.Fatal(err)
	}

	if absWorkDir != absExpectedDir {
		t.Errorf("WorkDir = %q, want %q", absWorkDir, absExpectedDir)
	}
}
