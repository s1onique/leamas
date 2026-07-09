# ACT-LEAMAS-WEB-COCKPIT-CLI01 Close Report

## Summary

Added `leamas cockpit serve` command to start the local embedded web cockpit server.

## Command Added

```bash
leamas cockpit serve
```

## Flags Added

| Flag | Default | Description |
|------|---------|-------------|
| `--listen` | `127.0.0.1:0` | Listen address (loopback only) |

## Loopback Enforcement Behavior

The CLI rejects non-loopback listen addresses:

- **Allowed**: `127.0.0.1:*`, `localhost:*`
- **Rejected**: `0.0.0.0:*`, `[::]:*`, `:*`, IPv6 addresses

Error message for rejected addresses:
```
ERROR: unsafe listen address: 0.0.0.0:8080 (only loopback allowed: 127.0.0.1, localhost)
```

## Files Changed

- `cmd/leamas/main.go` - Added cockpit command handling and serve function
- `cmd/leamas/cockpit_test.go` - Added tests for CLI parsing and validation
- `docs/factory/web-cockpit.md` - Updated with CLI usage documentation

## Verification Commands and Results

```bash
# Build test
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
# PASSED

# CLI tests
go test ./cmd/leamas/... -v
# PASSED (11 tests)

# Cockpit package tests
go test ./internal/web/cockpit/... -v
# PASSED (11 tests)

# All Go tests
go test ./...
# PASSED

# Go vet
go vet ./...
# PASSED

# Factory factorize
make factorize
# PASSED

# Factory gate
make gate
# PASSED
```

## Behavior Summary

- Uses existing `internal/web/cockpit` package
- Pre-binds TCP listener to get actual port when using `:0`
- Prints actual URL: `Leamas cockpit listening on http://127.0.0.1:54321`
- Graceful shutdown on SIGINT/SIGTERM
- Non-zero exit on startup errors

## Skipped/Deferred Items

- Browser auto-open (deferred to `ACT-LEAMAS-WEB-COCKPIT-BROWSER-OPEN01`)
- Witness proxy CLI wiring (deferred to `ACT-LEAMAS-WITNESS-PROXY-CLI01`)
- Domain boundary assertion expansion (deferred)

## Not Implemented (Hard Stops)

- No React/Vite/Node/npm/yarn/pnpm
- No auth/RBAC/database
- No persistence
- No witness proxy runtime
- No provider/model routing
- No release publishing work
