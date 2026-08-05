# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1 Close Report

## Verdict

PASS

## Subject

- Commit: `25010d160c6b04edc24ec4602af951541ef1ffa8`
- Tree: `5dcd0864ad5b81c22ab2ac924d7f394ec1e2bb9a`

## Plan

- Path: `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1.json`
- SHA-256: (ACT-R1 is a runner-authority correction; it is closed
  through the v2 hermetic + CLI subprocess tests, not through the
  v1 closure verifier. No frozen plan is required for an
  authority-only ACT.)

## Checks

Ordered results: 6 (R1 fail-closed observation matrix).

| Check | Result | Duration | Exit |
|---|---|---:|---:|
| TestV2Runner_RejectBeforeHeadLookupFailure | PASS | <1s | 0 |
| TestV2Runner_RejectBeforeStatusFailure | PASS | <1s | 0 |
| TestV2Runner_RejectBeforeWorktreeListFailure | PASS | <1s | 0 |
| TestV2Runner_RejectAfterHeadLookupFailure | PASS | <1s | 0 |
| TestV2Runner_RejectAfterStatusFailure | PASS | <1s | 0 |
| TestV2Runner_RejectAfterWorktreeListFailure | PASS | <1s | 0 |

(Plus the existing TestV2MacCanaryFullRunnerDescendantProof and
CLI subprocess TestClosureCLIV2MacCanaryDogfood continue to pass.)

## Artifacts

None committed. The installed-style dogfood evidence (manifest,
stdout, stderr, evidence directory) lives in
`/tmp/leamas-mac-canary-dogfood-r1` and is intentionally NOT
committed to the Leamas repository. The Mac canary handoff
treats the ClineMM repository similarly: evidence and manifest
paths live outside `<clinemm>`.

## Excluded checks

None.

## Patch hygiene

- Git diff check: PASS
- Diagnostics: 0
- Tracked full digest policy: PASS
- Closure-policy diagnostics: 0

## Runner identity

- Leamas version: `0.1.0+dev`
- Binary SHA-256: `dfbddcc8c20c353b6b9fbb3d03eb3b8392a817f794c14f485d988e1059352db6`
- VCS revision: `25010d160c6b04edc24ec4602af951541ef1ffa8`
- VCS modified: `false`

## Lifecycle transition

Verification state: VERIFIED

## Final report fields

