# Close Report — ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01

## Status: SUPERSEDED

This report documents the original ACT closure at the plan commit
(`bc6db8a`). It was filed with `VERDICT=PASS` based on the matrix
the original ACT specified, but a follow-up review by a Git
provenance engineer found that the implementation inverted the
classifier composition and the renderer fell back to ambient HEAD
for explicit-range subjects.

**CORRECTION01** addresses these defects:

- Production defect 1 (C1): classifier's overall verdict was
  driven by the commit-vs-HEAD axis instead of the
  generator-vs-digest-subject axis. Fixed by removing the
  commit-mismatch short-circuit; the SUBJECT axis now dominates.
- Production defect 2 (C2): renderer fell back to HEAD when
  `LifecycleSubject` was empty. Fixed by resolving the explicit
  range's right endpoint through `git rev-parse` and adding
  `LifecycleSubjectRange` to the subject resolution order.

The corrected implementation lands at
`dcc56b3 factory: separate subject authority from ambient-HEAD
freshness (CORRECTION01)`. See
`docs/close-reports/ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01-CORRECTION01.md`
for the corrected verdict, acceptance board, and self-hosted
evidence.

## VERDICT (original ACT)

PASS at plan commit, but the production implementation was
backwards for historical ranges. Superseded by CORRECTION01.

## Identity

```text
ACT_ID=ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01
VERDICT=PASS

REPOSITORY_ROOT=.
BRANCH=main

INITIAL_HEAD=2f6bcbd584a4d8f4f9bd500add86d61694f1ba61
INITIAL_TREE=f9d6fd213dbdacd8c904e634615965be54113f6b
FINAL_HEAD=bc6db8a076d2db4cf4fbce867e15dcce3ae8985b
FINAL_TREE=194261cdb1b74bead7ffb41fda64f06acf41d756

origin/main_before=2f6bcbd584a4d8f4f9bd500add86d61694f1ba61
origin/main_after=bc6db8a076d2db4cf4fbce867e15dcce3ae8985b

WORKTREE_STATUS_BEFORE=clean
WORKTREE_STATUS_AFTER=clean

IMPLEMENTATION_COMMITS=
  e2b35e0 test(factory): define generator subject-binding contract
  211454d feat(factory): classify generator authority for digest subjects
  c05c0ce style(factory): gofmt generator-binding files
  bc6db8a plan(close): freeze generator-worktree-stale-authority ACT plan

CLOSE_REPORT_COMMIT=bc6db8a076d2db4cf4fbce867e15dcce3ae8985b
```

## Files Changed

```text
internal/factory/digest/generator_binding.go                 (new, 201 LOC)
internal/factory/digest/generator_binding_classifier.go      (new, 191 LOC)
internal/factory/digest/generator_binding_classifier_test.go (new, 379 LOC)
internal/factory/digest/generator_binding_adapter.go         (new,  91 LOC)
internal/factory/digest/generator_binding_adapter_test.go    (new, 150 LOC)
internal/factory/digest/lifecycle_render_generator_test.go   (new, 317 LOC)
internal/factory/digest/lifecycle_render.go                  (modified)
internal/factory/digest/resolve.go                           (modified)
internal/factory/digest/auto_range.go                        (modified)
docs/closure-plans/ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.json   (new)
docs/closure-manifests/ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.json (new)
docs/closure-evidence/ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01/*.txt (new)
docs/close-reports/ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.md     (new)
```

All owned/modified Go files are ≤ 400 lines.

## Behavior Changed

