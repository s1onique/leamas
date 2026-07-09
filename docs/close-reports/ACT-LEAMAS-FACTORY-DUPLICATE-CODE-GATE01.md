# Close Report: ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01

## Summary

Implemented a native Go duplicate code detector with baseline+ratchet pattern for
Leamas Factory quality gate. The detector uses scanner-based tokenization and
normalization to detect copy-paste duplication, with path+count comparison to avoid
false positives from line-number shifts.

## Files Changed

### New Files
- `internal/factory/dupcode/baseline.go` - Baseline types, LoadBaseline, WriteBaseline, CompareToBaseline
- `internal/factory/dupcode/baseline_test.go` - Baseline unit tests
- `cmd/leamas/factory_verify_dupcode.go` - CLI handler with --baseline, --update-baseline, --json flags
- `.factory/dupcode-baseline.json` - Committed baseline (681 findings at 40/400 thresholds)

### Modified Files
- `internal/factory/dupcode/check.go` - Added StableFingerprint field, DefaultConfig returns 40/400
- `internal/factory/dupcode/check_test.go` - Updated test expectations for 40/400 thresholds
- `internal/factory/gate/gate.go` - Refactored to use dupcode_verifier.go
- `internal/factory/gate/dupcode_verifier.go` - Moved dupcode verifier logic
- `internal/factory/llmfriendly/check.go` - Added baseline file to ignore list
- `cmd/leamas/main.go` - Added dupcode case handler
- `docs/factory/duplicate-code.md` - Updated documentation
- `.gitignore` - Added negation for `.factory/dupcode-baseline.json`

## Behavior Changed

- Baseline+ratchet model: only NEW or WORSENED duplication fails the gate
- Thresholds lowered from 100/1000 to 40/400
- CLI supports `--baseline`, `--update-baseline`, `--json`, `--min-lines`, `--min-tokens`
- Baseline file is tracked in git for CI-safety

## Thresholds

| Parameter | Value |
|-----------|-------|
| MinLines | 40 |
| MinTokens | 400 |

Policy validation: LoadBaseline validates thresholds match policy (40/400).

## R1 Review Fixes

1. **Baseline file tracked**: Added to .gitignore with negation pattern
2. **Path normalization**: All paths normalized to repo-relative with forward slashes
3. **Deterministic timestamps**: BaselineWriter allows test injection
4. **Path+count comparison**: Avoids false positives from line shifts
5. **CLI error handling**: Flag parse errors are properly reported
6. **JSON output**: Uses encoding/json instead of hand-built strings
7. **LLM-friendly**: Baseline file added to ignore list

## Verification Commands

```bash
# Run the verifier directly
go run ./cmd/leamas factory verify dupcode

# Update baseline
go run ./cmd/leamas factory verify dupcode --update-baseline

# Run factorize (includes dupcode)
make factorize

# Run gate (includes dupcode)
make gate

# Run tests
go test ./...

# Run vet
go vet ./...

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

## Verification Results

- [x] `go test ./...` - PASSED
- [x] `go vet ./...` - PASSED
- [x] `make factorize` - PASSED
- [x] `make gate` - PASSED
- [x] `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` - PASSED
- [x] New tests pass (21 tests in dupcode package)
- [x] Deterministic output verified
- [x] Baseline file committed to git
- [x] Documentation updated

## Skipped/Deferred

- No skipped checks
- Coverage gate requires running `make coverage` first (existing behavior)

## Follow-up ACTs

1. **Polyglot duplicate detection**: Add optional jscpd-compatible backend for
   TypeScript/Python repos (deferred, not in scope)
2. **Config file**: If Leamas adopts a Factory config pattern, consider
   `.leamas.yaml` for dupcode settings (deferred)
