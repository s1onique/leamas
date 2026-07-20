# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01

## Status
IN PROGRESS

## Motivation

Review of committed NORMALIZATION01 identified several contract defects:

1. **Epic was replaced** rather than updated; authoritative content was lost → FIXED
2. **Duplicate-name diagnostic paths** use name instead of index → FIXED
3. **Projection ignores returned errors** → FIXED
4. **Sealed-document validation incomplete** → FIXED
5. **NormalizeWithFault exported** → FIXED
6. **newIntegerFromWire doesn't return errors** → FIXED
7. **Stale findDuplicateWireNames helper** → FIXED

## Remaining Scope (P1)

1. Add literal 41-row corpus matrix
2. Add missing semantic matrices
3. Strengthen source-document aliasing tests

## Blocking

- `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` BLOCKED pending this correction
