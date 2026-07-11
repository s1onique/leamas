# Close Report: ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01-R3

## ACT Title
Executable Contract First R3 Blocker Fixes

## Parent Epic
ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01

## Problem
R3 review identified one critical gap and three quality issues:

1. **Critical**: Parent-directory symlink escapes remained possible. The
   previous implementation only checked the final path component via
   `os.Lstat`, so a regular file reached through a symlinked parent
   directory (e.g., `repo/.github -> /outside/.github` with a regular
   `copilot-instructions.md` inside) was read successfully.

2. **OpenInRoot over-classification**: Any error from `os.OpenInRoot`
   was treated as confinement (ECF010), including in-root unreadable
   targets and dangling symlinks that should be ECF011.

3. **Cascading diagnostics**: After a primary read failure (missing,
   escape, bounds, I/O) the marker check still ran on empty content,
   producing a second, weaker diagnostic alongside the primary one.

4. **Close-report claim**: The R2 close report claimed a gate summary
   was generated, but the digest reported `source_status=missing`.

## Goal
Replace the Lstat-based detection with root-confined opening for every
configured file (final and intermediate components), distinguish
confinement from ordinary read failures, suppress cascading diagnostics,
and re-baseline the evidence.

## Scope
All R3 blockers identified by the reviewer.

## Non-goals
Refactors not required to address R3 findings.

## Executable contract

### Stable boundary
`CheckExecutableContractFirst(root) []checks.Finding` is the only
exported symbol. Internally, `readECFConfined` opens each configured
file through `os.OpenInRoot`, which rejects a path whose final or any
intermediate component resolves outside the supplied root.

### Test matrix

| Case | Dimension | Given | When | Expected |
|------|-----------|-------|------|----------|
| Final symlink escape | ECF010 | AGENTS.md -> /tmp | CheckExecutableContractFirst | ECF010 on AGENTS.md |
| Parent-dir symlink escape | ECF010 | .github -> /outside, regular file inside | CheckExecutableContractFirst | ECF010 on copilot-instructions.md |
| Parent-dir docs | ECF010 | docs/doctrine -> /outside, regular file inside | CheckExecutableContractFirst | ECF010 on doctrine |
| Parent-dir templates | ECF010 | docs/templates -> /outside, regular file inside | CheckExecutableContractFirst | ECF010 on ACT |
| In-root readable symlink | OK | AGENTS.md -> regular file in root (relative) | CheckExecutableContractFirst | no ECF010/ECF011 |
| In-root unreadable symlink | ECF011 | AGENTS.md -> chmod 0 regular file in root | CheckExecutableContractFirst | ECF011, no ECF010 |
| Dangling in-root symlink | ECF011/003 | AGENTS.md -> non-existent target | CheckExecutableContractFirst | ECF011 or ECF003, no ECF010 |
| Empty canonical | ECF002 | Agent instruction exists but empty | CheckExecutableContractFirst | ECF002 |
| Empty projection | ECF004 | AGENTS.md exists but empty | CheckExecutableContractFirst | ECF004 |
| Empty ACT | ECF008 | act.md exists but empty | CheckExecutableContractFirst | ECF008 |
| No cascading | Single | Escape on AGENTS.md | CheckExecutableContractFirst | exactly one ECF010, no ECF004 |

### RED evidence
- Command: `go test ./internal/factory/doctrine/...`
- Expected failing case (before R3 fixes):
  - `TestCheckECF_PathEscape_ParentDir_GitHub` (parent-dir escape not detected)
  - `TestCheckECF_PathEscape_ParentDir_Docs` (parent-dir escape not detected)
  - `TestCheckECF_PathEscape_ParentDir_Templates` (parent-dir escape not detected)
  - `TestCheckECF_PathEscape_InRootReadableSymlink` (false ECF010)
  - `TestCheckECF_PathEscape_InRootUnreadableSymlink` (false ECF010)
  - `TestCheckECF_PathEscape_DanglingInRootSymlink` (false ECF010)
  - `TestCheckECF_EmptyCanonicalInstruction` (empty file not detected)
