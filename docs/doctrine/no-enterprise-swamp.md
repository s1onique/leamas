# Doctrine: No Enterprise Swamp

Leamas is a developer tool, not an enterprise platform.

## Core Principle

Complexity is earned, not assumed. Enterprise features are opt-in, not foundational.

## Explicitly Excluded

### Authentication & Identity

- ❌ OAuth 2.0 / OAuth1
- ❌ OpenID Connect (OIDC)
- ❌ LDAP / Active Directory
- ❌ SAML / SSO integration
- ❌ API keys / tokens (unless for LLM providers in future)

### Authorization

- ❌ Role-Based Access Control (RBAC)
- ❌ Attribute-Based Access Control (ABAC)
- ❌ Permission matrices
- ❌ Multi-tenant isolation

### Data & Storage

- ❌ SQL databases (PostgreSQL, MySQL, etc.)
- ❌ NoSQL databases (MongoDB, DynamoDB, etc.)
- ❌ Cache layers (Redis, Memcached)
- ❌ Message queues
- ❌ Multi-tenancy data isolation

### Infrastructure

- ❌ Kubernetes manifests (for Leamas itself)
- ❌ Helm charts
- ❌ Terraform providers
- ❌ Service meshes

## What Leamas DOES Support

- Local file system access
- Environment variables
- CLI flags
- Optional config file (with defaults)
- Single-user operation

## When These May Change

Enterprise features may be considered when:
1. A specific, concrete use case is documented
2. The community requests it
3. A maintainer volunteers to own it
4. It doesn't complicate the core use case

## Agent Contract

### Always

- Reject features that add enterprise complexity by default.
- Keep Leamas focused on single-user, local operation.
- Document any exception that requires enterprise features.

### Never

- Add OAuth, OIDC, LDAP, or SAML support.
- Add RBAC, ABAC, or permission matrices.
- Add database dependencies (SQL or NoSQL).
- Add Kubernetes, Helm, or Terraform manifests for Leamas itself.

### Ask / Escalate

- If a requested feature appears to require enterprise infrastructure.
- If multi-user or multi-tenant behavior is implied.

### Verification Hooks

- `scripts/verify_forbidden_patterns.sh`

## References

- ADR-0004: No OIDC until shared rig
