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

## References

- ADR-0001: Local-first single binary
- ADR-0003: Web-first local cockpit
