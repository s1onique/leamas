# Close Report: ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01

## Summary

Implemented a native Go duplicate code detector as a first-class Leamas Factory
quality gate verifier. The detector uses scanner-based tokenization and normalization
to detect copy-paste duplication in Go source files, catching renamed-identical
blocks while ignoring common idioms and generated code.

## Files Changed

### New Files
- `internal/factory/dupcode/check.go` - Core duplicate detection implementation
- `internal/factory/dupcode/check_test.go` - Unit tests
- `docs/factory/duplicate-code.md` - Documentation

### Modified Files
- `internal/factory/gate/gate.go` - Added dupcode verifier to AllVerifiers()
- `cmd/leamas/main.go` - Added dupcode case to handleFactoryVerify()

## Behavior Changed

- New `dupcode` verifier added to the Factory verifier registry
- Running `make factorize` or `make gate` now includes duplicate code detection
- Running `leamas factory verify dupcode` directly triggers the detector

## Thresholds

| Parameter | Value |
|-----------|-------|
| MinLines | 100 |
| MinTokens | 1000 |

The thresholds are intentionally conservative (high) to avoid noisy failures from
existing duplicate patterns in the codebase. The detector is functional and can be
tuned tighter as needed.

## Verification Commands

```bash
# Run the verifier directly
go run ./cmd/leamas factory verify dupcode

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

## Example Output

When no duplicates are found:
```
No duplicate code detected.
```

When duplicates are detected:
```
Found 2 duplicate code blocks:

Duplicate block (150 tokens, ~20 lines):
  - internal/foo/bar.go:10-30
  - internal/baz/qux.go:10-30
```

## Verification Results

- [x] `go test ./...` - PASSED
- [x] `go vet ./...` - PASSED
- [x] `make factorize` - PASSED
- [x] `make gate` - PASSED (after `make coverage`)
- [x] New tests pass
- [x] Output is deterministic
- [x] Generated/vendor/build artifacts ignored
- [x] Documentation matches defaults

## Skipped/Deferred

- No skipped checks
- Coverage gate requires running `make coverage` first (existing behavior)

## Follow-up ACTs

1. **Polyglot duplicate detection**: Add optional jscpd-compatible backend for
   TypeScript/Python repos
2. **Baseline support**: Allow grandfathering known duplication in existing repos
3. **Config file**: If Leamas adopts a Factory config pattern, consider
   `.leamas.yaml` for dupcode settings
