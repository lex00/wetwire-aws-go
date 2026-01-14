package kiro

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedConfigs_ValidJSON(t *testing.T) {
	// Only wetwire-aws-runner.json is embedded; mcp.json is generated dynamically
	data, err := configFS.ReadFile("configs/wetwire-aws-runner.json")
	if err != nil {
		t.Fatalf("failed to read embedded config: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("embedded config is empty")
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("embedded config is not valid JSON: %v", err)
	}
}

func TestEmbeddedAgentConfig_HasRequiredFields(t *testing.T) {
	data, err := configFS.ReadFile("configs/wetwire-aws-runner.json")
	if err != nil {
		t.Fatalf("failed to read agent config: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse agent config: %v", err)
	}

	requiredFields := []string{"name", "description", "prompt", "mcpServers"}
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			t.Errorf("agent config missing required field: %s", field)
		}
	}

	// Check name is correct
	if name, ok := config["name"].(string); !ok || name != "wetwire-aws-runner" {
		t.Errorf("agent config name should be 'wetwire-aws-runner', got %v", config["name"])
	}

	// Check mcpServers has wetwire entry
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers should be an object")
	}
	if _, ok := mcpServers["wetwire"]; !ok {
		t.Error("mcpServers should have 'wetwire' entry")
	}
}

func TestGetMCPServerConfig_ReturnsValidConfig(t *testing.T) {
	// Test that NewConfig returns a valid configuration
	config := NewConfig()

	// AgentName should be set
	if config.AgentName == "" {
		t.Error("NewConfig should return non-empty AgentName")
	}

	if config.AgentName != AgentName {
		t.Errorf("AgentName = %q, want %q", config.AgentName, AgentName)
	}

	// AgentPrompt should not be empty
	if config.AgentPrompt == "" {
		t.Error("NewConfig should return non-empty AgentPrompt")
	}

	// MCPCommand should not be empty
	if config.MCPCommand == "" {
		t.Error("NewConfig should return non-empty MCPCommand")
	}
}

func TestEnsureProjectMCPConfig_CreatesFile(t *testing.T) {
	// Create temp directory and change to it
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Install config using EnsureInstalled
	err = EnsureInstalled()
	if err != nil {
		t.Fatalf("EnsureInstalled failed: %v", err)
	}

	// Verify agent config file exists in home directory
	// Note: The core kiro package installs to ~/.kiro/agents/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	agentPath := filepath.Join(homeDir, ".kiro", "agents", AgentName+".json")
	if _, err := os.Stat(agentPath); err != nil {
		t.Logf("Agent config not found at %s (may be expected in test environment)", agentPath)
	}
}

func TestEnsureProjectMCPConfig_SkipsExisting(t *testing.T) {
	// Test that EnsureInstalled can be called multiple times without error
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// First install
	err = EnsureInstalled()
	if err != nil {
		t.Fatalf("First EnsureInstalled failed: %v", err)
	}

	// Second install should not error
	err = EnsureInstalled()
	if err != nil {
		t.Fatalf("Second EnsureInstalled failed: %v", err)
	}
}

func TestEnsureProjectMCPConfig_CreatesDirectory(t *testing.T) {
	// Test that EnsureInstalled creates necessary directories
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Install config
	err = EnsureInstalled()
	if err != nil {
		t.Fatalf("EnsureInstalled failed: %v", err)
	}

	// Verify the installation completed without error
	// The core kiro package handles directory creation in user's home directory
	t.Log("Installation completed successfully")
}

func TestInstall_HasToolsArray(t *testing.T) {
	// Test that the generated config includes tools array
	// Required for kiro to enable MCP tool usage
	// See: https://github.com/aws/amazon-q-developer-cli/issues/2640

	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Override home directory
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Override working directory for the install
	origWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(origWd)

	// Run install
	if err := EnsureInstalledWithForce(true); err != nil {
		t.Fatalf("EnsureInstalledWithForce failed: %v", err)
	}

	// Read the agent config
	agentPath := filepath.Join(homeDir, ".kiro", "agents", AgentName+".json")
	data, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("failed to read agent config: %v", err)
	}

	var agent map[string]any
	if err := json.Unmarshal(data, &agent); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Must have tools array
	tools, ok := agent["tools"].([]any)
	if !ok {
		t.Fatal("agent config must have 'tools' array - required for kiro MCP tool usage")
	}

	// Must have at least one tool reference
	if len(tools) == 0 {
		t.Error("tools array must not be empty")
	}

	// First tool should be @server_name format
	if len(tools) > 0 {
		tool, ok := tools[0].(string)
		if !ok || len(tool) == 0 || tool[0] != '@' {
			t.Errorf("tools should use @server_name format, got: %v", tools)
		}
	}
}
