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

	"github.com/lex00/wetwire-aws-go/internal/kiro"
	"github.com/lex00/wetwire-core-go/agent/agents"
	"github.com/lex00/wetwire-core-go/agent/orchestrator"
	"github.com/lex00/wetwire-core-go/agent/results"
	"github.com/spf13/cobra"
)

func newDesignCmd() *cobra.Command {
	var outputDir string
	var maxLintCycles int
	var stream bool
	var provider string
	var mcpServerMode bool

	cmd := &cobra.Command{
		Use:   "design [prompt]",
		Short: "AI-assisted infrastructure design",
		Long: `Start an interactive AI-assisted session to design and generate infrastructure code.

The AI agent will:
1. Ask clarifying questions about your requirements
2. Generate Go code using wetwire-aws patterns
3. Run the linter and fix any issues
4. Build the CloudFormation template

Providers:
    anthropic (default) - Uses Anthropic API directly (requires prompt)
    kiro                - Uses Kiro CLI with wetwire-runner agent

With the Kiro provider, you can omit the prompt and the agent will ask what
you'd like to create. The Anthropic provider requires an initial prompt.

Example:
    wetwire-aws design "Create a serverless API with Lambda and API Gateway"
    wetwire-aws design --provider kiro "Create an S3 bucket with versioning"
    wetwire-aws design --provider kiro`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hidden mode: run as MCP server (used by Kiro internally)
			if mcpServerMode {
				return runMCPServer()
			}

			prompt := strings.Join(args, " ")
			if prompt == "" && provider != "kiro" {
				return fmt.Errorf("prompt is required for the %s provider", provider)
			}
			return runDesign(prompt, outputDir, maxLintCycles, stream, provider)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")
	cmd.Flags().IntVarP(&maxLintCycles, "max-lint-cycles", "l", 3, "Maximum lint/fix cycles")
	cmd.Flags().BoolVarP(&stream, "stream", "s", true, "Stream AI responses")
	cmd.Flags().StringVar(&provider, "provider", "anthropic", "AI provider: 'anthropic' or 'kiro'")
	cmd.Flags().BoolVar(&mcpServerMode, "mcp-server", false, "Run as MCP server (internal use)")
	cmd.Flags().MarkHidden("mcp-server")

	return cmd
}

func runDesign(prompt, outputDir string, maxLintCycles int, stream bool, provider string) error {
	switch provider {
	case "kiro":
		return runDesignKiro(prompt, outputDir)
	case "anthropic":
		return runDesignAnthropic(prompt, outputDir, maxLintCycles, stream)
	default:
		return fmt.Errorf("unknown provider: %s (use 'anthropic' or 'kiro')", provider)
	}
}

func runDesignKiro(prompt, outputDir string) error {
	// Change to output directory if specified
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		if err := os.Chdir(outputDir); err != nil {
			return fmt.Errorf("changing to output directory: %w", err)
		}
	}

	fmt.Println("Starting Kiro CLI design session...")
	fmt.Println()

	// Launch Kiro CLI chat (handles config installation internally)
	return kiro.LaunchChat("wetwire-runner", prompt)
}

func runDesignAnthropic(prompt, outputDir string, maxLintCycles int, stream bool) error {
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
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

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
