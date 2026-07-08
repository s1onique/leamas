# ADR-0006: Filesystem run bundles

## Status

Accepted

## Context

Leamas needs to represent and store verification "runs" - records of test execution that include:
- Input test cases
- Execution context
- Results and evidence
- Timestamps and metadata

The choice of storage format affects:
- Portability
- Auditing capabilities
- Integration with existing tools
- Query and searchability
- Migration path

## Decision

Run bundles are stored as **directories on the filesystem** with a defined structure.

### Bundle Structure

```
run-bundle/
├── manifest.yaml       # Metadata, timestamps, context
├── cases/
│   ├── case-001.yaml   # Individual test case inputs
│   ├── case-002.yaml
│   └── ...
├── evidence/
│   ├── case-001/       # Evidence for case-001
│   │   ├── input.json
│   │   ├── output.json
│   │   └── metadata.yaml
│   └── ...
└── results.yaml        # Aggregated results
```

### Manifest Fields

```yaml
version: "1.0"
id: "uuid-v4"
started_at: "2024-01-01T00:00:00Z"
completed_at: "2024-01-01T00:01:00Z"
tool_version: "0.1.0"
context:
  working_dir: "/path/to/cwd"
  environment: {...}
```

### Benefits of Filesystem Storage

1. **Portable**: Works on any filesystem
2. **Auditable**: Standard version control compatible
3. **Inspectable**: Human-readable files
4. **Composable**: Can be archived, copied, shared
5. **Tool-friendly**: Works with existing CLI tools

## Rationale

1. **Local-first alignment**: No database required
2. **Git-compatible**: Can be version controlled
3. **Simple**: No storage server needed
4. **Explicit**: Clear where data lives
5. **Portable**: Bundle can be shared as archive

## Consequences

### Positive

- No database dependency
- Can be backed up with standard tools
- Easy to inspect and debug
- Portable across machines
- Version control friendly

### Negative

- No built-in querying (must use external tools)
- Large bundles may be unwieldy
- No transactional guarantees
- Filename conflicts possible

### Neutral

- Can add database backend later (if needed)
- Existing tools can query (jq, grep, etc.)

## Implementation Notes

### Bundle Creation

```bash
leamas run ./cases --output ./run-bundle-001
```

### Bundle Replay

```bash
leamas replay ./run-bundle-001
```

### Bundle Inspection

```bash
ls ./run-bundle-001/evidence/
cat ./run-bundle-001/manifest.yaml
```

## Alternatives Considered

| Alternative | Rejected Reason |
|-------------|-----------------|
| SQLite database | Adds dependency, less portable |
| PostgreSQL/MySQL | Contradicts single-binary goal |
| S3/object storage | Contradicts local-first principle |
| JSON single file | Hard to version control large runs |
| Custom binary format | Not human-inspectable |

## References

- ADR-0001: Local-first single binary
- Doctrine: Local First
- Doctrine: Verification Witness

## Revisit Criteria

This decision may be revisited if:
- Query performance becomes unacceptable
- Large-scale run archival is needed
- A compelling database use case emerges
