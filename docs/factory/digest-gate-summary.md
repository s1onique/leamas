# Factory: Digest GATE_SUMMARY Section

**ACT**: ACT-LEAMAS-FACTORY-DIGEST-GATE-SUMMARY01

## Purpose

The `GATE_SUMMARY` section provides evidence of gate verification results without running gates during digest generation. It answers "Did the gate pass?" with structured check results.

## Design Goals

1. **Read-only**: Digest reads pre-existing gate summary, never runs gates
2. **Fast**: Digest generation stays fast by avoiding expensive checks
3. **Graceful degradation**: Missing/invalid summary renders as unavailable
4. **Deterministic**: Check rows sorted by name for stable output

## Gate Summary Artifact

Location: `.factory/gate-summary.json`

Schema v1:

```json
{
  "schema_version": 1,
  "generated_at": "2026-07-09T21:10:00Z",
  "tool": "leamas factory gate",
  "overall_status": "pass",
  "checks": [
    {
      "name": "gate",
      "status": "pass",
      "duration_ms": 1234,
      "evidence": "leamas factory gate"
    }
  ]
}
```

### Check Status Values

| Status | Meaning |
|--------|---------|
| `pass` | Check passed |
| `fail` | Check failed |
| `skip` | Check was skipped |
| `unavailable` | Check result not available |

## GATE_SUMMARY Section Format

### Present Case

```
## GATE_SUMMARY
source=.factory/gate-summary.json
source_status=present
schema_version=1
generated_at=2026-07-09T21:10:00Z
overall_status=pass
checks_total=1
checks_passed=1
checks_failed=0
checks_skipped=0
checks_unavailable=0
checks:
  - name=gate status=pass duration_ms=1234 evidence=leamas factory gate
```

### Missing Case

```
## GATE_SUMMARY
source=.factory/gate-summary.json
source_status=missing
schema_version=0
generated_at=
overall_status=unavailable
checks_total=0
checks_passed=0
checks_failed=0
checks_skipped=0
checks_unavailable=0
```

## Position in Digest

`GATE_SUMMARY` appears after `EVIDENCE_HASHES` and before `## Changed files`:

```
## EVIDENCE_HASHES
...

## GATE_SUMMARY
...

## Changed files
...
```

## Commands

Run the quality gate and regenerate the canonical observed-result artifact:

```bash
make gate
```

Read and copy an existing artifact without recursively running the gate:

```bash
leamas factory gate-summary --output .factory/gate-summary.json
```

The literal gate writes one aggregate `gate` check after execution. A missing
source remains `unavailable`; the read/copy command never invents a passing
result. The default output is `.factory/gate-summary.json`.

## Implementation

- **Artifact**: `internal/factory/gate/summary.go`, `run_summary.go`
- **Tests**: `internal/factory/gate/summary_test.go`
- **Digest**: `internal/factory/digest/digest.go` (reads artifact)
- **CLI**: `cmd/leamas/factory.go` (handles gate-summary command)

## Recorded Check

| Name | Command | Description |
|------|---------|-------------|
| `gate` | `leamas factory gate` | Aggregate result and elapsed duration of the literal gate run |

The gate's normal output retains the individual verifier and Go-toolchain
results. The summary check is deliberately aggregate so generation does not
rerun or duplicate any check.

## Verification

```bash
# Run the gate and generate the observed summary
make gate

# Verify artifact
cat .factory/gate-summary.json

# Generate digest with GATE_SUMMARY
./bin/leamas factory digest --dirty --output /tmp/digest.txt
grep -A 15 "GATE_SUMMARY" /tmp/digest.txt

# Run tests
go test ./...

# Full verification
make factorize
make gate
```

## Evidence Hash Participation

The `GATE_SUMMARY` section is included in `digest_evidence_sha256`:

```
digest_evidence_sha256 = SHA256(
  changeset_manifest_sha256 +
  changeset_stats_sha256 +
  review_map_sha256 +
  risk_signals_sha256 +
  patch_hygiene_sha256 +
  gate_summary_sha256 +
  file_evidence_sha256
)
```

This means any change to gate results will change the digest evidence hash.

## Related

- [Digest Contract](./digest-contract.md)
- [Digest Evidence Hashes](./digest-evidence-hashes.md)
- [Gate Package](../gate/)
