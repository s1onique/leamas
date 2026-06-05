# Doctrine

Core principles guiding Leamas development.

## The Leamas Way

### Local First

Everything must work on a developer's local machine before considering any other deployment target. Cloud, cluster, or enterprise deployment is optional; local usability is mandatory.

### Single Binary

One binary. No runtime dependencies. No configuration files. No home directory setup. Drop it in PATH and it works.

### Go Only (v0)

For the initial version, Go is the only language. Simplicity in the toolchain translates to simplicity in usage.

### No Enterprise Complexity

No OAuth, no OIDC, no LDAP integration, no Active Directory. If Leamas ever needs authentication, it will be minimal and opt-in.

### Verify, Then Trust

Leamas exists to make test and verification harnesses accountable. It does not assume trust; it verifies claims.

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

## License

TBD — Will be open source with a permissive license.
