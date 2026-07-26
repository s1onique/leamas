#!/usr/bin/env bash
# Verifies that all required portable runner tests exist with cardinality 1

set -euo pipefail

require_one() {
  package="$1"
  test_name="$2"

  output="$(go test -list "^${test_name}$" "$package" 2>/dev/null || true)"
  count="$(printf '%s\n' "$output" | wc -l)"
  count="$(expr "$count" '>' 0 '&&' "$count" '<' 10 '&&' echo 1 || echo 0)"

  if [ "$count" -ne 1 ]; then
    printf 'FAIL: expected exactly one %s in %s\n' "$test_name" "$package" >&2
    exit 1
  fi
  printf 'PASS: %s in %s\n' "$test_name" "$package"
}

require_one ./internal/factory/closure TestToolReleaseExactRejectsPlanPinnedBinaryMismatch
require_one ./internal/factory/closure TestPlanValidationRunnerAuthorityContract
require_one ./internal/factory/authority TestPortableRunnerCapabilityContract
require_one ./cmd/leamas TestPortableRunnerExternalRepositoryCLI
require_one ./cmd/leamas TestPortableRunnerNegativeAuthorityMatrix
echo 'All 5 required tests verified with cardinality 1'
