package kiro

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestNewTestRunner(t *testing.T) {
	runner := NewTestRunner("/tmp/test")

	if runner.AgentName != "wetwire-aws-runner" {
		t.Errorf("AgentName = %q, want %q", runner.AgentName, "wetwire-aws-runner")
	}

	if runner.WorkDir != "/tmp/test" {
		t.Errorf("WorkDir = %q, want %q", runner.WorkDir, "/tmp/test")
	}

	if runner.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", runner.Timeout, 5*time.Minute)
	}
}

func TestTestRunner_Run_KiroNotInstalled(t *testing.T) {
	// Save original PATH and restore after test
	origPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", origPath) }()

	// Set PATH to empty to simulate kiro-cli not being installed
	_ = os.Setenv("PATH", "")

	runner := NewTestRunner(".")
	_, err := runner.Run(context.Background(), "test prompt")

	if err == nil {
		t.Fatal("expected error when kiro-cli is not in PATH")
	}
}

func TestTestRunner_ParseOutputLine(t *testing.T) {
	runner := NewTestRunner(".")
	result := &TestResult{}

	tests := []struct {
		line       string
		wantLint   bool
		wantBuild  bool
		wantFiles  int
		wantErrors int
	}{
		{"wetwire_lint: success", true, false, 0, 0},
		{"wetwire_lint passed", true, false, 0, 0},
		{"wetwire_build: success", false, true, 0, 0},
		{"Created storage.go", false, false, 1, 0},
		{"Error: something went wrong", false, false, 0, 1},
		{"random line", false, false, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result = &TestResult{} // Reset for each test
			runner.parseOutputLine(tt.line, result)

			if result.LintPassed != tt.wantLint {
				t.Errorf("LintPassed = %v, want %v", result.LintPassed, tt.wantLint)
			}
			if result.BuildPassed != tt.wantBuild {
				t.Errorf("BuildPassed = %v, want %v", result.BuildPassed, tt.wantBuild)
			}
			if len(result.FilesCreated) != tt.wantFiles {
				t.Errorf("FilesCreated = %d, want %d", len(result.FilesCreated), tt.wantFiles)
			}
			if len(result.ErrorMessages) != tt.wantErrors {
				t.Errorf("ErrorMessages = %d, want %d", len(result.ErrorMessages), tt.wantErrors)
			}
		})
	}
}

func TestTestRunner_EnsureTestEnvironment(t *testing.T) {
	// Skip if we can't create temp directories
	tmpDir := t.TempDir()

	runner := NewTestRunner(tmpDir)

	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	err = runner.EnsureTestEnvironment()
	if err != nil {
		t.Fatalf("EnsureTestEnvironment failed: %v", err)
	}

	// Verify .kiro/mcp.json was created in the work directory
	mcpPath := tmpDir + "/.kiro/mcp.json"
	if _, err := os.Stat(mcpPath); err != nil {
		t.Errorf(".kiro/mcp.json should exist in work directory: %v", err)
	}
}

// TestTestRunner_Integration is an integration test that requires kiro-cli.
// Set SKIP_KIRO_TESTS=1 to skip this test in CI.
func TestTestRunner_Integration(t *testing.T) {
	if os.Getenv("SKIP_KIRO_TESTS") == "1" {
		t.Skip("Skipping Kiro integration test (SKIP_KIRO_TESTS=1)")
	}

	// Skip if kiro-cli is not installed
	if _, err := exec.LookPath("kiro-cli"); err != nil {
		t.Skip("kiro-cli not installed, skipping integration test")
	}

	t.Log("kiro-cli found - integration test would run here")
	// Note: Actually running kiro-cli would require authentication
	// and would make network calls, so we skip the actual execution.
}
