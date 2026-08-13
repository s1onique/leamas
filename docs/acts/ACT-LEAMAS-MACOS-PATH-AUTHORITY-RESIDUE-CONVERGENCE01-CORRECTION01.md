# ACT-LEAMAS-MACOS-PATH-AUTHORITY-RESIDUE-CONVERGENCE01-CORRECTION01

## STATUS

```text
OPEN
```

## MISSION

Restore fail-closed authority to the `RootResolver.SplitRepoPath`
changes shipped in `ACT-LEAMAS-MACOS-PATH-AUTHORITY-RESIDUE-CONVERGENCE01`
(subject `a81d88d`). The previous ACT introduced a walk-up
fallback for non-existent leaves and a non-`IsNotExist`
`EvalSymlinks` short-circuit, but it then silently converted
canonicalization failures, walk-up failures, and `filepath.Rel`
failures into an apparent success tuple `(root, "", nil)`. That
collapses three genuinely different failure modes into one
indistinguishable "successful" return, violating the Leamas
fail-closed contract.

The previous ACT also routed every `os.Lstat` error in
`canonicalizeExistingPrefix` through the same walk-up branch,
silently treating permission failures, I/O errors, and
symlink-loop errors as though the path were merely nonexistent.

This correction propagates every non-`IsNotExist` failure
distinctly, distinguishes absent components from unreadable ones
inside the walk-up loop, and adds adversarial tests for the
adversarial matrix the previous ACT left implicit.

## OWNED RESIDUE

The previous ACT's owned failure set:

```text
R01 TestInventoryRepositoryWorktrees_RealGitPorcelainZAndNewlinePath
R02 TestConfineDestination_RootNameMatchesOpenedParent
R03 TestPublication_PostPublishCloseFailureIsObserverState
R04 TestPublication_Success
R05 TestPublication_AcceptsExistingFile
R06 TestPublication_TempFilesAbsentAfterSuccess
R07 TestPublication_SetPermission
R08 TestPublication_DoublePublishFails
R09 TestPublication_CloseBeforePublishIsStateInvariant
R10 TestPublication_IO_DestinationReadBackRoundTrip
R11 TestPublication_AuthoritativeDirectory
R12 TestRootResolver_SplitRepoPath
```

All twelve are PASS at the start of this correction; this ACT
must keep them PASS.

## REQUIRED BEHAVIOR

`SplitRepoPath` propagates non-`IsNotExist` `EvalSymlinks` errors.

`canonicalizeExistingPrefix` distinguishes `os.IsNotExist` from
other `Lstat` failures (permission, I/O, symlink-loop).

`filepath.Rel` failure is propagated; the function MUST NOT
silently return `(root, "", nil)`.

The `IsNotExist` walk-up fallback is preserved for the legitimate
"future output leaf" lifecycle, where the repo root exists but
the leaf does not.

## ADVERSARIAL MATRIX

Require:

```text
normal repository                   PASS
symlinked ancestor                  PASS
linked worktree                     PASS
nonexistent leaf                    PASS (walk-up to existing prefix)
symlink loop                        FAIL CLOSED (typed error)
permission-denied ancestor          FAIL CLOSED (typed error)
unreadable ancestor                 FAIL CLOSED (typed error)
malformed absPath                   FAIL CLOSED (typed error)
Rel-failure (different volume)      FAIL CLOSED (typed error)
```

Forbidden:

```text
silently substituting empty string for relPath
treating permission errors as nonexistent
treating symlink loops as nonexistent
swallowing filepath.Rel errors
```

## OUT OF SCOPE

- Out-of-scope per parent ACT Section 4: 73 unrelated baseline failures.
- No new ACT ID for the simplified-closure product; that is
  `ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01`, a future
  product ACT that bootstraps itself via the explicit F/S/C
  protocol exactly once.

## COMMIT

```text
fix(forbidden): make SplitRepoPath fail closed on canonicalization errors
```

Subject commit on top of `a81d88d`; no amend; no rebase.
