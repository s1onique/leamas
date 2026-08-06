# Mac ClineMM Verifier Handoff (Correction01)

The ClineMM v2 closure verifier is ready for the Mac
handoff under the correction01 contract. This file
supersedes `clinemm-verifier-handoff.md` for ACTs that
require the exact correction01 verifier.

## Known anchors

```text
S = 56fd526e1923f2546fa0aeb53a0dc6e7501e5061
F = 01822bf5c8b99e5a4b89a6761a713ec3603754b0
```

The Mac operator must recover the following three values
by non-mutating inspection:

```text
P = frozen plan path at F
C = real closure commit after the v2 runner
M = committed v2 runner manifest path at C
```

`P` and `M` are recovered from the repository's tree at
`F` and `C` respectively; they are NOT inferred from
convention.

## Non-mutating inspection

Every command in this section reads from the ClineMM
repository and writes nothing to it. None of the
commands create refs, tags, worktrees, or branches.
None of the commands checkout anything.

```bash
# 1. Both commits must be resolvable in the object
#    database.
git -C <clinemm> cat-file -e \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061^{commit}

git -C <clinemm> cat-file -e \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0^{commit}

# 2. S must be a strict ancestor of F.
git -C <clinemm> merge-base --is-ancestor \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0

# 3. List the F tree so the operator can identify the
#    canonical plan path P.
git -C <clinemm> ls-tree -r --name-only \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0
```

After the v2 runner is executed and `C` is produced, the
operator can list the C tree to identify the canonical
manifest path `M`:

```bash
# 4. C must be a strict descendant of F.
git -C <clinemm> merge-base --is-ancestor \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0 \
  <C>

# 5. List the C tree to identify the canonical manifest
#    path M.
git -C <clinemm> ls-tree -r --name-only <C>
```

## Mac verify command

Once `P`, `C`, and `M` are recovered, the operator runs
the verifier against the ClineMM repository. The CLI
REJECTS `--output` paths that resolve inside the ClineMM
working tree BEFORE any Git observation.

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

`<mac-leamas-binary>` MUST be built from the exact
correction01 final commit with `vcs.modified=false`. The
binary's reported VCS revision MUST equal the correction01
final commit.

`<outside-clinemm-verification.json>` MUST live outside
the ClineMM working tree. The CLI rejects any output path
inside the ClineMM checkout.

## Expected outcome

The command exits 0 and emits a single deterministic
JSON envelope on stdout and the same envelope (with the
documented trailing-newline policy) in
`<outside-clinemm-verification.json>`. The envelope's
`verification.valid` is `true` and the typed
`verification` payload contains every literal S/F/C
identity and SHA-256 of the exact raw `F:P` and `C:M`
bytes.

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

## Evidence

The operator captures the bounded subprocess outcome
(literal `ExitCode`, `TimedOut`, `StdoutTruncated`,
`StderrTruncated`) and the typed JSON envelope. Both
live outside the ClineMM checkout. The envelope's
SHA-256 of the exact raw F:P and C:M bytes is
independently re-derivable via `git cat-file blob
<oid>` and SHA-256 over the literal bytes.
