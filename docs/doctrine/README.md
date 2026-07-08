# Doctrine

Core principles guiding Leamas development.

## Agent Use

Doctrine files are written for both humans and LLM/agent-assisted development.

Agents must treat doctrine as active project instructions:

- use doctrine to constrain implementation plans
- use Agent Contract sections during patch generation and review
- preserve Always/Never boundaries unless an ACT explicitly changes doctrine
- report conflicts instead of silently resolving them
- use Verification Hooks to decide what to run before closure

A doctrine is not complete until it is understandable by a future maintainer and operationally useful to an agent.

## The Leamas Way

### Agent-Assisted Development

LLMs and agents may help with Leamas, but their claims must be grounded in evidence and bounded by ACT scope.

→ See [agent-assisted-development.md](agent-assisted-development.md)

### Local First

Everything must work on a developer's local machine before considering any other deployment target. Cloud, cluster, or enterprise deployment is optional; local usability is mandatory.

→ See [local-first.md](local-first.md)

### Web First

The primary human interface is a local web cockpit. Developers interact through a browser on localhost.

→ See [web-first.md](web-first.md)

### Single Binary

One binary. No runtime dependencies. No configuration files. No home directory setup. Drop it in PATH and it works.

→ See [single-binary.md](single-binary.md)

### Go Only (v0)

For the initial version, Go is the only language. Simplicity in the toolchain translates to simplicity in usage.

→ See [go-only.md](go-only.md)

### No Enterprise Complexity

No OAuth, no OIDC, no LDAP integration, no Active Directory. If Leamas ever needs authentication, it will be minimal and opt-in.

→ See [no-enterprise-swamp.md](no-enterprise-swamp.md)

### Not a Gateway

Leamas is not an LLM gateway, provider router, or model control plane. It verifies, it doesn't route.

→ See [not-a-gateway.md](not-a-gateway.md)

### Verify, Then Trust

Leamas exists to make test and verification harnesses accountable. It does not assume trust; it verifies claims.

→ See [verification-witness.md](verification-witness.md)

### Factory Meta-Loop

The Factory is self-documenting, self-verifying, and self-maintaining. The tools that build Leamas must also verify Leamas.

→ See [factory-meta-loop.md](factory-meta-loop.md)

### Minimal Dependencies

Every external dependency is a liability. Prefer standard library. Prefer zero dependencies over one.

### Developer Ergonomics

The tool should feel natural. Clear output, sensible defaults, helpful error messages. If a developer needs a manual to understand basic operations, we failed.

## Anti-Patterns We Avoid

- Adding a database "just in case"
- Assuming Kubernetes is the deployment target
- Adding enterprise governance features prematurely
- Including OAuth/OIDC "for future-proofing"
- Creating configuration formats before understanding the real configuration needs
- Routing LLM requests between providers
- Implementing multi-tenancy before a shared rig exists

## Architecture Decision Records (ADRs)

See [docs/adr/README.md](../adr/README.md) for architectural decisions.

## License

TBD — Will be open source with a permissive license.
