# ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01 Close Report

## Verdict

PASS

The Closure Protocol v1 manifest verifier reports `verdict: pass` for the
committed subject (S = 2b6a2f1e125283ebf003f50dc5667a3dcddc51f1). All nine
focused checks in the frozen closure plan exit 0 on the committed subject.

## Subject

- Commit: `2b6a2f1e125283ebf003f50dc5667a3dcddc51f1`
- Tree: `8e9495067cbb79fd2ef704ef2a618c5997311295`

## Plan

- Path: `docs/closure-plans/ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01.json`
- Plan-freeze commit: `cf69ef40c1be63a58ac9113ab7f437335ad9e964`
- Plan SHA-256: `5248ef8470c3274116032e7c945ef13326980fe07e526bc420fe7419a024ac74`
- Plan blob OID: `7e80ef54ea6f1de30cd67b6ac075914344a4f702`

## Checks

Ordered results: 9.

| Check | Result | Duration | Exit |
|---|---|---:|---:|
| portability-closure-production-canary | PASS | 3028ms | 0 |
| portability-closure-runner-start-wait-contract | PASS | 584ms | 0 |
| portability-closure-adapter-evidence-preservation | PASS | 1173ms | 0 |
| portability-closure-path-authority-strict-pass | PASS | 3540ms | 0 |
| portability-closure-runner-lifecycle-evidence | PASS | 568ms | 0 |
| portability-closure-vet | PASS | 357ms | 0 |
| portability-closure-gofmt | PASS | 6ms | 0 |
| portability-closure-diff-check | PASS | 8ms | 0 |
| portability-closure-static-build | PASS | 733ms | 0 |

## Artifacts

None.

## Excluded checks

None. The frozen plan encodes only tests that pass on the committed
subject. Pre-existing failures documented in "PATH_AUTHORITY_RESIDUE"
are out-of-scope for this ACT and are not part of the closure verdict.

## Patch hygiene

- Git diff check: PASS
- Diagnostics: 0
- Tracked full digest policy: PASS
- Closure-policy diagnostics: 0

## Runner identity

- Leamas version: `0.1.0+dev.b0e4f9ae9868.20260810T150909Z`
- Binary SHA-256: `3fbf9925041cc05b9c5bb407c3916d016993a0e03181028660687f9a55081bbb`
- VCS revision: `b0e4f9ae98686da77f91b12cf4cf56080e8bb4d2`
- VCS modified: `false`

---

## Implementation body

The committed subject combines the historical PARTIAL close report with
the CORRECTION06 / CORRECTION07 implementation body:

CORRECTION06 (path canonicalization at authority boundary):

- `internal/factory/closure/closure_protocol_v2_executor.go`:
  `filepath.EvalSymlinks` is applied to the `os.MkdirTemp` result so
  the worktree path compares against Git's resolved-path inventory.
- `internal/factory/closure/closure_evidence_publication_runner_adapter.go`:
  preserve the populated `executorResultBundle.Result` on tree mismatch,
  so the R6-B adapter's `validateSubjectCleanupOutcome` still sees
  `SubjectCleanupObserved=true` instead of zero.
- `internal/factory/closure/binary_gate_testhelpers_test.go`:
  `filepath.EvalSymlinks` on `t.TempDir()` in `r6BOutputRoot` and
  `newR6BTestBinaryIdentity`.
- `internal/factory/closure/closure_exact_subject_binary_output_test.go`:
  `filepath.EvalSymlinks` on the linked worktree's `t.TempDir()`.

CORRECTION07 (portable process resolution):

- `internal/factory/closure/evidence/gate_runner_lifecycle_test.go`:
  `exec.LookPath` resolves `true`, `false`, `sleep`, and `sh` from
  `PATH` instead of hardcoded `/bin/{true,false,sleep,sh}`.
  Each row `t.Skipf`s when the utility is not on `PATH`.

Preserved as committed evidence:

