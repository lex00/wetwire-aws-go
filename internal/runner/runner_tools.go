// Package runner provides runtime execution of Go packages to extract resource values.
//
// This file contains helper functions for runner execution including:
// - Vendor directory detection
// - Runner subdirectory creation
// - Go binary location
// - go.mod parsing and manipulation
package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hasVendorDir checks if the given directory has a vendor subdirectory.
func hasVendorDir(dir string) bool {
	vendorPath := filepath.Join(dir, "vendor")
	info, err := os.Stat(vendorPath)
	return err == nil && info.IsDir()
}

// shouldUseSubdirRunner returns true if we should use the in-module runner approach.
// This is preferred when vendor/ exists as it allows offline builds.
func shouldUseSubdirRunner(dir string) bool {
	return hasVendorDir(dir)
}

// createRunnerSubdir creates a _wetwire_runner subdirectory in the given module directory.
// Returns the path to the runner directory and a cleanup function.
func createRunnerSubdir(moduleDir string) (string, func(), error) {
	runnerDir := filepath.Join(moduleDir, "_wetwire_runner")

	// Remove any existing runner directory
	_ = os.RemoveAll(runnerDir)

	if err := os.MkdirAll(runnerDir, 0755); err != nil {
		return "", nil, fmt.Errorf("creating runner directory: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(runnerDir)
	}

	return runnerDir, cleanup, nil
}

// findGoBinary locates the Go executable.
// It first checks PATH, then common installation locations.
func findGoBinary() string {
	// Check if go is in PATH
	if path, err := exec.LookPath("go"); err == nil {
		return path
	}

	// Check common locations
	commonPaths := []string{
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
		"/usr/bin/go",
		"/usr/local/bin/go",
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fall back to "go" and let exec fail with a clearer message
	return "go"
}

// goModInfo contains parsed go.mod information.
type goModInfo struct {
	ModulePath string
	GoModDir   string
	Replaces   []string // replace directive lines
	Synthetic  bool     // true if auto-generated (no go.mod found)
}

// findGoModInfo reads go.mod to find the module path and replace directives.
// If no go.mod is found, it returns synthetic module info based on the directory name.
func findGoModInfo(dir string) (*goModInfo, error) {
	originalDir := dir

	// Walk up to find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err == nil {
			info := &goModInfo{GoModDir: dir}

			// Parse go.mod
			lines := strings.Split(string(data), "\n")
			inReplaceBlock := false

			for _, line := range lines {
				trimmed := strings.TrimSpace(line)

				// Look for module directive
				if strings.HasPrefix(trimmed, "module ") {
					info.ModulePath = strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
				}

				// Handle replace block: replace ( ... )
				if trimmed == "replace (" {
					inReplaceBlock = true
					continue
				}
				if inReplaceBlock {
					if trimmed == ")" {
						inReplaceBlock = false
						continue
					}
					// Inside replace block, each line is a replace directive
					if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
						info.Replaces = append(info.Replaces, "replace "+trimmed)
					}
					continue
				}

				// Handle single-line replace directives
				if strings.HasPrefix(trimmed, "replace ") && !strings.HasPrefix(trimmed, "replace (") {
					info.Replaces = append(info.Replaces, trimmed)
				}
			}

			if info.ModulePath == "" {
				return nil, fmt.Errorf("no module directive in go.mod")
			}
			return info, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// No go.mod found - create synthetic module info
			return createSyntheticGoModInfo(originalDir)
		}
		dir = parent
	}
}

// createSyntheticGoModInfo generates module info when no go.mod exists.
// Uses the directory name as the module path.
func createSyntheticGoModInfo(dir string) (*goModInfo, error) {
	// Use directory name as module path
	modulePath := filepath.Base(dir)
	if modulePath == "" || modulePath == "." || modulePath == "/" {
		modulePath = "template"
	}

	return &goModInfo{
		ModulePath: modulePath,
		GoModDir:   dir,
		Synthetic:  true,
	}, nil
}

// resolveReplacePath converts a relative replace path to absolute.
func resolveReplacePath(replaceLine, goModDir string) string {
	// Parse: replace old => new
	parts := strings.Split(replaceLine, " => ")
	if len(parts) != 2 {
		return replaceLine
	}

	oldPart := parts[0]
	newPart := strings.TrimSpace(parts[1])

	// If newPart is a relative path, make it absolute
	if strings.HasPrefix(newPart, ".") || strings.HasPrefix(newPart, "/") {
		if !filepath.IsAbs(newPart) {
			absPath := filepath.Join(goModDir, newPart)
			return oldPart + " => " + absPath
		}
	}
	return replaceLine
}
