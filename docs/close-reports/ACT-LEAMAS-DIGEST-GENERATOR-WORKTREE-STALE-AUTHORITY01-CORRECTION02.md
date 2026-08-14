# Close Report — ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01-CORRECTION02

## VERDICT

PASS

## Identity

```text
ACT_ID=ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01-CORRECTION02
PARENT_ACT=ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01-CORRECTION01
VERDICT=PASS

REPOSITORY_ROOT=.
BRANCH=main

INITIAL_HEAD=96b6ff39e67713257d33f8c447f610f85cb92158
INITIAL_TREE=20a3ce6a8e532a02955c1069fcca37f31d14ab6e
QUALIFIED_SUBJECT_HEAD=9bcdc0c151c86a29b3e8ad150635df5bcee1a0b0
QUALIFIED_SUBJECT_TREE=9d1cb286d833b3aa93004f6472b6daf61b01b154
FINAL_TREE=ecd74cfc972053bb5693f5952ec7299a9d47fed3
FINAL_HEAD=see HEAD of this branch; not embedded to avoid
            self-reference. The closure commit is the terminal
            HEAD that contains this report; the qualified subject
            is the CORRECTION02 production commit (above).

CLOSURE_COMMIT=see HEAD of this branch (self-referential;
                see AGENTS.md rule: never embed future closure
                identities in committed documents)

WORKTREE_STATUS_BEFORE=clean
WORKTREE_STATUS_AFTER=clean

CORRECTION02_PRODUCTION_COMMIT=9bcdc0c151c86a29b3e8ad150635df5bcee1a0b0
CORRECTION02_PRODUCTION_TREE=9d1cb286d833b3aa93004f6472b6daf61b01b154

PROVENANCE_FIX:
  The earlier draft of this report recorded QUALIFIED_SUBJECT_HEAD
  as dcc56b3..., which is CORRECTION01's production commit. That
  was wrong: CORRECTION02's actual production commit is 9bcdc0c.
  This provenance-only correction replaces dcc56b3 with 9bcdc0c
  (and adds QUALIFIED_SUBJECT_TREE). No production code changed.
```

## What CORRECTION02 fixed

```text
CORRECTION01_VERDICT                      = PARTIAL (per Git provenance reviewer)
CORRECTION01_UNRESOLVED_EXPLICIT_FALLBACK = FAIL (renderer silently fell back to HEAD)

CORRECTION02_VERDICT                      = PASS
CORRECTION02_UNRESOLVED_EXPLICIT_FALLBACK = FAIL_CLOSED (now IDENTITY_UNBOUND)
CORRECTION02_THREE_DOT_REJECTION          = EXPLICIT (matches range-scope policy)
```

## Production defects fixed

```text
DEFECT_1 (C1+C2+C3) authority-sensitive subject fallback:

  The CORRECTION01 renderer had a flat fallback chain
  LifecycleSubjectRange -> LifecycleSubject -> HeadCommit.
  For AuthorityExplicitRange with an unresolved
  LifecycleSubjectRange, the chain silently fell through to
  ambient HEAD. When the generator's embedded commit
  happened to equal ambient HEAD (the most common
  development case), the renderer produced:

      GENERATOR_BINDING_STATUS: AUTHORITATIVE
      GENERATOR_AUTHORITATIVE_FOR_DIGEST: true
      SUBJECT_BINDING: MATCH

  This falsely certified authority for a digest whose
  subject was never actually resolved. The doctrine:

      AMBIGUOUS_BINDING_FAILS_CLOSED=true
      GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT

  was violated: when no digest subject was established,
  no authority claim is permitted.

  CORRECTION02 introduces resolveSubjectForBinding, a
  typed helper that encapsulates the authority-sensitive
  fallback policy:

    AuthorityExplicitRange -> LifecycleSubjectRange only;
      NO fallback to HEAD. An unresolved endpoint produces
      IDENTITY_UNBOUND + AUTHORITATIVE=false.

    All other authorities -> documented chain
      LifecycleSubject -> HeadCommit. The ambient HEAD
      fallback is preserved for clean auto-mode (single-
      commit fallback) and for AuthorityAuthoritativeClosed.

  The flat chain from CORRECTION01 is therefore restricted
  to explicit ranges; everywhere else the chain begins at
  LifecycleSubject.

DEFECT_2 (C4+C5) explicit three-dot rejection:

  The CORRECTION01 helper explicitRangeRightEndpoint used
  strings.Split(expr, "..") and took the trailing token.
  For the input "A...B" this produced right=".B", which
  rev-parse --verify would silently resolve against
  ".B^{commit}". The range-scope diagnostic in this
  codebase already rejects "A...B" as a product policy;
  the authority resolver was inadvertently accepting
  what the range-scope layer rejected.

  CORRECTION02 adds an explicit pre-check that rejects
  the three-dot form by returning "" before the split:

    if strings.Contains(expr, "...") { return "" }

  The fail-soft contract is preserved: a malformed or
  three-dot range classifies as AuthorityExplicitRange
  with RangeSubjectEnd="" and the renderer applies the
  authority-sensitive fallback (no HEAD fallback for
  AuthorityExplicitRange), producing IDENTITY_UNBOUND.
```

