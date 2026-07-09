# ACT-LEAMAS-FACTORY-DIGEST-RANGE-DEDUPE01

**Date**: 2026-07-09
**Status**: CLOSED

## Summary

Fixed duplicate file entries in Leamas range-mode targeted digest output. Each changed file now appears exactly once in both the `## Changed files` inventory and the `## Diffs` section.

## Root Cause

The duplication originated in `internal/factory/digest/range.go` in the `GetRangeFiles` function. Two separate Git commands were collecting the same added files:

1. Line 23: `git diff --name-status -z revRange` - already collected all changed files including those with status "A" (added)
2. Lines 68-81: `git diff --name-only --diff-filter=A -z revRange` - redundantly re-collected all added files

This caused every added file to appear twice in the returned slice, resulting in duplicate entries in the rendered digest.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/digest/range.go` | Removed redundant Git command; added `UniqueRangeFiles` dedupe helper |
| `internal/factory/digest/range_dedup_test.go` | New test file with 7 regression tests |

## Tests Added

| Test | Description |
|------|-------------|
| `TestUniqueRangeFiles_DedupesDuplicates` | Unit test for dedupe helper with duplicates |
| `TestUniqueRangeFiles_EmptySlice` | Edge case: empty slice returns empty |
| `TestUniqueRangeFiles_SingleElement` | Edge case: single element unchanged |
| `TestRangeDigest_DedupesAddedFilesInInventory` | Integration test: added files appear once in inventory |
| `TestRangeDigest_DedupesAddedFileDiffBlocks` | Integration test: added files appear once in diff blocks |
| `TestRangeDigest_DedupesMultipleAddedFilesWithoutDroppingAny` | Integration test: 5 files each appear exactly once |
| `TestRangeDigest_IntegrationWithRealGit` | Full integration test with temp Git repo |

## Commands Run

### Before Fix (hypothetical - duplication observed in manual testing)
```bash
leamas factory digest --range HEAD~1..HEAD --output /tmp/digest.txt
# Would show:
# cmd/leamas/factory_coverage.go  [added]
# cmd/leamas/factory_coverage.go  [added]
```

### After Fix
```bash
leamas factory digest --range HEAD~1..HEAD --output /tmp/leamas-range-digest.txt
# Now shows:
# cmd/leamas/factory_coverage.go  [added]
```

### Verification
```bash
# Each file appears exactly once in inventory
for f in Makefile cmd/leamas/main.go cmd/leamas/version.go; do
  grep -c "^$f  " /tmp/leamas-range-digest.txt
done
# Output: 1 for each file

# Each file appears exactly once in diff blocks
for f in Makefile cmd/leamas/main.go cmd/leamas/version.go; do
  grep -c "^=== $f ===" /tmp/leamas-range-digest.txt
done
# Output: 1 for each file
```

## Verification Results

All checks passed:

```bash
go test ./internal/factory/digest/... -v
# PASS: TestUniqueRangeFiles_DedupesDuplicates
# PASS: TestUniqueRangeFiles_EmptySlice
# PASS: TestUniqueRangeFiles_SingleElement
# PASS: TestRangeDigest_DedupesAddedFilesInInventory
# PASS: TestRangeDigest_DedupesAddedFileDiffBlocks
# PASS: TestRangeDigest_DedupesMultipleAddedFilesWithoutDroppingAny
# PASS: TestRangeDigest_IntegrationWithRealGit

go test ./...
go vet ./...
make factorize
make gate
# *** GATE PASSED ***
```

## Behavior Preserved

- Dirty/staged digest behavior is unchanged
- The dedupe is applied to range-mode output only
- Files are de-duplicated by path, preserving first-seen occurrence
- Stable ordering maintained (sorted by path after dedupe)

## Follow-Up ACTs

None required. The fix is complete and tested.
