// Command design provides AI-assisted infrastructure design.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lex00/wetwire-core-go/agent/agents"
	"github.com/lex00/wetwire-core-go/agent/orchestrator"
	"github.com/lex00/wetwire-core-go/agent/results"
	"github.com/spf13/cobra"
)

func newDesignCmd() *cobra.Command {
	var outputDir string
	var maxLintCycles int
	var stream bool

	cmd := &cobra.Command{
		Use:   "design [prompt]",
		Short: "AI-assisted infrastructure design",
		Long: `Start an interactive AI-assisted session to design and generate infrastructure code.

The AI agent will:
1. Ask clarifying questions about your requirements
2. Generate Go code using wetwire-aws patterns
3. Run the linter and fix any issues
4. Build the CloudFormation template

Example:
    wetwire-aws design "Create a serverless API with Lambda and API Gateway"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.Join(args, " ")
			return runDesign(prompt, outputDir, maxLintCycles, stream)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")
	cmd.Flags().IntVarP(&maxLintCycles, "max-lint-cycles", "l", 3, "Maximum lint/fix cycles")
	cmd.Flags().BoolVarP(&stream, "stream", "s", true, "Stream AI responses")

	return cmd
}

func runDesign(prompt, outputDir string, maxLintCycles int, stream bool) error {
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

	// Create session for tracking
	session := results.NewSession("human", "design")

	// Create human developer (reads from stdin)
	reader := bufio.NewReader(os.Stdin)
	developer := orchestrator.NewHumanDeveloper(func() (string, error) {
		return reader.ReadString('\n')
	})

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

	fmt.Println("Starting AI-assisted design session...")
	fmt.Println("The AI will ask questions and generate infrastructure code.")
	fmt.Println("Press Ctrl+C to stop.\n")

	// Run the agent
	if err := runner.Run(ctx, prompt); err != nil {
		return fmt.Errorf("design session failed: %w", err)
	}

	// Print summary
	fmt.Println("\n--- Session Summary ---")
	fmt.Printf("Generated files: %d\n", len(runner.GetGeneratedFiles()))
	for _, f := range runner.GetGeneratedFiles() {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Printf("Lint cycles: %d\n", runner.GetLintCycles())
	fmt.Printf("Lint passed: %v\n", runner.LintPassed())

	return nil
}
