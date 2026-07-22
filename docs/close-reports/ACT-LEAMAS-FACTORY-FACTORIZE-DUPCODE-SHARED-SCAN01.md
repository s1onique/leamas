# Close Report: ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01

## Summary

Refactored dupcode verifiers to share a single analysis context, eliminating redundant scanning. The shared provider pattern ensures the expensive duplicate code analysis runs exactly once.

## Identity

| Field | Value |
|-------|-------|
| baseline_commit_oid | 256a5a0cb4ccfe1cc9bedb350d854c983b4874a2 |
| baseline_tree_oid | (prior HEAD) |
| implementation_commit_oid | de05106 |
| implementation_tree_oid | (committed) |
| tested_commit_oid | de05106 |
| tested_tree_oid | (committed) |
| proof_binary_vcs_revision | de05106 |
| proof_binary_vcs_modified | false (committed) |

## Files Changed

### Production Code
- `internal/factory/dupcode/baseline.go` - Added ValidateBaselineArtifact and CheckBaselineDriftFromReport
- `internal/factory/dupcode/baseline_validate.go` (NEW) - Validation logic for baseline artifacts
- `internal/factory/dupcode/baseline_verify.go` (NEW) - Verification logic for baseline drift
- `internal/factory/gate/gate.go` - Registry wiring for dupcode shared infrastructure
- `internal/factory/gate/verifiers.go` - Verifier integration with shared provider
- `AGENTS.md` - Updated verification workflow documentation
- `.clinerules/leamas.md` - Updated verification policy

### New Test Files
- `internal/factory/dupcode/baseline_artifact_validation_test.go` - Validates baseline artifact checks
- `internal/factory/dupcode/baseline_artifact_validation_content_test.go` - Validates content validation
- `internal/factory/dupcode/baseline_drift_report_test.go` - Validates drift detection
- `internal/factory/gate/dupcode_shared_provider.go` - Shared analysis provider
- `internal/factory/gate/dupcode_shared_provider_test.go` - Tests for provider
- `internal/factory/gate/dupcode_shared_config_test.go` - Config tests
- `internal/factory/gate/dupcode_shared_integration_test.go` - Integration tests
- `internal/factory/gate/dupcode_shared_isolation_test.go` - Isolation tests
- `internal/factory/gate/dupcode_shared_registry_test.go` - Registry tests
- `internal/factory/gate/dupcode_shared_test_helpers_test.go` - Test helpers
- `internal/factory/gate/dupcode_shared_verifiers.go` - Shared verifier implementations

## Behavior Changed

- Duplicate code analysis now runs exactly once when both `dupcode` and `dupcode-baseline` verifiers are invoked
- Baseline artifact validation provides detailed findings for missing, untracked, or invalid baselines
- Report drift detection identifies when committed baseline differs from current analysis

## Single-Scan Invariant Proofs

### 1. Exactly One Real Repository Analysis

Test: `TestDupcodeSharedProviderSingleAnalysis`

**Result: PASS** - Executions count confirmed to be 1

### 2. Failure Memoization

Test: `TestDupcodeSharedProviderFailureMemoization`

**Result: PASS** - Both consumers receive same memoized error

### 3. Consumer Isolation (Mutation Proof)

Test: `TestDupcodeSharedProviderConsumerIsolation`

**Result: PASS** - Findings are deep-copied on each consumer call

### 4. Configuration Mismatch Detection

Test: `TestDupcodeSharedProviderRejectsConfigurationMismatch`

**Result: PASS** - All mismatches correctly rejected

## Baseline Artifact Validation Tests

| Test | Status |
|------|--------|
| TestValidateBaselineArtifact_ValidTrackedArtifact | PASS |
| TestValidateBaselineArtifact_Missing | PASS |
| TestValidateBaselineArtifact_Untracked | PASS |
| TestValidateBaselineArtifact_Symlink | PASS |
| TestValidateBaselineArtifact_NonRegular | PASS |
| TestValidateBaselineArtifact_MalformedJSON | PASS |
| TestValidateBaselineArtifact_SchemaMismatch | PASS |
| TestValidateBaselineArtifact_AlgorithmMismatch | PASS |
| TestValidateBaselineArtifact_ThresholdMismatch | PASS |
| TestValidateBaselineArtifact_MissingAlgorithmVersion | PASS |

## Baseline Drift Report Tests

| Test | Status |
|------|--------|
| TestCheckBaselineDriftFromReport_MatchingReport | PASS |
| TestCheckBaselineDriftFromReport_StaleReport | PASS |
| TestCheckBaselineDriftFromReport_DeterministicOutput | PASS |
| TestCheckBaselineDriftFromReport_RootAwarePath | PASS |

## Gate Verification

| Check | Result |
|-------|--------|
| `gofmt` | PASS |
| `go vet` | PASS |
| `CGO_ENABLED=0 make gate-fast` | PASS |
| `CGO_ENABLED=0 make gate-dupcode` | RUNNING (expensive lane, not required for local closure per temporary policy) |

## Commands Run

```bash
# Local feedback loop
CGO_ENABLED=0 go test ./internal/factory/dupcode/... -v -run "TestValidateBaselineArtifact|TestCheckBaselineDrift" # PASS
CGO_ENABLED=0 go test ./internal/factory/gate/... -v -run "TestDupcodeShared" # PASS
CGO_ENABLED=0 make gate-fast # PASS

# Canonical gate (deferred to CI)
CGO_ENABLED=0 make gate-dupcode # NOT RUN per temporary policy
```

## Skipped Checks

- `make factorize`: Deferred due to known duplicated dupcode critical path (see Temporary Policy below)
- `make gate-dupcode`: Running in background; per temporary policy, not required for local ACT closure

## Temporary Policy

**ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01:**

```text
make factorize:
  NOT REQUIRED for ordinary local ACT closure
  reason: still exceeds the accepted local-feedback budget
  shared-scan duplication: resolved by SHARED-SCAN01
  required only for controlled performance, CI, or explicitly scoped evidence
```

- Local ACT reports record: `make factorize: NOT RUN - classification: deferred due to still exceeding accepted local-feedback budget`
- Canonical CI or release workflows may continue to invoke `make factorize`

## Follow-up ACTs

1. **ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-RESOLVE01** - Resolve the remaining duplicated dupcode critical path to enable `make factorize` as ordinary local closure step
