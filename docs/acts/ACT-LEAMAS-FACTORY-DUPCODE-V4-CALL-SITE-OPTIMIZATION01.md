# ACT-LEAMAS-FACTORY-DUPCODE-V4-CALL-SITE-OPTIMIZATION01

## Status: CLOSED

## Objective
Optimize dupcode v4 `normalizeFingerprint` function identified as a hot spot in `findCommonWindows` call site.

## Files Changed
- `internal/factory/dupcode/v4_legacy_helpers.go`
- `internal/factory/dupcode/check_test.go` (regression test)

## Behavior Changed
- `normalizeFingerprint` now uses `strings.Builder` with heuristic pre-allocated capacity instead of dynamic `bytes.Buffer`
- Retained original token encoding behavior: only IDENT, STRING/CHAR, and numeric literals are normalized; all other tokens use `tok.String()` for standard library representation

## Exact Commands Run
```bash
# Build verification
go build -trimpath -o bin/leamas ./cmd/leamas

# Targeted test
go test -short -run TestNormalize ./internal/factory/dupcode -v -count=1

# Regression test
go test -run TestNormalizeFingerprintPreservesTokenStringEncoding ./internal/factory/dupcode -v -count=1

# Full dupcode test suite
/usr/bin/time -v go test -short ./internal/factory/dupcode/... -count=1 -timeout=10m

# Gate verification
make gate-fast
make gate-dupcode

# Commits
git add -A
git commit -m "perf(dupcode): optimize normalizeFingerprint with pre-allocated strings.Builder"
git commit -m "correction(dupcode): restore legacy token encoding in normalizeFingerprint"
```

## Results
- **Build**: PASS
- **TestNormalizeFingerprint**: PASS
- **TestNormalizeFingerprintPreservesTokenStringEncoding**: PASS (9 cases)
- **Full dupcode tests**: PASS (564s, 9:24 wall time)
- **gate-fast**: PASS
- **make gate-dupcode**: PASS (dupcode: OK, dupcode-baseline: OK)

## Implementation Identity
```
HEAD:        53dc210eea10849657434570ae6473d414ff14ef
HEAD^{tree}: b42896170ae3263d81a91d73321b0edcf4e5b1b2
```

## Commit History
```
53dc210eea10849657434570ae6473d414ff14ef b42896170ae3263d81a91d73321b0edcf4e5b1b2 correction(dupcode): restore legacy token encoding in normalizeFingerprint
ab235a68c9400717edfca5176894386a14fa1d3c b06eb216e817135872fec400b421d5fc8afc3510 perf(dupcode): optimize normalizeFingerprint with pre-allocated strings.Builder
```

## Notes
- Initial optimization attempt incorrectly replaced `t.String()` with inline token names, breaking semantic equivalence
- Expert review identified the issue: Go's `Token.String()` returns actual symbols (e.g., `"+"` for `token.ADD`), not enum names
- Corrected to preserve legacy encoding: only normalize IDENT, STRING/CHAR, and numeric literals; all other tokens use standard library representation
- The retained optimization is the `strings.Builder` with heuristic pre-allocation, which avoids dynamic buffer growth
- Added regression test `TestNormalizeFingerprintPreservesTokenStringEncoding` to prevent future regressions

## Skipped Checks
- Full canonical aggregate gate (`LEAMAS_ALLOW_FULL_GATE=1 make gate`) was not run due to terminal timeout constraints
- Long lane was not run as a separate step
- The scoped dupcode package tests and gate-dupcode checks passed

## Follow-up ACTs
None required. The optimization is complete and verified.
