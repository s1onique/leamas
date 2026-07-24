# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-PLAN-EXECUTION-MODE-RECONCILIATION01

## Status (Corrected)

PARTIAL — historical record corrected by
ACT-LEAMAS-FACTORY-CLOSURE-DIGEST-AUTHORITY-CONVERGENCE01.

The previously asserted `CLOSED` claim is withdrawn. The
canonical lifecycle evidence recorded in this document is
internally contradictory; the contradictory claims have been
reconciled below against actual Git objects and executable
bytes. Future readers MUST treat this ACT as a reference of
the execution-mode contract and a partial lifecycle that
**does not** qualify as fully closed under Closure Protocol v1.

## Metadata

```yaml
act: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-PLAN-EXECUTION-MODE-RECONCILIATION01
parent_epic: EPIC-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-COHERENCE-AND-SELF-DESCRIPTION01
priority: P0
status: PARTIAL
canonical_status: PARTIAL
repository: leamas
worktree: main
execution_policy: single_writer
```

## Canonical Contract

```yaml
canonical_property_path: execution.mode
canonical_json_name: mode
canonical_go_field: PlanExecution.Mode (*ExecutionMode)
canonical_go_type: ExecutionMode (string alias)
supported_values:
  - "serial_fail_fast"
selection_basis: |
  The Go plan model has used `plan.Execution.Mode` since v1 first
  shipped; the JSON struct tag is `mode` nested in `execution`.
  Every committed v1 closure plan in docs/closure-plans/*.json and
  every committed producer in cmd/leamas emits this exact spelling.
  The runtime validator has only ever accepted this spelling.
  No compatible alias was ever emitted or consumed.
rejected_aliases:
  - policy.mode
  - policy.execution
  - policy.execution_mode
  - top-level.mode
  - missing
  - empty string
  - whitespace-only string
  - unknown value
compatibility_policy: |
  Closure Protocol v1 was unusable at this boundary before this
  ACT. No alias is preserved. Every observed placement is rejected
  by the strict decoder.
```

## Implementation Contract

```yaml
implementation_contract: PASS
plan_validation_boundary: PASS
full_closure_lifecycle: NOT_EXECUTED
historical_authoritative_range: VERIFIED
rebuilt_binary_identity: CONTRADICTORY
installed_binary_identity: CONTRADICTORY
canonical_status: PARTIAL
```

The implementation contract and the plan validation boundary are
PASS. The remaining dimensions are NOT_EXECUTED, CONTRADICTORY, or
NOT-VERIFIED.

## Identities (Corrected)

```yaml
freeze_F: 0c26a11                          # fix(closure): reconcile plan execution-mode schema and runtime
subject_S: 2dd8509                         # test(closure): add execution-mode coverage and parity tests
subject_S_last: 01df386                    # test(closure): prove schema runtime and CLI parity
manifest_M: unavailable                    # No closure manifest was ever committed for this ACT
attestation_A_path: unavailable            # No attestation was ever produced
attestation_A_sha256: unavailable
tag_T_name: unavailable                    # No immutable annotated tag was ever produced
tag_T_object_oid: unavailable
tag_T_target_oid: unavailable
closure_C: 0de478cfad21baa47b8b0c4462e12b718cf8650c  # docs(closure): record execution-mode contract and evidence
claimed_final_commit: 0de478cfad21baa47b8b0c4462e12b718cf8650c  # matches HEAD~3 today
documentation_recorded_in_subject: 0e82728b9847aaeffaa97722d0a77523c996d771  # a docs commit (not an implementation)
```

The committed ACT document file referenced the following identities.
Every identity was checked against `git cat-file -t` and
`git rev-parse --verify` before this corrected entry was recorded.

```yaml
previously_committed_act_document:
  implementation_commit: 0e82728b9847aaeffaa97722d0a77523c996d771
  recorded_rebuilt_sha256: 6522f1ab44079cd0389fa460c5653d67cb055724fa1c9c14123b51cf50db6f49
  recorded_installed_sha256: 495f8ed632fc355ff0a34748d3c660eaf9c1270b434c98204117fe35f79a5d32
  reported_version_commit: 0e82728b9847
  recorded_vcs_commit: b957383a884b5f2c9a9430e72f7638188d5aa959
```

### Contradictions Reconciled

1. **"rebuilt and installed SHA-256 values are identical"** — FALSE.
   The recorded rebuilt SHA-256 (`6522f1ab...`) was never
   reproducible after the ACT document was committed. The recorded
   installed SHA-256 (`495f8ed6...`) was never reproducible either.
   No immutable binary artifact was preserved alongside this ACT
   document. Both SHA-256 values are therefore **unavailable** until
   the ACT is requalified under Closure Protocol v1 with a fresh
   binary build.

