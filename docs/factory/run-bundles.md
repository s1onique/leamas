# Run Bundles

Run bundles are Leamas's durable local evidence unit for verification witness operations.

## Purpose

Run bundles provide a local, filesystem-backed, portable, and JSON-readable structure for recording verification evidence. They enable:

- **Local-first evidence storage**: No network, database, or external service required
- **Portable evidence**: Easy to copy, archive, and inspect
- **Safe by construction**: Run ID validation prevents path traversal
- **Reviewable format**: JSON metadata is human-readable

## Directory Layout

```
.leamas/runs/
  <run-id>/
    metadata.json
    claims/
    evidence/
    digests/
    traces/
    verifier-results/
```

### Layout Details

| Directory | Purpose |
|-----------|---------|
| `metadata.json` | Structured metadata about the run bundle |
| `claims/` | Claim artifacts (future ACT) |
| `evidence/` | Evidence artifacts (future ACT) |
| `digests/` | Digest files (future ACT) |
| `traces/` | Trace files (future ACT) |
| `verifier-results/` | Verifier output (future ACT) |

## Metadata Contract

The `metadata.json` file has the following schema:

```json
{
  "schema_version": "leamas.runbundle.v1",
  "run_id": "run-20260709T071704Z-smoke01",
  "created_at": "2026-07-09T07:17:04Z",
  "tool": {
    "name": "leamas",
    "version": "v0.1.0"
  },
  "doctrine": {
    "local_only": true,
    "read_only": true,
    "no_database": true
  }
}
```

### Field Definitions

| Field | Type | Description |
|-------|------|-------------|
| `schema_version` | string | Must be exactly `leamas.runbundle.v1` |
| `run_id` | string | Unique identifier for this run |
| `created_at` | string | RFC3339/RFC3339Nano timestamp |
| `tool.name` | string | Tool that created this bundle (default: `leamas`) |
| `tool.version` | string | Tool version (optional) |
| `doctrine.local_only` | bool | Always `true` for run bundles |
| `doctrine.read_only` | bool | Always `true` for run bundles |
| `doctrine.no_database` | bool | Always `true` for run bundles |

## Safety Rules

### Run ID Contract

Run IDs must be safe path segments:

- Non-empty
- Maximum 128 characters
- Must start with `run-`
- Only `[a-zA-Z0-9._-]` characters
- Not `.` or `..`
- Not absolute paths
- No path separators (`/` or `\`)
- No traversal components (`..`)

**Valid examples:**
- `run-20260709T071704Z-smoke01`
- `run-20260709T071704Z-abcdef`
- `run-abc123`

**Invalid examples:**
- `""` (empty)
- `../escape` (traversal)
- `/absolute` (absolute path)
- `run/bad` (path separator)
- `.hidden` (hidden file)

### Bundle Path Contract

- Root directory must be non-empty
- Run ID must pass validation
- Result must remain under root lexically
- No symlink resolution required in this seed

## Non-Goals

This seed ACT intentionally does NOT implement:

- Witness proxy persistence wiring
- Cockpit run-bundle list UI
- Claim/evidence full domain model
- Trace capture storage
- Export/import bundle archive format
- Database/sql, SQLite, Postgres, MySQL
- Auth/session/cookie/OIDC/RBAC
- Provider routing or model routing
- Background daemon
- Filesystem watcher

## Package Location

```
internal/witness/runbundle
```

### Allowed Imports

- `encoding/json`
- `errors`
- `fmt`
- `io`
- `os`
- `path/filepath`
- `strings`
- `time`

### Public API

```go
type RunID string
type Bundle struct { Root, ID, Path string }
type Metadata struct { SchemaVersion, RunID, CreatedAt, Tool, Doctrine }
type CreateOptions struct { Root, RunID, Now, ToolName, Version }

func Create(opts CreateOptions) (Bundle, error)
func Open(root string, id RunID) (Bundle, *Metadata, error)
func ValidateRunID(id RunID) error
func BundlePath(root string, id RunID) (string, error)
```

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| `ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01` | Add CLI to create/list/inspect local run bundles | P0 |
| `ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01` | Seed claim/evidence domain models | P1 |
| `ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01` | CLI to inspect witness proxy captures | P1 |
| `ACT-LEAMAS-WEB-RUN-BUNDLE-LIST01` | Web cockpit run bundle list UI | P2 |
| `ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01` | Hulk run bundle core integration | P2 |

## References

- [Doctrines](../doctrine/)
- [Verification Witness](../doctrine/verification-witness.md)
- [Local-First](../doctrine/local-first.md)
- [Go Only](../doctrine/go-only.md)
- [ADR-0006: Filesystem Run Bundles](../adr/0006-filesystem-run-bundles.md)
