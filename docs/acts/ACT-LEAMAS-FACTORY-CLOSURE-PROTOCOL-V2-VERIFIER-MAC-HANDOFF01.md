# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01

## Status

OPEN — INSTALLED-STYLE FINAL DOGFOOD AND REAL MAC CLINEMM HANDOFF

## Mission

Build the exact final verifier binary, run installed-style
verification against a meaningful hermetic `S < F < C`
repository, capture literal evidence, and produce the real
Mac ClineMM handoff.

This ACT owns only:

1. detached exact build at the final subject commit;
2. a meaningful hermetic `S < F < C` repository with a
   real committed manifest `M` at `C`;
3. installed-style `factory close verify-v2-authority`
   dogfood from outside the Leamas checkout;
4. literal evidence captured outside both repositories;
5. a no-later-commit guarantee;
6. the Mac ClineMM handoff command template (non-mutating
   inspection + verify commands).

It does not own new verifier architecture unless dogfood
reveals a defect.

## Phase 0 — prepare all source before final commit

Prepare before creating the final commit:

```text
ACT spec (this file)
closure plan
Mac handoff command template
installed-style dogfood harness (Go test)
```

The harness reuses the existing
`factory_close_v2_r2c_helpers_test.go` build, git, and
SHA-256 helpers. The harness lives at
`cmd/leamas/factory_close_v2_verifier_mac_handoff_test.go`.

## Phase 1 — exact final commit

Create exactly one commit:

```text
factory: prove v2 closure verifier readiness
```

Record:

```text
FINAL_COMMIT
FINAL_TREE
CURRENT_HEAD
WORKTREE_STATUS
COMMITS_AFTER_BASE
```

`FINAL_COMMIT` is the commit that introduces the
verifier readiness subject.

## Phase 2 — detached exact build

Create a temporary detached worktree at `FINAL_COMMIT` and
build the leamas binary there. The build must inject the
production LDFLAGS so the running binary identity reports
the actual `FINAL_COMMIT` and `vcs.modified=false`.

Require:

```text
HEAD = FINAL_COMMIT
tree = FINAL_TREE
status before = clean
status after  = clean
binary VCS revision     = FINAL_COMMIT
binary VCS modified     = false
```

## Phase 3 — meaningful hermetic repository

Construct, in a fresh temp directory outside the Leamas
checkout:

```text
S:
  subject-only file
  baseline tree (S^{tree})

F:
  child of S
  valid plan P at docs/closure-plans/MAC-HANDOFF.json
  freeze-only file
  F:^{tree} != S^{tree}

C:
  child of F
  valid committed manifest M at docs/closure-manifests/MAC-HANDOFF.json
  closure-only report file
  C:^{tree} != F^{tree}
```

Construction order:

1. Build S with a `subject-only.txt` file.
2. Build F with the plan P and a `freeze-only.txt` file.
3. Run the production v2 runner (same binary, in detached
   subprocess) to produce a manifest bytes blob. The runner
   binds against S and F and writes the manifest to a
   detached path.
4. Commit the manifest bytes as a new file at
   `docs/closure-manifests/MAC-HANDOFF.json` on top of F.
   That commit is C.
5. C must NOT contain `closure_commit` in its committed
   manifest bytes. The manifest is the one produced by the
   runner; the verifier never requires C to appear inside
   M.

Require:

```text
S < F < C
execution_tree = S^{tree}   (asserted via the committed M)
plan_blob     = F:P blob    (asserted via the committed M)
manifest_blob = C:M blob
manifest SHA-256 matches the committed bytes
```

## Phase 4 — installed-style CLI dogfood

Run from a temp working directory OUTSIDE both the Leamas
checkout and the hermetic repository:

```bash
<leamas-binary> factory close verify-v2-authority \
  --protocol-version 2 \
  --plan-contract-version 1 \
  --repository <hermetic-repo> \
  --subject <S> \
  --freeze  <F> \
  --closure <C> \
  --plan-path docs/closure-plans/MAC-HANDOFF.json \
  --manifest-path docs/closure-manifests/MAC-HANDOFF.json \
  --output <detached-outside/verification.json>
```

Require:

```text
exit 0
not timed out
stdout not truncated
stderr not truncated
valid=true
subject=    <S>
freeze=     <F>
closure=    <C>
manifest_sha256 = <SHA-256(C:M)>
plan_sha256     = <SHA-256(F:P)>
```

## Phase 5 — independent state proof

Before and after the dogfood invocation:

```text
HEAD                  = <C>
HEAD tree             = <C^{tree}>
porcelain-v2 status   = <unchanged>
worktree inventory    = <unchanged>
refs snapshot         = <unchanged>
```

The harness MUST NOT create or remove worktrees, refs, or
tags. No worktree creation, no tag/ref mutation unless a
separate optional tag fixture is explicitly part of the
dogfood.

## Phase 6 — literal evidence

Write deterministic evidence outside the Leamas checkout
and outside the hermetic repository.

Required literal fields (no placeholders):

