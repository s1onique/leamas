# Factory: Digest Redaction Policy

**ACT**: ACT-LEAMAS-FACTORY-DIGEST-SOURCE-REDACTION-POLICY01

## Purpose

The digest implements a source-aware redaction policy that preserves review fidelity for source files while still protecting secrets in non-source artifacts.

## Policy Overview

| File Type | Default Behavior | Rationale |
|-----------|-----------------|-----------|
| Source files (.py, .go, .ts, etc.) | Preserve content; emit warnings | Review fidelity - source must be syntactically valid |
| Non-source files (.log, .env, .json, etc.) | Redact secrets | Operational secret risk |

## Source File Extensions

Files with these extensions are treated as source for redaction purposes:

- `.py` (Python)
- `.go` (Go)
- `.ts`, `.tsx` (TypeScript)
- `.js`, `.jsx` (JavaScript)
- `.rs` (Rust)
- `.zig` (Zig)
- `.java`, `.kt`, `.kts` (Java/Kotlin)
- `.c`, `.h`, `.cpp`, `.hpp` (C/C++)
- `.sh`, `.bash`, `.zsh`, `.fish` (Shell)

## Per-File Redaction Metadata

Each file section in the digest includes redaction policy metadata:

**Source files:**
```
REDACTION_POLICY:
  class=source
  decision=preserve_and_warn
  redaction_applied=false
  source_secret_scan=warn_only

SOURCE_SECRET_WARNINGS:
  - line=42 kind=source.password_assignment confidence=pattern
  - line=43 kind=source.secret_assignment confidence=pattern
```

**Non-source files:**
```
REDACTION_POLICY:
  class=non_source
  decision=redact
  redaction_applied=true
```

## Source Secret Warnings

When secret-like patterns are detected in source files, warnings are emitted instead of redaction:

| Field | Description |
|-------|-------------|
| `line` | Line number (1-indexed) |
| `kind` | Pattern ID (e.g., `source.password_assignment`) |
| `confidence` | Detection confidence (`pattern` or `high`) |

**Warning fields are safe:** The warning metadata does NOT include raw secret values.

## Pattern IDs for Source Warnings

| Pattern ID | Description |
|------------|-------------|
| `source.password_assignment` | Password literal assignment |
| `source.secret_assignment` | Secret literal assignment |
| `source.token_assignment` | Token literal assignment |
| `source.api_key_assignment` | API key literal assignment |
| `source.bearer_token` | Bearer token format |
| `source.pem_private_key` | PEM private key header |

## Future Mode Reservation

A future `--llm-safe` or `--public-safe` mode may:

- Omit source files entirely
- Redact source files with explicit opt-in

This mode is reserved but not yet implemented.

## Migration Notes

Before this policy:

- Source files were sometimes mutated (e.g., `"password=secret123"` → `"password=[REDACTED)`)
- This created false syntax errors in digest output

After this policy:

- Source files are preserved byte-faithfully
- Secret-like patterns in source trigger warnings
- Non-source artifacts continue to be redacted

This is an intentional change that improves review fidelity.

## Implementation

- **Source**: `internal/factory/digest/redaction_policy.go`
- **Source**: `internal/factory/digest/source_secret_warning.go`
- **Source**: `internal/factory/digest/redaction_handler.go`
- **Tests**: `internal/factory/digest/redaction_policy_test.go`
- **Tests**: `internal/factory/digest/source_secret_warning_test.go`
