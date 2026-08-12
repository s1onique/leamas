// Package schema exposes the canonical, byte-exact Gate Summary v1, v2,
// and v3 JSON Schemas that the Leamas binary prints through the
// `leamas gate-summary schema` command surface.
//
// The package is the **single source of truth** for the wire-format
// reference. The CLI, the embedded validator, and any downstream consumer
// that needs to validate Gate Summary documents without consulting the
// repository all read the same byte sequences from this package.
//
// The schemas are embedded at compile time via Go's `//go:embed` directive.
// The package performs no runtime filesytem or network lookup, no runtime
// JSON marshalling, and no schema generation. Callers receive clones of
// the embedded bytes so the compiled-in authority cannot be mutated.
//
// V3 is the canonical InDeep Factory Gate Summary wire format. The
// InDeep repository authors the authoritative v3 schema; Leamas mirrors
// a byte-exact copy so its decoder pipeline can validate InDeep-produced
// documents without consulting any external repository or filesystem.
package schema

import "embed"

// files is the embedded Draft 2020-12 schema set. All files are
// authored in canonical, reviewable form: two-space indentation, LF
// line endings, exactly one trailing LF, no BOM, no timestamps, no host
// paths.
//
//go:embed gate-summary-v1.schema.json gate-summary-v2.schema.json gate-summary-v3.schema.json
var files embed.FS

// Status is the descriptive CLI metadata for a supported schema.
// The status values documented under `leamas gate-summary schema list`
// are CLI-only labels; they are not part of either JSON Schema.
type Status string

const (
	// StatusSupported marks a schema that is still accepted by the
	// decoder but is no longer the current authority.
	StatusSupported Status = "supported"
	// StatusCurrent marks the schema that is the current wire format.
	StatusCurrent Status = "current"
)

// Version is the canonical, case-sensitive textual name of a Gate Summary
// schema version. The CLI requires callers to spell the version exactly
// as the constant values below; mutable aliases such as "latest" or
// "current" are rejected by the command surface.
type Version string

const (
	// VersionV1 is the supported legacy Leamas Gate Summary format.
	VersionV1 Version = "v1"
	// VersionV2 is the supported Leamas Gate Summary format.
	VersionV2 Version = "v2"
	// VersionV3 is the current Leamas Gate Summary wire format.
	//
	// V3 is a strict superset of v2 that introduces the InDeep canonical
	// `counts` aggregate block, per-check `id`/`order`/`execution_class`/
	// `argv`/`implementation`/`deadline_exceeded`/`raw_process_exit_code`/
	// `termination_signal`/`stdout_bytes`/`stderr_bytes`/`stdout_truncated`/
	// `stderr_truncated` evidence fields, and the registry/events/transcript
	// SHA-256 binding. V3 documents are produced by the InDeep Go runner
	// under ACT-INDEEP-FACTORY-QUALITY-GATE-GO-RUNNER01-CORRECTION02.
	VersionV3 Version = "v3"
)

// Schema identifiers are stable URNs defined by the wire-format contract.
// They are not network-fetch requirements; the schema-printing path
// never reads them from outside the binary.
//
// SchemaIDV3 mirrors the InDeep canonical URN
// (`urn:indeep:factory:gate-summary:v3`) so the embedded v3 schema and
// the InDeep-authored v3 schema share the same wire identifier. The
// identifier is metadata; it does not change validation semantics.
const (
	SchemaIDV1 = "urn:leamas:gate-summary:v1"
	SchemaIDV2 = "urn:leamas:gate-summary:v2"
	SchemaIDV3 = "urn:indeep:factory:gate-summary:v3"
)

// schemaFileName returns the canonical file name for a version.
// The set of valid versions is closed; unknown versions return "".
func schemaFileName(v Version) string {
	switch v {
	case VersionV1:
		return "gate-summary-v1.schema.json"
	case VersionV2:
		return "gate-summary-v2.schema.json"
	case VersionV3:
		return "gate-summary-v3.schema.json"
	}
	return ""
}
