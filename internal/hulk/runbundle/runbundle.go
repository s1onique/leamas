// Package runbundle provides typed domain models for Leamas run bundles.
//
// Run bundles are local, reviewable units that group an execution/run,
// its metadata, artifacts, claims, evidence references, and validation state.
//
// This package is pure domain logic with no filesystem, process, network,
// database, or UI dependencies. Timestamps are passed as strings to keep
// the model pure and avoid formatting policy creep.
package runbundle

import "sort"

// Typed identifiers.

// RunBundleID identifies a run bundle.
type RunBundleID string

// RunID identifies a run.
type RunID string

// ArtifactID identifies an artifact.
type ArtifactID string

// ClaimID identifies a claim.
type ClaimID string

// EvidenceID identifies evidence.
type EvidenceID string

// Status and kind types.

// RunBundleStatus represents the status of a run bundle.
type RunBundleStatus string

const (
	RunBundleDraft    RunBundleStatus = "draft"
	RunBundleComplete RunBundleStatus = "complete"
	RunBundleInvalid  RunBundleStatus = "invalid"
)

// ArtifactKind represents the kind of an artifact.
type ArtifactKind string

const (
	ArtifactDigest      ArtifactKind = "digest"
	ArtifactCloseReport ArtifactKind = "close_report"
	ArtifactProof       ArtifactKind = "proof"
	ArtifactLog         ArtifactKind = "log"
	ArtifactOther       ArtifactKind = "other"
)

// Core model types.

// RunBundle represents a run bundle: a local, reviewable unit of execution.
type RunBundle struct {
	ID          RunBundleID
	RunID       RunID
	Status      RunBundleStatus
	Summary     string
	CreatedAt   string
	Artifacts   []ArtifactRef
	Claims      []ClaimRef
	Evidence    []EvidenceRef
	Limitations []string
}

// ArtifactRef references an artifact within a run bundle.
type ArtifactRef struct {
	ID     ArtifactID
	Kind   ArtifactKind
	Path   string
	Role   string
	Digest string
}

// ClaimRef references a claim within a run bundle.
// Minimal reference until ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01.
type ClaimRef struct {
	ID      ClaimID
	Summary string
}

// EvidenceRef references evidence within a run bundle.
// Minimal reference until ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01.
type EvidenceRef struct {
	ID         EvidenceID
	ArtifactID ArtifactID
	Summary    string
}

// Constructors and helpers.

// NewRunBundle creates a new RunBundle with the given fields.
func NewRunBundle(id RunBundleID, runID RunID, createdAt string, summary string) RunBundle {
	return RunBundle{
		ID:        id,
		RunID:     runID,
		CreatedAt: createdAt,
		Summary:   summary,
		Status:    RunBundleDraft,
	}
}

// IsValidStatus returns true if the status is a known valid status.
func IsValidStatus(status RunBundleStatus) bool {
	switch status {
	case RunBundleDraft, RunBundleComplete, RunBundleInvalid:
		return true
	}
	return false
}

// IsValidArtifactKind returns true if the artifact kind is a known valid kind.
func IsValidArtifactKind(kind ArtifactKind) bool {
	switch kind {
	case ArtifactDigest, ArtifactCloseReport, ArtifactProof, ArtifactLog, ArtifactOther:
		return true
	}
	return false
}

// Validation API.

// ValidationFinding represents a single validation finding.
type ValidationFinding struct {
	Path    string
	Message string
}

// ValidationResult represents the result of validation.
type ValidationResult struct {
	Findings []ValidationFinding
}

// OK returns true if there are no validation findings.
func (r ValidationResult) OK() bool {
	return len(r.Findings) == 0
}