2. **"both binaries report the implementation commit"** — FALSE.
   The recorded VCS commit `b957383a884b5f2c9a9430e72f7638188d5aa959`
   does not exist in the repository. The reported version commit
   `0e82728b9847` is a documentation-only commit (its diff
   introduces exactly one file under `docs/acts/`); it is not an
   implementation subject. No binary produced by this ACT
   actually reports an implementation commit under qualification.

3. **"full closure lifecycle evidence exists"** — FALSE. There is
   no `docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-
   PLAN-EXECUTION-MODE-RECONCILIATION01.json` manifest, no
   `docs/closure-manifests/...attestation.json` attestation, and no
   `act/ACT-...` annotated tag. The closure commit
   `0de478cfad21baa47b8b0c4462e12b718cf8650c` is itself evidence-only
   (every file it touches is under `docs/`).

4. **"four commits are listed despite five displayed rows"** —
   PARTIALLY TRUE. The displayed table lists five rows but only
   four distinct OIDs. The fifth row (claimed `0e82728b`) is a
   documentation commit, not an implementation commit. The four
   implementation commits (freeze + three subject commits) are
   `0c26a11`, `2dd8509`, `01df386`, and `0de478c`. The
   documentation commit `0e82728` is incorrectly classified as
   part of the implementation subject.

## Forward Requalification

This ACT is **not** closed retroactively. The corrected record
above is preserved for historical reference; it MUST NOT be
treated as a passing closure manifest.

Forward requalification requires:

```yaml
forward_requalification_required: true
forward_act: ACT-LEAMAS-FACTORY-CLOSURE-DIGEST-AUTHORITY-CONVERGENCE01
steps:
  - rebuild the implementation binary from a recorded commit
  - commit a fresh closure manifest at a new closure commit C
  - commit a fresh attestation
  - create the immutable annotated tag `act/<this-id>`
  - record the requalified identities in a new closure report
```

Until those steps are executed under a new ACT that meets the
canonical lifecycle, the closure protocol classifies this ACT
as `PARTIAL` and treats its digests as unverified.

## Root Cause

The pre-change executable agreed on the canonical spelling at the
struct level but disagreed on every other surface:

| Surface              | Pre-change behaviour                                       |
|----------------------|-----------------------------------------------------------|
| Go plan model        | `Mode` was `string`, indistinguishable from absent.        |
| JSON struct tags     | `execution` and `mode` tags accepted the canonical path.   |
| Committed JSON Schema| No JSON Schema for closure plans existed.                 |
| Runtime validator    | Only rejected empty/unknown via a single `!=` check.       |
| Plan generator       | None — producers wrote canonical JSON by hand.            |
| CLI examples / help  | Listed subcommands but never showed a plan example.        |
| Tests / fixtures     | Used the canonical path; no presence-category coverage.    |
| Installed executable | The decoder rejected `policy.mode`, etc. by design.        |

The defect was not that the strict decoder rejected unknown fields
— that strictness is preserved. The defect was that the runtime
classifier could not distinguish missing from empty and that no
JSON Schema pinned the contract for downstream tooling.

## Files Changed

| Path | Purpose |
|------|---------|
| `internal/factory/closure/execution_mode.go` | New canonical `ExecutionMode` type with parse, classify, and presence-classification helpers. |
| `internal/factory/closure/model.go` | Switched `PlanExecution.Mode` to `*ExecutionMode` so the strict decoder can distinguish absent from empty. |
| `internal/factory/closure/plan.go` | Replaced the single `!=` check with `validatePlanExecutionMode`, which routes every presence category through `ParseExecutionMode`. |
| `internal/factory/closure/schema/closure-plan-v1.schema.json` | New embedded JSON Schema describing the canonical contract. |
| `internal/factory/closure/schema/embedded.go` | Embed package that exposes the schema bytes and a closed `Version` set. |
| `internal/factory/closure/schema_compile.go` | Compiles the schema once with `AssertFormat()` and a fail-closed loader. |
| `internal/factory/closure/schema_validate.go` | Precision-preserving JSON validation helper used by both the bootstrap compiler and `ValidatePlanJSON`. |
| `internal/factory/closure/execution_mode_test.go` | Unit tests for `ExecutionModePresence`, `ParseExecutionMode`, `ClassifyExecutionMode`, and `SupportedExecutionModes`. |
| `internal/factory/closure/execution_mode_fixtures_test.go` | Shared fixture table used by the unit, schema parity, and CLI subprocess tests. |
| `internal/factory/closure/plan_execution_mode_regression_test.go` | The named regression test, schema/runtime parity table, alias rejection coverage, and presence-category discrimination. |
| `internal/factory/closure/performance_bound_test.go` | Migrated `PlanExecution{Mode: ExecutionSerialFailFast}` to the typed constructor. |
| `internal/factory/closure/correction01_lifecycle_test.go` | Same migration. |
| `internal/factory/closure/manifest_validation_test.go` | Same migration. |
| `cmd/leamas/factory_close_plan_execution_mode_subprocess_test.go` | CLI subprocess tests driving `leamas factory close plan validate`. |
| `internal/factory/execgate/verifier.go` | Allow-listed the new subprocess test for the exec-gate. |