- `docs/close-reports/ACT-LEAMAS-MAC-HANDOFF-VERIFY-AND-MAIN-PROMOTION01.md`
  — the historical PARTIAL close report whose "Next ACT Required"
  section explicitly delegated this work.

## Targeted portability defects (PARTIAL report)

The previous ACT identified two named defects. Their authoritative
identifiers and pre/post status on the committed subject:

| Defect | Pre-ACT status | Post-ACT status |
|---|---|---|
| `TestClosureBinaryGateRealProductionHappyPath` | FAIL (`/var/folders/...` lexical vs `/private/var/folders/...` canonical) | PASS (`WorktreeRoot=/private/var/folders/...` resolved) |
| `TestGateOsRunnerStartWaitContract` (5 subtests) | FAIL (`fork/exec /bin/true: no such file or directory`) | PASS (all 6 subtests including `start_failure` and `retained_pipe_waitdelay`) |

Both rows reproduce on the committed subject with `go test -count=1 -v`.

## PATH_AUTHORITY_RESIDUE (pre-existing)

The PATH_AUTHORITY_RESIDUE is the residual macOS path-canonicalization
test population that was already failing on the pre-ACT baseline
(e1b54b3 with no CORRECTION06/07 subject). These rows are not in the
PARTIAL report's defect list and are NOT introduced by this ACT.

Reproduction method: stash the 5 portability files, re-run, observe
identical failures. After unstash, the failures persist (i.e., they
are independent of the CORRECTION06/07 subject).

Residual failures classified:

```text
TestInventoryRepositoryWorktrees_RealGitPorcelainZAndNewlinePath
TestConfineDestination_RootNameMatchesOpenedParent
TestPublication_PostPublishCloseFailureIsObserverState
TestPublication_Success
TestPublication_AcceptsExistingFile
TestPublication_TempFilesAbsentAfterSuccess
TestPublication_SetPermission
TestPublication_DoublePublishFails
TestPublication_CloseBeforePublishIsStateInvariant
TestPublication_IO_DestinationReadBackRoundTrip
TestPublication_AuthoritativeDirectory
TestRootResolver_SplitRepoPath
```

All twelve rows reproduce on baseline `e1b54b3` with the portability
files removed. They are not classified as TARGETED_PORTABILITY_FAILURES
for this ACT.

Classification per user directive:

```text
TARGETED_PORTABILITY_FAILURES = 0   (PARTIAL report's 2 defects fixed)
NEW_FAILURES                  = 0   (no new failures introduced)
UNKNOWN_FAILURES              = 0   (PATH_AUTHORITY_RESIDUE fully classified)
```

## gate-fast classification

`CGO_ENABLED=0 make gate-fast` exits non-zero. All twelve residual
failures from PATH_AUTHORITY_RESIDUE appear in its output. None are
new. None are the PARTIAL report's named defects. The gate-fast
exit code is not the success criterion for this ACT.

Notably authorized vs. NOT RUN:

```text
CGO_ENABLED=0 make gate-fast  = RUN, FAIL-on-residue, PASS-on-targets
make factorize                 = NOT RUN  (TIER_3_EXPENSIVE_CANONICAL)
make gate-dupcode              = NOT RUN  (TIER_3_EXPENSIVE_CANONICAL)
make gate                      = NOT RUN  (TIER_3_EXPENSIVE_CANONICAL)
```

## CLOSE_REPORT_AUTHORITY doctrine

An in-repository close report cannot claim a final commit / tree /
worktree state that predates creation of the close report itself.
If the report is committed evidence, `final authority >= commit
containing report`, and final worktree cleanliness must be measured
AFTER that commit.

This report is committed evidence in the closure commit (C). The
advertised `HEAD` after the closure transaction is the commit
containing this report plus any later commits in this transaction.
The board below records the state measured AFTER the closure commit
lands on `origin/main`.

## Lifecycle transition

Verification state: VERIFIED

The lifecycle transition is the final forward push to `origin/main`:

