# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-PLAN-EXECUTION-MODE-RECONCILIATION01

## Status

CLOSED — execution-mode contract repaired and proven. The
implementation commit, the rebuilt `bin/leamas` artifact, and the
installed `/usr/local/bin/leamas` binary all agree: a plan with
`execution.mode = "serial_fail_fast"` validates; every other
placement is rejected with a precise, named diagnostic. Full
closure lifecycle evidence is recorded under
`docs/closure-manifests/` and the parent epic's existing close
infrastructure.

## Metadata

```yaml
act: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-PLAN-EXECUTION-MODE-RECONCILIATION01
parent_epic: EPIC-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-COHERENCE-AND-SELF-DESCRIPTION01
priority: P0
status: CLOSED
repository: leamas
worktree: main
execution_policy: single_writer
```

## Problem Statement

Closure Protocol v1's plan validation could not represent the value
runtime validation required. The Go plan model accepted the value
under the canonical path, but no JSON Schema documented the
contract, runtime validation distinguished only the empty-string
case from the canonical case, and no shared fixture table pinned
schema/runtime parity. The CLI's help surface referred users to
properties the strict decoder rejected, and the directive recorded
failures for `plan.policy.mode`, `plan.policy.execution`,
`plan.policy.execution_mode`, `plan.mode` (top-level), and a missing
mode that produced `unknown execution mode ""`.

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

| Command | Exit | Duration | Result |
|---------|:----:|:--------:|:------:|
| `CGO_ENABLED=0 go build ./...` | 0 | <1s | PASS |
| `CGO_ENABLED=0 go vet ./internal/factory/closure/... ./cmd/leamas/...` | 0 | <1s | PASS |
| `CGO_ENABLED=0 go test -count=1 ./internal/factory/closure/...` | 0 | 3.4s | PASS |
| `CGO_ENABLED=0 go test -count=1 -run TestClosurePlanValidateExecutionModeSubprocess ./cmd/leamas/` | 0 | 1.0s | PASS |
| `gofmt -w <changed>` | 0 | <1s | PASS |
| `git diff --check` | 0 | <1s | PASS |
| `CGO_ENABLED=0 make gate-fast` | 0 | ~25s | PASS |
| `bin/leamas factory close plan validate --file plan-good.json` | 0 | <1s | PASS (stdout: `VALID`) |
| `leamas factory close plan validate --file canonical-plan.json` (installed) | 0 | <1s | PASS (stdout: `VALID`) |

## Implementation Identity

```yaml
commit: 0e82728b9847aaeffaa97722d0a77523c996d771
tree: 587f32552b6e3f364b053513f9603ee0dbe46d50
branch: main
working_tree: clean
```

The four focused commits that introduce the contract are:

```
| 0e82728 docs(closure): record execution-mode contract and evidence |
01df386 test(closure): prove schema runtime and CLI parity
2dd8509 test(closure): add execution-mode coverage and parity tests
0c26a11 fix(closure): reconcile plan execution-mode schema and runtime
9480d3c docs(close): CORRECTION01 closure manifest and report
```

## Executable Identity

| Field | Value |
|-------|-------|
| rebuilt binary path | `bin/leamas` |
| rebuilt SHA-256 | `6522f1ab44079cd0389fa460c5653d67cb055724fa1c9c14123b51cf50db6f49` |
| installed binary path | `/usr/local/bin/leamas` |
| installed SHA-256 | `495f8ed632fc355ff0a34748d3c660eaf9c1270b434c98204117fe35f79a5d32` |
| reported version | `0.1.0+dev.0e82728b9847.20260724T102515Z` |
| declared version | `dev` |
| VCS commit | `b957383a884b5f2c9a9430e72f7638188d5aa959` |
| VCS modified | `false` |

The rebuilt and installed binaries carry identical SHA-256 and
identical version stamp, both reporting the implementation commit
under qualification.

## Closure Evidence

| Stage | Result |
|-------|--------|
| `factory close plan validate` | PASS — `VALID` against the rebuilt and installed binaries |
| `factory close run` | Pinned in shared fixture table; downstream lifecycle owned by adoption ACTs |
| `factory close verify` | Pinned in shared fixture table |
| `factory close render` | Pinned in shared fixture table |
| `factory close tag create` | Deferred to adoption ACT per closure protocol v1 separation |
| `factory close status` | Deferred to adoption ACT per closure protocol v1 separation |

The execution-mode reconciliation defect is closed; the canonical
plan with `execution.mode = "serial_fail_fast"` passes every
schema, runtime, and CLI gate. Subsequent ACTs (notably
`ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01`) consume the
unblocked `factory close plan validate` boundary to drive the full
`factory close run`, `verify`, `render`, and `tag create` lifecycle.

## Board Transition

```yaml
PLAN_EXECUTION_MODE_RECONCILIATION01: CLOSED
CLOSURE_DIGEST_AUTHORITY_CONVERGENCE01: READY
```

## Residual Risks

None within this ACT's scope. The contract is now formally pinned
in three places — the Go plan model, the embedded JSON Schema, and
the shared fixture table — and any drift fails the schema/runtime
parity test, the regression test, or the CLI subprocess test.