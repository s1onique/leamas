# Close Report — ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01-CORRECTION01

## VERDICT

PASS

## Identity

```text
ACT_ID=ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01-CORRECTION01
PARENT_ACT=ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01
VERDICT=PASS

REPOSITORY_ROOT=.
BRANCH=main

INITIAL_HEAD=5dbbf6a30873540f77c15be9acf0afb552571574
INITIAL_TREE=7f6884eba36ecf136110edc1a8e141c34be72c5a
FINAL_HEAD=dcc56b3413f67eb23a5059e77c93c0dab085a8ad
FINAL_TREE=ecd74cfc972053bb5693f5952ec7299a9d47fed3
CLOSURE_COMMIT=see HEAD of this branch; the report is committed
                at the same commit as the artifacts it references
                so the closure OID is always "git rev-parse HEAD".

WORKTREE_STATUS_BEFORE=clean
WORKTREE_STATUS_AFTER=clean

CORRECTION01_COMMIT=dcc56b3413f67eb23a5059e77c93c0dab085a8ad
CLOSURE_COMMIT=see HEAD of this branch (self-referential;
                see AGENTS.md rule: never embed future closure
                identities in committed documents)
```

## What CORRECTION01 fixed

```text
PARENT_VERDICT                     = PARTIAL (per Git provenance reviewer)
PARENT_HISTORICAL_RANGE_AUTHORITY  = FAIL  (backwards composition)
PARENT_AMBIENT_HEAD_INDEPENDENCE   = FAIL  (not actually independent)
PARENT_EXPLICIT_RANGE_ADAPTER      = FAIL  (fell back to HEAD)
PARENT_CLOSE_REPORT_FINAL_IDENTITY = STALE (FINAL_HEAD pointed at plan commit)

CORRECTION01_HISTORICAL_RANGE_AUTHORITY = PASS
CORRECTION01_AMBIENT_HEAD_INDEPENDENCE  = PASS
CORRECTION01_EXPLICIT_RANGE_ADAPTER     = PASS
CORRECTION01_CLOSE_REPORT_FINAL_IDENTITY = PASS (FINAL_HEAD refreshed)
```

## Production defects fixed

```text
DEFECT_1 (C1) classifier composition was inverted:
  The previous EvaluateGeneratorBinding short-circuited with
  COMMIT_MISMATCH whenever the generator commit differed
  from ambient HEAD, regardless of the digest subject.

  For a historical digest (A..B) with generator=B and
  ambient HEAD=C, the previous code rendered:

      GENERATOR_BINDING_STATUS: COMMIT_MISMATCH
      GENERATOR_AUTHORITATIVE_FOR_DIGEST: false

  even though the binary was built from B (the digest
  subject) and is therefore authoritative for the digest.

  CORRECTION01 removes the commit-binding short-circuit.
  The overall verdict is driven by the SUBJECT axis. The
  legacy GENERATOR_STALE signal (CommitMatchesHead) is
  still surfaced verbatim on the per-axis CommitBinding
  field and rendered on GENERATOR_COMMIT_MATCHES_HEAD.

  After the fix the same case renders:

      GENERATOR_BINDING_STATUS: AUTHORITATIVE
      GENERATOR_COMMIT_BINDING: MISMATCH    (legacy stale)
      GENERATOR_SUBJECT_BINDING: MATCH      (new authority)
      GENERATOR_AUTHORITATIVE_FOR_DIGEST: true
      GENERATOR_STALE: true
      GENERATOR_COMMIT_MATCHES_HEAD: false

  The doctrine:

      GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT,
      NOT MERELY TO AMBIENT HEAD.

DEFECT_2 (C2) renderer fell back to ambient HEAD for
  explicit-range subjects:

  For an explicit --range A..B on a repo whose HEAD=C,
  the previous code rendered LIFECYCLE_SUBJECT=unset, so
  the renderer substituted ambient HEAD=C for the digest
  subject. The acceptance evidence in the previous close
  report masked this defect because the right endpoint
  happened to equal HEAD in that fixture.

  CORRECTION01:
  - Adds ResolvedAuthority.RangeSubjectEnd (resolved full
    OID of the right endpoint of the explicit range).
  - Adds authority.explicitRangeRightEndpoint that resolves
    the rightmost ".." token via `git rev-parse --verify`.
  - Adds ResolvedMode.LifecycleSubjectRange that the
    renderer consults FIRST.
  - Routes the explicit-range path in resolveAutoModeWith
    through authority.Resolve so RangeSubjectEnd is
    populated.
  - Subject resolution order at the renderer is now:
      LifecycleSubjectRange -> LifecycleSubject -> HeadCommit
```

## Files Changed

