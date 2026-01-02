#!/usr/bin/env bash
#
# Run the same checks as CI locally.
#
# Usage:
#   ./scripts/ci.sh          # Run all checks
#   ./scripts/ci.sh --quick  # Skip code generation test
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PACKAGE_ROOT"

QUICK=""
for arg in "$@"; do
    case $arg in
        --quick)
            QUICK="1"
            ;;
    esac
done

echo "=== wetwire-aws Go CI checks ==="
echo ""

# Verify Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: go is not installed"
    exit 1
fi

# Download dependencies
echo ">>> Downloading dependencies..."
go mod download
echo ""

# Lint (if golangci-lint is available)
if command -v golangci-lint &> /dev/null; then
    echo ">>> Running golangci-lint..."
    golangci-lint run ./...
    echo ""
else
    echo ">>> Skipping lint (golangci-lint not installed)"
    echo "    Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    echo ""
fi

# Tests (exclude examples directory which contains generated test files)
echo ">>> Running tests..."
go test -v -race $(go list ./... | grep -v /examples/)
echo ""

# Build
echo ">>> Building CLI..."
go build -v ./cmd/wetwire-aws
echo ""

# Test codegen (unless --quick)
if [ -z "$QUICK" ]; then
    echo ">>> Testing code generation pipeline..."

    # Fetch spec
    if [ ! -f "specs/CloudFormationResourceSpecification.json" ]; then
        echo "    Fetching CloudFormation spec..."
        go run ./codegen/fetch.go
    fi

    # Parse spec
    echo "    Parsing spec..."
    go run ./codegen/parse.go

    echo "    Code generation pipeline works"
    echo ""
fi

echo "=== All checks passed! ==="
