# ACT-LEAMAS-FACTORY-DUPCODE-V4-CALL-SITE-OPTIMIZATION01

## Status: CLOSED

## Objective
Optimize dupcode v4 `normalizeFingerprint` function identified as a hot spot in `findCommonWindows` call site.

## Files Changed
- `internal/factory/dupcode/v4_legacy_helpers.go`

## Behavior Changed
- `normalizeFingerprint` now uses `strings.Builder` with heuristic pre-allocated capacity instead of dynamic `bytes.Buffer`
- Retained original token encoding behavior: only IDENT, STRING/CHAR, and numeric literals are normalized; all other tokens use `tok.String()` for standard library representation

## Exact Commands Run
```bash
# Build verification
go build -trimpath -o bin/leamas ./cmd/leamas

# Targeted test
go test -short -run TestNormalize ./internal/factory/dupcode -v -count=1

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
- **Full dupcode tests**: PASS (564s, 9:24 wall time)
- **gate-fast**: PASS
- **make gate-dupcode**: PASS (dupcode: OK, dupcode-baseline: OK)

## Implementation Identity
- Correction commit: `4404154ef61de5a8b60d4c1f9e92c6c5df0b2e7d`
- Tree: see git log

## Notes
- Initial optimization attempt incorrectly replaced `t.String()` with inline token names, breaking semantic equivalence
- Expert review identified the issue: Go's `Token.String()` returns actual symbols (e.g., `"+"` for `token.ADD`), not enum names
- Corrected to preserve legacy encoding: only normalize IDENT, STRING/CHAR, and numeric literals; all other tokens use standard library representation
- The retained optimization is the `strings.Builder` with heuristic pre-allocation, which avoids dynamic buffer growth

## Skipped Checks
None. All canonical checks passed.

## Follow-up ACTs
None required. The optimization is complete and verified.
