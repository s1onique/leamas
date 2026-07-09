# ACT-LEAMAS-WEB-COCKPIT-SEED01 Close Report

## Summary

Added a local-only, read-only web cockpit seed for reviewing Leamas status and static/demo evidence.

## Files Changed

```
A internal/web/cockpit/cockpit.go
A internal/web/cockpit/cockpit_test.go
A internal/web/cockpit/static/index.html
A internal/web/cockpit/static/style.css
A docs/factory/web-cockpit.md
A docs/close-reports/ACT-LEAMAS-WEB-COCKPIT-SEED01.md
```

## Behavior Changed

- New `internal/web/cockpit` package providing:
  - `Config` struct with `ListenAddr` field (default: `127.0.0.1:0`)
  - `Cockpit` struct with `Handler()` returning `http.Handler`
  - `GET /` serves embedded HTML page
  - `GET /api/status` returns JSON status
  - `GET /api/components` returns JSON component list
  - `GET /assets/*` serves embedded static assets
- All API routes return 405 for non-GET methods
- Unknown routes return 404
- No Set-Cookie headers emitted

## Exact Commands Run

```bash
go test ./internal/web/cockpit/... -v
go test ./...
go vet ./...
make factorize
make gate
```

## Honest Results

| Check | Result |
|-------|--------|
| go test ./internal/web/cockpit/... | PASS |
| go test ./... | PASS |
| go vet ./... | PASS |
| make factorize | PASS |
| make gate | PASS |

## Skipped or Deferred

- CLI wiring to `cmd/leamas` (deferred to `ACT-LEAMAS-WEB-COCKPIT-CLI01`)
- Boundary verifier for forbidden imports (deferred to `ACT-LEAMAS-WEB-COCKPIT-BOUNDARY-VERIFY01`)

## Key Constraints Satisfied

- Go-only implementation
- Local-only by default (loopback address)
- Read-only API
- Embedded static assets (no filesystem at runtime)
- No database
- No auth/RBAC/OIDC
- No persistence
- Does not start witness proxy runtime
- Does not route providers or models
- Not a LiteLLM replacement
- Not a gateway/control plane

## Follow-up ACT Candidates

1. `ACT-LEAMAS-WEB-COCKPIT-CLI01` - Wire CLI to start cockpit
2. `ACT-LEAMAS-WITNESS-PROXY-CLI01` - CLI for witness proxy
3. `ACT-LEAMAS-HULK-DOMAIN-BOUNDARY-VERIFY01` - Hulk boundary verification
