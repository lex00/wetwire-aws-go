package kiro

import (
	"fmt"
	"os"
	"os/exec"
)

// LaunchChat starts an interactive Kiro CLI session with the specified agent.
// The initial prompt is passed to the chat session.
// If no prompt is provided, a default greeting is used to start the conversation.
func LaunchChat(agentName, initialPrompt string) error {
	// Check if kiro-cli is installed
	if _, err := exec.LookPath("kiro-cli"); err != nil {
		return fmt.Errorf("kiro-cli not found in PATH\n\nInstall Kiro CLI: https://kiro.dev/docs/cli/installation/")
	}

	// Force reinstall configs every time to ensure latest agent prompt is used
	if err := EnsureInstalledWithForce(true); err != nil {
		return fmt.Errorf("installing kiro config: %w", err)
	}

	// Build command with --trust-all-tools for smoother experience
	args := []string{"chat", "--agent", agentName, "--model", "claude-sonnet-4", "--trust-all-tools"}

	// Always send an initial message to start the conversation
	// If user provided a prompt, use it; otherwise ask agent to introduce itself
	message := initialPrompt
	if message == "" {
		message = "Hello! I'm ready to design some AWS infrastructure."
	}
	args = append(args, message)

	cmd := exec.Command("kiro-cli", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
