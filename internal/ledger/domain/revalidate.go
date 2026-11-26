package domain

import "fmt"

// EvidenceValidation is the outcome of re-validating one evidence record
// against the repository.
type EvidenceValidation struct {
	// EvidenceID is the re-validated evidence identity.
	EvidenceID EvidenceID
	// Valid reports whether the artifact still carries the declared
	// identity.
	Valid bool
	// Reason explains an invalidation.
	Reason string
}

// RevalidateEvidence re-checks the artifact identity of one evidence
// record against the repository.
//
// This is the mechanism that detects invalidated or changed evidence: an
// artifact that was modified, removed or replaced after its verification
// no longer carries the declared identity.
//
// A record that declares no artifact cannot be invalidated by artifact
// drift; it is reported as valid because there is nothing to re-check.
func RevalidateEvidence(repoRoot string, rec EvidenceRefRecord) EvidenceValidation {
	out := EvidenceValidation{EvidenceID: rec.EvidenceID}
	if rec.RelativePath == "" || rec.SHA256 == "" {
		out.Valid = true
		return out
	}
	identity, err := VerifyArtifact(repoRoot, rec.RelativePath, rec.SHA256, rec.ByteSize)
	if err != nil {
		out.Reason = err.Error()
		return out
	}
	if identity.SHA256 != rec.SHA256 {
		out.Reason = fmt.Sprintf("the artifact digest changed from %s to %s",
			rec.SHA256, identity.SHA256)
		return out
	}
	out.Valid = true
	return out
}
