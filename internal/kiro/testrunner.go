package kiro

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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
// It uses PTY handling via the 'script' command because kiro-cli requires
// a TTY even with --no-interactive mode.
func (r *TestRunner) Run(ctx context.Context, prompt string) (*TestResult, error) {
	// Check if kiro-cli is installed
	if _, err := exec.LookPath("kiro-cli"); err != nil {
		return nil, fmt.Errorf("kiro-cli not found in PATH\n\nInstall Kiro CLI: https://kiro.dev/cli")
	}

	// Use PTY-based execution for reliable kiro-cli interaction
	return r.runWithPTY(ctx, prompt)
}

// runWithPTY executes kiro-cli with a pseudo-terminal using the 'script' command.
// kiro-cli requires a TTY even with --no-interactive, so we use the 'script'
// utility to provide one. This is more robust than using Go's pty module directly.
func (r *TestRunner) runWithPTY(ctx context.Context, prompt string) (*TestResult, error) {
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	// Create temp file for output
	outputFile, err := os.CreateTemp("", "kiro-output-*.txt")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	outputPath := outputFile.Name()
	_ = outputFile.Close()
	defer func() { _ = os.Remove(outputPath) }()

	// Build kiro-cli command string
	kiroArgs := []string{
		"chat",
		"--agent", r.AgentName,
		"--model", "claude-sonnet-4",
		"--no-interactive",
		"--trust-all-tools",
		prompt,
	}

	// Build script command - differs between macOS and Linux
	var scriptCmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// macOS: script -q output_file command args...
		args := append([]string{"-q", outputPath, "kiro-cli"}, kiroArgs...)
		scriptCmd = exec.CommandContext(ctx, "script", args...)
	} else {
		// Linux: script -q -c "command args..." output_file
		cmdStr := "kiro-cli"
		for _, arg := range kiroArgs {
			cmdStr += " " + shellescape(arg)
		}
		scriptCmd = exec.CommandContext(ctx, "script", "-q", "-c", cmdStr, outputPath)
	}

	scriptCmd.Dir = r.WorkDir

	// Connect stdin to /dev/null to prevent blocking
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("opening /dev/null: %w", err)
	}
	defer func() { _ = devNull.Close() }()
	scriptCmd.Stdin = devNull

	// Capture stderr
	var stderrBuf strings.Builder
	scriptCmd.Stderr = &stderrBuf

	// Run command
	startTime := time.Now()
	err = scriptCmd.Run()
	duration := time.Since(startTime)

	// Read output from file
	output, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		output = []byte{}
	}

	// Parse results
	result := &TestResult{
		Duration: duration,
		Output:   string(output),
	}

	// Parse output lines for results
	for _, line := range strings.Split(string(output), "\n") {
		r.parseOutputLine(line, result)
		if r.StreamHandler != nil {
			r.StreamHandler(line + "\n")
		}
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("test timed out after %v", r.Timeout))
		} else {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("kiro-cli error: %v", err))
		}
		if stderrBuf.Len() > 0 {
			result.ErrorMessages = append(result.ErrorMessages, stderrBuf.String())
		}
		return result, nil
	}

	result.Success = true
	return result, nil
}

// shellescape escapes a string for safe use in a shell command.
func shellescape(s string) string {
	// If string contains no special characters, return as-is
	safe := true
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(c < '0' || c > '9') && c != '-' && c != '_' && c != '.' && c != '/' {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	// Otherwise, single-quote it and escape any single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
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
// Note: With PTY-based execution and --no-interactive mode, persona responses
// are included in the initial prompt since kiro-cli runs autonomously.
func (r *TestRunner) runWithResponses(ctx context.Context, prompt string, personaResponses []string) (*TestResult, error) {
	// Build a combined prompt that includes context about expected responses
	combinedPrompt := prompt
	if len(personaResponses) > 0 {
		combinedPrompt += "\n\nWhen making decisions, assume the user would respond with preferences like: " +
			strings.Join(personaResponses[:min(3, len(personaResponses))], "; ")
	}

	return r.runWithPTY(ctx, combinedPrompt)
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
		defer func() { _ = os.Chdir(origDir) }()
	}

	// Ensure Kiro configs are installed
	return EnsureInstalled()
}
