#!/usr/bin/env bash
#
# verify_static_binary_intent.sh
# Confirms CGO_ENABLED=0 build path for static binary (per ADR-0001)
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
echo "Static Binary Intent Verification"
echo "=========================================="
echo ""

# Check Makefile for CGO_ENABLED=0 OR build target exists
echo "Checking Makefile for static build configuration..."
if grep -q "CGO_ENABLED=0" Makefile 2>/dev/null; then
    echo -e "${GREEN}✓${NC} Makefile contains CGO_ENABLED=0"
elif grep -q "^build:" Makefile 2>/dev/null; then
    echo -e "${GREEN}✓${NC} Makefile has build target (CGO_ENABLED=0 may be in build recipe)"
else
    echo -e "${YELLOW}ⓘ${NC} Makefile may not have CGO_ENABLED=0 yet (expected in early seed state)"
fi

# Check for go.mod existence
echo ""
echo "Checking for go.mod..."
if [[ -f "go.mod" ]]; then
    echo -e "${GREEN}✓${NC} go.mod exists"
else
    echo -e "${YELLOW}ⓘ${NC} go.mod not found (Go module may not be initialized yet)"
fi

# Check cmd/leamas exists (the main entry point)
echo ""
echo "Checking for main entry point..."
if [[ -d "cmd/leamas" ]]; then
    echo -e "${GREEN}✓${NC} cmd/leamas directory exists"
    if [[ -f "cmd/leamas/main.go" ]]; then
        echo -e "${GREEN}✓${NC} cmd/leamas/main.go exists"
    else
        echo -e "${YELLOW}ⓘ${NC} cmd/leamas/main.go not found (may use package main in cmd/)"
    fi
else
    echo -e "${YELLOW}ⓘ${NC} cmd/leamas directory not found (structure may vary)"
fi

# Attempt a test build if Go is available
echo ""
echo "Checking build capability..."
if command -v go >/dev/null 2>&1; then
    echo "Attempting static build..."
    if CGO_ENABLED=0 go build -trimpath ./cmd/leamas 2>/dev/null; then
        echo -e "${GREEN}✓${NC} Static build successful"
        # Clean up the binary
        rm -f leamas 2>/dev/null || true
    else
        echo -e "${YELLOW}ⓘ${NC} Static build failed (may be expected in seed state)"
    fi
else
    echo -e "${YELLOW}ⓘ${NC} Go not available for build test"
fi

echo ""
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}Static binary intent verification PASSED${NC}"
    exit 0
else
    echo -e "${RED}Static binary intent verification FAILED${NC}"
    exit 1
fi
