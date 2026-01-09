package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasVendorDir(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// No vendor - should return false
	if hasVendorDir(tmpDir) {
		t.Error("hasVendorDir should return false when no vendor directory")
	}

	// Create vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.Mkdir(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	// With vendor - should return true
	if !hasVendorDir(tmpDir) {
		t.Error("hasVendorDir should return true when vendor directory exists")
	}
}

func TestCreateRunnerSubdir(t *testing.T) {
	// Create temp directory with basic Go structure
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/test

go 1.23
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.Mkdir(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create runner subdir
	runnerDir, cleanup, err := createRunnerSubdir(tmpDir)
	if err != nil {
		t.Fatalf("createRunnerSubdir failed: %v", err)
	}
	defer cleanup()

	// Verify runner directory was created
	expected := filepath.Join(tmpDir, "_wetwire_runner")
	if runnerDir != expected {
		t.Errorf("runnerDir = %q, want %q", runnerDir, expected)
	}

	// Verify directory exists
	if _, err := os.Stat(runnerDir); err != nil {
		t.Errorf("runner directory should exist: %v", err)
	}

	// Run cleanup and verify directory is removed
	cleanup()
	if _, err := os.Stat(runnerDir); !os.IsNotExist(err) {
		t.Error("runner directory should be removed after cleanup")
	}
}

func TestRunnerModeSelection(t *testing.T) {
	tests := []struct {
		name       string
		hasVendor  bool
		wantSubdir bool // true = use _wetwire_runner subdir, false = use temp dir
	}{
		{"no vendor uses temp dir", false, false},
		{"with vendor uses subdir", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			if tt.hasVendor {
				if err := os.Mkdir(filepath.Join(tmpDir, "vendor"), 0755); err != nil {
					t.Fatal(err)
				}
			}

			gotSubdir := shouldUseSubdirRunner(tmpDir)
			if gotSubdir != tt.wantSubdir {
				t.Errorf("shouldUseSubdirRunner = %v, want %v", gotSubdir, tt.wantSubdir)
			}
		})
	}
}