// Validate validates a run bundle and returns the validation result.
// Validation is deterministic and side-effect free.
func Validate(bundle RunBundle) ValidationResult {
	var findings []ValidationFinding

	// Check required fields
	if bundle.ID == "" {
		findings = append(findings, ValidationFinding{
			Path:    "RunBundle.ID",
			Message: "RunBundle.ID is required",
		})
	}
	if bundle.RunID == "" {
		findings = append(findings, ValidationFinding{
			Path:    "RunBundle.RunID",
			Message: "RunBundle.RunID is required",
		})
	}
	if bundle.CreatedAt == "" {
		findings = append(findings, ValidationFinding{
			Path:    "RunBundle.CreatedAt",
			Message: "RunBundle.CreatedAt is required",
		})
	}
	if bundle.Summary == "" {
		findings = append(findings, ValidationFinding{
			Path:    "RunBundle.Summary",
			Message: "RunBundle.Summary is required",
		})
	}

	// Check status
	if !IsValidStatus(bundle.Status) {
		findings = append(findings, ValidationFinding{
			Path:    "RunBundle.Status",
			Message: "Status must be one of: draft, complete, invalid",
		})
	}

	// Build artifact ID set for reference validation
	artifactIDs := make(map[ArtifactID]bool)
	for _, a := range bundle.Artifacts {
		if a.ID == "" {
			findings = append(findings, ValidationFinding{
				Path:    "ArtifactRef.ID",
				Message: "Artifact ID must be non-empty",
			})
		} else {
			artifactIDs[a.ID] = true
		}
	}

	// Check for duplicate artifact IDs
	if len(artifactIDs) > 0 {
		seen := make(map[ArtifactID]bool)
		for _, a := range bundle.Artifacts {
			if a.ID == "" {
				continue
			}
			if seen[a.ID] {
				findings = append(findings, ValidationFinding{
					Path:    "ArtifactRef.ID",
					Message: "Duplicate artifact ID: " + string(a.ID),
				})
			}
			seen[a.ID] = true
		}
	}

	// Check artifact kinds and paths
	for i, a := range bundle.Artifacts {
		if !IsValidArtifactKind(a.Kind) {
			findings = append(findings, ValidationFinding{
				Path:    "ArtifactRef.Kind",
				Message: "Invalid artifact kind: " + string(a.Kind),
			})
		}
		if a.Path == "" {
			findings = append(findings, ValidationFinding{
				Path:    "ArtifactRef.Path",
				Message: "Artifact path must be non-empty",
			})
		}
		_ = i // index reserved for future use
	}

	// Build claim ID set and check for duplicates
	claimIDs := make(map[ClaimID]bool)
	if len(bundle.Claims) > 0 {
		seen := make(map[ClaimID]bool)
		for _, c := range bundle.Claims {
			if c.ID == "" {
				findings = append(findings, ValidationFinding{
					Path:    "ClaimRef.ID",
					Message: "Claim ID must be non-empty",
				})
			} else {
				claimIDs[c.ID] = true
				if seen[c.ID] {
					findings = append(findings, ValidationFinding{
						Path:    "ClaimRef.ID",
						Message: "Duplicate claim ID: " + string(c.ID),
					})
				}
				seen[c.ID] = true
			}
		}
	}

	// Check evidence references
	if len(bundle.Evidence) > 0 {
		seen := make(map[EvidenceID]bool)
		for _, e := range bundle.Evidence {
			if e.ID == "" {
				findings = append(findings, ValidationFinding{
					Path:    "EvidenceRef.ID",
					Message: "Evidence ID must be non-empty",
				})
			} else {
				if seen[e.ID] {
					findings = append(findings, ValidationFinding{
						Path:    "EvidenceRef.ID",
						Message: "Duplicate evidence ID: " + string(e.ID),
					})
				}
				seen[e.ID] = true
			}

			// Check that ArtifactID references an existing artifact
			if e.ArtifactID != "" && !artifactIDs[e.ArtifactID] {
				findings = append(findings, ValidationFinding{
					Path:    "EvidenceRef.ArtifactID",
					Message: "EvidenceRef.ArtifactID references non-existent artifact: " + string(e.ArtifactID),
				})
			}
		}
	}

	// Check limitations for empty strings
	for _, lim := range bundle.Limitations {
		if lim == "" {
			findings = append(findings, ValidationFinding{
				Path:    "RunBundle.Limitations",
				Message: "Limitations must not contain empty strings",
			})
			break
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
