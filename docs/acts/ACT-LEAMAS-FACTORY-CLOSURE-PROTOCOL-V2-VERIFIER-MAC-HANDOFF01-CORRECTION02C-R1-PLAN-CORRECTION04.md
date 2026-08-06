# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1-PLAN-CORRECTION04

## Status

OPEN — FINAL PLAN-MECHANICS CORRECTION

## Mission

The F4 freeze at
`7990de63f814bc48a927c56b1ba41ee63ee53324` is
docs-only but its plan still has six P0 mechanics
defects. This ACT preserves F, F2, F3, and F4 as
historical superseded freezes and publishes a fifth
corrected freeze `F5` whose plan mechanics are
executable as written and whose artifacts are
correctly bound.

This is the final freeze correction. After F5, the
implementation body of the original ACT may begin
against the F5 baseline.

## Base

```text
BASE_COMMIT=7990de63f814bc48a927c56b1ba41ee63ee53324
BASE_TREE=61badb546b6fba79a4d22f3f54e4dba81e6b1d7a
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Plan-mechanics corrections

F4 has six P0 mechanics defects. F5 MUST fix all
six.

1. **One aggregate fast-lane execution per closure
   run.**

   The previous plan invoked the fast lane three
   times (exec-gate extraction, aggregate status
   capture, gate-summary freshness). That can yield
   three different repository observations and
   wastes substantial execution time. F5 has a
   single check that runs the fast lane once and
   derives all evidence from the same bounded
   capture.

2. **One immutable raw gate capture.**

   The single check writes the raw fast-lane output
   to a single immutable file. All derived evidence
   is computed from that file. The file is SHA-256
   fingerprinted and recorded as the canonical
   capture for this closure run.

3. **Derive aggregate status, exec-gate status,
   summary hash, freshness, and pre-existing
   findings from the same capture.**

   A single bash wrapper runs the gate, captures
   the raw output, and writes derived evidence to
   `.factory/gate-summary.derived`. The wrapper
   records:
   - `aggregate_rc` (the literal exit code)
   - `raw_sha256` (the canonical capture fingerprint)
   - one line per lane with parsed status
   - one block of pre-existing findings

4. **Correct artifact producer/path bindings.**

   The previous plan declared the gate-summary
   artifact with one producer but wrote the summary
   to a different path. F5 binds the producer to
   the path it actually writes, and adds the
   derived-evidence file as a separate declared
   artifact.

5. **Fail on stale or missing required gate
   summary.**

   The previous freshness wrapper exited 0 in all
   three cases (`fresh`, `stale`, `missing`). F5
   changes the contract:
   - `fresh` exits 0
   - `stale` exits 2 with a typed diagnostic
   - `missing` exits 2 with a typed diagnostic

   The freshness check now also produces an explicit
   `.factory/gate-summary.json` that the
   derived-evidence wrapper writes from the same
   raw capture, so freshness is always fresh
   immediately after the gate runs.

6. **Distinguish evidence collection from acceptance
   verdicts.**

   The previous wrappers conflated evidence
   collection (always exit 0, record what
   happened) with acceptance verdicts (must satisfy
   the predicate). F5 splits them:
   - The aggregate-evidence wrapper always exits 0
     and records the literal status. Its exit
     code does NOT bind `AGGREGATE_GATE_STATUS`.
   - The acceptance verdict is derived from the
     captured evidence and the run-policy decision
     is made by the closure runner, not the wrapper.

   The same wrapper records both
   `EXEC_GATE_OBSERVED_STATUS` (the parsed lane
   status) and lets the runner bind
   `ACT_OWNED_EXEC_GATE_RESULT` according to the
   recorded policy.

7. **Enforce the built-binary predicates or
   downgrade the acceptance key.**

   The previous built-binary wrapper only captured
   `file`/`ldd`/`sha256sum` output. F5 enforces:
   - The binary MUST be statically linked.
   - The binary MUST execute.
   - The binary VCS identity MUST match S (when S
     is known).
   - `vcs.modified` MUST be `false` (when reported).

   The wrapper exits 1 on any predicate failure.
   The acceptance key `BUILT_BINARY_VERIFIED` is
   only true when all predicates pass.

8. **Bind the implementation diff and subject
   range to F5..S through executor-resolved freeze
   identity.**

   The plan MUST NOT embed a future F5 OID. The
   diff check is wrapped in a bash script that
   reads `F5_COMMIT` from the environment. The
   closure runner supplies `F5_COMMIT` after
   freezing the plan, before running the check.
   If `F5_COMMIT` is unset, the check fails
   immediately with a typed diagnostic.

9. **Add a digest self-consistency check.**

   A precheck that verifies the repository's own
   internal state matches the planned evidence:
   - HEAD tree OID matches what the closure
     runner expects
   - `.factory/gate-summary.json` raw SHA-256
     matches the captured raw output
   - every declared artifact exists with a fresh
     mtime within the run window

## Acceptance

```text
PLAN_FREEZE_SINGLE_FAST_LANE_RUN=true
PLAN_FREEZE_SINGLE_RAW_CAPTURE=true
PLAN_FREEZE_DERIVED_EVIDENCE_FROM_SAME_CAPTURE=true
PLAN_FREEZE_ARTIFACT_BINDINGS_CORRECT=true
PLAN_FREEZE_FRESHNESS_FAILS_ON_STALE=true
PLAN_FREEZE_FRESHNESS_FAILS_ON_MISSING=true
PLAN_FREEZE_EVIDENCE_VS_VERDICT_SEPARATED=true
PLAN_FREEZE_BUILT_BINARY_PREDICATES_ENFORCED=true
PLAN_FREEZE_DIFF_RANGE_EXECUTOR_RESOLVED=true
PLAN_FREEZE_DIGEST_SELF_CONSISTENCY=true
F5_NE_F4=true
F5_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```

## Closure Protocol

```text
implementation_subject   = this correction ACT (single commit)
plan_freeze_commit       = F5 (this ACT)
closure_evidence_commit  = C  (next ACT or runner-emitted)
tag_target               = C
TOPOLOGY = F < F2 < F3 < F4 < F5 < S < C
```

`F`, `F2`, `F3`, `F4` are historical superseded
freezes. `F5` is this final correction. `S` is the
implementation subject of the original ACT, distinct
from `F5`. `C` follows `S`.

## Publication

Exactly one implementation commit:

```text
factory: correct v2 verifier Mac handoff R1 plan F5
```

## Expected final status

```text
STATUS=PASS
PLAN_FREEZE_CORRECTION04_CLOSURE=true
F5_NE_F4=true
F5_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```