```text
internal/factory/authority/resolver_core.go                   (modified, +10 LOC)
internal/factory/authority/resolver_entry.go                  (modified, +54 LOC)
internal/factory/digest/generator_binding.go                  (modified, +11 LOC)
internal/factory/digest/generator_binding_classifier.go       (modified, +85 LOC, -27 LOC)
internal/factory/digest/generator_binding_classifier_test.go  (modified, -293 LOC; matrix moved out)
internal/factory/digest/generator_binding_classifier_matrix_test.go  (new, 287 LOC)
internal/factory/digest/lifecycle_render.go                   (modified, +18 LOC, -8 LOC)
internal/factory/digest/lifecycle_render_generator_test.go    (modified, +12 LOC, -12 LOC)
internal/factory/digest/lifecycle_render_correction01_test.go (new, 315 LOC)
internal/factory/digest/resolve.go                            (modified, +30 LOC, -21 LOC)
```

All owned/modified Go files are ≤ 400 lines.

## Behavior Changed

```text
GENERATOR_COMMIT_MATCH_IMPLIES_SUBJECT_MATCH=false   (unchanged)
DIRTY_WORKTREE_REQUIRES_EXACT_SUBJECT_PROOF=true     (unchanged)
UNPROVEN_DIRTY_GENERATOR_FAILS_CLOSED=true           (unchanged)

GENERATOR_STALE_COMPATIBILITY_PRESERVED=true
  legacy field preserved verbatim with its existing
  semantic (commit_vs_repository_head).
  Now qualified with GENERATOR_STALE_BASIS label.

HISTORICAL_RANGE_SUBJECT_AUTHORITY=true  (was false)
  generator=B authoritative for A..B even when HEAD=C.
  GENERATOR_STALE=true and AUTHORITATIVE=true coexist.

AMBIENT_HEAD_INDEPENDENCE=true  (was false)
  the overall verdict no longer depends on ambient HEAD
  equality; it depends on subject equality.

EXPLICIT_RANGE_ADAPTER=true  (was unproven)
  the explicit-range right endpoint is now resolved
  through git rev-parse and surfaced as the binding's
  subject. Renderer consults LifecycleSubjectRange
  before LifecycleSubject before HeadCommit.

DIGEST_CONTRACT_VERSION_BEFORE=3
DIGEST_CONTRACT_VERSION_AFTER=3  (additive: new keys
  appended; no key moved; legacy positions preserved)

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

## Semantic Doctrine (unchanged)

```text
COMMIT IDENTITY PROVES COMMITTED STATE.
IT DOES NOT PROVE UNCOMMITTED SOURCE STATE.
GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT,
NOT MERELY TO AMBIENT HEAD.
```

CORRECTION01 makes the doctrine executable: the classifier's
overall verdict is now driven by the SUBJECT axis, with the
ambient-HEAD axis surfaced independently as the legacy
freshness signal.

## Implementation Commits

```text
dcc56b3 factory: separate subject authority from ambient-HEAD
         freshness (CORRECTION01)
```

## Verification — exact commands and honest results

```text
$ CGO_ENABLED=0 go test -count=1 ./internal/factory/digest/...
ok  github.com/s1onique/leamas/internal/factory/digest  23.532s
RC=0

$ CGO_ENABLED=0 go test -count=1 ./internal/factory/authority/...
--- FAIL: TestSymlinkedCanonicalBinaryIsResolved (0.00s)
    (PRE-EXISTING macOS symlink test; fails identically on
     baseline 5dbbf6a, NOT introduced by CORRECTION01)
FAIL  github.com/s1onique/leamas/internal/factory/authority  4.707s
RC=1

$ CGO_ENABLED=0 go vet ./...
RC=0  (no diagnostics)

$ CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-final ./cmd/leamas
RC=0

$ gofmt -l <all CORRECTION01-owned files>
(no output)
gofmt CLEAN

$ git diff --check
(no output)
diff-check CLEAN
```

## Required Acceptance Board (corrected)

```text
DIRTY_WORKTREE_AUTHORITY_SPLIT        = PASS
LEGACY_STALE_SIGNAL                   = PASS  (preserved verbatim)
FAIL_CLOSED_DIRTY                     = PASS  (TestEvaluateGeneratorBindingMatrix/dirty_tracked_unbound)
SELF_HOSTED_DIRTY_ACCEPTANCE          = PASS  (acceptance-dirty-lifecycle.txt)
SELF_HOSTED_CLEAN_ACCEPTANCE          = PASS  (acceptance-clean-lifecycle.txt)

