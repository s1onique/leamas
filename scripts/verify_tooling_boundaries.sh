#!/usr/bin/env bash
#
# verify_tooling_boundaries.sh
# Enforces tooling language boundaries for Leamas
#
# Policy:
# - Python is banned everywhere
# - Bash is allowed only as small glue (≤50 meaningful LOC)
# - Existing long Bash scripts are grandfathered until migrated to Go
#
# This is part of ACT-LEAMAS-FACTORY-TOOLING-BOUNDARIES01

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

failed=0
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
cd "$repo_root"

# Count meaningful LOC (non-blank, non-comment)
meaningful_loc() {
  grep -vE '^[[:space:]]*($|#)' "$1" | wc -l | tr -d ' '
}

# Check if path is ignored
is_ignored_path() {
  case "$1" in
    ./.git/*|./vendor/*|./build/*|./bin/*) return 0 ;;
    *) return 1 ;;
  esac
}

# Grandfathered long Bash scripts (temporary, until migrated to Go)
# Note: This verifier itself is over 50 LOC (bootstrapping exception)
is_long_bash_grandfathered() {
  case "$1" in
    # This verifier - bootstrapping exception
    ./scripts/verify_tooling_boundaries.sh) return 0 ;;
    # Factory seed scripts - temporarily grandfathered
    ./scripts/quality_gate.sh) return 0 ;;
    ./scripts/make_targeted_digest.sh) return 0 ;;
    ./scripts/verify_doctrine_agent_contracts.sh) return 0 ;;
    ./scripts/verify_doctrine_inventory.sh) return 0 ;;
    ./scripts/verify_factory_docs.sh) return 0 ;;
    ./scripts/verify_forbidden_patterns.sh) return 0 ;;
    ./scripts/verify_single_language.sh) return 0 ;;
    ./scripts/verify_static_binary_intent.sh) return 0 ;;
    *) return 1 ;;
  esac
}

echo "=========================================="
echo "Tooling Boundary Verification"
echo "=========================================="
echo ""

# 1. Check for Python files (absolute ban)
echo "Checking for Python files (strictly forbidden)..."
python_found=0
while IFS= read -r file; do
  [[ -z "$file" ]] && continue
  if is_ignored_path "$file"; then
    continue
  fi
  echo -e "${RED}✗${NC} Python file forbidden: $file"
  failed=1
  python_found=1
done < <(find . \
  -path ./.git -prune -o \
  -path ./vendor -prune -o \
  -path ./build -prune -o \
  -path ./bin -prune -o \
  -name '*.py' -print 2>/dev/null)

if [[ $python_found -eq 0 ]]; then
  echo -e "${GREEN}✓${NC} No Python files found"
fi

echo ""

# 2. Check Bash scripts for LOC violations
echo "Checking Bash script LOC limits (≤50 meaningful LOC for new scripts)..."
bash_found=0
while IFS= read -r file; do
  [[ -z "$file" ]] && continue
  bash_found=1

  if [[ "$file" != ./scripts/* ]]; then
    echo -e "${RED}✗${NC} Shell script outside scripts/: $file"
    failed=1
    continue
  fi

  if [[ -x "$file" ]]; then
    loc="$(meaningful_loc "$file")"
    if (( loc > 50 )); then
      if is_long_bash_grandfathered "$file"; then
        echo -e "${YELLOW}ⓘ${NC} Grandfathered long Bash: $file (${loc} meaningful LOC)"
        echo -e "${YELLOW}   →${NC} Migration note: See docs/factory/tooling-boundaries.md"
        echo -e "${YELLOW}   →${NC} Target ACT: ACT-LEAMAS-FACTORY-GO-VERIFIERS01"
      else
        echo -e "${RED}✗${NC} Bash script too long: $file (${loc} meaningful LOC > 50)"
        echo -e "${RED}   →${NC} New Bash scripts must stay ≤50 meaningful LOC"
        echo -e "${RED}   →${NC} For substantial automation, implement in Go instead"
        failed=1
      fi
    else
      echo -e "${GREEN}✓${NC} $file (${loc} meaningful LOC)"
    fi
  fi
done < <(find . \
  -path ./.git -prune -o \
  -path ./vendor -prune -o \
  -path ./build -prune -o \
  -path ./bin -prune -o \
  -name '*.sh' -print 2>/dev/null)

if [[ $bash_found -eq 0 ]]; then
  echo -e "${GREEN}✓${NC} No executable Bash scripts found"
fi

echo ""
echo "=========================================="
if [[ $failed -eq 0 ]]; then
  echo -e "${GREEN}Tooling boundary verification PASSED${NC}"
  echo "=========================================="
  exit 0
else
  echo -e "${RED}Tooling boundary verification FAILED${NC}"
  echo "=========================================="
  echo ""
  echo "Policy summary:"
  echo "  • Python is banned everywhere (no exceptions)"
  echo "  • Bash is allowed as small glue only"
  echo "  • New executable Bash scripts must be ≤50 meaningful LOC"
  echo "  • Substantial automation belongs in Go"
  echo ""
  echo "For grandfathered Bash scripts, see:"
  echo "  docs/factory/tooling-boundaries.md"
  echo ""
  echo "Follow-up ACT for migration:"
  echo "  ACT-LEAMAS-FACTORY-GO-VERIFIERS01"
  exit 1
fi
