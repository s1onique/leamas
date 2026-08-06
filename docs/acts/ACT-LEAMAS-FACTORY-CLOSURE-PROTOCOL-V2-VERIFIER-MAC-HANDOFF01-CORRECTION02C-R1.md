# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1

## Status

OPEN — INTERNAL-CONSISTENCY, METADATA-READING, AND PUBLIC-NEGATIVE-CLI
CORRECTIONS

## Base

```text
BASE_COMMIT=5ba0cf0cfcb14e133ea4425c930a2badade0e0fd
BASE_TREE=5392ad67261d55405fbff4f84c1d959c6fbbcd51
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Parent verdict

The closure report of
`ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C`
classified the ACT as `PASS`, but the verdict document
disagrees. The verdict records ten P0 blockers and
several report defects and demands a follow-up ACT
narrowly scoped to fixing them. Until those are closed:

```text
STATUS=PARTIAL
CORRECTION02C_IMPLEMENTATION_CLOSURE=false
CORRECTION02C_R1_REQUIRED=true
CORRECTION02D_READY=false
```

This ACT is that follow-up.

## Parent ACT constraint (frozen)

The parent ACT explicitly froze the exit taxonomy:

```text
unsafe output path supplied by caller -> exit 2
```

That classification is the authoritative reading. The
R1 work freezes the same reading across the help text,
the constant, the writers, and the public tests.

## Reference documents

Detailed matrices and enumerations live in the support
document, which is referenced normatively by this ACT:

```text
docs/acts/support/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C-R1-MATRICES.md
```

It contains §1 prescribed 14-flag × 4-form duplicate
matrix, §2 public-negative CLI matrix, §3 adversarial
trailer / terminal-block cases, §4 JSON single-document
EOF rationale. The support document does not relax any
acceptance criterion stated here.

## Mission

Correct the ten P0 blockers and the report defects
recorded in the verdict. Scope is intentionally narrow.

1. **Separate annotated-tag OID from peeled target commit.**

   The structural tag assertion holds two fields with
   distinct identities. The implementation MUST rename
   the existing `tagAssertion.Target` to:

   ```text
   tagAssertion.TargetCommitOID  (the peeled commit)
   ```

   and add a new field:

   ```text
   tagAssertion.TagObjectOID     (the annotated tag object)
   ```

   The metadata reader MUST load bytes via
   `git cat-file tag <TagObjectOID>`. It MUST NOT pass
   a commit OID where a tag object is required. The
   orchestrator passes `TagObjectOID` to the metadata
   reader and compares `TargetCommitOID` against the
   closure commit `C`.

2. **Bind metadata validity to a direct predicate.**

   The verification's tag-validity predicate MUST be:

   ```text
   tagValid =
     structural tag assertion valid
     AND Observed
     AND Parsed
     AND Bound
     AND metadata diagnostics empty
   ```

   `Observed`, `Parsed`, and `Bound` are the
   authoritative phase-state facts emitted by the
   metadata observation. Diagnostics are the
   authoritative failure details. There is exactly
   one source of truth: the three phase facts plus
   the empty-diagnostics invariant.

   `Parsed` implies `Observed`. `Bound` implies
   `Parsed`. A field mismatch yields
   `Observed=true && Parsed=true && Bound=false`
   plus the corresponding `closure_tag_metadata_*`
   diagnostic. An unreadable tag yields
   `Observed=true && Parsed=false && Bound=false`
   plus `closure_tag_unreadable`.

   The orchestrator's `assembleVerification` MUST
   accept `tagMetadataObs` as an explicit parameter
   and MUST read phase-state facts and diagnostics
   from that argument, never from an undefined
   local:

   ```text
   assembleVerification(
       req, topology, frozenPlan, committedManifest,
       optionalAssertion, tagAssertion,
       tagMetadataObs, manifestFacts, combinedDiags,
   )
   ```

3. **Freeze the unsafe-output classification as exit 2.**

   The `--output` path is caller-supplied. The CLI
   rejects unsafe paths before any Git observation,
   so the rejection is a usage-class failure
   (`exit 2`), not an observer-class failure. Freeze
   the classification in:

   - `v2VerifierExitUsage` constant;
   - the help text under `Exit codes`;
   - the `parseV2VerifierFlags` rejection branch;
   - the text-mode writer;
   - the JSON-mode writer;
   - the public negative matrix.

   In particular, a non-repository target path that
   makes Git observation impossible MUST surface as
   exactly `exit 4` (observer failure), never as
   `exit 3` (verifier rejection). Tests that accept
   `exit 3` or `exit 4` for the same condition MUST
   be tightened to assert `exit 4` exactly.

4. **Replace the claimed duplicate matrix.**

   The previous ACT claimed `14 flags × 2-3 spellings
   each` but the actual matrix covered only value
   flags in two spellings and three boolean flags in
   a separate test. Replace the hand-written table
   with a generated prescribed-form matrix. The full
   matrix is in support document §1.

5. **Add explicit pre-observation counters.**

   Inject a counting seam:

   ```text
   GitObservationCount
   WorktreeInventoryCount
   OutputPreparationCount
   WorkingAssertionReadCount
   ```

   The duplicate-flag, unsafe-output, and help paths
   MUST be proved by asserting all four counters are
   zero AFTER the rejection.

6. **Implement the real public negative CLI matrix.**

   Remove the no-op `TestV2VerifierExitThreeVerifierMatrix`
   loop. Replace it with a real matrix that drives
   the public `runFactoryCloseVerifyV2Authority`
   entry point against each public failure family.
   The full matrix is in support document §2. Each
   row asserts one exact exit code, one exact
   diagnostic code, `valid=false`, no success
   summary, target repository state unchanged, and
   detached output absent on pre-publication failure.

7. **Enforce the TERMINAL_TRAILER_BLOCK and reject
   every malformed Leamas-* line.**

   The tag-body model is a **mixed terminal trailer
   suffix**. The final non-empty region of the tag
   body is a trailer block. The block may contain
   unrelated trailers. The seven required Leamas
   keys must occur exactly once within that block.
   Lines outside the block are not part of the
   trailer region.

   The block is identified by:

   ```text
   1. Remove only terminal empty lines caused by the
      tag message's final newline convention.
   2. Then identify the longest contiguous suffix of
      syntactically valid trailer lines.
   3. Within that suffix, the seven Leamas keys must
      each appear exactly once. Non-Leamas trailers
      in the same suffix are allowed.
   4. A Leamas- line in the body that is NOT in the
      terminal trailer block is malformed.
   ```

   Detection MUST inspect the raw trimmed line
   prefix BEFORE parsing. A `Leamas-` line that
   yields `key=""` from `parseV2ClosureTagTrailerLine`
   MUST still be classified as
   `closure_tag_metadata_malformed`, never silently
   dropped.

   Classification by line shape:

   ```text
   Leamas-Unknown: value              -> unknown
   Leamas-Known value                 -> malformed (no colon)
   Leamas-Known:value                 -> malformed (no space)
   Leamas-Known:  value               -> malformed (extra space)
   Leamas-Known: <empty>              -> malformed (empty value)
   ```

   `unknown` is reserved for keys with the correct
   shape but not in the closed metadata set.
   `malformed` is reserved for known keys with bad
   shape. Both must yield `Valid=false`. The full
   adversarial case list is in support document §3.

8. **Correct the JSON EOF assertions.**

   After decoding the single JSON envelope, the EOF
   check MUST be a second `json.Decoder.Decode` that
   returns `io.EOF`. The current `dec.Token()` call
   conflates token-state errors with absence of
   trailing JSON or garbage. The full rationale and
   the affected test files are in support document §4.

9. **Record literal full commit/tree OIDs and truthful
   gate/lifecycle evidence.**

   The closure digest MUST record `BASE_COMMIT`,
   `BASE_TREE`, `FINAL_COMMIT`, `FINAL_TREE` as
   40-char lowercase hex OIDs; `CLINEMM_FILES_CHANGED=none`;
   `ACT_OWNED_EXEC_GATE_RESULT=PASS` for a focused
   exec-gate verification of the files this ACT owns
   (a separate, narrow check, NOT a partial result
   of the aggregate fast lane); `GATE_SUMMARY_PRESENT=true`;
   `AGGREGATE_GATE_STATUS` as the literal result
   from the full fast lane (a failed aggregate lane
   does not block this ACT when findings are
   pre-existing and unrelated to the missions listed
   above); `PRE_EXISTING_GATE_FINDINGS` as the
   literal list of pre-existing findings; and the
   closure plan, manifest, close report, and
   lifecycle anchors at the standard locations.

No new verifier architecture beyond the predicate and
OID split above. No new ClineMM files may change.

## Acceptance

```text
TAG_OBJECT_OID_SEPARATED=true
TARGET_COMMIT_OID_RENAMED_FROM_TARGET=true
ASSEMBLE_VERIFICATION_TAKES_TAG_METADATA_OBS=true
METADATA_OBSERVED_PARSED_BOUND_DIAGS_EMPTY=true
UNSAFE_OUTPUT_EXIT=2
NON_REPOSITORY_EXIT=4
DUPLICATE_MATRIX_PRESCRIBED_COVERAGE=14
DUPLICATE_MATRIX_PRESCRIBED_FORMS_PER_FLAG=4
PRE_OBSERVATION_COUNTERS_INJECTED=true
DUPLICATE_REJECTION_BEFORE_OBSERVATION=true
UNSAFE_OUTPUT_REJECTION_BEFORE_OBSERVATION=true
PUBLIC_NEGATIVE_CLI_MATRIX_IMPLEMENTED=true
NO_OP_EXIT_THREE_TEST_REMOVED=true
TERMINAL_TRAILER_BLOCK_ENFORCED=true
LEAMAS_DASH_MALFORMED_LINE_REJECTED=true
LEAMAS_DASH_UNKNOWN_LINE_REJECTED=true
RAW_LINE_MALFORMED_DETECTION=true
TERMINAL_BLANK_LINE_NORMALIZATION=true
TERMINAL_BLOCK_ADVERSARIAL_TESTS_PINNED=true
JSON_EOF_ASSERTION_USES_SECOND_DECODE=true
BASE_COMMIT_LITERAL=true
BASE_TREE_LITERAL=true
FINAL_COMMIT_LITERAL=true
FINAL_TREE_LITERAL=true
CLINEMM_FILES_CHANGED=none
ACT_OWNED_EXEC_GATE_RESULT=PASS
GATE_SUMMARY_PRESENT=true
AGGREGATE_GATE_STATUS=literal
PRE_EXISTING_GATE_FINDINGS=literal
TOPOLOGY=F_LT_S_LT_C
F_NE_S=true
S_NE_C=true
F_NE_C=true
PLAN_FREEZE_PRECEDES_SUBJECT=true
PLAN_FREEZE_DISTINCT_FROM_SUBJECT=true
IMPLEMENTATION_COMMITS_AFTER_FREEZE=1
CLOSURE_EVIDENCE_FOLLOWS_SUBJECT=true
LIFECYCLE_CLOSURE_COMMITS_PERMITTED=true
LLM_FRIENDLY_ACT_FILES=PASS
```

## Closure Protocol

This ACT uses Closure Protocol v1 with the strict
F < S < C topology:

```text
PLAN_FREEZE_COMMIT=F     # frozen plan
IMPLEMENTATION_SUBJECT=S # exactly one implementation commit
CLOSURE_EVIDENCE_COMMIT=C # manifest + close report

