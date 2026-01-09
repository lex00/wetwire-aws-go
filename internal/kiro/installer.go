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
	"os/exec"
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
	Cwd     string   `json:"cwd,omitempty"`
}

// EnsureInstalled checks if Kiro configs are installed and installs them if needed.
// It installs:
//   - ~/.kiro/agents/wetwire-runner.json (user-level agent config)
//   - .kiro/mcp.json (project-level MCP config)
//
// Existing files are not overwritten unless force is true.
func EnsureInstalled() error {
	return EnsureInstalledWithForce(false)
}

// EnsureInstalledWithForce installs Kiro configs, optionally overwriting existing ones.
// When force is true, configs are always reinstalled to ensure latest prompt is used.
func EnsureInstalledWithForce(force bool) error {
	agentInstalled, err := ensureAgentConfig(force)
	if err != nil {
		return fmt.Errorf("installing agent config: %w", err)
	}

	mcpInstalled, err := ensureProjectMCPConfig(force)
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
// Returns true if the file was installed.
func ensureAgentConfig(force bool) (bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("getting home directory: %w", err)
	}

	agentDir := filepath.Join(homeDir, ".kiro", "agents")
	agentPath := filepath.Join(agentDir, "wetwire-runner.json")

	// Check if already exists (skip if not forcing)
	if _, err := os.Stat(agentPath); err == nil && !force {
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

	// Update MCP server with correct command
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]mcpServer)
	}
	config.MCPServers["wetwire"] = getMCPServerConfig()

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
// Returns true if the file was installed.
func ensureProjectMCPConfig(force bool) (bool, error) {
	mcpDir := ".kiro"
	mcpPath := filepath.Join(mcpDir, "mcp.json")

	// Check if already exists (skip if not forcing)
	if _, err := os.Stat(mcpPath); err == nil && !force {
		return false, nil // Already exists, don't overwrite
	}

	// Create directory
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return false, fmt.Errorf("creating .kiro directory: %w", err)
	}

	// Generate config with MCP server
	config := mcpConfig{
		MCPServers: map[string]mcpServer{
			"wetwire": getMCPServerConfig(),
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

// findWetwireBinaryPath returns the path to wetwire-aws.
// It looks for the current executable first, then checks PATH,
// then returns empty string for go run fallback.
func findWetwireBinaryPath() string {
	// Try to use the current executable
	exe, err := os.Executable()
	if err == nil {
		// Resolve symlinks to get the actual path
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		return exe
	}

	// Check if wetwire-aws is in PATH
	if path, err := exec.LookPath("wetwire-aws"); err == nil {
		return path
	}

	// Return empty to trigger go run fallback
	return ""
}

// getMCPServerConfig returns the mcpServer config for the embedded MCP server.
// Uses the current wetwire-aws binary with "design --mcp-server" flag.
// Sets cwd to ensure paths resolve correctly in the project directory.
func getMCPServerConfig() mcpServer {
	cwd, _ := os.Getwd()

	wetwirePath := findWetwireBinaryPath()
	if wetwirePath != "" {
		return mcpServer{
			Command: wetwirePath,
			Args:    []string{"design", "--mcp-server"},
			Cwd:     cwd,
		}
	}

	// Fallback to go run
	return mcpServer{
		Command: "go",
		Args:    []string{"run", "github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest", "design", "--mcp-server"},
		Cwd:     cwd,
	}
}
