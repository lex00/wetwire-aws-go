// Command wetwire-aws generates CloudFormation templates from Go resource declarations.
//
// Usage:
//
//	wetwire-aws build ./infra/...     Generate CloudFormation template
//	wetwire-aws lint ./infra/...      Check for issues
//	wetwire-aws init myproject        Create new project
//	wetwire-aws version               Show version
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "wetwire-aws",
		Short: "Generate CloudFormation templates from Go",
		Long: `wetwire-aws generates CloudFormation templates from Go resource declarations.

Define your infrastructure using native Go syntax:

    var MyBucket = s3.Bucket{
        BucketName: "my-bucket",
    }

Then generate CloudFormation JSON:

    wetwire-aws build ./infra/...`,
	}

	rootCmd.AddCommand(
		newBuildCmd(),
		newValidateCmd(),
		newListCmd(),
		newLintCmd(),
		newInitCmd(),
		newImportCmd(),
		newDesignCmd(),
		newTestCmd(),
		newVersionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("wetwire-aws %s\n", getVersion())
		},
	}
}
