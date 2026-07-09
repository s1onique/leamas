# Factory: Digest Evidence Hashes

**ACT**: ACT-LEAMAS-FACTORY-DIGEST-EVIDENCE-HASHES01

## Purpose

The `EVIDENCE_HASHES` section provides deterministic SHA-256 fingerprints over normalized digest sections. This answers the question "What exact evidence did we review?" with stable content hashes.

## Design Goals

1. **Determinism**: Same digest content produces same hashes across runs
2. **Normalization**: Input normalization ensures consistent hashes despite whitespace differences
3. **Isolation**: Each section hashed independently for granular tracking
4. **Composability**: `digest_evidence_sha256` aggregates all section hashes

## Hash Algorithm

Uses Go's standard `crypto/sha256` package (SHA-256, 256-bit output).

## Normalization Rules

Before hashing, input is normalized:

1. **CRLF → LF**: Convert all `\r\n` to `\n`
2. **Trailing newline**: Ensure exactly one trailing newline
3. **Volatile exclusion**: These fields are excluded from hash input:
   - `DIGEST_CREATED_AT` (timestamps vary)
   - Absolute repository paths (machine-specific)

## Section Hashes

Each section gets its own SHA-256 hash:

| Field | Source |
|-------|--------|
| `changeset_manifest_sha256` | Content of `## CHANGESET_MANIFEST` section |
| `changeset_stats_sha256` | Content of `## CHANGESET_STATS` section |
| `review_map_sha256` | Content of `## REVIEW_MAP` section |
| `risk_signals_sha256` | Content of `## RISK_SIGNALS` section |
| `patch_hygiene_sha256` | Content of `## PATCH_HYGIENE` section |
| `file_evidence_sha256` | Combined `## Changed files` + `## Diffs` sections |

## Digest Evidence Hash

`digest_evidence_sha256` is computed by concatenating all section hashes in order, then hashing:

```
digest_evidence = SHA256(
  changeset_manifest_sha256 +
  changeset_stats_sha256 +
  review_map_sha256 +
  risk_signals_sha256 +
  patch_hygiene_sha256 +
  file_evidence_sha256
)
```

**Note**: `EVIDENCE_HASHES` section does NOT include itself in the `digest_evidence` computation (avoids circular dependency).

## Key Order (Stable)

```
hash_algorithm
hash_scope
changeset_manifest_sha256
changeset_stats_sha256
review_map_sha256
risk_signals_sha256
patch_hygiene_sha256
file_evidence_sha256
digest_evidence_sha256
```

## Example

```
## EVIDENCE_HASHES
hash_algorithm=SHA-256
hash_scope=section
changeset_manifest_sha256=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
changeset_stats_sha256=a591d800c7b4a1e5888ddabfe2bc528c0c40e8d6d2d1e8a7f1b8a4c5d6e7f8a
review_map_sha256=b7a2c9d0e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b
risk_signals_sha256=c8b3d0e1f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c
patch_hygiene_sha256=d9c4e1f2a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d
file_evidence_sha256=eabcd2f3b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e
digest_evidence_sha256=fbcde3a4c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f
```

## Position in Digest

`EVIDENCE_HASHES` appears after `PATCH_HYGIENE` and before `## Changed files`:

```
## PATCH_HYGIENE
...

## EVIDENCE_HASHES
...

## Changed files
...
```

## Implementation

- **Source**: `internal/factory/digest/evidence_hashes.go`
- **Tests**: `internal/factory/digest/evidence_hashes_test.go`
- **Integration**: `internal/factory/digest/evidence_hashes_integration_test.go`

## Verification

```bash
# Run evidence hash tests
go test ./internal/factory/digest/... -run EvidenceHashes -v

# Generate digest and verify EVIDENCE_HASHES section
go build -o bin/leamas ./cmd/leamas
./bin/leamas factory digest --dirty --output /tmp/digest.txt
grep -A 10 "EVIDENCE_HASHES" /tmp/digest.txt

# Verify hash stability (run twice, hashes should match)
./bin/leamas factory digest --dirty --output /tmp/digest1.txt
./bin/leamas factory digest --dirty --output /tmp/digest2.txt
grep "digest_evidence_sha256" /tmp/digest1.txt
grep "digest_evidence_sha256" /tmp/digest2.txt
```

## Related

- [Digest Contract](./digest-contract.md)
- [Digest Documentation](./digest.md)