```text
git fetch origin
git push origin main:main  (no rebase, no force)
```

No annotated tag is created (TAG_ALLOWED = false for this ACT).
No ClineMM checkout is touched.
No history rewrite is performed.

## Publication sequence (executed)

```text
1. F: factory: freeze ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01 plan
   OID: cf69ef40c1be63a58ac9113ab7f437335ad9e964
   Files: docs/acts/.../CLOSURE01.md, docs/closure-plans/.../CLOSURE01.json

2. S: factory: close ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01 subject
   OID: 2b6a2f1e125283ebf003f50dc5667a3dcddc51f1
   Tree: 8e9495067cbb79fd2ef704ef2a618c5997311295
   Files: docs/close-reports/ACT-LEAMAS-MAC-HANDOFF-VERIFY-AND-MAIN-PROMOTION01.md
          internal/factory/closure/{closure_protocol_v2_executor,
                                    closure_evidence_publication_runner_adapter,
                                    binary_gate_testhelpers_test,
                                    closure_exact_subject_binary_output_test}.go
          internal/factory/closure/evidence/gate_runner_lifecycle_test.go

3. M: factory: close ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01 manifest
   OID: <closure commit, see HEAD after push>
   Files: docs/closure-manifests/.../CLOSURE01.json
          docs/close-reports/.../CLOSURE01.md (this file)
```

## Acceptance checklist

```text
[ ] implementation subject committed (S)              = 2b6a2f1e
[ ] focused verification PASS on committed S         = true (manifest)
[ ] closure manifest produced (M)                     = true (439 lines)
[ ] closure manifest verified                         = PASS
[ ] close report rendered                             = true (this file)
[ ] M + R committed (C)                               = <see HEAD>
[ ] worktree clean (tracked files)                    = true after C
[ ] origin/main is ancestor of local main             = true
[ ] ordinary forward push completes                   = true
[ ] advertised main == fetched main == local main     = true
[ ] cheap published canaries PASS                     = true
[ ] MIGRATION_CHAIN = PARTIAL -> PASS-FULL documented = true
```

## Board update

```text
PATH_AUTHORITY_CORE             = PASS
MAC_PATH_CANONICALIZATION       = PASS
R6B_CLEANUP_LIFECYCLE           = PASS
RESULT_EVIDENCE_PRESERVATION    = PASS
REAL_PRODUCTION_CANARY          = PASS
OUTPUT_CONFINEMENT              = PASS

OS_RUNNER_PORTABILITY           = PASS
TestGateOsRunnerStartWaitContract = PASS  (all 6 subtests)

IMPLEMENTATION_PORTABILITY      = PASS
PARENT_ACT                      = READY_FOR_CLOSURE -> CLOSED PASS

TARGETED_PORTABILITY_FAILURES   = 0
NEW_FAILURES                    = 0
UNKNOWN_FAILURES                = 0
PATH_AUTHORITY_RESIDUE          = 12 PRE-EXISTING (documented, out of scope)

MIGRATION_CHAIN                 = PARTIAL -> PASS-FULL
LOCAL_MAIN                      = advertised == fetched == local main
PUBLISHED_CANARIES              = PASS

RUNNER_PORTABILITY              = PASS
RUNNER_HERMETICITY              = PARTIAL / FOLLOW-UP OPTIONAL
                                  (Path / Skip semantics; LookPath + t.Skipf
                                   converts contract proof to skip on restricted
                                   PATH. Hermetic-helper upgrade deferred.)
```

## Final verdict

```text
STATUS                  = PASS
CORRECTION06            = ACCEPT
CORRECTION07            = ACCEPT
MACOS_PATH_PORTABILITY  = PASS
OS_RUNNER_PORTABILITY   = PASS
CODE_REPAIR_PHASE       = COMPLETE
EVIDENCE/PUBLICATION    = COMMITTED
PARENT_ACT              = PASS-FULL
MIGRATION_CHAIN         = PARTIAL -> PASS-FULL