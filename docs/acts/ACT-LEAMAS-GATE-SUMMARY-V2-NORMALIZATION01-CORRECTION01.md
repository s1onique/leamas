# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01

## Status
IN PROGRESS

## Motivation

Review of committed NORMALIZATION01 identified several contract defects:

1. **Epic was replaced** rather than updated; authoritative content was lost
2. **Duplicate-name diagnostic paths** use name instead of index → deduplication hazard
3. **Projection ignores returned errors** → impossible integer states treated as zero
4. **Sealed-document validation incomplete** → both pointers populated returns v1
5. **NormalizeWithFault exported** → should be unexported
6. **Corpus test is too weak** → doesn't pin complete diagnostic sets

## Scope

1. Restore authoritative epic
2. Fix duplicate-name paths: `/checks/<index>/name`
3. Make projection return and propagate errors
4. Add sealed-document invariant validation
5. Unexport NormalizeWithFault
6. Add literal 41-row corpus matrix
7. Add missing semantic matrices
8. Strengthen aliasing tests

## Blocking

- `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` BLOCKED pending this correction
