# Close Report: ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01-R5

## ACT Title
Executable Contract First R5 Blocker Fixes

## Parent Epic
ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01

## Problem
R5 review identified one remaining edge in the path confinement
classifier:

1. **Multi-hop symlink chains were misclassified**: The previous
   single-pass resolver only inspected the immediate textual target
   of a symlink, not the chain. A path like `AGENTS.md -> safe/next`,
   `safe/next -> /outside/missing.md` was not classified as ECF010 —
   `os.Root.Open` blocked the actual escape, but the verifier reported
   ECF011 (or a missing diagnostic) instead of the promised ECF010.

2. **`isNotExist` used string matching**: The fallback
   `strings.Contains(err.Error(), "no such file")` is platform- and
   wording-dependent.

3. **Five missing diagnostics when OpenRoot failed**: When the
   root could not be opened, the verifier emitted a per-file
   "missing" finding for each of the five configured files, which is
   misleading and noisy.

## Goal
Replace the single-pass symlink check with a bounded recursive
resolver, use `errors.Is` for not-exist detection, and emit a single
root-level ECF011 when the root itself is inaccessible.

## Scope
All R5 blockers identified by the reviewer.

## Non-goals
Refactors not required to address R5 findings.

## Executable contract

### Stable boundary
`CheckExecutableContractFirst(root) []checks.Finding` is the only
exported symbol. Internally, it acquires a persistent `os.Root` and
walks each configured file via that root using a bounded recursive
symlink resolver (`ecfConfinedByWalk`) that splices symlink targets
into the pending component queue and stops after
`ecfMaxSymlinkDepth` hops.

### Test matrix

| Case | Dimension | Given | When | Expected |
|------|-----------|-------|------|----------|
| Multi-hop final-symlink escape | ECF010 | AGENTS.md -> safe/next, safe/next -> /outside/missing.md | CheckExecutableContractFirst | exactly one ECF010 on AGENTS.md |
| Multi-hop parent-symlink escape | ECF010 | .github -> inside_dir, inside_dir -> /outside/missing-dir | CheckExecutableContractFirst | exactly one ECF010 on copilot-instructions.md |
| Two-hop chain in root | OK | AGENTS.md -> safe/next, safe/next -> ../real_agents.md | CheckExecutableContractFirst | no ECF010 / ECF011 |
| Symlink loop | ECF010/ECF011 | AGENTS.md -> AGENTS.md | CheckExecutableContractFirst | deterministic finding, no hang |
| OpenRoot failure | ECF011 | invalid root path | CheckExecutableContractFirst | exactly one ECF011 with root path |

### RED evidence
- Command: `go test ./internal/factory/doctrine/...`
- Expected failing case (before R5 fixes):
  - `TestCheckECF_PathEscape_Chained_FinalSymlink` (no ECF010 emitted)
  - `TestCheckECF_PathEscape_Chained_ParentSymlink` (no ECF010 emitted)
  - `TestCheckECF_PathEscape_Chained_InRoot` (false ECF010 emitted due to incorrect cursor computation)
  - `TestCheckECF_PathEscape_Chained_SymlinkLoop` (would hang without bound)
- Observed reason: implementation only walked the immediate target
  of each symlink. The lexical cursor was computed without the root
  prefix, so legitimate in-root chains that traversed `..` were
  mis-classified.
- Evidence: pre-R5 test logs showed ECF011/ECF003 instead of ECF010
  for multi-hop escapes.

### GREEN evidence
- Focused command: `go test ./internal/factory/doctrine/... ./internal/factory/gate/...`
- Affected subsystem command: `go test ./...`
- Repository gate command: `make gate`
- Result: *** GATE PASSED ***

### Exceptions
None.

## Acceptance Criteria
- [x] R5.1: `ecfConfinedByWalk` uses a pending-component queue and
  splices relative symlink targets into it so multi-hop chains are
  resolved correctly. Absolute targets are rejected. Bounded by
  `ecfMaxSymlinkDepth` (40) hops to guarantee termination. No use of
  `indexOf`.
- [x] R5.2: `isNotExist` now uses `errors.Is(err, fs.ErrNotExist) ||
  os.IsNotExist(err)` — no string fallback.
- [x] R5.3: When `os.OpenRoot(root)` fails the verifier emits a
  single root-level `ECF011` finding and skips per-file checks.
- [x] R5.4: New regression tests in
  `executable_contract_first_chain_test.go`:
  - `TestCheckECF_PathEscape_Chained_FinalSymlink` (multi-hop final)
  - `TestCheckECF_PathEscape_Chained_ParentSymlink` (multi-hop parent)
  - `TestCheckECF_PathEscape_Chained_InRoot` (in-root chain accepted)
  - `TestCheckECF_PathEscape_Chained_SymlinkLoop` (no hang)
- [x] R5.5: `go test ./...`, `make factorize`, `make gate` all pass.

## Verification Commands
```bash
go test ./internal/factory/doctrine/... ./internal/factory/gate/...
go test ./...
make factorize
make gate
```

## Reviewer Focus
- The pending-component queue in `ecfConfinedByWalk` is mutable; each
  iteration pops the next component and either advances the relative
  cursor (regular file/dir, "..") or splices the relative target's
  components in front (symlink). Absolute targets are rejected
  immediately. The lexical cursor is rebuilt as
  `absRoot + relCursor` so the containment check remains correct even
  after a symlink target that contains `..`.
- `indexOf` is no longer used; the queue's index is implicit in the
  slice operations.
- `os.OpenRoot` failure path emits one finding, not five.

## Files Changed
- `internal/factory/doctrine/executable_contract_first_pathconf.go` —
  replaced `isECFConfinementError` / `pathInsideRoot` walker with
  `ecfConfinedByWalk` (bounded recursive) and helpers
  `splitPathComponents`, `pathInsideRoot`.
- `internal/factory/doctrine/executable_contract_first.go` —
  `isNotExist` uses `errors.Is(err, fs.ErrNotExist) ||
  os.IsNotExist(err)`; root failure emits a single ECF011.
- `internal/factory/doctrine/executable_contract_first_chain_test.go` —
  new file with R5 chained symlink tests.
- `internal/factory/doctrine/executable_contract_first_symlink_test.go`
  — slimmed down (chained tests removed into the new file).
- `internal/factory/doctrine/executable_contract_first_test.go` —
  `TestCheckECF_CanonicalDoctrinMissing` now passes `tmpDir` as the
  root, not a nested path.

## Behavior Changed
- Multi-hop symlink chains ending outside the root now produce
  ECF010 instead of ECF011.
- In-root two-hop chains are accepted without false ECF010.
- Symlink loops terminate deterministically via the bounded
  resolver rather than hanging.
- Empty / invalid root produces a single root-level ECF011.

## Exact Commands Run
```bash
go test ./internal/factory/doctrine/... -v
go test ./internal/factory/gate/ -v
go test ./...
make factorize
make gate
```

## Honest Results
- All 90+ unit tests in doctrine package pass.
- All gate tests pass.
- `make factorize`: *** FACTORIZE PASSED ***
- `make gate`: *** GATE PASSED ***

## Skipped or Deferred Checks
None.

## Follow-up ACTs
None.