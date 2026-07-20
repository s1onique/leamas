# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01

## Status
IN PROGRESS — P0 Complete; P1 Pending

## Motivation

Review of committed NORMALIZATION01 identified several contract defects:

1. **Epic was replaced** rather than updated; authoritative content was lost → FIXED
2. **Duplicate-name diagnostic paths** use name instead of index → FIXED
3. **Projection ignores returned errors** → FIXED
4. **Sealed-document validation incomplete** → FIXED
5. **NormalizeWithFault exported** → FIXED
6. **newIntegerFromWire doesn't return errors** → FIXED
7. **Stale findDuplicateWireNames helper** → FIXED
8. **Test blind spot: malformed WireInteger projection** → FIXED

## Completed

**P0 Production Fixes (c39fcfa..c5fc16d):**
- `validateSealed()` rejects both invalid pointer states
- `projectV1` and `projectV2` now propagate integer-conversion errors
- `newIntegerFromWire` rejects empty values and validates complete decimal string with `big.Int.SetString`
- Duplicate names produce `/checks/<index>/name`
- Multiple later occurrences retain distinct paths
- Stale duplicate helper removed
- `normalizeWithFault` unexported
- Authoritative epic restored

**P0 Test Fixes:**
- Sealed-document validation tests (neither/both populated)
- Invalid integer conversion tests
- Malformed WireInteger projection tests (v1 duration_ms, v2 duration_ms, v2 exit_code, v2 test_total)
- Duplicate-name multiple occurrences test (3 names → 2 diagnostics)

**Epic Board Updated:**
```
NORMALIZATION01    = CLOSED — superseded by CORRECTION01
CORRECTION01       = IN PROGRESS
DIGEST01           = BLOCKED
CLI01              = PENDING
```

## Remaining Scope (P1)

1. Add literal 41-row ownership/result corpus matrix
2. Generated exit-code matrix
3. Totals matrix
4. Aggregate-status matrix
5. Cleanliness matrix
6. Strengthen source-document aliasing tests

## Verification

```bash
go test -race ./internal/gatesummary/...  # PASS
go vet ./internal/gatesummary/...         # PASS
CGO_ENABLED=0 go build ./cmd/leamas       # PASS
```

## Blocking

- `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` BLOCKED pending CORRECTION01 P1 completion
