# ACT-LEAMAS-GATE-SUMMARY-V2-SCHEMA-INTROSPECTION01 (CORRECTION01 identity addendum)

## Identity chain (CORRECTION01 reconciliation)

The identity chain is recorded with literal full OIDs (no
placeholders). The chain is recorded in the order requested by
the ACT.

```
baseline_commit_oid       = dff6f847000130f66a8d950da667c4924a818a9f
baseline_tree_oid         = b89356a429d5558ccf769cd18a4c3cc61dc8be6f

implementation_commit_oid = 0d9d30561004c2cd66fe516fd55db0988759794b
implementation_tree_oid   = 7e40b24b05b16946334fff9bc82fc97a0d4e2aae

tested_commit_oid         = 0d9d30561004c2cd66fe516fd55db0988759794b
tested_tree_oid           = 7e40b24b05b16946334fff9bc82fc97a0d4e2aae

evidence_commit_oid       = bd13513908c784f82ae26e0e9adc787dd2584aff
evidence_tree_oid         = 7e40b24b05b16946334fff9bc82fc97a0d4e2aae

close_commit_oid          = 0aac7c074764c45051c2b73e32b4c0ce545dfc84
close_tree_oid            = 9d8199868aa493402ddb6965675f5193a8f08cd8

tag_object_oid            = f37190a80735ab60ebebb57be24bfd81f51b0c71
tag_target_oid            = bd13513908c784f82ae26e0e9adc787dd2584aff
tag_target_tree_oid       = 7e40b24b05b16946334fff9bc82fc97a0d4e2aae
```

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

A fresh targeted digest was generated from the clean tested tree
on 2026-07-23T04:32:25Z covering the range
`dff6f847000130f66a8d950da667c4924a818a9f..HEAD`. The digest is
preserved at `build/fresh-gate-summary.txt` and
`build/fresh-gate-summary-LSATFORINTROSPECTION01.txt`.

The digest header is:

```
LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3
LEAMAS_VERSION: 0.1.0
LEAMAS_COMMIT: unknown
LEAMAS_BUILD_TIME: unknown
DIGEST_MODE: range
DIGEST_CREATED_AT: 2026-07-23T04:32:25Z
```

The digest sha256 is `786b26408fbb3464fbc3c61d172e31ce13df6fc0bbb16dee03d9339672bbb69e`.

The digest proves the schema-introspection implementation is
present in the tested tree (commit `0d9d305`) and was merged to
HEAD. The digest contains the full implementation diff:

```
A  cmd/leamas/gate_summary.go
A  cmd/leamas/gate_summary_schema.go
A  cmd/leamas/gate_summary_schema_failure_test.go
A  cmd/leamas/gate_summary_schema_subprocess_test.go
A  cmd/leamas/gate_summary_schema_test.go
```

plus the embedded schema subpackage and the documentation.
