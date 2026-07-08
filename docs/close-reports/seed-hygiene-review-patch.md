# Seed Hygiene Review Patch

**Date**: 2026-07-08
**Reviewer**: Digest-driven review (Factory-style Project Fitness Review)
**Status**: Closed

## Summary

This patch addresses seed hygiene issues discovered during a digest-driven review
of the Leamas repository. The review identified documentation inconsistencies where
templates referenced non-existent directories and files, and a script contained
hardcoded references to frontend files from another project.

## Reviewer Findings Addressed

### Issue 1: Missing docs/epics/ and docs/acts/ directories

**Finding**: `docs/templates/README.md` instructed users to copy templates into `docs/epics/` and `docs/acts/`, but those directories did not exist.

**Resolution**: Added placeholder files:
- `docs/epics/.gitkeep`
- `docs/acts/.gitkeep`

### Issue 2: Missing docs/templates/adr.md template

**Finding**: `docs/adr/README.md` instructed users to copy `docs/templates/adr.md`, but that template did not exist.

**Resolution**: Created `docs/templates/adr.md` with required sections:
- Title
- Status
- Context
- Decision
- Consequences
- Alternatives Considered
- Revisit Criteria

### Issue 3: Hardcoded workflow anchors in make_targeted_digest.sh

**Finding**: `scripts/make_targeted_digest.sh` contained hardcoded workflow anchors for `frontend/src/App.tsx`, `frontend/src/__tests__/app.test.tsx`, and `frontend/src/index.css` from another project.

**Resolution**: Replaced with Leamas-neutral behavior that prints "No workflow anchors configured." when no anchors are present.

## Files Added

| File | Purpose |
|------|---------|
| `docs/epics/.gitkeep` | Placeholder to preserve directory in git |
| `docs/acts/.gitkeep` | Placeholder to preserve directory in git |
| `docs/templates/adr.md` | Architecture Decision Record template |
| `docs/close-reports/seed-hygiene-review-patch.md` | This close report |

## Files Modified

| File | Change |
|------|--------|
| `docs/adr/README.md` | Replaced "TBD - to be created" with live link to ../templates/adr.md |
| `docs/templates/README.md` | Added ADR template to table and usage section |
| `scripts/make_targeted_digest.sh` | Removed frontend/src/App.tsx anchors, replaced with neutral message |
| `scripts/quality_gate.sh` | Added checks for new files (.gitkeep, adr.md) |

## Verification

```bash
# Make scripts executable
chmod +x scripts/quality_gate.sh scripts/make_targeted_digest.sh

# Run quality gate
./scripts/quality_gate.sh

# Run make gate
make gate

# Run make test
make test

# Generate digest
mkdir -p build
./scripts/make_targeted_digest.sh --dirty --output build/leamas-hygiene-digest.txt
test -s build/leamas-hygiene-digest.txt

# Verify no frontend references
grep -R "frontend/src/App.tsx" -n scripts docs && exit 1 || true

# Verify anchor message present
grep -R "No workflow anchors configured" -n build/leamas-hygiene-digest.txt

# Cleanup
rm -f build/leamas-hygiene-digest.txt
```

**Verification Status**: All checks passed.

## Follow-up ACTs

None required for this patch. The hygiene issues have been addressed.