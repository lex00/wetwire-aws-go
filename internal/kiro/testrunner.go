package kiro

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corekiro "github.com/lex00/wetwire-core-go/kiro"
	"github.com/lex00/wetwire-core-go/agent/personas"
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
	OutputDir     string
	Timeout       time.Duration
	StreamHandler func(string)
}

// NewTestRunner creates a new Kiro test runner.
func NewTestRunner(outputDir string) *TestRunner {
	return &TestRunner{
		OutputDir: outputDir,
		Timeout:   5 * time.Minute,
	}
}

// EnsureTestEnvironment prepares the output directory for testing.
func (r *TestRunner) EnsureTestEnvironment() error {
	if err := os.MkdirAll(r.OutputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return nil
}

// RunWithPersona executes a test with the specified persona.
func (r *TestRunner) RunWithPersona(ctx context.Context, prompt string, persona personas.Persona) (*TestResult, error) {
	start := time.Now()
	result := &TestResult{}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	// Change to output directory for test
	origDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current directory: %w", err)
	}
	if err := os.Chdir(r.OutputDir); err != nil {
		return nil, fmt.Errorf("change to output directory: %w", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Run test using core kiro package
	output, err := RunTest(ctx, prompt)
	if err != nil {
		// Check if it's just a context timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.ErrorMessages = append(result.ErrorMessages, "test timed out")
		} else {
			result.ErrorMessages = append(result.ErrorMessages, err.Error())
		}
	}
	result.Output = output

	// Parse output
	r.parseOutput(result)

	// Calculate duration
	result.Duration = time.Since(start)

	// Determine overall success
	result.Success = result.LintPassed && result.BuildPassed && len(result.ErrorMessages) == 0

	return result, nil
}

// parseOutput extracts test metrics from the command output.
func (r *TestRunner) parseOutput(result *TestResult) {
	scanner := bufio.NewScanner(strings.NewReader(result.Output))
	for scanner.Scan() {
		line := scanner.Text()
		r.parseOutputLine(result, line)
	}
}

// parseOutputLine processes a single line of output for metrics.
func (r *TestRunner) parseOutputLine(result *TestResult, line string) {
	lineLower := strings.ToLower(line)

	// Check for lint results
	if strings.Contains(lineLower, "wetwire_lint") || strings.Contains(lineLower, "lint") {
		if strings.Contains(lineLower, "success") || strings.Contains(lineLower, "passed") || strings.Contains(lineLower, "valid") {
			result.LintPassed = true
		}
	}

	// Check for build results
	if strings.Contains(lineLower, "wetwire_build") || strings.Contains(lineLower, "build") {
		if strings.Contains(lineLower, "success") || strings.Contains(lineLower, "generated") {
			result.BuildPassed = true
		}
	}

	// Check for file creation
	if strings.Contains(lineLower, "created") || strings.Contains(lineLower, "wrote") {
		parts := strings.Fields(line)
		for _, part := range parts {
			if strings.HasSuffix(part, ".go") || strings.HasSuffix(part, ".yml") {
				result.FilesCreated = append(result.FilesCreated, filepath.Base(part))
			}
		}
	}

	// Check for errors
	if strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "error:") {
		result.ErrorMessages = append(result.ErrorMessages, line)
	}
}

// RunAllPersonas runs the test with all available personas and returns aggregate results.
func (r *TestRunner) RunAllPersonas(ctx context.Context, prompt string) (map[string]*TestResult, error) {
	results := make(map[string]*TestResult)

	for _, name := range personas.Names() {
		persona, err := personas.Get(name)
		if err != nil {
			return nil, fmt.Errorf("get persona %s: %w", name, err)
		}

		// Create persona-specific output directory
		personaDir := filepath.Join(r.OutputDir, name)
		personaRunner := &TestRunner{
			OutputDir:     personaDir,
			Timeout:       r.Timeout,
			StreamHandler: r.StreamHandler,
		}

		if err := personaRunner.EnsureTestEnvironment(); err != nil {
			return nil, fmt.Errorf("prepare %s environment: %w", name, err)
		}

		result, err := personaRunner.RunWithPersona(ctx, prompt, persona)
		if err != nil {
			return nil, fmt.Errorf("run %s persona: %w", name, err)
		}
		results[name] = result
	}

	return results, nil
}

// RunTest runs the Kiro agent in non-interactive test mode.
// Returns the output from the test session.
func RunTest(ctx context.Context, prompt string) (string, error) {
	// Ensure config is installed
	if err := EnsureInstalledWithForce(true); err != nil {
		return "", fmt.Errorf("installing kiro config: %w", err)
	}

	config := NewConfig()
	result, err := corekiro.RunTest(ctx, config, prompt)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}
