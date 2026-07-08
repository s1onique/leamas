# Doctrine: Factory Meta-Loop

The Factory is self-documenting, self-verifying, and self-maintaining.

## Core Principle

The tools that build Leamas must also verify Leamas. The Factory is part of the product.

## The Meta-Loop

```
┌─────────────────────────────────────────────────────────┐
│                   Factory Meta-Loop                     │
│                                                         │
│  ┌──────────┐    ┌──────────┐    ┌──────────────────┐  │
│  │  Write   │───▶│  Verify  │───▶│  Document        │  │
│  │  Code    │    │  Gate    │    │  Decision        │  │
│  └──────────┘    └──────────┘    └──────────────────┘  │
│       ▲                                       │        │
│       └───────────────────────────────────────┘        │
│              (doctrine informs next decision)          │
└─────────────────────────────────────────────────────────┘
```

## Self-Verification Requirements

### Every Change Must

1. Pass `make gate` before merge
2. Update doctrine if behavior changes
3. Add ADR if decision is made
4. Record in close report if ACT is closed

### Doctrine Verifies Itself

- `verify_doctrine_inventory.sh`: Ensures all doctrine docs exist
- `verify_factory_docs.sh`: Ensures ADRs/ACTs are linked
- `verify_forbidden_patterns.sh`: Catches drift
- `verify_doctrine_agent_contracts.sh`: Ensures agent contract sections exist

### Factory Docs Must Be Factory Docs

- ADRs are located in `docs/adr/`
- ACTs are located in `docs/acts/`
- Templates are located in `docs/templates/`
- Verifiers are located in `scripts/`

## What This Prevents

- ❌ Forgotten documentation
- ❌ Orphaned ADRs
- ❌ Unverified assumptions
- ❌ Doctrine drift from implementation

## Factorization Principle

**Factorize early.** Before product features, establish the Factory. The Factory protects the product.

## Agent Contract

### Always

- Run `make factorize` before claiming verification is complete.
- Update doctrine when behavior changes.
- Link decisions to ADRs.
- Include verification hooks in doctrine files.

### Never

- Skip verifiers to meet deadlines.
- Leave doctrine drift from implementation.
- Create orphan ADRs that don't link to code.
- Weaken gates to make patches pass.

### Ask / Escalate

- If a verifier is blocking a legitimate change.
- If doctrine needs updating but ACT scope is limited.
- If a new pattern requires a new verifier.

### Verification Hooks

- `make factorize` (runs all factory verifiers)
- `scripts/verify_doctrine_*.sh`
- `scripts/verify_factory_docs.sh`
- `scripts/verify_forbidden_patterns.sh`

## References

- This document (meta-level)
- `scripts/verify_*.sh` (implementation)
- `docs/templates/` (workflow artifacts)
