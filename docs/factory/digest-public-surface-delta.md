# PUBLIC_SURFACE_DELTA Section

The `PUBLIC_SURFACE_DELTA` section answers: **"Did this patch change the public Go API/CLI surface a reviewer should inspect?"**

This section provides a deterministic delta of exported Go symbols and CLI commands, enabling reviewers to quickly identify API surface changes that may require careful attention.

## Placement in Digest

In digest v2, sections appear in this order:

```
...
## GATE_SUMMARY
...
## PUBLIC_SURFACE_DELTA  ← NEW
...
## Changed files
...
```

## Format

```markdown
## PUBLIC_SURFACE_DELTA
language=<lang>
source_status=<status>
packages_changed=<count>
symbols_added=<count>
symbols_removed=<count>
symbols_modified=<count>
cli_commands_changed=<count>

packages:
  - <package path>
  - ...

added:
  - <symbol>
  - ...

removed:
  - <symbol>
  - ...

modified:
  - <symbol>
  - ...

cli_commands:
  - <command>
  - ...
```

## Fields

| Field | Description |
|-------|-------------|
| `language` | Always "go" for this section |
| `source_status` | Always "present" when section is rendered |
| `packages_changed` | Number of Go packages whose exported symbols changed |
| `symbols_added` | Count of new exported functions, types, constants, variables, and methods |
| `symbols_removed` | Count of removed exported symbols |
| `symbols_modified` | Count of modified exported symbols (signature changes) |
| `cli_commands_changed` | Count of new, removed, or modified Cobra command definitions |

## Symbol Types Tracked

- **func**: Standalone exported functions
- **type**: Exported type definitions (struct, interface, alias)
- **const**: Exported constant declarations
- **var**: Exported variable declarations
- **method**: Methods on exported types (format: `Method.methodname(ReceiverType)`)
- **field**: Exported fields in exported struct types
- **interface_method**: Methods defined in exported interfaces

## Detection Logic

### Go Packages
- Scans changed `.go` files for exported (capitalized) identifiers
- Parses AST to identify symbol types and their signatures
- Only tracks symbols in `pkg/` directories or non-`main` packages
- **Package-level comparison**: Exported symbols are merged across all files in the same package before comparison. This prevents false removals when a symbol is deleted from one file but still exists in another file of the same package.

### CLI Commands
- Scans `cmd/` directories for Cobra command definitions
- Detects new commands via `&cobra.Command{...}` or `cobra.Command{...}` patterns
- Detects command additions via `rootCmd.AddCommand()` calls

### Symbol Identity
- Symbol identity is based on package-level qualified name, not file path
- Format: `<package-path>.<symbol-name>(<kind>)`
- Examples:
  - `pkg.example.Generate(func)`
  - `internal.factory.digest.ChangedFile(type)`
  - `pkg.example.MyStruct.Value.field(MyStruct)`
- File paths are evidence locations, not symbol identity

### Intra-Package File Splits
Moving an exported symbol between files in the same package is not reported as a public-surface addition or removal. The comparison operates at package level:

```text
Before: type ChangedFile in digest.go
After:  type ChangedFile in file_operations.go
Result: no public surface delta (symbol preserved at package level)
```

This ensures that refactoring code across files within a package does not trigger false positives in the digest.

## Deterministic Output

The section output is deterministic:
- Symbol keys are sorted alphabetically within each category
- Counts are computed from sorted, deduplicated symbol sets
- Package paths are normalized (no leading `./`)

## Evidence Hash

The section content is included in `EVIDENCE_HASHES` for reproducibility verification:

```markdown
public_surface_delta_sha256=<sha256 hex>
```

## Example

### Before (no public API changes)
```markdown
## PUBLIC_SURFACE_DELTA
language=go
source_status=present
packages_changed=0
symbols_added=0
symbols_removed=0
symbols_modified=0
cli_commands_changed=0

packages:
  - none

added:
  - none

removed:
  - none

modified:
  - none

cli_commands:
  - none
```

### After (API additions)
```markdown
## PUBLIC_SURFACE_DELTA
language=go
source_status=present
packages_changed=1
symbols_added=2
symbols_removed=0
symbols_modified=1
cli_commands_changed=1

packages:
  - pkg.example

added:
  - pkg.example.AddedFunc
  - pkg.example.NewType

removed:
  - none

modified:
  - pkg.example.OldFunc

cli_commands:
  - new
```

## Design Rationale

1. **Reviewer Focus**: Helps reviewers quickly identify if a patch touches public APIs, which typically warrant extra scrutiny.

2. **Deterministic**: Uses AST parsing for reliable symbol detection, not heuristics.

3. **Minimal Overhead**: Only processes changed files, not entire codebase.

4. **Extensible**: Format allows adding new symbol categories without breaking existing parsers.

## Related Sections

- [GATE_SUMMARY](./digest-gate-summary.md) - Precedes PUBLIC_SURFACE_DELTA
- [EVIDENCE_HASHES](./digest-evidence-hashes.md) - Includes PUBLIC_SURFACE_DELTA in hash

## Contract Version

This section is part of digest v2. It is not present in v1 digests.
