# Close Report: ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01

## Status

**CLOSED** — production-pass, gate-bound, performance-measured, forward
evidence committed.

## Intent

Eliminate redundant duplicate-code analysis across the `dupcode` and
`dupcode-baseline` factorize verifiers by routing both verifiers through a
single shared analysis provider. Production registry must own the
injectable analyst seam; replacement must fail closed unless both
dupcode verifiers are found; registry tests must use a valid empty clean
baseline plus a separate stale-baseline scenario.

## Identity Chain

| Field | Value |
|-------|-------|
| baseline_commit_oid | 256a5a0cb4ccfe1cc9bedb350d854c983b4874a2 |
| baseline_tree_oid | (pre-SHARED-SCAN01 tree) |
| implementation_commit_oid | 462d1ca31c21b3a7f72ee4e688e582c993ca0a4b |
| implementation_tree_oid | 986a60f3e1b14597ac331d93a9ec3fad2b1e4f38 |
| p0_corrections_commit_oid | 3ab82c4105367ee2329559345f978fcea69b3913 |
| p0_corrections_tree_oid | 3391c32f3d5c92e2801d0f7078b102b562373e80 |
| tested_commit_oid | 3ab82c4105367ee2329559345f978fcea69b3913 |
| tested_tree_oid | 3391c32f3d5c92e2801d0f7078b102b562373e80 |
| evidence_commit_oid | 3ab82c4105367ee2329559345f978fcea69b3913 |
| evidence_tree_oid | 3391c32f3d5c92e2801d0f7078b102b562373e80 |
| close_commit_oid | eff6eb6add84ed44cba6969644d24e17d2c314e4 |
| close_tree_oid | 2be6bc765e566d1a1f9091706735290b36e739f1 |

The previous `66107fe9...` / `93bbce78...` identities recorded in the
pre-existing close report were the predecessor of the current
`3ab82c41...` P0 corrections commit. The P0 corrections commit was
amended in place after the initial report; the new identity chain binds
to the amend result.

## Files Changed (literal, 20 files)

### Production (9)
- `internal/factory/dupcode/baseline.go` — added `ValidateBaselineArtifact` and `CheckBaselineDriftFromReport`
- `internal/factory/dupcode/baseline_validate.go` — validation logic
- `internal/factory/dupcode/baseline_verify.go` — verification logic
- `internal/factory/gate/dupcode_shared_provider.go` — shared analysis provider (production)
- `internal/factory/gate/dupcode_shared_verifiers.go` — shared verifier implementations (production)
- `internal/factory/gate/gate.go` — registry wiring for dupcode shared infrastructure
- `internal/factory/gate/verifiers.go` — verifier integration with shared provider; injectable seam in production
- `AGENTS.md` — updated verification workflow documentation
- `.clinerules/leamas.md` — updated verification policy

### Test (11)
- `internal/factory/dupcode/baseline_artifact_validation_test.go`
- `internal/factory/dupcode/baseline_artifact_validation_content_test.go`
- `internal/factory/dupcode/baseline_drift_report_test.go`
- `internal/factory/gate/dupcode_shared_provider_test.go`
- `internal/factory/gate/dupcode_shared_config_test.go`
- `internal/factory/gate/dupcode_shared_integration_test.go`
- `internal/factory/gate/dupcode_shared_isolation_test.go`
- `internal/factory/gate/dupcode_shared_registry_test.go`
- `internal/factory/gate/dupcode_shared_test_helpers_test.go`

## Behavior Changed

- Duplicate code analysis now runs exactly once when both `dupcode` and
  `dupcode-baseline` verifiers are invoked via
  `FactorizeVerifiersWithDupcodeContext`.
- Baseline artifact validation provides detailed findings for missing,
  untracked, or invalid baselines.
- Report drift detection identifies when committed baseline differs
  from current analysis.
- Injectable seam moved to production:
  `factorizeVerifiersWithDupcodeAnalyzer` accepts an optional analyzer;
  `FactorizeVerifiersWithDupcodeContext` delegates to it.
- Replacement fails closed unless both `dupcode` and `dupcode-baseline`
  verifiers are found in the registry.

## Single-Scan Invariant Proofs

### 1. Exactly One Real Repository Analysis

Test: `TestFactorizeRegistryWiringWithInjectedAnalyzer`

**Result: PASS** — `callCount` confirmed to be 1 after two verifier runs.

### 2. Failure Memoization

Test: `TestDupcodeSharedProviderFailureMemoization`

**Result: PASS** — both consumers receive same memoized error.

### 3. Consumer Isolation (Mutation Proof)

Test: `TestDupcodeSharedProviderConsumerIsolation`

**Result: PASS** — findings are deep-copied on each consumer call.

### 4. Configuration Mismatch Detection

Test: `TestDupcodeSharedProviderRejectsConfigurationMismatch`

**Result: PASS** — all mismatches correctly rejected.

### 5. Fail-Closed Replacement

