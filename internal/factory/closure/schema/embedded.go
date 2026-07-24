// Package schema exposes the canonical Closure Protocol v1 plan JSON
// Schema that runtime validators and external tools must agree on.
//
// The schema is the **single source of truth** for the JSON shape of
// closure plans. It is embedded at compile time via Go's `//go:embed`
// directive. The package performs no runtime filesystem or network
// lookup and no runtime JSON marshalling.
//
// The ACT
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-PLAN-EXECUTION-MODE-RECONCILIATION01
// added the schema and pinned schema/runtime parity through a shared
// fixture table. New fields, enum values, or required constraints
// MUST be appended here and only here; no other site in the
// repository may enumerate supported execution modes.
package schema

import (
	"embed"
)

// files holds the embedded Closure Protocol v1 JSON Schema. The file
// is authored in canonical, reviewable form: two-space indentation,
// LF line endings, exactly one trailing LF, no BOM, no timestamps, no
// host paths.
//
//go:embed closure-plan-v1.schema.json
var files embed.FS

// SchemaIDV1 is the stable URN identifier of the v1 closure-plan
// schema. It is not a network-fetch requirement; the validation path
// never reads it from outside the binary.
const SchemaIDV1 = "urn:leamas:closure-plan:v1"

// Version is the canonical textual identifier of the schema version.
type Version string

// VersionV1 is the supported Closure Protocol v1 schema version.
const VersionV1 Version = "v1"

// supportedVersions is the closed set of versions the package
// recognises. New versions MUST be appended here and only here.
var supportedVersions = []Version{VersionV1}

// Bytes returns a defensive copy of the embedded schema bytes for
// the given version. Unknown versions return a typed
// *UnknownVersionError so callers can dispatch a dedicated diagnostic
// without exposing the concrete error type.
func Bytes(v Version) ([]byte, error) {
	name := schemaFileName(v)
	if name == "" {
		return nil, &UnknownVersionError{Version: v}
	}
	raw, err := files.ReadFile(name)
	if err != nil {
		return nil, &UnknownVersionError{Version: v}
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out, nil
}

// MustBytes returns the embedded schema bytes for the given version
// and panics on error. It is intended for tests and for callers that
// have already validated the version.
func MustBytes(v Version) []byte {
	b, err := Bytes(v)
	if err != nil {
		panic(err)
	}
	return b
}

// IsKnownVersion reports whether v is in the closed supported set.
func IsKnownVersion(v Version) bool {
	for _, candidate := range supportedVersions {
		if candidate == v {
			return true
		}
	}
	return false
}

// UnknownVersionError is returned when a request names a version
// that is not part of the closed version set.
type UnknownVersionError struct {
	Version Version
}

// Error implements the error interface.
func (e *UnknownVersionError) Error() string {
	return "closure-plan schema: unknown version " + string(e.Version)
}

// schemaFileName returns the canonical file name for a version. The
// set of valid versions is closed; unknown versions return "".
func schemaFileName(v Version) string {
	switch v {
	case VersionV1:
		return "closure-plan-v1.schema.json"
	}
	return ""
}