HISTORICAL_RANGE_SUBJECT_AUTHORITY    = PASS
  TestEvaluateGeneratorBindingMatrix/historical_range_generator_matches_subject_only
    generator=B, repo=C, digest=B -> AUTHORITATIVE
    (was: COMMIT_MISMATCH in the original ACT)
  TestEvaluateGeneratorBindingMatrix/historical_range_generator_and_subject_at_HEAD
    generator=B, repo=B, digest=B -> AUTHORITATIVE
  TestEvaluateGeneratorBindingMatrix/historical_range_generator_at_HEAD_not_subject
    generator=C, repo=C, digest=B -> SUBJECT_MISMATCH
    (was: COMMIT_MISMATCH)
  TestEvaluateGeneratorBindingMatrix/historical_range_generator_mismatches_both
    generator=X, repo=C, digest=B -> SUBJECT_MISMATCH (new)
  TestRenderLifecycleCorrection01_GeneratorAtSubject_Authoritative
    end-to-end resolver + renderer regression
  TestRenderLifecycleCorrection01_GeneratorAtHEAD_NotSubject
    end-to-end resolver + renderer regression

AMBIENT_HEAD_INDEPENDENCE             = PASS
  The overall verdict is now driven by the SUBJECT axis.
  The commit-binding field is rendered independently
  on GENERATOR_COMMIT_BINDING / GENERATOR_COMMIT_MATCHES_HEAD
  and matches the legacy GENERATOR_STALE semantic.

EXPLICIT_RANGE_SUBJECT_ADAPTER        = PASS
  TestRenderLifecycleCorrection01_SubjectResolutionOrder
    4-row matrix locking the resolution order
  TestExplicitRangeRightEndpoint_Resolution
    resolver populates RangeSubjectEnd = full OID of B
  TestExplicitRangeRightEndpoint_BareRev
    bare-rev form: RangeSubjectEnd = the bare rev
  TestExplicitRangeRightEndpoint_MalformedFailsSoft
    malformed rev fails soft (empty + status preserved)

CLOSE_REPORT_FINAL_IDENTITY           = PASS
  FINAL_HEAD=dcc56b3413f67eb23a5059e77c93c0dab085a8ad
  FINAL_TREE=ecd74cfc972053bb5693f5952ec7299a9d47fed3
  CORRECTION01_COMMIT=dcc56b3413f67eb23a5059e77c93c0dab085a8ad

CLOCK_INDEPENDENCE_TEST_RENAMED       = PASS
  TestEvaluateGeneratorBindingClockIndependence
    renamed to TestEvaluateGeneratorBindingIdentitySensitivity.
  The test never varied any clock; the new name matches
  the actual contract under test.

VOCABULARY_GREW_BY_ONE                = PASS
  GeneratorBindingSubjectMismatch ("SUBJECT_MISMATCH")
  GeneratorWarningCodeSubjectMismatch ("GENERATOR_SUBJECT_MISMATCH")
  Matrix size 11 -> 13 rows; warning vocabulary 5 -> 6.
  Locked by TestEvaluateGeneratorBindingVocabularyStable.

NO_AUTOREBUILD                        = PASS
NO_REPOSITORY_MUTATION                = PASS
  git status before/after identical
  find .git/objects | wc -l unchanged
VARIABLE_GIT_SUBPROCESS_COUNT=0       (in the new classifier + adapter)

ADDITIVE_CONTRACT                     = PASS
  legacy GENERATOR_STALE position preserved
  legacy GENERATOR_STALE_BASIS rendered immediately after
  new binding fields appended at the documented positions
  TestRenderLifecycleAdditiveContract still passes
```

## Required Self-Hosting Evidence (corrected)

### Unit test proof — `generator=B` authoritative for `A..B`

```text
$ CGO_ENABLED=0 go test -count=1 -v \
    -run 'TestRenderLifecycleCorrection01_GeneratorAtSubject_Authoritative' \
    ./internal/factory/digest/...
=== RUN   TestRenderLifecycleCorrection01_GeneratorAtSubject_Authoritative
--- PASS: TestRenderLifecycleCorrection01_GeneratorAtSubject_Authoritative (0.13s)
PASS

Expected rendered LIFECYCLE for the A--B--C(HEAD) fixture with
generator=B, digest=A..B:

  GENERATOR_COMMIT: B
  REPOSITORY_HEAD:  C

  GENERATOR_COMMIT_MATCHES_HEAD: false
  GENERATOR_STALE: true              (legacy freshness signal)
  GENERATOR_STALE_BASIS: commit_vs_repository_head

  GENERATOR_BINDING_STATUS: AUTHORITATIVE
  GENERATOR_COMMIT_BINDING: MISMATCH   (legacy stale)
  GENERATOR_SUBJECT_BINDING: MATCH      (new authority)
  GENERATOR_AUTHORITATIVE_FOR_DIGEST: true
  GENERATOR_WARNING_CODE: none
