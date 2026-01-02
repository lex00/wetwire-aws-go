// Command codegen generates Go types from the CloudFormation Resource Specification.
//
// Usage:
//
//	go run ./codegen                    # Generate all resource types
//	go run ./codegen --service s3       # Generate only S3 resources
//	go run ./codegen --dry-run          # Show what would be generated
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lex00/cloudformation-schema-go/spec"
)

var (
	outputDir  = ""
	service    = ""
	dryRun     = false
	forceRegen = false
)

func init() {
	flag.StringVar(&outputDir, "output", "", "Output directory (default: parent of codegen dir)")
	flag.StringVar(&service, "service", "", "Generate only this service (e.g., s3)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be generated without writing files")
	flag.BoolVar(&forceRegen, "force", false, "Force regeneration even if spec hasn't changed")
}

func main() {
	flag.Parse()

	// Determine output directory
	if outputDir == "" {
		// Default to parent directory of codegen
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("getting working directory: %v", err)
		}
		// If we're in codegen dir, go up one level
		if filepath.Base(wd) == "codegen" {
			outputDir = filepath.Dir(wd)
		} else {
			outputDir = wd
		}
	}

	fmt.Printf("Output directory: %s\n", outputDir)

	// Step 1: Fetch the CloudFormation spec
	fmt.Println("Fetching CloudFormation Resource Specification...")
	cfnSpec, err := spec.FetchSpec(&spec.FetchOptions{Force: forceRegen})
	if err != nil {
		log.Fatalf("fetching spec: %v", err)
	}
	fmt.Printf("Spec version: %s\n", cfnSpec.ResourceSpecificationVersion)
	fmt.Printf("Resource types: %d\n", len(cfnSpec.ResourceTypes))
	fmt.Printf("Property types: %d\n", len(cfnSpec.PropertyTypes))

	// Step 2: Parse and organize by service
	fmt.Println("\nParsing specification...")
	services := parseSpec(cfnSpec, service)
	fmt.Printf("Services to generate: %d\n", len(services))

	// Step 3: Generate Go code
	fmt.Println("\nGenerating Go code...")
	stats, err := generateCode(services, outputDir, dryRun)
	if err != nil {
		log.Fatalf("generating code: %v", err)
	}

	// Step 4: Generate property type registry
	fmt.Println("\nGenerating property type registry...")
	if err := generateRegistry(services, outputDir, dryRun); err != nil {
		log.Fatalf("generating registry: %v", err)
	}

	fmt.Printf("\nGeneration complete:\n")
	fmt.Printf("  Services: %d\n", stats.Services)
	fmt.Printf("  Resources: %d\n", stats.Resources)
	fmt.Printf("  Property types: %d\n", stats.PropertyTypes)
	fmt.Printf("  Files written: %d\n", stats.FilesWritten)
}
