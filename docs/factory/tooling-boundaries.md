# Tooling Boundaries: Grandfathered Bash

**ACT**: ACT-LEAMAS-FACTORY-TOOLING-BOUNDARIES01

## Overview

The Leamas Factory now forbids new executable Bash scripts over 50 meaningful LOC.

Existing long Bash Factory scripts are temporarily grandfathered because they were created before the tooling-boundary verifier existed.

## Policy

- **Python**: Immediate and absolute ban everywhere in the repository.
- **Bash >50 LOC**: Forbidden for new scripts immediately.
- **Existing long Bash**: Temporarily grandfathered until migrated to Go.

## Grandfathered Scripts

The following Factory scripts are currently over 50 meaningful LOC and are temporarily allowed:

| Script | Meaningful LOC | Reason for Grandfathering | Migration Target |
|--------|---------------|---------------------------|------------------|
| `scripts/verify_tooling_boundaries.sh` | ~114 | This ACT's bootstrapping exception | Go verifier |
| `scripts/quality_gate.sh` | ~218 | Quality gate runner with multiple verifiers | Go subcommand |
| `scripts/make_targeted_digest.sh` | ~211 | Digest generation for staged changes | Go subcommand |
| `scripts/verify_doctrine_agent_contracts.sh` | ~105 | Agent contract verification | Go verifier |
| `scripts/verify_single_language.sh` | ~70 | Single language enforcement | Go verifier |
| `scripts/verify_forbidden_patterns.sh` | ~67 | Forbidden pattern detection | Go verifier |
| `scripts/verify_static_binary_intent.sh` | ~60 | Static binary intent check | Go verifier |
| `scripts/verify_factory_docs.sh` | ~58 | Factory docs structure verification | Go verifier |

## Migration Requirements

All grandfathered scripts must be migrated to Go under:

**`ACT-LEAMAS-FACTORY-GO-VERIFIERS01`**

Migration criteria:
1. Functionality moved into Go subcommands under `cmd/`
2. Original Bash scripts removed or reduced to trivial wrappers
3. `verify_tooling_boundaries.sh` allowlist updated
4. Quality gate updated to use Go verifiers

## No New Long Bash Scripts

No new Bash scripts over 50 meaningful LOC may be added.

If an automation task is too large for a tiny Bash wrapper:
1. Implement the substantial logic in Go
2. Keep the Bash wrapper minimal (dispatch, environment setup)
3. If in doubt, ask for an ACT/ADR update

## Verification

Run `scripts/verify_tooling_boundaries.sh` to check compliance:

```bash
chmod +x scripts/verify_tooling_boundaries.sh
./scripts/verify_tooling_boundaries.sh
```

Or via Make:

```bash
make verify-tooling-boundaries
```

## References

- [ACT-LEAMAS-FACTORY-TOOLING-BOUNDARIES01](../close-reports/ACT-LEAMAS-FACTORY-TOOLING-BOUNDARIES01.md)
- [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html)
- [Go-only Doctrine](../doctrine/go-only.md)