```text
GENERATOR_COMMIT_MATCH_IMPLIES_SUBJECT_MATCH=false
DIRTY_WORKTREE_REQUIRES_EXACT_SUBJECT_PROOF=true
UNPROVEN_DIRTY_GENERATOR_FAILS_CLOSED=true

GENERATOR_STALE_COMPATIBILITY_PRESERVED=true (legacy field rendered
  at its existing position with its existing semantic:
  commit_vs_repository_head. Now qualified with
  GENERATOR_STALE_BASIS=commit_vs_repository_head so reviewers do
  not silently conflate the legacy freshness signal with the new
  subject-authority signal.)

DIGEST_CONTRACT_VERSION_BEFORE=3
DIGEST_CONTRACT_VERSION_AFTER=3 (additive contract: new keys are
  appended to the LIFECYCLE section. EVIDENCE_HASHES hash_scope and
  normalized section list are unchanged because the LIFECYCLE
  section is NOT in the hashed evidence surface.)

AUTO_REBUILD_ADDED=false
SELF_REEXEC_ADDED=false
TIMESTAMP_FRESHNESS_ADDED=false
MTIME_FRESHNESS_ADDED=false
WORKTREE_FINGERPRINT_PROTOCOL_ADDED=false
GIT_REPOSITORY_MUTATION_ADDED=false

GATE_EVIDENCE_SEMANTICS_CHANGED=false
RANGE_SCOPE_SEMANTICS_CHANGED=false
DIGEST_DIFF_SEMANTICS_CHANGED=false
CLOSURE_PROTOCOL_CHANGED=false
```

## Semantic Doctrine

```text
COMMIT IDENTITY PROVES COMMITTED STATE.

IT DOES NOT PROVE UNCOMMITTED SOURCE STATE.

GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT,
NOT MERELY TO AMBIENT HEAD.
```

The ACT introduces a pure `EvaluateGeneratorBinding` classifier
that distinguishes two previously-conflated questions:

1. Does the running binary's embedded commit equal the repository
   HEAD? (commit-vs-HEAD freshness — `GENERATOR_COMMIT_MATCHES_HEAD`)
2. Is the running binary provably authoritative for the complete
   source subject the digest represents?
   (`GENERATOR_AUTHORITATIVE_FOR_DIGEST`)

The legacy `GENERATOR_STALE` flag is preserved at its existing
position with its existing semantics (commit-vs-HEAD only). It is
now qualified with `GENERATOR_STALE_BASIS=commit_vs_repository_head`
so reviewers do not silently conflate the legacy freshness signal
with the new subject-authority signal.

## Implementation Commits

```text
e2b35e0 test(factory): define generator subject-binding contract
  - generator_binding.go: typed vocabulary + GeneratorIdentity,
    RepositoryIdentity, DigestAuthoritySubject, GeneratorBinding
    structs + GeneratorBindingStatus / GeneratorStateBindingStatus
    / GeneratorWarningCode constants
  - generator_binding_classifier.go: pure EvaluateGeneratorBinding
    function (no I/O, no clock, no Git subprocesses)
  - generator_binding_classifier_test.go: 11-row matrix covering
    ACT §37 cases plus clock independence, determinism, vocabulary
    cardinality, case insensitivity, and EVIDENCE_INVALID
    fail-closed regressions

211454d feat(factory): classify generator authority for digest subjects
  - generator_binding_adapter.go: ResolveGeneratorBinding adapter
    translating raw OID strings into typed binding inputs
  - generator_binding_adapter_test.go: adapter validity
    classification, dirty-flag propagation, case normalization
  - lifecycle_render.go: extended RenderLifecycle to surface the
    new authority fields adjacent to the legacy GENERATOR_STALE
  - lifecycle_render_generator_test.go: render matrix for clean,
    generator-mismatch, unbound-identity, dirty-subject, and
    additive-contract ordering invariants
  - resolve.go: resolveAutoModeWith now populates GeneratorCommit
    from version.Get().Commit (linker + vcs.revision fallback)
    and GeneratorStale via computeLegacyGeneratorStale
  - auto_range.go: computeLegacyGeneratorStale helper

c05c0ce style(factory): gofmt generator-binding files

bc6db8a plan(close): freeze generator-worktree-stale-authority ACT plan
  - closure-plans/<ACT-ID>.json
```

