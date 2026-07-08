# ADR-0005: Not an LLM Gateway

## Status

Accepted

## Context

The AI tooling landscape includes many "LLM gateways" or "LLM proxies":
- LiteLLM: Unified API over multiple providers
- PortKey: AI gateway with observability
- FreeAI: Open-source LLM gateway
- Generic proxies: Custom implementations

These tools share common patterns:
- Route requests between LLM providers
- Manage API keys centrally
- Implement rate limiting
- Provide cost tracking/budgeting
- Offer fallback/failover between models

Leamas has a different primary purpose: verification and accountability of AI-assisted work.

## Decision

Leamas is NOT an LLM gateway, provider router, or model control plane.

### What Leamas Does NOT Do

- ❌ Route requests between LLM providers
- ❌ Manage API keys for multiple providers
- ❌ Implement rate limiting across providers
- ❌ Provide budget tracking or cost management
- ❌ Offer fallback or failover between models
- ❌ Provide a unified API over providers
- ❌ Implement virtual key systems
- ❌ Act as a multi-provider control plane

### What Leamas DOES Do

- ✅ Run verification test cases
- ✅ Record evidence of AI interactions
- ✅ Provide local tooling for AI quality assurance
- ✅ Work with existing integrations (not replace them)
- ✅ **May implement a local witness proxy for capture/evidence**

### Local Witness Proxy (Allowed)

Leamas may sit between a harness and an LLM provider as a **local capture proxy**.

This does not make Leamas a gateway. The proxy is an evidence boundary, not a provider-management layer.

The witness proxy:
- Captures traffic for verification purposes
- Passes requests through to the actual provider
- Records evidence without modifying behavior
- Is local-only (localhost), never a production proxy

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

## Consequences

### Positive

- Clear differentiation from other AI tooling
- No API key security concerns
- Works with any existing setup
- Can be used alongside gateways (doesn't conflict)
- Local witness proxy enables deep verification

### Negative

- Doesn't solve provider key management
- Doesn't provide unified API
- Doesn't offer cost optimization

### Neutral

- Could work alongside a gateway (record its traffic)
- Future gateway integration possible (as observer, not participant)

## What This Enables

Leamas can be used:
- Against direct provider APIs
- Behind an existing gateway (as a testing layer)
- With a local witness proxy (capture mode)
- With any AI integration that provides observability hooks
- For local-only verification without network access to providers

## References

- Doctrine: Not a Gateway
- Doctrine: Verification Witness

## Revisit Criteria

This decision may be revisited if:
- A compelling use case for gateway features emerges
- The community requests routing capabilities
- Integration with specific providers requires it
