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

## Evaluation Capabilities

Leamas may produce:
- findings
- scores
- warnings
- verdicts
- summaries

Leamas must distinguish:
- **Observed evidence**: Direct output from runs
- **Inferred evaluation**: Analysis based on rules or heuristics
- **Recommendations**: Human or agent guidance

Leamas may evaluate, score, and warn.

Leamas must not present an evaluation as an observed fact unless evidence supports it.

Every verdict must be traceable to evidence, a rule, or an explicitly marked heuristic.

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

## Agent Contract

### Always

- Separate observation from evaluation.
- Link verdicts to evidence, rule IDs, or explicitly marked heuristics.
- Preserve raw or redacted raw artifacts when possible.
- Mark unsupported claims explicitly.

### Never

- Treat LLM output as proof by itself.
- Emit green closure without evidence.
- Hide skipped, missing, or unavailable evidence.
- Rewrite history to match a desired conclusion.

### Ask / Escalate

- If evidence is missing for a verdict claim.
- If LLM output is being used as primary proof.
- If evaluation requires assumption that cannot be verified.

### Verification Hooks

- `scripts/verify_doctrine_agent_contracts.sh` (checks observation/evaluation separation)

## References

- ADR-0001: Local-first single binary (evidence stays local)
