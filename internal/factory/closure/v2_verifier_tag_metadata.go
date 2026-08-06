// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_tag_metadata.go defines the typed
// V2ClosureTagMetadata value object and the small surface
// the parser exposes to the v2 verifier orchestrator.
//
// The metadata block lives inside the annotated tag object's
// raw bytes: the tag-object header (`object`, `type`, `tag`,
// `tagger`) is parsed by ParseV2ClosureTagObjectHeaders in
// v2_verifier_tag_metadata_parse.go; the annotation body is
// the input to ParseV2ClosureTagMetadataTrailers in the same
// file.
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C
// pins the metadata contract v1:
//
//	Leamas-Closure-Protocol-Version: 2
//	Leamas-Plan-Contract-Version: 1
//	Leamas-Subject-Commit: <S>
//	Leamas-Freeze-Commit: <F>
//	Leamas-Closure-Commit: <C>
//	Leamas-Plan-Path: <P>
//	Leamas-Manifest-Path: <M>
//
// Each required key MUST appear exactly once. The metadata
// block is case-sensitive. OIDs MUST be 40-char lowercase
// hex; abbreviated OIDs are rejected.

// V2ClosureTagMetadataKey is the canonical metadata key
// emitted inside an annotated tag object. Values are the
// exact trailer key strings the verifier matches; downstream
// tooling MUST NOT re-parse message text.
type V2ClosureTagMetadataKey string

const (
	V2TagMetadataClosureProtocolVersion V2ClosureTagMetadataKey = "Leamas-Closure-Protocol-Version"
	V2TagMetadataPlanContractVersion    V2ClosureTagMetadataKey = "Leamas-Plan-Contract-Version"
	V2TagMetadataSubjectCommit          V2ClosureTagMetadataKey = "Leamas-Subject-Commit"
	V2TagMetadataFreezeCommit           V2ClosureTagMetadataKey = "Leamas-Freeze-Commit"
	V2TagMetadataClosureCommit          V2ClosureTagMetadataKey = "Leamas-Closure-Commit"
	V2TagMetadataPlanPath               V2ClosureTagMetadataKey = "Leamas-Plan-Path"
	V2TagMetadataManifestPath           V2ClosureTagMetadataKey = "Leamas-Manifest-Path"
)

// V2ClosureTagMetadata is the typed, fully-parsed
// representation of the metadata block extracted from an
// annotated tag object. Every field is populated from a
// required trailer; absent keys result in an error from the
// parser, never a zero value.
type V2ClosureTagMetadata struct {
	ClosureProtocolVersion ClosureProtocolVersion
	PlanContractVersion    PlanContractVersion
	SubjectCommit          string
	FreezeCommit           string
	ClosureCommit          string
	PlanPath               string
	ManifestPath           string
}

// V2TagMetadataPropertyPrefix is the prefix every Leamas
// metadata trailer must carry. Any other Leamas-* key is
// reported as closure_tag_metadata_unknown so the
// orchestrator can surface an unknown alias without parsing
// message text.
const V2TagMetadataPropertyPrefix = "Leamas-"

// V2TagMetadataPropertyName returns the stable property
// name used by V2VerifierDiagnostic.PropertyName for
// metadata-key failures. The returned string is unique per
// key and never collides with the existing
// tag-metadata.X family used by the rejection diagnostics.
func V2TagMetadataPropertyName(key V2ClosureTagMetadataKey) string {
	switch key {
	case V2TagMetadataClosureProtocolVersion:
		return "tag_metadata.closure_protocol_version"
	case V2TagMetadataPlanContractVersion:
		return "tag_metadata.plan_contract_version"
	case V2TagMetadataSubjectCommit:
		return "tag_metadata.subject_commit"
	case V2TagMetadataFreezeCommit:
		return "tag_metadata.freeze_commit"
	case V2TagMetadataClosureCommit:
		return "tag_metadata.closure_commit"
	case V2TagMetadataPlanPath:
		return "tag_metadata.plan_path"
	case V2TagMetadataManifestPath:
		return "tag_metadata.manifest_path"
	}
	return "tag_metadata"
}

// V2TagMetadataAllKeys is the closed set of required
// metadata keys. The slice preserves the canonical ordering
// used by both the parser (insertion order) and the binder
// (left-to-right comparison).
var V2TagMetadataAllKeys = []V2ClosureTagMetadataKey{
	V2TagMetadataClosureProtocolVersion,
	V2TagMetadataPlanContractVersion,
	V2TagMetadataSubjectCommit,
	V2TagMetadataFreezeCommit,
	V2TagMetadataClosureCommit,
	V2TagMetadataPlanPath,
	V2TagMetadataManifestPath,
}

// V2ClosureTagMetadataMismatch describes one field-level
// mismatch between the parsed metadata and the externally
// supplied S/F/C/P/M. The orchestrator converts each entry
// into a closure_tag_metadata_mismatch diagnostic with
// property_name set to V2TagMetadataPropertyName(key).
type V2ClosureTagMetadataMismatch struct {
	Key      V2ClosureTagMetadataKey
	Expected string
	Observed string
}

// V2ClosureTagMetadataObservation records the full outcome
// of the optional metadata assertion phase. The struct is
// the single value the orchestrator surfaces to
// assembleVerification; only the verdict booleans drive the
// final acceptance decision.
type V2ClosureTagMetadataObservation struct {
	// Read is true when the tag-object bytes were fetched
	// successfully and the metadata block parsed without a
	// structural diagnostic. False on any structural
	// rejection (missing, duplicate, unknown, malformed,
	// tag-target-mismatch, tag-unreadable).
	Read bool

	// Bound is true when every metadata field matches the
	// externally supplied S/F/C/P/M (after Read succeeded).
	// A false Bound with Read=true surfaces a per-field
	// mismatch diagnostic.
	Bound bool

	// Metadata is the parsed metadata block. Zero-valued
	// when Read is false; populated when Read is true
	// regardless of Bound.
	Metadata V2ClosureTagMetadata

	// TagObjectOID is the annotated tag-object OID that
	// supplied the metadata bytes. Empty when the tag was
	// not observed (--expected-tag absent).
	TagObjectOID string

	// TagName is the short name of the ref whose
	// dereferenced annotated tag supplied the bytes.
	// Empty when the tag was not observed.
	TagName string

	// Diagnostics is the closed set of typed diagnostics
	// emitted by the metadata phase. Order is preserved.
	Diagnostics V2VerifierDiagnostics

	// Mismatches is the per-field mismatch summary. The
	// list is empty when Bound is true or when Read is
	// false.
	Mismatches []V2ClosureTagMetadataMismatch
}
