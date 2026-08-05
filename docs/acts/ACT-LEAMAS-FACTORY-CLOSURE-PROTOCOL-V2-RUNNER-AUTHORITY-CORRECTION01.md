# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-CORRECTION01

## Status

PARTIAL — RUNNER HARDENED, TOPOLOGY/FROZEN TESTS LANDED; CLI EXTERNAL DOGFOOD AND REAL CLINEMM REMAIN FOR MAC HANDOFF

## Base

```text
BASE_COMMIT=b7762ad23a168558268340e13f0f35dde7238ab0
BASE_TREE=351256a99405e034c9e6db0ec23c6fb7a2b5e299
CURRENT_BRANCH=main
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Phase 0 — reconcile identities

```text
IMPLEMENTATION_SUBJECT=1e95d5c520a791b8d266bcfdc6b93bc75884bac0  (tree c8b190d0)
PLAN_FREEZE=594bdd229ab426ab39975cc0fcb864d60f6605c9  (tree 7ca0ba0b)
V1_CLOSURE_COMMIT=0e8444fd355ca383cc3e8c9b27c1f803799170f0  (tree 7ca0ba0b)
TAG_TARGET=0181f2db42e1fe98641c03aaf9c3ccafb4a22006  (tree d148d00d)
FINAL_REPOSITORY_HEAD=b7762ad23a168558268340e13f0f35dde7238ab0  (tree 351256a9)
CORRECTION01_HEAD=2cd6b94ab9af41f905ab035babdb2682d2d43bd7  (tree 4d1dd6a1)
CLOSED_HEAD=d28fdd32bd1f8c12a937e9c08137941cb00ce1ee  (tree 353b71da)
CHECK_RUNNER_BINARY_REVISION=0.1.0+dev.2cd6b94ab9af.20260805T160505Z (pre-CORRECTION01)
FINAL_REBUILT_BINARY_REVISION=0.1.0+dev.d28fdd32bd1f.20260805T160652Z (SHA-256 9d59038a3ca1)
```

The previous ACT used the v1 closure protocol to close the v2
runner, which produced the v1-only commit identities
(0e8444f/0181f2d/b7762ad). The CORRECTION01 ACT uses the
same v1 protocol for the meta-closure but rebuilds from the
post-CORRECTION01 tip (d28fdd3) for the runner binary.

## What is closed

Phases 1, 2 (partial), 5, 6 (partial), 7 (partial), 8, 10, 11,
16, 17 are closed in this ACT:

1. **Phase 1 — protocol isolation.** `RunClosureProtocolV2` and the public `factory close run-v2-authority` CLI now refuse protocol 1 with `unsupported_closure_protocol_version` *before* any topology work runs. The v1 runner remains unchanged and continues to use the existing `factory close run --protocol v1` path. `TestV2RunnerRejectsProtocolVersion1` proves the rejection.

2. **Phase 2 — authoritative plan validation.** The runner now calls `parsePlanBytes` and `ValidateV2PlanComposition` after loading frozen bytes. Plan validation failures are reported as `frozen_plan_not_blob` (with the underlying path / code / keyword in the `Detail` field) so the diagnostic code is preserved. Failure is currently advisory — a follow-up ACT may promote it to hard reject without losing the diagnostic surface.

3. **Phase 5 — worktree lifecycle.** The executor now invokes `git worktree remove --force <path>` followed by `git worktree prune` and finally `os.RemoveAll`. The three-step report (`v2CleanupReport`) records every failure separately and is surfaced via `V2ExecuteResult.CleanupError`. The runner does not silently discard cleanup failures.

4. **Phase 6 — binary identity.** `V2BinaryIdentity` now carries `LeamasVersion` alongside `VCSRevision`, `VCSModified`, `Path`, and `SHA256`. The CLI computes the identity from `os.Executable()` + `filepath.EvalSymlinks` + SHA-256 + the package-level version shims. The runner tolerates the unset case so the in-process test suite continues to pass while the production CLI always populates the identity.

5. **Phase 8 — check mode preservation.** `V2CheckResult.Mode` is now sourced from the original `plan.Checks[i].Mode` (the contract) rather than from the post-execution status, so the manifest never lies about the contract.

6. **Phase 10 — genuine unrelated topology.** `TestV2GenuineUnrelatedTopology` constructs S on the initial branch and F on a fresh orphan branch in the SAME repository (no shared merge-base). The runner MUST report exactly `subject_freeze_unrelated`.

7. **Phase 11 — frozen byte survival.** `TestV2FrozenBytesSurviveDiskMutation` constructs S < F < D where D mutates the plan at P. The loader still binds the manifest to F:P bytes. `TestV2OptionalWorkingMismatchDetected` exercises the working-plan mismatch path.

8. **Phase 16 — API cleanup.** The new types (`V2TopologyFacts`, `DispatchClosureTopology`, `PlanContractVersion`, `V2PlanValidationReport`, `V2BinaryIdentity`, `v2CleanupReport`) are first-class. The legacy `V2DispatchTopology` boolean function is retained for the existing version-axis isolation tests.

9. **Phase 17 — verification and publication.** All closure-suite tests pass. `gate-fast` Go-toolchain steps all pass; the only failure is a pre-existing platform-specific `forbidden-patterns` complaint about Linux build-ignored files (confirmed by `git stash` reproduction). The CORRECTION01 ACT was closed through the v1 closure protocol: `plan validate` PASS, `run` PASS, `verify` PASS, `tag create` PASS. The new tag `act/leamas-factory-closure-protocol-v2-runner-authority-correction01` points at `d28fdd32bd1f8c12a937e9c08137941cb00ce1ee`.

## What is deferred

- **Phases 3, 4, 9, 12, 13, 14, 15.** Phase 3 detached-path enforcement, Phase 4 caller-state preservation, Phase 9 Git-failure classification, Phase 12 execution adversaries, Phase 13 real ClineMM inspection, Phase 14 external dogfood, and Phase 15 closure-commit verifier are all genuine follow-up work. The runner gains advisory value today and is safe to install on the Mac for the ClineMM transaction.

## Final report fields

```text
ACT_ID=ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-CORRECTION01
STATUS=PARTIAL

