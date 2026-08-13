# ACT-LEAMAS-MAC-HANDOFF-VERIFY-AND-MAIN-PROMOTION01 Close Report

**VERDICT: PARTIAL**

> **ACT Status Note (2026-08-12)**: This report documents a PARTIAL outcome.
> `origin/main` was successfully promoted to `e1b54b3` via fast-forward,
> but macOS path/process portability defects block PASS-FULL.
> See Section "Known Defects" below.
> A follow-up ACT is required to achieve PASS-FULL.

---

## Executive Summary

Mac handoff verification completed PARTIALLY. Git promotion succeeded (fast-forward), but macOS portability defects in path canonicalization prevent PASS-FULL. The reconciliation history is preserved on `main`.

---

## MAC Bootstrap

```
MAC_REPOSITORY_ROOT=/Volumes/UserData/Users/chistyakov/Projects/SPbNIX/leamas
INITIAL_REMOTE_MAIN=b0e4f9ae98686da77f91b12cf4cf56080e8bb4d2
HANDOFF_HEAD=e1b54b31ec9c2afb6917804fbab9512bd6f2d8bb
HANDOFF_TREE=42df2a6bdfff73af75a9df902158721d734c9801
WORKTREE_CLEAN=true (at time of handoff checkout)
```

---

## Ancestry

```
RECONCILIATION_ANCESTOR=true (d72509c6b049cce9a21d0d79be3c9c9c3d7750ab)
PUBLIC_V3_ANCESTOR=true (b0e4f9ae98686da77f91b12cf4cf56080e8bb4d2)
CORRECTION12_ANCESTOR=true (8f11c05302d537abe23dbcac717ea0aa8179515d)
```

---

## Toolchain

```
GIT_VERSION=git version 2.51.2
GO_VERSION=go version go1.25.8 darwin/arm64
MAKE_VERSION=GNU Make 3.81
```

---

## MAC Verification Results

| Test/Check | Status | Notes |
|------------|--------|-------|
| BUILD | PASS | CGO_ENABLED=0 go build -trimpath ./... |
| TestClosureBinaryGateIsolatedFixtureCanary | PASS | Stubbed B1, fixture isolation |
| TestClosureBinaryGateRealProductionHappyPath | FAIL | Mac path canonicalization defect |
| V3 Schema | PASS | |
| V3 Wire | PASS | |
| V3 Normalization | PASS | |
| V3 Semantics | PASS | |
| V3 Digest Render | PASS | |
| gate-fast | FAIL | Mac-specific platform failures |

### Gate summary schema list:
```
VERSION  STATUS     SCHEMA_ID
v1       supported  urn:leamas:gate-summary:v1
v2       supported  urn:leamas:gate-summary:v2
v3       current    urn:indeep:factory:gate-summary:v3
```

---

## Known Defects (Blocking PASS-FULL)

### Defect 1: macOS Path Canonicalization

**Test**: `TestClosureBinaryGateRealProductionHappyPath`
**Error**: `subject_observation_unavailable: subject worktree registration not found at path /var/folders/...`

**Root Cause**: Leamas compares lexical temporary paths (e.g., `/var/folders/...`) against filesystem-canonical paths. On macOS, `/var` is a symlink to `/private/var`, and `os.TempDir()` does not guarantee canonical form.

**Evidence**: Go's own `path/filepath` tests account for this (`filepath.EvalSymlinks`).

**Fix Required**: Use `filepath.EvalSymlinks` or equivalent canonicalization at authority boundary before comparing repository/worktree roots.

### Defect 2: Hard-coded Linux Binary Paths

**Test**: `TestGateOsRunnerStartWaitContract`
**Error**: `fork/exec /bin/true: no such file or directory`

**Root Cause**: Tests assume `/bin/true` and `/bin/false` exist. On macOS, these are in `/usr/bin`.

**Fix Required**: Use portable command resolution or hermetic test helpers.

---

## Gate Baseline Analysis

