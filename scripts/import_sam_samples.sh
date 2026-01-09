#!/usr/bin/env bash
#
# AWS SAM samples import script for wetwire-aws (Go)
#
# This script tests wetwire-aws's SAM import functionality against official
# AWS SAM template repositories.
#
# Workflow:
# 1. Clone SAM template repositories to temp directory
# 2. Import templates
# 3. Validate each package compiles
# 4. Report final statistics
#
# Usage:
#   ./scripts/import_sam_samples.sh                        # Full import with validation
#   ./scripts/import_sam_samples.sh --clean                # Clean output before running
#   ./scripts/import_sam_samples.sh --template NAME        # Test specific template
#   ./scripts/import_sam_samples.sh --skip-validation      # Skip package validation
#   ./scripts/import_sam_samples.sh --verbose              # Show detailed progress
#

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
SAM_REPOS=(
    "https://github.com/aws/aws-sam-cli-app-templates.git"
    "https://github.com/aws-samples/sessions-with-aws-sam.git"
    "https://github.com/aws-samples/sam-python-crud-sample.git"
)
OUTPUT_DIR="$PROJECT_ROOT/examples/aws-sam-templates"

# Templates to exclude (non-SAM files, config files, etc.)
EXCLUDE_PATTERNS=(
    "*.md"
    "*.txt"
    "*.py"
    "*.js"
    "*.ts"
    "*.sh"
    "*.toml"
    "package.json"
    "package-lock.json"
    "requirements.txt"
    "Makefile"
    "samconfig.toml"
    ".gitignore"
    "cookiecutter.json"
)

cd "$PROJECT_ROOT"

# Helper functions
info() {
    echo -e "${BLUE}==>${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

header() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Parse arguments
CLEAN_OUTPUT=false
SKIP_VALIDATION=false
VERBOSE=false
SINGLE_TEMPLATE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN_OUTPUT=true
            shift
            ;;
        --skip-validation)
            SKIP_VALIDATION=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --template)
            SINGLE_TEMPLATE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Import SAM templates from official AWS repositories"
            echo "and validate them with wetwire-aws (Go)."
            echo ""
            echo "Options:"
            echo "  --clean              Clean examples directory before running"
            echo "  --skip-validation    Skip running each package to validate it works"
            echo "  --verbose, -v        Show detailed progress for each template"
            echo "  --template NAME      Test only a specific template file"
            echo "  --help, -h           Show this help message"
            echo ""
            echo "Source Repositories:"
            for repo in "${SAM_REPOS[@]}"; do
                echo "  - $repo"
            done
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Find Go binary
GO_BIN=$(command -v go 2>/dev/null || echo "/usr/local/go/bin/go")
if [ ! -x "$GO_BIN" ]; then
    error "Go is not installed. Please install Go first:"
    echo "  https://go.dev/doc/install"
    exit 1
fi

# Build the CLI first
header "Building wetwire-aws CLI"
"$GO_BIN" build -o wetwire-aws ./cmd/wetwire-aws
success "Built wetwire-aws CLI"

# Step 1: Optionally clean entire output directory
if [ "$CLEAN_OUTPUT" = true ] && [ -d "$OUTPUT_DIR" ]; then
    header "Cleaning Output Directory"
    rm -rf "$OUTPUT_DIR"
    success "Removed existing $OUTPUT_DIR"
fi
mkdir -p "$OUTPUT_DIR"

# Step 2: Clone SAM template repositories
header "Cloning SAM Template Repositories"
TEMP_DIR=$(mktemp -d)
info "Cloning to temp directory: $TEMP_DIR"

for repo in "${SAM_REPOS[@]}"; do
    repo_name=$(basename "$repo" .git)
    info "Cloning $repo_name..."
    git clone --depth 1 "$repo" "$TEMP_DIR/$repo_name" 2>/dev/null || {
        warn "Failed to clone $repo_name, skipping..."
        continue
    }
    success "Cloned $repo_name"
done

