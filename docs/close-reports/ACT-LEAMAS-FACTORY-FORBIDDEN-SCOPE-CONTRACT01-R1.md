# Close Report: ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01-R1

## ACT Reference

**ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01-R1**: Fix forbidden-pattern scan boundary implementation

## Summary

Fixed the forbidden-pattern verifier to actually scan `scripts/` and `githooks/` directories. The original ACT documented these directories but the implementation skipped non-Go files.

## Problem

The original `CheckForbiddenPatterns` implementation had this line:

```go
if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
    return nil
}
```

This skipped ALL non-Go files, meaning shell scripts in `scripts/` and hooks in `githooks/` were never scanned despite being declared in the scan boundary contract.

## Fix Applied

Refactored `internal/factory/forbidden/check.go` with explicit file type handling:

- `isGoProductionFile(relPath)` - Returns true for Go non-test files in `cmd/` and `internal/` (except `internal/factory/`)
- `isTextPolicyFile(relPath)` - Returns true for all text files in `scripts/` and `githooks/`
- `shouldScanFile(relPath)` - Combines the above with `isInAllowedDir()` check

This ensures:
- `cmd/**` scans only `.go` non-test files
- `internal/**` scans only `.go` non-test files, except `internal/factory/**`
- `scripts/**` scans all text files
- `githooks/**` scans all text files

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/forbidden/check.go` | Added `shouldSkipDir`, `isGoProductionFile`, `isTextPolicyFile`, `shouldScanFile` functions |
| `internal/factory/forbidden/integration_test.go` | Added `TestScriptsForbiddenPatternDetected` and `TestGithooksForbiddenPatternDetected` regression tests |
| `docs/factory/forbidden-patterns.md` | Added File Types column to scan boundary table |
| `docs/close-reports/ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01.md` | Updated to reference R1 fix |
| `docs/close-reports/ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01-R1.md` | NEW - R1 close report |

## Tests Added

1. `TestScriptsForbiddenPatternDetected` - Regression test ensuring forbidden patterns in `scripts/bad.sh` produce findings
2. `TestGithooksForbiddenPatternDetected` - Regression test ensuring forbidden patterns in `githooks/pre-push` produce findings

## Verification

### Commands Run

```bash
go test ./internal/factory/forbidden/... -v
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory verify forbidden-patterns
make verify-forbidden
make factorize
make gate
go test ./...
go vet ./...
```

### Results

- [x] All tests pass
- [x] Factorize passes
- [x] Binary builds successfully
- [x] Forbidden-pattern verifier passes
- [x] LLM-friendly checks pass
- [x] make gate passes
- [x] go test ./... passes
- [x] go vet ./... passes

## Decisions Made

1. **File type separation**: Explicitly separate Go production files from text policy files
2. **Cross-platform paths**: Handle both forward slash and backslash for Windows compatibility
3. **Minimal refactor**: Keep changes scoped to the bug fix, not a broader rewrite
4. **AGENTS/.clinerules policy**: AGENTS.md and .clinerules/ are policy documents, not production code. They may mention forbidden terms. Agent context enforcement is handled by the agent-context verifier and LLM-friendliness gate.

## Agent Doctrine Impact

None. This is Factory tooling improvement, fixing a bug in verification code.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| ACT-LEAMAS-FACTORY-SECRET-REDACTION01 | Secret redaction wiring | Candidate |
| ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01 | Digest anchors wiring | Candidate |
| ACT-LEAMAS-FACTORY-REMOTE-BRANCH-PROTECTION01 | Branch protection operational proof | Candidate |

## Skipped/Deferred Checks

None.
