# ACT-LEAMAS-WITNESS-PROXY-SEED01 Close Report

## Summary

Seeded a local-only HTTP witness proxy that captures bounded request/response metadata as witness evidence for AI-assisted development review loops.

## Files Changed

```
A internal/witness/proxy/proxy.go
A internal/witness/proxy/proxy_test.go
A docs/factory/witness-proxy.md
A docs/close-reports/ACT-LEAMAS-WITNESS-PROXY-SEED01.md
```

## Package Path

```
internal/witness/proxy
```

## Behavior Changed

- Added `proxy.Config`, `proxy.WitnessProxy`, and `proxy.WitnessRecord` types
- Added `proxy.New()` constructor with upstream URL validation
- Added `proxy.Handler()` returning an `http.Handler` for request proxying
- Added `proxy.Records()` returning a defensive copy of captured records
- Added `proxy.Reset()` to clear captured records
- Added bounded in-memory ring buffer (default 100 records, oldest dropped when exceeded)
- Added header sanitization for sensitive headers when `CaptureHeaders: true`
- Default bind address is loopback-only (`127.0.0.1:0`)
- Headers are NOT captured by default
- Body content is NEVER captured

## Commands Run

```bash
go test ./internal/witness/proxy/... -v
go test ./...
go vet ./...
make factorize
make gate
```

## Test Results

All 17 tests pass:
- `TestNewRejectsEmptyUpstreamURL` (3 subtests)
- `TestNewUsesLoopbackDefaultListenAddr`
- `TestProxyForwardsGETRequest`
- `TestProxyPreservesQueryString`
- `TestProxyReturnsUpstreamStatusCode` (9 subtests)
- `TestWitnessRecordCapturesMethodPathStatus`
- `TestRecordsAreBoundedAndOldestDropped`
- `TestRecordsReturnsDefensiveCopy`
- `TestDefaultConfigDoesNotCaptureHeaders`
- `TestCaptureHeadersCapturesNonSensitiveHeaders`
- `TestSensitiveRequestHeadersAreRedacted`
- `TestSensitiveResponseHeadersAreRedacted`
- `TestUpstreamFailureRecordsErrorWitness`
- `TestNoProviderRoutingSingleUpstreamUsed`
- `TestResetClearsRecords`
- `TestResetIsSafeForConcurrentAccess`
- `TestNewRejectsInvalidUpstreamURL`

## Skipped/Deferred

- CLI wiring (deferred to `ACT-LEAMAS-WITNESS-PROXY-CLI01`)
- Boundary verifier (deferred to `ACT-LEAMAS-WITNESS-PROXY-BOUNDARY-VERIFY01`)
- Integration with Hulk cores (runbundle, claimevidence)
- TLS support

## Acceptance Criteria Status

| Criterion | Status |
|-----------|--------|
| local witness proxy package exists | ✅ |
| New() validates upstream config | ✅ |
| proxy forwards requests to exactly one upstream | ✅ |
| proxy captures bounded witness records | ✅ |
| records are exposed through defensive-copy API | ✅ |
| headers are not captured by default | ✅ |
| sensitive headers are redacted when captured | ✅ |
| full bodies are not captured by default | ✅ |
| upstream errors produce witness records | ✅ |
| tests cover forwarding, capture, redaction, bounds, and copy behavior | ✅ |
| docs exist | ✅ |
| close report exists | ✅ |
| go test ./... passes | ✅ |
| go vet ./... passes | ✅ |
| make factorize passes | ✅ |
| make gate passes | ✅ |
| no provider routing is introduced | ✅ |
| no model routing is introduced | ✅ |
| no auth/RBAC/database is introduced | ✅ |
| no witness run-bundle generation is introduced | ✅ |
| no cockpit UI work is started | ✅ |
| no release publishing work is started | ✅ |

## No CLI Wiring Added

CLI wiring was explicitly deferred in this ACT per the task requirements. The proxy can be used programmatically.

## Suggested Commit

```bash
git add internal/witness/proxy docs/factory/witness-proxy.md docs/close-reports/ACT-LEAMAS-WITNESS-PROXY-SEED01.md
git commit -m "ACT-LEAMAS-WITNESS-PROXY-SEED01 add local witness proxy seed"
```

## Next Candidate

`ACT-LEAMAS-WEB-COCKPIT-SEED01`
