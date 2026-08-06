# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01

## Status

OPEN — CLI AND READ-ONLY-STATE PROOF FOR THE V2 CLOSURE VERIFIER

## Mission

Expose the verifier publicly and prove it is read-only,
caller-state independent, and optionally capable of
validating an annotated closure tag.

This ACT owns only:

1. public CLI wiring (`leamas factory close verify-v2-authority`);
2. text and JSON output contracts;
3. dedicated help contract;
4. optional annotated-tag assertion;
5. repository read-only guarantee;
6. dirty worktree independence;
7. CLI test matrix.

It does not own exact-tip final dogfood or Mac handoff.

## Phase 1 — public command

The CLI exposes:

```text
leamas factory close verify-v2-authority
```

with required flags:

```text
--repository
--protocol-version 2
--plan-contract-version 1
--subject S
--freeze  F
--closure C
--plan-path P
--manifest-path M
```

and optional flags:

```text
--working-manifest-assertion <file>
--expected-tag <name>
--output <path>
--json
--capture-caller-state
--help
```

The CLI never infers `C` from HEAD, `M` from convention, or
`P` from the working tree. Every required field is typed and
required; the parser rejects unknown flags before any Git
observation.

## Phase 2 — CLI outcome contract

Text success:

```text
exit 0
single summary line: factory close verify-v2-authority subject=… freeze=… closure=… manifest_sha256=… plan_sha256=… valid=true
no ambiguous partial output
```

Text failure:

```text
exit 3 (verifier) or 4 (observer)
typed diagnostics
no success summary
```

JSON success/failure:

```text
stdout: exactly one JSON document
        stable envelope {ok, verification:{…}}
stderr: empty on success; diagnostics on failure
exit code preserved from underlying run
```

## Phase 3 — help contract

The dedicated `--help` (and empty) invocation prints the
verifier contract:

```text
S = execution subject
F = frozen-plan authority
C = closure commit (NOT inferred from HEAD)
P loaded from F
M loaded from C
HEAD is not authority
C need not appear inside M
```

## Phase 4 — optional annotated tag assertion

When `--expected-tag` is supplied, the verifier asserts:

```text
tag exists
tag is annotated (not lightweight)
tag targets C exactly
```

A lightweight tag triggers `closure_tag_lightweight`. A
mismatched target triggers `closure_tag_target_mismatch`.
When the flag is absent the verifier does not look at
refs.

## Phase 5 — read-only state capture

The optional `--capture-caller-state` flag snapshots caller
state before and after, then verifies byte-for-byte
equality under:

```text
HEAD commit
HEAD tree
porcelain-v2 status
worktree list --porcelain
refs snapshot
```

A mutation triggers `state_mutation_detected`. The capture
MUST NOT mutate the target repository.

## Phase 6 — dirty worktree independence

Verification works when the worktree is dirty (untracked
files, tracked modifications). Authority remains pinned to
Git objects (S/F/C, F:P, C:M); the dirty state never
influences the verdict.

## Phase 7 — CLI matrix

Covered matrix:

```text
--help                       exit 0 + usage
no arguments                 exit 2
missing required flag        exit 2
unknown flag                 exit 2
unsupported protocol         exit 2
unsupported plan contract    exit 2
non-repository path          exit 3 or 4
JSON success                 single doc on stdout
JSON failure                 single doc on stdout
```

## Publication

Exactly one commit:

```text
factory: expose v2 closure verifier CLI
```

## Acceptance

Closed only when:

1. public command exists;
2. required flags are explicit;
3. text output is stable;
4. JSON is one document;
5. tag assertion is optional and strict;
6. verifier is read-only;
7. dirty worktree does not affect authority;
8. state remains unchanged on all paths;
9. help documents non-self-reference;
10. no ClineMM files change.

## Final report

Closed ACT produces:

```text
ACT_ID
STATUS
FINAL_COMMIT
FINAL_TREE
WORKTREE_STATUS

VERIFIER_COMMAND
CLI_HELP_RESULT
CLI_TEXT_MATRIX
CLI_JSON_MATRIX
CLI_EXIT_CODES

TAG_ASSERTION_MODE
TAG_MATRIX
TAG_RESULT

CALLER_STATE_MATRIX
DIRTY_WORKTREE_RESULT
READ_ONLY_RESULT
REFS_UNCHANGED_RESULT
WORKTREES_UNCHANGED_RESULT

LOCAL_GATES
REFUSED_EXPENSIVE_GATES
PRE_EXISTING_GATE_FINDINGS
UNRESOLVED_BLOCKERS
```

## Expected blockers

```text
exact-tip installed dogfood and Mac ClineMM handoff (ACT 5).
```
