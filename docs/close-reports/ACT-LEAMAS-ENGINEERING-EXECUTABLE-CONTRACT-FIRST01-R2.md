# Close Report: ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01-R2

## ACT Title
Executable Contract First R2 Blocker Fixes

## Parent Epic
ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01

## Problem
Go security reviewer flagged R2 blockers that R1 left unresolved:

1. Path confinement was a directory-wide symlink scanner (false positives),
   not root-confined opening. Applied only to one of five files. Did not
   stop on ECF010 - still read the escaped target.
2. Factorization wiring test still called the verifier directly.
3. Empty files (canonical, projection, ACT template) failed open.
4. Export minimization incomplete: 15 exported symbols reported.
5. Bounds not applied to canonical doctrine.
6. Deterministic ordering tested only path order, not path→code→message.
7. Fixture helpers ignored setup errors.
8. Heading tests not orthogonal.
9. Doctrine corrections outstanding (go test label, dependency updates
   as refactoring, timeout exit 124 as universal contract, ACT template
   exceptions not requiring category/justification/verification).

## Goal
Address every R2 blocker with proper fail-closed semantics and orthogonal
tests, regenerate evidence.

## Scope
All R2 blockers identified by the reviewer.

## Non-goals
Refactors not required to address R2 findings.

## Executable contract

### Stable boundary
`CheckExecutableContractFirst(root string) []checks.Finding` is the only
exported ECF symbol. All diagnostic codes, path constants, and helpers
are unexported. `gate.AllVerifiers()` is the only external entry point
that references it.

### Test matrix

| Case | Dimension | Given | When | Expected |
|------|-----------|-------|------|----------|
| PathEscape per file | ECF010 | Each of 5 files is a symlink to /tmp | CheckExecutableContractFirst | ECF010 on the configured path |
| Empty canonical | ECF002 | Agent instruction file exists but empty | CheckExecutableContractFirst | ECF002 |
| Empty projection | ECF004 | AGENTS.md exists but empty | CheckExecutableContractFirst | ECF004 |
| Empty ACT | ECF008 | act.md exists but empty | CheckExecutableContractFirst | ECF008 |
| Wiring | AllVerifiers | Drift in AGENTS.md | gate.AllVerifiers()["executable-contract-first"] | ECF007 on AGENTS.md |
| Ordering | path→code→message | Multi-violation repo | CheckExecutableContractFirst | findings match sort contract |

### RED evidence
- Command: `go test ./internal/factory/doctrine/...`
- Expected failing case (before R2 fixes):
  - `TestCheckECF_PathEscape_AGENTS` (no real symlink escape)
  - `TestCheckECF_EmptyCanonicalInstruction` (empty file accepted)
  - `TestCheckECF_UnreadableFile` (ECF010 instead of ECF011)
  - `TestCheckECF_DeterministicOrdering_MultiViolations` (path-only sort)
- Observed reason: implementation lacked root-confined reading,
  empty-file rejection, and orthogonal ordering proof.
- Evidence: pre-R2 test logs showed multiple FAIL/0-finding mismatches.

### GREEN evidence
- Focused command: `go test ./internal/factory/doctrine/... ./internal/factory/gate/...`
- Affected subsystem command: `go test ./...`
- Repository gate command: `make gate`
- Result: *** GATE PASSED ***

### Exceptions
None.

## Acceptance Criteria
- [x] R2.1: Root-confined opening for all 5 files using os.Lstat +
  os.OpenInRoot (symlink) / os.Open (regular); stops on ECF010.
- [x] R2.2: `internal/factory/gate/ecf_wiring_test.go` proves
  AllVerifiers() wiring and drift detection through the gate.
- [x] R2.3: Empty canonical -> ECF002, empty projection -> ECF004,
  empty ACT -> ECF008.
- [x] R2.4: Only `CheckExecutableContractFirst` exported from ECF.
- [x] R2.5: Bounds applied to doctrine (read + bounded), not just FileExists.
- [x] R2.6: Ordering test verifies path→code→message tuple contract.
- [x] R2.7: mustMkdirAll / mustWriteFile / mustRemove / mustSymlink /
  mustChmod / mustReadFile all t.Fatal on setup error.
- [x] R2.8: Each heading test asserts that the ECF008 message names the
  specific heading.
- [x] R2.9: Doctrine corrections applied to executable-contract-first.md
  and docs/templates/act.md.
- [x] R2.10: `make gate` output listed `executable-contract-first: OK`
  (live gate run; no separate gate-summary artifact was generated).
- [x] R2.11: `make factorize` and `make gate` both pass.

## Verification Commands
```bash
make factorize
make gate
```

## Reviewer Focus
- `readECFConfined` distinguishes symlink vs regular file before opening
  so ECF010 vs ECF011 is correctly classified.
- `gate.ecf_wiring_test.go` exercises the wired verifier through
  AllVerifiers() (not a direct call into doctrine).
- Deterministic ordering test uses `sort.SliceStable` on the full
  (path, code, message) tuple.
- Fixture helpers all t.Fatal on setup failure - no silent os.ErrNotExist
  swallowing.

## Files Changed
- `internal/factory/doctrine/executable_contract_first.go` - rewritten with
  os.Lstat classification, os.OpenInRoot for symlinks, fail-closed bounds,
  unexported constants, empty-content rejection.
- `internal/factory/doctrine/executable_contract_first_helpers.go` - kept.
- `internal/factory/doctrine/executable_contract_first_fixtures_test.go` -
  new file with must-helpers that fail on setup errors.
- `internal/factory/doctrine/executable_contract_first_test.go` - split,
  smaller file with diagnostic coverage + ordering tests.
- `internal/factory/doctrine/executable_contract_first_symlink_test.go` -
  new file with per-file path-escape tests.
- `internal/factory/doctrine/executable_contract_first_act_test.go` -
  rewritten with orthogonal heading tests.
- `internal/factory/doctrine/executable_contract_first_helpers_test.go` -
  minimal helper verification tests.
- `internal/factory/gate/ecf_wiring_test.go` - new wiring integration test.
- `docs/doctrine/executable-contract-first.md` - dependency updates
  removed from refactor list; verification hooks clarified; exit 124
  marked as GNU convention not universal.
- `docs/templates/act.md` - exceptions now require category /
  justification / verification / why-no-regression-test table.

## Behavior Changed
- ECF010 now rejects each of the 5 configured files when symlinked
  outside the repository root. ECF010 is fail-closed: target not read.
- ECF011 used for regular-file read failures (permission denied, I/O
  errors). Permission denied is no longer mis-classified as ECF010.
- Empty files (canonical, projection, ACT template) produce distinct
  diagnostic codes instead of silently passing.
- Doctrinal claim that "dependency updates" are refactors was removed.
- Doctrinal claim that exit code 124 is universal Leamas contract was
  marked as GNU convention; tests must assert their own exit contract.
- ACT template exceptions now require four fields per exception.

## Exact Commands Run
```bash
go test ./internal/factory/doctrine/... -v
go test ./internal/factory/gate/ -run "TestAllVerifiersIncludesECF|TestECFFactorizationWiring_DriftDetected" -v
make factorize
make gate
```

## Honest Results
- All 56+ unit tests in doctrine package pass (including new path
  escape, empty file, ordering, and orthogonal heading tests).
- Wiring test in gate package passes.
- `make factorize`: *** FACTORIZE PASSED ***
- `make gate`: *** GATE PASSED ***

## Skipped or Deferred Checks
None.

## Follow-up ACTs
None.