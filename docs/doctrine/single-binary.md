# Doctrine: Single Binary

One binary. No dependencies. No configuration files. No setup.

## Core Principle

Download, `chmod +x`, run. That's the entire installation process.

## Requirements

A valid Leamas binary must:

1. **Be statically compiled**: No libc dynamically loaded at runtime
2. **Be self-contained**: No external config files required
3. **Work on first run**: No setup, no initialization, no wizard
4. **Be portable**: Same binary across macOS, Linux, WSL

## Build Requirements

```bash
CGO_ENABLED=0 go build -trimpath ./cmd/leamas
```

## What This Is NOT

- This is NOT "zero configuration" (sensible defaults are still config)
- This is NOT rejecting all CLI flags (flags are not config files)
- This is NOT forbidding environment variables (standard practice)
- This is NOT preventing optional config files (but they must have sane defaults)

## Anti-Patterns

❌ **Wrong**:
- "First run: create ~/.leamas/config.yaml"
- "Please run `leamas init` before use"
- "Edit /etc/leamas.conf to get started"

✅ **Right**:
- `./leamas run ./test-cases`
- `leamas --help` just works
- All flags have working defaults

## Verification

Binary is "single binary" compliant when:
- `file leamas` shows "statically linked" or builds succeed with `CGO_ENABLED=0`
- Running without any config files produces useful output
- No external dependencies required at runtime

## Agent Contract

### Always

- Build with `CGO_ENABLED=0` to ensure static linking.
- Test the binary runs without any setup.
- Include sensible defaults so the tool works on first run.

### Never

- Require users to run an init command before first use.
- Create hidden config directories or files automatically.
- Add runtime dependencies on shared libraries.

### Ask / Escalate

- If a feature genuinely requires external configuration files.
- If initialization logic appears necessary.

### Verification Hooks

- `scripts/verify_static_binary_intent.sh`

## References

- ADR-0001: Local-first single binary
