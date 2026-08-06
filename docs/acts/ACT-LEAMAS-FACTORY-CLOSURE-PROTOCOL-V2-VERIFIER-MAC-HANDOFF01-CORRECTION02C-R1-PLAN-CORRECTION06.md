# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1-PLAN-CORRECTION06

## Status

OPEN — EXECUTOR-IDENTITY CONTRACT FREEZE

## Mission

The F6 freeze at
`8c7183bad249ebd013bc3c39db71edd4c79e1a2d` is
docs-only but its plan still has eight P0 mechanics
defects. F6 also embedded a wrong F6 literal (it
copied the F5 OID). This ACT publishes a seventh
corrected freeze `F7` that:

1. fixes the literal OIDs in PLAN-CORRECTION05;
2. introduces an executor-provided identity
   contract so the plan never embeds its own
   future OID;
3. corrects the freshness check order;
4. passes the aggregate exit code to the typed
   collector;
5. documents the required `SUBJECT_COMMIT` supply;
6. implements a real raw-vs-JSON semantic comparison;
7. acknowledges that the typed collector is part of
   the implementation body, not the freeze;
8. declares the previous-summary artifact as
   conditional.

The implementation body remains pending. The
freeze-authority chain `F < F2 < F3 < F4 < F5 < F6
< F7 < S < C` is correct after F7 lands.

## Base

