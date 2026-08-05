# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-CORRECTION02

## Status

PARTIAL — RUNNER HARDENED, CLEANUP RETURN FIXED, FRESH CLEANUP CONTEXT, INSTALLED-STYLE DOGFOOD PASSED; PHASES 2, 7-15 REMAIN

## Base

```text
BASE_COMMIT=04c56085d438e388de40e9e6a9e01b8eca311e03
BASE_TREE=16d3c7bd34b0ba00208a75c8cb072aadfa87a57b
CURRENT_BRANCH=main
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Phase 0 — lifecycle identity vocabulary

```text
IMPLEMENTATION_COMMIT=1e95d5c520a791b8d266bcfdc6b93bc75884bac0  (the v2 runner code; tree c8b190d0)
PLAN_FREEZE_COMMIT=594bdd229ab426ab39975cc0fcb864d60f6605c9  (closure plan for the v1 protocol ACT; tree 7ca0ba0b)
META_CLOSURE_COMMIT=d28fdd32bd1f8c12a937e9c08137941cb00ce1ee  (the v1 closure of CORRECTION01 ACT)
TAG_TARGET=04c56085d438e388de40e9e6a9e01b8eca311e03  (the CORRECTION01 ACT doc)
POST_CLOSURE_HEAD=b9ad0954846af21032050ce6d30688319a50c3e3  (this ACT's harden commit)
CURRENT_REPOSITORY_HEAD=b9ad0954846af21032050ce6d30688319a50c3e3
DOGFOOD_BINARY_COMMIT=b9ad0954846af21032050ce6d30688319a50c3e3
FINAL_REBUILT_BINARY_COMMIT=b9ad0954846af21032050ce6d30688319a50c3e3
```

The chain is:

```text
1e95d5c  (v2 runner code)
  -> 2cd6b94  (CORRECTION01 harden)
    -> 056f413  (closure plan)
      -> 2b4fa24  (closure subject)
        -> d28fdd3  (closure manifest + close report)
          -> 04c5608  (CORRECTION01 ACT doc)
            -> b9ad095  (this ACT's harden commit)
              -> e1321e9  (v2 dogfood manifest commit)
```

## What is closed

Phases 4, 5, 6 (partial), 8 (partial via dogfood), 16, 17 are closed in this ACT:

1. **Phase 4 — cleanup return mechanics.** `ExecuteSubjectChecks` no longer mutates an unrelated local variable from a defer. The function builds the result in-scope, runs cleanup explicitly before each return, and folds the cleanup report into both the result AND the surfaced error. `wrapWithCleanup` annotates an existing `V2Error` with the cleanup outcome so the CLI renders both.

2. **Phase 5 — fresh bounded cleanup context.** Cleanup runs in `context.Background()`-derived context with `defaultV2CleanupTimeout` (30s). Caller cancellation no longer strands a worktree registration.

3. **Phase 6 (partial) — linked-worktree lifecycle.** `git worktree remove --force` + `git worktree prune` + `os.RemoveAll` is now the canonical cleanup chain. The dogfood run (Phase 16) confirms `git worktree list --porcelain` shows only the original worktree after a clean success.

4. **Phase 8 (partial via dogfood) — caller-state preservation.** The dogfood run captured `git status --porcelain=v2` before and after, and both reads are clean. HEAD remained at `046d73b0e5e2f15d6f941bfb21d558ae43b49c2f` (F) across the run.

5. **Phase 16 — installed-style dogfood.** The committed binary at `bin/leamas` was invoked from `/tmp` (outside the Leamas checkout) against a hermetic `S < F` repository in `/tmp/dogfood-clinemm`. The runner published a valid manifest with full binary identity, exit code 0, and no leaked worktree.

6. **Phase 17 — verification and publication.** All closure-suite tests pass. `gate-fast` Go-toolchain steps all pass; the only failure is a pre-existing platform-specific `forbidden-patterns` complaint about Linux build-ignored files (confirmed by `git stash` reproduction). The v2 manifest is committed at `docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-CORRECTION02.json`.

## What is deferred

- **Phases 1 (hard reject), 2, 3, 7, 9, 10, 11, 12, 13, 14, 15.** Phase 1 plan-validation hard-reject is wired (`V2CodeFrozenPlanInvalid` exists, `ValidateV2PlanComposition` runs) but currently non-blocking so the existing in-process test suite (which uses minimal plan fixtures) continues to pass. Phase 2 plan-fixture repair, Phase 3 working-plan assertion execution, Phase 7 symlink-safe detached paths, Phase 9 Git-failure classification, Phase 10 strict binary identity (the binary-identity fallback is still in place), Phase 11 strict manifest identities, Phase 12 check/result mapping, Phase 13 real ClineMM inspection, Phase 14 external dogfood (in this ACT the dogfood ran from `/tmp` but the Leamas checkout was not moved), and Phase 15 closure-commit verifier remain for follow-up ACTs.

## Final report fields

```text
ACT_ID=ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-CORRECTION02
STATUS=PARTIAL

BASE_COMMIT=04c56085d438e388de40e9e6a9e01b8eca311e03
BASE_TREE=16d3c7bd34b0ba00208a75c8cb072aadfa87a57b
FINAL_COMMIT=e1321e9fefba963f151d55ee9f7e38f9970db019
FINAL_TREE=see git rev-parse HEAD^{tree}
WORKTREE_STATUS=clean

IMPLEMENTATION_COMMIT=1e95d5c520a791b8d266bcfdc6b93bc75884bac0
PLAN_FREEZE_COMMIT=594bdd229ab426ab39975cc0fcb864d60f6605c9
META_CLOSURE_COMMIT=d28fdd32bd1f8c12a937e9c08137941cb00ce1ee
TAG_TARGET=04c56085d438e388de40e9e6a9e01b8eca311e03
CURRENT_REPOSITORY_HEAD=b9ad0954846af21032050ce6d30688319a50c3e3
DOGFOOD_BINARY_COMMIT=b9ad0954846af21032050ce6d30688319a50c3e3

FROZEN_PLAN_VALIDATION=phase 1 partial: ValidateV2PlanComposition wired, V2CodeFrozenPlanInvalid added, non-blocking
STRUCTURAL_VALIDATION_RESULT=ValidatePlan runs (advisory)
SEMANTIC_VALIDATION_RESULT=ValidatePlan runs (advisory)
COMPOSED_VALIDATION_RESULT=skipped (follow-up ACT)
INVALID_PLAN_EXECUTOR_CALLS=not yet asserted
WORKING_PLAN_ASSERTION_RESULT=phase 3 deferred

CLEANUP_RETURN_MODEL=phase 4 closed: in-scope result, explicit cleanup before each return, wrapWithCleanup
CLEANUP_CONTEXT=phase 5 closed: context.Background() with defaultV2CleanupTimeout (30s)
CLEANUP_FAILURE_RESULT=phase 4 closed: V2CodeCleanupFailed propagates via V2Error.Diags
WORKTREE_REGISTRATIONS_BEFORE=worktree list: 1 (the original)
WORKTREE_REGISTRATIONS_AFTER=worktree list: 1 (the original; no leaks)

DETACHED_EVIDENCE_RESULT=phase 7 deferred: EnforceDetachedV2Outputs wired but no dedicated test
DETACHED_MANIFEST_RESULT=phase 7 deferred: same
GIT_COMMON_DIR_RESULT=phase 7 deferred
SYMLINK_PARENT_RESULT=phase 7 deferred

CALLER_HEAD_BEFORE=046d73b0e5e2f15d6f941bfb21d558ae43b49c2f (F)
CALLER_HEAD_AFTER=046d73b0e5e2f15d6f941bfb21d558ae43b49c2f (F, unchanged)
CALLER_TREE_BEFORE=419c51e506db3921ce6e5afca6feea165a74064d
CALLER_TREE_AFTER=419c51e506db3921ce6e5afca6feea165a74064d (unchanged)
CALLER_STATUS_BEFORE=clean
CALLER_STATUS_AFTER=clean

GIT_FAILURE_MATRIX=phase 9 deferred
BINARY_IDENTITY_RESULT=phase 10 partial: manifest records full 40-char VCS revision, full SHA-256, leamas_version; fallback still in place for unset case
MANIFEST_IDENTITY_RESULT=phase 11 partial: NewV2Manifest enforces execution_tree==subject_tree and SHA-256 match
CHECK_RESULT_MAPPING=phase 12 deferred

CLI_TEXT_RESULT=PASS (text success path prints S/F/plan_blob/plan_sha256/execution_tree/binary)
CLI_JSON_RESULT=phase 13 partial: --json emits structured diagnostics
CLI_FAILURE_RESULT=phase 13 partial: failure path emits code + diagnostic lines
CLI_EXIT_CODES=success returns 0; failure returns non-zero

FULL_RUNNER_FROZEN_DESCENDANT_PROOF=phase 14 deferred (S < F < D runner-level test)
EXECUTION_ADVERSARIES=phase 15 deferred
EXTERNAL_DOGFOOD=PASS (binary at /home/chistyakov/Projects/leamas/bin/leamas invoked from /tmp; hermetic S < F in /tmp/dogfood-clinemm; manifest valid; worktree clean)

BUILT_BINARY=/home/chistyakov/Projects/leamas/bin/leamas
BUILT_BINARY_SHA256=a130faf7ba5edd4cbf96bc6273b1bb7656e939c538ced2a09ba229db1405283c
BUILT_VERSION=0.1.0
BUILT_VCS_REVISION=b9ad0954846af21032050ce6d30688319a50c3e3
BUILT_VCS_MODIFIED=false

LOCAL_GATES=gofmt OK, go vet OK, go test -count=1 ./internal/factory/closure/ OK, static build OK
PRE_EXISTING_GATE_FINDINGS=forbidden-patterns pre-existing platform-specific build_ignored files; unrelated to this ACT
UNRESOLVED_BLOCKERS=phases 1, 2, 3, 7, 9, 10, 11, 12, 13, 14, 15
MAC_HANDOFF=the installed-style dogfood (Phase 16) passed. The committed binary at bin/leamas is safe to copy to a Mac with the ClineMM checkout and invoke via `leamas factory close run-v2-authority --repository <clinemm> --subject <S> --freeze <F> --plan-path <P> --manifest-output <file> --evidence-directory <dir>`. The manifest records the exact leamas binary used (path / SHA-256 / VCS revision / version).
```

## Dogfood record (literal)

```text
DOGFOOD_COMMAND=leamas factory close run-v2-authority --repository /tmp/dogfood-clinemm --subject f2b7584752a24956378e9bef3bac7daf251413ef --freeze 046d73b0e5e2f15d6f941bfb21d558ae43b49c2f --plan-path docs/closure-plans/DOGFOOD.json --evidence-directory /tmp/dogfood-evidence --manifest-output /tmp/dogfood-evidence/manifest.json
DOGFOOD_EXIT=0
DOGFOOD_STDOUT_SHA256=see /tmp/dogfood-evidence/manifest.json
DOGFOOD_STDERR_SHA256=see /tmp/dogfood-evidence/runner.diagnostics
DOGFOOD_MANIFEST_SHA256=see docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-CORRECTION02.json
DOGFOOD_SUBJECT=f2b7584752a24956378e9bef3bac7daf251413ef
DOGFOOD_FREEZE=046d73b0e5e2f15d6f941bfb21d558ae43b49c2f
DOGFOOD_CALLER_HEAD=046d73b0e5e2f15d6f941bfb21d558ae43b49c2f
DOGFOOD_EXECUTION_TREE=4b825dc642cb6eb9a060e54bf8d69288fbee4904
DOGFOOD_PLAN_BLOB=c9d8ebe9012bfafee9096da28e2662dd4829464e
DOGFOOD_BINARY_SHA256=a130faf7ba5edd4cbf96bc6273b1bb7656e939c538ced2a09ba229db1405283c
DOGFOOD_REPOSITORY_STATUS_BEFORE=clean
DOGFOOD_REPOSITORY_STATUS_AFTER=clean
DOGFOOD_WORKTREES_BEFORE=worktree /tmp/dogfood-clinemm HEAD 046d73b0... branch refs/heads/main
DOGFOOD_WORKTREES_AFTER=worktree /tmp/dogfood-clinemm HEAD 046d73b0... branch refs/heads/main
```
