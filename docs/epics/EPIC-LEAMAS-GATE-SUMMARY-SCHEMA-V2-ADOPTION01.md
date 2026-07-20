# EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01

## Epic
Gate Summary Schema v2 Adoption

## Motivation

The gate summary schema v2 introduces structural extensions (scope, parent, execution, worktree) that require schema migration and backward compatibility with v1 consumers.

## Goals

1. **Schema migration**: Decode both v1 and v2 wire formats
2. **Semantic validation**: Reject invalid documents with actionable diagnostics
3. **Backward compatibility**: v1 consumers see equivalent v1-shaped output
4. **Forward compatibility**: Unknown fields are ignored, not rejected
5. **Digest integration**: Normalized summaries participate in digest computation

## Constraints

- No database, OAuth/OIDC, or RBAC
- Local-first, web-first, single binary
- 100% Go, no Python or shell for product code
- Verifiers in Go, bash only for glue

## Scope

- Gate Summary v2 wire format (JSON)
- Version projection (v1 vs v2)
- Semantic validation (check names, exit codes, test totals, overall status)
- CLI surface for reading gate summaries
- Digest integration

## Out of Scope

- Gateway or model control plane
- Production deployment automation
- Multi-tenant support

## Success Criteria

- All 41 corpus fixtures pass decode+normalize
- Semantic invalid fixtures are rejected with actionable diagnostics
- Diagnostic ordering is deterministic
- No race conditions in concurrent normalization
- Digest integration produces stable hashes

## Status
IN PROGRESS

## Milestones

### M1: Schema Contract (v1 stable, v2 draft)
- [x] Gate summary schema v1 documented
- [x] Gate summary schema v2 documented
- [x] Forward-compatibility model defined
- [x] Test corpus created (41 fixtures)

### M2: Decoder (v1 stable, v2 functional)
- [x] v1 decoder implemented
- [x] v2 decoder implemented
- [x] Forward-compatibility handling
- [x] Duplicate key detection
- [x] Error recovery and partial decode
- [x] Diagnostic codes and messages
- [x] CLI decoder command

### M3: Normalization (v1 stable, v2 stable)
- [x] Semantic validation for v2
- [x] Version projection (v1 vs v2)
- [x] Diagnostic ordering (deterministic)
- [x] Canonical internal model
- [x] Concurrency safety
- [x] Aliasing safety

### M4: Digest Integration (v1, v2)
- [ ] Normalized summaries participate in digest
- [ ] Stable hash computation
- [ ] Digest range and delta

### M5: CLI Surface (v1, v2)
- [ ] Read gate summary from file
- [ ] Read gate summary from stdin
- [ ] Format options (json, table, summary)
- [ ] Filter by status
- [ ] Validate-only mode

### M6: Conformance Testing
- [ ] Golden fixtures
- [ ] Fuzz testing
- [ ] Edge cases
- [ ] Performance benchmarks

## ACT Inventory

### Closed ACTs

| ACT | Status | Notes |
|-----|--------|-------|
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01` | CLOSED | Schema contract frozen |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01` | CLOSED | Wire vocabulary clarified |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02` | CLOSED (PARTIAL — superseded) | Validator proof accepted; reader-contract semantics superseded by CORRECTION03 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03` | CLOSED (PARTIAL) | Reader contract frozen and committed; `DECODER01` unblocked |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` | CLOSED (PARTIAL — retained unrelated baseline failures) | Strict v1/v2 decoder plus forward P0 fix committed; contract review accepted |
| `ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01` | CLOSED | Semantic normalization pipeline implemented; 41 corpus fixtures pass; deterministic diagnostics |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03-R1` | CLOSED | Reader contract reviewed and accepted |

### Ready ACTs

| ACT | Status | Notes |
|-----|--------|-------|
| `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` | READY | Digest integration |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CLI01` | READY | CLI surface |

### Pending ACTs

| ACT | Status | Notes |
|-----|--------|-------|
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` | PENDING | Golden + fuzz |

## Related

- [ADR-0006: Filesystem Run Bundles](./adr/0006-filesystem-run-bundles.md)
- [Verification Witness](./docs/doctrine/verification-witness.md)
- [Gate Summary Schema v1](./docs/factory/gate-summary-schema-v1.md)
- [Gate Summary Schema v2](./docs/factory/gate-summary-schema-v2.md)
