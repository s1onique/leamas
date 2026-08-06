# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1-PLAN-CORRECTION01

## Status

OPEN — PLAN-FREEZE CORRECTION FOR
`ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1`

## Mission

The freeze commit
`056c1b4fd4aa5a4c2b94a42cc6d3b2040a949d09` is
procedurally clean but does not execute several
acceptance-critical checks. The original ACT
explicitly forbids changing the specification after
F. Therefore, this ACT preserves 056c1b4 as a
historical superseded freeze and publishes a
corrected authoritative freeze `F2` that contains
the full set of required checks.

`F2` becomes the freeze commit for the original ACT.
The original ACT is the subject; this correction ACT
is the new freeze authority. The execution body of
the original ACT is unchanged.

## Base

```text
BASE_COMMIT=056c1b4fd4aa5a4c2b94a42cc6d3b2040a949d09
BASE_TREE=9f090baafcdf54db3315814ca542803a484cbb9c
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Required corrections to the closure plan

The original closure plan
`docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1.json`
has eight plan defects. The corrected plan MUST:

1. **Range-bound `git diff --check`**.

   Inspect the implementation subject, not the clean
   worktree. The check MUST be range-bound, for
   example `git diff --check $F2 $S` where `$F2` and
   `$S` are closure-executor-provided identifiers
   that the executor substitutes from the recorded
   freeze and subject identities. The frozen plan
   MUST NOT embed future `$S` literally.

2. **Full closure-package tests**.

   Add a check that runs:

   ```text
   go test -count=1 ./internal/factory/closure/...
   ```

   so unrelated closure-package regressions cannot
   pass silently under a v2-verifier-only plan.

3. **ACT-owned exec-gate**.

   Add a focused exec-gate check that runs
   `make gate-fast` (or the equivalent
   `leamas factory gate --lane=fast`) and binds its
   literal result to `ACT_OWNED_EXEC_GATE_RESULT`.
   A failed gate MUST be reported as a failed gate;
   it MUST NOT be coerced to PASS.

4. **ACT-owned LLM-friendliness**.

   Add a check that runs the existing
   `make verify-llm-friendly` verifier and binds the
   result to `LLM_FRIENDLY_ACT_FILES`. The previous
   plan checked only `gofmt` and missed this
   acceptance criterion.

5. **Required policy checks**.

   Add explicit checks for the three named policy
   lanes:

   ```text
   tooling-boundaries
   long-test-policy
   static-binary
   ```

   The exact recipes are the existing
   `make verify-tooling-boundaries`,
   `make test-long`, and `make verify-static`
   targets. Each is bound to a separate acceptance
   key so a single policy failure cannot be hidden
   inside a passing aggregate.

6. **Hermetic static-build destination**.

   Replace the fixed shared `/tmp/leamas-v2verifier-r1`
   path with a hermetic, run-scoped destination:

   ```text
   $BUILD_DIR/leamas-v2verifier-r1
   ```

   where `$BUILD_DIR` is supplied by the closure
   executor (typically `.factory/build/<run-id>/`).
   The recipe MUST create the directory if it does
   not exist and MUST remove the build artefact at
   the end of the run.

7. **Stable ACT-specific focused-test selector**.

   The current `TestV2Verifier` regex is both too
   broad and too narrow. The plan MUST name the
   exact test families the matrices require. The
   corrected selector is:

   ```text
   -run TestV2VerifierDuplicate|TestV2VerifierExit|TestV2VerifierPublic|TestV2VerifierTerminal|TestV2VerifierJSON|TestV2VerifierMetadataObs|TestV2VerifierPreObservation|TestV2VerifierHelp
   ```

   The plan MUST also assert a row count for each
   family that matches the matrix in the support
   document.

8. **Gate-summary-producing check**.

   Add a check that runs the aggregate fast lane
   and writes a literal `.factory/gate-summary.json`
   artefact. The check MUST report the literal
   aggregate result for `AGGREGATE_GATE_STATUS` and
   the literal list of pre-existing findings for
   `PRE_EXISTING_GATE_FINDINGS`. A failed gate
   summary MUST be reported as a failed gate
   summary.

## Acceptance

```text
PLAN_FREEZE_RANGE_DIFF_CHECK=range_bound
PLAN_FREEZE_CLOSURE_PACKAGE_TESTS=full
PLAN_FREEZE_ACT_OWNED_EXEC_GATE=focused
PLAN_FREEZE_LLM_FRIENDLY=verified
PLAN_FREEZE_TOOLING_BOUNDARIES=verified
PLAN_FREEZE_LONG_TEST_POLICY=verified
PLAN_FREEZE_STATIC_BINARY=verified
PLAN_FREEZE_HERMETIC_BUILD=run_scoped
PLAN_FREEZE_FOCUSED_SELECTOR=stable
PLAN_FREEZE_GATE_SUMMARY=literal
F2_NE_F=true
F2_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```

## Closure Protocol

```text
implementation_subject   = this correction ACT (single commit)
plan_freeze_commit       = F2 (this ACT)
closure_evidence_commit  = C  (next ACT or runner-emitted)
tag_target               = C
TOPOLOGY = F < F2 < S < C
```

`F` is the historical superseded freeze
`056c1b4fd4aa5a4c2b94a42cc6d3b2040a949d09`.
`F2` is this correction. `S` is the implementation
subject of the original ACT, distinct from `F2`.
`C` follows `S`.

## Publication

Exactly one implementation commit:

```text
factory: correct v2 verifier Mac handoff R1 plan F2
```

## Expected final status

```text
STATUS=PASS
PLAN_FREEZE_CORRECTION_CLOSURE=true
F2_NE_F=true
F2_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```
