# ACT-LEAMAS-WITNESS-PROXY-SEED01-R1 Close Report

## Summary

R1 fixes for witness proxy seed addressing capture truth issues.

## Files Changed

```
M internal/witness/proxy/proxy.go
M internal/witness/proxy/proxy_integration_test.go
A internal/witness/proxy/proxy_deep_copy_test.go
A internal/witness/proxy/proxy_capture_test.go
M docs/close-reports/ACT-LEAMAS-WITNESS-PROXY-SEED01.md
A docs/close-reports/ACT-LEAMAS-WITNESS-PROXY-SEED01-R1.md
```

## Behavior Changed

- `ResponseHeaders` now gated behind `CaptureHeaders` (was always captured)
- `Records()` now deep-copies header maps (was shallow copy only)
- `ReverseProxy.ErrorHandler` wired to capture upstream errors
- `CompletedAt` now set from `time.Now()` at completion (was `start`)
- Upstream failure now records non-empty `Error` field with 502 status

## Tests Added

- `TestRecordsDeepCopySliceImmutability`
- `TestRecordsDeepCopyRequestHeadersImmutability`
- `TestRecordsDeepCopyResponseHeadersImmutability`
- `TestResponseHeadersNotCapturedWhenCaptureHeadersFalse`
- `TestCaptureHeadersCapturesBothRequestAndResponse`
- Strengthened `TestUpstreamFailureRecordsErrorWitness` to assert Error non-empty

## Commands Run

```bash
go test ./internal/witness/proxy/... -v
go test ./...
go vet ./...
make factorize
make gate
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

## Test Results

All 26 tests pass (21 original + 5 new).

## Skipped/Deferred

- CLI wiring (deferred to `ACT-LEAMAS-WITNESS-PROXY-CLI01`)
- Integration with Hulk cores (runbundle, claimevidence)

## Acceptance Criteria Status

| Criterion | Status |
|-----------|--------|
| ResponseHeaders gated behind CaptureHeaders | ✅ |
| Records() deep copies header maps | ✅ |
| ReverseProxy.ErrorHandler wired | ✅ |
| CompletedAt reflects actual completion time | ✅ |
| Upstream failure records non-empty Error | ✅ |
| Tests prove immutability | ✅ |
| Close report Files Changed fixed | ✅ |
| LLM-friendly gate passes | ✅ |
| go test ./... passes | ✅ |
| go vet ./... passes | ✅ |
| make factorize passes | ✅ |
| make gate passes | ✅ |
| static build succeeds | ✅ |

## Suggested Commit

```bash
git add docs/close-reports/ACT-LEAMAS-WITNESS-PROXY-SEED01-R1.md
git commit -m "ACT-LEAMAS-WITNESS-PROXY-SEED01-R2 fix close report truth"
```

## Next Candidate

`ACT-LEAMAS-WEB-COCKPIT-SEED01`
