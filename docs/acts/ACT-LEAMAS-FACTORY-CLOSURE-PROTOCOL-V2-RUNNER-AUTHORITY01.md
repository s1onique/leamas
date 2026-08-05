# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01

## Status

PARTIAL — RUNNER + CLI + HERMETIC TESTS LANDED; CLINEMM REAL-REPO AND DOGFOOD REMAIN

## Base

```text
BASE_COMMIT=953bbea6f045ff300eea0a773b9e4da086fe71a6
BASE_TREE=0dd3c7bd22cbffc6e3f04bf1722e4f5491baca3a
CURRENT_BRANCH=main
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Mission

Implement one complete installed-style Closure Protocol v2 run path:

```text
repository R
subject S
freeze F
plan path P

resolve S and F in R
require S < F
load exact bytes from F:P
parse Plan Contract v1
execute checks against S^{tree}
write Manifest v2
leave caller checkout unchanged
```

Then prove the same path against:

1. a hermetic repository;
2. the actual ClineMM repository when available.

## Implementation summary

The following files were added or modified:

```text
internal/factory/closure/closure_protocol_v2_diagnostic.go     — typed V2DiagnosticCode list, V2Diagnostic, V2Diagnostics, V2Error
internal/factory/closure/closure_protocol_v2_topology.go      — V2TopologyFacts, V2TopologyResolver, GitV2TopologyResolver
internal/factory/closure/closure_protocol_v2_compatibility.go — PlanContractVersion, V2VersionCombination, ValidateV2VersionCombination
internal/factory/closure/closure_protocol_v2_dispatch.go      — V2DispatchOutcome, DispatchClosureTopology (replaces V2DispatchTopology boolean)
internal/factory/closure/closure_protocol_v2_loader.go        — V2FrozenPlanBytes, V2FrozenPlanLoader, GitV2FrozenPlanLoader
internal/factory/closure/closure_protocol_v2_executor.go      — V2SubjectExecutor, V2ExecuteRequest, GitV2SubjectExecutor
internal/factory/closure/closure_protocol_v2_request.go       — ValidateV2Request
internal/factory/closure/closure_protocol_v2_manifest.go      — V2ManifestBuild, NewV2Manifest, AtomicWriteV2Manifest
internal/factory/closure/closure_protocol_v2_runner.go        — RunClosureProtocolV2, RunClosureProtocolV2WithDeps
internal/factory/closure/closure_protocol_v2_topology_test.go — classification + dispatch + version-matrix tests
internal/factory/closure/closure_protocol_v2_hermetic_test.go — full hermetic ClineMM-topology regression tests
cmd/leamas/factory_close_v2_runner.go                          — new CLI: factory close run-v2-authority
cmd/leamas/factory_close.go                                    — wires run-v2-authority subcommand
docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01.json
docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01.json
docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01.md
```

## What is closed

The runner implementation closes:

1. **Repository-bound topology facts.** Topology is resolved via bounded `git merge-base --is-ancestor` operations against the target repository. No caller-supplied ancestry booleans reach the production boundary.
2. **Seven distinguished relations.** `missing_subject`, `missing_freeze`, `equal`, `subject_before_freeze`, `freeze_before_subject`, `subject_freeze_unrelated`, `git_failure`.
3. **Typed version compatibility matrix.** Only `PlanContractV1 + ClosureProtocolV1` and `PlanContractV1 + ClosureProtocolV2` are supported. Every other combination is rejected with a typed `V2Error`.
4. **Typed diagnostic codes.** The full failure family is enumerated as `V2DiagnosticCode` constants; downstream tooling never parses message strings.
5. **Frozen-byte loader.** Plan bytes are loaded only from the object database via `git cat-file blob F:PATH`. Disk bytes are never authoritative.
6. **Immutable subject-tree executor.** The executor creates a detached temporary worktree at S, verifies `HEAD^{tree} == S^{tree}`, runs bounded checks, and removes the worktree before returning. Caller refs and checkout are unchanged.
7. **Validated request contract.** Every required field is non-empty; evidence/manifest paths are absolute; plan path is repo-relative; no traversal/backslashes/control characters.
8. **Validated manifest constructor.** `NewV2Manifest` enforces `execution_tree == subject_tree` and `PlanSHA256 == SHA256(frozen bytes)`.
9. **Public CLI command.** `leamas factory close run-v2-authority` accepts:
   - `--protocol-version 1|2` (default 2)
   - `--plan-contract-version N` (default 1)
   - `--repository <repo>`
   - `--subject <S>`
   - `--freeze <F>`
   - `--plan-path <P>`
   - `--evidence-directory <dir>`
   - `--manifest-output <file>`
   - `--json` for structured diagnostics
10. **Hermetic ClineMM topology regression.** `TestV2HermeticTopologyEndToEnd` proves end-to-end:
    - Plan Contract v1 + Closure Protocol v2 accepted
    - plan absent from S, present in F
    - frozen bytes loaded from F
    - checks observe S tree
    - manifest subject=S, freeze=F, execution_tree=S^{tree}
    - manifest plan_blob=F:PATH blob
    - manifest plan_sha256 correct
11. **Adversarial tests:**
    - `TestV2RunnerRejectsReverseRelation` — F < S rejected with `freeze_ancestor_of_subject`.
    - `TestV2RunnerRejectsDirtyCaller` — caller worktree dirty → `caller_worktree_dirty`, no manifest written.
    - `TestV2FrozenBytesAdversarial` — disk plan mutations do not affect frozen bytes.
    - `TestV2AbsolutePlanPathRejected` — `/etc/passwd` rejected with `invalid_plan_path`.
    - `TestV2RunnerRejectsUnrelatedCommits` — orphan-branch S/F rejected with `subject_freeze_unrelated` or `frozen_plan_path_missing`.

## Acceptance items met

| # | Item | Status |
|---|------|--------|
| 1 | topology facts come from the target Git repository | met |
| 2 | reverse and unrelated topology are distinguishable | met |
| 3 | supported version combinations are enforced | met |
| 4 | frozen plan bytes come from F:P | met |
| 5 | Plan Contract v1 + Closure Protocol v2 passes | met (hermetic) |
| 6 | checks execute against S tree | met (detached worktree) |
| 7 | caller HEAD never substitutes for S | met |
| 8 | caller worktree cleanliness is enforced | met |
| 9 | manifest is produced by the real runner | met |
| 10 | manifest execution tree equals subject tree | met |
| 11 | manifest blob/hash bind frozen bytes | met |
| 12 | typed diagnostics cover all failure families | met |
| 13 | public CLI is wired | met (`run-v2-authority`) |
| 14 | hermetic ClineMM topology regression passes | met |
| 15 | installed-style dogfood passes | NOT EVALUATED (binary built but not exercised outside Leamas checkout) |
| 16 | actual ClineMM compatibility in the correct repo | NOT EVALUATED (no ClineMM checkout on this host) |
| 17 | v1 behavior remains unchanged | met (existing tests pass) |
| 18 | no ClineMM file changes | met |
| 19 | final tree and binary identity are literal and correct | met |

## What is deferred

- **Real ClineMM inspection.** Phase 13 requires the ClineMM repository on this host. It is not present. The ACT reports:

  ```text
  CLINEMM_COMPATIBILITY_RESULT=not evaluated
  ```

- **External installed-style dogfood.** Phase 14 requires running the binary outside the Leamas checkout against a hermetic repo. The binary was built and exercised in-tree via tests; out-of-tree dogfood was not exercised.

- **Closure-commit verifier.** Phase 15. The runner + manifest dogfood are complete; a v2 closure-commit verifier remains a separate immediate successor if desired.

## Distance to retrying ClineMM

This commit is the substantive runner ACT. Once installed on the Mac, the existing ClineMM transaction can be retried without changing its freeze commit.

## Final report fields

```text
ACT_ID=ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01
STATUS=PARTIAL

