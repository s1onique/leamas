#!/usr/bin/env bash
#
# verify_factory_docs.sh
# Ensures ADRs, ACTs, and templates exist and are properly structured
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

# Required factory documentation
REQUIRED_FACTORY_DOCS=(
    "docs/adr/0001-local-first-single-binary.md"
    "docs/adr/0002-go-only-for-v0.md"
    "docs/adr/0003-web-first-local-cockpit.md"
    "docs/adr/0004-no-oidc-until-shared-rig.md"
    "docs/adr/0005-not-an-llm-gateway.md"
    "docs/adr/0006-filesystem-run-bundles.md"
    "docs/adr/README.md"
    "docs/templates/act.md"
    "docs/templates/adr.md"
    "docs/templates/close-report.md"
    "docs/templates/reviewer-prompt.md"
    "docs/templates/epic.md"
    "docs/acts/.gitkeep"
    "docs/epics/.gitkeep"
    "docs/factory/tooling-boundaries.md"
)

echo "=========================================="
echo "Factory Documentation Verification"
echo "=========================================="
echo ""

echo "Checking required factory documents..."
for file in "${REQUIRED_FACTORY_DOCS[@]}"; do
    if [[ -f "$file" ]]; then
        echo -e "${GREEN}✓${NC} $file"
    else
        echo -e "${RED}✗${NC} MISSING: $file"
        failed=1
    fi
done

echo ""
echo "Checking ADR structure..."
for adr in docs/adr/0*.md; do
    if [[ -f "$adr" && "$(basename "$adr")" != "README.md" ]]; then
        # Check for required ADR sections
        if grep -q "^## Status" "$adr" && grep -q "^## Context" "$adr" && grep -q "^## Decision" "$adr"; then
            echo -e "${GREEN}✓${NC} $(basename "$adr") has valid structure"
        else
            echo -e "${RED}✗${NC} $(basename "$adr") missing required sections (Status, Context, Decision)"
            failed=1
        fi
    fi
done

echo ""
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}Factory documentation verification PASSED${NC}"
    exit 0
else
    echo -e "${RED}Factory documentation verification FAILED${NC}"
    exit 1
fi
