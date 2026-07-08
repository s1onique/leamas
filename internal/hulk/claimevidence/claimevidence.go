// Package claimevidence provides typed domain models for claims, evidence,
// and sources in Leamas review artifacts.
//
// This package is pure domain logic with no filesystem, process, network,
// database, clock, or UI dependencies.
package claimevidence

import "sort"

// Typed identifiers.

type ClaimID string
type EvidenceID string
type SourceID string
type ArtifactID string

// Narrow string types.

type ClaimStatus string
type ClaimKind string
type EvidenceKind string
type SourceKind string
type ConfidenceLevel string

// ClaimStatus constants.

const (
	ClaimStatusOpen      ClaimStatus = "open"
	ClaimStatusSupported ClaimStatus = "supported"
	ClaimStatusRefuted   ClaimStatus = "refuted"
	ClaimStatusUnknown   ClaimStatus = "unknown"
)

// ClaimKind constants.

const (
	ClaimKindFact           ClaimKind = "fact"
	ClaimKindInterpretation ClaimKind = "interpretation"
	ClaimKindRisk           ClaimKind = "risk"
	ClaimKindLimitation     ClaimKind = "limitation"
)

// EvidenceKind constants.

const (
	EvidenceKindDigest      EvidenceKind = "digest"
	EvidenceKindLog         EvidenceKind = "log"
	EvidenceKindProof       EvidenceKind = "proof"
	EvidenceKindCloseReport EvidenceKind = "close_report"
	EvidenceKindObservation EvidenceKind = "observation"
	EvidenceKindOther       EvidenceKind = "other"
)

// SourceKind constants.

const (
	SourceKindArtifact SourceKind = "artifact"
	SourceKindHuman    SourceKind = "human"
	SourceKindAgent    SourceKind = "agent"
	SourceKindVerifier SourceKind = "verifier"
)

// ConfidenceLevel constants.

const (
	ConfidenceLow    ConfidenceLevel = "low"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceHigh   ConfidenceLevel = "high"
)

// Core model types.

type Claim struct {
	ID          ClaimID
	Kind        ClaimKind
	Status      ClaimStatus
	Summary     string
	Detail      string
	Confidence  ConfidenceLevel
	EvidenceIDs []EvidenceID
	Limitations []string
}

type Evidence struct {
	ID         EvidenceID
	Kind       EvidenceKind
	Summary    string
	SourceID   SourceID
	ArtifactID ArtifactID
	Excerpt    string
}

type Source struct {
	ID         SourceID
	Kind       SourceKind
	Summary    string
	ArtifactID ArtifactID
}

type ClaimEvidenceBundle struct {
	Claims   []Claim
	Evidence []Evidence
	Sources  []Source
}

// Constructors.

func NewClaim(id ClaimID, kind ClaimKind, summary string) Claim {
	return Claim{
		ID:         id,
		Kind:       kind,
		Status:     ClaimStatusOpen,
		Summary:    summary,
		Confidence: ConfidenceMedium,
	}
}

func NewEvidence(id EvidenceID, kind EvidenceKind, summary string) Evidence {
	return Evidence{
		ID:      id,
		Kind:    kind,
		Summary: summary,
	}
}

func NewSource(id SourceID, kind SourceKind, summary string) Source {
	return Source{
		ID:      id,
		Kind:    kind,
		Summary: summary,
	}
}

// Helper validity functions.

func IsValidClaimStatus(status ClaimStatus) bool {
	switch status {
	case ClaimStatusOpen, ClaimStatusSupported, ClaimStatusRefuted, ClaimStatusUnknown:
		return true
	}
	return false
}

func IsValidClaimKind(kind ClaimKind) bool {
	switch kind {
	case ClaimKindFact, ClaimKindInterpretation, ClaimKindRisk, ClaimKindLimitation:
		return true
	}
	return false
}

func IsValidEvidenceKind(kind EvidenceKind) bool {
	switch kind {
	case EvidenceKindDigest, EvidenceKindLog, EvidenceKindProof,
		EvidenceKindCloseReport, EvidenceKindObservation, EvidenceKindOther:
		return true
	}
	return false
}

func IsValidSourceKind(kind SourceKind) bool {
	switch kind {
	case SourceKindArtifact, SourceKindHuman, SourceKindAgent, SourceKindVerifier:
		return true
	}
	return false
}

func IsValidConfidence(confidence ConfidenceLevel) bool {
	switch confidence {
	case ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
		return true
	}
	return false
}

// Validation API.

type ValidationFinding struct {
	Path    string
	Message string
}

type ValidationResult struct {
	Findings []ValidationFinding
}

func (r ValidationResult) OK() bool {
	return len(r.Findings) == 0
}

