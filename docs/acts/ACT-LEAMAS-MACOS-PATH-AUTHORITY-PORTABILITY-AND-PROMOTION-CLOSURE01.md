# ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01

## Status

OPEN — IMPLEMENTATION SUBJECT (PORTABILITY CLOSURE)

## Mission

Close the macOS portability defects that the previous ACT
(`ACT-LEAMAS-MAC-HANDOFF-VERIFY-AND-MAIN-PROMOTION01`) documented as
PARTIAL. That ACT's PARTIAL close report identified exactly two
blocking defects:

1. macOS path canonicalization
   (`TestClosureBinaryGateRealProductionHappyPath`)
2. Hard-coded Linux binary paths
   (`TestGateOsRunnerStartWaitContract`)

This ACT publishes the implementation subject that fixes both,
preserves the historical PARTIAL report as committed evidence, and
declares the chain PASS-FULL.

## Authority

The previous ACT's PARTIAL close report explicitly delegated the
following scope (see "Next ACT Required" section):

```text
ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01

Scope:
  1. Preserve origin/main=e1b54b3 as starting authority
  2. Reproduce and fix TestClosureBinaryGateRealProductionHappyPath
  3. Fix path canonicalization at authority boundary
  4. Fix /bin/true /bin/false assumptions (use hermetic helpers)
  5. Require: TestClosureBinaryGateRealProductionHappyPath=PASS,
              MAC_NEW_FAILURES=0
  6. Commit this report with corrected PARTIAL verdict
  7. Write portability close report before final commit
  8. Push normally (no rebase, no force)
  9. Verify: local main == advertised == fetched, worktree clean
 10. Declare whole chain PASS-FULL
```

Publication authorities delegated by the PARTIAL close report:

```text
COMMIT_ALLOWED   = true   (PARTIAL report + portability subject)
PUSH_ALLOWED     = true   (origin main:main, no rebase, no force)
TAG_ALLOWED      = false  (no tag requested by PARTIAL scope)
HISTORY_REWRITE  = false  (forward corrective commits only)
EXPENSIVE_GATE   = NOT RUN (no delegation for make gate)
```

## Base

```text
BASE_COMMIT     = e1b54b31ec9c2afb6917804fbab9512bd6f2d8bb
BASE_TREE       = 42df2a6bdfff73af75a9df902158721d734c9801
BASE_IN_ANCESTRY= true
WORKTREE_STATUS = dirty (PARTIAL report + 5 portability changes)
```

The base commit `e1b54b3` is the verified promotion authority from
the previous ACT. It is preserved as the starting identity of this
ACT.

## Phases

### Phase 1 — Implementation subject (S)

Commit the implementation body exactly once. The subject combines:

1. The historical PARTIAL close report at
   `docs/close-reports/ACT-LEAMAS-MAC-HANDOFF-VERIFY-AND-MAIN-PROMOTION01.md`
   (preserved as committed evidence).
2. The five CORRECTION06 / CORRECTION07 portability changes:
   - `internal/factory/closure/closure_protocol_v2_executor.go`
     (CORRECTION06: `filepath.EvalSymlinks` on `MkdirTemp` result)
   - `internal/factory/closure/closure_evidence_publication_runner_adapter.go`
     (CORRECTION06: preserve `execResult` on tree mismatch)
   - `internal/factory/closure/binary_gate_testhelpers_test.go`
     (CORRECTION06: `filepath.EvalSymlinks` on `t.TempDir()`)
   - `internal/factory/closure/closure_exact_subject_binary_output_test.go`
     (CORRECTION06: `filepath.EvalSymlinks` on `t.TempDir()`)
   - `internal/factory/closure/evidence/gate_runner_lifecycle_test.go`
     (CORRECTION07: `exec.LookPath` for `true`, `false`,
     `sleep`, `sh`)

The commit MUST record the implementation identity in the manifest
after the commit lands.

### Phase 2 — Focused verification on committed subject

Run the focused test matrix listed in the frozen closure plan:

```text
portability-closure-real-production-canary
portability-closure-isolated-fixture-canary
portability-closure-output-confinement
portability-closure-runner-start-wait-contract
portability-closure-executor-evalsymlinks
portability-closure-adapter-evidence-preservation
portability-closure-static-build
portability-closure-gate-fast-lane
```

The focused tests prove that the previous ACT's two named defects
no longer reproduce on the committed subject.

### Phase 3 — Render closure (C)

Use `leamas factory close run`, `verify`, and `render` to produce:

```text
docs/closure-manifests/ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01.json
docs/close-reports/ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01.md
```

The close report records:

- the implementation subject's commit and tree OID;
- the focused verification results;
- a `CLOSE_REPORT_AUTHORITY` note that final authority is
  the commit containing the report;
- the published state after the forward push;
- a `MIGRATION_CHAIN = PARTIAL -> READY_FOR_FINAL_PASS-FULL PROOF`
  resolution.

### Phase 4 — Publish

```text
git fetch origin
verify origin/main is ancestor of local main
git push origin main:main (no rebase, no force)
verify advertised == fetched == local main
```

No annotated tag is created (TAG_ALLOWED=false for this ACT).
No ClineMM checkout is touched.
No history rewrite is performed.

## Acceptance

Closed only when:

```text
1. implementation subject committed (S)
2. focused verification PASS on committed S
3. closure manifest produced (M)
4. closure manifest verified by `leamas factory close verify`
5. close report rendered (R)
6. M + R committed (C)
7. worktree clean (tracked files)
8. origin/main is ancestor of local main
9. ordinary forward push completes
10. advertised main == fetched main == local main
11. cheap published canaries PASS
12. MIGRATION_CHAIN = PARTIAL -> PASS-FULL documented
```

## CLOSE_REPORT_AUTHORITY doctrine

The committed close report's `FINAL_HEAD` references the commit
containing the report itself (or a later commit). Final authority
is measured AFTER the report commit lands. This is the same
doctrine the PARTIAL report establishes in its "Close Report
Authority Note".

## No B3. No tag. No lifecycle commit. No ClineMM changes.

This ACT does not create a B3 ACT, a tag, a lifecycle commit, or
touch the ClineMM checkout. It only closes the portability chain
on `main`.

## Expected final status

```text
STATUS                  = PASS
TARGETED_PORTABILITY_FAILURES = 0
NEW_FAILURES            = 0
UNKNOWN_FAILURES        = 0
MIGRATION_CHAIN         = PARTIAL -> PASS-FULL
LOCAL_MAIN              = advertised == fetched == local main
PUBLISHED_CANARIES      = PASS