#!/usr/bin/env bash
#
# Quality gate for Leamas
# Verifies documentation completeness and runs tests if applicable

set -euo pipefail

# Color output helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

failed=0

check_file() {
    local file="$1"
    if [[ -f "$file" ]]; then
        echo -e "${GREEN}✓${NC} $file"
    else
        echo -e "${RED}✗${NC} MISSING: $file"
        failed=1
    fi
}

echo "=========================================="
echo "Leamas Quality Gate"
echo "=========================================="
echo ""

echo "Checking required documentation files..."
check_file "README.md"
check_file "docs/README.md"
check_file "docs/adr/0001-local-first-single-binary.md"
check_file "docs/adr/0002-go-only-for-v0.md"
check_file "docs/templates/act.md"
check_file "scripts/make_targeted_digest.sh"

echo ""
echo "Checking scripts executability..."
if [[ -x "scripts/make_targeted_digest.sh" ]]; then
    echo -e "${GREEN}✓${NC} scripts/make_targeted_digest.sh is executable"
else
    echo -e "${RED}✗${NC} scripts/make_targeted_digest.sh is not executable (run: chmod +x scripts/make_targeted_digest.sh)"
    failed=1
fi

echo ""
echo "Checking Go module status..."
if [[ -f "go.mod" ]]; then
    echo -e "${YELLOW}ⓘ${NC} go.mod found - will run tests"
    echo ""
    echo "Running go test..."
    if go test ./...; then
        echo -e "${GREEN}✓${NC} Tests passed"
    else
        echo -e "${RED}✗${NC} Tests failed"
        failed=1
    fi
else
    echo -e "${YELLOW}ⓘ${NC} Go module not initialized yet; skipping go test."
fi

echo ""
echo "=========================================="
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}Quality gate PASSED${NC}"
    echo "=========================================="
    exit 0
else
    echo -e "${RED}Quality gate FAILED${NC}"
    echo "=========================================="
    exit 1
fi