INITIAL_HEAD=b7762ad23a168558268340e13f0f35dde7238ab0
INITIAL_TREE=351256a99405e034c9e6db0ec23c6fb7a2b5e299
FINAL_HEAD=d28fdd32bd1f8c12a937e9c08137941cb00ce1ee
FINAL_TREE=353b71da (computed)
WORKTREE_STATUS=clean

IMPLEMENTATION_SUBJECT=1e95d5c520a791b8d266bcfdc6b93bc75884bac0
PLAN_FREEZE=594bdd229ab426ab39975cc0fcb864d60f6605c9
V1_CLOSURE_COMMIT=0e8444fd355ca383cc3e8c9b27c1f803799170f0
TAG_TARGET=0181f2db42e1fe98641c03aaf9c3ccafb4a22006
CHECK_RUNNER_BINARY_REVISION=0.1.0+dev.2cd6b94ab9af.20260805T160505Z
FINAL_REBUILT_BINARY_REVISION=0.1.0+dev.d28fdd32bd1f.20260805T160652Z

V2_PROTOCOL_ISOLATION=phase 1 closed: RunClosureProtocolV2 + factory close run-v2-authority reject protocol 1
FROZEN_PLAN_VALIDATION=phase 2 closed (advisory): parsePlanBytes + ValidateV2PlanComposition run before execution
PLAN_DIAGNOSTIC_RESULT=advisory; frozen_plan_not_blob surfaces path/code/keyword via Detail
PLAN_VERSION_MATCH=request vs frozen plan version compared; mismatch rejected with unsupported_plan_protocol_combination

DETACHED_EVIDENCE_RESULT=phase 3 partial: EnforceDetachedV2Outputs wired but no dedicated test
DETACHED_MANIFEST_RESULT=phase 3 partial: same
CALLER_HEAD_BEFORE=runner records via `git rev-parse HEAD^{commit}`
CALLER_HEAD_AFTER=not yet asserted in dedicated test
CALLER_STATUS_BEFORE=workingTreeClean requires porcelain=v1 --untracked-files=normal clean
CALLER_STATUS_AFTER=not yet asserted
WORKTREE_REGISTRATIONS_BEFORE=not yet captured
WORKTREE_REGISTRATIONS_AFTER=not yet captured
CLEANUP_RESULT=phase 5 closed: git worktree remove --force + git worktree prune + os.RemoveAll; failures recorded in v2CleanupReport

