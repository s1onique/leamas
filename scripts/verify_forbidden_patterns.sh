#!/usr/bin/env bash
#
# verify_forbidden_patterns.sh
# Fail on accidental product drift into enterprise/gateway patterns
#
# FORBIDDEN in production code (allowed ONLY in doctrine/ADR files that explicitly
# describe them as non-goals):
#   - OIDC/OAuth/RBAC implementation
#   - tenants/multi-tenancy
#   - database-backed storage
#   - generic LiteLLM replacement
#   - provider routing/budget tracking
#   - copied foreign-project anchors
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

# Patterns that are forbidden in production code (outside doctrine/adr dirs)
# These are allowed in docs/doctrine/ and docs/adr/ when describing non-goals
FORBIDDEN_PATTERNS=(
    # OIDC/OAuth/RBAC - only allowed in docs/doctrine/ and docs/adr/
    "OIDC\|oidc"
    "OAuth\|oauth"
    "RBAC\|rbac"
    "ABAC\|abac"
    
    # Multi-tenancy
    "multi.tenant\|multitenancy\|multi_tenant"
    "tenant\|tenants"
    
    # Database storage
    "postgres\|postgresql\|mysql\|mariadb\|sqlite"
    "mongodb\|dynamodb\|cassandra"
    "redis\|memcached\|cockroachdb"
    
    # Gateway patterns
    "LiteLLM\|litellm"
    "provider.*rout\|route.*provider"
    "budget.*track\|cost.*track"
    
    # Generic database driver imports (in Go code)
    "database/sql"
    "github.com/lib/pq"
    "github.com/go-sql-driver"
)

echo "=========================================="
echo "Forbidden Pattern Verification"
echo "=========================================="
echo ""

# Find Go source files (excluding test files and vendor)
echo "Scanning Go source files..."
mapfile -t go_files < <(find . -name "*.go" -not -path "./vendor/*" -not -name "*_test.go" 2>/dev/null || true)

if [[ ${#go_files[@]} -eq 0 ]]; then
    echo -e "${YELLOW}ⓘ${NC} No Go source files found to scan"
else
    for pattern in "${FORBIDDEN_PATTERNS[@]}"; do
        matches=$(grep -rlE "$pattern" "${go_files[@]}" 2>/dev/null || true)
        if [[ -n "$matches" ]]; then
            echo -e "${RED}✗${NC} Found forbidden pattern '$pattern' in:"
            while IFS= read -r file; do
                # Skip if file is in doctrine or adr directory
                if [[ "$file" == docs/doctrine/* || "$file" == docs/adr/* ]]; then
                    echo -e "  ${YELLOW}→${NC} $file (allowed in doctrine/adr)"
                else
                    echo -e "  ${RED}✗${NC} $file"
                    failed=1
                fi
            done <<< "$matches"
        fi
    done
fi

# Check for database/sql import in Go files
echo ""
echo "Checking for database driver imports..."
if [[ ${#go_files[@]} -gt 0 ]]; then
    if grep -rE "import.*database/sql|lib/pq|go-sql-driver|mongodb" "${go_files[@]}" 2>/dev/null | grep -v "vendor\|docs/doctrine\|docs/adr" > /dev/null; then
        echo -e "${RED}✗${NC} Found database driver imports in production code"
        failed=1
    fi
fi

echo ""
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}Forbidden pattern verification PASSED${NC}"
    exit 0
else
    echo -e "${RED}Forbidden pattern verification FAILED${NC}"
    echo "Found forbidden patterns in production code"
    echo "These patterns are only allowed in docs/doctrine/ and docs/adr/"
    exit 1
fi
