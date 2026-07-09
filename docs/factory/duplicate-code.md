# Duplicate Code Detection

**ACT**: ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01

## Purpose

The duplicate code verifier detects substantial copy-paste duplication in Go source
files. It uses scanner-based tokenization and normalization to catch blocks of code that
have been copied between files, even when identifiers have been renamed.

## Why Native Go Instead of jscpd/dupl?

- **Hermetic**: No external runtime dependencies (Node/npm/Python)
- **Fast**: Tokenization via `go/scanner` is O(n) and local
- **Deterministic**: Same input always produces same output
- **Go-owned**: Fits Leamas' Go-only doctrine

The native verifier handles Go code. Future ACTs may add optional polyglot support
via jscpd-compatible reporting.

## Default Thresholds

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MinLines` | 100 | Minimum lines for a duplicate block |
| `MinTokens` | 1000 | Minimum tokens for a duplicate block |

These thresholds are intentionally conservative (high) to avoid noisy failures from
existing duplicate patterns in the codebase. The detector can be tuned tighter as needed.

## Ignored Paths

The verifier ignores these directories by default:

- `.git/` - Git internals
- `vendor/` - External dependencies
- `node_modules/` - Node packages
- `dist/` - Build artifacts
- `build/` - Build artifacts
- `.factory/` - Leamas factory artifacts
- `bin/` - Compiled binaries
- `testdata/` - Test fixtures

## Ignored File Patterns

- Files with `_test.go` suffix (tests are not checked)
- Files with `.pb.go` suffix (generated protobuf)
- Files with `.gen.go` suffix (generated code)
- Files containing `Code generated` marker
- Files containing `DO NOT EDIT` marker

Note: `//go:generate` comments are NOT used to detect generated files.

## How to Run Manually

```bash
# Via factory verify
leamas factory verify dupcode
```

## Interpreting Findings

Each finding represents a duplicate block:

```
Found 2 duplicate code blocks:

1. Duplicate block (200 tokens, ~25 lines):
   - internal/foo/bar.go:10-35
   - internal/baz/qux.go:10-35
```

A finding shows:
- **Token count**: Number of normalized tokens in the block
- **Line count**: Approximate source lines
- **Occurrences**: All locations where this block appears

## Exit Codes

- `0` - No substantial duplicates detected
- `1` - Duplicates found (verification failed)
- `2` - Internal error (config issue, file access, etc.)

## Algorithm

1. **File Discovery**: Walk repository, collect `.go` files, apply exclusions
2. **Tokenization**: Use `go/scanner` to extract tokens (excluding comments) with
   line positions
3. **Normalization**: Replace identifiers with `IDENT`, strings with `STRING`,
   numbers with `NUMBER`
4. **Fingerprinting**: Rolling window (step=1) over tokens creates fingerprints
5. **Grouping**: Identical fingerprints across 2+ files become findings
6. **Deduplication**: Overlapping windows in the same file are merged
7. **Sorting**: Results sorted by token count (desc), then path, then line

## Design Decisions

### Why Token-Based Detection?

Text-based detection (diff) catches identical code but misses renamed copy-paste.
Token-based detection normalizes identifiers, making renamed duplicates detectable.

### Why Minimum Thresholds?

Small repeated idioms (error handling patterns, logging helpers) are common and
acceptable. Thresholds prevent noise.

### Why No Cross-Language Support Now?

Supporting multiple languages requires either:
1. Multiple tokenizers (Go, TypeScript, Python, etc.)
2. Text-based detection (less accurate)
3. External tool integration (adds dependency)

Go-only is the right scope for v1. Polyglot support is a future follow-up.

## References

- [Go-only Doctrine](../doctrine/go-only.md)
- [Tooling Boundaries](./tooling-boundaries.md)
- [ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01](../close-reports/
  ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01.md)
