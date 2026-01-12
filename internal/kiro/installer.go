package kiro

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	corekiro "github.com/lex00/wetwire-core-go/kiro"
)

// EnsureInstalled installs the Kiro agent configuration if not already present.
func EnsureInstalled() error {
	return EnsureInstalledWithForce(false)
}

// EnsureInstalledWithForce installs the Kiro agent configuration.
// If force is true, overwrites any existing configuration.
func EnsureInstalledWithForce(force bool) error {
	// Use core kiro package to install
	config := NewConfig()

	// Override MCP command/args to use binary path detection
	mcpPath := findMCPBinaryPath()
	if mcpPath != "" {
		// Found binary, use it directly with mcp subcommand
		config.MCPCommand = mcpPath
		config.MCPArgs = []string{"mcp"}
	} else {
		// Fallback to go run with mcp subcommand
		config.MCPCommand = "go"
		config.MCPArgs = []string{"run", "github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest", "mcp"}
	}

	if err := corekiro.Install(config); err != nil {
		return fmt.Errorf("install kiro config: %w", err)
	}

	return nil
}

// findMCPBinaryPath returns the path to wetwire-aws binary.
// It looks in the same directory as the current executable first,
// then checks PATH, then returns empty string for go run fallback.
func findMCPBinaryPath() string {
	// Try to find wetwire-aws next to current executable
	exe, err := os.Executable()
	if err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		mcpPath := filepath.Join(filepath.Dir(exe), "wetwire-aws")
		if _, err := os.Stat(mcpPath); err == nil {
			return mcpPath
		}
	}

	// Check if wetwire-aws is in PATH
	if path, err := exec.LookPath("wetwire-aws"); err == nil {
		return path
	}

	// Return empty to trigger go run fallback
	return ""
}