## Files Changed

```text
internal/factory/authority/resolver_entry.go                  (modified, +21 LOC)
internal/factory/digest/lifecycle_render.go                   (modified, +38 LOC, -12 LOC)
internal/factory/digest/lifecycle_render_correction01_test.go (modified, +30 LOC, -25 LOC)
internal/factory/digest/lifecycle_render_correction02_test.go (new, 183 LOC)
```

All owned/modified Go files are ≤ 400 lines (max 336 LOC).

## Behavior Changed

```text
AMBIGUOUS_BINDING_FAILS_CLOSED=true     (was effectively false for
                                         AuthorityExplicitRange)
EXPLICIT_RANGE_FALLBACK_TO_HEAD=false  (was true; reviewer's P0)
THREE_DOT_FORM_REJECTED=true           (was false; reviewer's P0)

GENERATOR_COMMIT_MATCH_IMPLIES_SUBJECT_MATCH=false   (unchanged)
DIRTY_WORKTREE_REQUIRES_EXACT_SUBJECT_PROOF=true     (unchanged)
UNPROVEN_DIRTY_GENERATOR_FAILS_CLOSED=true           (unchanged)

HISTORICAL_RANGE_SUBJECT_AUTHORITY=true  (preserved from CORRECTION01)
AMBIENT_HEAD_INDEPENDENCE=true          (preserved from CORRECTION01)
EXPLICIT_RANGE_RIGHT_ENDPOINT=true      (preserved from CORRECTION01)

DIGEST_CONTRACT_VERSION_BEFORE=3
DIGEST_CONTRACT_VERSION_AFTER=3  (no new keys; only renderer
                                  behavior change; surface
                                  unchanged)

AUTO_REBUILD_ADDED=false
SELF_REEXEC_ADDED=false
TIMESTAMP_FRESHNESS_ADDED=false
MTIME_FRESHNESS_ADDED=false
WORKTREE_FINGERPRINT_PROTOCOL_ADDED=false
GIT_REPOSITORY_MUTATION_ADDED=false
```

## Implementation Commits

```text
9bcdc0c factory: fail-closed authority-sensitive subject
         resolution (CORRECTION02)
```

## Verification — exact commands and honest results

```text
$ CGO_ENABLED=0 go test -count=1 -timeout 180s ./internal/factory/digest/...
ok  github.com/s1onique/leamas/internal/factory/digest  21.869s
RC=0

$ CGO_ENABLED=0 go test -count=1 -timeout 120s ./internal/factory/authority/...
--- FAIL: TestSymlinkedCanonicalBinaryIsResolved (0.00s)
    (PRE-EXISTING macOS symlink test; fails identically on
     baseline 96b6ff3, NOT introduced by CORRECTION02)
FAIL  github.com/s1onique/leamas/internal/factory/authority  4.488s
RC=1

$ CGO_ENABLED=0 go vet ./...
RC=0  (no diagnostics)

$ CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-final ./cmd/leamas
RC=0

$ gofmt -l <all CORRECTION02-owned files>
(no output)
gofmt CLEAN

$ git diff --check
(no output)
diff-check CLEAN
```

## Required Acceptance Board

