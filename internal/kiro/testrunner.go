package kiro

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TestResult contains the results of a Kiro test run.
type TestResult struct {
	Success       bool
	Output        string
	Duration      time.Duration
	LintPassed    bool
	BuildPassed   bool
	FilesCreated  []string
	ErrorMessages []string
}

// TestRunner runs automated tests through kiro-cli.
type TestRunner struct {
	AgentName     string
	WorkDir       string
	Timeout       time.Duration
	StreamHandler func(string)
}

// NewTestRunner creates a new Kiro test runner.
func NewTestRunner(workDir string) *TestRunner {
	return &TestRunner{
		AgentName: "wetwire-runner",
		WorkDir:   workDir,
		Timeout:   5 * time.Minute,
	}
}

// Run executes a test scenario through kiro-cli.
// It pipes the prompt to kiro-cli and captures the output.
func (r *TestRunner) Run(ctx context.Context, prompt string) (*TestResult, error) {
	// Check if kiro-cli is installed
	if _, err := exec.LookPath("kiro-cli"); err != nil {
		return nil, fmt.Errorf("kiro-cli not found in PATH\n\nInstall Kiro CLI: https://kiro.dev/cli")
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	// Build kiro-cli command
	// --no-interactive: Don't wait for user input
	// --trust-all-tools: Auto-approve tool calls (required for non-interactive)
	args := []string{"chat", "--agent", r.AgentName, "--model", "claude-sonnet-4", "--no-interactive", "--trust-all-tools"}
	cmd := exec.CommandContext(ctx, "kiro-cli", args...)
	cmd.Dir = r.WorkDir

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	// Start command
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting kiro-cli: %w", err)
	}

	// Write prompt to stdin
	go func() {
		defer stdin.Close()
		fmt.Fprintln(stdin, prompt)
		// Send /exit to end the session
		fmt.Fprintln(stdin, "/exit")
	}()

	// Capture output
	result := &TestResult{}
	var outputBuilder strings.Builder

	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")

			if r.StreamHandler != nil {
				r.StreamHandler(line + "\n")
			}

			// Parse output for results
			r.parseOutputLine(line, result)
		}
	}()

	// Read stderr
	stderrOutput, _ := io.ReadAll(stderr)

	// Wait for command to complete
	err = cmd.Wait()
	result.Duration = time.Since(startTime)
	result.Output = outputBuilder.String()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.ErrorMessages = append(result.ErrorMessages, "test timed out")
		} else {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("kiro-cli error: %v", err))
		}
		if len(stderrOutput) > 0 {
			result.ErrorMessages = append(result.ErrorMessages, string(stderrOutput))
		}
		return result, nil
	}

	result.Success = true
	return result, nil
}

// parseOutputLine extracts test results from kiro-cli output.
func (r *TestRunner) parseOutputLine(line string, result *TestResult) {
	line = strings.TrimSpace(line)

	// Check for lint results
	if strings.Contains(line, "wetwire_lint") {
		if strings.Contains(line, "success") || strings.Contains(line, "passed") {
			result.LintPassed = true
		}
	}

	// Check for build results
	if strings.Contains(line, "wetwire_build") {
		if strings.Contains(line, "success") {
			result.BuildPassed = true
		}
	}

	// Check for file creation
	if strings.Contains(line, "Created") || strings.Contains(line, "created") {
		if strings.HasSuffix(line, ".go") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, ".go") {
					result.FilesCreated = append(result.FilesCreated, part)
				}
			}
		}
	}

	// Check for errors
	if strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "error:") {
		result.ErrorMessages = append(result.ErrorMessages, line)
	}
}

// RunWithPersona runs a test with simulated persona responses.
//
// LIMITATION: With --no-interactive mode, kiro-cli runs autonomously without
// waiting for user input. The persona responses are sent but likely ignored.
// This means Kiro tests don't truly simulate different personas - the agent
// runs the same way regardless of persona. For true persona simulation, use
// the Anthropic provider which has proper AI developer integration.
//
// The persona parameter is kept for API compatibility and future improvements.
func (r *TestRunner) RunWithPersona(ctx context.Context, prompt string, persona *Persona) (*TestResult, error) {
	if persona == nil {
		return r.Run(ctx, prompt)
	}
	return r.runWithResponses(ctx, prompt, persona.Responses)
}

// runWithResponses runs a test with the given responses to clarifying questions.
func (r *TestRunner) runWithResponses(ctx context.Context, prompt string, personaResponses []string) (*TestResult, error) {
	// Check if kiro-cli is installed
	if _, err := exec.LookPath("kiro-cli"); err != nil {
		return nil, fmt.Errorf("kiro-cli not found in PATH\n\nInstall Kiro CLI: https://kiro.dev/cli")
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	// Build kiro-cli command
	// --no-interactive: Don't wait for user input
	// --trust-all-tools: Auto-approve tool calls (required for non-interactive)
	args := []string{"chat", "--agent", r.AgentName, "--model", "claude-sonnet-4", "--no-interactive", "--trust-all-tools"}
	cmd := exec.CommandContext(ctx, "kiro-cli", args...)
	cmd.Dir = r.WorkDir

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	// Start command
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting kiro-cli: %w", err)
	}

	// Write inputs to stdin
	go func() {
		defer stdin.Close()
		// Send initial prompt
		fmt.Fprintln(stdin, prompt)

		// Send persona responses
		for _, response := range personaResponses {
			time.Sleep(500 * time.Millisecond) // Give kiro-cli time to process
			fmt.Fprintln(stdin, response)
		}

		// Send /exit to end the session
		time.Sleep(500 * time.Millisecond)
		fmt.Fprintln(stdin, "/exit")
	}()

	// Capture output
	result := &TestResult{}
	var outputBuilder strings.Builder

	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")

			if r.StreamHandler != nil {
				r.StreamHandler(line + "\n")
			}

			r.parseOutputLine(line, result)
		}
	}()

	// Read stderr
	stderrOutput, _ := io.ReadAll(stderr)

	// Wait for command to complete
	err = cmd.Wait()
	result.Duration = time.Since(startTime)
	result.Output = outputBuilder.String()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.ErrorMessages = append(result.ErrorMessages, "test timed out")
		} else {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("kiro-cli error: %v", err))
		}
		if len(stderrOutput) > 0 {
			result.ErrorMessages = append(result.ErrorMessages, string(stderrOutput))
		}
		return result, nil
	}

	result.Success = true
	return result, nil
}

// EnsureTestEnvironment prepares the test environment.
// It ensures configs are installed and the working directory exists.
func (r *TestRunner) EnsureTestEnvironment() error {
	// Create work directory if needed
	if r.WorkDir != "" && r.WorkDir != "." {
		if err := os.MkdirAll(r.WorkDir, 0755); err != nil {
			return fmt.Errorf("creating work directory: %w", err)
		}
	}

	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Change to work directory for config installation
	if r.WorkDir != "" && r.WorkDir != "." {
		if err := os.Chdir(r.WorkDir); err != nil {
			return err
		}
		defer os.Chdir(origDir)
	}

	// Ensure Kiro configs are installed
	return EnsureInstalled()
}
