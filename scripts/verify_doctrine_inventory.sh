#!/usr/bin/env bash
#
# verify_doctrine_inventory.sh
# Ensures all required doctrine documents exist
#
# This is part of the Factory meta-loop (see docs/doctrine/factory-meta-loop.md)

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

failed=0
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
cd "$repo_root"

# Required doctrine documents
REQUIRED_DOCTRINE_FILES=(
    "docs/doctrine/README.md"
    "docs/doctrine/agent-assisted-development.md"
    "docs/doctrine/local-first.md"
    "docs/doctrine/web-first.md"
    "docs/doctrine/go-only.md"
    "docs/doctrine/single-binary.md"
    "docs/doctrine/no-enterprise-swamp.md"
    "docs/doctrine/not-a-gateway.md"
    "docs/doctrine/verification-witness.md"
    "docs/doctrine/factory-meta-loop.md"
)

echo "=========================================="
echo "Doctrine Inventory Verification"
echo "=========================================="
echo ""

echo "Checking required doctrine documents..."
for file in "${REQUIRED_DOCTRINE_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        echo -e "${GREEN}✓${NC} $file"
    else
        echo -e "${RED}✗${NC} MISSING: $file"
        failed=1
    fi
done

echo ""
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}Doctrine inventory verification PASSED${NC}"
    exit 0
else
    echo -e "${RED}Doctrine inventory verification FAILED${NC}"
    echo "Missing $(("$failed")) required doctrine document(s)"
    exit 1
fi
