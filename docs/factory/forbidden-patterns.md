# Forbidden Patterns

**ACT**: ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01

## Overview

The forbidden-pattern verifier enforces Leamas doctrine by detecting prohibited patterns in production code. This document defines the explicit scan boundary contract.

## Scan Boundary Contract

The verifier implements an explicit contract for what is scanned and what is allowed.

### SCAN

The following directories and files are scanned for forbidden patterns:

| Path | Scope | File Types |
|------|-------|------------|
| `cmd/` | All production code in cmd | `.go` (non-test only) |
| `internal/` (except `internal/factory/`) | All non-factory internal code | `.go` (non-test only) |
| `scripts/` | Shell scripts | All text files |
| `githooks/` | Git hooks | All text files |

### ALLOW (Forbidden-Policy Terms Permitted)

The following directories and file types are explicitly excluded from scanning:

| Path | Reason |
|------|--------|
| `internal/factory/` | Factory verification code must reference forbidden terms |
| `docs/doctrine/` | Doctrine documents discuss policy |
| `docs/adr/` | Architecture decision records |
| `docs/factory/` | Factory documentation |
| `docs/close-reports/` | Close reports |
| `*_test.go` | Test files |
| `testdata/` | Test fixtures |
| `AGENTS.md` | Policy document discussing forbidden terms |
| `.clinerules/` | Policy documents discussing forbidden terms |

## Forbidden Patterns

The following patterns are prohibited in production code:

| Pattern | Description |
|---------|-------------|
| `OIDC\|oidc` | OIDC implementation |
| `OAuth\|oauth` | OAuth implementation |
| `RBAC\|rbac` | RBAC implementation |
| `ABAC\|abac` | ABAC implementation |
| `multi.tenant\|multitenancy\|multi_tenant` | Multi-tenancy |
| `tenant\|tenants` | Tenancy reference |
| `postgres\|postgresql\|mysql\|mariadb\|sqlite` | Database storage |
| `mongodb\|dynamodb\|cassandra` | NoSQL database |
| `redis\|memcached\|cockroachdb` | Cache/database |
| `LiteLLM\|litellm` | LiteLLM/provider routing |
| `database/sql` | database/sql import |
| `github.com/lib/pq` | PostgreSQL driver |
| `github.com/go-sql-driver` | SQL driver |

## Pattern Matching

The verifier uses substring matching with alternation support (e.g., `OIDC|oidc` matches either term).

### Pattern Types

1. **Term alternations**: `pattern1|pattern2|pattern3`
   - Simple substring match for each alternative
   - Example: `LiteLLM|litellm` matches "LiteLLM" OR "litellm"

2. **Word-like patterns**: `multi.tenant|multitenancy`
   - These use `.` as a literal character in "multi.tenant"
   - Not regex; dot is just a character in the pattern

3. **Import patterns**: `database/sql`, `github.com/lib/pq`
   - Full import path matching
   - Flagged as "forbidden_import" kind

## Verification Commands

```bash
# Run via Go verifier
./bin/leamas factory verify forbidden-patterns

# Via Make target
make verify-forbidden

# Via Bash wrapper
./scripts/verify_forbidden_patterns.sh
```

## Exit Codes

- `0` - No forbidden patterns found
- `1` - Forbidden patterns detected

## Why This Boundary?

The boundary exists to:
1. Prevent forbidden patterns in production code
2. Allow Factory verification code to reference policy terms
3. Keep doctrine documents readable without triggering false positives
4. Exclude test files that may test against forbidden patterns

## Implementation

- **Location**: `internal/factory/forbidden/`
- **Language**: Go (verifier, not Bash)
- **Tests**: `internal/factory/forbidden/check_test.go`

## References

- [Go-only Doctrine](../doctrine/go-only.md)
- [Not a Gateway Doctrine](../doctrine/not-a-gateway.md)
- [ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01](../close-reports/ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01.md)
