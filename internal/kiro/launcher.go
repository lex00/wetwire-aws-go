package kiro

import (
	"context"
	"fmt"

	corekiro "github.com/lex00/wetwire-core-go/kiro"
)

// LaunchChat launches an interactive Kiro CLI chat session with the wetwire-aws agent.
// It connects stdin/stdout directly to the terminal for interactive use.
func LaunchChat(agentName, initialPrompt string) error {
	// Ensure latest config is installed
	if err := EnsureInstalledWithForce(true); err != nil {
		return fmt.Errorf("installing kiro config: %w", err)
	}

	// Use core kiro package to launch
	config := NewConfig()
	return corekiro.Launch(context.Background(), config, initialPrompt)
}