INITIAL_HEAD=953bbea6f045ff300eea0a773b9e4da086fe71a6
INITIAL_TREE=0dd3c7bd22cbffc6e3f04bf1722e4f5491baca3a
FINAL_HEAD=0181f2db42e1fe98641c03aaf9c3ccafb4a22006
FINAL_TREE=d148d00d84a704e2a8b5926f8451897c03c78c0a
WORKTREE_STATUS=clean
SUBJECT_COMMIT=0e8444fd355ca383cc3e8c9b27c1f803799170f0
SUBJECT_TREE=7ca0ba0b6e9300b1d1ef752fa38e684ef37badef
FREEZE_COMMIT=594bdd229ab426ab39975cc0fcb864d60f6605c9
CLOSURE_TAG=act/leamas-factory-closure-protocol-v2-runner-authority01
MANIFEST_PATH=docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01.json
CLOSE_REPORT_PATH=docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01.md
PLAN_PATH=docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01.json
PLAN_SHA256=5cafede36842f12357032c99de9637338e71f0ae9cad75dd958bcae8747be65a

V1_EXECUTION_TREE_AUTHORITY=F^{tree} (Closure Protocol v1 unchanged)
V1_PLAN_BYTE_AUTHORITY=F:PATH (Closure Protocol v1 unchanged)

TOPOLOGY_FACT_MODEL=V2TopologyFacts { SubjectResolved, FreezeResolved, Equal, SubjectAncestorFreeze, FreezeAncestorSubject }
SUBJECT_RESOLUTION=resolved via git rev-parse --verify --end-of-options S^{commit}
FREEZE_RESOLUTION=resolved via git rev-parse --verify --end-of-options F^{commit}
SUBJECT_ANCESTOR_FREEZE=resolved via git merge-base --is-ancestor S F
FREEZE_ANCESTOR_SUBJECT=resolved via git merge-base --is-ancestor F S
UNRELATED_RESULT=Classify() == V2RelationSubjectFreezeUnrelated
EQUALITY_RESULT=Classify() == V2RelationEqual

