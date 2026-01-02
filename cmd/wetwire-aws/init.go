package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [project-name]",
		Short: "Create a new wetwire-aws project",
		Long: `Init creates a new Go project with wetwire-aws configured.

Examples:
    wetwire-aws init myinfra
    wetwire-aws init ./projects/myinfra`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(args[0])
		},
	}
}

func runInit(projectPath string) error {
	// Create project directory
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	// Create infra subdirectory
	infraDir := filepath.Join(projectPath, "infra")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		return fmt.Errorf("creating infra directory: %w", err)
	}

	// Get the module name from the project path
	moduleName := filepath.Base(projectPath)

	// Write go.mod
	goMod := fmt.Sprintf(`module %s

go 1.22

require github.com/lex00/wetwire-aws-go v0.1.0
`, moduleName)

	if err := os.WriteFile(filepath.Join(projectPath, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	// Write main.go
	mainGo := `package main

import (
	"fmt"

	_ "` + moduleName + `/infra" // Register resources
)

func main() {
	fmt.Println("Run: wetwire-aws build ./infra/...")
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "main.go"), []byte(mainGo), 0644); err != nil {
		return fmt.Errorf("writing main.go: %w", err)
	}

	// Write infra/resources.go with common imports
	resourcesGo := `package infra

import (
	// Common AWS services - add/remove as needed
	"github.com/lex00/wetwire-aws-go/s3"
	"github.com/lex00/wetwire-aws-go/iam"
	"github.com/lex00/wetwire-aws-go/ec2"
	"github.com/lex00/wetwire-aws-go/lambda"
	"github.com/lex00/wetwire-aws-go/dynamodb"
	"github.com/lex00/wetwire-aws-go/sqs"
	"github.com/lex00/wetwire-aws-go/sns"
	"github.com/lex00/wetwire-aws-go/apigateway"
)

// Define your infrastructure resources below

// Example bucket - uncomment and modify:
// var MyBucket = s3.Bucket{
//     BucketName: "my-bucket",
// }

// Placeholder imports to prevent unused import errors
// Remove these as you add real resources
var _ = s3.Bucket{}
var _ = iam.Role{}
var _ = ec2.VPC{}
var _ = lambda.Function{}
var _ = dynamodb.Table{}
var _ = sqs.Queue{}
var _ = sns.Topic{}
var _ = apigateway.RestApi{}
`
	if err := os.WriteFile(filepath.Join(infraDir, "resources.go"), []byte(resourcesGo), 0644); err != nil {
		return fmt.Errorf("writing resources.go: %w", err)
	}

	// Write .gitignore
	gitignore := `# Build output
template.json
template.yaml

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`
	if err := os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	fmt.Printf("Created project at %s\n\n", projectPath)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", projectPath)
	fmt.Println("  # Edit infra/resources.go to define your infrastructure")
	fmt.Println("  wetwire-aws build ./infra/...")
	fmt.Println()

	return nil
}
