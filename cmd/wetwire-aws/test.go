// Command test runs automated persona-based testing.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lex00/wetwire-aws-go/internal/kiro"
	"github.com/lex00/wetwire-core-go/agent/agents"
	"github.com/lex00/wetwire-core-go/agent/orchestrator"
	"github.com/lex00/wetwire-core-go/agent/personas"
	"github.com/lex00/wetwire-core-go/agent/results"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	var outputDir string
	var personaName string
	var scenario string
	var maxLintCycles int
	var stream bool
	var provider string
	var allPersonas bool

	cmd := &cobra.Command{
		Use:   "test [prompt]",
		Short: "Run automated persona-based testing",
		Long: `Run automated testing with AI personas to evaluate code generation quality.

Available personas:
  - beginner: New to AWS, asks many clarifying questions
  - intermediate: Familiar with AWS basics, asks targeted questions
  - expert: Deep AWS knowledge, asks advanced questions
  - terse: Gives minimal responses
  - verbose: Provides detailed context

Providers:
  - anthropic (default): Uses Anthropic API with wetwire-core-go
  - kiro: Uses Kiro CLI (set SKIP_KIRO_TESTS=1 to skip in CI)

Example:
    wetwire-aws test --persona beginner "Create an S3 bucket with versioning"
    wetwire-aws test --provider kiro "Create an S3 bucket"
    wetwire-aws test --provider kiro --all-personas "Create an S3 bucket"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := args[0]
			if allPersonas {
				return runTestAllPersonas(prompt, outputDir, scenario, maxLintCycles, stream, provider)
			}
			return runTest(prompt, outputDir, personaName, scenario, maxLintCycles, stream, provider)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")
	cmd.Flags().StringVarP(&personaName, "persona", "p", "intermediate", "Persona to use (beginner, intermediate, expert, terse, verbose)")
	cmd.Flags().StringVarP(&scenario, "scenario", "S", "default", "Scenario name for tracking")
	cmd.Flags().IntVarP(&maxLintCycles, "max-lint-cycles", "l", 3, "Maximum lint/fix cycles")
	cmd.Flags().BoolVarP(&stream, "stream", "s", false, "Stream AI responses")
	cmd.Flags().StringVar(&provider, "provider", "anthropic", "AI provider: 'anthropic' or 'kiro'")
	cmd.Flags().BoolVar(&allPersonas, "all-personas", false, "Run test with all personas")

	return cmd
}

func runTest(prompt, outputDir, personaName, scenario string, maxLintCycles int, stream bool, provider string) error {
	switch provider {
	case "kiro":
		return runTestKiro(prompt, outputDir, personaName, scenario, stream)
	case "anthropic":
		return runTestAnthropic(prompt, outputDir, personaName, scenario, maxLintCycles, stream)
	default:
		return fmt.Errorf("unknown provider: %s (use 'anthropic' or 'kiro')", provider)
	}
}

func runTestAllPersonas(prompt, outputDir, scenario string, maxLintCycles int, stream bool, provider string) error {
	personaNames := kiro.AllPersonaNames()
	results := make(map[string]*kiro.TestResult)
	var failed []string

	fmt.Printf("Running tests with all %d personas\n\n", len(personaNames))

	for _, personaName := range personaNames {
		// Create persona-specific output directory
		personaOutputDir := fmt.Sprintf("%s/%s", outputDir, personaName)

		fmt.Printf("=== Running persona: %s ===\n", personaName)

		var err error
		switch provider {
		case "kiro":
			err = runTestKiro(prompt, personaOutputDir, personaName, scenario, stream)
		case "anthropic":
			err = runTestAnthropic(prompt, personaOutputDir, personaName, scenario, maxLintCycles, stream)
		default:
			return fmt.Errorf("unknown provider: %s", provider)
		}

		if err != nil {
			fmt.Printf("Persona %s: FAILED - %v\n\n", personaName, err)
			failed = append(failed, personaName)
		} else {
			fmt.Printf("Persona %s: PASSED\n\n", personaName)
		}
	}

	// Print summary
	fmt.Println("\n=== All Personas Summary ===")
	fmt.Printf("Total: %d\n", len(personaNames))
	fmt.Printf("Passed: %d\n", len(personaNames)-len(failed))
	fmt.Printf("Failed: %d\n", len(failed))
	if len(failed) > 0 {
		fmt.Printf("Failed personas: %v\n", failed)
		return fmt.Errorf("%d personas failed", len(failed))
	}

	_ = results // For future detailed result tracking
	return nil
}

func runTestKiro(prompt, outputDir, personaName, scenario string, stream bool) error {
	// Check if Kiro tests are disabled
	if os.Getenv("SKIP_KIRO_TESTS") == "1" {
		fmt.Println("Skipping Kiro test (SKIP_KIRO_TESTS=1)")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, cleaning up...")
		cancel()
	}()

	// Get persona
	persona, _ := kiro.GetPersona(personaName)

	// Create test runner
	runner := kiro.NewTestRunner(outputDir)

	// Set up streaming if enabled
	if stream {
		runner.StreamHandler = func(text string) {
			fmt.Print(text)
		}
	}

	// Ensure test environment is ready
	if err := runner.EnsureTestEnvironment(); err != nil {
		return fmt.Errorf("preparing test environment: %w", err)
	}

	fmt.Printf("Running Kiro test with persona '%s' and scenario '%s'\n", personaName, scenario)
	fmt.Printf("Prompt: %s\n\n", prompt)

	// Run the test with persona
	result, err := runner.RunWithPersona(ctx, prompt, persona)
	if err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	// Print summary
	fmt.Println("\n--- Test Summary ---")
	fmt.Printf("Provider: kiro\n")
	fmt.Printf("Persona: %s\n", personaName)
	fmt.Printf("Scenario: %s\n", scenario)
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Lint passed: %v\n", result.LintPassed)
	fmt.Printf("Build passed: %v\n", result.BuildPassed)
	fmt.Printf("Files created: %d\n", len(result.FilesCreated))
	for _, f := range result.FilesCreated {
		fmt.Printf("  - %s\n", f)
	}
	if len(result.ErrorMessages) > 0 {
		fmt.Println("Errors:")
		for _, e := range result.ErrorMessages {
			fmt.Printf("  - %s\n", e)
		}
	}

	if !result.Success {
		return fmt.Errorf("test failed")
	}

	return nil
}

func runTestAnthropic(prompt, outputDir, personaName, scenario string, maxLintCycles int, stream bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, cleaning up...")
		cancel()
	}()

	// Get persona
	persona, err := personas.Get(personaName)
	if err != nil {
		return fmt.Errorf("invalid persona: %w", err)
	}

	// Create session for tracking
	session := results.NewSession(personaName, scenario)

	// Create AI developer with persona
	responder := agents.CreateDeveloperResponder("")
	developer := orchestrator.NewAIDeveloper(persona, responder)

	// Create stream handler if streaming enabled
	var streamHandler agents.StreamHandler
	if stream {
		streamHandler = func(text string) {
			fmt.Print(text)
		}
	}

	// Create runner agent
	runner, err := agents.NewRunnerAgent(agents.RunnerConfig{
		WorkDir:       outputDir,
		MaxLintCycles: maxLintCycles,
		Session:       session,
		Developer:     developer,
		StreamHandler: streamHandler,
	})
	if err != nil {
		return fmt.Errorf("creating runner: %w", err)
	}

	fmt.Printf("Running test with persona '%s' and scenario '%s'\n", personaName, scenario)
	fmt.Printf("Prompt: %s\n\n", prompt)

	// Run the agent
	if err := runner.Run(ctx, prompt); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	// Complete session
	session.Complete()

	// Write results
	writer := results.NewWriter(outputDir)
	if err := writer.Write(session); err != nil {
		fmt.Printf("Warning: failed to write results: %v\n", err)
	} else {
		fmt.Printf("\nResults written to: %s\n", outputDir)
	}

	// Print summary
	fmt.Println("\n--- Test Summary ---")
	fmt.Printf("Persona: %s\n", personaName)
	fmt.Printf("Scenario: %s\n", scenario)
	fmt.Printf("Generated files: %d\n", len(runner.GetGeneratedFiles()))
	for _, f := range runner.GetGeneratedFiles() {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Printf("Lint cycles: %d\n", runner.GetLintCycles())
	fmt.Printf("Lint passed: %v\n", runner.LintPassed())
	fmt.Printf("Questions asked: %d\n", len(session.Questions))

	return nil
}
