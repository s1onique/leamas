# Close Report: ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R2

## ACT Reference

ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R2-TRUTHFUL-ERRORS-AND-REDACTION-PROOF

## Summary

Fixed R1 overclaim about malformed anchor config handling and strengthened the anchor redaction integration test with explicit assertions.

## Option Chosen

**Option A - Implement honest malformed-config errors**

R1 claimed that "Malformed/unreadable anchor file returns an honest error" but the validation was not actually implemented. This R2 implements the validation.

## Malformed Config Handling

The lightweight TOML-ish parser now validates input and returns `ErrMalformedAnchors` for:

- Missing closing quotes in values (e.g., `id = "ACT-001`)
- Unknown keys in `[[anchors]]` blocks (e.g., `unknown = "field"`)
- Wrong section names (e.g., `[[wrong_section]]`)
- Content outside `[[anchors]]` blocks

Comments (lines starting with `#`) are allowed anywhere.

## Tests Added/Changed

### Package-level tests (anchors_test.go)

Added tests for malformed config handling:
- `TestLoadAnchors_MalformedMissingClosingQuote` - verifies missing closing quote returns error
- `TestLoadAnchors_MalformedUnknownKey` - verifies unknown keys return error
- `TestLoadAnchors_MalformedWrongSection` - verifies wrong section names return error
- `TestLoadAnchors_MalformedContentOutsideAnchorsBlock` - verifies content outside anchors block returns error
- `TestLoadAnchors_ValidWithComments` - verifies comments are allowed

### Integration test strengthened (digest_anchors_integration_test.go)

`TestDigestAnchors_WriteRedactsAnchorSecrets` now explicitly asserts:
1. **Anchor ID remains present** - digest contains `ACT-001`
2. **Anchor table remains present** - digest contains `| ID | Type | Summary | URL |`
3. **Fake secret is absent** - digest does NOT contain `sk-1234567890abcdefghijklmnop`
4. **Redaction marker is present** - digest contains `sk-[REDACTED]`

## Files Changed

| File | Change |
|------|--------|
| internal/factory/digest/anchors.go | Added `ErrMalformedAnchors` error, validation logic in `LoadAnchorsFrom()` |
| internal/factory/digest/anchors_test.go | Added 5 tests for malformed config handling |
| internal/factory/digest/digest_anchors_integration_test.go | Strengthened `TestDigestAnchors_WriteRedactsAnchorSecrets` with explicit assertions |
| docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1.md | Added note clarifying R1 overclaim |
| docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R2-TRUTHFUL-ERRORS-AND-REDACTION-PROOF.md | This file |

## Verification Commands

```bash
go test ./internal/factory/digest/... -v
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory digest --output /tmp/leamas-anchors-r2-check.md
make factorize
make gate
go test ./...
go vet ./...
```

## Verification Results

- [x] Digest package tests pass (including new malformed config tests)
- [x] All integration tests pass (including strengthened redaction test)
- [x] Binary builds successfully
- [x] Generated digest includes configured anchors
- [x] Missing anchors still render "No workflow anchors configured."
- [x] Malformed anchor config returns honest tested error
- [x] `make factorize` passes
- [x] `make gate` passes
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Behavior Changed

### Malformed Config Validation

The parser now validates:
1. Only `[[anchors]]` section is supported
2. Only `id`, `type`, `summary`, `url` keys are valid within anchors blocks
3. Values must have matching quotes
4. No non-comment content outside of `[[anchors]]` blocks

### R1 Overclaim Fixed

R1 claimed "Malformed/unreadable anchor file returns an honest error" but did not implement or test this behavior. R2 adds the implementation and tests.

## Agent Doctrine Impact

None - this R2 tightens truth/proof without changing agent-facing behavior.

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| | | |

## Notes

- Chose Option A because R1 close report already claimed honest malformed errors
- Validation is lightweight - no full TOML parser added
- Comments are allowed for future extensibility