TOPOLOGY = F < S < C
F_NE_S = true
S_NE_C = true
F_NE_C = true

IMPLEMENTATION_COMMITS_AFTER_FREEZE = 1
LIFECYCLE_CLOSURE_COMMITS_PERMITTED = true
```

Publication order:

```text
1. Commit the ACT document, support document, and
   frozen plan at F. F is a strict ancestor of S
   and is NOT equal to S.
2. Implement all production and test changes in
   exactly one commit S. S is a strict ancestor of
   C and is NOT equal to C.
3. Run verification against F:S.
4. Commit the manifest and close report at C.
5. Create the annotated closure tag targeting C.
```

Closure artifacts and commands:

```text
docs/closure-plans/<ACT-ID>.json        # at F
docs/closure-manifests/<ACT-ID>.json    # at C
docs/close-reports/<ACT-ID>.md          # at C
leamas factory close plan   <ACT-ID>
leamas factory close run    <ACT-ID>
leamas factory close verify <ACT-ID>
leamas factory close render <ACT-ID>
leamas factory close tag    create <ACT-ID>
leamas factory close status <ACT-ID>
```

The frozen plan never embeds future closure, tree,
tag, or commit identities or raw evidence.

## Publication

The implementation subject S is exactly one commit:

```text
factory: correct v2 verifier Mac handoff R1
```

The plan freeze commit F precedes S and is distinct
from S. The closure evidence commit C follows S.
Both are required for closure.

## Expected final status

```text
STATUS=PASS
CORRECTION02C_R1_CLOSURE=true
CORRECTION02D_READY=true
UNRESOLVED_BLOCKERS=None
CLINEMM_FILES_CHANGED=none
ACT_OWNED_EXEC_GATE_RESULT=PASS
GATE_SUMMARY=published
LIFECYCLE_ANCHORS=published
TOPOLOGY=F_LT_S_LT_C
```
