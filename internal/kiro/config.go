package kiro

import (
	"embed"
	"encoding/json"
	"os"

	corekiro "github.com/lex00/wetwire-core-go/kiro"
)

// AgentName is the identifier for the wetwire-aws Kiro agent.
const AgentName = "wetwire-aws-runner"

// MCPCommand is the command to run the MCP server.
const MCPCommand = "wetwire-aws"

//go:embed configs/wetwire-aws-runner.json
var configFS embed.FS

// agentConfig represents the Kiro agent configuration structure.
type agentConfig struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Prompt      string               `json:"prompt"`
	Model       string               `json:"model"`
	MCPServers  map[string]mcpServer `json:"mcpServers"`
	Tools       []string             `json:"tools"`
}

// mcpServer represents an MCP server configuration.
type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd,omitempty"`
}

// NewConfig creates a new Kiro config for the wetwire-aws agent.
func NewConfig() corekiro.Config {
	workDir, _ := os.Getwd()

	// Read embedded agent prompt
	data, err := configFS.ReadFile("configs/wetwire-aws-runner.json")
	if err != nil {
		// Fallback prompt if config can't be read
		return corekiro.Config{
			AgentName:   AgentName,
			AgentPrompt: "You are an AWS infrastructure expert using wetwire-aws to create CloudFormation templates.",
			MCPCommand:  MCPCommand,
			MCPArgs:     []string{"mcp"},
			WorkDir:     workDir,
		}
	}

	var config agentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// Fallback prompt if config can't be parsed
		return corekiro.Config{
			AgentName:   AgentName,
			AgentPrompt: "You are an AWS infrastructure expert using wetwire-aws to create CloudFormation templates.",
			MCPCommand:  MCPCommand,
			MCPArgs:     []string{"mcp"},
			WorkDir:     workDir,
		}
	}

	return corekiro.Config{
		AgentName:   AgentName,
		AgentPrompt: config.Prompt,
		MCPCommand:  MCPCommand,
		MCPArgs:     []string{"mcp"},
		WorkDir:     workDir,
	}
}
