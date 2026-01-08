// Package kiro provides Kiro CLI integration for wetwire-aws.
//
// This package handles:
//   - Auto-installation of Kiro agent configuration
//   - Project-level MCP configuration
//   - Launching Kiro CLI chat sessions
package kiro

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed configs/wetwire-runner.json
//go:embed configs/mcp.json
var configFS embed.FS

// EnsureInstalled checks if Kiro configs are installed and installs them if needed.
// It installs:
//   - ~/.kiro/agents/wetwire-runner.json (user-level agent config)
//   - .kiro/mcp.json (project-level MCP config)
//
// Existing files are not overwritten.
func EnsureInstalled() error {
	agentInstalled, err := ensureAgentConfig()
	if err != nil {
		return fmt.Errorf("installing agent config: %w", err)
	}

	mcpInstalled, err := ensureProjectMCPConfig()
	if err != nil {
		return fmt.Errorf("installing project MCP config: %w", err)
	}

	if agentInstalled {
		fmt.Println("Installed Kiro agent config: ~/.kiro/agents/wetwire-runner.json")
	}
	if mcpInstalled {
		fmt.Println("Installed project MCP config: .kiro/mcp.json")
	}

	return nil
}

// ensureAgentConfig installs the wetwire-runner agent to ~/.kiro/agents/
// Returns true if the file was installed (didn't exist before).
func ensureAgentConfig() (bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("getting home directory: %w", err)
	}

	agentDir := filepath.Join(homeDir, ".kiro", "agents")
	agentPath := filepath.Join(agentDir, "wetwire-runner.json")

	// Check if already exists
	if _, err := os.Stat(agentPath); err == nil {
		return false, nil // Already exists, don't overwrite
	}

	// Create directory
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return false, fmt.Errorf("creating agents directory: %w", err)
	}

	// Read embedded config
	data, err := configFS.ReadFile("configs/wetwire-runner.json")
	if err != nil {
		return false, fmt.Errorf("reading embedded config: %w", err)
	}

	// Write config
	if err := os.WriteFile(agentPath, data, 0644); err != nil {
		return false, fmt.Errorf("writing config: %w", err)
	}

	return true, nil
}

// ensureProjectMCPConfig installs the MCP config to .kiro/mcp.json in the current directory.
// Returns true if the file was installed (didn't exist before).
func ensureProjectMCPConfig() (bool, error) {
	mcpDir := ".kiro"
	mcpPath := filepath.Join(mcpDir, "mcp.json")

	// Check if already exists
	if _, err := os.Stat(mcpPath); err == nil {
		return false, nil // Already exists, don't overwrite
	}

	// Create directory
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return false, fmt.Errorf("creating .kiro directory: %w", err)
	}

	// Read embedded config
	data, err := configFS.ReadFile("configs/mcp.json")
	if err != nil {
		return false, fmt.Errorf("reading embedded config: %w", err)
	}

	// Write config
	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		return false, fmt.Errorf("writing config: %w", err)
	}

	return true, nil
}
