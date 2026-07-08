# Tooling Boundaries

**ACT**: ACT-LEAMAS-FACTORY-GO-VERIFIERS01

## Overview

All Factory verification logic is now implemented in Go. Bash verifier scripts are forbidden.

## Policy

- **Python**: Immediate and absolute ban everywhere in the repository.
- **Bash >50 LOC**: Forbidden for new scripts immediately.
- **All verifiers**: Must be implemented in Go under `leamas factory verify ...`.

## Verifiers Are Go

All verifiers are Go commands under `leamas factory verify ...`.

```bash
leamas factory verify doctrine
leamas factory verify doctrine-agent-contracts
leamas factory verify docs
leamas factory verify forbidden-patterns
leamas factory verify language
leamas factory verify static-binary
leamas factory verify tooling-boundaries
leamas factory verify llm-friendly
leamas factory verify agent-context
leamas factory verify git-hooks
```

## Bash Wrappers (Tiny Glue Only)

Bash `scripts/verify_*.sh` files are compatibility wrappers only. Each is ≤50 meaningful LOC and delegates to `leamas factory verify ...`.

```bash
scripts/verify_doctrine_inventory.sh
scripts/verify_doctrine_agent_contracts.sh
scripts/verify_factory_docs.sh
scripts/verify_forbidden_patterns.sh
scripts/verify_single_language.sh
scripts/verify_static_binary_intent.sh
scripts/verify_tooling_boundaries.sh
```

These wrappers:
- Do not contain verification logic
- Delegate to `go run ./cmd/leamas factory verify ...` or `./bin/leamas factory verify ...`
- Pass the tooling-boundaries check

## Allowed Bash Glue

The following Git hooks and installers are permitted as tiny Bash glue:

| Script | Purpose | LOC |
|--------|---------|-----|
| `githooks/pre-push` | Pre-push hook to prevent force-pushes | ~24 |
| `scripts/install_git_hooks.sh` | Hook installer wrapper | ~6 |

## Quality Gate

The quality gate is now implemented in Go:

```bash
leamas factory gate
```

Or via Make:

```bash
make gate
```

## Factory Factorize

Factory verifiers run without toolchain checks:

```bash
leamas factory factorize
```

Or via Make:

```bash
make factorize
```

## Factory Digest

Targeted digest generation is now implemented in Go:

```bash
leamas factory digest --dirty --output build/digest.txt
leamas factory digest --staged --output build/staged-digest.txt
```

The Bash wrapper `scripts/make_targeted_digest.sh` is a tiny glue script that delegates to `leamas factory digest`.

## References

- [ACT-LEAMAS-FACTORY-GO-VERIFIERS01](../close-reports/ACT-LEAMAS-FACTORY-GO-VERIFIERS01.md)
- [Go-only Doctrine](../doctrine/go-only.md)
- [Agent-Assisted Development](../doctrine/agent-assisted-development.md)
