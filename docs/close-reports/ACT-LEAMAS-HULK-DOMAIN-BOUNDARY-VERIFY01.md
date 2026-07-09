# ACT-LEAMAS-HULK-DOMAIN-BOUNDARY-VERIFY01 Close Report

## Summary

Added a Go-owned Factory verifier that statically checks import boundaries for Hulk/Witness/Cockpit packages. The verifier protects the declared scope of intentionally constrained internal packages from future drift.

## Files Changed

- `internal/factory/boundary/boundary.go` - Verifier implementation using Go AST parsing
- `internal/factory/boundary/boundary_test.go` - Main test coverage (repo root path fix, missing dir test)
- `internal/factory/boundary/boundary_hulk_test.go` - Hulk-specific boundary tests
- `internal/factory/boundary/boundary_witness_test.go` - Witness proxy-specific boundary tests
- `internal/factory/boundary/boundary_cockpit_test.go` - Cockpit-specific boundary tests
- `cmd/leamas/main.go` - Added `domain-boundaries` to CLI verify commands
- `internal/factory/gate/gate.go` - Added verifier to AllVerifiers()
- `Makefile` - Added `verify-domain-boundaries` target
- `docs/factory/domain-boundaries.md` - Documentation of policies
- `docs/close-reports/ACT-LEAMAS-HULK-DOMAIN-BOUNDARY-VERIFY01.md` - This close report

## Behavior Changed

- New Factory verifier added to check import boundaries for protected packages
- Protected packages:
  - `internal/hulk/runbundle` - Pure domain logic (allows `sort`, `strings`)
  - `internal/hulk/claimevidence` - Pure domain logic (allows `sort`, `strings`)
  - `internal/witness/proxy` - Local HTTP proxy seed (allows `net/http`, `net/http/httputil`)
  - `internal/web/cockpit` - Local read-only web UI seed (allows `embed`, `net/http`)

## Verifier Command

```bash
leamas factory verify domain-boundaries
```

## Make Target Added

```bash
make verify-domain-boundaries
```

## Make Gate Integration

Yes. The verifier is wired into `make gate` and `make factorize` via `AllVerifiers()`.

## Exact Verification Commands and Results

```bash
# Unit tests
go test ./internal/factory/boundary/... -v
# PASS - all 18 tests passed

# Binary build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
# PASS - binary built successfully

# CLI command
./bin/leamas factory verify domain-boundaries
# domain-boundaries verification PASSED

# Make target
make verify-domain-boundaries
# Running domain boundaries verifier...
# domain-boundaries verification PASSED

# Factorize
make factorize
# *** FACTORIZE PASSED ***

# Gate
make gate
# *** GATE PASSED ***
```

## Current Protected Packages

| Package | Purpose | Allowed Imports | Forbidden |
|---------|---------|---------------|----------|
| `internal/hulk/runbundle` | Pure domain logic | `sort`, `strings` | `net/http`, `time`, `database/sql`, etc. |
| `internal/hulk/claimevidence` | Pure domain logic | `sort`, `strings` | `net/http`, `time`, `database/sql`, etc. |
| `internal/witness/proxy` | Local HTTP proxy seed | `net/http`, `net/http/httputil`, `errors`, `sync`, `time`, etc. | `database/sql`, `embed`, provider imports |
| `internal/web/cockpit` | Local read-only web UI seed | `embed`, `net/http`, `encoding/json`, `fmt`, `strings` | `net/http/httputil`, `database/sql`, auth imports |

## Deferred Follow-up Candidates

- `ACT-LEAMAS-WEB-COCKPIT-CLI01` - Web cockpit CLI wiring
- `ACT-LEAMAS-WITNESS-PROXY-CLI01` - Witness proxy CLI wiring
- `ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01` - Expand boundary assertions

## Notes

- Test files (`*_test.go`) are intentionally ignored by this verifier
- Verifier uses `go/parser.ParseFile` for AST parsing (no shell commands)
- Findings are deterministic (sorted by file, import, reason)
- No network access required
