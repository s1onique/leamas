# Mac ClineMM Verifier Handoff

The ClineMM v2 closure verifier is ready for the Mac handoff.
This file is the operator command template referenced by
`ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01`.

## Known anchors

```text
S = 56fd526e1923f2546fa0aeb53a0dc6e7501e5061
F = 01822bf5c8b99e5a4b89a6761a713ec3603754b0
```

The Mac operator must recover the following three values by
non-mutating inspection:

```text
P = frozen plan path at F
C = real closure commit after the v2 runner
M = committed v2 runner manifest path at C
```

`P` and `M` are recovered from the repository's tree at `F`
and `C` respectively; they are NOT inferred from convention.

## Non-mutating inspection

Every command in this section reads from the ClineMM
repository and writes nothing to it. None of the commands
create refs, tags, worktrees, or branches. None of the
commands checkout anything.

```bash
# 1. Both commits must be resolvable in the object
#    database. cat-file -e returns 0 if the object
#    exists, 1 otherwise.
git -C <clinemm> cat-file -e \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061^{commit}

git -C <clinemm> cat-file -e \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0^{commit}

# 2. S must be a strict ancestor of F.
git -C <clinemm> merge-base --is-ancestor \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0

# 3. List the F tree so the operator can identify the
#    canonical plan path P. The frozen plan path MUST
#    live under docs/closure-plans/ in the F tree.
git -C <clinemm> ls-tree -r --name-only \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0
```

After the v2 runner is executed and `C` is produced, the
operator can list the C tree to identify the canonical
manifest path `M`:

```bash
# 4. C must be a strict descendant of F (post-run).
git -C <clinemm> merge-base --is-ancestor \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0 \
  <C>

# 5. List the C tree so the operator can identify the
#    canonical manifest path M. The committed manifest
#    path MUST live under docs/closure-manifests/ in the
#    C tree.
git -C <clinemm> ls-tree -r --name-only <C>
```

## Mac verify command

Once `P`, `C`, and `M` are recovered, the operator runs
the verifier against the ClineMM repository. The verifier
NEVER writes to the ClineMM checkout; the `--output` flag
points at a directory outside the ClineMM working tree.

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

`<mac-leamas-binary>` is the Leamas binary built on the
Mac from the same source tree that produced
`factory: prove v2 closure verifier readiness`.

`<outside-clinemm-verification.json>` is a path on a
filesystem OUTSIDE the ClineMM working tree. The verifier
must never write inside the target repository.

## Expected outcome

The command exits 0 and prints a single line:

```text
factory close verify-v2-authority subject=56fd526e1923f2546fa0aeb53a0dc6e7501e5061 freeze=01822bf5c8b99e5a4b89a6761a713ec3603754b0 closure=<C> manifest_sha256=<sha256> plan_sha256=<sha256> valid=true
```

The `--output` file is a single deterministic JSON
document whose `verification.valid` is `true` and whose
`verification.subject_commit`, `freeze_commit`,
`closure_commit`, `plan_blob`, `plan_sha256`,
`manifest_blob`, and `manifest_sha256` fields all match
the values the operator can independently verify with
`git cat-file blob <oid>` and `git hash-object <path>`.

## Caller-state guarantee

The verifier is read-only. Before and after the
invocation, the ClineMM repository's:

```text
HEAD commit
HEAD tree
porcelain-v2 status
worktree list
refs snapshot
```

are byte-for-byte identical. The verifier never
checkouts, never creates worktrees, never creates or
removes refs, never creates or removes tags, and never
writes inside the target repository.
