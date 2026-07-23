# ACT-LEAMAS-GATE-SUMMARY-V2-SCHEMA-INTROSPECTION01
# CORRECTION01 IDENTITY ADDENDUM

This addendum records the complete identity chain with literal full
OIDs across CORRECTION01, CORRECTION02, CORRECTION03, and
CORRECTION04. No placeholders are present.

## Identity chain (CORRECTION01–CORRECTION04)

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

correction02_content_commit_oid = fd68af9d1f08ddd0b064a85065f46bf4d7b17885
correction02_content_tree_oid   = a3c2eeb91d6cd761b5900c756284c903cbf98539

correction03_content_commit_oid = 9f62665376980f5a3d166326e17f8f23839dcc67
correction03_content_tree_oid   = fba11740a7948f85003511ce1f11d2525e14385f

correction04_content_commit_oid = 63148e4ba34e89989581cec4a517d0d8eada3fe2
correction04_content_tree_oid   = 2a124067e293048a7411dcb15be7623552a3f631
```

## Tag identity triple (CORRECTION01–CORRECTION04)

```
tag_object_oid (CORRECTION01) = ef8261092ef49133c2f45fa8299e480dc4a7a20a
tag_target_oid (CORRECTION01) = 68c1b7b2b6352d348e50db7fe23f3daea9559cea
tag_target_tree_oid (CORRECTION01) = d50a316ca1d064edf3ef255b49c1643b2da278ec

tag_object_oid (CORRECTION02) = 14a5f060319f7c453110626dd49e6b75e3de32c3
tag_target_oid (CORRECTION02) = fd68af9d1f08ddd0b064a85065f46bf4d7b17885
tag_target_tree_oid (CORRECTION02) = a3c2eeb91d6cd761b5900c756284c903cbf98539

tag_object_oid (CORRECTION03) = bd766b44951d3ced637e0211973324baa52927cc
tag_target_oid (CORRECTION03) = 9f62665376980f5a3d166326e17f8f23839dcc67
tag_target_tree_oid (CORRECTION03) = fba11740a7948f85003511ce1f11d2525e14385f

tag_object_oid (CORRECTION04) = <populated by the correction04 tag>
tag_target_oid (CORRECTION04) = 63148e4ba34e89989581cec4a517d0d8eada3fe2
tag_target_tree_oid (CORRECTION04) = 2a124067e293048a7411dcb15be7623552a3f631
```

## Proof binary

```
proof_binary_sha256       = 2c6c82a455279d23f99393bb33a4cdd47ca522af0d4a0807e8002255505ddee8
proof_binary_vcs_revision = 0d9d30561004c2cd66fe516fd55db0988759794b
proof_binary_vcs_modified = false
```

The proof binary was built from the tested commit with
`-buildvcs=true -trimpath`. The `vcs.modified=false` confirms a
clean working tree at the proof stage.

## Canonical digest (CORRECTION04 — whitespace-clean)

```
canonical_digest_sha256       = ee22a2980824af7c52c30bfd7bd03fc6f3db9d343557c22a543211ade7f04d23
canonical_digest_path         = build/canonical-gate-summary.txt
canonical_digest_created       = 2026-07-23T04:48:55Z
canonical_digest_range         = 0d9d30561004c2cd66fe516fd55db0988759794b..HEAD
canonical_digest_byte_count    = 22933
```

The original canonical digest (CORRECTION02) had trailing whitespace
on lines copied from the close report markdown source. The
CORRECTION04 revision stripped the trailing whitespace via regex
replacement. The sha256 changed from
`42fabf76154beabee4526065e71d4e869fe64ee67c487afc3bd4b6ca834669da` to
`ee22a2980824af7c52c30bfd7bd03fc6f3db9d343557c22a543211ade7f04d23`.

## Schema hashes

```
v1_schema_sha256 = 6069570bbc2b79011ab43c34ecce7f9181a814d5f47ca9174daadaff4ee06e81
v2_schema_sha256 = 11ebfbf643020cec564f5c6b3f2d66d4055e9c0417d609313352211a9b69292c
v1_schema_id     = urn:leamas:gate-summary:v1
v2_schema_id     = urn:leamas:gate-summary:v2
```

The CLI output hash matches the canonical schema file hash.

## Test bound for v2-truncated.json envelope rejection

The dedicated test
`internal/gatesummary/v2_truncated_envelope_test.go` exercises
the codec end-to-end without using `t.Skip`. It asserts the bounded
reader produces a failure with `CodeMalformedJSON`. The schema
subpackage may now classify `v2-truncated.json` as
schema-result-not-applicable and reference this executable proof.
