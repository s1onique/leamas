# Doctrine: Not a Gateway

Leamas is not an LLM gateway, provider router, or model control plane.

## Core Principle

Leamas does NOT:
- ❌ Route requests between LLM providers
- ❌ Manage API keys for multiple providers
- ❌ Implement rate limiting across providers
- ❌ Provide budget tracking or cost management
- ❌ Offer fallback or failover between models
- ❌ Provide unified provider APIs
- ❌ Implement virtual key systems

## What Leamas IS

A **verification witness** that:
- Runs test cases against existing LLM integrations
- Records evidence of AI-assisted work
- Provides local tooling for AI quality assurance

### Local Witness Proxy (Allowed)

Leamas **may** implement a local witness proxy whose purpose is to capture and evaluate AI-assisted verification traffic.

The witness proxy is:
- A capture/evidence boundary, not a provider-management layer
- Local-only (localhost), never a production proxy
- Focused on recording, not routing

## What This Is NOT

- ❌ **Not LiteLLM**: Leamas doesn't abstract provider APIs
- ❌ **Not a provider router**: No intelligent request routing
- ❌ **Not a budget tracker**: No cost monitoring
- ❌ **Not a unified API**: No common interface over providers
- ❌ **Not a control plane**: No multi-provider orchestration

## Architecture Distinction

### LiteLLM / Gateway Pattern
```
Client → Gateway → [OpenAI | Anthropic | Azure | ...]
         ↑
    (manages keys,
     routes requests,
     tracks costs)
```

### Leamas Witness Proxy Pattern
```
Harness → Leamas (local capture) → Provider
                ↑
           (records evidence,
            passes through)
```

## Rationale

1. **Scope clarity**: One job, done well
2. **No credential management**: Doesn't touch API keys
3. **No single point of failure**: Not a critical path dependency
4. **Simpler security model**: No MITM concerns for routing
5. **Verification focus**: Evidence recording, not traffic management

## When Gateway Behavior May Be Added

Only if a compelling use case emerges that:
- Doesn't conflict with the verification witness role
- Maintains local-first principles
- Doesn't introduce enterprise complexity

## Agent Contract

### Always

- Treat local proxy work as evidence capture only.
- Preserve pass-through semantics unless a future ACT explicitly changes this.
- Keep provider-management features out of scope.

### Never

- Add provider routing.
- Add virtual keys.
- Add budget tracking.
- Add fallback/failover policy.
- Claim Leamas replaces LiteLLM.

### Ask / Escalate

- If an ACT requires non-local proxying.
- If provider abstraction appears necessary.
- If capture would require modifying request semantics.

### Verification Hooks

- `scripts/verify_forbidden_patterns.sh`
- `scripts/verify_doctrine_agent_contracts.sh` (checks for gateway behavior)

## References

- ADR-0005: Not an LLM gateway
- Doctrine: Verification Witness
