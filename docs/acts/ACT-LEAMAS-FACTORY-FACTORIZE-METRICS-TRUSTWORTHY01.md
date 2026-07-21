# ACT-LEAMAS-FACTORY-FACTORIZE-METRICS-TRUSTWORTHY01

## Status
**CORRECTION (PARTIAL)** — v3 implementation drafted; single-authority migration and contract tests required

## Objective
Upgrade factorize metrics from v2 to v3 with trustworthy evidence contracts:
- Explicit run identity binding with validation
- Resource sampler interface for testability
- Subject identity bound to metrics document
- Fail-closed error propagation
- Unique temp file publication

## Background
The existing v2 metrics collection has no validation of subject identity, no resource sampling interface for testing, and inconsistent error handling.

## Implementation Notes

### File Structure
```
internal/factory/gate/
  factorize_metrics_types.go    # Types: MetricsSchema, FactorizeMetrics, etc.
  factorize_metrics.go          # MetricsCollection, AddCheck, Finalize
  factorize_fingerprint.go      # FingerprintError, executionFingerprint
  factorize_publication.go     # PublishMetrics (unique temp file)
  platform_sampler.go          # ResourceSampler interface
  subject_identity.go          # Subject identity collection
```

### Single Authority Rule
- `MetricsSchema` must appear exactly once and equal `"factorize-performance-v3"`
- No parallel v2/v3 declarations
- No build tags for production code

### Metrics Disabled Behavior (Lazy Init)
When `LEAMAS_FACTELINE_METRICS_FILE` is absent:
- No Git subject walk
- No resource sampling
- No configuration parsing
- Normal factorize behavior unchanged

## Required Tests
- `TestMetricsSchema_IsExactlyV3`
- `TestMetricsDisabled_DoesNotCollectSubject`
- `TestMetricsDisabled_DoesNotSampleResources`
- `TestMetricsDisabled_PreservesFactorizeBehavior`
- `TestMetricsConfig_RequiresScenarioAndSequence`
- `TestMetricsConfig_RejectsUnknownScenario`
- `TestMetricsPublication_UsesUniqueSiblingTemp`
- `TestMetricsPublicationFailure_FailsFactorize`
- `TestExecutionFingerprint_CurrentContract`

## Acceptance Criteria
- [ ] Package and cmd/leamas build pass
- [ ] MetricsSchema exists exactly once and equals factorize-performance-v3
- [ ] No duplicate package declarations
- [ ] old v2 production schema removed
- [ ] metrics-disabled behavior unchanged and lazy
- [ ] publication failure remains fail-closed
- [ ] make gate-fast stays below 30 seconds and runs no dupcode work
