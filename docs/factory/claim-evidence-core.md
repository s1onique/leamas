# Claim/Evidence Core

The claim/evidence core provides typed domain models for claims, evidence, and sources in Leamas review artifacts.

## Overview

Claim/evidence is a pure domain core that models the relationships between:
- **Claims**: Assertions about the system under review
- **Evidence**: Verifiable artifacts that support or refute claims
- **Sources**: The origin of evidence (human, agent, verifier, artifact)

## Package Location

```text
internal/hulk/claimevidence
```

## Design Constraints

The claim/evidence core is:

- **Go-only**: Implemented entirely in Go
- **Pure domain logic**: No side effects, no global mutable state
- **Deterministic**: Same input always produces same output
- **Testable without filesystem**: No file reads, no process execution
- **Testable without Git**: No Git operations
- **Testable without clocks**: No timestamps, no time dependencies
- **No network**: No HTTP, no database connections
- **No database**: No database/sql or similar
- **No UI dependencies**: No web servers, no Cockpit dependencies

## What This Package Does

- Defines typed identifiers (`ClaimID`, `EvidenceID`, `SourceID`, `ArtifactID`)
- Defines narrow status and kind types (`ClaimStatus`, `ClaimKind`, `EvidenceKind`, `SourceKind`, `ConfidenceLevel`)
- Provides the core domain model (`Claim`, `Evidence`, `Source`, `ClaimEvidenceBundle`)
- Provides pure validation (`Validate()`) that returns deterministic findings
- Provides helper validity functions for each narrow type

## What This Package Does NOT Do

The claim/evidence core is intentionally minimal. It does NOT:

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

## Relationship to Run Bundle Core

The claim/evidence core is independent from `internal/hulk/runbundle`.

The run bundle core (`ClaimRef`, `EvidenceRef`) remains intentionally minimal until a future ACT integrates the full claim/evidence model.

The claim/evidence core can be used independently for pure domain modeling without run bundle dependencies.

## Import Boundary

The package imports only standard library packages:

- `sort` (for deterministic finding order)

No other imports are permitted.

## Related Documentation

- [Run Bundle Core](./run-bundle-core.md)
- [Doctrines](../doctrine/)
- [Verification Witness](../doctrine/verification-witness.md)
- [Close Report: ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01](../close-reports/ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01.md)
