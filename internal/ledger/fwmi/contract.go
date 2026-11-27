// Package fwmi implements the FWMI v1 migration interchange contract.
//
// FWMI v1 (Factory Work Migration Interchange) is a MIGRATION INPUT format,
// not a second canonical ledger format. It carries the source-bound
// interpretation of a historical engineering board, produced by a separate
// agent, into the deterministic ledger.
//
// The adapter is deliberately CONSERVATIVE:
//
//   - it validates the interchange records strictly;
//   - it preserves every source identity and provenance;
//   - it preserves an unknown status distinctly from an intentional deferral;
//   - it preserves unresolved migration records;
//   - it never promotes a historical qualification assertion into a newly
//     verified acceptance decision.
package fwmi

// SchemaVersion is the version of the FWMI interchange contract this
// adapter understands.
const SchemaVersion = 1

// Normalized lifecycle vocabulary.
const (
	LifecycleOpen       = "OPEN"
	LifecycleActive     = "ACTIVE"
	LifecycleBlocked    = "BLOCKED"
	LifecycleDeferred   = "DEFERRED"
	LifecycleTerminal   = "TERMINAL"
	LifecycleUnresolved = "UNRESOLVED"
)

// Qualification vocabulary.
//
// These are HISTORICAL SOURCE ASSERTIONS. They are not acceptance decisions.
const (
	QualificationUnknown        = "UNKNOWN"
	QualificationUnqualified    = "UNQUALIFIED"
	QualificationPartial        = "PARTIAL"
	QualificationStructuralOnly = "STRUCTURAL_ONLY"
	QualificationLiveQualified  = "LIVE_QUALIFIED"
	QualificationAccepted       = "ACCEPTED"
	QualificationInsufficient   = "INSUFFICIENT"
	QualificationRejected       = "REJECTED"
)

// Coverage vocabulary.
//
// Coverage distinguishes "there are none" from "they were not recovered".
const (
	CoverageComplete = "COMPLETE"
	CoveragePartial  = "PARTIAL"
	CoverageUnknown  = "UNKNOWN"
)

// Relationship kinds supported by the interchange contract.
const (
	RelDependsOn        = "DEPENDS_ON"
	RelRelatedTo        = "RELATED_TO"
	RelSupersedes       = "SUPERSEDES"
	RelCrossLink        = "CROSS_LINK"
	RelAddresses        = "ADDRESSES"
	RelExternalDependec = "EXTERNAL_DEPENDENCY"
)

// IsValidLifecycle reports whether the lifecycle value is in the vocabulary.
func IsValidLifecycle(v string) bool {
	switch v {
	case LifecycleOpen, LifecycleActive, LifecycleBlocked, LifecycleDeferred,
		LifecycleTerminal, LifecycleUnresolved:
		return true
	}
	return false
}

// IsValidQualification reports whether the qualification value is in the
// vocabulary.
func IsValidQualification(v string) bool {
	switch v {
	case QualificationUnknown, QualificationUnqualified, QualificationPartial,
		QualificationStructuralOnly, QualificationLiveQualified,
		QualificationAccepted, QualificationInsufficient, QualificationRejected:
		return true
	}
	return false
}

// IsValidCoverage reports whether the coverage value is in the vocabulary.
func IsValidCoverage(v string) bool {
	switch v {
	case CoverageComplete, CoveragePartial, CoverageUnknown:
		return true
	}
	return false
}

// IsValidRelationshipKind reports whether the relationship kind is known.
func IsValidRelationshipKind(v string) bool {
	switch v {
	case RelDependsOn, RelRelatedTo, RelSupersedes, RelCrossLink,
		RelAddresses, RelExternalDependec:
		return true
	}
	return false
}

// SourceRef is a byte-exact source reference.
//
// The line range is one-based and inclusive. The digest covers the exact
// original bytes of that range, including the newline bytes.
type SourceRef struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	SHA256    string `json:"sha256"`
	// SourceHead binds the reference to a source revision when it differs
	// from the parent source manifest head.
	SourceHead string `json:"source_head,omitempty"`
}

// Relationship is a known relationship between records.
type Relationship struct {
	Kind string `json:"kind"`
	// Target is the identity the relationship points at.
	Target string `json:"target"`
	// RequiredOutcome states what the relationship requires, if applicable.
	RequiredOutcome string      `json:"required_outcome,omitempty"`
	SourceRefs      []SourceRef `json:"source_refs,omitempty"`
}

// SourceFile is one file of the source manifest.
type SourceFile struct {
	Path string `json:"path"`
	// Bytes is the size of the file.
	Bytes int64 `json:"bytes"`
	// Lines is the line count of the file.
	Lines int `json:"lines"`
	// SHA256 is the digest of the whole file.
	SHA256 string `json:"sha256"`
	// GitBlob is the committed blob identity of the file.
	GitBlob string `json:"git_blob"`
	// Disposition accounts for the file's role in the migration.
	Disposition string `json:"disposition"`
}

// SourceManifest establishes exactly what was examined.
type SourceManifest struct {
	SchemaVersion int          `json:"schema_version"`
	Repository    string       `json:"repository"`
	SourceHead    string       `json:"source_head"`
	Files         []SourceFile `json:"files"`
}
