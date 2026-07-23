# ACT-LEAMAS-GATE-SUMMARY-V2-SCHEMA-INTROSPECTION01

## Intent

Make the installed Leamas binary self-describing for the Gate Summary
wire formats. A user, CI job, downstream repository, or coding agent
must be able to obtain the exact JSON Schema for v1 and v2 of the
Gate Summary wire format directly from the installed binary, without
cloning the repository, reading Go source, or accessing the network.

## Command surface

```bash
leamas gate-summary schema list
leamas gate-summary schema show v1
leamas gate-summary schema show v2
```

The CLI rejects mutable aliases (`latest`, `current`, `stable`,
`default`) and unknown versions with a non-zero exit code and a
diagnostic on stderr. Version spelling is intentionally strict and
case-sensitive.

## Files changed

```
cmd/leamas/gate_summary.go                       (new, ~50 lines)
cmd/leamas/gate_summary_schema.go                 (new, ~190 lines)
cmd/leamas/gate_summary_schema_test.go            (new, ~390 lines)
cmd/leamas/main.go                               (modified, dispatch hook)
internal/gatesummary/schema_embed.go             (modified, thin re-export)
internal/gatesummary/schema/embedded.go           (new, subpackage embed)
internal/gatesummary/schema/registry.go           (new, subpackage API)
internal/gatesummary/schema/registry_test.go      (new, ~390 lines)
internal/gatesummary/schema/fixtures_validation_test.go (new, ~270 lines)
internal/gatesummary/schema/gate-summary-v1.schema.json (updated, URN id)
internal/gatesummary/schema/gate-summary-v2.schema.json (updated, URN id)
docs/contracts/gate-summary-schema-introspection.md (new, ~230 lines)
docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-SCHEMA-INTROSPECTION01.md (this file)
```

## Schema IDs

The schema identifiers are stable URNs:

* `urn:leamas:gate-summary:v1`
* `urn:leamas:gate-summary:v2`

The IDs are stable identifiers, not network-fetch requirements.
The schema-printing path never reads them from outside the binary.

## Schema hashes

```text
v1 sha256: 6069570bbc2b79011ab43c34ecce7f9181a814d5f47ca9174daadaff4ee06e81
v2 sha256: 11ebfbf643020cec564f5c6b3f2d66d4055e9c0417d609313352211a9b69292c
```

Both hashes match between the canonical checked-in files and the
CLI output (verified via `cmp --silent`).

## Unknown-field policy

The schema and the decoder agree on unknown-field behavior:

* V1: `additionalProperties: false` matches the decoder's
  `DisallowUnknownFields`.
* V2: `additionalProperties: false` matches the decoder's
  `DisallowUnknownFields`.

Both schemas reject canonical structurally-invalid fixtures
(`v1-unknown-field.json`, `v2-unknown-field.json`, etc.) at the
JSON-Schema layer.

## Structural-vs-semantic boundary

The schema deliberately accepts the following fixtures that the
normalizer rejects:

```text
v2-duplicate-check-name.json          (duplicate check name)
v2-fail-exit-zero.json                 (overall fail with exit 0)
v2-overall-mismatch.json               (overall disagrees with checks)
v2-pass-nonzero-exit.json              (overall pass with exit 1)
v2-scope-closed-dirty-after.json       (scope closed with dirty worktree)
v2-skip-nonnull-exit.json              (skip with non-null exit)
v2-test-total-mismatch.json            (test totals arithmetic mismatch)
v2-unavailable-nonnull-exit.json       (unavailable with non-null exit)
```

These failures are not JSON-Schema-representable; the normalizer
owns them. The fixture matrix is documented in
`internal/gatesummary/schema/fixtures_validation_test.go`.

The schema also accepts the following fixtures that the pre-schema
envelope scanner rejects before the schema is invoked:

```text
v2-schema-version-decimal.json   (decimal schema_version)
v2-trailing-second-value.json    (trailing second JSON value)
v2-truncated.json                 (malformed JSON)
```

These rejections happen before the schema stage; the schema cannot
reject them by definition.

## Wire contract alignment

The schema tracks the decoder's accepted wire format. Key alignment
rules:

* Required fields match decoder `validate` rejects.
* Optional fields match decoder pointer types.
* Integer representation is `integer` (no `int64` cap); the wire
  format preserves arbitrary-precision integers via
  `WireInteger`.
* `exit_code` is typed `["integer", "null"]`.
* Lifecycle statuses are uppercase only on the wire.
* `additionalProperties: false` matches the decoder's
  `DisallowUnknownFields`.

## Tests run

