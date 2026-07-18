# R1 Failure Inventory

Pre-convergence test and gate failures caused by the production
remediation in
ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01.

## Captured artifacts

* `pre-convergence-go-test.txt`     — `go test ./... -count=1`
* `pre-convergence-factorize.txt`   — `make factorize`
* `pre-convergence-baseline-verify.json` — baseline verify
* `pre-convergence-gate.txt`        — `make gate`
* `dupcode-baseline-before.json`    — committed baseline before
* `dupcode-baseline-after.json`     — committed baseline after

## Category A — Live-tree convergence tests

These tests prove that the committed baseline equals the current
repository. New expected state:

```text
live findings     = 0
baseline findings = 0
live == baseline
thresholds        = 40 / 400
```

### TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline

* Source: `internal/factory/dupcode/v4_baseline_delta_test.go`
* Asserted: `len(findings)==1` and `len(baseline.Findings)==1`
* Reason stale: live tree now reports zero findings
* Replacement: assert `len(findings)==0` and
  `len(baseline.Findings)==0` and threshold 40/400

### TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline

* Source: `internal/factory/dupcode/v4_baseline_audit_test.go`
* Asserted: line ranges `claim_commands.go:268-340` and
  `evidence_commands.go:310-382`
* Reason stale: line ranges invalidated by refactor's re-flow
* Replacement: same as above; no longer pins line ranges

### TestRunFactorize

* Source: `internal/factory/gate/gate_test.go`
* Asserted: factorize returns exit 0
* Reason stale: factorize fails on baseline drift until regenerated
* Replacement: factorize returns exit 0 once baseline converges

## Category B — Historical remediation witness

### TestV4BaselineForensics_877_LockFacts

* Source: `internal/factory/dupcode/v4_baseline_forensics_facts_test.go`
* Asserted: left owner count 4, right slice has unowned token
* Reason stale: the 877 line range now maps to 5 regions because
  the refactor's re-flow changed executable-region boundaries
* Replacement: read the frozen predecessor `dupcode-before.json`
  and assert its 504 finding has the original fingerprint, token
  count, and occurrences

## Category C — Detector semantic / geometry contract

These tests use production source as convenient test data. After
the refactor the duplicate is gone. Each test moves to a stable
synthetic fixture under `internal/factory/dupcode/testdata/`.

### TestV4PipelineTrace_StagesNonEmpty

* Source: `v4_pipeline_trace_test.go`
* Reason stale: `FilteredWindows is empty` because the live scan
  no longer finds any duplicate
* Replacement: use a stable fixture pair; assert each stage is
  non-empty

### TestV4PipelineTrace_ComponentsBeforeShadowContains504

* Source: `v4_pipeline_trace_test.go`
* Reason stale: `ComponentsBeforeShadow is empty`
* Replacement: use the fixture pair; assert the canonical
  component is present before shadow suppression

### TestV4BaselineForensics_504_NoLargerLiveChain

* Source: `v4_baseline_forensics_504_maximality_test.go`
* Reason stale: trace has zero findings
* Replacement: use a fixture whose canonical component is the
  maximum chain

### TestV4BaselineForensics_504_NoLargerLiveComponent

* Source: `v4_baseline_forensics_504_maximality_test.go`
* Reason stale: same
* Replacement: same

### TestV4BaselineForensics_504_SurvivesStructuralShadow

* Source: `v4_baseline_forensics_504_maximality_test.go`
* Reason stale: same
* Replacement: same

### TestV4BaselineForensics_504_IsCanonicalExactDuplicate

* Source: `v4_baseline_forensics_504_test.go`
* Reason stale: `v4PipelineInternal must emit exactly one finding`
* Replacement: use a fixture pair containing a known duplicate

### TestV4BaselineForensics_504_IsMaximalFromPrePublication

* Source: `v4_baseline_forensics_504_test.go`
* Reason stale: same
* Replacement: same

### TestV4BaselineForensics_504_CannotExtendOneToken

* Source: `v4_baseline_forensics_504_trace_test.go`
* Reason stale: trace has zero findings
* Replacement: use a fixture; assert every legal one-token
  extension is rejected by width, digest, owner, or chain
  existence

### TestV4BaselineForensics_AllCasesClassified

* Source: `v4_baseline_forensics_all_test.go`
* Reason stale: 504 closure branch fails because
  `v4PipelineInternal` returns zero findings
* Replacement: use fixture for the 504 case; 877 and 514 cases
  already pass

### TestV4BaselineForensics_504_SortedFingerprintStable

* Source: `v4_baseline_forensics_facts_test.go`
* Reason stale: `trace must emit exactly one final finding`
* Replacement: use a fixture; assert pre-shadow fingerprint
  equals final fingerprint

## Category D — Debug-only test

### TestDebugBaselines

* Source: `internal/factory/dupcode/debug_test.go`
* Reason stale: logs the 504 finding as current truth and uses
  verbose `Printf`
* Replacement: convert into a deterministic equality test that
  asserts zero findings on both committed and canonical and that
  the two are byte-equal

## Tests expected to remain green

The following tests already pass and must not regress:

* `TestRemediationDelta_504FindingRemoved` (strengthened by R4 to
  read frozen predecessor evidence)
* All `TestV4_*` semantic, geometry, shadow-suppression, and fuzz
  tests in the dupcode package
* All other top-level tests outside the dupcode package

## Baseline verifier (separate from `go test ./...`)

`bin/leamas factory verify dupcode-baseline` reports the canonical
`dupcode_baseline_drift: dupcode baseline is stale` because the
committed baseline still records the removed 504-token finding.
This is expected and will resolve once R7 regenerates the
baseline.