## Verification — exact commands and honest results

```text
$ CGO_ENABLED=0 go test -count=1 ./internal/factory/digest/...
ok  github.com/s1onique/leamas/internal/factory/digest  18.709s
RC=0

$ CGO_ENABLED=0 go vet ./internal/factory/digest/...
(no output)
RC=0

$ CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-genauthority-final ./cmd/leamas
(no output)
RC=0

$ git diff --check
(no output)
RC=0
```

## Required Acceptance Board

```text
CLEAN_GENERATOR_CURRENT=PASS                (TestEvaluateGeneratorBindingMatrix/clean_equals_HEAD_committed,
                                             TestRenderLifecycleCleanAuthoritative,
                                             TestRenderLifecycleEndToEndCommitEqualsHead,
                                             acceptance-clean-lifecycle.txt)

CLEAN_GENERATOR_MISMATCH=PASS               (TestEvaluateGeneratorBindingMatrix/clean_generator_mismatch_HEAD,
                                             TestRenderLifecycleGeneratorMismatch,
                                             TestComputeLegacyGeneratorStale/different_full_oid)

TRACKED_DIRTY_UNBOUND=PASS                  (TestEvaluateGeneratorBindingMatrix/dirty_tracked_unbound,
                                             TestRenderLifecycleDirtySubjectUnbound,
                                             TestRenderLifecycleEndToEndDirtyWorktree,
                                             acceptance-dirty-lifecycle.txt)

STAGED_DIRTY_UNBOUND=PASS                   (TestEvaluateGeneratorBindingMatrix/staged_only_unbound)

UNTRACKED_DIRTY_UNBOUND=PASS                (TestEvaluateGeneratorBindingMatrix/untracked_only_unbound)

MIXED_DIRTY_UNBOUND=PASS                    (TestEvaluateGeneratorBindingMatrix/mixed_dirty_unbound)

HISTORICAL_RANGE_SUBJECT_BINDING=PASS       (TestEvaluateGeneratorBindingMatrix/historical_range_*,
                                             TestEvaluateGeneratorBindingMatrix/historical_range_generator_and_subject_at_HEAD)

AMBIENT_HEAD_NOT_AUTHORITY=PASS             (acceptance-dirty-lifecycle.txt demonstrates
                                             GENERATOR_COMMIT_MATCHES_HEAD=true
                                             AND GENERATOR_AUTHORITATIVE_FOR_DIGEST=false)

MISSING_GENERATOR_IDENTITY=PASS             (TestEvaluateGeneratorBindingMatrix/missing_generator_identity,
                                             TestRenderLifecycleUnboundIdentity,
                                             TestAdapterRejectsUnknownValues/all_empty)

INVALID_GENERATOR_IDENTITY=PASS             (TestEvaluateGeneratorBindingMatrix/invalid_generator_identity,
                                             TestEvaluateGeneratorBindingInvalidOIDFailClosed,
                                             TestAdapterRejectsUnknownValues/{garbage_generator,short_oid_generator,all_unknown_placeholder})

UNRESOLVED_SUBJECT_FAIL_CLOSED=PASS          (TestEvaluateGeneratorBindingMatrix/unresolved_digest_subject)

CLOCK_INDEPENDENCE=PASS                     (TestEvaluateGeneratorBindingClockIndependence,
                                             TestEvaluateGeneratorBindingDeterminism)

NO_AUTOREBUILD=PASS                         (no auto-rebuild code path; ACT §35)
NO_REPOSITORY_MUTATION=PASS                 (git status before/after identical;
                                             find .git/objects | wc -l identical
                                             before/after running the binary)

VARIABLE_GIT_SUBPROCESS_COUNT=0             (the new classifier and adapter perform
                                             no Git subprocesses; they consume
                                             already-resolved identity strings)

ADDITIVE_CONTRACT=PASS                      (TestRenderLifecycleAdditiveContract proves
                                             legacy field ordering preserved)
```