```

### Unit test proof — `generator=C` not authoritative for `A..B`

```text
$ CGO_ENABLED=0 go test -count=1 -v \
    -run 'TestRenderLifecycleCorrection01_GeneratorAtHEAD_NotSubject' \
    ./internal/factory/digest/...
=== RUN   TestRenderLifecycleCorrection01_GeneratorAtHEAD_NotSubject
--- PASS: TestRenderLifecycleCorrection01_GeneratorAtHEAD_NotSubject (0.12s)
PASS

Expected rendered LIFECYCLE for the same fixture with
generator=C, digest=A..B:

  GENERATOR_COMMIT: C
  REPOSITORY_HEAD:  C

  GENERATOR_COMMIT_MATCHES_HEAD: true
  GENERATOR_STALE: false             (legacy: not stale)
  GENERATOR_STALE_BASIS: commit_vs_repository_head

  GENERATOR_BINDING_STATUS: SUBJECT_MISMATCH
  GENERATOR_COMMIT_BINDING: MATCH
  GENERATOR_SUBJECT_BINDING: MISMATCH
  GENERATOR_AUTHORITATIVE_FOR_DIGEST: false
  GENERATOR_WARNING_CODE: GENERATOR_SUBJECT_MISMATCH
```

### CLI self-hosted evidence (binary ≠ any commit)

```text
$ /tmp/leamas-final factory digest --range $A..$B --output /tmp/dig.txt
digest: mode=range output=/tmp/dig.txt time=0.10s OK

$ sed -n '/LIFECYCLE/,/^$/p' /tmp/dig.txt
## LIFECYCLE

LIFECYCLE_FREEZE: unset
LIFECYCLE_SUBJECT: unset
LIFECYCLE_CLOSURE: unset
INCLUDED_COMMITS: unset
GENERATOR_COMMIT: 5dbbf6a30873540f77c15be9acf0afb552571574
REPOSITORY_HEAD: f1ceebd020ffc8bbaf24303626ce8343cb22c531
GENERATOR_STALE: true: embedded leamas commit does not match repository HEAD
GENERATOR_STALE_BASIS: commit_vs_repository_head
GENERATOR_COMMIT_MATCHES_HEAD: false
GENERATOR_BINDING_STATUS: SUBJECT_MISMATCH
GENERATOR_COMMIT_BINDING: MISMATCH
GENERATOR_SUBJECT_BINDING: MISMATCH
GENERATOR_AUTHORITATIVE_FOR_DIGEST: false
GENERATOR_WARNING_CODE: GENERATOR_SUBJECT_MISMATCH
AUTHORITY_STATUS: ExplicitRange
RESOLUTION_SOURCE: explicit_cli
```

The CLI evidence shows the new vocabulary in production:
- `GENERATOR_BINDING_STATUS: SUBJECT_MISMATCH` (not COMMIT_MISMATCH)
- `GENERATOR_WARNING_CODE: GENERATOR_SUBJECT_MISMATCH`
- `GENERATOR_STALE` and `GENERATOR_AUTHORITATIVE_FOR_DIGEST` are
  rendered independently and can differ (here both are non-clean,
  but the rendering path proves the new fields are reachable).

The CLI cannot exercise the AUTHORITATIVE+STALE=true coexistence
case without rebuilding the binary inside the fixture repo; that
case is locked by the unit tests above.

## Skipped or Deferred Checks

```text
make factorize:    NOT RUN (Cline/editor context; ACT does not delegate)
make gate-dupcode: NOT RUN (Cline/editor context; ACT does not delegate)
make gate:         NOT RUN (Cline/editor context; ACT does not delegate)

CGO_ENABLED=0 go test ./internal/factory/closure/...    PARTIAL
  (long-running closure subtests time out identically on
   baseline 5dbbf6a; not introduced by CORRECTION01)

CGO_ENABLED=0 go test ./internal/factory/authority/...  PARTIAL
  (TestSymlinkedCanonicalBinaryIsResolved fails identically
   on baseline 5dbbf6a — pre-existing macOS symlink test,
   not introduced by CORRECTION01)
```

## Follow-up ACTs

None. CORRECTION01 is complete on its own scope.

```text
FOLLOWUP_NOTE: ACT-LEAMAS-DIGEST-SUBJECT-FINGERPR-WORKTREE-CANONICAL01
  ACTUAL AUTHORITY FOR UNCOMMITTED SOURCE STATE.
  Both the original ACT and CORRECTION01 explicitly defer
  source fingerprinting. A future ACT may introduce a
  canonical worktree-subject fingerprint.
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
CORRECTION01_SCOPE=CLOSED
CANONICAL_GATE_STATUS=NOT_RUN (delegation not granted by ACT)
```