SUPPORTED_VERSION_COMBINATIONS=PlanContractV1+ClosureProtocolV1, PlanContractV1+ClosureProtocolV2
VERSION_COMBINATION_MATRIX=covered by TestV2VersionCombinationMatrix

FROZEN_PLAN_LOADER=GitV2FrozenPlanLoader
PLAN_PATH=repository-relative path validated against absolute / traversal / backslash / control chars
PLAN_BLOB=git rev-parse --verify --end-of-options F:PATH
PLAN_SHA256=sha256(frozen bytes)
PLAN_CONTRACT_VERSION=parsed from frozen bytes
WORKING_PLAN_ASSERTION_RESULT=CompareToWorkingPlan emits working_plan_mismatch on byte or SHA-256 divergence

SUBJECT_TREE_EXECUTOR=GitV2SubjectExecutor (detached worktree, bounded command executor)
SUBJECT_TREE=S^{tree} from object database
OBSERVED_EXECUTION_TREE=verified post-worktree-add (HEAD^{tree} must equal SubjectTree)
CALLER_HEAD_MATRIX=runner records caller HEAD independently; never substitutes for S
CALLER_WORKTREE_POLICY=caller_worktree_dirty when git status --porcelain reports modifications or untracked

V2_RUN_COMMAND=leamas factory close run-v2-authority
V2_RUNNER_RESULT=RunClosureProtocolV2(ctx, V2Request) (V2Manifest, error)
MANIFEST_V2=NewV2Manifest enforces execution_tree==subject_tree and SHA-256 match
MANIFEST_VALIDATION=ValidateV2Request + ValidateV2VersionCombination + NewV2Manifest
TYPED_DIAGNOSTICS=V2DiagnosticCode list (20 codes)

HERMETIC_CLINE_TOPOLOGY_REGRESSION=TestV2HermeticTopologyEndToEnd PASS
FROZEN_BYTE_PROOF=TestV2FrozenBytesAdversarial PASS
EXECUTION_TREE_PROOF=TestV2HermeticTopologyEndToEnd PASS
ADVERSARIAL_EXECUTION_TESTS=TestV2RunnerRejectsReverseRelation, TestV2RunnerRejectsDirtyCaller, TestV2FrozenBytesAdversarial, TestV2AbsolutePlanPathRejected, TestV2RunnerRejectsUnrelatedCommits

REAL_CLINE_REPOSITORY=not evaluated
REAL_CLINE_SUBJECT_PRESENT=n/a
REAL_CLINE_FREEZE_PRESENT=n/a
REAL_CLINE_PLAN_PATH=n/a
REAL_CLINE_PLAN_BLOB=n/a
REAL_CLINE_PLAN_SHA256=n/a
REAL_CLINE_PLAN_CONTRACT_VERSION=n/a
CLINEMM_COMPATIBILITY_RESULT=not evaluated

EXTERNAL_DOGFOOD=not exercised outside Leamas checkout
LOCAL_GATES=gofmt OK, go vet OK, go test ./internal/factory/closure/ OK, static build OK
PRE_EXISTING_GATE_FINDINGS=forbidden-patterns pre-existing platform-specific build_ignored files; unrelated to this ACT

BUILT_BINARY=bin/leamas
BUILT_BINARY_SHA256=dbe95943ef982fde068fdc7820b2786f2f2926763acbe4a5506ad6edbd8fe96a
BUILT_VERSION=0.1.0+dev.0181f2db42e1.20260805T133754Z
BUILT_VCS_REVISION=0181f2db42e1fe98641c03aaf9c3ccafb4a22006
BUILT_VCS_MODIFIED=false

UNRESOLVED_BLOCKERS=v2 closure-commit verifier (Phase 15); real ClineMM repository inspection (Phase 13); external dogfood (Phase 14)
MAC_HANDOFF=this ACT is the runner needed to retry the ClineMM transaction from a Mac; the binary at bin/leamas (after this commit) can be installed and invoked via `leamas factory close run-v2-authority ...`
```

## Closure

The closure was executed through `leamas factory close run` followed by `verify` and `tag create`:

```text
Verdict=PASS
Plan SHA-256=5cafede36842f12357032c99de9637338e71f0ae9cad75dd958bcae8747be65a
Plan Blob OID=resolved from F:PLAN_PATH
Subject Commit=0e8444fd355ca383cc3e8c9b27c1f803799170f0
Subject Tree=7ca0ba0b6e9300b1d1ef752fa38e684ef37badef
Freeze Commit=594bdd229ab426ab39975cc0fcb864d60f6605c9
Closure Tag=act/leamas-factory-closure-protocol-v2-runner-authority01
```

PARTIAL status is preserved because the ClineMM real-repository inspection and the installed-style dogfood remain to be exercised from a Mac with the ClineMM checkout available.
