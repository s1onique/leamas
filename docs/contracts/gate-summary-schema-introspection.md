# Gate Summary Schema Introspection

The Leamas binary exposes the canonical Gate Summary v1 and v2 JSON
Schemas through a small, offline command surface. Downstream consumers
(CI jobs, editors, coding agents, validators) can obtain the exact
embedded schemas without cloning the repository, reading Go source,
or accessing the network.

## Command surface

```bash
leamas gate-summary schema list
leamas gate-summary schema show v1
leamas gate-summary schema show v2
```

Help surfaces:

```bash
leamas gate-summary --help
leamas gate-summary schema --help
leamas gate-summary schema show --help
```

### `schema list`

Prints the supported versions and their status. The output is a
fixed-column table:

```text
VERSION  STATUS     SCHEMA_ID
v1       supported  urn:leamas:gate-summary:v1
v2       current    urn:leamas:gate-summary:v2
```

Rules:

* stdout contains the table only;
* stderr is empty on success;
* versions are sorted lexicographically (v1, v2);
* `v1` remains listed as supported;
* `v2` is listed as current;
* output ends with exactly one LF.

The status value is descriptive CLI metadata. It is not part of
either JSON Schema.

### `schema show`

Prints the exact embedded bytes of the named schema:

```bash
leamas gate-summary schema show v1
leamas gate-summary schema show v2
```

Rules:

* stdout contains JSON Schema bytes only;
* no heading, explanation, logging, color, or shell prompt is
  written to stdout;
* stderr is empty on success;
* output ends with exactly one LF;
* output is byte-identical on every invocation;
* output is byte-identical to the canonical checked-in schema file;
* no runtime JSON marshalling or formatting is performed;
* no runtime filesystem lookup is performed;
* no network access is performed;
* unknown versions fail with a non-zero exit code;
* output write failures fail closed;
* diagnostics go to stderr only.

The command rejects mutable aliases such as `latest`, `current`,
`stable`, and `default`. Version spelling is intentionally strict and
case-sensitive.

The following invocations must fail:

```bash
leamas gate-summary schema show
leamas gate-summary schema show v3
leamas gate-summary schema show current
leamas gate-summary schema show V2
leamas gate-summary schema show 2
leamas gate-summary schema unknown v2
```

## Supported versions

| Version | Status    | Schema ID                       |
|---------|-----------|---------------------------------|
| v1      | supported | `urn:leamas:gate-summary:v1`    |
| v2      | current   | `urn:leamas:gate-summary:v2`    |

Gate Summary v1 remains supported. Gate Summary v2 is the current
wire format.

A schema version is never changed in place. Backward-incompatible
wire changes require a new Gate Summary schema version and a new
schema ID.

## Wire schema versus semantic contract

The emitted JSON Schemas define structural wire compatibility. Leamas
Decode and Normalize remain the executable authority for lifecycle,
diagnostic, aggregate-status, ordering, and cross-field semantics.

The following constraints are intentionally executable-only and are
not encoded in the schema:

* contradictory lifecycle combinations;
* aggregate status derivation;
* ownership rules;
* diagnostic precedence;
* totals consistency;
* cross-field arithmetic;
* scope/parent/overall interpretation;
* normalization failure classification.

The schema follows the decoder. The decoder is the authoritative
source of the contract.

## Exact examples

```bash
# List supported versions
leamas gate-summary schema list

# Print the v1 schema to stdout
leamas gate-summary schema show v1

# Capture the v2 schema into a file for IDE integration
leamas gate-summary schema show v2 > gate-summary-v2.schema.json

# Use the schema with an external validator
leamas gate-summary schema show v2 | jsonschema -i summary.json -
```

## Offline behavior

The command surface is fully offline:

* no network access;
* no filesystem lookup outside the embedded Go binary;
* no runtime schema generation;
* no runtime JSON marshalling.

The embedded schemas are baked into the binary at compile time via
Go's `//go:embed` directive. The same byte sequence is used by the
production decoder path and by the CLI introspection surface.

## Stable schema IDs

The schema identifiers are stable URNs:

```text
v1: urn:leamas:gate-summary:v1
v2: urn:leamas:gate-summary:v2
```

The IDs are stable identifiers, not network-fetch requirements. The
schema-printing path never reads them from outside the binary.

The IDs do not embed build versions, release numbers, or commit
identifiers.

## Downstream editor/CI usage

CI jobs and editor integrations can pipe the embedded schema into
any Draft 2020-12 validator:

```bash
# JSON Schema validation in CI
leamas gate-summary schema show v2 > /tmp/gate-summary-v2.schema.json
jsonschema -i .factory/gate-summary.json /tmp/gate-summary-v2.schema.json
```

The output is deterministic and stable across binary rebuilds. The
same byte sequence is reproduced by every invocation.

## Compatibility policy

The introspection surface is a frozen contract. Changes to the
canonical schema bytes require:

1. A new Gate Summary schema version with a new URN identifier;
2. A corresponding entry in the `schema list` table;
3. A new entry in the Supported versions table above;
4. A release note describing the schema change.

The schema bytes are byte-identical to the canonical checked-in
files. Drift between the embedded bytes and the canonical files
fails the release acceptance gate.

## Release policy

The introspection surface is included in every public Leamas
release. The release notes describe the binary as self-contained
for v1/v2 schema reference.

The introspection surface is not a substitute for the decoder or
normalization contract. Downstream consumers that need lifecycle,
diagnostic, or aggregate-status semantics must use the Leamas
decoder directly.

## Known limits

* The CLI prints schema bytes only. It does not validate, parse, or
  normalize Gate Summary documents.
* The CLI does not perform schema validation against Gate Summary
  documents. The validators live in the `internal/gatesummary`
  package.
* The CLI does not provide a `latest` or `current` alias. Callers
  must name an explicit version.
* The CLI does not provide YAML or OpenAPI output. The schema is
  always JSON Schema Draft 2020-12.

## Canonical files

The canonical schema files are the byte-exact source of truth:

```text
internal/gatesummary/schema/gate-summary-v1.schema.json
internal/gatesummary/schema/gate-summary-v2.schema.json
```

The CLI, the embedded validator, and any downstream consumer that
needs to validate Gate Summary documents without consulting the
repository all read the same byte sequences from this package.