```text
ACT_ID=ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1
STATUS=PASS

BASE_COMMIT=3fb13db2f323cd86d96b49d3c1375b2a3a8370f9
BASE_TREE=7d76e1aefb9a3e7d34a4f7c1c2d1bf2c9d4ac83f
FINAL_COMMIT=25010d160c6b04edc24ec4602af951541ef1ffa8
FINAL_TREE=5dcd0864ad5b81c22ab2ac924d7f394ec1e2bb9a
CURRENT_HEAD=25010d160c6b04edc24ec4602af951541ef1ffa8
WORKTREE_STATUS=clean

RUNNER_IMPLEMENTATION_COMMIT=25010d160c6b04edc24ec4602af951541ef1ffa8
ACT5_CLOSE_ARTIFACT_COMMIT=d0780d5ab0854f81317bd62ee7d4d40aec460c04
ACT5_DOGFOOD_BINARY_COMMIT=20a5c6387655b7bff0236ea2becfed595c949ee0
R1_DOGFOOD_BINARY_COMMIT=25010d160c6b04edc24ec4602af951541ef1ffa8
DOGFOOD_BINARY_MATCHES_CURRENT_HEAD=true

BUILD_SOURCE_COMMIT=25010d160c6b04edc24ec4602af951541ef1ffa8
BUILD_SOURCE_TREE=5dcd0864ad5b81c22ab2ac924d7f394ec1e2bb9a
BUILD_SOURCE_STATUS=empty
DIRTY_STAMP_PROOF=BUILT_VCS_MODIFIED=false (verified via go version -m)

CALLER_STATE_FAIL_CLOSED_MATRIX=
  before HEAD failure   -> V2CodeCallerStateUnavailable
  before status failure -> V2CodeCallerStateUnavailable
  before worktree-list failure -> V2CodeWorktreeInventoryUnavailable + V2CodeCallerStateUnavailable
  after HEAD failure    -> V2CodeCallerStateUnavailable
  after status failure  -> V2CodeCallerStateUnavailable
  after worktree-list failure -> V2CodeWorktreeInventoryUnavailable + V2CodeCallerStateUnavailable

WORKTREE_INVENTORY_FAIL_CLOSED_MATRIX=
  before worktree-list failure -> snapshot.Available=false, runner refuses to execute
  after worktree-list failure  -> snapshot.Available=false, runner refuses to claim success

REAL_GIT_INVARIANT_TESTS=TestV2Lifecycle_CallerStateUnchangedOnSuccess, TestV2Lifecycle_WorktreeRegistrationNoLeak, TestV2Lifecycle_CleanupSurvivesCancellation, TestV2Lifecycle_SnapshotCallerStateIsDeterministic (all use RealGit{} instead of nil)

BOUNDED_SUBPROCESS_RESULT=cmd/leamas/bounded_subprocess_v2_test.go provides boundedSubprocessV2 with finite timeout (default 5m), bounded stdout/stderr (default 1 MiB), WaitDelay cleanup, and exit-code extraction

DOGFOOD_EXIT=0
DOGFOOD_BINARY_SHA256=dfbddcc8c20c353b6b9fbb3d03eb3b8392a817f794c14f485d988e1059352db6
DOGFOOD_VCS_REVISION=25010d160c6b04edc24ec4602af951541ef1ffa8
DOGFOOD_VCS_MODIFIED=false
DOGFOOD_CALLER_STATUS_BEFORE=
DOGFOOD_CALLER_STATUS_AFTER=
DOGFOOD_WORKTREES_BEFORE=1 (main only)
DOGFOOD_WORKTREES_AFTER=1 (main only)

LOCAL_GATES=gofmt OK, go vet ./... OK, go test -count=1 ./internal/factory/closure/... OK, go test -count=1 ./cmd/leamas/... OK, exec-gate OK, static build OK
PRE_EXISTING_GATE_FINDINGS=llm-friendly long_line complaints in CORRECTION01/02/03 docs (unrelated to this ACT); forbidden-patterns pre-existing platform-specific build_ignored files (unrelated to this ACT)
UNRESOLVED_BLOCKERS=v2 closure-commit verifier (sole permitted unresolved item per ACT 5 acceptance; deferred to ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-CLOSURE-VERIFIER01)

MAC_HANDOFF=
  Non-mutating inspection (Linux-side proof; Mac-side path is the
  same):
    git -C <clinemm> cat-file -e 56fd526e1923f2546fa0aeb53a0dc6e7501e5061^{commit}
    git -C <clinemm> cat-file -e 01822bf5c8b99e5a4b89a6761a713ec3603754b0^{commit}
    git -C <clinemm> merge-base --is-ancestor 56fd526e1923f2546fa0aeb53a0dc6e7501e5061 01822bf5c8b99e5a4b89a6761a713ec3603754b0
    git -C <clinemm> ls-tree -r --name-only 01822bf5c8b99e5a4b89a6761a713ec3603754b0
    git -C <clinemm> show 01822bf5c8b99e5a4b89a6761a713ec3603754b0:"$P"

  Mac run command (uses the exact-final-tip binary built from
  25010d1):
    /home/chistyakov/Projects/leamas/bin/leamas factory close run-v2-authority \
        --protocol-version 2 \
        --plan-contract-version 1 \
        --repository <clinemm> \
        --subject 56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
        --freeze 01822bf5c8b99e5a4b89a6761a713ec3603754b0 \
        --plan-path <P> \
        --evidence-directory <outside-clinemm-evidence> \
        --manifest-output <outside-clinemm-manifest.json>

  Linux-side equivalent dogfood ran successfully with
  DOGFOOD_EXIT=0 against a hermetic S < F < D repository.
```

## Closure

ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1
closes four narrow defects from ACT-5:

1. **exact-current-tip dogfood**: the dogfood binary's
   vcs.revision now equals the current HEAD
   (`25010d160c6b04edc24ec4602af951541ef1ffa8`).

2. **immutable / clean dogfood build source**: the binary is
   built from a clean source tree with `Dirty=false` proved by
   `go version -m` (BUILD_SOURCE_STATUS=empty).

3. **fail-closed caller-state snapshots**: `snapshotCallerState`
   and `snapshotWorktreeRegistrations` now return result-bearing
   snapshots with an `Available` bool and a typed
   `V2Diagnostics` slice. The runner refuses to execute when
   the BEFORE snapshot is unavailable, and refuses to claim
   clean success when the AFTER snapshot is unavailable. Six new
   tests cover the BEFORE/AFTER failure combinations for
   `rev-parse HEAD^{commit}`, `rev-parse HEAD^{tree}`,
   `status --porcelain=v2`, and `worktree list --porcelain`.

4. **bounded installed subprocess execution**:
   `cmd/leamas/bounded_subprocess_v2_test.go` provides
   `boundedSubprocessV2` with a finite timeout (default 5m),
   bounded stdout/stderr (default 1 MiB each), `WaitDelay`
   cleanup, and exit-code extraction via `*exec.ExitError`.
   The CLI dogfood test continues to exercise the public v2
   authority CLI end-to-end and the harness satisfies the
   R1 requirement that installed subprocess execution is bounded.

The runner-readiness sequence (ACTs 1..5 + R1) is now closed.
The sole remaining blocker — the v2 closure-commit verifier —
is intentionally deferred and documented above; it is the sole
permitted unresolved item per ACT 5 acceptance.
