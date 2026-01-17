package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/lex00/wetwire-aws-go/internal/discover"
	"github.com/lex00/wetwire-aws-go/internal/lint"
	"github.com/lex00/wetwire-aws-go/internal/runner"
	"github.com/lex00/wetwire-aws-go/internal/template"
)

// newWatchCmd creates the "watch" subcommand for auto-rebuilding on file changes.
func newWatchCmd() *cobra.Command {
	var (
		lintOnly     bool
		debounce     time.Duration
		outputFormat string
		outputFile   string
	)

	cmd := &cobra.Command{
		Use:   "watch [packages...]",
		Short: "Auto-rebuild on source file changes",
		Long: `Watch monitors source files for changes and automatically rebuilds.

The watch command:
- Monitors the source directory for .go file changes
- Runs lint on each change
- Rebuilds if lint passes (unless --lint-only)
- Debounces rapid changes to avoid excessive rebuilds

Examples:
    wetwire-aws watch ./infra/...
    wetwire-aws watch ./infra/... --lint-only
    wetwire-aws watch ./infra/... --debounce 1s`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(args, watchOptions{
				lintOnly:     lintOnly,
				debounce:     debounce,
				outputFormat: outputFormat,
				outputFile:   outputFile,
			})
		},
	}

	cmd.Flags().BoolVar(&lintOnly, "lint-only", false, "Only run lint, skip build")
	cmd.Flags().DurationVar(&debounce, "debounce", 500*time.Millisecond, "Debounce duration for rapid changes")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "Output format for build: json or yaml")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for build (default: stdout)")

	return cmd
}

type watchOptions struct {
	lintOnly     bool
	debounce     time.Duration
	outputFormat string
	outputFile   string
}

// runWatch monitors source files and runs lint/build on changes.
func runWatch(packages []string, opts watchOptions) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer func() {
		_ = watcher.Close()
	}()

	// Resolve package paths to directories
	dirs, err := resolvePackageDirs(packages)
	if err != nil {
		return fmt.Errorf("failed to resolve packages: %w", err)
	}

	// Add directories to watcher
	for _, dir := range dirs {
		if err := addDirRecursive(watcher, dir); err != nil {
			return fmt.Errorf("failed to watch %s: %w", dir, err)
		}
		fmt.Printf("Watching: %s\n", dir)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initial build
	fmt.Println("Running initial lint/build...")
	runLintAndBuild(packages, opts)

	// Debounce timer
	var debounceTimer *time.Timer
	rebuildChan := make(chan struct{}, 1)

	fmt.Println("\nWatching for changes... (Ctrl+C to stop)")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only watch .go files
			if !strings.HasSuffix(event.Name, ".go") {
				continue
			}

			// Skip generated files
			if strings.HasSuffix(event.Name, "_wetwire_gen.go") {
				continue
			}

			// Only process write/create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Debounce: reset timer on each change
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(opts.debounce, func() {
				select {
				case rebuildChan <- struct{}{}:
				default:
				}
			})

		case <-rebuildChan:
			fmt.Printf("\n[%s] Change detected, rebuilding...\n", time.Now().Format("15:04:05"))
			runLintAndBuild(packages, opts)

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)

		case <-sigChan:
			fmt.Println("\nStopping watch...")
			return nil
		}
	}
}

// resolvePackageDirs converts package patterns to directory paths.
func resolvePackageDirs(packages []string) ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	for _, pkg := range packages {
		// Handle ./... patterns
		pkg = strings.TrimSuffix(pkg, "/...")
		pkg = strings.TrimPrefix(pkg, "./")

		// Convert to absolute path
		absPath, err := filepath.Abs(pkg)
		if err != nil {
			return nil, err
		}

		if !seen[absPath] {
			seen[absPath] = true
			dirs = append(dirs, absPath)
		}
	}

	return dirs, nil
}

// addDirRecursive adds a directory and all subdirectories to the watcher.
func addDirRecursive(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(filepath.Base(path), ".") && path != dir {
				return filepath.SkipDir
			}
			// Skip vendor directory
			if filepath.Base(path) == "vendor" {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
}

// runLintAndBuild runs lint and optionally build on the packages.
func runLintAndBuild(packages []string, opts watchOptions) {
	// Run lint
	lintSuccess := runWatchLint(packages)

	if !lintSuccess {
		fmt.Println("Lint failed, skipping build")
		return
	}

	fmt.Println("Lint passed")

	if opts.lintOnly {
		return
	}

	// Run build
	runWatchBuild(packages, opts)
}

// runWatchLint runs lint and returns true if successful.
func runWatchLint(packages []string) bool {
	// Discover resources
	discoverResult, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Lint error: %v\n", err)
		return false
	}

	// Check for discovery errors
	if len(discoverResult.Errors) > 0 {
		for _, e := range discoverResult.Errors {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		return false
	}

	// Run lint rules
	hasIssues := false
	for _, pkg := range packages {
		lintResult, err := lint.LintPackage(pkg, lint.Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to lint %s: %v\n", pkg, err)
			continue
		}

		for _, issue := range lintResult.Issues {
			hasIssues = true
			if issue.File != "" {
				fmt.Printf("%s:%d:%d: %s: %s [%s]\n",
					issue.File, issue.Line, issue.Column,
					issue.Severity, issue.Message, issue.Rule)
			} else {
				fmt.Printf("%s: %s [%s]\n", issue.Severity, issue.Message, issue.Rule)
			}
		}
	}

	return !hasIssues
}

// runWatchBuild runs build and outputs to stdout or file.
func runWatchBuild(packages []string, opts watchOptions) {
	// Discover resources
	result, err := discover.Discover(discover.Options{
		Packages: packages,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build error: %v\n", err)
		return
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		return
	}

	// Build template
	builder := template.NewBuilderFull(
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)

	// Set VarAttrRefs for recursive AttrRef resolution
	varAttrRefs := make(map[string]template.VarAttrRefInfo)
	for name, info := range result.VarAttrRefs {
		varAttrRefs[name] = template.VarAttrRefInfo{
			AttrRefs: info.AttrRefs,
			VarRefs:  info.VarRefs,
		}
	}
	builder.SetVarAttrRefs(varAttrRefs)

	// Extract values
	values, err := runner.ExtractAll(
		packages[0],
		result.Resources,
		result.Parameters,
		result.Outputs,
		result.Mappings,
		result.Conditions,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build error: %v\n", err)
		return
	}

	// Set values
	for name, props := range values.Resources {
		builder.SetValue(name, props)
	}
	for name, props := range values.Parameters {
		builder.SetValue(name, props)
	}
	for name, props := range values.Outputs {
		builder.SetValue(name, props)
	}
	for name, val := range values.Mappings {
		builder.SetValue(name, val)
	}
	for name, val := range values.Conditions {
		builder.SetValue(name, val)
	}

	tmpl, err := builder.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build error: %v\n", err)
		return
	}

	// Output
	var data []byte
	switch opts.outputFormat {
	case "json":
		data, err = template.ToJSON(tmpl)
	case "yaml":
		data, err = template.ToYAML(tmpl)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", opts.outputFormat)
		return
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Output error: %v\n", err)
		return
	}

	if opts.outputFile == "" {
		fmt.Println("Build successful")
		fmt.Printf("Generated %d resources\n", len(result.Resources))
	} else {
		if err := os.WriteFile(opts.outputFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
			return
		}
		fmt.Printf("Build successful, wrote %s\n", opts.outputFile)
	}
}
