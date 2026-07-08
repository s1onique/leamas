# Close Report: ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01

## ACT Reference

ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01

## Summary

Original ACT added anchor loading/rendering helpers. R1 wired anchors into generated digest output and corrected the behavior claim.

## Files Changed

| File | Change |
|------|--------|
| internal/factory/digest/anchors.go | Added LoadAnchors and RenderAnchors functions with simple TOML parsing |
| internal/factory/digest/anchors_test.go | Package-level tests for anchor loading and rendering |
| internal/factory/digest/digest_anchors_integration_test.go | Integration tests proving anchors appear in digest output |
| internal/factory/digest/digest.go | R1: Wired LoadAnchors into RenderDigest |
| internal/factory/digest/range.go | R1: Wired LoadAnchors into RenderRangeDigestWithResolved and RenderDigestWithResolved |
| docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01.md | This file (original close report, updated) |
| docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1.md | R1 close report |

## Behavior Changed

### Original ACT Behavior

The original ACT added `LoadAnchors()` and `RenderAnchors()` functions in
`internal/factory/digest/anchors.go`. However, prior review found these
functions were not wired into the actual digest generation code, so configured
anchors were not appearing in generated digests.

### R1 Correction

R1 wired the anchor functions into all three digest render functions:
- `RenderDigest()` - for dirty/staged mode
- `RenderRangeDigestWithResolved()` - for range mode with resolved info
- `RenderDigestWithResolved()` - for dirty mode with resolved info

Now when `.leamas/anchors.toml` exists, configured anchors appear as a markdown table in the digest output. When the file doesn't exist, the digest correctly shows "No workflow anchors configured."

## Verification

### Commands Run

```bash
go test ./internal/factory/digest/... -v
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory digest --output /tmp/leamas-anchors-check.md
make factorize
make gate
go test ./...
go vet ./...
```

### Results

- [x] Tests pass
- [x] Quality gate passes
- [x] Digest output includes configured anchors when anchors.toml exists
- [x] Digest output shows "No workflow anchors configured." when anchors.toml is absent

## Decisions Made

- Used existing simple TOML-ish parser (no third-party dependencies)
- Anchors pass through the same redaction boundary as other digest content
- Missing anchor file is not an error (returns nil config)

## Agent Doctrine Impact

None - this ACT does not change agent-facing doctrine or verifier behavior.

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| | | |

## Notes

The anchor config path is `.leamas/anchors.toml` relative to the repository root. Example format:

```toml
[[anchors]]
id = "EPIC-LEAMAS-FACTORY-HARDENING-AND-NEXT-BOOTSTRAP"
type = "epic"
summary = "Factory hardening and next bootstrap"
url = "docs/epics/EPIC-LEAMAS-FACTORY-HARDENING-AND-NEXT-BOOTSTRAP.md"

[[anchors]]
id = "ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1"
type = "act"
summary = "Wire workflow anchors into digest output"
url = "docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1.md"
```
