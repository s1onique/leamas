# Close Report: ACT-LEAMAS-FACTORY-DIGEST-VERSIONED-CONTRACT01

## ACT Reference

**ACT-LEAMAS-FACTORY-DIGEST-VERSIONED-CONTRACT01**: Create versioned targeted digest contract with Leamas producer version metadata in every targeted digest output.

## Summary

Implemented a versioned contract header (v1) that prepends metadata to every targeted digest output. The header includes contract version, Leamas build metadata (version, commit, build time), digest mode, and timestamp.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/digest/contract.go` | Created - Contract constants, HeaderInfo, RenderContractHeader, ParseContractHeader, ValidateContractHeader |
| `internal/factory/digest/contract_test.go` | Created - Unit tests for contract parsing/validation (193 lines) |
| `internal/factory/digest/contract_integration_test.go` | Created - Integration tests for digest output with contract header (192 lines) |
| `internal/factory/digest/digest.go` | Modified - RenderDigest prepends contract header to output |
| `internal/factory/digest/range.go` | Modified - Range digest functions prepend contract header |
| `docs/factory/digest-contract.md` | Created - Contract documentation |
| `docs/factory/digest.md` | Updated - Added contract header section |

## Behavior Changed

- **Digest output format**: Every digest now starts with a 6-field contract header followed by a blank line separator before the existing digest body
- **Auto mode resolution**: When `auto` mode resolves to `dirty` or `range`, the header reports the effective mode
- **Metadata injection**: Leamas build metadata (version, commit, build time) is embedded in every digest
- **Pure formatter**: `RenderContractHeader` is now a pure formatter accepting all values via `HeaderInfo`

## Verification

### Commands Run

```bash
# Build and generate proof digest
go build -o bin/leamas ./cmd/leamas
./bin/leamas factory digest --dirty --output /tmp/leamas-digest.txt
head -n 12 /tmp/leamas-digest.txt
```

### Proof Digest Output

```
LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 1
LEAMAS_VERSION: dev
LEAMAS_COMMIT: 245cb609afa56c59c0e131d02c5f769f90fa5256
LEAMAS_BUILD_TIME: 2026-07-09T10:48:52Z
DIGEST_MODE: dirty
DIGEST_CREATED_AT: 2026-07-09T11:08:18Z

# Targeted digest

Generated at: 2026-07-09T11:08:18Z
```

### Quality Gate Results

```bash
make factorize  # *** FACTORIZE PASSED ***
make gate       # *** GATE PASSED ***
#   go mod tidy... OK
#   gofmt... OK
#   go vet ./... OK
#   go test ./... OK
#   static build... OK
```

## R1 Items Addressed

1. **Pure formatter**: `RenderContractHeader` removed internal `version.Get()` and `time.Now()` calls
2. **Single timestamp**: Both contract header and legacy "Generated at" use the same `createdAt` timestamp
3. **Removed misleading helper**: Deleted `GetEffectiveMode` which didn't actually resolve auto mode
4. **Documentation defaults**: Corrected contract docs to reflect version package defaults (`dev` for Version)

## Decisions Made

- Contract version is integer `1` (additive-stable policy)
- Future contract changes must append fields only, never reorder or remove
- `auto` mode reports effective mode (dirty/range), not `auto`
- Header fields use `KEY: VALUE` format with trailing space
- Version metadata defaults: `Version="dev"`, `Commit="unknown"`, `BuildTime="unknown"`

## Agent Doctrine Impact

- Verifiers parsing digest output should check for contract header starting with `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION`
- Consumers should ignore unknown header fields for forward compatibility

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| | | |

## Notes

- Split contract_test.go into unit and integration tests to stay within 400-line LLM-friendliness limit
- Documentation includes example header, field descriptions, and verification commands
