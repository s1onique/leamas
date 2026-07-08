# Close Report: ACT-LEAMAS-FACTORY-TOOLING-BOUNDARIES01

**ACT**: `ACT-LEAMAS-FACTORY-TOOLING-BOUNDARIES01`  
**Status**: CLOSED  
**Date**: 2026-08-07  
**Author**: Factory tooling architect  

## Summary

Added explicit repository tooling-language boundaries for Leamas:
- Python is banned everywhere (immediate and absolute)
- Bash is allowed only as small glue (≤50 meaningful LOC)
- Existing long Bash Factory scripts are temporarily grandfathered until migrated to Go

## Files Changed

| File | Action | Notes |
|------|--------|-------|
| `docs/doctrine/go-only.md` | Modified | Added repository language boundary, Bash LOC rule, updated agent contract |
| `docs/doctrine/agent-assisted-development.md` | Modified | Added tooling language rules for agents |
| `scripts/verify_tooling_boundaries.sh` | Created | New verifier enforcing Python ban and Bash LOC limits |
| `docs/factory/tooling-boundaries.md` | Created | Grandfathered Bash scripts inventory with migration notes |
| `Makefile` | Modified | Added `verify-tooling-boundaries` target, included in `factorize` |
| `scripts/quality_gate.sh` | Modified | Added tooling boundary verification |
| `scripts/verify_factory_docs.sh` | Modified | Added tooling-boundaries.md to required docs |

## Policy Details

### Python Ban (Immediate and Absolute)

Python is forbidden everywhere in the repository:
- Production code
- Tests
- Labs
- Verifiers
- Digest tools
- Build scripts
- One-off helper scripts
- Generated helper code committed to the repo

No `*.py` files are allowed. Exceptions require a future ADR.

### Bash LOC Rule (New Code)

New executable Bash scripts must be no more than 50 meaningful LOC.

Meaningful LOC = non-blank, non-comment lines:
```bash
grep -vE '^[[:space:]]*($|#)' "$file" | wc -l
```

### Grandfathered Existing Bash (Temporary)

The following Factory scripts are temporarily grandfathered over 50 LOC:

| Script | LOC | Target Migration ACT |
|--------|-----|---------------------|
| `scripts/quality_gate.sh` | ~120 | ACT-LEAMAS-FACTORY-GO-VERIFIERS01 |
| `scripts/make_targeted_digest.sh` | ~80 | ACT-LEAMAS-FACTORY-GO-VERIFIERS01 |
| `scripts/verify_doctrine_*.sh` | ~60 | ACT-LEAMAS-FACTORY-GO-VERIFIERS01 |
| `scripts/verify_factory_docs.sh` | ~50 | ACT-LEAMAS-FACTORY-GO-VERIFIERS01 |
| `scripts/verify_forbidden_patterns.sh` | ~50 | ACT-LEAMAS-FACTORY-GO-VERIFIERS01 |
| `scripts/verify_single_language.sh` | ~50 | ACT-LEAMAS-FACTORY-GO-VERIFIERS01 |
| `scripts/verify_static_binary_intent.sh` | ~50 | ACT-LEAMAS-FACTORY-GO-VERIFIERS01 |

## Verification Commands

```bash
# Make executable
chmod +x scripts/verify_tooling_boundaries.sh

# Run individual verifier
make verify-tooling-boundaries

# Run all factory verifiers
make factorize

# Run quality gate
make gate

# Generate digest
mkdir -p build
./scripts/make_targeted_digest.sh --dirty --output build/leamas-tooling-boundaries-digest.txt
test -s build/leamas-tooling-boundaries-digest.txt
```

## Verification Results

| Command | Result | Notes |
|---------|--------|-------|
| `make verify-tooling-boundaries` | PASSED | Grandfathered scripts acknowledged |
| `make verify-agent-doctrine` | PASSED | |
| `make verify-doctrine` | PASSED | |
| `make verify-factory` | PASSED | |
| `make verify-forbidden` | PASSED | |
| `make verify-single-lang` | PASSED | |
| `make verify-static` | PASSED | |
| `make factorize` | PASSED | All verifiers pass |
| `make gate` | PASSED | Go checks deferred (no go.mod yet) |

### Honest Assessment

- Python ban: VERIFIED - no Python files exist in repo
- Bash LOC >50 for new scripts: VERIFIED - verifier correctly fails on violations
- Grandfathered scripts: VERIFIED - all expected scripts are grandfathered
- All verifiers pass

## Follow-up ACTs

1. **ACT-LEAMAS-SEED-GO-MOD-CMD01** - Initialize Go module and minimal cmd/leamas
2. **ACT-LEAMAS-FACTORY-GO-VERIFIERS01** - Migrate substantial Factory verifiers and digest generation into Go

## References

- [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html)
- [Go-only Doctrine](docs/doctrine/go-only.md)
- [Agent Development Doctrine](docs/doctrine/agent-assisted-development.md)
- [Tooling Boundaries Grandfathered Bash](docs/factory/tooling-boundaries.md)
