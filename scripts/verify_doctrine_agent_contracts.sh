#!/usr/bin/env bash
#
# verify_doctrine_agent_contracts.sh
# Ensures all required doctrine files have Agent Contract sections
#
# This is part of the Factory meta-loop (see docs/doctrine/factory-meta-loop.md)
# and implements doctrine for agent-assisted development (see docs/doctrine/agent-assisted-development.md)

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

failed=0
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
cd "$repo_root"

echo "=========================================="
echo "Doctrine Agent Contract Verification"
echo "=========================================="
echo ""

# Required doctrine documents
required=(
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

# Required sections in each doctrine file
sections=(
    "## Agent Contract"
    "### Always"
    "### Never"
    "### Ask / Escalate"
    "### Verification Hooks"
)

echo "Checking required doctrine documents..."
for file in "${required[@]}"; do
    if [[ ! -f "$file" ]]; then
        echo -e "${RED}✗${NC} MISSING: $file"
        failed=1
        continue
    fi

    echo -e "${GREEN}✓${NC} $file exists"
    missing_sections=()
    
    for section in "${sections[@]}"; do
        if ! grep -qF "$section" "$file"; then
            missing_sections+=("$section")
        fi
    done
    
    if [[ ${#missing_sections[@]} -gt 0 ]]; then
        echo -e "  ${RED}✗${NC} Missing sections:"
        for section in "${missing_sections[@]}"; do
            echo -e "       - $section"
        done
        failed=1
    else
        echo -e "  ${GREEN}✓${NC} All Agent Contract sections present"
    fi
done

echo ""
echo "Checking README links..."
if ! grep -qF "agent-assisted-development.md" docs/doctrine/README.md; then
    echo -e "${RED}✗${NC} docs/doctrine/README.md does not link agent-assisted-development.md"
    failed=1
else
    echo -e "${GREEN}✓${NC} README links agent-assisted-development.md"
fi

echo ""
echo "Checking not-a-gateway doctrine boundaries..."
if ! grep -qF "local witness proxy" docs/doctrine/not-a-gateway.md; then
    echo -e "${RED}✗${NC} not-a-gateway doctrine must permit local witness proxy"
    failed=1
else
    echo -e "${GREEN}✓${NC} not-a-gateway permits local witness proxy"
fi

if ! grep -qF "provider router" docs/doctrine/not-a-gateway.md; then
    echo -e "${RED}✗${NC} not-a-gateway doctrine must forbid provider router behavior"
    failed=1
else
    echo -e "${GREEN}✓${NC} not-a-gateway forbids provider router"
fi

if ! grep -qF "model control plane" docs/doctrine/not-a-gateway.md; then
    echo -e "${RED}✗${NC} not-a-gateway doctrine must forbid model control plane behavior"
    failed=1
else
    echo -e "${GREEN}✓${NC} not-a-gateway forbids model control plane"
fi

echo ""
echo "Checking verification-witness observation/evaluation separation..."
if ! grep -qF "Separate observation from evaluation" docs/doctrine/verification-witness.md; then
    echo -e "${RED}✗${NC} verification-witness must require observation/evaluation separation"
    failed=1
else
    echo -e "${GREEN}✓${NC} verification-witness requires observation/evaluation separation"
fi

if ! grep -qF "LLM output as proof" docs/doctrine/verification-witness.md; then
    echo -e "${RED}✗${NC} verification-witness must forbid treating LLM output as proof"
    failed=1
else
    echo -e "${GREEN}✓${NC} verification-witness forbids treating LLM output as proof"
fi

echo ""
echo "=========================================="
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}Doctrine agent contract verification PASSED${NC}"
    exit 0
else
    echo -e "${RED}Doctrine agent contract verification FAILED${NC}"
    exit 1
fi