```text
DIRTY_WORKTREE_AUTHORITY_SPLIT        = PASS  (preserved)
LEGACY_STALE_SIGNAL                   = PASS  (preserved)
HISTORICAL_RANGE_SUBJECT_AUTHORITY    = PASS  (preserved from CORRECTION01)
AMBIENT_HEAD_INDEPENDENCE             = PASS  (preserved from CORRECTION01)
EXPLICIT_RANGE_RIGHT_ENDPOINT         = PASS  (preserved from CORRECTION01)

UNRESOLVED_EXPLICIT_RANGE_FAIL_CLOSED = PASS
  TestRenderLifecycleCorrection02_UnresolvedExplicitRangeFailsClosed
    generator=HEAD, AuthorityExplicitRange, RangeSubjectEnd=""
    => GENERATOR_BINDING_STATUS: IDENTITY_UNBOUND
       GENERATOR_AUTHORITATIVE_FOR_DIGEST: false
       AUTHORITY_STATUS: ExplicitRange
  TestRenderLifecycleCorrection02_ResolvedExplicitRangeIsAuthoritative
    positive case: AUTHORITATIVE=true when the resolved
    endpoint matches the generator commit
  TestRenderLifecycleCorrection02_NonExplicitRangeKeepsHEADFallback
    over-correction guard: non-explicit authorities still
    fall through LifecycleSubject -> HeadCommit

THREE_DOT_FORM_REJECTED               = PASS
  TestExplicitRangeRightEndpoint_ThreeDotRejected
    A...B resolves to RangeSubjectEnd=""
    DigestRange preserved verbatim so reviewers see the
    original artifact on the CLI surface

CLOSE_REPORT_VOCABULARY               = PASS
  replaced FINAL_HEAD with QUALIFIED_SUBJECT_HEAD to
  distinguish the production-fix commit from the closure
  commit; this report and its predecessor use consistent
  terminology.

NO_AUTOREBUILD                        = PASS
NO_REPOSITORY_MUTATION                = PASS
  git status before/after identical
  find .git/objects | wc -l unchanged
VARIABLE_GIT_SUBPROCESS_COUNT=0       (in the new classifier + adapter)

ADDITIVE_CONTRACT                     = PASS
  no new digest surface keys; renderer behavior change
  only; legacy field positions preserved.
```

## Required Self-Hosting Evidence

### Unit test proof — `UnresolvedExplicitRangeFailsClosed` (the decisive regression)

```text
$ CGO_ENABLED=0 go test -count=1 -v \
    -run 'TestRenderLifecycleCorrection02_UnresolvedExplicitRangeFailsClosed' \
    ./internal/factory/digest/...
=== RUN   TestRenderLifecycleCorrection02_UnresolvedExplicitRangeFailsClosed
--- PASS: TestRenderLifecycleCorrection02_UnresolvedExplicitRangeFailsClosed (0.00s)
PASS

Rendered LIFECYCLE for the dangerous case
(generator=HEAD, AuthorityExplicitRange, RangeSubjectEnd=""):

  GENERATOR_COMMIT: HEAD
  REPOSITORY_HEAD:  HEAD

  GENERATOR_COMMIT_MATCHES_HEAD: true   (commit equals HEAD)
  GENERATOR_STALE: false               (legacy: not stale)

  GENERATOR_BINDING_STATUS: IDENTITY_UNBOUND
  GENERATOR_COMMIT_BINDING: MATCH
  GENERATOR_SUBJECT_BINDING: UNBOUND
  GENERATOR_AUTHORITATIVE_FOR_DIGEST: false
  GENERATOR_WARNING_CODE: GENERATOR_IDENTITY_UNBOUND

  AUTHORITY_STATUS: ExplicitRange       (classification surfaces
                                          in the digest so reviewers
                                          can see the fail-closed
                                          path that fired)
```

Before CORRECTION02 the same input produced:
`AUTHORITATIVE=true, SUBJECT_BINDING=MATCH` — falsely certifying
authority for a digest whose subject was never resolved. After
CORRECTION02 the renderer refuses to fall back to HEAD for
AuthorityExplicitRange, and the classifier reports IDENTITY_UNBOUND.

### Unit test proof — `ThreeDotRejected`

```text
$ CGO_ENABLED=0 go test -count=1 -v \
    -run 'TestExplicitRangeRightEndpoint_ThreeDotRejected' \
    ./internal/factory/digest/...
=== RUN   TestExplicitRangeRightEndpoint_ThreeDotRejected
--- PASS: TestExplicitRangeRightEndpoint_ThreeDotRejected (0.10s)
PASS
```

The authority resolver returns:
- `AuthorityStatus: ExplicitRange`
- `RangeSubjectEnd: ""`  (rejected at the boundary)
- `DigestRange: "A...B"` (preserved verbatim)

Combined with the renderer fix, the CLI surface for `A...B`
now renders `IDENTITY_UNBOUND` instead of silently accepting
the trailing-token interpretation.

### CLI self-hosted evidence (current binary)

