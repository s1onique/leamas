# Doctrine: Go Only

For v0, Go is the only permitted implementation language.

## Core Principle

The entire Leamas core is implemented in Go. No runtime language mixing.

## Scope

### Permitted

- Core tool implementation in Go
- Build scripts in shell (bash/sh)
- Documentation in Markdown
- External tools called via `exec`
- Configuration in TOML/YAML/JSON

### Not Permitted

- Plugin systems in other languages
- Runtime script evaluation (Python, JavaScript, etc.)
- Native extensions
- Foreign language FFI in production code

## Rationale

1. **Single binary**: Go compiles to static executables
2. **Minimal runtime**: No interpreter, no JVM
3. **Toolchain simplicity**: `go build` is sufficient
4. **Explicit dependencies**: go.mod is auditable
5. **Fast feedback**: Quick compile-test cycle

## What "Go Only" Does NOT Imply

- Future versions may add other languages
- Shell scripts for packaging/CI are allowed
- Documentation can use any format
- Calling external tools is not "language mixing"

## Verification

A file is "Go only" compliant when:
- Extension is `.go` for production code
- Or extension is `.sh`/`.bash` for build scripts
- Or extension is `.md`/`.yaml`/`.toml`/`.json` for config/docs

Production code directories should only contain `.go` files.

## References

- ADR-0002: Go only for v0
