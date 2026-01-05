package main

import "runtime/debug"

// version can be set via ldflags: -ldflags "-X main.version=v1.0.0"
// If not set, getVersion() will try to read from build info (go install @version).
var version = ""

// getVersion returns the version string.
// Priority:
// 1. If version was set via ldflags, use that
// 2. If installed via "go install @version", read from build info
// 3. Otherwise return "dev"
func getVersion() string {
	// If version was set via ldflags, use it
	if version != "" {
		return version
	}

	// Try to get version from build info (works with go install @version)
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}

	return "dev"
}
