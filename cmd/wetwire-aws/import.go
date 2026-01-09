package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lex00/wetwire-aws-go/internal/importer"
)

// newImportCmd creates the "import" subcommand for converting YAML templates to Go.
func newImportCmd() *cobra.Command {
	var (
		outputDir   string
		packageName string
		modulePath  string
		singleFile  bool
		noScaffold  bool
	)

	cmd := &cobra.Command{
		Use:   "import <template>",
		Short: "Import CloudFormation template to Go code",
		Long: `Import a CloudFormation YAML/JSON template and generate Go code.

By default, generates a complete project with go.mod, README.md, CLAUDE.md,
.gitignore, and cmd/main.go. Use --no-scaffold to generate only resource files.

Examples:
  # Import a template and generate a complete Go project
  wetwire-aws import template.yaml -o ./myproject

  # Import with custom package name
  wetwire-aws import template.yaml -o ./infra --package mystack

  # Generate only resource files (no go.mod, README, etc.)
  wetwire-aws import template.yaml -o ./infra --no-scaffold

  # Generate a single file instead of a package
  wetwire-aws import template.yaml -o ./infra --single-file`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templatePath := args[0]

			// Parse template
			ir, err := importer.ParseTemplate(templatePath)
			if err != nil {
				return fmt.Errorf("failed to parse template: %w", err)
			}

			// Derive package name if not specified
			if packageName == "" {
				packageName = importer.DerivePackageName(templatePath)
			}

			// Generate code
			files := importer.GenerateCode(ir, packageName)

			// Add scaffold files by default (unless --no-scaffold)
			if !noScaffold && !singleFile {
				if modulePath == "" {
					modulePath = packageName
				}
				scaffoldFiles := importer.GenerateTemplateFiles(packageName, modulePath)
				for filename, content := range scaffoldFiles {
					files[filename] = content
				}
			}

			// Determine output location
			if outputDir == "" {
				outputDir = "."
			}

			// Create output directory
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Write files
			for filename, content := range files {
				outPath := filepath.Join(outputDir, filename)

				// Create parent directory if needed
				if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %w", outPath, err)
				}

				if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", outPath, err)
				}

				fmt.Printf("Generated: %s\n", outPath)
			}

			// Run go mod tidy if scaffold files were generated
			if !noScaffold && !singleFile {
				fmt.Println("\nRunning go mod tidy...")
				tidyCmd := exec.Command("go", "mod", "tidy")
				tidyCmd.Dir = outputDir
				tidyCmd.Stdout = os.Stdout
				tidyCmd.Stderr = os.Stderr
				if err := tidyCmd.Run(); err != nil {
					return fmt.Errorf("go mod tidy failed: %w", err)
				}
			}

			// Print summary
			fmt.Printf("\nImported %d resources, %d parameters, %d outputs\n",
				len(ir.Resources), len(ir.Parameters), len(ir.Outputs))

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: current directory)")
	cmd.Flags().StringVarP(&packageName, "package", "p", "", "Package name (default: derived from template filename)")
	cmd.Flags().StringVar(&modulePath, "module", "", "Go module path (default: same as package name)")
	cmd.Flags().BoolVar(&singleFile, "single-file", false, "Generate a single file instead of a package")
	cmd.Flags().BoolVar(&noScaffold, "no-scaffold", false, "Skip scaffold files (go.mod, README.md, CLAUDE.md, .gitignore)")

	return cmd
}
