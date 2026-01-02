// Command test runs automated persona-based testing.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

Example:
    wetwire-aws test --persona beginner "Create an S3 bucket with versioning"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := args[0]
			return runTest(prompt, outputDir, personaName, scenario, maxLintCycles, stream)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")
	cmd.Flags().StringVarP(&personaName, "persona", "p", "intermediate", "Persona to use (beginner, intermediate, expert, terse, verbose)")
	cmd.Flags().StringVarP(&scenario, "scenario", "S", "default", "Scenario name for tracking")
	cmd.Flags().IntVarP(&maxLintCycles, "max-lint-cycles", "l", 3, "Maximum lint/fix cycles")
	cmd.Flags().BoolVarP(&stream, "stream", "s", false, "Stream AI responses")

	return cmd
}

func runTest(prompt, outputDir, personaName, scenario string, maxLintCycles int, stream bool) error {
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
