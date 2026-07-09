# ACT: PUBLIC_SURFACE_DELTA Section for Digest v2

**ACT ID**: ACT-LEAMAS-FACTORY-DIGEST-PUBLIC-SURFACE-DELTA01

**Status**: Closed

## Summary

Implemented `PUBLIC_SURFACE_DELTA` section for digest v2. This deterministic Go exported-symbol delta section answers "Did this patch change the public Go API/CLI surface a reviewer should inspect?"

## Files Changed

### New Files
- `internal/factory/digest/public_surface_delta.go` - Core implementation (291 lines)
- `internal/factory/digest/public_surface_types.go` - Type definitions (43 lines)
- `internal/factory/digest/public_surface_parse.go` - AST parsing functions (183 lines)
- `internal/factory/digest/public_surface_delta_integration_test.go` - Integration tests (230 lines)
- `internal/factory/digest/public_surface_delta_parse_test.go` - Unit tests (240 lines)
- `docs/factory/digest-public-surface-delta.md` - Section specification

### Modified Files
- `internal/factory/digest/evidence_hashes.go` - Added `PublicSurfaceDeltaSHA256` field
- `internal/factory/digest/digest.go` - Integrated PUBLIC_SURFACE_DELTA computation
- `internal/factory/digest/range.go` - Integrated PUBLIC_SURFACE_DELTA in range mode
- `docs/factory/digest-contract.md` - Updated section order documentation

## Behavior Changed

### Digest Output
The digest now includes a `PUBLIC_SURFACE_DELTA` section after `GATE_SUMMARY` and before `## Changed files`. The section contains:

```markdown
## PUBLIC_SURFACE_DELTA

language=go
source_status=present
packages_changed=<count>
symbols_added=<count>
symbols_removed=<count>
symbols_modified=<count>
cli_commands_changed=<count>

packages:
  - <package1>
  - <package2>

added:
  - pkg/path.Symbol

removed:
  - ...

modified:
  - ...

cli_commands:
  - ...
```

### Evidence Hashes
The `EVIDENCE_HASHES` section now includes a `public_surface_delta_sha256` field for reproducibility verification.

## Commands Run

```bash
# Tests
go test ./internal/factory/digest/... -run "PublicSurface" -v
go test ./internal/factory/digest/...

# Formatting
gofmt -w internal/factory/digest/public_surface_delta.go
gofmt -w internal/factory/digest/public_surface_delta_parse_test.go

# Verification
make factorize
make gate
```

## Results

| Check | Status |
|-------|--------|
| `make factorize` | PASSED |
| `make gate` | PASSED |
| `go test ./internal/factory/digest/...` | PASSED |
| `go vet ./...` | PASSED |
| Static build | PASSED |
| LLM-friendly gate | PASSED |

## Symbol Types Tracked

- **func**: Standalone exported functions
- **type**: Exported type definitions (struct, interface, alias)
- **const**: Exported constant declarations
- **var**: Exported variable declarations
- **method**: Methods on exported types
- **field**: Exported fields in exported struct types
- **interface_method**: Methods defined in exported interfaces

## Skipped/Deferred

- None

## Follow-up ACTs

- None required

## Notes

- The implementation uses Go AST parsing (`go/ast`, `go/parser`) for reliable symbol detection
- CLI command detection uses regex patterns for Cobra command definitions
- All files are LLM-friendly (under 400 lines each)
- Deterministic output: symbols sorted alphabetically, counts computed from sorted sets
