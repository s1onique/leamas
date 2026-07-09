# Factory: Digest Contract

**ACT**: ACT-LEAMAS-FACTORY-DIGEST-VERSIONED-CONTRACT01

## Purpose

The targeted digest contract establishes a stable, versioned header format for all digest output. This enables:

- **Consumers** to parse digest metadata reliably (human or LLM)
- **Producers** to emit deterministic, reviewable output
- **Evolution** to happen via explicit version bumps, not silent drift

## Contract vs. Product Version

| Field | Meaning |
|-------|---------|
| `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION` | Format version of the digest output (currently `1`) |
| `LEAMAS_VERSION` | Leamas application version (e.g., `0.1.0`, `dev`) |

The contract version governs **output shape**. The application version is **build metadata**.

- Contract version changes = breaking format changes (require bump)
- Application version changes = normal version drift (no contract bump needed)

## Version Compatibility Policy

### Contract Version `1`: Additive-Stable

Contract version `1` is stable. Future versions must:

- **Add** new fields only (appended to header, not inserted)
- **Never remove** or **rename** existing fields
- **Never change** field order

Breaking changes (removing fields, changing semantics, reordering) require a new contract version (`2`, `3`, etc.).

### Contract Version Lifecycle

```
1 → 2 → 3 ...
↑        ↑
Stable   Future
Current
```

Consumers should:
1. Check `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION`
2. Parse fields they recognize
3. Ignore fields they don't (forward compatibility)

## Header Fields

Each field is `KEY: VALUE` with no extra whitespace before the colon.

### `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION`

- **Type**: Integer
- **Current value**: `1`
- **Meaning**: Digest output follows contract version 1 format

### `LEAMAS_VERSION`

- **Type**: String
- **Values**: Injected at build time; default is `dev`
- **Meaning**: Leamas application version (from `-X github.com/.../version.Version=...`)

### `LEAMAS_COMMIT`

- **Type**: String (git SHA or `unknown`)
- **Meaning**: Git commit of the Leamas binary

### `LEAMAS_BUILD_TIME`

- **Type**: RFC3339 timestamp or `unknown`
- **Meaning**: When the Leamas binary was built

### `DIGEST_MODE`

- **Type**: String enum
- **Values**: `auto`, `dirty`, `staged`, `range`
- **Meaning**: Effective digest mode

**Note on `auto` mode**: When `auto` is requested, Leamas resolves to the actual
mode (`dirty` or `range`) based on working tree state. The header reports the
**resolved** mode, not `auto`, because the effective mode is what matters.

### `DIGEST_CREATED_AT`

- **Type**: RFC3339 UTC timestamp
- **Meaning**: When the digest was generated

## Example Header

```
LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 1
LEAMAS_VERSION: 0.2.0
LEAMAS_COMMIT: abc1234
LEAMAS_BUILD_TIME: 2026-07-09T10:24:46Z
DIGEST_MODE: dirty
DIGEST_CREATED_AT: 2026-09-07T10:50:00Z
```

## Complete Digest Structure

```
<CONTRACT HEADER (7 lines)>
# Targeted digest

Generated at: <timestamp>
Repo: /path/to/repo
Mode: <mode>
...

## Changed files
...

## Diffs
...

## Workflow anchors
...
```

## Implementation

- **Source**: `internal/factory/digest/contract.go`
- **Tests**: `internal/factory/digest/contract_test.go`
- **Constants**: Field names and contract version defined in `contract.go`

## Verification

```bash
# Run contract tests
go test ./internal/factory/digest/...

# Run all digest tests
go test ./internal/factory/digest/... -v

# Generate a real digest
go build -o bin/leamas ./cmd/leamas
./bin/leamas factory digest --dirty --output /tmp/digest.txt
head -n 10 /tmp/digest.txt
```

## Related

- [Digest Documentation](./digest.md)
- [LLM-Friendliness](./llm-friendliness.md)
- [Version Metadata](./version.md)
