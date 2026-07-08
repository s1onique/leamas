#!/usr/bin/env bash
#
# Quality gate for Leamas
# Verifies documentation completeness, code quality, and runs tests if applicable

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

# 1. Check required documentation files
echo "Checking required documentation files..."
check_file "README.md"
check_file "docs/README.md"
check_file "docs/doctrine/README.md"
check_file "docs/doctrine/agent-assisted-development.md"
check_file "docs/adr/0001-local-first-single-binary.md"
check_file "docs/adr/0002-go-only-for-v0.md"
check_file "docs/adr/0003-web-first-local-cockpit.md"
check_file "docs/adr/0004-no-oidc-until-shared-rig.md"
check_file "docs/adr/0005-not-an-llm-gateway.md"
check_file "docs/adr/0006-filesystem-run-bundles.md"
check_file "docs/templates/act.md"
check_file "docs/templates/adr.md"
check_file "docs/templates/close-report.md"
check_file "docs/templates/reviewer-prompt.md"
check_file "docs/epics/.gitkeep"
check_file "docs/acts/.gitkeep"
check_file "scripts/make_targeted_digest.sh"
check_file "scripts/verify_doctrine_inventory.sh"
check_file "scripts/verify_doctrine_agent_contracts.sh"
check_file "scripts/verify_factory_docs.sh"
check_file "scripts/verify_forbidden_patterns.sh"
check_file "scripts/verify_single_language.sh"
check_file "scripts/verify_static_binary_intent.sh"
check_file "scripts/verify_tooling_boundaries.sh"
check_file "scripts/verify_llm_friendliness.sh"
check_file "internal/factory/llmfriendly/check.go"
check_file "docs/factory/tooling-boundaries.md"
check_file "docs/factory/llm-friendliness.md"
check_file "docs/factory/agent-context-files.md"
check_file "AGENTS.md"
check_file ".clinerules/leamas.md"
check_file "scripts/verify_agent_context.sh"
check_file "internal/factory/agentcontext/check.go"
check_file "githooks/pre-push"
check_file "scripts/install_git_hooks.sh"
check_file "internal/factory/githooks/check.go"
check_file "docs/factory/git-safety.md"

echo ""
echo "Checking scripts executability..."
for script in scripts/make_targeted_digest.sh scripts/verify_*.sh; do
    if [[ -f "$script" ]]; then
        if [[ -x "$script" ]]; then
            echo -e "${GREEN}✓${NC} $script is executable"
        else
            echo -e "${YELLOW}ⓘ${NC} $script is not executable (will be skipped)"
        fi
    fi
done

# 2. Run factory verifiers
echo ""
echo "Running factory verifiers..."

# Doctrine agent contracts
if [[ -x scripts/verify_doctrine_agent_contracts.sh ]]; then
    echo ""
    echo "--- Doctrine Agent Contracts ---"
    if scripts/verify_doctrine_agent_contracts.sh; then
        echo -e "${GREEN}✓${NC} Doctrine agent contracts passed"
    else
        echo -e "${RED}✗${NC} Doctrine agent contracts failed"
        failed=1
    fi
fi

# Doctrine inventory
if [[ -x scripts/verify_doctrine_inventory.sh ]]; then
    echo ""
    echo "--- Doctrine Inventory ---"
    if scripts/verify_doctrine_inventory.sh; then
        echo -e "${GREEN}✓${NC} Doctrine inventory passed"
    else
        echo -e "${RED}✗${NC} Doctrine inventory failed"
        failed=1
    fi
fi

# Factory docs
if [[ -x scripts/verify_factory_docs.sh ]]; then
    echo ""
    echo "--- Factory Documentation ---"
    if scripts/verify_factory_docs.sh; then
        echo -e "${GREEN}✓${NC} Factory docs passed"
    else
        echo -e "${RED}✗${NC} Factory docs failed"
        failed=1
    fi
fi

# Forbidden patterns
if [[ -x scripts/verify_forbidden_patterns.sh ]]; then
    echo ""
    echo "--- Forbidden Patterns ---"
    if scripts/verify_forbidden_patterns.sh; then
        echo -e "${GREEN}✓${NC} Forbidden patterns passed"
    else
        echo -e "${RED}✗${NC} Forbidden patterns failed"
        failed=1
    fi
fi

# Single language
if [[ -x scripts/verify_single_language.sh ]]; then
    echo ""
    echo "--- Single Language ---"
    if scripts/verify_single_language.sh; then
        echo -e "${GREEN}✓${NC} Single language passed"
    else
        echo -e "${RED}✗${NC} Single language failed"
        failed=1
    fi
fi

# Static binary intent
if [[ -x scripts/verify_static_binary_intent.sh ]]; then
    echo ""
    echo "--- Static Binary Intent ---"
    if scripts/verify_static_binary_intent.sh; then
        echo -e "${GREEN}✓${NC} Static binary intent passed"
    else
        echo -e "${RED}✗${NC} Static binary intent failed"
        failed=1
    fi
fi

