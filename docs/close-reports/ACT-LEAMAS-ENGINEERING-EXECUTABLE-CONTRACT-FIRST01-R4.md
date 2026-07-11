# Close Report: ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01-R4

## ACT Title
Executable Contract First R4 Blocker Fixes

## Parent Epic
ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01

## Problem
R4 review identified a remaining gap in the path confinement
classifier:

1. **Existence precheck bypassed `OpenInRoot`**: `readECFConfined`
   started with `checks.FileExists`, which uses `os.Stat`. `os.Stat`
   follows symlinks; for a dangling outside escape such as
   `AGENTS.md -> /outside/missing.md` the stat failed and the verifier
   emitted ECF003 ("file missing") without ever consulting `OpenInRoot`,
   so the escape was never classified as ECF010.

2. **`EvalSymlinks` failed on dangling targets**: The fallback
   `isECFConfinementError` used `filepath.EvalSymlinks`. When the target
   did not exist, `EvalSymlinks` returned an error and the classifier
   returned false — the escape would have been reported as ECF011
   instead of ECF010.

3. **Empty canonical doctrine accepted**: The verifier read and bounded
   the doctrine file but otherwise discarded it, so an empty doctrine
   passed silently.

4. **`normalizeECFContent` stripped all trailing newlines** with
   `TrimRight`, weakening exact projection drift detection. The
   original contract specifies removing one final newline only.

## Goal
Replace the stat-based detection with a persistent `os.Root` walker
that uses `Lstat` + `Readlink` for component-by-component symlink
classification (without requiring the target to exist), reject empty
canonical doctrine, and tighten the whitespace normalization.

## Scope
All R4 blockers identified by the reviewer.

## Non-goals
Refactors not required to address R4 findings.

## Executable contract

### Stable boundary
`CheckExecutableContractFirst(root) []checks.Finding` is the only
exported symbol. Internally, it acquires a persistent `os.Root` and
walks each configured file via that root. Symlink classification uses
`r.Lstat` + `r.Readlink` so dangling escapes are detected without
requiring the target to exist.

### Test matrix

| Case | Dimension | Given | When | Expected |
|------|-----------|-------|------|----------|
| Dangling outside final symlink | ECF010 | AGENTS.md -> /outside/missing.md | CheckExecutableContractFirst | exactly one ECF010 on AGENTS.md |
| Dangling outside parent symlink | ECF010 | .github -> /outside/missing-dir | CheckExecutableContractFirst | ECF010 on copilot |
| Dangling outside docs parent | ECF010 | docs/doctrine -> /outside/missing | CheckExecutableContractFirst | ECF010 on doctrine |
| Relative dangling outside | ECF010 | AGENTS.md -> ../../outside/missing.md | CheckExecutableContractFirst | ECF010 on AGENTS.md |
| In-root dangling relative | OK | AGENTS.md -> missing_target.md | CheckExecutableContractFirst | ECF003 or ECF011, not ECF010 |
| Unreadable in-root parent | ECF011 | chmod 0 .github | CheckExecutableContractFirst | ECF011, not ECF010 |
| Empty canonical doctrine | ECF001 | doctrine file empty | CheckExecutableContractFirst | ECF001 |

### RED evidence
- Command: `go test ./internal/factory/doctrine/...`
- Expected failing case (before R4 fixes):
  - `TestCheckECF_PathEscape_DanglingOutside_FinalSymlink` (ECF003 instead of ECF010)
  - `TestCheckECF_PathEscape_DanglingOutside_ParentSymlink` (no ECF010)
  - `TestCheckECF_PathEscape_DanglingOutside_ParentSymlink_Docs` (no ECF010)
  - `TestCheckECF_PathEscape_DanglingOutside_RelativeSymlink` (no ECF010)
- Observed reason: precheck used `os.Stat` which follows symlinks, so
  dangling outside escapes were mis-classified as missing. The
  classifier used `EvalSymlinks` which fails on dangling targets.
- Evidence: pre-R4 test logs showed precheck produced "missing"
  findings instead of confinement findings.

### GREEN evidence
- Focused command: `go test ./internal/factory/doctrine/... ./internal/factory/gate/...`
- Affected subsystem command: `go test ./...`
- Repository gate command: `make gate`
- Result: *** GATE PASSED ***

### Exceptions
None.

## Acceptance Criteria
- [x] R4.1: `os.OpenInRoot` is the first filesystem operation against
  each configured relative path; no `os.Stat` precheck.
- [x] R4.2: `ecfConfinedByWalk` uses `r.Lstat` + `r.Readlink` to detect
  symlink escapes without requiring the target to exist. Absolute
  symlink targets are rejected per the os.Root contract. Relative
  targets whose lexical resolution leaves the root are rejected.
- [x] R4.3: Dangling-escape regression tests added
  (`*_DanglingOutside_*`); in-root unreadable parent dir produces
  ECF011.
- [x] R4.4: Empty canonical doctrine produces ECF001.
- [x] R4.5: `normalizeECFContent` strips at most one trailing
  newline; other whitespace preserved for exact projection drift
  detection.
- [x] R4.6: `go test ./...`, `make factorize`, `make gate` all pass.

## Verification Commands
```bash
go test ./internal/factory/doctrine/... ./internal/factory/gate/...
go test ./...
make factorize
make gate
```

## Reviewer Focus
- The persistent `os.Root` is acquired once via `os.OpenRoot(root)` and
  reused for every configured file. Each `r.Lstat` and `r.Readlink`
  call is root-confined.
- Absolute symlink targets are rejected unconditionally, matching the
  os.Root contract: `Symbolic links must not be absolute.`
- Relative symlink targets are resolved against the parent's absolute
  path and checked lexically against the root.
- Empty canonical doctrine now triggers ECF001 ("canonical doctrine
  file is empty"), mirroring the canonical agent instruction
  behavior.
- `normalizeECFContent` strips at most one trailing newline.

## Files Changed
- `internal/factory/doctrine/executable_contract_first.go` — verifier
  uses persistent `os.Root`; precheck removed; empty doctrine check
  added.
- `internal/factory/doctrine/executable_contract_first_pathconf.go`
  — replaced with `ecfConfinedByWalk` using `r.Lstat`/`r.Readlink`.
- `internal/factory/doctrine/executable_contract_first_helpers.go`
  — `normalizeECFContent` now strips at most one trailing newline.
- `internal/factory/doctrine/executable_contract_first_fixtures_test.go`
  — adjusted marked-block fixture to match the one-trailing-newline
  contract.
- `internal/factory/doctrine/executable_contract_first_symlink_test.go`
  — added dangling outside regression tests and unreadable parent
  test.

## Behavior Changed
- Dangling external final-component symlinks now produce ECF010
  instead of ECF003.
- Dangling external parent-directory symlinks now produce ECF010
  instead of no finding at all.
- Relative dangling external symlinks now produce ECF010.
- Empty canonical doctrine produces ECF001.
- In-root unreadable parent directory produces ECF011, not ECF010.

## Exact Commands Run
```bash
go test ./internal/factory/doctrine/... -v
go test ./internal/factory/gate/ -v
go test ./...
make factorize
make gate
```

## Honest Results
- All 80+ unit tests in doctrine package pass, including the new
  dangling-escape regression tests.
- All gate tests pass.
- `make factorize`: *** FACTORIZE PASSED ***
- `make gate`: *** GATE PASSED ***

## Skipped or Deferred Checks
None.

## Follow-up ACTs
None.