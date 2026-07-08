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

## References

- This document (meta-level)
- `scripts/verify_*.sh` (implementation)
- `docs/templates/` (workflow artifacts)