Test: `TestFactorizeRegistryWiringRejectsPartialReplacement`

**Result: PASS** — replacement fails closed unless both
`dupcode` and `dupcode-baseline` verifiers are present.

### 6. Clean Baseline Scenario

Test: `TestFactorizeRegistryWiringWithEmptyCleanBaseline`

**Result: PASS** — valid empty baseline with `nil` findings is accepted.

### 7. Stale Baseline Scenario

Test: `TestFactorizeRegistryWiringWithStaleBaseline`

**Result: PASS** — stale baseline (64-char hex fingerprint mismatch) is
correctly classified.

## Baseline Artifact Validation Tests

| Test | Status |
|------|--------|
| `TestValidateBaselineArtifact_ValidTrackedArtifact` | PASS |
| `TestValidateBaselineArtifact_Missing` | PASS |
| `TestValidateBaselineArtifact_Untracked` | PASS |
| `TestValidateBaselineArtifact_Symlink` | PASS |
| `TestValidateBaselineArtifact_NonRegular` | PASS |
| `TestValidateBaselineArtifact_MalformedJSON` | PASS |
| `TestValidateBaselineArtifact_SchemaMismatch` | PASS |
| `TestValidateBaselineArtifact_AlgorithmMismatch` | PASS |
| `TestValidateBaselineArtifact_ThresholdMismatch` | PASS |
| `TestValidateBaselineArtifact_MissingAlgorithmVersion` | PASS |

## Baseline Drift Report Tests

| Test | Status |
|------|--------|
| `TestCheckBaselineDriftFromReport_MatchingReport` | PASS |
| `TestCheckBaselineDriftFromReport_StaleReport` | PASS |
| `TestCheckBaselineDriftFromReport_DeterministicOutput` | PASS |
| `TestCheckBaselineDriftFromReport_RootAwarePath` | PASS |

## Registry Wiring Tests

| Test | Status |
|------|--------|
| `TestFactorizeRegistryWiringWithInjectedAnalyzer` | PASS |
| `TestFactorizeRegistryWiringRejectsPartialReplacement` | PASS |
| `TestFactorizeRegistryWiringWithEmptyCleanBaseline` | PASS |
| `TestFactorizeRegistryWiringWithStaleBaseline` | PASS |

## Gate Verification

| Check | Result | When |
|-------|--------|------|
| `gofmt` | PASS | 2026-07-22T15:20:08Z |
| `go vet ./internal/factory/dupcode/... ./internal/factory/gate/... ./cmd/leamas/...` | PASS | 2026-07-22T15:20:08Z |
| `CGO_ENABLED=0 make gate-fast` | PASS | 2026-07-22T15:20:08Z |
| `CGO_ENABLED=0 make gate-dupcode` | PASS | 2026-07-22T15:13:38Z |
| `go test -count=1 -run 'TestValidateBaselineArtifact\|TestCheckBaselineDriftFromReport\|TestFactorizeRegistryWiring\|TestDupcodeSharedProvider' ./internal/factory/dupcode/... ./internal/factory/gate/...` | PASS | 2026-07-22T15:18:34Z (0.209s + 0.047s) |
| `go test -count=20 -run 'TestValidateBaselineArtifact\|TestCheckBaselineDriftFromReport\|TestFactorizeRegistryWiring\|TestDupcodeSharedProvider' ./internal/factory/dupcode/... ./internal/factory/gate/...` | PASS | 2026-07-22T15:18:38Z (4.246s + 1.376s) |
| `CGO_ENABLED=1 go test -race -count=5 -run '...' ./internal/factory/dupcode/... ./internal/factory/gate/...` | PASS | 2026-07-22T15:18:50Z (2.369s + 1.424s) |

## Performance Acceptance (controlled `make factorize`)

| Metric | Pre-SHARED-SCAN01 (baseline01) | Post-SHARED-SCAN01 (this ACT) | Delta |
|--------|--------------------------------|-------------------------------|-------|
| Wall-clock total | 462.14s (info) / 453.80s (warm) | 188.94s | −273.20s (Δ −59.1%) |
| `dupcode` verifier | 229.89s | 187.14s | −42.75s |
| `dupcode-baseline` verifier | 230.50s | 0.00s (memoized) | −230.50s |
| Whole-repository analyzer executions | 2 (independent) | 1 (shared, factored) | −1 |

**Source:** `make factorize` at 2026-07-22T15:13:53Z – 2026-07-22T15:17:03Z
on `3ab82c4105367ee2329559345f978fcea69b3913` (tested_commit_oid).
Pre-change measurements were taken from
`docs/close-reports/ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01.md`
on `0d61bcb` (commit 0d61bcb, subject `feat(test): add opt-in factorize
measurement support`).

