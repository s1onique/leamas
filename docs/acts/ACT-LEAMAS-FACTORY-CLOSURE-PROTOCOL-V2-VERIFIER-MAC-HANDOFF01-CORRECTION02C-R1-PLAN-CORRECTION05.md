# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1-PLAN-CORRECTION05

## Status

OPEN — FINAL PLAN-MECHANICS CORRECTION

## Mission

The F5 freeze at
`4578148cb43a808fe5626b74c9ece25e026b4c68` is
docs-only but its plan still has six P0 mechanics
defects. This ACT preserves F, F2, F3, F4, and F5 as
historical superseded freezes and publishes a sixth
corrected freeze `F6` whose plan mechanics are
executable as written and whose artifacts are
correctly bound with full evidence.

## Base

```text
BASE_COMMIT=4578148cb43a808fe5626b74c9ece25e026b4c68
BASE_TREE=9379732ed623518cd81a135c5a6189c4facadc34
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Frozen literals

F6 is now known and immutable:

```text
F6_LITERAL=4578148cb43a808fe5626b74c9ece25e026b4c68
```

The F5 OID that becomes the implementation baseline
is now also known:

```text
F5_LITERAL=4578148cb43a808fe5626b74c9ece25e026b4c68
```

These literals MAY be embedded in the executable
plan. The closure runner does not need to inject
them at execution time.

## Plan-mechanics corrections

F5 has six P0 mechanics defects. F6 MUST fix all
six.

1. **F5 identity is now literal in the plan.**

   The diff check uses the literal F5 OID:

   ```json
   ["git", "diff", "--check", "4578148cb43a808fe5626b74c9ece25e026b4c68", "HEAD"]
   ```

   The closure runner does NOT need to inject
   anything. The diff check is range-bound from
   authoritative F5 to HEAD.

   The plan also freezes the literal F6 OID as
   the F5..HEAD range start for the next ACT. The
   plan MUST NOT require any runner-supplied
   identity substitution for the range check.

2. **Stale-summary baseline is atomically captured.**

   The single-capture wrapper MUST atomically
   move any existing `.factory/gate-summary.json`
   to `.factory/gate-summary.prev.json` BEFORE
   running the gate. The freshness check then
   compares the new summary to the prev file:

   ```bash
   set -euo pipefail
   if [ -e .factory/gate-summary.json ]; then
     mv .factory/gate-summary.json .factory/gate-summary.prev.json
   fi
   bin/leamas factory gate --lane=fast > .factory/gate-summary.raw 2>&1
   # ... rest of capture
   if [ -e .factory/gate-summary.prev.json ] && \
      cmp -s .factory/gate-summary.prev.json .factory/gate-summary.json; then
     echo 'GATE_SUMMARY_FRESHNESS=stale'; exit 2
   fi
   ```

   The previous summary is gone after the gate
   runs. The freshness check cannot be satisfied
   by reading a file that was never produced in
   this run.

3. **Self-consistency compares JSON to raw output
   semantically, not by SHA-256.**

   The self-consistency check MUST parse the raw
   output and verify the JSON contents match the
   raw, lane by lane. It MUST NOT compare SHA-256
   hashes of different representations:

   ```bash
   set -euo pipefail
   raw=.factory/gate-summary.raw
   json=.factory/gate-summary.json
   # Extract aggregate_rc from JSON
   json_rc="$(jq -r '.aggregate_rc' "$json")"
   # Compute aggregate_rc from raw (exit code captured separately)
   raw_rc="$(awk '/^EXIT_RC=/' .factory/gate-summary.derived | cut -d= -f2)"
   if [ "$json_rc" != "$raw_rc" ]; then
     echo 'DIGEST_SELF_CONSISTENCY=aggregate_rc_mismatch'; exit 2
   fi
   # For each lane in JSON, verify the JSON lane status matches the raw lane line
   for lane in agent-context docs doctrine doctrine-agent-contracts domain-boundaries exec-gate executable-contract-first forbidden-patterns git-hooks language llm-friendly; do
     json_status="$(jq -r ".lanes[\"$lane\"]" "$json")"
     raw_status="$(awk -v l="$lane" '$0 ~ "^--- " l " " {print $3; exit}' "$raw")"
     if [ "$json_status" != "$raw_status" ]; then
       echo "DIGEST_SELF_CONSISTENCY=lane_mismatch:$lane"; exit 2
     fi
   done
   echo 'DIGEST_SELF_CONSISTENCY=ok'
   ```

   This requires `jq` to be available. The plan
   MUST list `jq` as a prerequisite.

4. **Built-binary predicates are exhaustive.**

   The built-binary wrapper MUST:

   - Invoke the exact built binary's `version`
     command and parse the machine-readable
     identity.
   - Verify the binary executes (exit 0).
   - Verify `vcs.modified=false` when reported.
   - Verify the reported commit matches the
     executor-supplied `SUBJECT_COMMIT`.
   - Capture the binary path and SHA-256 as
     independently validated typed fields.

   ```bash
   set -euo pipefail
   out="$BUILD_DIR/leamas-v2verifier-r1"
   # Build ...
   # Verify execution
   if ! "$out" version > .factory/evidence/built-binary.version 2>&1; then
     echo 'BUILT_BINARY_VERIFIED=NO:execution_failed'; exit 1
   fi
   # Parse identity
   version_line="$(grep '^version:' .factory/evidence/built-binary.version | head -1)"
   commit_line="$(grep '^commit:' .factory/evidence/built-binary.version | head -1)"
   modified_line="$(grep '^vcs_modified:' .factory/evidence/built-binary.version | head -1)"
   reported_commit="$(echo "$commit_line" | sed 's/^commit: //')"
   reported_modified="$(echo "$modified_line" | sed 's/^vcs_modified: //')"
   # Compare with subject
   if [ -z "${SUBJECT_COMMIT:-}" ]; then
     echo 'BUILT_BINARY_VERIFIED=NO:SUBJECT_COMMIT_not_supplied'; exit 1
   fi
   if [ "$reported_commit" != "$SUBJECT_COMMIT" ]; then
     echo "BUILT_BINARY_VERIFIED=NO:commit_mismatch:$reported_commit:$SUBJECT_COMMIT"; exit 1
   fi
   if [ "$reported_modified" != 'false' ]; then
     echo "BUILT_BINARY_VERIFIED=NO:vcs_modified:$reported_modified"; exit 1
   fi
   echo 'BUILT_BINARY_VERIFIED=YES'
   ```

5. **Canonical JSON contains all required evidence.**

   The single-capture wrapper MUST produce a JSON
   summary with every required field. The JSON
   MUST be written by a typed producer (a small
   Go helper), not by shell `echo` concatenation:

   ```go
   // internal/factory/closure/gate_summary_collector.go
   func CollectGateSummary(rawPath, jsonPath, derivedPath string) error {
     // Read raw
     raw, err := os.ReadFile(rawPath)
     // Parse lanes from raw
     // Compute SHA-256s
     // Parse pre-existing findings
     // Determine EXEC_GATE_OBSERVED_STATUS
     // Determine ACT_OWNED_EXEC_GATE_RESULT per rule
     // Write JSON
   }
   ```

   The wrapper invokes this Go helper. The JSON
   contains:
   - `aggregate_rc`
   - `raw_sha256`
   - `raw_path`
   - `captured_at`
   - `lanes` (object with one entry per known lane)
   - `pre_existing_findings` (array)
   - `exec_gate_observed_status`
   - `act_owned_exec_gate_result`
   - `subject_commit` (when supplied)
   - `head_oid`
   - `head_tree`
   - `run_id`

6. **ACT-owned exec-gate verdict rule is deterministic.**

   `ACT_OWNED_EXEC_GATE_RESULT` is bound by this
   rule:

   ```text
   if EXEC_GATE_OBSERVED_STATUS == OK
      and all pre-existing findings are out of scope:
         ACT_OWNED_EXEC_GATE_RESULT = PASS
   elif EXEC_GATE_OBSERVED_STATUS in {OK, FAILED}
      and any pre-existing finding is in scope:
         ACT_OWNED_EXEC_GATE_RESULT = FAIL
   elif EXEC_GATE_OBSERVED_STATUS in {SKIP, UNKNOWN}:
         ACT_OWNED_EXEC_GATE_RESULT = UNAVAILABLE
   else:
         ACT_OWNED_EXEC_GATE_RESULT = FAIL
   ```

   "In scope" is defined as: the finding's file
   path is under `cmd/leamas/factory_close_v2_*`
   or `internal/factory/closure/v2_verifier_*`.
   The wrapper applies the rule and writes the
   verdict to the JSON summary.

## Acceptance

```text
PLAN_FREEZE_F5_IDENTITY_LITERAL=true
PLAN_FREEZE_STALE_BASELINE_ATOMIC=true
PLAN_FREEZE_SELF_CONSISTENCY_SEMANTIC=true
PLAN_FREEZE_BUILT_BINARY_PREDICATES_EXHAUSTIVE=true
PLAN_FREEZE_JSON_FULL_EVIDENCE=true
PLAN_FREEZE_EXEC_GATE_VERDICT_RULE=true
PLAN_FREEZE_ONE_FAST_LANE_INVOCATION_GUARDED=true
F6_NE_F5=true
F6_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```

## Closure Protocol

```text
implementation_subject   = this correction ACT (single commit)
plan_freeze_commit       = F6 (this ACT)
closure_evidence_commit  = C  (next ACT or runner-emitted)
tag_target               = C
TOPOLOGY = F < F2 < F3 < F4 < F5 < F6 < S < C
```

`F`, `F2`, `F3`, `F4`, `F5` are historical superseded
freezes. `F6` is this final correction. `S` is the
implementation subject of the original ACT, distinct
from `F6`. `C` follows `S`.

## Publication

Exactly one implementation commit:

```text
factory: correct v2 verifier Mac handoff R1 plan F6
```

## Expected final status

```text
STATUS=PASS
PLAN_FREEZE_CORRECTION05_CLOSURE=true
F6_NE_F5=true
F6_NE_BASE=true
LLM_FRIENDLY_ACT_FILES=PASS
```