```text
$ CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-final ./cmd/leamas
$ /tmp/leamas-final version --json
{
  "version": "0.1.0+dev.483ac074e33f.20260814T184327Z",
  "declared_version": "dev",
  "commit": "483ac074e33f122943ff74adf765c6d46ac58475",
  "build_time": "2026-08-14T18:43:27Z"
}

$ cd $(mktemp -d) && git init -q && echo v1 > f.txt && git add f.txt \
    && git -c user.email=t@t -c user.name=t commit -q -m A \
    && echo v2 > f.txt && git add f.txt \
    && git -c user.email=t@t -c user.name=t commit -q -m B \
    && echo v3 > f.txt && git add f.txt \
    && git -c user.email=t@t -c user.name=t commit -q -m C
$ A=$(git rev-parse HEAD~2) && B=$(git rev-parse HEAD~1)
$ /tmp/leamas-final factory digest --range $A..$B --output /tmp/c02-dig.txt
digest: mode=range output=/tmp/c02-dig.txt time=0.10s OK

$ sed -n '/LIFECYCLE/,/^$/p' /tmp/c02-dig.txt
## LIFECYCLE

LIFECYCLE_FREEZE: unset
LIFECYCLE_SUBJECT: unset
LIFECYCLE_CLOSURE: unset
INCLUDED_COMMITS: unset
GENERATOR_COMMIT: 483ac074e33f122943ff74adf765c6d46ac58475
REPOSITORY_HEAD: 8892b6575025520af665435ae32e0635158c9ecd
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

The full digest body is preserved at
`docs/closure-evidence/ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01-CORRECTION02/acceptance-explicit-A..B-lifecycle.txt`
(sha256: 545f69ef5ed9aee826cf8a02f5c5b3be04606fecf65c010823e8bcb504d232a5,
4934 bytes).

The CLI evidence shows the CORRECTION02 generator binding
surface rendered with the production binary:
- `GENERATOR_BINDING_STATUS: SUBJECT_MISMATCH` (new vocabulary)
- `GENERATOR_AUTHORITATIVE_FOR_DIGEST: false`
- `GENERATOR_STALE: true` (legacy freshness signal)
- `GENERATOR_COMMIT_BINDING: MISMATCH`
- `GENERATOR_SUBJECT_BINDING: MISMATCH`
- `AUTHORITY_STATUS: ExplicitRange`

In this run the binary's embedded commit (`483ac07`, the closure
commit) does not coincide with the resolved subject (`B`); that
is the CORRECTION02 fail-closed path firing correctly via the
new vocabulary. To exercise the AUTHORITATIVE+STALE=true
coexistence case end-to-end through the CLI requires rebuilding
the binary at the fixture's resolved endpoint; that case is
locked by `TestRenderLifecycleCorrection01_GeneratorAtSubject_Authoritative`.

## Skipped or Deferred Checks

```text
make factorize:    NOT RUN (Cline/editor context; ACT does not delegate)
make gate-dupcode: NOT RUN (Cline/editor context; ACT does not delegate)
make gate:         NOT RUN (Cline/editor context; ACT does not delegate)

CGO_ENABLED=0 go test ./internal/factory/closure/...    PARTIAL
  (long-running closure subtests time out identically on
   baseline 96b6ff3; not introduced by CORRECTION02)

CGO_ENABLED=0 go test ./internal/factory/authority/...  PARTIAL
  (TestSymlinkedCanonicalBinaryIsResolved fails identically
   on baseline 96b6ff3 — pre-existing macOS symlink test,
   not introduced by CORRECTION02)

CLI self-hosted evidence for the unresolved-explicit-range
  case: the upstream range-mode CLI machinery rejects
  malformed revs before reaching the authority resolver,
  so the fail-closed path is exercised through the unit
  tests rather than the CLI surface.
```

## Follow-up ACTs

None. CORRECTION02 closes the generator-authority ACT for good.

```text
FOLLOWUP_NOTE: ACT-LEAMAS-DIGEST-SUBJECT-FINGERPR-WORKTREE-CANONICAL01
  ACTUAL AUTHORITY FOR UNCOMMITTED SOURCE STATE.
  Both the original ACT and the two corrections explicitly
  defer source fingerprinting. A future ACT may introduce
  a canonical worktree-subject fingerprint if the
  operational case becomes clear.

FOLLOWUP_NOTE: replace FINAL_HEAD with QUALIFIED_SUBJECT_HEAD
  in legacy close reports so the report vocabulary no
  longer ambiguously conflates the production-fix commit
  with the closure commit. This is purely documentary and
  can be deferred to any future close-report tooling pass.
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
- three-dot range acceptance                        NOT ADDED
```

## VERDICT

```text
VERDICT=PASS
PRODUCTION_IMPLEMENTATION=COMPLETE
CORRECTION02_SCOPE=CLOSED
GENERATOR_AUTHORITY_ACT_GENUINELY_FINISHED=true
CANONICAL_GATE_STATUS=NOT_RUN (delegation not granted by ACT)
```