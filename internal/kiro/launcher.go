package kiro

import (
	"fmt"
	"os"
	"os/exec"
)

// LaunchChat starts an interactive Kiro CLI session with the specified agent.
// The initial prompt is passed to the chat session.
func LaunchChat(agentName, initialPrompt string) error {
	// Check if kiro-cli is installed
	if _, err := exec.LookPath("kiro-cli"); err != nil {
		return fmt.Errorf("kiro-cli not found in PATH\n\nInstall Kiro CLI: https://kiro.dev/cli")
	}

	args := []string{"chat", "--agent", agentName, "--model", "claude-sonnet-4"}
	if initialPrompt != "" {
		args = append(args, initialPrompt)
	}

	cmd := exec.Command("kiro-cli", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
