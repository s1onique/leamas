# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION01

## Status

OPEN — EXACT-TIP DOGFOOD, LITERAL VERIFIER BINDINGS, AND READ-ONLY OUTPUT AUTHORITY

## Base

```text
BASE_COMMIT=ee0ab5265c1712ba2abcff5514a48924ebb06f07
BASE_TREE=a01b626d0b2ed95a4fe5b051e95293ff6a5c231c
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Mission

Correct the false-PASS closure of
`ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01`
by proving all of the following simultaneously:

1. the dogfood binary is built from the exact final commit;
2. no commit follows dogfood;
3. the detached build source remains clean;
4. the binary is written outside the build worktree;
5. bounded timeout and truncation outcomes are asserted;
6. every public verifier result binding is decoded and checked;
7. evidence contains literal observed values, never placeholders;
8. verifier output cannot mutate the repository being verified;
9. output publication is atomic;
10. observer exit classification is exact;
11. duplicate CLI flags fail closed;
12. optional tag metadata is verified or explicitly removed from the contract.

No ClineMM files may change.

## Phase 0 — lifecycle reconciliation

Record literal identities:

```text
ORIGINAL_DOGFOOD_BINARY_COMMIT=1fb07f055d9d1842401ff0e8f93afe01be0ec2fa
ORIGINAL_REPORTED_FINAL_COMMIT=ee0ab5265c1712ba2abcff5514a48924ebb06f07
ORIGINAL_DOGFOOD_MATCHED_FINAL_COMMIT=false
ORIGINAL_NO_LATER_COMMIT_RULE=false
ORIGINAL_LITERAL_EVIDENCE_COMPLETE=false
```

Do not preserve the previous `PASS` classification.

## Phase 1 — read-only output path authority

The CLI rejects `--output` paths that resolve inside the
target repository (or any of its linked worktrees) before
any Git observation. The verifier surfaces a typed
diagnostic:

```text
verifier_output_path_not_detached
```

The CLI does not create or truncate the output before the
verifier verdict is known.

## Phase 2 — atomic verifier output publication

Replace direct `os.WriteFile` publication with an atomic
writer:

```text
create temp file in destination directory
write complete bytes
fsync temp file where supported
close
rename temp -> final
fsync parent directory where supported
remove temp on every failure
```

The output bytes are generated once. JSON on stdout and
JSON on `--output` are byte-identical except for the
documented trailing-newline policy.

## Phase 3 — exact observer exit classification

Pin the exit contract:

```text
0 = valid verifier result
2 = CLI usage failure
3 = authoritative verification rejection
4 = observer/infrastructure failure
```

Observer failures include repository unavailable, object
format observation unavailable, Git spawn failure, Git
timeout, Git cancellation, Git output overflow,
unclassifiable Git exit, and state-capture observation
failure. Tests assert one exact exit code, never `3 or 4`.

JSON observer failure contains:

```json
{
  "ok": false,
  "failure_class": "observer",
  "verification": {
    "valid": false,
    "diagnostics": [...]
  }
}
```

No observer error may degrade into an empty `{ok:false}`
envelope.

## Phase 4 — duplicate CLI flag rejection

The parser rejects repeated occurrences of every scalar
flag, including repeats with the same value:

```text
--repository A --repository A
--subject S --subject S2
--json --json
--output A --output B
--expected-tag X --expected-tag X
```

Stable error: `duplicate flag: --<name>`. Exit: 2. No
filesystem read and no Git observation may happen after a
duplicate is detected.

## Phase 5 — tag metadata contract

The preferred contract validates canonical metadata
fields in the annotated tag message:

```text
closure_protocol_version=2
plan_contract_version=1
subject_commit=<S>
freeze_commit=<F>
closure_commit=<C>
plan_path=<P>
manifest_path=<M>
```

Reject: missing, duplicate, wrong protocol, wrong plan
contract, wrong S/F/C/P/M, or unknown mandatory metadata.

Stable code: `closure_tag_metadata_mismatch`.

The CLI never promises a contract it does not validate.

## Phase 6 — exact detached build source

The correction implementation, tests, ACT text, and
evidence schema are committed before dogfood. Create
exactly one final implementation commit:

```text
factory: correct v2 verifier Mac handoff authority
```

Then:

```text
FINAL_COMMIT=$(git rev-parse HEAD)
FINAL_TREE=$(git rev-parse HEAD^{tree})
```

Create a detached temporary worktree at `FINAL_COMMIT`.
Before building, assert literally:

```text
build HEAD == FINAL_COMMIT
build HEAD^{tree} == FINAL_TREE
build status --porcelain=v2 == empty
detached HEAD == true
```

Build the binary to a path outside the main Leamas
checkout, the detached build worktree, and the hermetic
target repository. After building, assert the build
source is still clean.

## Phase 7 — bounded subprocess outcome authority

For both `run-v2-authority` and `verify-v2-authority`:

```text
ExitCode == 0
TimedOut == false
StdoutTruncated == false
StderrTruncated == false
Err == nil
```

Apply the existing truncation rejection policy to the
actual result. Evidence fields are assigned from the
actual subprocess result, not left at zero values.

## Phase 8 — typed verifier result decoding

Decode the public JSON envelope into typed structures.
Do not retain `json.RawMessage` as the final assertion
boundary. Assert literally every identity in the
verifier's `V2ClosureVerification` result, including:

```text
envelope.ok == true
verification.valid == true
verification.topology_valid == true
verification.manifest_valid == true
verification.result_set_valid == true
verification.diagnostics length == 0
verification.repository_root == canonical target repository
verification.closure_protocol_version == 2
verification.plan_contract_version == 1
verification.subject_commit == S
verification.subject_tree == S^{tree}
verification.freeze_commit == F
verification.freeze_tree == F^{tree}
verification.closure_commit == C
verification.closure_tree == C^{tree}
verification.plan_path == P
verification.plan_blob == independently resolved F:P blob
verification.plan_sha256 == SHA-256(exact F:P bytes)
verification.manifest_path == M
verification.manifest_blob == independently resolved C:M blob
verification.manifest_sha256 == SHA-256(exact C:M bytes)
```

Independently resolve bytes with `git cat-file blob <oid>`.
Git's raw-object contract is the byte authority.

## Phase 9 — negative public-dogfood matrix

The installed-style dogfood invokes the public CLI
against mutations and asserts one exact exit code, one
exact diagnostic code, `valid=false`, no success summary,
target repository state unchanged, and detached output
absent on pre-publication failure.

## Phase 10 — complete caller-state proof

Capture before and after:

```text
HEAD commit
HEAD tree
status --porcelain=v2 --untracked-files=all
worktree list --porcelain
for-each-ref canonical snapshot
```

Compare raw canonical bytes and SHA-256. Populate literal
evidence fields.

## Phase 11 — durable literal evidence

Use explicit JSON-safe fields (`run_error_present`,
`run_error_text`) instead of error interface fields.
Validate the evidence before publication: all required
strings nonempty, all OIDs 40 lowercase hex, all SHA-256
values 64 lowercase hex, all boolean outcome fields
assigned, exit code assigned.

Write:

```text
correction01-evidence.json
correction01-evidence.json.sha256
```

Publish both atomically enough that a stale sidecar
cannot accompany new evidence.

## Phase 12 — literal final report

Generate the report directly from the validated
evidence. Required literal fields include
`ACT_ID`, `STATUS`, `BASE_COMMIT`, `BASE_TREE`,
`FINAL_COMMIT`, `FINAL_TREE`, `CURRENT_HEAD`,
`CURRENT_TREE`, `WORKTREE_STATUS`,
`DOGFOOD_BINARY_COMMIT`, `DOGFOOD_BINARY_SHA256`,
`DOGFOOD_VCS_REVISION`, `DOGFOOD_VCS_MODIFIED`,
`RUNNER_*`, `VERIFIER_*`, `DOGFOOD_*`,
`CALLER_*`, `DOGFOOD_EVIDENCE_*`.

The report does not embed its own digest. The report
lives outside the Leamas checkout.

## Phase 13 — exact-final-tip lifecycle

After the one correction commit: build, dogfood, write
detached evidence, generate detached report output. Do
not commit any file afterward.

At final handoff require:

```text
CURRENT_HEAD == FINAL_COMMIT
CURRENT_TREE == FINAL_TREE
COMMITS_AFTER_FINAL == 0
WORKTREE_STATUS == clean
DOGFOOD_BINARY_COMMIT == FINAL_COMMIT
DOGFOOD_VCS_REVISION == FINAL_COMMIT
DOGFOOD_VCS_MODIFIED == false
```

## Phase 14 — Mac handoff correction

Update the handoff so it requires:

```text
exact correction FINAL_COMMIT binary
vcs.modified=false
output outside ClineMM
evidence outside ClineMM
literal P recovered from F
literal C supplied externally
literal M recovered from C
bounded invocation
full typed JSON result inspection
caller state captured before and after
```

The Mac operation rejects an output path inside ClineMM
before Git observation. No ClineMM file may be modified
by this ACT.

## Publication

Exactly one forward commit:

```text
factory: correct v2 verifier Mac handoff authority
```

No later close-artifact commit.

## Expected final status

```text
STATUS=PASS
UNRESOLVED_BLOCKERS=None
DOGFOOD_BINARY_MATCHES_FINAL_COMMIT=true
COMMITS_AFTER_FINAL=0
VERIFIER_VALID=true
CLINEMM_FILES_CHANGED=none
MAC_HANDOFF=ready
```