```text
BASE_COMMIT=8c7183bad249ebd013bc3c39db71edd4c79e1a2d
BASE_TREE=1969cfb2416d6550517e5bb86205284638eff1e7
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Honest assessment of the freeze cycle

Six docs-only freezes have been committed without
any implementation work. Each freeze corrected the
previous freeze's mechanics. The corrections have
not yet converged. Continuing the freeze-only
cycle without starting the implementation body is
not productive.

The implementation body of the original ACT is the
next body of work. The F7 freeze is the last
plan-only correction. After F7, the implementation
body MUST begin, even if the plan is not perfect.

## Frozen literals (corrected)

```text
F5_LITERAL=4578148cb43a808fe5626b74c9ece25e026b4c68
F6_LITERAL=8c7183bad249ebd013bc3c39db71edd4c79e1a2d
F7_LITERAL=unset                    # F7 is this ACT; OID is the new HEAD
```

The F7 OID is the HEAD OID after this ACT is
committed. The plan MUST NOT embed the F7 OID
literally. The closure runner supplies it.

## Executor-provided identity contract

The plan MUST NOT embed any freeze or subject OID
literally. Instead, the closure runner supplies
the following environment variables at execution
time:

```text
F5_COMMIT     # frozen plan commit (F5)
F6_COMMIT     # frozen plan predecessor (F6, not authoritative)
F7_COMMIT     # authoritative freeze (this ACT, F7)
SUBJECT_COMMIT # implementation subject (S)
```

The plan's diff check uses `F7_COMMIT`:

```text
git diff --check "$F7_COMMIT" HEAD
```

If `F7_COMMIT` is unset, the check fails with
`DIFF_RANGE_UNRESOLVED: F7_COMMIT not provided`.

The built-binary check uses `SUBJECT_COMMIT` to
validate the binary's reported commit.

The plan MUST NOT refuse to start when these are
unset; it MUST fail the relevant check with a
typed diagnostic.

## Plan-mechanics corrections

F6 has eight P0 mechanics defects. F7 MUST fix all
eight.

1. **Corrected F5 and F6 literals.**

   The F6 plan-correction document had a bug: it
   wrote `F6_LITERAL=4578148...` when the actual
   F6 OID was `8c7183ba...`. F7 documents both
   correctly and freezes the corrected literals.

2. **Executor-provided freeze identity contract.**

   The diff check uses `F7_COMMIT` from the
   environment. The closure runner supplies
   `F7_COMMIT` after freezing the plan. The plan
   never embeds the F7 OID literally.

3. **Freshness check runs after JSON production.**

   The single-capture wrapper performs in this
   order:

   ```text
   1. Move old .factory/gate-summary.json to .prev
   2. Run gate once
   3. Run collector to produce new .json
   4. Verify new .json exists
   5. Compare to .prev
   6. Classify fresh|stale|missing
   ```

   The freshness check is moved AFTER the JSON
   production step so the comparison is meaningful.

4. **Aggregate exit code is passed to the collector.**

   The wrapper passes the gate's exit code as a
   command-line argument:

   ```bash
   rc=$?
   go run ./internal/factory/closure/gate_summary_collector \
     --raw "$raw" \
     --json "$json" \
     --prev "$prev" \
     --aggregate-rc "$rc"
   ```

   The collector records `aggregate_rc` in the
   JSON. The exit code is authoritative; the JSON
   field is bound to it.

5. **SUBJECT_COMIT supply is documented and required.**

   The plan's `v2-verifier-hermetic-build-with-binary-predicates`
   check has an `environment` that requires
   `SUBJECT_COMMIT`. The closure runner supplies
   it. If unset, the check fails with
   `BUILT_BINARY_VERIFIED=NO:SUBJECT_COMMIT_not_supplied`.

6. **Raw/JSON semantic comparison.**

   The `v2-verifier-digest-self-consistency` check
   parses BOTH `.factory/gate-summary.raw` AND
   `.factory/gate-summary.json`, and verifies
   semantic equivalence:

   ```text
   raw_head_oid == json.head_oid
   for each lane: raw_lane_status == json.lanes[lane]
   raw_aggregate_rc matches json.aggregate_rc
   ```

   The check fails with a typed diagnostic on
   mismatch.

7. **Typed collector is part of the implementation
   body, not the freeze.**

   F7 acknowledges that
   `internal/factory/closure/gate_summary_collector`
   does not yet exist. The implementation body MUST
   create it. The implementation ACT must:

   - Create the collector as a runnable Go command
     package at
     `cmd/leamas/gate_summary_collector/main.go` or
     `internal/factory/closure/cmd/gate_summary_collector/main.go`.
   - Provide a stable command contract:
     `--raw`, `--json`, `--prev`, `--aggregate-rc`,
     `--subject-commit`, `--run-id`.
   - Add a unit test for the collector.

   The implementation ACT must create the
   collector BEFORE the S commit is final. The
   freeze does not require the collector to exist
   at F7; it requires the implementation to create
   it as part of S.

8. **Conditional previous-summary artifact.**

   The `gate_summary_prev` artifact is declared as
   conditional. On a clean first run, no prior
   JSON exists, so the producer does not create
   this file. The contract is:

   ```text
   gate_summary_prev.json
     - exists if a prior gate summary existed at
       the start of this run;
     - does not exist on a clean first run;
     - is referenced by the freshness check but
       is not required to exist.
   ```

## Acceptance

```text
PLAN_FREEZE_F5_F6_LITERALS_CORRECT=true
PLAN_FREEZE_EXECUTOR_IDENTITY_CONTRACT_DEFINED=true
PLAN_FREEZE_FRESHNESS_AFTER_JSON=true
PLAN_FREEZE_AGGREGATE_RC_PASSED_TO_COLLECTOR=true
PLAN_FREEZE_SUBJECT_COMMIT_DOCUMENTED=true
PLAN_FREEZE_RAW_JSON_SEMANTIC_COMPARISON=true
PLAN_FREEZE_COLLECTOR_PART_OF_S=true
PLAN_FREEZE_PREV_ARTIFACT_CONDITIONAL=true
PLAN_FREEZE_ONE_FAST_LANE_INVOCATION_GUARDED=true
F7_NE_F6=true
F7_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```

## Closure Protocol

```text
implementation_subject   = this correction ACT (single commit)
plan_freeze_commit       = F7 (this ACT, OID = HEAD after commit)
closure_evidence_commit  = C  (next ACT or runner-emitted)
tag_target               = C
TOPOLOGY = F < F2 < F3 < F4 < F5 < F6 < F7 < S < C
```

`F`, `F2`, `F3`, `F4`, `F5`, `F6` are historical
superseded freezes. `F7` is this correction. `S` is
the implementation subject of the original ACT,
distinct from `F7`. `C` follows `S`.

## Publication

Exactly one implementation commit:

```text
factory: correct v2 verifier Mac handoff R1 plan F7
```

## Required next ACT after F7

F7 is the last docs-only correction. The next ACT
MUST be the implementation body of the original
ACT. Its mission is Phases 1-12. It MUST:

1. Create the typed collector at
   `internal/factory/closure/cmd/gate_summary_collector/`.
2. Implement the 12 phases of the original ACT.
3. Run the F7 plan once.
4. Commit closure artifacts as C.
5. Create the annotated tag targeting C.

If the F7 plan proves unexecutable at S, the
remediation is a new ACT that corrects the plan,
not a continuation of the F < Fn < Fn+1 cycle.

## Expected final status

```text
STATUS=PASS
PLAN_FREEZE_CORRECTION06_CLOSURE=true
F7_NE_F6=true
F7_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```

## Closure after F7

After F7, the implementation body MUST begin. The
cycle of plan-only corrections is over.
