#!/usr/bin/env bash
#
# verify_single_language.sh
# Ensures production code is only in Go (per ADR-0002)
#
# This is part of the Factory anti-drift checks

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

failed=0
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
cd "$repo_root"

echo "=========================================="
echo "Single Language Verification (Go Only)"
echo "=========================================="
echo ""

# Directories that should contain only Go code
# These are production code directories
PRODUCTION_DIRS=(
    "cmd/"
    "internal/"
)

# Permitted extensions for production code
PERMITTED_EXTENSIONS=("go")

# Permitted non-Go files in production directories
PERMITTED_NON_GO=(
    # None currently permitted - this is Go-only for v0
)

echo "Checking production directories for non-Go files..."

for dir in "${PRODUCTION_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
        # Find all non-Go files in the directory
        non_go_files=$(find "$dir" -type f \( \
            -name "*.py" -o -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" \
            -o -name "*.java" -o -name "*.rs" -o -name "*.c" -o -name "*.cpp" -o -name "*.h" \
            -o -name "*.rb" -o -name "*.php" -o -name "*.swift" -o -name "*.kt" \
            \) 2>/dev/null || true)
        
        if [[ -n "$non_go_files" ]]; then
            echo -e "${RED}✗${NC} Found non-Go files in $dir:"
            while IFS= read -r file; do
                echo -e "  ${RED}✗${NC} $file"
            done <<< "$non_go_files"
            failed=1
        else
            echo -e "${GREEN}✓${NC} $dir contains only Go files (or is empty)"
        fi
    fi
done

# Check that shell scripts are only in scripts/
echo ""
echo "Checking for shell scripts outside scripts/..."
shell_scripts=$(find . -name "*.sh" -not -path "./scripts/*" -not -path "./vendor/*" 2>/dev/null || true)

if [[ -n "$shell_scripts" ]]; then
    echo -e "${RED}✗${NC} Found shell scripts outside scripts/:"
    while IFS= read -r file; do
        echo -e "  ${RED}✗${NC} $file"
    done <<< "$shell_scripts"
    failed=1
else
    echo -e "${GREEN}✓${NC} Shell scripts only in scripts/ directory"
fi

# Check for Node package files indicating non-Go production code
echo ""
echo "Checking for Node.js package files..."
if [[ -f "package.json" ]]; then
    echo -e "${RED}✗${NC} Found package.json - Node.js not permitted for production code"
    failed=1
fi

if [[ -f "yarn.lock" || -f "pnpm-lock.yaml" || -f "package-lock.json" ]]; then
    echo -e "${RED}✗${NC} Found Node.js lock files"
    failed=1
fi

echo ""
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}Single language verification PASSED${NC}"
    echo "Production code is Go-only as per ADR-0002"
    exit 0
else
    echo -e "${RED}Single language verification FAILED${NC}"
    echo "Found non-Go production code"
    exit 1
fi
