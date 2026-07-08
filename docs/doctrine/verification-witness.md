# Doctrine: Verification Witness

Leamas exists to make AI-assisted verification loops accountable.

## Core Principle

**Verify, then trust.** Leamas doesn't assume correctness; it provides evidence.

## Role Definition

Leamas is a **witness**, not a:
- Judge (doesn't decide pass/fail alone)
- Enforcer (doesn't block based on its output)
- Proxy (doesn't stand in for real integrations)

## What "Witness" Means

1. **Records what happened**: Captures evidence of AI interactions
2. **Provides traceability**: Links outputs to inputs
3. **Enables replay**: Run bundles can be re-executed
4. **Maintains audit trail**: Evidence persists beyond the session

## Verification Loop

```
┌─────────────────────────────────────────────────────────┐
│                    Verification Loop                     │
│                                                         │
│  ┌──────────┐    ┌──────────┐    ┌──────────────────┐  │
│  │  Define  │───▶│   Run    │───▶│  Record Evidence │  │
│  │  Cases   │    │  Bundle  │    │  & Compare       │  │
│  └──────────┘    └──────────┘    └──────────────────┘  │
│       ▲                                   │            │
│       └───────────────────────────────────┘            │
│              (iterate based on results)                 │
└─────────────────────────────────────────────────────────┘
```

## Non-Goals

- ❌ Leamas does NOT fix failed verifications
- ❌ Leamas does NOT tune models
- ❌ Leamas does NOT generate test cases automatically
- ❌ Leamas does NOT replace human review

## Evidence Properties

Evidence recorded by Leamas should be:
- **Verifiable**: Can be independently checked
- **Complete**: Includes inputs, outputs, and context
- **Reproducible**: Same input → same evidence (where applicable)
- **Searchable**: Can be queried and filtered

## References

- ADR-0001: Local-first single binary (evidence stays local)
