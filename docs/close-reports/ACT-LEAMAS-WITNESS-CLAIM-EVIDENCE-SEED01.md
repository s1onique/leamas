# Close Report: ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01

**Lane**: Product Primitive / Witness / Evidence Model
**Priority**: P1
**Status**: Closed
**Meta-epic**: `ME-05-WITNESS-RUN-BUNDLE-FOUNDATION`

## Summary

Seeded Leamas's typed claim/evidence domain model in `internal/witness/claim/`. The package provides typed IDs, schemas, filesystem store, and strict JSON validation.

**No CLI added. No claim evaluation added. No LLM calls added. No witness proxy persistence added. No cockpit UI added. No database added.**

## Files Changed

```
internal/witness/claim/id.go
internal/witness/claim/id_test.go
internal/witness/claim/claim.go
internal/witness/claim/claim_test.go
internal/witness/claim/evidence.go
internal/witness/claim/evidence_test.go
internal/witness/claim/evidence_constructor_test.go
internal/witness/claim/json.go
internal/witness/claim/store.go
internal/witness/claim/store_write_test.go
internal/witness/claim/store_link_test.go
internal/witness/claim/store_mismatch_test.go
docs/factory/claims-evidence.md
docs/close-reports/ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01.md
```

## API Added

### Types

```go
type ClaimID string
type EvidenceID string
type ClaimStatus string
type EvidenceKind string
type EvidenceRole string
type Verdict string
```

### Constants

```go
const ClaimSchemaVersion = "leamas.claim.v1"
const EvidenceSchemaVersion = "leamas.evidence.v1"
```

### Constructors

```go
func NewClaim(id ClaimID, runID runbundle.RunID, statement string, now time.Time) (Claim, error)
func NewEvidence(id EvidenceID, runID runbundle.RunID, kind EvidenceKind, role EvidenceRole, title string, now time.Time) (Evidence, error)
```

### Validation

```go
func ValidateClaimID(id ClaimID) error
func ValidateEvidenceID(id EvidenceID) error
func ValidateRelativePath(path string) error
```

### JSON

```go
func MarshalClaimJSON(c Claim) ([]byte, error)
func MarshalEvidenceJSON(e Evidence) ([]byte, error)
func StrictDecodeClaim(data []byte) (*Claim, error)
func StrictDecodeEvidence(data []byte) (*Evidence, error)
```

### Store

```go
type Store struct { Bundle runbundle.Bundle }
func NewStore(bundle runbundle.Bundle) Store
func (s Store) WriteClaim(c Claim) error
func (s Store) ReadClaim(id ClaimID) (Claim, error)
func (s Store) WriteEvidence(e Evidence) error
func (s Store) ReadEvidence(id EvidenceID) (Evidence, error)
func (s Store) AddEvidenceToClaim(claimID ClaimID, evidenceID EvidenceID, now func() time.Time) (Claim, error)
```

## Schemas Added

### Claim Schema

```json
{
  "schema_version": "leamas.claim.v1",
  "id": "claim-<id>",
  "run_id": "run-<id>",
  "created_at": "RFC3339",
  "updated_at": "RFC3339",
  "statement": "...",
  "status": "open|supported|rejected|unknown",
  "verdict": "unreviewed|pass|fail|mixed",
  "evidence_ids": [],
  "notes": ""
}
```

### Evidence Schema

```json
{
  "schema_version": "leamas.evidence.v1",
  "id": "evidence-<id>",
  "run_id": "run-<id>",
  "created_at": "RFC3339",
  "kind": "command_output|digest|log|file|trace|verifier_result",
  "role": "primary|supporting|contradicting|context",
  "title": "...",
  "relative_path": "",
  "summary": "",
  "metadata": {}
}
```

## Filesystem Contract

Claims: `claims/<claim-id>.json`
Evidence: `evidence/<evidence-id>.json`

## Tests Added

41 tests covering ID validation, claim/evidence constructors, JSON round-trip, strict decode, store operations, and tamper detection.

## Verification Commands and Results

```bash
go test ./internal/witness/claim/... -count=1
go test ./...
go vet ./...
make factorize
make gate
```

Results: All tests pass. All gates pass.

## Skipped / Deferred

- No CLI commands added
- No claim evaluation engine
- No witness proxy persistence wiring
- No cockpit UI

## Hard Stops Honored

- No Python
- No shell verifier logic
- No Node/Vite/React/npm/yarn/pnpm
- No database imports
- No network imports
- No cockpit imports
- No witness proxy imports

## Follow-up Candidates

| Priority | ACT | Description |
|----------|-----|-------------|
| P1 | `ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-CLI01` | Add CLI to create claims, attach evidence |
| P1 | `ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01` | CLI to inspect witness proxy captures |

## Suggested Commit

```bash
git add internal/witness/claim \
        docs/factory/claims-evidence.md \
        docs/close-reports/ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01.md

git commit -m "ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01 add claim evidence seed"
git push
```