# Cleanup temp directory and binary on exit
cleanup_temp() {
    if [ -n "${TEMP_DIR:-}" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
    if [ -f "$PROJECT_ROOT/wetwire-aws" ]; then
        rm -f "$PROJECT_ROOT/wetwire-aws"
    fi
}
trap cleanup_temp EXIT

# Step 3: Find all SAM templates
header "Discovering SAM Templates"

# Build exclude pattern for find
EXCLUDE_ARGS=""
for pattern in "${EXCLUDE_PATTERNS[@]}"; do
    EXCLUDE_ARGS="$EXCLUDE_ARGS ! -name '$pattern'"
done

# Find all yaml/json templates that contain AWS::Serverless
TEMPLATES=()
while IFS= read -r -d '' template; do
    # Skip non-template files
    if [[ "$template" == *"/node_modules/"* ]] || \
       [[ "$template" == *"/.git/"* ]] || \
       [[ "$template" == *"/venv/"* ]] || \
       [[ "$template" == *"/__pycache__/"* ]]; then
        continue
    fi

    # Skip unrendered cookiecutter templates
    if grep -q '{{cookiecutter' "$template" 2>/dev/null; then
        continue
    fi

    # Check if file contains AWS::Serverless (SAM template indicator)
    if grep -q "AWS::Serverless" "$template" 2>/dev/null || \
       grep -q "Transform.*AWS::Serverless" "$template" 2>/dev/null; then

        # Skip if single template specified and this isn't it
        if [ -n "$SINGLE_TEMPLATE" ]; then
            if [[ "$template" != *"$SINGLE_TEMPLATE"* ]]; then
                continue
            fi
        fi

        TEMPLATES+=("$template")
    fi
done < <(find "$TEMP_DIR" -type f \( -name "*.yaml" -o -name "*.yml" -o -name "template.json" -o -name "template.yaml" \) -print0)

TOTAL_TEMPLATES=${#TEMPLATES[@]}
info "Found $TOTAL_TEMPLATES SAM templates to import"

if [ "$TOTAL_TEMPLATES" -eq 0 ]; then
    if [ -n "$SINGLE_TEMPLATE" ]; then
        error "Template not found: $SINGLE_TEMPLATE"
    else
        error "No SAM templates found in repositories"
    fi
    exit 1
fi

# Step 4: Import templates
header "Importing SAM Templates"

IMPORT_ERRORS_FILE="$OUTPUT_DIR/import_errors.log"
> "$IMPORT_ERRORS_FILE"

IMPORT_OK=0
IMPORT_FAIL=0

for template in "${TEMPLATES[@]}"; do
    # Generate a unique package name from the template path
    rel_path="${template#$TEMP_DIR/}"
    pkg_name=$(echo "$rel_path" | sed 's/[^a-zA-Z0-9]/_/g' | tr '[:upper:]' '[:lower:]')
    # Truncate if too long
    pkg_name="${pkg_name:0:50}"
    pkg_output="$OUTPUT_DIR/$pkg_name"

    # Remove existing to ensure fresh import
    if [ -d "$pkg_output" ]; then
        rm -rf "$pkg_output"
    fi

    if error_output=$("$PROJECT_ROOT/wetwire-aws" import "$template" -o "$pkg_output" 2>&1); then
        IMPORT_OK=$((IMPORT_OK + 1))

        if [ "$VERBOSE" = "true" ]; then
            success "Imported: $rel_path"
        fi
    else
        IMPORT_FAIL=$((IMPORT_FAIL + 1))
        {
            echo "=== $template ==="
            echo "$error_output"
            echo ""
        } >> "$IMPORT_ERRORS_FILE"
        if [ "$VERBOSE" = "true" ]; then
            error "Failed: $rel_path"
        fi
    fi
done

success "Imported: $IMPORT_OK  Failed: $IMPORT_FAIL"

# Step 5: Validate generated Go code
VALIDATION_FAILED=()

if [ "$SKIP_VALIDATION" = false ]; then
    header "Validating Generated Packages"

    for pkg_dir in "$OUTPUT_DIR"/*/; do
        [ -d "$pkg_dir" ] || continue

        pkg_name=$(basename "$pkg_dir")

        # Try to build the package (must cd into it since each has its own go.mod)
        if (cd "$pkg_dir" && "$GO_BIN" build ./... 2>/dev/null); then
            if [ "$VERBOSE" = "true" ]; then
                success "$pkg_name"
            fi
        else
            VALIDATION_FAILED+=("$pkg_name")
            if [ "$VERBOSE" = "true" ]; then
                error "$pkg_name (build failed)"
            fi
        fi
    done

    VALIDATED_COUNT=$((IMPORT_OK - ${#VALIDATION_FAILED[@]}))
    success "Validated: $VALIDATED_COUNT/$IMPORT_OK packages"
fi

# Step 6: Report
header "Summary"

echo ""
success "Total SAM templates found: $TOTAL_TEMPLATES"
success "Successful imports: $IMPORT_OK"
if [ "$IMPORT_FAIL" -gt 0 ]; then
    warn "Failed imports: $IMPORT_FAIL"
fi
if [ ${#VALIDATION_FAILED[@]} -gt 0 ]; then
    warn "Failed validation: ${#VALIDATION_FAILED[@]}"
fi
echo ""

info "Output directory: $OUTPUT_DIR"
if [ -s "$IMPORT_ERRORS_FILE" ]; then
    info "Import errors: $IMPORT_ERRORS_FILE"
fi
echo ""

# Calculate success rate
if [ "$TOTAL_TEMPLATES" -gt 0 ]; then
    SUCCESS_RATE=$((IMPORT_OK * 100 / TOTAL_TEMPLATES))
    info "Success rate: ${SUCCESS_RATE}%"
fi

# Exit with appropriate code
if [ "$IMPORT_FAIL" -gt 0 ] || [ ${#VALIDATION_FAILED[@]} -gt 0 ]; then
    warn "Completed with failures - examples preserved for inspection"
    echo ""
    echo "Note: SAM support is currently in development (Issue #17)."
    echo "Failures are expected until Phase 3 (Integration) is complete."
    exit 1
else
    success "All SAM templates imported and validated successfully!"
    exit 0
fi