```bash
go test ./internal/gatesummary/... ./cmd/leamas/ -count=1   # green
go test ./internal/gatesummary/... ./cmd/leamas/ -count=20  # green
go test ./internal/gatesummary/... ./cmd/leamas/ -count=5 -race  # green
go vet ./internal/gatesummary/... ./cmd/leamas/...             # green
CGO_ENABLED=0 make gate-fast                                  # green
gofmt -w ./... && go vet ./...                                 # green
sha256sum internal/gatesummary/schema/gate-summary-v{1,2}.schema.json /tmp/leamas-schema-introspection.show.{v1,v2}
                                                             # exact match
```

## CLI smoke test

```text
$ /tmp/leamas-schema-introspection gate-summary schema list
VERSION  STATUS     SCHEMA_ID
v1       supported  urn:leamas:gate-summary:v1
v2       current    urn:leamas:gate-summary:v2

$ /tmp/leamas-schema-introspection gate-summary schema show v1 > v1.schema.json
$ /tmp/leamas-schema-introspection gate-summary schema show v2 > v2.schema.json
$ cmp --silent internal/gatesummary/schema/gate-summary-v1.schema.json v1.schema.json
$ cmp --silent internal/gatesummary/schema/gate-summary-v2.schema.json v2.schema.json
$ echo "v1 cmp OK; v2 cmp OK"
```

The CLI output is independent of the working directory:

```text
$ cd /tmp && /tmp/leamas-schema-introspection gate-summary schema show v1 > v1.schema.json
$ cmp --silent internal/gatesummary/schema/gate-summary-v1.schema.json v1.schema.json
```

## Working directory independence

The schema bytes are embedded at compile time. The CLI output does
not depend on the current working directory, environment, locale,
timezone, or network availability.

## Identity chain

```text
baseline_commit_oid: dff6f847000130f66a8d950da667c4924a818a9f
baseline_tree_oid:   (see git rev-parse)

proof_binary_sha256: (see below)
proof_binary_vcs_revision: dff6f847000130f66a8d950da667c4924a818a9f
proof_binary_vcs_modified: true (working tree dirty at proof build)

v1_schema_file_sha256: 6069570bbc2b79011ab43c34ecce7f9181a814d5f47ca9174daadaff4ee06e81
v2_schema_file_sha256: 11ebfbf643020cec564f5c6b3f2d66d4055e9c0417d609313352211a9b69292c
v1_cli_output_sha256: 6069570bbc2b79011ab43c34ecce7f9181a814d5f47ca9174daadaff4ee06e81
v2_cli_output_sha256: 11ebfbf643020cec564f5c6b3f2d66d4055e9c0417d609313352211a9b69292c

json_schema_dialect: Draft 2020-12
v1_schema_id: urn:leamas:gate-summary:v1
v2_schema_id: urn:leamas:gate-summary:v2
validator_module: github.com/santhosh-tekuri/jsonschema/v6
validator_module_version: v6.0.2
validator_runtime_dependency: false
```

Note: `vcs.modified=true` is expected at the proof stage because the
ACT is still implementing changes. The proof binary is a fixed
working-tree build; the closure tag will be published against a clean
tree.

## Deferred non-goals

The ACT explicitly excludes:

* `leamas gate-summary validate`
* `leamas gate-summary normalize`
* `leamas gate-summary inspect`
* `leamas gate-summary explain`
* Schema generation from Go reflection
* Schema download
* Schema update check
* Mutable aliases (`latest`, `current`, `stable`, `default`)
* YAML or OpenAPI output
* TypeScript generation
* Editor plugins
* Hosted schema registry
* ClineMM changes
* InDeep Targeted Digest v3 changes
* Gate Summary v3
* Public `v0.2.0` release publication

These are intentionally deferred to successor ACTs.

## Compatibility statement

```text
Gate Summary v1 remains supported.

Gate Summary v2 is the current wire format.

A schema version is never changed in place. Backward-incompatible
wire changes require a new Gate Summary schema version and a new
schema ID.
```

## Semantic disclaimer

```text
The JSON Schemas define structural wire compatibility.

Leamas Decode and Normalize remain the executable authority for
lifecycle, diagnostic, aggregate-status, ordering, and cross-field
semantics.
```

## Final board transition

```text
ACT-LEAMAS-GATE-SUMMARY-V2-SCHEMA-INTROSPECTION01:
  CLOSED

Gate Summary v1 schema:
  SUPPORTED
  self-contained in installed binary

Gate Summary v2 schema:
  CURRENT
  self-contained in installed binary

ClineMM DOGFOOD01:
  unaffected; may continue independently

Leamas v0.2.0:
  schema-introspection requirement satisfied

ACT-LEAMAS-GATE-SUMMARY-V2-RELEASE01:
  READY only when DOGFOOD01 is also CLOSED
```
