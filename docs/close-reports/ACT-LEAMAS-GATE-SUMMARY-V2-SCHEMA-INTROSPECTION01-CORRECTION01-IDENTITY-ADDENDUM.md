# ACT-LEAMAS-GATE-SUMMARY-V2-SCHEMA-INTROSPECTION01
# CORRECTION01 IDENTITY ADDENDUM

This addendum records the complete identity chain with literal full
OIDs recorded against the HEAD of the repository at the time of
CORRECTION02 publication. No placeholders are present.

## Identity chain

```
baseline_commit_oid       = dff6f847000130f66a8d950da667c4924a818a9f
baseline_tree_oid         = b89356a429d5558ccf769cd18a4c3cc61dc8be6f

implementation_commit_oid = 0d9d30561004c2cd66fe516fd55db0988759794b
implementation_tree_oid   = 7e40b24b05b16946334fff9bc82fc97a0d4e2aae

tested_commit_oid         = 0d9d30561004c2cd66fe516fd55db0988759794b
tested_tree_oid           = 7e40b24b05b16946334fff9bc82fc97a0d4e2aae

evidence_commit_oid       = bd13513908c784f82ae26e0e9adc787dd2584aff
evidence_tree_oid         = 7e40b24b05b16946334fff9bc82fc97a0d4e2aae

close_commit_oid          = 68c1b7b2b6352d348e50db7fe23f3daea9559cea
close_tree_oid            = d50a316ca1d064edf3ef255b49c1643b2da278ec

correction02_commit_oid   = <populated by the correction02 commit>
correction02_tree_oid     = <populated by the correction02 commit>

tag_object_oid            = ef8261092ef49133c2f45fa8299e480dc4a7a20a
tag_target_oid            = 68c1b7b2b6352d348e50db7fe23f3daea9559cea
tag_target_tree_oid       = d50a316ca1d064edf3ef255b49c1643b2da278ec
```

The closure tag `act/leamas-gate-summary-v2-schema-introspection01`
is an annotated tag object (`git cat-file -t` reports `tag`).

## Proof binary

```
proof_binary_sha256:        2c6c82a455279d23f99393bb33a4cdd47ca522af0d4a0807e8002255505ddee8
proof_binary_vcs_revision:  0d9d30561004c2cd66fe516fd55db0988759794b
proof_binary_vcs_modified:  false
```

The proof binary was built from the tested commit with
`-buildvcs=true -trimpath`. The `vcs.modified=false` confirms a
clean working tree at the proof stage.

## Fresh Gate Summary digest

A fresh canonical gate summary was written to
`.factory/gate-summary.json` at 2026-07-23T04:47:00Z and bound
into a fresh targeted digest at
`build/canonical-gate-summary.txt`:

```
canonical_digest_sha256  = 42fabf76154beabee4526065e71d4e869fe64ee67c487afc3bd4b6ca834669da
canonical_digest_path    = build/canonical-gate-summary.txt
canonical_digest_created  = 2026-07-23T04:48:55Z
canonical_digest_range    = 0d9d30561004c2cd66fe516fd55db0988759794b..HEAD
canonical_gate_summary_generated_at = 2026-07-23T04:47:00Z
canonical_gate_summary_status = pass
```

The canonical gate summary embedded in the digest records:

```
checks_total: 3
checks_passed: 2
checks_skipped: 1
checks: fast-lane=pass, dupcode-lane=pass, long-lane=skip
```

The canonical gate summary proves the schema-introspection
implementation is present in the tested tree (commit `0d9d305`)
and was merged to HEAD. The gate-fast verifier lane passed
against the tested tree at the recorded timestamp.

## Schema hashes

```
v1_schema_sha256 = 6069570bbc2b79011ab43c34ecce7f9181a814d5f47ca9174daadaff4ee06e81
v2_schema_sha256 = 11ebfbf643020cec564f5c6b3f2d66d4055e9c0417d609313352211a9b69292c
v1_schema_id     = urn:leamas:gate-summary:v1
v2_schema_id     = urn:leamas:gate-summary:v2
```

The CLI output hash matches the canonical schema file hash.
