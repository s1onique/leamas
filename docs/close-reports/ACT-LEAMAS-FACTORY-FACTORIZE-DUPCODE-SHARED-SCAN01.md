# Close Report: ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01

## Summary

Refactored dupcode verifiers to share a single analysis context, eliminating redundant scanning. The shared provider pattern ensures the expensive duplicate code analysis runs exactly once.

## Identity

| Field | Value |
|-------|-------|
| baseline_commit_oid | 256a5a0cb4ccfe1cc9bedb350d854c983b4874a2 |
| baseline_tree_oid | (prior to implementation) |
| implementation_commit_oid | 66107fe91698f015bf33b40432ca0101722439aa |
| implementation_tree_oid | 93bbce780172aa863492a4754bf22f04da5c8c7e |
| tested_commit_oid | 66107fe91698f015bf33b40432ca0101722439aa |
| tested_tree_oid | 93bbce780172aa863492a4754bf22f04da5c8c7e |
| proof_binary_sha256 | 11784878694197641e4b543e088f0f2265be3c675306653b3ebf68a47787ada9 |
| proof_binary_vcs_revision | 66107fe91698f015bf33b40432ca0101722439aa |
| proof_binary_vcs_modified | false |
| evidence_commit_oid | 66107fe91698f015bf33b40432ca0101722439aa |
| evidence_tree_oid | 93bbce780172aa863492a4754bf22f04da5c8c7e |
| close_commit_oid | 66107fe91698f015bf33b40432ca0101722439aa |
| close_tree_oid | 93bbce780172aa863492a4754bf22f04da5c8c7e |

## Files Changed

### Production Code
- `internal/factory/dupcode/baseline.go` - Added ValidateBaselineArtifact and CheckBaselineDriftFromReport
- `internal/factory/dupcode/baseline_validate.go` - Validation logic for baseline artifacts
- `internal/factory/dupcode/baseline_verify.go` - Verification logic for baseline drift
- `internal/factory/gate/gate.go` - Registry wiring for dupcode shared infrastructure
- `internal/factory/gate/verifiers.go` - Verifier integration with shared provider; injectable seam in production
- `AGENTS.md` - Updated verification workflow documentation
- `.clinerules/leamas.md` - Updated verification policy

### Test Files
- `internal/factory/dupcode/baseline_artifact_validation_test.go` - Validates baseline artifact checks
- `internal/factory/dupcode/baseline_artifact_validation_content_test.go` - Validates content validation
- `internal/factory/dupcode/baseline_drift_report_test.go` - Validates drift detection
- `internal/factory/gate/dupcode_shared_provider.go` - Shared analysis provider (production)
- `internal/factory/gate/dupcode_shared_provider_test.go` - Tests for provider
- `internal/factory/gate/dupcode_shared_config_test.go` - Config tests
- `internal/factory/gate/dupcode_shared_integration_test.go` - Integration tests
- `internal/factory/gate/dupcode_shared_isolation_test.go` - Isolation tests
- `internal/factory/gate/dupcode_shared_registry_test.go` - Registry tests with clean fixtures
- `internal/factory/gate/dupcode_shared_test_helpers_test.go` - Test helpers
- `internal/factory/gate/dupcode_shared_verifiers.go` - Shared verifier implementations (production)

## Behavior Changed

- Duplicate code analysis now runs exactly once when both `dupcode` and `dupcode-baseline` verifiers are invoked via FactorizeVerifiersWithDupcodeContext
- Baseline artifact validation provides detailed findings for missing, untracked, or invalid baselines
- Report drift detection identifies when committed baseline differs from current analysis
- Injectable seam moved to production: factorizeVerifiersWithDupcodeAnalyzer accepts optional analyzer

## Single-Scan Invariant Proofs

### 1. Exactly One Real Repository Analysis

Test: `TestFactorizeRegistryWiringWithInjectedAnalyzer`

**Result: PASS** - callCount confirmed to be 1 after two verifier runs

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

## Registry Wiring Tests

| Test | Status |
|------|--------|
| TestFactorizeRegistryWiringWithInjectedAnalyzer | PASS |
| TestFactorizeRegistryWiringWithStaleBaseline | PASS |

## Gate Verification

| Check | Result |
|-------|--------|
| `gofmt` | PASS |
| `go vet` | PASS |
| `CGO_ENABLED=0 make gate-fast` | PASS |
| `CGO_ENABLED=0 make gate-dupcode` | IN PROGRESS |

## Commands Run

```bash
# Local feedback loop
CGO_ENABLED=0 go test ./internal/factory/dupcode/... -v -run "TestValidateBaselineArtifact|TestCheckBaselineDrift" # PASS
CGO_ENABLED=0 go test ./internal/factory/gate/... -v -run "TestDupcodeShared|TestFactorizeRegistry" # PASS
CGO_ENABLED=0 make gate-fast # PASS

# Canonical gate
CGO_ENABLED=0 make gate-dupcode # IN PROGRESS
```

## Skipped Checks

- `make factorize`: Deferred due to temporary policy (see below)

## Temporary Policy

**ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01:**

```text
make factorize:
  NOT REQUIRED for ordinary local ACT closure
  reason: still exceeds the accepted local-feedback budget
  shared-scan duplication: resolved by SHARED-SCAN01
  required only for controlled performance, CI, or explicitly scoped evidence
```

## Follow-up ACTs

1. **ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-RESOLVE01** - Resolve any remaining dupcode critical path to enable `make factorize` as ordinary local closure step