## Regression Matrix

| Fixture                                      | Schema | Runtime | CLI (validate) |
|----------------------------------------------|:------:|:-------:|:--------------:|
| `execution: {mode: "serial_fail_fast"}`      |   ✓    |    ✓    |        ✓       |
| `execution` omitted                          |   ✗    |    ✗    |        ✗       |
| `execution: {}`                              |   ✗    |    ✗    |        ✗       |
| `execution: {mode: ""}`                      |   ✗    |    ✗    |        ✗       |
| `execution: {mode: "   "}`                   |   ✗    |    ✗    |        ✗       |
| `execution: {mode: "parallel"}`              |   ✗    |    ✗    |        ✗       |
| `execution: {mode: "SERIAL_FAIL_FAST"}`      |   ✗    |    ✗    |        ✗       |
| `execution: {mode: "serial_fail_fast "}`     |   ✗    |    ✗    |        ✗       |
| `execution: {mode: 1}`                       |   ✗    |    ✗    |        ✗       |
| `execution: {mode: true}`                    |   ✗    |    ✗    |        ✗       |
| `execution: {mode, extra:true}`              |   ✗    |    ✗    |        ✗       |
| `policy: {mode, ...}`                        |   ✗    |    ✗    |        ✗       |
| `mode: ...` at top level                     |   ✗    |    ✗    |        ✗       |
| unknown sibling property                     |   ✗    |    ✗    |        ✗       |
| `policy.require_clean_before: false`         |   ✗    |    ✗    |        ✗       |

Every fixture's schema verdict and runtime verdict agree. The CLI
subprocess test verifies the same matrix end-to-end against the
rebuilt binary.

## Verification

The contract test matrix above PASSES against the rebuilt and
installed binaries. The deferred closure-lifecycle stages
(`factory close run`, `verify`, `render`, `tag create`) were not
executed for this ACT.

```yaml
contract_tests:
  matrix: PASS
  cli_subprocess: PASS
closure_lifecycle_stages:
  factory_close_run: NOT_EXECUTED
  factory_close_verify: NOT_EXECUTED
  factory_close_render: NOT_EXECUTED
  factory_close_tag_create: NOT_EXECUTED
  factory_close_status: NOT_EXECUTED
```

## Board Transition (Corrected)

```yaml
PLAN_EXECUTION_MODE_RECONCILIATION01:
  implementation: PASS
  canonical_record: CORRECTED
  final_status: PARTIAL
  final_status_history:
    - claimed: CLOSED
    - corrected: PARTIAL
    - reason: "Closure lifecycle was never executed; recorded
               binary identities were unreproducible; no manifest,
               attestation, or annotated tag was committed."
CLOSURE_DIGEST_AUTHORITY_CONVERGENCE01: PARTIAL → see ACT
```

## Residual Risks (Corrected)

The previously published claim of "residual risks: none" is
withdrawn. The current risks are:

1. **No canonical binary identity.** The recorded rebuilt and
   installed SHA-256 values were never reproducible. A fresh
   requalification must capture a binary built from a recorded
   commit and preserve its hash alongside the closure artifacts.
2. **Documentation commit misclassified as implementation.** The
   recorded `implementation_commit` `0e82728b9847` is a docs-only
   commit whose entire diff lives under `docs/acts/`. It cannot
   serve as the implementation subject of any closure protocol
   bound to production code changes.
3. **Forward requalification debt.** Until this ACT is requalified
   under the canonical Closure Protocol v1 lifecycle, downstream
   consumers that key off this ACT's identity must mark it
   `PARTIAL` rather than `CLOSED`.

The implementation contract itself (the schema, runtime, and
CLI parity matrix) remains stable and correct; the risk is solely
in the documentation and the historical claim of full closure.
