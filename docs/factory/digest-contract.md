# Factory: Digest Contract
**ACT**: ACT-LEAMAS-FACTORY-DIGEST-REVIEW-EVIDENCE-V2-01,
ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01,
ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION01 (in progress)

## Purpose

The targeted digest contract establishes a stable, versioned header format for all digest output. This enables:

- **Consumers** to parse digest metadata reliably (human or LLM)
- **Producers** to emit deterministic, reviewable output
- **Evolution** to happen via explicit version bumps, not silent drift

## Contract vs. Product Version

| Field | Meaning |
|-------|---------|
| `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION` | Format version of the digest output (currently `2`) |
| `LEAMAS_VERSION` | Leamas application version (e.g., `0.1.0`, `dev`) |

The contract version governs **output shape**. The application version is **build metadata**.

- Contract version changes = breaking format changes (require bump)
- Application version changes = normal version drift (no contract bump needed)

## Version Compatibility Policy

### Contract Version `2`: Frozen

v2 promised only the subset `A/M/D/R/C/U/?` and the v2 stats-key
order; any digest matching that exact shape still satisfies v2.

### Contract Version `3`: Full Git status alphabet and rendering

Contract version `3` introduces:

* The full Git `--name-status -z` status alphabet:
  `A`, `M`, `D`, **`T`** (type change), **`R<N>`** (renamed, score
  dropped), **`C<N>`** (copied, score dropped), `U`, `?`,
  **`X`** (unknown), **`B`** (pairing broken).
* Three additional `CHANGESET_STATS` keys:
  `type_changed_files`, `unknown_files`, `broken_pair_files`.
  The canonical v3 key order is documented below.
* For **R** and **C**, both `old -> new` paths are preserved on
  the manifest entry. (v2 preserved only the rename source path.)
* Pathnames are rendered through `PathEscape` so unusual
  characters (tab, newline, CR, backslash, control bytes) do not
  split a single manifest entry across multiple visual lines.
* Internal semantic models continue to carry raw paths; only
  the rendering boundary applies `PathEscape`. `ComputeStats`
  generated / binary / source / test / doc / config classification
  still address the on-disk path.

v3 is **not** wire-compatible with v2: the new status letters
and three new statistics keys appear, and `CHANGESET_STATS` keys
now appear in a different order. v2 consumers that split the
section on `=`-and-newline will continue to parse the fields v2
promised; the v3 additions are silently ignored.

### Contract Version Lifecycle

```
1 → 2 (frozen) → 3 (current) → 4 ...
```

Consumers should check the version header, parse fields they
recognise for that version, and ignore fields they do not.

## Header Fields

Each field is `KEY: VALUE` with no extra whitespace before the colon.

### `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION`

- **Type**: Integer
- **Current value**: `2`
- **Meaning**: Digest output follows contract version 2 format

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
LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3
LEAMAS_VERSION: 0.2.0
LEAMAS_COMMIT: abc1234
LEAMAS_BUILD_TIME: 2026-07-09T10:24:46Z
DIGEST_MODE: dirty
DIGEST_CREATED_AT: 2026-09-07T10:50:00Z
```

## Review Evidence Sections (v2)

Contract version `2` adds deterministic review evidence sections before the file evidence.
These sections provide structured metadata for reviewer orientation.

### Section Order

The review evidence sections appear in this order:

1. `## CHANGESET_MANIFEST` - Stable list of changed files with status codes
2. `## CHANGESET_STATS` - Aggregated counts by file type and status
3. `## REVIEW_MAP` - Files grouped by reviewer role
4. `## RISK_SIGNALS` - Deterministic facts for reviewer focus
5. `## PATCH_HYGIENE` - Conflict marker and whitespace checks
6. `## EVIDENCE_HASHES` - Deterministic SHA-256 fingerprints over digest sections
7. `## GATE_SUMMARY` - Digest gate pass/fail status
8. `## PUBLIC_SURFACE_DELTA` - Public Go API/CLI surface changes
9. `## DEPENDENCY_DELTA` - Go module dependency changes

### CHANGESET_MANIFEST

Stable, sorted list of changed files with Git-style status codes.

**Status codes (v3):**
- `A` - Added (file present in index, absent in HEAD)
- `M` - Modified (file content differs from HEAD)
- `D` - Deleted (file present in HEAD, absent in index)
- `T` - Type changed (regular file ↔ symlink / submodule; file mode differs)
- `R` - Renamed (renders as `R  old/path.go -> new/path.go`; similarity score is dropped)
- `C` - Copied (renders as `C  source.go -> copy.go`)
- `U` - Unmerged / conflicted
- `?` - Untracked (`ls-files --others`; not from `git diff --name-status`)
- `X` - Unknown change type
- `B` - Pairing broken

**Requirements:**
- Deterministic ordering: lexical by normalized repository-relative path
- Repository-relative paths only
- Ignores ignored files

**Example:**
```
## CHANGESET_MANIFEST
A  internal/factory/digest/review_evidence.go
M  internal/factory/digest/digest.go
M  internal/factory/digest/range.go
A  internal/factory/digest/review_evidence_test.go
```

### CHANGESET_STATS

Deterministic counts derived from the manifest.

**Key order (v3, canonical):**
```
files_changed
added_files
modified_files
deleted_files
type_changed_files
renamed_files
copied_files
unmerged_files
unknown_files
broken_pair_files
untracked_files
binary_files
generated_files
test_files
doc_files
source_files
config_files
```

