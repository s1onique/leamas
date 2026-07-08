# Doctrine: Local First

Everything must work on a developer's local machine before considering any other deployment target.

## Core Principle

The local machine is the primary deployment target. Cloud, cluster, or enterprise deployment is optional; local usability is **mandatory**.

## Implications

- Features are designed, built, and tested locally first
- Documentation assumes a local development workflow
- CI/CD integration is a bonus, not a requirement
- Performance benchmarks target local hardware (laptop-class)
- Offline operation is the default, not an edge case

## What This Is NOT

- This is not "offline-first" in the cloud-native sense
- This is not a rejection of cloud deployment
- This is not anti-networking or anti-collaboration

## Verification

A feature is "local first" when:
1. It runs without network connectivity
2. It requires no external services to function
3. Its behavior is identical locally vs. remotely (when both are available)

## Agent Contract

### Always

- Design and test features locally before assuming remote/cloud behavior is equivalent.
- Verify features work without network connectivity.
- Flag any feature that requires external services to function.

### Never

- Introduce features that require cloud infrastructure for basic operation.
- Add network dependency assumptions without explicit documentation.
- Assume cloud deployment is equivalent to local behavior without verification.

### Ask / Escalate

- If a feature cannot be reasonably tested without external services.
- If local operation would require significant architectural changes.

### Verification Hooks

- `scripts/verify_forbidden_patterns.sh` (checks for cloud-specific dependencies)

## References

- ADR-0001: Local-first single binary
- ADR-0003: Web-first local cockpit
