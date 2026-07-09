# Close Report: ACT-LEAMAS-DOMAIN-BOUNDARY-RUNTIME-SMOKE01

## Summary

Added bounded runtime smoke tests proving that Leamas's CLI runtime boundaries behave as declared. The tests bridge the gap between static boundary verification (import checking) and actual runtime behavior.

## Files Changed

### Added
- `cmd/leamas/runtime_smoke_test.go` - Main runtime smoke tests (311 lines)
- `cmd/leamas/runtime_smoke_validation_test.go` - Validation-specific tests (93 lines)
- `docs/close-reports/ACT-LEAMAS-DOMAIN-BOUNDARY-RUNTIME-SMOKE01.md` - This close report

## Runtime Behaviors Proved

### Help Commands
- `leamas cockpit serve --help` works and mentions loopback constraint
- `leamas witness proxy --help` works and mentions --upstream requirement
- `leamas factory verify domain-boundaries` command exists and runs

### Address Validation
- Cockpit rejects unsafe listen addresses: `0.0.0.0:*`, `:8080`, `[::]:*`, `192.168.1.10:8080`, `10.0.0.1:8080`
- Cockpit accepts safe addresses: `127.0.0.1:*`, `localhost:*`
- Witness proxy rejects unsafe listen addresses (same cases)
- Witness proxy accepts safe addresses (same cases)

### Upstream Validation
- Witness proxy requires `--upstream` flag
- Witness proxy rejects invalid schemes: `ftp://`, `file://`, no scheme, `ws://`
- Witness proxy CLI exits non-zero for ftp:// scheme

### Runtime Behavior
- Cockpit serves `/api/status` on loopback with HTTP 200
- Cockpit `/api/status` returns `mode=local-only`, `read_only=true`, `storage=none`, `auth=none`
- Cockpit emits no Set-Cookie header
- Witness proxy forwards exactly one request to one loopback upstream
- Witness proxy path is preserved through proxy
- Witness proxy does not capture bodies (only has Method, Path, StatusCode in records)

### Bounded Execution
- All subprocess tests use context timeouts (2s-10s)
- All in-process tests use bounded HTTP servers with explicit shutdown
- No external network required
- No Docker required
- No browser required
- No fixed ports (uses dynamic ports with `:0`)

## Tests Added

### Help Smoke Tests
- `TestRuntimeSmokeCockpitHelp`
- `TestRuntimeSmokeWitnessProxyHelp`
- `TestRuntimeSmokeFactoryDomainBoundariesCommand`

### Unsafe Address Rejection Tests
- `TestRuntimeSmokeCockpitRejectsUnsafeListenAddresses` (5 subcases)
- `TestRuntimeSmokeWitnessRejectsUnsafeListenAddresses` (5 subcases)

### Safe Address Acceptance Tests
- `TestRuntimeSmokeCockpitAllowSafeListenAddresses` (4 subcases)
- `TestRuntimeSmokeWitnessAllowSafeListenAddresses` (4 subcases)

### Upstream Validation Tests
- `TestRuntimeSmokeWitnessRequiresUpstream`
- `TestRuntimeSmokeWitnessRejectsInvalidUpstreamScheme` (4 subcases)
- `TestRuntimeSmokeWitnessProxyRejectsFtpScheme` (subprocess)

### Runtime Smoke Tests
- `TestRuntimeSmokeCockpitServesStatusOnLoopback`
- `TestRuntimeSmokeWitnessProxyForwardsToSingleLoopbackUpstream`
- `TestRuntimeSmokeCockpitDoesNotSetCookie`
- `TestRuntimeSmokeWitnessDoesNotCaptureBodies`

### Documentation Tests
- `TestRuntimeSmokeCommandsAreBounded`

## Verification Commands and Results

```bash
# Runtime smoke tests
go test ./cmd/leamas/... -run RuntimeSmoke -v -count=1
# PASS - all 18 runtime smoke tests pass

# Full test suite
go test ./...
# PASS - all packages pass

# Go vet
go vet ./...
# PASS - no issues

# Factory factorize
make factorize
# PASS - all verifiers pass including llm-friendly

# Factory gate
make gate
# PASS - all verifiers + Go toolchain pass
```

## Skipped / Deferred

- Full CLI subprocess start/stop smoke (in-process tests provide sufficient coverage)
- Witness proxy inspect CLI (not part of this ACT's scope)
- Browser auto-open (not implemented in current CLI)
- Multi-upstream proxying (single upstream only by design)

## Hard Stops Honored

- ✅ No Python added
- ✅ No shell-based verification logic
- ✅ No Node/Vite/React/npm/yarn/pnpm added
- ✅ No external network tests
- ✅ No Docker-based tests
- ✅ No browser required
- ✅ No fixed ports
- ✅ No browser auto-open
- ✅ No persistence added
- ✅ No auth/session/cookie behavior
- ✅ No database imports
- ✅ No provider/model routing
- ✅ Witness proxy remains single-upstream proxy, not gateway

## Follow-up Candidates

1. **ACT-LEAMAS-WITNESS-RUN-BUNDLE-SEED01** - After runtime smoke, the remaining architectural center is durable local evidence: run bundles.

2. **ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01** - Optional: Add inspect CLI to view captured records (requires bounded display).

3. **ACT-LEAMAS-WEB-RUN-BUNDLE-LIST01** - Optional: List run bundles from cockpit.

4. **ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01** - Optional: Core run bundle functionality.
