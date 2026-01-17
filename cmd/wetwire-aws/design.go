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
	"github.com/lex00/wetwire-aws-go/internal/providers/gemini"
	"github.com/lex00/wetwire-aws-go/internal/providers/openai"
	"github.com/lex00/wetwire-core-go/agent/agents"
	"github.com/lex00/wetwire-core-go/agent/orchestrator"
	"github.com/lex00/wetwire-core-go/agent/results"
	"github.com/lex00/wetwire-core-go/providers"
	anthropicprovider "github.com/lex00/wetwire-core-go/providers/anthropic"
	"github.com/spf13/cobra"
)

// newDesignCmd creates the "design" subcommand for AI-assisted infrastructure design.
// It supports both Anthropic API and Kiro CLI providers for interactive code generation.
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
    openai              - Uses OpenAI API (requires prompt)
    gemini              - Uses Google Gemini API (requires prompt)
    kiro                - Uses Kiro CLI with wetwire-aws-runner agent

With the Kiro provider, you can omit the prompt and the agent will ask what
you'd like to create. Other providers require an initial prompt.

Example:
    wetwire-aws design "Create a serverless API with Lambda and API Gateway"
    wetwire-aws design --provider openai "Create an S3 bucket with versioning"
    wetwire-aws design --provider gemini "Create a Lambda function"
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
	cmd.Flags().StringVar(&provider, "provider", "anthropic", "AI provider: 'anthropic', 'openai', 'gemini', or 'kiro'")
	cmd.Flags().BoolVar(&mcpServerMode, "mcp-server", false, "Run as MCP server (internal use)")
	_ = cmd.Flags().MarkHidden("mcp-server")

	return cmd
}

// runDesign starts an AI-assisted design session with the specified provider.
// It dispatches to either Kiro CLI or one of the AI API providers.
func runDesign(prompt, outputDir string, maxLintCycles int, stream bool, provider string) error {
	switch provider {
	case "kiro":
		return runDesignKiro(prompt, outputDir)
	case "anthropic":
		return runDesignWithProvider(prompt, outputDir, maxLintCycles, stream, "anthropic")
	case "openai":
		return runDesignWithProvider(prompt, outputDir, maxLintCycles, stream, "openai")
	case "gemini":
		return runDesignWithProvider(prompt, outputDir, maxLintCycles, stream, "gemini")
	default:
		return fmt.Errorf("unknown provider: %s (use 'anthropic', 'openai', 'gemini', or 'kiro')", provider)
	}
}

// runDesignKiro launches an interactive Kiro CLI session with the wetwire-aws-runner agent.
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
	return kiro.LaunchChat("wetwire-aws-runner", prompt)
}

// runDesignWithProvider runs an interactive design session using the specified AI provider.
// It creates a runner agent that generates code, runs the linter, and fixes issues.
func runDesignWithProvider(prompt, outputDir string, maxLintCycles int, stream bool, providerName string) error {
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

	// Create the AI provider
	var provider providers.Provider
	var err error
	switch providerName {
	case "anthropic":
		provider, err = anthropicprovider.New(anthropicprovider.Config{})
	case "openai":
		provider, err = openai.New(openai.Config{})
	case "gemini":
		provider, err = gemini.New(gemini.Config{})
	default:
		return fmt.Errorf("unsupported provider: %s", providerName)
	}
	if err != nil {
		return fmt.Errorf("creating %s provider: %w", providerName, err)
	}

	// Create session for tracking
	session := results.NewSession("human", "design")

	// Create human developer (reads from stdin)
	reader := bufio.NewReader(os.Stdin)
	developer := orchestrator.NewHumanDeveloper(func() (string, error) {
		return reader.ReadString('\n')
	})

	// Create stream handler if streaming enabled
	var streamHandler providers.StreamHandler
	if stream {
		streamHandler = func(text string) {
			fmt.Print(text)
		}
	}

	// Create runner agent
	runner, err := agents.NewRunnerAgent(agents.RunnerConfig{
		Provider:      provider,
		WorkDir:       outputDir,
		MaxLintCycles: maxLintCycles,
		Session:       session,
		Developer:     developer,
		StreamHandler: streamHandler,
		Domain:        DefaultAWSDomain(),
	})
	if err != nil {
		return fmt.Errorf("creating runner: %w", err)
	}

	fmt.Printf("Starting AI-assisted design session with %s...\n", providerName)
	fmt.Println("The AI will ask questions and generate infrastructure code.")
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	// Run the agent
	if err := runner.Run(ctx, prompt); err != nil {
		return fmt.Errorf("design session failed: %w", err)
	}

	// Print summary
	fmt.Println("\n--- Session Summary ---")
	fmt.Printf("Provider: %s\n", providerName)
	fmt.Printf("Generated files: %d\n", len(runner.GetGeneratedFiles()))
	for _, f := range runner.GetGeneratedFiles() {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Printf("Lint cycles: %d\n", runner.GetLintCycles())
	fmt.Printf("Lint passed: %v\n", runner.LintPassed())

	return nil
}
