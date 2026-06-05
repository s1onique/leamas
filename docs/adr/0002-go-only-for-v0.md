# ADR-0002: Go only for v0

## Status

Accepted

## Context

Programming language choice affects:
- Build toolchain requirements
- Dependency management philosophy
- Ecosystem and library availability
- Team expertise and learning curve
- Distribution strategy

For a local-first developer tool prioritizing simplicity, choosing a language that aligns with the single-binary constraint is critical.

## Decision

For Leamas v0 (initial development phase), **Go is the only permitted implementation language**.

### Scope of "Go only"

- Core tool implementation in Go
- Build scripts may use shell (bash/sh)
- Documentation and templates may use Markdown
- No runtime language mixing (no plugins in other languages)

### What this does NOT imply

- Future versions may add other languages (evaluated later)
- Shell scripts for packaging/ci are allowed
- Documentation can use any markup format
- External tools called via exec are not "language mixing"

## Rationale

1. **Single binary friendliness**: Go compiles to static executables natively.
2. **Minimal runtime**: No interpreter, no JVM, no .NET runtime.
3. **Fast compilation**: Quick feedback loops during development.
4. **Explicit dependencies**: go.mod makes dependency tracking transparent.
5. **Toolchain availability**: Go is commonly installed on developer machines.
6. **Team alignment**: Assumed developer comfort with Go.

## Consequences

### Positive

- Clean, explicit dependency management
- Fast, reliable builds
- Single toolchain for core development
- Strong standard library coverage

### Negative

- Limited ecosystem compared to Python/JavaScript
- Some libraries only available in other languages (workaround: exec)
- Language party tricks unavailable (generics are Go 1.18+)

### Neutral

- Concurrency model differs from other languages
- Error handling pattern is explicit (if err != nil)

## Alternatives Considered

| Alternative | Rejected Reason |
|-------------|-----------------|
| Python | Requires Python runtime on target; packaging complexity |
| Rust | Steeper learning curve, longer compile times |
| Node.js | Requires Node runtime; different ecosystem |
| C/C++ | Manual memory management overhead |
| Java/Kotlin | JVM dependency contradicts single-binary goal |

## Revisit Criteria

This decision may be revisited if:
- A compelling use case for another language emerges
- Go ecosystem lacks critical functionality with no workaround
- Team composition changes significantly
- Performance requirements exceed Go's strengths
