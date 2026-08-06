# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1-PLAN-CORRECTION02

## Status

OPEN — PLAN-MECHANICS CORRECTION ON TOP OF
`ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1-PLAN-CORRECTION01`

## Mission

The F2 freeze at
`a5cfbf4adbde497dd0ca31961f7f6434683d9855` is
content-hygiene-clean but has nine plan-mechanics
defects. This ACT preserves both `F` and `F2` as
historical superseded freezes and publishes a third
corrected freeze `F3` whose plan mechanics are
executable as written.

The original ACT and its implementation body are
unchanged. Only the closure plan mechanics change.

## Base

```text
BASE_COMMIT=a5cfbf4adbde497dd0ca31961f7f6434683d9855
BASE_TREE=72398902a1cc298c17435f532cc65380ced70f23
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Plan-mechanics corrections

The F2 closure plan has nine executable defects.
F3 MUST fix all nine.

1. **Range diff-check with literal F2 OID.**

   Plan argv is executed directly; environment
   variables are not shell-expanded. Embed the
   literal F2 OID:

   ```json
   ["git", "diff", "--check", "a5cfbf4adbde497dd0ca31961f7f6434683d9855", "HEAD"]
   ```

   The diff is the comparison between F2 and HEAD
   at execution time, not a future `$S` identity.

2. **Focused ACT-owned exec-gate.**

   Add a separate check that runs the existing
   `forbidden-patterns` verifier (the focused
   exec-gate equivalent that the current binary
   exposes), not the aggregate lane:

   ```json
   ["bin/leamas", "factory", "verify", "forbidden-patterns"]
   ```

   This is the focused check that establishes
   `ACT_OWNED_EXEC_GATE_RESULT` independently of
   unrelated repository findings.

3. **Long-test-policy check.**

   Add a separate check that runs the existing
   `long-test-policy` verifier:

   ```json
   ["bin/leamas", "factory", "verify", "long-test-policy"]
   ```

   This satisfies
   `PLAN_FREEZE_LONG_TEST_POLICY=verified`.

4. **Aggregate gate with the correct CLI surface.**

   The current binary exposes the gate as
   `leamas factory gate --lane=fast`, not
   `leamas gate --lane=fast`. Use the verified
   surface:

   ```json
   ["bin/leamas", "factory", "gate", "--lane=fast"]
   ```

   Drop the speculative `--output` and
   `--json-output` flags; the existing
   `--json-output` flag has not been confirmed by
   help evidence in this digest. The aggregate
   lane is the one that records the literal
   `AGGREGATE_GATE_STATUS` and
   `PRE_EXISTING_GATE_FINDINGS`.

5. **Gate summary declared as a produced artifact.**

   Plan Contract v1 supports a produced-artifact
   list. The gate-summary check MUST declare:

   ```json
   "artifacts": [
     {
       "kind": "gate_summary",
       "path": ".factory/gate-summary.json",
       "produced_by": "v2-verifier-aggregate-fast-lane"
     }
   ]
   ```

   The artifact declaration binds the summary into
   the closure result; the runner MUST preserve it
   across the run, not delete it as ephemeral
   scratch.

6. **Run-scoped build destination.**

   Replace the hardcoded
   `.factory/build/v2verifier-r1` with a unique
   run-scoped directory the recipe creates itself
   and removes through a failure-safe trap:

   ```bash
   set -euo pipefail
   BUILD_DIR="$(mktemp -d -p .factory/build v2verifier-r1.XXXXXX)"
   out="$BUILD_DIR/leamas-v2verifier-r1"
   trap 'rm -rf "$BUILD_DIR"' EXIT
   CGO_ENABLED=0 go build -trimpath -o "$out" ./cmd/leamas
   ```

   No fixed path. No leftover residue on
   build failure.

7. **Worktree-neutral build.**

   Drop the `make build` check that may rewrite
   `bin/leamas` and violate the
   `require_clean_after` policy. The hermetic
   build is the only build check. Static-binary
   policy is verified separately by the existing
   `bin/leamas factory verify static-binary` check.

8. **Focused test selector with row-count authority.**

   The implementation MUST create stable tests
   that assert the matrix row counts. The plan
   names the tests it requires by exact name and
   asserts their row counts as `go test` arguments
   with `-v` and `-count=1`:

   ```text
   TestV2VerifierDuplicateMatrixHas56Rows
   TestV2VerifierExitThreeMatrixHas13Rows
   TestV2VerifierPublicNegativeMatrixHas16Rows
   TestV2VerifierTerminalTrailerMatrixHas15Rows
   TestV2VerifierJSONSingleDocumentUsesSecondDecode
   TestV2VerifierPreObservationCountersAreZero
   TestV2VerifierMetadataObsStateMachineHoldsInvariants
   TestV2VerifierHelpContractExitsZero
   ```

   The plan MUST select these tests by name and
   the implementation MUST add them as part of
   Phase 1-9.

9. **No literal future-`S` references in the plan.**

   F2's `git diff --check` already uses
   `HEAD` as the right endpoint, which the runner
   resolves against the implementation subject.
   F3 keeps that pattern. No plan file embeds
   a future commit OID that does not yet exist.

## Acceptance

```text
PLAN_FREEZE_RANGE_DIFF_LITERAL_F2=true
PLAN_FREEZE_FOCUSED_EXEC_GATE=focused_forbidden_patterns
PLAN_FREEZE_LONG_TEST_POLICY=verified
PLAN_FREEZE_AGGREGATE_GATE_CLI_SURFACE=verified
PLAN_FREEZE_GATE_SUMMARY_DECLARED=true
PLAN_FREEZE_BUILD_RUN_SCOPED=true
PLAN_FREEZE_BUILD_TRAP_SAFE=true
PLAN_FREEZE_BUILD_WORKTREE_NEUTRAL=true
PLAN_FREEZE_ROW_COUNT_TESTS_NAMED=true
PLAN_FREEZE_NO_FUTURE_S_LITERAL=true
F3_NE_F2=true
F3_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```

## Closure Protocol

```text
implementation_subject   = this correction ACT (single commit)
plan_freeze_commit       = F3 (this ACT)
closure_evidence_commit  = C  (next ACT or runner-emitted)
tag_target               = C
TOPOLOGY = F < F2 < F3 < S < C
```

`F` and `F2` are historical superseded freezes.
`F3` is this correction. `S` is the implementation
subject of the original ACT, distinct from `F3`.
`C` follows `S`.

## Publication

Exactly one implementation commit:

```text
factory: correct v2 verifier Mac handoff R1 plan F3
```

## Expected final status

```text
STATUS=PASS
PLAN_FREEZE_CORRECTION02_CLOSURE=true
F3_NE_F2=true
F3_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```