v3 inserts `type_changed_files`, `unknown_files`, and
`broken_pair_files` after the corresponding tracked-status keys,
and pulls `untracked_files` ahead of the file-classification
fields.

**File classification rules:**

| Category | Patterns |
|----------|----------|
| `test_files` | `*_test.go`, `*_test.ts`, `*_test.tsx`, `*.test.ts`, `*.test.tsx`, `test_*.py`, `tests/**` |
| `doc_files` | `*.md`, `*.adoc`, `*.rst`, `docs/**` |
| `config_files` | `Makefile`, `*.mk`, `*.yaml`, `*.yml`, `*.toml`, `*.json`, `.github/**`, `.gitlab-ci.yml` |
| `source_files` | Changed files not classified as test/doc/config/generated/binary |
| `generated_files` | Files with canonical `// Code generated by ... DO NOT EDIT.` marker |
| `binary_files` | Files containing null bytes |

**Example:**
```
## CHANGESET_STATS
files_changed=4
added_files=2
modified_files=2
deleted_files=0
type_changed_files=0
renamed_files=0
copied_files=0
unmerged_files=0
unknown_files=0
broken_pair_files=0
untracked_files=1
binary_files=0
generated_files=0
test_files=1
doc_files=1
source_files=2
config_files=0
```

### REVIEW_MAP

Files grouped by reviewer role for routing purposes. Fixed group order: production, tests, docs, config, generated, binary. Empty groups render as `  - none`. See [digest.md](./digest.md) for examples.

### RISK_SIGNALS

Deterministic facts that help the reviewer focus. These are **not** opinions—they are derived facts.

**Signal definitions:**

| Signal | Definition |
|--------|------------|
| `production_without_tests` | At least one production file changed and zero test files changed |
| `tests_without_production` | At least one test file changed and zero production files changed |
| `docs_without_code` | At least one doc file changed and zero production/test files changed |
| `generated_files_changed` | At least one generated file changed |
| `config_files_changed` | At least one config file changed |
| `deleted_files_changed` | At least one deleted file exists |
| `unmerged_files_present` | At least one unmerged/conflicted file exists |
| `large_file_changed` | At least one changed file exceeds `large_file_threshold_bytes` |
| `large_file_threshold_bytes` | Fixed threshold for large file detection (1 MiB) |

**Key order (stable):**
```
production_without_tests
tests_without_production
docs_without_code
generated_files_changed
config_files_changed
deleted_files_changed
unmerged_files_present
large_file_changed
large_file_threshold_bytes
```

**Example:**
```
## RISK_SIGNALS
production_without_tests=false
tests_without_production=false
docs_without_code=false
generated_files_changed=false
config_files_changed=false
deleted_files_changed=false
unmerged_files_present=false
large_file_changed=false
large_file_threshold_bytes=1048576
```

### PATCH_HYGIENE

Deterministic patch hygiene checks using `git diff --check`.

See [digest-patch-hygiene.md](./digest-patch-hygiene.md) for full specification.

### EVIDENCE_HASHES

Deterministic SHA-256 fingerprints over normalized digest sections. Answers "What exact evidence did we review?" with stable content hashes.

See [digest-evidence-hashes.md](./digest-evidence-hashes.md) for full specification.

## Redaction Policy (v2)

The digest implements a source-aware redaction policy. Source files are preserved for review fidelity while non-source artifacts are redacted.

See [digest-redaction-policy.md](./digest-redaction-policy.md) for full specification.

## Complete Digest Structure

```
<CONTRACT HEADER (7 lines)>
# Targeted digest

Generated at: <timestamp>
Repo: /path/to/repo
Mode: <mode>
...

## CHANGESET_MANIFEST
...

## CHANGESET_STATS
...

## REVIEW_MAP
...

## RISK_SIGNALS
...

## PATCH_HYGIENE
...

## EVIDENCE_HASHES
...

## GATE_SUMMARY
...

## PUBLIC_SURFACE_DELTA
...

## DEPENDENCY_DELTA
...

## Changed files
...

## Diffs
...

## Workflow anchors
...
```

## Future Extensions

See [digest-public-surface-delta.md](./digest-public-surface-delta.md) for PUBLIC_SURFACE_DELTA specification.

## File Content Contract

**Digest file bodies are never truncated by default.**

| Case | Expected behavior |
|------|-------------------|
| Untracked text file | Include full worktree content |
| Tracked unstaged text file | Include Git unstaged diff, without Leamas-side truncation |
| Tracked staged text file | Include Git staged diff, without Leamas-side truncation |
| Binary file | Do not dump bytes; show binary marker + size/hash |

The digest stays concise by **selecting fewer files**, not by **clipping included files**.

### Labeling

Untracked file content sections use the label `--- untracked file content ---` (not `preview`).

### Rationale

LLM-friendly does not mean "truncate selected files." It means:

```
small enough file set, full enough context
```

A digest with incomplete files is worse than no digest, because it creates false confidence during review.

## Implementation

- **Source**: `internal/factory/digest/contract.go` (contract constants)
- **Source**: `internal/factory/digest/review_*.go` (v2 evidence sections)
- **Tests**: `internal/factory/digest/*_test.go`
- **Constants**: Field names and contract version defined in `contract.go`

## Verification

```bash
# Run contract tests
go test ./internal/factory/digest/...

# Generate a real digest
go build -o bin/leamas ./cmd/leamas
./bin/leamas factory digest --dirty --output /tmp/digest.txt
```

## Related

- [Digest Documentation](./digest.md)
- [PATCH_HYGIENE Specification](./digest-patch-hygiene.md)