```text
binary SHA-256
stdout SHA-256
stderr SHA-256
verification result SHA-256
evidence SHA-256

S / S^{tree}
F / F^{tree}
C / C^{tree}

P
F:P blob
F:P SHA-256

M
C:M blob
C:M SHA-256

caller state before/after
worktree inventory before/after
refs before/after
```

Use a detached checksum sidecar (the same pattern the
existing R2C-R3 evidence writer uses). The on-disk evidence
file does not embed its own SHA-256; the SHA-256 is in a
sibling sidecar so the digest is verifiable externally.

## Phase 7 — no-later-commit rule

After dogfood:

```text
HEAD = FINAL_COMMIT
worktree clean
no later commit
```

If any source file changes, create a new final commit and
repeat build plus dogfood. The closure protocol
implementation MUST support a re-run path.

## Phase 8 — Mac ClineMM handoff

Known Mac ClineMM anchors:

```text
S = 56fd526e1923f2546fa0aeb53a0dc6e7501e5061
F = 01822bf5c8b99e5a4b89a6761a713ec3603754b0
```

The Mac operator must recover three values by non-mutating
inspection:

```text
P = frozen plan path at F
C = real closure commit after the v2 runner
M = committed v2 runner manifest path at C
```

Non-mutating inspection (no checkout, no ref creation):

```bash
git -C <clinemm> cat-file -e \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061^{commit}

git -C <clinemm> cat-file -e \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0^{commit}

git -C <clinemm> merge-base --is-ancestor \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0

git -C <clinemm> ls-tree -r --name-only \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0
```

After the v2 runner is executed and `C` is produced:

```bash
<mac-leamas-binary> factory close verify-v2-authority \
  --protocol-version 2 \
  --plan-contract-version 1 \
  --repository <clinemm> \
  --subject 56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
  --freeze  01822bf5c8b99e5a4b89a6761a713ec3603754b0 \
  --closure <C> \
  --plan-path <P> \
  --manifest-path <M> \
  --output <outside-clinemm-verification.json>
```

The output path MUST live outside the ClineMM checkout so
the verifier never writes inside the target repository.

## Publication

Exactly one commit:

```text
factory: prove v2 closure verifier readiness
```

## Acceptance

Closed only when:

1. exact-final-tip binary built;
2. meaningful `S < F < C` dogfood passes;
3. plan and manifest object bindings are exact;
4. verifier reports `valid=true`;
5. output is bounded and untruncated;
6. target repository remains unchanged;
7. literal evidence is captured with sidecar digest;
8. no later commit exists;
9. Mac commands are non-mutating;
10. no ClineMM files change.

## Final report

Closed ACT produces:

```text
ACT_ID
STATUS
BASE_COMMIT
BASE_TREE
FINAL_COMMIT
FINAL_TREE
CURRENT_HEAD
WORKTREE_STATUS
COMMITS_AFTER_BASE

DOGFOOD_BINARY_COMMIT
DOGFOOD_BINARY_SHA256
DOGFOOD_VCS_REVISION
DOGFOOD_VCS_MODIFIED

DOGFOOD_EXIT
DOGFOOD_TIMED_OUT
DOGFOOD_STDOUT_TRUNCATED
DOGFOOD_STDERR_TRUNCATED
DOGFOOD_STDOUT_SHA256
DOGFOOD_STDERR_SHA256
DOGFOOD_RESULT_SHA256

DOGFOOD_SUBJECT
DOGFOOD_SUBJECT_TREE
DOGFOOD_FREEZE
DOGFOOD_FREEZE_TREE
DOGFOOD_CLOSURE
DOGFOOD_CLOSURE_TREE

DOGFOOD_PLAN_PATH
DOGFOOD_PLAN_BLOB
DOGFOOD_PLAN_SHA256

DOGFOOD_MANIFEST_PATH
DOGFOOD_MANIFEST_BLOB
DOGFOOD_MANIFEST_SHA256

CALLER_HEAD_BEFORE
CALLER_HEAD_AFTER
CALLER_TREE_BEFORE
CALLER_TREE_AFTER
CALLER_STATUS_BEFORE_SHA256
CALLER_STATUS_AFTER_SHA256
WORKTREE_INVENTORY_BEFORE_SHA256
WORKTREE_INVENTORY_AFTER_SHA256
REFS_BEFORE_SHA256
REFS_AFTER_SHA256

DOGFOOD_EVIDENCE_PATH
DOGFOOD_EVIDENCE_SIDECAR_PATH
DOGFOOD_EVIDENCE_SHA256

LOCAL_GATES
REFUSED_EXPENSIVE_GATES
PRE_EXISTING_GATE_FINDINGS
UNRESOLVED_BLOCKERS

CLINEMM_FILES_CHANGED
MAC_INSPECTION_COMMANDS
MAC_VERIFY_COMMAND
MAC_HANDOFF
```

Expected final status:

```text
STATUS=PASS
UNRESOLVED_BLOCKERS=None
CLINEMM_FILES_CHANGED=none
MAC_HANDOFF=ready
```
