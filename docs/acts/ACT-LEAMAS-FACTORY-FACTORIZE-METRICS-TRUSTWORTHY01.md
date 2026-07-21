# ACT-LEAMAS-FACTORY-FACTORIZE-METRICS-TRUSTWORTHY01

## Status
**CLOSED** - Migration complete, commit 36a1864

## Objective
Upgrade factorize metrics from v2 to v3 with trustworthy evidence contracts:
- Explicit run identity binding with validation
- Resource sampler interface for testability
- Subject identity bound to metrics document
- Fail-closed error propagation
- Unique temp file publication

## Completed Work

### Files Changed (commit 36a1864)
- `internal/factory/gate/factorize_metrics_types.go` - v3 types: MetricsSchema, FactorizeMetricsV3, MetricsCheckV3, MetricsCollectionV3, HostIdentity, ResourceSnapshot
- `internal/factory/gate/factorize_metrics.go` - v3 implementation: NewMetricsCollectionV3, executionFingerprintV3, PlatformSampler, FingerprintError
- `internal/factory/gate/factorize_metrics_publication.go` - Unique temp file publication using os.CreateTemp
- `internal/factory/gate/factorize.go` - Updated to use MetricsCollectionV3 and PlatformSampler
- `internal/factory/gate/gate.go` - Updated RunFactorize with lazy metrics initialization
- `internal/factory/gate/factorize_fingerprint_test.go` - Updated to v3 fingerprint function
- `internal/factory/gate/factorize_cache_test.go` - Updated schema test to v3

### Behavior Changed
- `MetricsSchema` upgraded from `"factorize-performance-v2"` to `"factorize-performance-v3"`
- Metrics collection now requires explicit scenario and sequence validation
- PlatformSampler interface enables testable resource sampling
- Fail-closed validation: metrics config errors cause factorize to exit 1
- Unique temp file publication using os.CreateTemp

### Single Authority Achieved
- `MetricsSchema` appears exactly once: `factorize_metrics_types.go:7`
- `FingerprintError` appears exactly once: `factorize_metrics.go:269`
- `MetricsCollectionV3` appears exactly once: `factorize_metrics_types.go:78`
- No v2/v3 parallel declarations

### Verification Results
```
go test ./internal/factory/gate/...     # PASS (0.038s)
CGO_ENABLED=0 go build ./cmd/leamas     # PASS
make gate-fast                          # PASS (11.33s)
gofmt                                   # PASS
go vet ./...                            # PASS
```

### Metrics Disabled Behavior
When `LEAMAS_FACTIZE_METRICS_FILE` is absent:
- No Git subject walk
- No resource sampling
- No configuration parsing
- Normal factorize behavior unchanged

## Not Addressed (Deferred)
- Pre-existing `TestCompareGoSum/multiple_additions` failure in digest package (unrelated to this ACT)
- Six-run controlled measurement matrix (requires separate ACT for evidence collection)
