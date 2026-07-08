# Run Bundle Core

The run-bundle core provides typed domain models for Leamas run bundles.

## Overview

Run bundles are local, reviewable units that group an execution/run, its metadata, artifacts, claims, evidence references, and validation state.

## Package Location

```text
internal/hulk/runbundle
```

## Design Constraints

The run-bundle core is:

- **Go-only**: Implemented entirely in Go
- **Pure domain logic**: No side effects, no global mutable state
- **Deterministic**: Same input always produces same output
- **Testable without filesystem**: No file reads, no process execution
- **Testable without Git**: No Git operations
- **Testable without clocks**: Timestamps passed as strings
- **No network**: No HTTP, no database connections
- **No database**: No database/sql or similar
- **No UI dependencies**: No web servers, no Cockpit dependencies

## What This Package Does

- Defines typed identifiers (`RunBundleID`, `RunID`, `ArtifactID`, `ClaimID`, `EvidenceID`)
- Defines narrow status and kind types (`RunBundleStatus`, `ArtifactKind`)
- Provides the core domain model (`RunBundle`, `ArtifactRef`, `ClaimRef`, `EvidenceRef`)
- Provides pure validation (`Validate()`) that returns deterministic findings

## What This Package Does NOT Do

The run-bundle core is intentionally minimal. It does NOT provide:

- Filesystem read/write operations
- Process or command execution
- Git integration
- Persistence or storage
- Network behavior or HTTP servers
- Database connections or queries
- Witness proxy behavior
- Cockpit/UI rendering
- Provider or model routing
- LLM gateway semantics

These concerns belong to later ACTs.

## ClaimRef and EvidenceRef

`ClaimRef` and `EvidenceRef` are intentionally minimal references until `ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01`.

The current model provides:

```go
type ClaimRef struct {
    ID      ClaimID
    Summary string
}

type EvidenceRef struct {
    ID         EvidenceID
    ArtifactID ArtifactID
    Summary    string
}
```

## Import Boundary

The package imports only standard library packages:

- `sort` (for deterministic finding order)

No other imports are permitted.

## Related Documentation

- [Doctrines](../doctrine/)
- [Verification Witness](../doctrine/verification-witness.md)
- [Close Report: ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01](../close-reports/ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01.md)