// Validate validates a claim/evidence bundle and returns the validation result.
// Validation is deterministic and side-effect free.
func Validate(bundle ClaimEvidenceBundle) ValidationResult {
	var findings []ValidationFinding

	// Build source ID set for reference validation
	sourceIDs := make(map[SourceID]bool)
	for _, s := range bundle.Sources {
		if s.ID == "" {
			findings = append(findings, ValidationFinding{
				Path:    "Source.ID",
				Message: "Source ID must be non-empty",
			})
		} else {
			sourceIDs[s.ID] = true
		}
	}

	// Check for duplicate source IDs
	if len(bundle.Sources) > 0 {
		seen := make(map[SourceID]bool)
		for _, s := range bundle.Sources {
			if s.ID == "" {
				continue
			}
			if seen[s.ID] {
				findings = append(findings, ValidationFinding{
					Path:    "Source.ID",
					Message: "Duplicate source ID: " + string(s.ID),
				})
			}
			seen[s.ID] = true
		}
	}

	// Validate sources
	for _, s := range bundle.Sources {
		if !IsValidSourceKind(s.Kind) {
			findings = append(findings, ValidationFinding{
				Path:    "Source.Kind",
				Message: "Invalid source kind: " + string(s.Kind),
			})
		}
		if s.Summary == "" {
			findings = append(findings, ValidationFinding{
				Path:    "Source.Summary",
				Message: "Source summary must be non-empty",
			})
		}
		// ArtifactID must be non-empty when SourceKindArtifact
		if s.Kind == SourceKindArtifact && s.ArtifactID == "" {
			findings = append(findings, ValidationFinding{
				Path:    "Source.ArtifactID",
				Message: "ArtifactID must be non-empty for SourceKindArtifact",
			})
		}
	}

	// Build evidence ID set
	evidenceIDs := make(map[EvidenceID]bool)
	for _, e := range bundle.Evidence {
		if e.ID == "" {
			findings = append(findings, ValidationFinding{
				Path:    "Evidence.ID",
				Message: "Evidence ID must be non-empty",
			})
		} else {
			evidenceIDs[e.ID] = true
		}
	}

	// Check for duplicate evidence IDs
	if len(bundle.Evidence) > 0 {
		seen := make(map[EvidenceID]bool)
		for _, e := range bundle.Evidence {
			if e.ID == "" {
				continue
			}
			if seen[e.ID] {
				findings = append(findings, ValidationFinding{
					Path:    "Evidence.ID",
					Message: "Duplicate evidence ID: " + string(e.ID),
				})
			}
			seen[e.ID] = true
		}
	}

	// Validate evidence
	for _, e := range bundle.Evidence {
		if !IsValidEvidenceKind(e.Kind) {
			findings = append(findings, ValidationFinding{
				Path:    "Evidence.Kind",
				Message: "Invalid evidence kind: " + string(e.Kind),
			})
		}
		if e.Summary == "" {
			findings = append(findings, ValidationFinding{
				Path:    "Evidence.Summary",
				Message: "Evidence summary must be non-empty",
			})
		}
		// SourceID must reference an existing source when non-empty
		if e.SourceID != "" && !sourceIDs[e.SourceID] {
			findings = append(findings, ValidationFinding{
				Path:    "Evidence.SourceID",
				Message: "Evidence.SourceID references non-existent source: " + string(e.SourceID),
			})
		}
	}

	// Validate claims
	for _, c := range bundle.Claims {
		if c.ID == "" {
			findings = append(findings, ValidationFinding{
				Path:    "Claim.ID",
				Message: "Claim ID must be non-empty",
			})
		}
		if !IsValidClaimKind(c.Kind) {
			findings = append(findings, ValidationFinding{
				Path:    "Claim.Kind",
				Message: "Invalid claim kind: " + string(c.Kind),
			})
		}
		if !IsValidClaimStatus(c.Status) {
			findings = append(findings, ValidationFinding{
				Path:    "Claim.Status",
				Message: "Invalid claim status: " + string(c.Status),
			})
		}
		if c.Summary == "" {
			findings = append(findings, ValidationFinding{
				Path:    "Claim.Summary",
				Message: "Claim summary must be non-empty",
			})
		}
		if !IsValidConfidence(c.Confidence) {
			findings = append(findings, ValidationFinding{
				Path:    "Claim.Confidence",
				Message: "Invalid confidence level: " + string(c.Confidence),
			})
		}
		// EvidenceIDs must reference existing evidence
		for _, eid := range c.EvidenceIDs {
			if !evidenceIDs[eid] {
				findings = append(findings, ValidationFinding{
					Path:    "Claim.EvidenceIDs",
					Message: "Claim references non-existent evidence: " + string(eid),
				})
			}
		}
		// Limitations must not contain empty strings
		for _, lim := range c.Limitations {
			if lim == "" {
				findings = append(findings, ValidationFinding{
					Path:    "Claim.Limitations",
					Message: "Limitations must not contain empty strings",
				})
				break
			}
		}
	}

	// Check for duplicate claim IDs
	if len(bundle.Claims) > 0 {
		seen := make(map[ClaimID]bool)
		for _, c := range bundle.Claims {
			if c.ID == "" {
				continue
			}
			if seen[c.ID] {
				findings = append(findings, ValidationFinding{
					Path:    "Claim.ID",
					Message: "Duplicate claim ID: " + string(c.ID),
				})
			}
			seen[c.ID] = true
		}
	}

	// Sort findings for deterministic order
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Message < findings[j].Message
	})

	return ValidationResult{Findings: findings}
}