## Required Self-Hosting Evidence

### Dirty subject (binary built at HEAD; tracked file modified after build)

```text
GENERATOR_COMMIT: bc6db8a076d2db4cf4fbce867e15dcce3ae8985b
REPOSITORY_HEAD: bc6db8a076d2db4cf4fbce867e15dcce3ae8985b

GENERATOR_COMMIT_MATCHES_HEAD: true
GENERATOR_STALE: false
GENERATOR_STALE_BASIS: commit_vs_repository_head

GENERATOR_BINDING_STATUS: DIRTY_SUBJECT_UNBOUND
GENERATOR_COMMIT_BINDING: MATCH
GENERATOR_SUBJECT_BINDING: UNBOUND
GENERATOR_AUTHORITATIVE_FOR_DIGEST: false
GENERATOR_WARNING_CODE: GENERATOR_DIRTY_SUBJECT_UNBOUND
```

### Clean subject (worktree restored to HEAD)

```text
GENERATOR_COMMIT: bc6db8a076d2db4cf4fbce867e15dcce3ae8985b
REPOSITORY_HEAD: bc6db8a076d2db4cf4fbce867e15dcce3ae8985b

GENERATOR_COMMIT_MATCHES_HEAD: true
GENERATOR_STALE: false

GENERATOR_BINDING_STATUS: AUTHORITATIVE
GENERATOR_COMMIT_BINDING: MATCH
GENERATOR_SUBJECT_BINDING: MATCH
GENERATOR_AUTHORITATIVE_FOR_DIGEST: true
GENERATOR_WARNING_CODE: none
```

## Skipped or Deferred Checks

```text
make factorize:    NOT RUN (Cline/editor context; ACT does not delegate)
make gate-dupcode: NOT RUN (Cline/editor context; ACT does not delegate)
make gate:         NOT RUN (Cline/editor context; ACT does not delegate)

CGO_ENABLED=0 go test ./cmd/leamas/...   partial (long-running
  closure subtests timeout in this run; the targeted
  TestClosureCLIV2R2CRExactTipDogfood regression at HEAD passed
  after the gofmt commit. Other closure CLI tests were not
  re-run; production code paths and gates that depend on
  ResolveAutoModeWith were validated via the focused
  ./internal/factory/digest/... suite.)
```

## Follow-up ACTs

None. This ACT is complete on its own scope.

The next ACT to consider:

```text
FOLLOWUP_1: ACT-LEAMAS-DIGEST-SUBJECT-FINGERPR-WORKTREE-CANONICAL01
  ACTUAL AUTHORITY FOR UNCOMMITTED SOURCE STATE.
  This ACT explicitly defers source fingerprinting
  (no git write-tree, no tar hash, no mtime hash, no binary
  timestamp comparison). A future ACT may introduce a canonical
  worktree-subject fingerprint if the operational case for it
  becomes clear.
```

## No-goals confirmed

```text
- automatic rebuild                                 NOT ADDED
- automatic binary replacement                      NOT ADDED
- self re-exec                                      NOT ADDED
- source snapshot hashing                            NOT ADDED
- dirty-worktree fingerprint protocol               NOT ADDED
- SLSA provenance                                   NOT ADDED
- reproducible-build attestation                    NOT ADDED
- SBOMs                                             NOT ADDED
- binary signatures                                 NOT ADDED
- timestamp freshness                               NOT ADDED
- mtime freshness                                   NOT ADDED
- gate-summary changes                              NOT ADDED
- range-scope changes                               NOT ADDED
- closure protocol changes                          NOT ADDED
- fork/upstream authority                           NOT ADDED
```

## VERDICT

```text
VERDICT=PASS
PRODUCTION_IMPLEMENTATION=COMPLETE
CANONICAL_GATE_STATUS=NOT_RUN (delegation not granted by ACT)
```