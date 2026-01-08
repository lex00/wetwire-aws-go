package kiro

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedConfigs_ValidJSON(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"wetwire-runner.json", "configs/wetwire-runner.json"},
		{"mcp.json", "configs/mcp.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := configFS.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("failed to read embedded config %s: %v", tt.path, err)
			}

			if len(data) == 0 {
				t.Fatalf("embedded config %s is empty", tt.path)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("embedded config %s is not valid JSON: %v", tt.path, err)
			}
		})
	}
}

func TestEmbeddedAgentConfig_HasRequiredFields(t *testing.T) {
	data, err := configFS.ReadFile("configs/wetwire-runner.json")
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
	if name, ok := config["name"].(string); !ok || name != "wetwire-runner" {
		t.Errorf("agent config name should be 'wetwire-runner', got %v", config["name"])
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

func TestEmbeddedMCPConfig_HasRequiredFields(t *testing.T) {
	data, err := configFS.ReadFile("configs/mcp.json")
	if err != nil {
		t.Fatalf("failed to read MCP config: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse MCP config: %v", err)
	}

	// Check mcpServers exists and has wetwire entry
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers should be an object")
	}

	wetwire, ok := mcpServers["wetwire"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers should have 'wetwire' entry as object")
	}

	// Check command is set
	if cmd, ok := wetwire["command"].(string); !ok || cmd != "wetwire-aws-mcp" {
		t.Errorf("wetwire.command should be 'wetwire-aws-mcp', got %v", wetwire["command"])
	}
}

func TestEnsureProjectMCPConfig_CreatesFile(t *testing.T) {
	// Create temp directory and change to it
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Ensure config doesn't exist
	mcpPath := filepath.Join(".kiro", "mcp.json")
	if _, err := os.Stat(mcpPath); err == nil {
		t.Fatal("mcp.json should not exist before test")
	}

	// Install config
	installed, err := ensureProjectMCPConfig()
	if err != nil {
		t.Fatalf("ensureProjectMCPConfig failed: %v", err)
	}

	if !installed {
		t.Error("expected installed=true for new installation")
	}

	// Verify file exists
	if _, err := os.Stat(mcpPath); err != nil {
		t.Fatalf("mcp.json should exist after installation: %v", err)
	}

	// Verify content is valid JSON
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("installed mcp.json is not valid JSON: %v", err)
	}
}

func TestEnsureProjectMCPConfig_SkipsExisting(t *testing.T) {
	// Create temp directory and change to it
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create existing config with custom content
	mcpDir := ".kiro"
	mcpPath := filepath.Join(mcpDir, "mcp.json")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatal(err)
	}

	customContent := []byte(`{"custom": "config"}`)
	if err := os.WriteFile(mcpPath, customContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Try to install config
	installed, err := ensureProjectMCPConfig()
	if err != nil {
		t.Fatalf("ensureProjectMCPConfig failed: %v", err)
	}

	if installed {
		t.Error("expected installed=false when file already exists")
	}

	// Verify original content is preserved
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(customContent) {
		t.Error("existing config should not be overwritten")
	}
}

func TestEnsureProjectMCPConfig_CreatesDirectory(t *testing.T) {
	// Create temp directory and change to it
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Verify .kiro directory doesn't exist
	if _, err := os.Stat(".kiro"); err == nil {
		t.Fatal(".kiro directory should not exist before test")
	}

	// Install config
	_, err = ensureProjectMCPConfig()
	if err != nil {
		t.Fatalf("ensureProjectMCPConfig failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(".kiro")
	if err != nil {
		t.Fatalf(".kiro directory should exist after installation: %v", err)
	}

	if !info.IsDir() {
		t.Error(".kiro should be a directory")
	}
}
