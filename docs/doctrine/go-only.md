# Doctrine: Go Only

For v0, Go is the only permitted implementation language.

## Core Principle

The entire Leamas core is implemented in Go. No runtime language mixing.

## Repository Language Boundary

Leamas v0 permits only Go and small Bash.

Python is forbidden everywhere in the repository.

Bash is permitted only as glue. New executable Bash scripts must stay at or under 50 non-comment, non-blank LOC.

Substantial automation, labs, verifiers, digest generation, and harness logic belong in Go.

## Scope

### Permitted

- Core tool implementation in Go
- Build scripts in shell (bash/sh) - small glue only
- Documentation in Markdown
- External tools called via `exec`
- Configuration in TOML/YAML/JSON

### Not Permitted

- Python anywhere (production code, tests, labs, verifiers, build scripts, helpers)
- Plugin systems in other languages
- Runtime script evaluation (JavaScript, etc.)
- Native extensions
- Foreign language FFI in production code
- Long Bash scripts with domain logic or complex verifiers

### Bash LOC Rule

New executable Bash scripts must be no more than 50 meaningful LOC.

Meaningful LOC means non-blank, non-comment lines:

```bash
grep -vE '^[[:space:]]*($|#)' "$file" | wc -l
```

Bash may:
- Dispatch Make targets
- Call Go commands
- Set environment variables
- Provide tiny compatibility wrappers
- Perform simple file-existence checks

Bash must not:
- Implement domain logic
- Implement labs or complex verifiers long-term
- Parse complex structured data
- Contain business rules or multi-phase workflows
- Replace Go subcommands

## Rationale

1. **Single binary**: Go compiles to static executables
2. **Minimal runtime**: No interpreter, no JVM
3. **Toolchain simplicity**: `go build` is sufficient
4. **Explicit dependencies**: go.mod is auditable
5. **Fast feedback**: Quick compile-test cycle
6. **Agent clarity**: Clear boundaries help agents write compliant code

## What "Go Only" Does NOT Imply

- Future versions may add other languages (via ADR process)
- Shell scripts for packaging/CI are allowed when small
- Documentation can use any format
- Calling external tools is not "language mixing"

## Verification

A file is "Go only" compliant when:
- Extension is `.go` for production code
- Or extension is `.sh`/`.bash` for small glue scripts
- Or extension is `.md`/`.yaml`/`.toml`/`.json` for config/docs

Production code directories should only contain `.go` files.

Use `scripts/verify_tooling_boundaries.sh` to enforce Python ban and Bash LOC limits.

## Agent Contract

### Always

- Write product code, labs, verifiers, and substantial automation in Go.
- Keep Bash as small glue only.
- Enforce the no-Python rule everywhere.
- Use shell scripts only for build/CI automation.
- Use standard library when possible before external dependencies.

### Never

- Add Python files anywhere in the repository.
- Add new executable Bash scripts over 50 meaningful LOC.
- Implement labs or substantial Factory logic in Bash.
- Add plugin systems in non-Go languages.
- Add FFI to other languages in production code.

### Ask / Escalate

- If a feature genuinely requires a different language runtime.
- If an external tool cannot be invoked via `exec`.
- If an automation task is too large for a tiny Bash wrapper (implement in Go).

### Verification Hooks

- `scripts/verify_single_language.sh`
- `scripts/verify_tooling_boundaries.sh`

## References

- ADR-0002: Go only for v0
- ACT-LEAMAS-FACTORY-TOOLING-BOUNDARIES01: Tooling language boundaries
- Google Shell Style Guide: shell should only be used for small utilities or simple wrapper scripts