- Observed reason: implementation used `os.Lstat` on the final component
  only, did not classify errors from `os.OpenInRoot`, and ran cascading
  marker checks on empty content.
- Evidence: pre-R3 test logs showed multiple FAIL mismatches.

### GREEN evidence
- Focused command: `go test ./internal/factory/doctrine/... ./internal/factory/gate/...`
- Affected subsystem command: `go test ./...`
- Repository gate command: `make gate`
- Result: *** GATE PASSED ***

### Exceptions
None.

## Acceptance Criteria
- [x] R3.1: Every configured file opened via `os.OpenInRoot`; final AND
  intermediate symlink components that escape root produce ECF010.
- [x] R3.2: Parent-directory symlink escape tests pass for `.github`,
  `docs/doctrine`, and `docs/templates`.
- [x] R3.3: `isECFConfinementError` (now in
  `executable_contract_first_pathconf.go`) resolves root AND each
  component before comparing, so in-root unreadable / dangling symlinks
  are classified as ECF011.
- [x] R3.4: Cascading diagnostics suppressed; marker checks run only when
  `readStatus == readOK || readStatus == readEmpty`.
- [x] R3.5: R3 close report no longer claims a gate summary artifact was
  generated; only the live `make gate` output is reported.
- [x] R3.6: `go test ./...`, `make factorize`, and `make gate` all pass.

## Verification Commands
```bash
go test ./internal/factory/doctrine/... ./internal/factory/gate/...
go test ./...
make factorize
make gate
```

## Reviewer Focus
- `isECFConfinementError` resolves both root and each path component
  with `filepath.EvalSymlinks` and uses a `pathInsideRoot` lexical
  helper to be robust against platform symlinks (e.g. `/var/folders`
  on macOS).
- `readResult` exposes a `readStatus` (readOK / readMissing / readEmpty /
  readEscaped / readTooLarge / readIO). Marker and template checks
  only run for `readOK` or `readEmpty`, eliminating cascading
  diagnostics.
- In-root unreadable symlinks use **relative** symlink targets
  (`real_agents.md`) because Go's `os.OpenInRoot` only follows
  relative symlinks; absolute symlinks to in-root targets trigger
  "path escapes from parent" on Linux/macOS. The tests document this
  limitation rather than working around it.

## Files Changed
- `internal/factory/doctrine/executable_contract_first.go` — every
  configured file now opens through `os.OpenInRoot`; `readResult`
  carries a `readStatus`; canonical empty check broadened.
- `internal/factory/doctrine/executable_contract_first_pathconf.go` —
  new file with `isECFConfinementError` and `pathInsideRoot`.
- `internal/factory/doctrine/executable_contract_first_symlink_test.go`
  — added parent-dir escape tests and in-root symlink variants.
- `internal/factory/doctrine/executable_contract_first_fixtures_test.go`
  — `mustRemove` now uses `os.RemoveAll` so directories can be torn
  down.
- `docs/close-reports/ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01-R2.md`
  — R2 gate-summary claim removed.

## Behavior Changed
- ECF010 now rejects any configured file whose final component OR any
  intermediate directory component resolves outside the root. The
  detection is component-aware, not just final-component-aware.
- ECF011 is emitted for in-root unreadable / dangling symlinks,
  distinguished from ECF010 confinement violations.
- Missing, escaped, oversized, and I/O-failed reads produce exactly one
  primary diagnostic; cascading marker/template findings are no longer
  emitted after a primary failure.

## Exact Commands Run
```bash
go test ./internal/factory/doctrine/... ./internal/factory/gate/...
go test ./...
make factorize
make gate
```

## Honest Results
- All 70+ unit tests in doctrine package pass, including the new
  parent-dir escape and in-root symlink variant tests.
- All gate tests pass.
- `make factorize`: *** FACTORIZE PASSED ***
- `make gate`: *** GATE PASSED ***

## Skipped or Deferred Checks
None.

## Follow-up ACTs
None.