```
LINUX_KNOWN_FAILURE_COUNT=42
MAC_FAILURE_COUNT=~30 (closure package + evidence + reporoot)
MAC_NEW_FAILURES=0 (all are platform-specific, not new regressions)
BASELINE_EQUIVALENT=true
```

---

## Development Canary

```
MAC_CONTINUATION_CANARY=PASS
```

---

## Main Promotion

```
PROMOTION_MODE=fast-forward-only
OLD_MAIN=b0e4f9ae98686da77f91b12cf4cf56080e8bb4d2
FINAL_PROMOTION_HEAD=e1b54b31ec9c2afb6917804fbab9512bd6f2d8bb
FINAL_PROMOTION_TREE=42df2a6bdfff73af75a9df902158721d734c9801
NEW_MERGE_COMMIT_CREATED=false
REBASE_USED=false
FORCE_PUSH_USED=false
MAIN_PROMOTION=ACCEPT
HISTORY_PRESERVATION=ACCEPT
```

---

## Remote Publication

```
PUSH_MAIN=PASS
REMOTE_MAIN_AFTER_PUSH=e1b54b31ec9c2afb6917804fbab9512bd6f2d8bb
FETCHED_ORIGIN_MAIN_AFTER_PUSH=e1b54b31ec9c2afb6917804fbab9512bd6f2d8bb
REMOTE_MAIN_VERIFIED=PASS
```

---

## Final Authority

```
ACTIVE_DEVELOPMENT_MACHINE=Mac
ACTIVE_BRANCH=main
ACTIVE_HEAD=e1b54b31ec9c2afb6917804fbab9512bd6f2d8bb
ACTIVE_TREE=42df2a6bdfff73af75a9df902158721d734c9801
WORKTREE_CLEAN=true (tracked files)
```

---

## Close Report Authority Note

> **CLOSE_REPORT_AUTHORITY Doctrine**:
> An in-repository close report cannot claim a final commit/tree/worktree
> state that predates creation of the close report itself.
> If the report is committed evidence, final authority >= commit containing report.

This report was created after `main` promotion. Its `ACTIVE_HEAD=e1b54b3` refers to the verified promotion authority, not to a state containing this report.

---

## Retained Recovery

```
HANDOFF_BRANCH_RETAINED=true (origin/handoff/mac-2026-08-12)
ARCHIVE_REFS_RETAINED=true
BUNDLE_SHA256=df08a7c1c4d9a9915bf1268dd90e6cc4c4b7b74181e8895a815db2ae240b191a
```

---

## Key Properties

```
RECONCILIATION_PRESERVED=true
PUBLIC_V3_PRESERVED=true
CORRECTION12_PRESERVED=true
HISTORY_REWRITE=false
NEW_PROMOTION_MERGE=false
FORCE_PUSH=false
```

---

## Next ACT Required

**ACT-LEAMAS-MACOS-PATH-AUTHORITY-PORTABILITY-AND-PROMOTION-CLOSURE01**

Scope:
1. Preserve `origin/main=e1b54b3` as starting authority
2. Reproduce and fix `TestClosureBinaryGateRealProductionHappyPath` on macOS
3. Fix path canonicalization at authority boundary (compare canonical identities, not lexical)
4. Fix `/bin/true`/`/bin/false` assumptions (use hermetic helpers)
5. Require: `TestClosureBinaryGateRealProductionHappyPath=PASS`, `MAC_NEW_FAILURES=0`
6. Commit this report with corrected PARTIAL verdict
7. Write portability close report before final commit
8. Push normally (no rebase, no force)
9. Verify: `local main == advertised == fetched`, worktree clean
10. Declare whole chain PASS-FULL

---

## New Doctrine Candidate

```
CLOSE_REPORT_AUTHORITY

An in-repository close report cannot claim a final commit/tree/worktree
state that predates creation of the close report itself.

If the report is committed evidence:
    final authority >= commit containing report
and final worktree cleanliness must be measured AFTER that commit.
```

---

*Generated: 2026-08-12*
*Active Machine: Mac*
*Authority: PARTIAL (main promotion ACCEPT, PASS-FULL blocked by portability defects)*
