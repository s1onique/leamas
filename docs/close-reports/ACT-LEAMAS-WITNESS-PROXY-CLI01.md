# ACT-LEAMAS-WITNESS-PROXY-CLI01 Close Report

## Summary

Added `leamas witness proxy` command to start the local witness proxy server for capturing bounded request/response metadata as witness evidence.

## Command Added

```bash
leamas witness proxy
```

## Flags Added

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--upstream` | Yes | - | Single target upstream URL (http:// or https://) |
| `--listen` | No | `127.0.0.1:0` | Listen address (loopback only) |
| `--max-records` | No | `100` | Maximum records to retain (0 = package default) |
| `--capture-headers` | No | `false` | Enable header capture with sanitization |

## Loopback Enforcement Behavior

The CLI rejects non-loopback listen addresses:

- **Allowed**: `127.0.0.1:*`, `localhost:*`
- **Rejected**: `0.0.0.0:*`, `[::]:*`, `:*`, private networks (`192.168.*`, `10.*`, `172.16.*`)

Error message for rejected addresses:
```
ERROR: unsafe listen address: 0.0.0.0:8080 (only loopback allowed: 127.0.0.1, localhost)
```

## Upstream Behavior

- `--upstream` is **required**
- Must be http:// or https:// URL
- Single upstream only (no routing tables)
- No provider/model routing

## Files Changed

- `cmd/leamas/main.go` - Added witness command dispatch and usage
- `cmd/leamas/witness.go` - Witness proxy implementation with graceful shutdown
- `cmd/leamas/witness_test.go` - Added tests for CLI parsing and validation
- `docs/factory/witness-proxy.md` - Updated with CLI usage documentation
- `docs/close-reports/ACT-LEAMAS-WITNESS-PROXY-CLI01.md` - Close report

## Verification Commands and Results

```bash
# CLI tests (22 witness tests + 11 cockpit tests + helpers = 39 total)
go test ./cmd/leamas/... -v
# PASS (39 tests)

# Witness proxy package tests (26 tests)
go test ./internal/witness/proxy/... -v
# PASS (26 tests)

# Build test
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
# PASS

# CLI help tests
./bin/leamas --help
./bin/leamas witness --help
./bin/leamas witness proxy --help
# PASS

# Factory factorize
make factorize
# PASS (11 checks)

# Factory gate
make gate
# PASS (gofmt, go vet, go test, static build)

# All Go tests
go test ./...
# PASS

# Go vet
go vet ./...
# PASS
```

## Behavior Summary

- Uses existing `internal/witness/proxy` package
- Pre-binds TCP listener to get actual port when using `:0`
- Prints actual URL: `Leamas witness proxy listening on http://127.0.0.1:54322`
- Prints upstream and header capture settings
- Graceful shutdown on SIGINT/SIGTERM
- Non-zero exit on startup errors
- Headers are not captured by default
- Bodies are never captured
- Records remain in-memory only
- No auth/RBAC/database
- No persistence

## Skipped/Deferred Items

- Witness proxy inspect CLI (`ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01`)
- Browser auto-open (`ACT-LEAMAS-WEB-COCKPIT-BROWSER-OPEN01`)
- Witness proxy boundary verification (`ACT-LEAMAS-WITNESS-PROXY-BOUNDARY-VERIFY01`)
- Domain boundary assertion expansion (`ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01`)
- IPv6 loopback `[::1]` not yet supported in CLI

## Not Implemented (Hard Stops)

- No React/Vite/Node/npm/yarn/pnpm
- No auth/RBAC/OIDC/OAuth/database
- No persistence
- No provider/model routing
- No LiteLLM-compatible APIs
- No cockpit integration
- No release publishing work
