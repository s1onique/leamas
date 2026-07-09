# Duplicate Code Detection

**ACT**: ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01  
**Ratchet ACT**: ACT-LEAMAS-FACTORY-DUPCODE-BASELINE-THRESHOLD-GATE01

## Purpose

The duplicate code verifier detects substantial copy-paste duplication in Go source
files. It uses scanner-based tokenization and normalization to catch blocks of code that
have been copied between files, even when identifiers have been renamed.

## Baseline + Ratchet Model

The verifier uses a **baseline + ratchet** model to prevent regression while tolerating
existing duplication:

1. **Baseline**: A committed snapshot of current duplicate findings (`.factory/dupcode-baseline.json`)
2. **Ratchet**: The gate only fails on **new** or **worsened** duplication

This means:
- Existing duplication is grandfathered (tolerated temporarily)
- Any new duplicate code is blocked
- Any expansion of existing duplicates (new occurrence locations) is blocked
- The codebase cannot get worse, only better

### Policy Summary

| Setting | Value |
|---------|-------|
| Detection `MinLines` | 40 |
| Detection `MinTokens` | 400 |
| Gate threshold | 0 new, 0 worsened |

## Why Native Go Instead of jscpd/dupl?

- **Hermetic**: No external runtime dependencies (Node/npm/Python)
- **Fast**: Tokenization via `go/scanner` is O(n) and local
- **Deterministic**: Same input always produces same output
- **Go-owned**: Fits Leamas' Go-only doctrine

The native verifier handles Go code. Future ACTs may add optional polyglot support
via jscpd-compatible reporting.

## Detection Thresholds

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MinLines` | 40 | Minimum lines for a duplicate block |
| `MinTokens` | 400 | Minimum tokens for a duplicate block |

These thresholds determine what size of clone is worth tracking. The gate threshold
is always **0** - any new or worsened duplication fails the gate.

## Baseline Commands

```bash
# Generate/update the baseline
make dupcode-baseline

# Or directly:
leamas factory verify dupcode --update-baseline

# Run verification (normal gate mode)
leamas factory verify dupcode

# Custom baseline path
leamas factory verify dupcode --baseline .factory/dupcode-baseline.json

# Custom thresholds
leamas factory verify dupcode --min-lines 40 --min-tokens 400
```

## When to Update the Baseline

Baseline updates are **allowed** for:
- Known historical duplication that pre-exists new code changes
- Accepting existing duplication that cannot be refactored immediately
- Periodic cleanup of the baseline when duplication is actually removed

Baseline updates are **NOT for**:
- Hiding new duplication introduced by recent changes
- Bypassing the gate without addressing the underlying issue

**Rule**: If a PR introduces new duplication, either:
1. Remove the duplication before merging, or
2. Document why it must be accepted temporarily, then address it in a follow-up

## Baseline Schema

```json
{
  "schema_version": 1,
  "generated_at": "2026-07-09T00:00:00Z",
  "tool": "leamas dupcode",
  "thresholds": {
    "min_lines": 40,
    "min_tokens": 400
  },
  "findings": [
    {
      "fingerprint": "sha256-hash-of-normalized-fingerprint",
      "token_count": 400,
      "line_count": 42,
      "occurrences": [
        {
          "path": "internal/example/foo.go",
          "start_line": 10,
          "end_line": 55
        }
      ]
    }
  ]
}
```

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

## Exit Codes

- `0` - No new or worsened duplicate code detected (gate passes)
- `1` - New or worsened duplicates found (verification failed)
- `2` - Internal error (config issue, file access, baseline error)

## Algorithm

1. **File Discovery**: Walk repository, collect `.go` files, apply exclusions
2. **Tokenization**: Use `go/scanner` to extract tokens (excluding comments) with
   line positions
3. **Normalization**: Replace identifiers with `IDENT`, strings with `STRING`,
   numbers with `NUMBER`
4. **Fingerprinting**: Rolling window (step=1) over tokens creates fingerprints
5. **Hashing**: SHA256 hash of normalized fingerprint for stable baseline matching
6. **Grouping**: Identical fingerprints across 2+ files become findings
7. **Deduplication**: Overlapping windows in the same file are merged
8. **Comparison**: Current findings compared against baseline for new/worsened

## Design Decisions

### Why Token-Based Detection?

Text-based detection (diff) catches identical code but misses renamed copy-paste.
Token-based detection normalizes identifiers, making renamed duplicates detectable.

### Why Baseline + Ratchet?

High thresholds pass the gate but don't prevent duplication from getting worse.
Low thresholds detect real issues but fail on existing debt.
Baseline + ratchet gets both: low thresholds that catch new issues while
tolerating known historical debt.

### Why SHA256 Fingerprint Hash?

Fingerprints can be long and may vary slightly between runs. SHA256 provides a
stable, fixed-length identifier for baseline comparison.

### Why No Polyglot Support Now?

Supporting multiple languages requires either:
1. Multiple tokenizers (Go, TypeScript, Python, etc.)
2. Text-based detection (less accurate)
3. External tool integration (adds dependency)

Go-only is the right scope for v1. Polyglot support is a future follow-up.

## Baseline Integrity Verifier

The baseline integrity verifier (`dupcode-baseline`) validates the committed baseline artifact itself:

```bash
# Run baseline integrity check
leamas factory verify dupcode-baseline

# With custom baseline path
leamas factory verify dupcode-baseline --baseline .factory/dupcode-baseline.json
```

### What It Checks

1. **Baseline presence**: Fails if `.factory/dupcode-baseline.json` is missing
2. **Git tracking**: Fails if baseline is not tracked by git
3. **Schema validation**: Fails on malformed JSON or unsupported schema version
4. **Threshold policy**: Fails if thresholds don't match policy (40/400)
5. **Path contract**: Fails on absolute paths, backslashes, parent traversal, empty paths
6. **Line validity**: Fails on invalid line numbers (≤0, end < start)
7. **Fingerprint contract**: Fails on empty/invalid SHA256 fingerprints or duplicates
8. **Ordering**: Fails if findings/occurrences are not sorted
9. **Drift check**: Re-runs scanner and fails if baseline is stale

### Why It Exists

Without the baseline verifier, the dupcode ratchet can become:
- **Noisy**: Stale baseline causes false positives
- **Toothless**: Missing/ignored baseline causes false negatives

The baseline verifier protects the ratchet itself.

### Command Summary

| Command | Purpose |
|---------|---------|
| `dupcode-baseline` | Validates baseline artifact integrity |
| `dupcode` | Enforces no new/worsened duplication |
| `dupcode --update-baseline` | Intentionally refreshes accepted debt |

### Warning

Never update the baseline merely to hide new duplication. Baseline changes should be reviewed like code changes.

## References

- [Go-only Doctrine](../doctrine/go-only.md)
- [Tooling Boundaries](./tooling-boundaries.md)
- [ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01](../close-reports/
  ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01.md)
- [ACT-LEAMAS-FACTORY-DUPCODE-BASELINE-THRESHOLD-GATE01](../close-reports/
  ACT-LEAMAS-FACTORY-DUPCODE-BASELINE-THRESHOLD-GATE01.md)