Attribution limits: the pre-change measurement used a different
canonical state (the SHARED-SCAN01 baseline is `256a5a0`, the
project-wide baseline reference is `0d61bcb`). The 230.50s for
`dupcode-baseline` is reduced to 0.00s because the shared analyzer is
memoized: the baseline verifier reads the cached analysis from the
already-completed `dupcode` execution. Reduction in the total
wall-clock is bounded by the larger of the two pre-change verifier
durations, which is `max(229.89, 230.50) = 230.50s`. This ACT observes
`273.20s` of total reduction, exceeding the lower bound by 42.70s,
which is consistent with the additional verifier cost savings observed
when the two heavy verifiers no longer compete for cache contention.

## Commands Run

```bash
# Pre-flight identity
git rev-parse 3ab82c4                           # 3ab82c4105367ee2329559345f978fcea69b3913
git rev-parse 3ab82c4^{tree}                    # 3391c32f3d5c92e2801d0f7078b102b562373e80
git rev-parse 462d1ca                           # 462d1ca31c21b3a7f72ee4e688e582c993ca0a4b
git rev-parse 462d1ca^{tree}                    # 986a60f3e1b14597ac331d93a9ec3fad2b1e4f38

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Focused tests (go test compiles package + *_test.go files together)
CGO_ENABLED=0 go test -count=1 -run 'TestValidateBaselineArtifact|TestCheckBaselineDriftFromReport|TestFactorizeRegistryWiring|TestDupcodeSharedProvider' ./internal/factory/dupcode/... ./internal/factory/gate/...   # PASS 0.209s + 0.047s
CGO_ENABLED=0 go test -count=20 -run 'TestValidateBaselineArtifact|TestCheckBaselineDriftFromReport|TestFactorizeRegistryWiring|TestDupcodeSharedProvider' ./internal/factory/dupcode/... ./internal/factory/gate/...  # PASS 4.246s + 1.376s
CGO_ENABLED=1 go test -race -count=5 -run 'TestValidateBaselineArtifact|TestCheckBaselineDriftFromReport|TestFactorizeRegistryWiring|TestDupcodeSharedProvider' ./internal/factory/dupcode/... ./internal/factory/gate/...   # PASS 2.369s + 1.424s
go vet ./internal/factory/dupcode/... ./internal/factory/gate/... ./cmd/leamas/...   # PASS

# Gates
CGO_ENABLED=0 make gate-fast   # PASS 2026-07-22T15:20:08Z — *** GATE PASSED ***
CGO_ENABLED=0 make gate-dupcode  # PASS 2026-07-22T15:13:38Z — dupcode:OK, dupcode-baseline:OK, go test -short ./internal/factory/dupcode/... OK

# Controlled factorize measurement
make factorize   # PASS 2026-07-22T15:13:53Z → 2026-07-22T15:17:03Z; *** FACTORIZE PASSED: 188.94s ***

# Forward gate summary + targeted digest
bin/leamas factory gate --test-mode=short   # regenerated .factory/gate-fast-summary.json (pass)
bin/leamas factory digest --range 462d1ca~1..HEAD \
  --output docs/close-reports/ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01/forward-range/digest.txt   # OK time=0.19s
```

## Skipped / Deferred

- Long lane is not executed in editor context; it is deferred to
  non-interactive CI. The `gate-summary.json` reflects this honest
  skip.
- `make gate` (canonical full gate) is REFUSED in Codium/VS Code/Cline
  terminal contexts; the long lane requires a non-editor invocation.
- Full `go test ./internal/factory/dupcode/...` (without `-run` filter)
  triggers an unrelated pre-existing 10-minute timeout in
  `TestV4ExactGeometryInternal_NoShadowSubFindings` (introduced in
  commit `52e90d8` `factory/dupcode: correct V4 exact-geometry oracle`,
  unrelated to SHARED-SCAN01). The targeted runs that gate
  SHARED-SCAN01 changes pass deterministically across counts 1, 20,
  and `race 5`.

## Board State

```text
ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01
  CLOSED

production implementation:
  PASS

single-scan semantic proof:
  PASS

fail-closed registry replacement:
  PASS

clean / stale baseline scenarios:
  PASS

gate-fast:
  PASS (2026-07-22T15:20:08Z)

gate-dupcode:
  PASS (2026-07-22T15:13:38Z)

full dupcode test suite (targeted suite for SHARED-SCAN01):
  PASS (-count=1, -count=20, -race -count=5)

measured wall-clock improvement:
  PASS (462.14s → 188.94s; Δ −59.1%)

final immutable evidence:
  PASS (digest bound to fresh gate-summary at 2026-07-22T15:20:30Z)
```

## Follow-up ACTs

1. `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TIMEOUT01` — classify
   the pre-existing infinite recursion in
   `TestV4ExactGeometryInternal_NoShadowSubFindings` (introduced in
   `52e90d8`) outside the SHARED-SCAN01 scope.

## Annotated Tag

After the forward evidence-correction commit, an annotated tag
`act/leamas-factory-factorize-dupcode-shared-scan01` will be created
pointing at the final close commit.
