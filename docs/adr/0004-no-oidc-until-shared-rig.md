# ADR-0004: No OIDC until shared rig

## Status

Accepted

## Context

Authentication and authorization are common enterprise requirements. OIDC (OpenID Connect) and OAuth 2.0 are standard protocols for:
- Single Sign-On (SSO)
- Identity federation
- Access control delegation

However, implementing OIDC correctly is complex:
- Security vulnerabilities are subtle and severe
- Token validation requires cryptographic verification
- Provider-specific quirks
- Session management complexity
- Privacy/GDPR considerations

Leamas is currently a single-user local tool. Enterprise features should be earned, not assumed.

## Decision

Leamas will NOT implement OIDC, OAuth, or any authentication/authorization system until:

1. A concrete multi-user or multi-tenant use case is documented
2. A shared infrastructure ("shared rig") exists that requires it
3. A maintainer volunteers to own the security implications

### What This Means

- No authentication by default
- Single-user operation assumed
- No multi-tenancy support
- No RBAC/ABAC
- No API key management

### When This May Change

This decision will be revisited when:
- A real enterprise customer needs it
- The project has shared deployment scenarios
- Security audits require it
- A maintainer has bandwidth to implement correctly

## Rationale

1. **Complexity budget**: Auth adds significant complexity
2. **Security surface**: Auth bugs are security bugs
3. **Local-first**: Single-user local tools don't need auth
4. **Earn it**: Enterprise features should be requested, not assumed
5. **Maintenance burden**: Auth requires ongoing security attention

## Consequences

### Positive

- Simpler codebase
- No security implications from auth code
- Faster iteration on core features
- Clear scope boundary

### Negative

- Not suitable for multi-user shared deployments (yet)
- Not suitable for untrusted network access (yet)
- May limit adoption in enterprise environments (until implemented)

### Neutral

- Can be added later when needed
- Doesn't prevent future auth implementation

## What IS Permitted

Even without OIDC, Leamas may support:
- Environment variables for configuration
- CLI flags for options
- Optional config files (with defaults)
- Local file permissions (standard OS mechanisms)

## References

- Doctrine: No Enterprise Swamp
- Doctrine: Local First

## Revisit Criteria

This decision should be revisited when:
- A documented use case requires multi-user support
- A maintainer proposes a concrete implementation
- Security requirements change