# Tooling boundaries
if [[ -x scripts/verify_tooling_boundaries.sh ]]; then
    echo ""
    echo "--- Tooling Boundaries ---"
    if scripts/verify_tooling_boundaries.sh; then
        echo -e "${GREEN}✓${NC} Tooling boundaries passed"
    else
        echo -e "${RED}✗${NC} Tooling boundaries failed"
        failed=1
    fi
fi

# LLM-friendliness (if Go module exists)
if [[ -f "go.mod" ]] && [[ -x scripts/verify_llm_friendliness.sh ]]; then
    echo ""
    echo "--- LLM-Friendliness ---"
    if scripts/verify_llm_friendliness.sh; then
        echo -e "${GREEN}✓${NC} LLM-friendliness passed"
    else
        echo -e "${RED}✗${NC} LLM-friendliness failed"
        failed=1
    fi
fi

# Agent context files (if Go module exists)
if [[ -f "go.mod" ]] && [[ -x scripts/verify_agent_context.sh ]]; then
    echo ""
    echo "--- Agent Context Files ---"
    if scripts/verify_agent_context.sh; then
        echo -e "${GREEN}✓${NC} Agent context files passed"
    else
        echo -e "${RED}✗${NC} Agent context files failed"
        failed=1
    fi
fi

# Git hooks (if Go module exists)
if [[ -f "go.mod" ]]; then
    echo ""
    echo "--- Git Hooks ---"
    if go run ./cmd/leamas factory verify git-hooks; then
        echo -e "${GREEN}✓${NC} Git hooks passed"
    else
        echo -e "${RED}✗${NC} Git hooks failed"
        failed=1
    fi
fi

# 3. Go toolchain checks (if Go module exists)
echo ""
echo "Checking Go toolchain status..."
if [[ -f "go.mod" ]]; then
    echo -e "${YELLOW}ⓘ${NC} go.mod found - running Go checks"
    echo ""
    
    # go mod tidy
    echo "--- go mod tidy ---"
    if go mod tidy 2>&1; then
        echo -e "${GREEN}✓${NC} go mod tidy passed"
        # Check if go.mod changed (go.sum may not exist yet)
        if [[ -f go.sum ]]; then
            if ! git diff --quiet go.mod go.sum 2>/dev/null; then
                echo -e "${RED}✗${NC} go.mod or go.sum changed after tidy"
                failed=1
            fi
        elif ! git diff --quiet go.mod 2>/dev/null; then
            echo -e "${RED}✗${NC} go.mod changed after tidy"
            failed=1
        fi
    else
        echo -e "${RED}✗${NC} go mod tidy failed"
        failed=1
    fi
    
    # go vet
    echo ""
    echo "--- go vet ---"
    if go vet ./... 2>&1; then
        echo -e "${GREEN}✓${NC} go vet passed"
    else
        echo -e "${RED}✗${NC} go vet found issues"
        failed=1
    fi
    
    # gofmt check
    echo ""
    echo "--- gofmt ---"
    unformatted=$(gofmt -l . 2>/dev/null || true)
    if [[ -z "$unformatted" ]]; then
        echo -e "${GREEN}✓${NC} All Go files are formatted"
    else
        echo -e "${RED}✗${NC} Unformatted files:"
        echo "$unformatted" | while read -r f; do echo "  - $f"; done
        failed=1
    fi
    
    # go test
    echo ""
    echo "--- go test ---"
    if go test ./... 2>&1; then
        echo -e "${GREEN}✓${NC} Tests passed"
    else
        echo -e "${RED}✗${NC} Tests failed"
        failed=1
    fi
    
    # CGO_ENABLED=0 build
    echo ""
    echo "--- Static build ---"
    if CGO_ENABLED=0 go build -trimpath ./cmd/leamas 2>&1; then
        echo -e "${GREEN}✓${NC} Static build successful"
        rm -f leamas 2>/dev/null || true
    else
        echo -e "${RED}✗${NC} Static build failed"
        failed=1
    fi
    
    # Optional: golangci-lint if installed
    if command -v golangci-lint >/dev/null 2>&1; then
        echo ""
        echo "--- golangci-lint (optional) ---"
        if golangci-lint run ./... 2>&1; then
            echo -e "${GREEN}✓${NC} golangci-lint passed"
        else
            echo -e "${RED}✗${NC} golangci-lint found issues"
            failed=1
        fi
    else
        echo ""
        echo -e "${YELLOW}ⓘ${NC} golangci-lint not installed (optional)"
    fi
    
    # Optional: staticcheck if installed
    if command -v staticcheck >/dev/null 2>&1; then
        echo ""
        echo "--- staticcheck (optional) ---"
        if staticcheck ./... 2>&1; then
            echo -e "${GREEN}✓${NC} staticcheck passed"
        else
            echo -e "${RED}✗${NC} staticcheck found issues"
            failed=1
        fi
    else
        echo -e "${YELLOW}ⓘ${NC} staticcheck not installed (optional)"
    fi
    
else
    echo -e "${YELLOW}ⓘ${NC} Go module not initialized yet; skipping Go checks."
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