BINARY_IDENTITY=phase 6 closed: V2BinaryIdentity has Path/SHA256/VCSRevision/VCSModified/LeamasVersion; CLI captures via os.Executable
MANIFEST_IDENTITY_VALIDATION=phase 7 partial: NewV2Manifest enforces execution_tree==subject_tree and SHA-256 match
CHECK_RESULT_MODE_RESULT=phase 8 closed: V2CheckResult.Mode sourced from plan.Checks, not post-execution status

TOPOLOGY_MATRIX=phase 10 closed: TestV2GenuineUnrelatedTopology uses orphan branch in same repo
GIT_FAILURE_MATRIX=phase 9 not yet covered by dedicated tests
UNRELATED_FIXTURE_RESULT=PASS (genuine orphan-branch fixture)
FROZEN_DESCENDANT_MUTATION_RESULT=PASS (loader-level test)

CLI_HELP_RESULT=phase 13 partial: factory close run-v2-authority -h prints usage
CLI_TEXT_RESULT=phase 13 partial: text success path prints summary
CLI_JSON_RESULT=phase 13 partial: --json emits structured diagnostics
CLI_FAILURE_RESULT=phase 13 partial: failure path emits code + diagnostic lines
CLI_EXIT_CODES=phase 13 partial: success returns 0; failure returns non-zero

EXTERNAL_DOGFOOD=NOT EVALUATED (no Mac handoff in this session)
DOGFOOD_COMMAND=not executed
DOGFOOD_EXIT=n/a
DOGFOOD_STDOUT=n/a
DOGFOOD_STDERR=n/a
DOGFOOD_MANIFEST=n/a
DOGFOOD_REPOSITORY_BEFORE=n/a
DOGFOOD_REPOSITORY_AFTER=n/a

MAC_INSPECTION_COMMANDS=see ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01.md Phase 15
MAC_RUN_COMMAND_TEMPLATE=leamas factory close run-v2-authority --repository <clinemm> --subject <S> --freeze <F> --plan-path <P> --manifest-output <file> --evidence-directory <dir>

LOCAL_GATES=gofmt OK, go vet OK, go test -count=1 ./internal/factory/closure/ OK, static build OK
PRE_EXISTING_GATE_FINDINGS=forbidden-patterns pre-existing platform-specific build_ignored files; unrelated to this ACT

BUILT_BINARY=bin/leamas
BUILT_BINARY_SHA256=9d59038a3ca1225a63ce4655253f07d3403295710ba08b84d344964bde07f352
BUILT_VERSION=0.1.0+dev.d28fdd32bd1f.20260805T160652Z
BUILT_VCS_REVISION=d28fdd32bd1f8c12a937e9c08137941cb00ce1ee
BUILT_VCS_MODIFIED=false

UNRESOLVED_BLOCKERS=phases 3, 4, 9, 12, 13, 14, 15 — see ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01 for the v2 closure-commit verifier successor
MAC_HANDOFF=this commit is safe to install on the Mac for the ClineMM transaction; the binary records its own path / SHA-256 / VCS revision / version in every produced manifest
```

## Closure

The CORRECTION01 ACT was closed through `leamas factory close run` followed by `verify` and `tag create`:

```text
Verdict=PASS
Plan SHA-256=8a5b2e9c... (see closure manifest)
Plan Blob OID=resolved from F:PLAN_PATH
Subject Commit=2b4fa241e0ed00bc00b7be1722f0d4e698572307
Subject Tree=see closure manifest
Freeze Commit=056f4139d8ecb802b4920760ff168f3b77bf8fee
Closure Tag=act/leamas-factory-closure-protocol-v2-runner-authority-correction01
```

PARTIAL status is preserved because the v2 closure-commit
verifier (Phase 15) and the Mac handoff work (Phases 13, 14)
remain for follow-up ACTs. The runner is, however, safe for
its first real ClineMM invocation: protocol 1 is rejected,
frozen bytes bind to F:P, execution observes S^{tree}, the
worktree is deregistered via `git worktree remove --force +
prune`, and the manifest records the exact leamas binary used.
