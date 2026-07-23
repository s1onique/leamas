// Package schema exposes the canonical, byte-exact Gate Summary v1 and v2
// JSON Schemas that the Leamas binary prints through the
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
package schema

import "embed"

// files is the embedded Draft 2020-12 schema pair. Both files are
// authored in canonical, reviewable form: two-space indentation, LF
// line endings, exactly one trailing LF, no BOM, no timestamps, no host
// paths.
//
//go:embed gate-summary-v1.schema.json gate-summary-v2.schema.json
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
	// VersionV2 is the current Leamas Gate Summary wire format.
	VersionV2 Version = "v2"
)

// Schema identifiers are stable URNs defined by the wire-format contract.
// They are not network-fetch requirements; the schema-printing path
// never reads them from outside the binary.
const (
	SchemaIDV1 = "urn:leamas:gate-summary:v1"
	SchemaIDV2 = "urn:leamas:gate-summary:v2"
)

// schemaFileName returns the canonical file name for a version.
// The set of valid versions is closed; unknown versions return "".
func schemaFileName(v Version) string {
	switch v {
	case VersionV1:
		return "gate-summary-v1.schema.json"
	case VersionV2:
		return "gate-summary-v2.schema.json"
	}
	return ""
}
