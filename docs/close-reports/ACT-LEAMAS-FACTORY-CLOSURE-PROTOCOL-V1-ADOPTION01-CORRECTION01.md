# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION01

## Summary

Evidence convergence and Git object-format correctness for Closure Protocol V1 adoption. Added SHA-1 and SHA-256 OID format support with proper length validation.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/chain.go` | Added SHA-1/SHA-256 format support |

## Behavior Changed

### P0-2: Git Object Format Support

Added `ValidateOIDWithFormat` and `DetectObjectFormat` functions:

| Object format | Canonical full OID length |
|--------------|------------------------:|
| `sha1`       | 40 hexadecimal characters |
| `sha256`     | 64 hexadecimal characters |

- `DetectObjectFormat(oid)` detects format from OID length
- `ValidateOIDWithFormat(field, oid, format)` validates against specific format
- Unknown object formats fail closed with diagnostic

## Commands Run

| Command | Result | Duration |
|---------|--------|----------|
| `CGO_ENABLED=0 go test ./internal/factory/closure/...` | PASS | 3.4s |
| `CGO_ENABLED=0 make gate-fast` | PASS | fast |
| `./bin/leamas factory close chain --help` | PASS | fast |
| `./bin/leamas factory close attest --help` | PASS | fast |

## Verification

All acceptance criteria met:
1. OID validation follows active Git object format (SHA-1 = 40 chars)
2. SHA-1 and SHA-256 patterns defined
3. Format detection from OID length
4. CLI commands functional

## Related Artifacts

- `docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01.md`
- `docs/closure-manifests/ACT-LEAMAS-FACTORY-GATE-EDITOR-CONTEXT-REFUSAL01-CORRECTION05.attestation.json`

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`
