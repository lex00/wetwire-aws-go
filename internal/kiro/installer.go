// Package kiro provides Kiro CLI integration for wetwire-aws.
//
// This package handles:
//   - Auto-installation of Kiro agent configuration
//   - Project-level MCP configuration
//   - Launching Kiro CLI chat sessions
package kiro

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed configs/wetwire-runner.json
var configFS embed.FS

// mcpConfig represents the MCP configuration structure.
type mcpConfig struct {
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

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

// agentConfig represents the Kiro agent configuration structure.
type agentConfig struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Prompt      string                  `json:"prompt"`
	Model       string                  `json:"model"`
	MCPServers  map[string]mcpServer    `json:"mcpServers"`
	Tools       []string                `json:"tools"`
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

	// Parse and update with full MCP binary path
	var config agentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return false, fmt.Errorf("parsing embedded config: %w", err)
	}

	// Update MCP server command with full path
	mcpBinaryPath := findMCPBinaryPath()
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]mcpServer)
	}
	config.MCPServers["wetwire"] = mcpServer{
		Command: mcpBinaryPath,
		Args:    []string{},
	}

	// Marshal back to JSON
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshaling config: %w", err)
	}

	// Write config
	if err := os.WriteFile(agentPath, updatedData, 0644); err != nil {
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

	// Find wetwire-aws-mcp binary path
	mcpBinaryPath := findMCPBinaryPath()

	// Generate config with full path
	config := mcpConfig{
		MCPServers: map[string]mcpServer{
			"wetwire": {
				Command: mcpBinaryPath,
				Args:    []string{},
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshaling config: %w", err)
	}

	// Write config
	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		return false, fmt.Errorf("writing config: %w", err)
	}

	return true, nil
}

// findMCPBinaryPath returns the path to wetwire-aws-mcp.
// It looks in the same directory as the current executable first,
// then falls back to just the binary name (requiring PATH).
func findMCPBinaryPath() string {
	// Try to find it next to the current executable
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		mcpPath := filepath.Join(exeDir, "wetwire-aws-mcp")
		if _, err := os.Stat(mcpPath); err == nil {
			return mcpPath
		}
	}

	// Fall back to requiring PATH
	return "wetwire-aws-mcp"
}
