// Package version provides version information for wetwire-aws-go
// using Go's runtime/debug build info.
package version

import "runtime/debug"

const modulePath = "github.com/lex00/wetwire-aws-go"

// Version returns the module version if available from build info.
// Returns "dev" if version information is not available (local development builds).
func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check if we're the main module
		if info.Main.Path == modulePath && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
		// Check dependencies for when used as a library
		for _, dep := range info.Deps {
			if dep.Path == modulePath {
				return dep.Version
			}
		}
	}
	return "dev"
}

// ModulePath returns the canonical module path.
func ModulePath() string {
	return modulePath
}
