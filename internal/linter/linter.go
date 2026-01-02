// Package linter provides lint rules for wetwire-aws Go code.
package linter

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Result contains the outcome of linting.
type Result struct {
	Success bool
	Issues  []Issue
}

// Options configures the linter.
type Options struct {
	// Rules to enable. If empty, all rules are enabled.
	EnabledRules []string
	// MaxResources for the FileTooLarge rule.
	MaxResources int
}

// LintFile lints a single Go file.
func LintFile(path string, opts Options) (Result, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return Result{}, err
	}

	rules := getRules(opts)
	var issues []Issue

	for _, rule := range rules {
		ruleIssues := rule.Check(file, fset)
		issues = append(issues, ruleIssues...)
	}

	return Result{
		Success: len(issues) == 0,
		Issues:  issues,
	}, nil
}

// LintPackage lints all Go files in a package directory.
func LintPackage(pkgPath string, opts Options) (Result, error) {
	// Handle ... pattern
	if strings.HasSuffix(pkgPath, "/...") {
		return lintRecursive(strings.TrimSuffix(pkgPath, "/..."), opts)
	}

	// Handle ./... pattern
	if strings.HasSuffix(pkgPath, "...") {
		return lintRecursive(strings.TrimSuffix(pkgPath, "..."), opts)
	}

	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, pkgPath, nil, parser.ParseComments)
	if err != nil {
		return Result{}, err
	}

	rules := getRules(opts)
	var allIssues []Issue

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, rule := range rules {
				issues := rule.Check(file, fset)
				allIssues = append(allIssues, issues...)
			}
		}
	}

	return Result{
		Success: len(allIssues) == 0,
		Issues:  allIssues,
	}, nil
}

// lintRecursive lints all Go packages recursively.
func lintRecursive(root string, opts Options) (Result, error) {
	var allIssues []Issue

	// Clean up root path
	if root == "" || root == "." {
		root = "."
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor directory
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Only process Go files
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			// Skip test files
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			result, err := LintFile(path, opts)
			if err != nil {
				// Log but don't fail on parse errors
				return nil
			}

			allIssues = append(allIssues, result.Issues...)
		}

		return nil
	})

	if err != nil {
		return Result{}, err
	}

	return Result{
		Success: len(allIssues) == 0,
		Issues:  allIssues,
	}, nil
}

// getRules returns the rules to use based on options.
func getRules(opts Options) []Rule {
	all := AllRules()

	// Update MaxResources if specified
	if opts.MaxResources > 0 {
		for i, r := range all {
			if ftl, ok := r.(FileTooLarge); ok {
				ftl.MaxResources = opts.MaxResources
				all[i] = ftl
			}
		}
	}

	// Filter by enabled rules if specified
	if len(opts.EnabledRules) == 0 {
		return all
	}

	enabled := make(map[string]bool)
	for _, id := range opts.EnabledRules {
		enabled[id] = true
	}

	var filtered []Rule
	for _, r := range all {
		if enabled[r.ID()] {
			filtered = append(filtered, r)
		}
	}

	return filtered
}
