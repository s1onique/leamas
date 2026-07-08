# Close Report: ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01

## ACT Reference

**ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01**: Document forbidden-pattern scan boundary

## Summary

Made the forbidden-pattern verifier's scan boundary explicit, documented, and tested. The verifier now implements a clear contract for what is scanned and what is allowed, eliminating the implicit blind spot around `internal/factory/`.

**Note:** This ACT documented the scan boundary contract. ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01-R1 fixed an implementation bug where `scripts/` and `githooks/` were not actually being scanned despite being declared in the boundary.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/forbidden/check.go` | Refactored with explicit scan boundary, exported ScanDirs/ScanFiles/SkipPatterns/AllowedDirs |
| `internal/factory/forbidden/check_test.go` | Split to stay under LLM-friendly threshold (400 lines) |
| `internal/factory/forbidden/scan_test.go` | NEW - Boundary contract tests |
| `internal/factory/forbidden/integration_test.go` | NEW - Integration tests |
| `internal/factory/forbidden/database.go` | NEW - Extracted database import checking |
| `docs/factory/forbidden-patterns.md` | NEW - Scan boundary documentation |
| `docs/close-reports/ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01.md` | NEW - Close report |

## Behavior Changed

### Scan Boundary Contract

**SCAN:**
- `cmd/` - All production code in cmd (`.go` non-test files)
- `internal/` (except `internal/factory/`) - All non-factory internal code (`.go` non-test files)
- `scripts/` - Shell scripts (all text files)
- `githooks/` - Git hooks (all text files)
- `AGENTS.md` - Agent contract file
- `.clinerules/leamas.md` - Cline rules file

**ALLOW (forbidden-policy terms permitted):**
- `internal/factory/` - Factory verification code must reference forbidden terms
- `docs/doctrine/` - Doctrine documents discuss policy
- `docs/adr/` - Architecture decision records
- `docs/factory/` - Factory documentation
- `docs/close-reports/` - Close reports
- `*_test.go` - Test files
- `testdata/` - Test fixtures

## Known Issue

**R1 Required:** The implementation initially skipped all non-Go files in ScanDirs. This was fixed in ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01-R1.

## Verification

### Commands Run

```bash
go test ./internal/factory/forbidden/... -v
make factorize
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory verify forbidden-patterns
```

### Results

- [x] All tests pass (12 tests across 3 test files)
- [x] Factorize passes (all verifiers OK)
- [x] Binary builds successfully
- [x] Forbidden-pattern verifier passes
- [x] LLM-friendly checks pass (files split to stay under 400 lines each)

## Decisions Made

1. **Exported boundary constants**: ScanDirs, ScanFiles, AllowedDirs, SkipPatterns are now exported for test verification
2. **File splitting**: Split check_test.go into check_test.go, scan_test.go, integration_test.go to stay under LLM-friendly thresholds
3. **Extracted database.go**: Separated database import checking into dedicated file
4. **Documentation**: Created docs/factory/forbidden-patterns.md with full contract documentation

## Agent Doctrine Impact

None. This is Factory tooling improvement, not a product code change.

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| | | |

## Skipped/Deferred Checks

None.

## Notes

- Pattern matching uses substring matching with `|` alternation (not regex)
- The boundary is enforced both in code and via exported constants that tests verify
- Files are explicitly sized to pass LLM-friendliness gate (<400 lines each)
