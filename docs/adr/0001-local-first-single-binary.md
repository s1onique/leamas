# ADR-0001: Local-first single binary

## Status

Accepted

## Context

Leamas is a developer tool intended to make test and verification harnesses accountable. For maximum portability and zero-friction adoption, the tool should be easy to install and run anywhere—a developer's laptop, a CI/CD pipeline, a container, or an SSH session.

Traditional distribution models requiring interpreters, runtime environments, or package managers create friction. Many developers work in heterogeneous environments where installing a specific runtime (Python, Node, Java) may not be feasible.

## Decision

Leamas will be distributed as a **single static binary** with the following properties:

- Statically compiled (no libc dependencies beyond what's statically linkable)
- Self-contained (no external configuration files or assets required)
- Drop-in executable (adds to PATH, runs immediately)

The tool must work on a developer's local machine as the primary deployment target. Cloud, cluster, or enterprise deployment is optional and secondary.

## Rationale

1. **Zero installation friction**: Download, chmod +x, run.
2. **Reproducible environments**: Same binary, same behavior everywhere.
3. **CI/CD friendly**: One curl or wget command in a pipeline.
4. **No dependency hell**: Developers have Go; that's enough.
5. **Local-first validation**: Test on your machine before anywhere else.

## Consequences

### Positive

- Developers can try Leamas in seconds
- Consistent behavior across macOS, Linux, and WSL
- No "works on my machine" issues from runtime differences
- Container images can use scratch or distroless base

### Negative

- Binary size will be larger than dynamic linking (acceptable tradeoff)
- Platform-specific builds required for Windows (if needed)
- Debugging may require more tooling awareness

### Neutral

- Release process requires cross-compilation pipeline
- Build times increase due to static linking

## Alternatives Considered

| Alternative | Rejected Reason |
|-------------|-----------------|
| Go module install | Requires Go toolchain on target machine |
| Docker container | Adds container runtime dependency |
| Package managers | Fragmented repos, version lag |
| Interpreted language | Runtime dependency on developer machine |
