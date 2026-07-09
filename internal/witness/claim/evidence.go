// Package claim provides typed domain models for claims and evidence
// in Leamas verification witness artifacts.
package claim

import (
	"errors"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// Schema version for evidence.
const EvidenceSchemaVersion = "leamas.evidence.v1"

// EvidenceKind represents the kind of evidence.
type EvidenceKind string

// Evidence kind constants.
const (
	EvidenceKindCommandOutput  EvidenceKind = "command_output"
	EvidenceKindDigest         EvidenceKind = "digest"
	EvidenceKindLog            EvidenceKind = "log"
	EvidenceKindFile           EvidenceKind = "file"
	EvidenceKindTrace          EvidenceKind = "trace"
	EvidenceKindVerifierResult EvidenceKind = "verifier_result"
)

// EvidenceRole represents the role of evidence in supporting claims.
type EvidenceRole string

// Evidence role constants.
const (
	EvidenceRolePrimary       EvidenceRole = "primary"
	EvidenceRoleSupporting    EvidenceRole = "supporting"
	EvidenceRoleContradicting EvidenceRole = "contradicting"
	EvidenceRoleContext       EvidenceRole = "context"
)

// Evidence represents a piece of evidence that supports or contradicts claims.
type Evidence struct {
	SchemaVersion string            `json:"schema_version"`
	ID            EvidenceID        `json:"id"`
	RunID         runbundle.RunID   `json:"run_id"`
	CreatedAt     time.Time         `json:"created_at"`
	Kind          EvidenceKind      `json:"kind"`
	Role          EvidenceRole      `json:"role"`
	Title         string            `json:"title"`
	RelativePath  string            `json:"relative_path,omitempty"`
	Summary       string            `json:"summary,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Errors for evidence validation.
var (
	ErrEmptyTitle              = errors.New("evidence title must be non-empty")
	ErrInvalidKind             = errors.New("evidence kind must be one of the declared EvidenceKind values")
	ErrInvalidRole             = errors.New("evidence role must be one of the declared EvidenceRole values")
	ErrInvalidEvidenceIDType   = errors.New("evidence ID must pass ValidateEvidenceID")
	ErrInvalidRunIDType        = errors.New("run ID must pass runbundle.ValidateRunID")
	ErrEvidenceSchemaMismatch  = errors.New("schema_version must be exactly leamas.evidence.v1")
	ErrInvalidRelativePathType = errors.New("relative path must be safe")
)

// IsValidEvidenceKind checks if the kind is valid.
func IsValidEvidenceKind(kind EvidenceKind) bool {
	switch kind {
	case EvidenceKindCommandOutput, EvidenceKindDigest, EvidenceKindLog,
		EvidenceKindFile, EvidenceKindTrace, EvidenceKindVerifierResult:
		return true
	}
	return false
}

// IsValidEvidenceRole checks if the role is valid.
func IsValidEvidenceRole(role EvidenceRole) bool {
	switch role {
	case EvidenceRolePrimary, EvidenceRoleSupporting, EvidenceRoleContradicting, EvidenceRoleContext:
		return true
	}
	return false
}

// NewEvidence creates a new Evidence with default values.
func NewEvidence(id EvidenceID, runID runbundle.RunID, kind EvidenceKind, role EvidenceRole, title string, now time.Time) (Evidence, error) {
	// Validate evidence ID
	if err := ValidateEvidenceID(id); err != nil {
		return Evidence{}, err
	}

	// Validate run ID
	if err := runbundle.ValidateRunID(runID); err != nil {
		return Evidence{}, ErrInvalidRunIDType
	}

	// Validate kind
	if !IsValidEvidenceKind(kind) {
		return Evidence{}, ErrInvalidKind
	}

	// Validate role
	if !IsValidEvidenceRole(role) {
		return Evidence{}, ErrInvalidRole
	}

	// Validate title
	if title == "" {
		return Evidence{}, ErrEmptyTitle
	}

	return Evidence{
		SchemaVersion: EvidenceSchemaVersion,
		ID:            id,
		RunID:         runID,
		CreatedAt:     now,
		Kind:          kind,
		Role:          role,
		Title:         title,
		Metadata:      map[string]string{},
	}, nil
}

// Validate validates an evidence record and returns an error if invalid.
func (e Evidence) Validate() error {
	// Schema version check
	if e.SchemaVersion != EvidenceSchemaVersion {
		return ErrEvidenceSchemaMismatch
	}

	// ID check
	if err := ValidateEvidenceID(e.ID); err != nil {
		return ErrInvalidEvidenceIDType
	}

	// Run ID check
	if err := runbundle.ValidateRunID(e.RunID); err != nil {
		return ErrInvalidRunIDType
	}

	// Kind check
	if !IsValidEvidenceKind(e.Kind) {
		return ErrInvalidKind
	}

	// Role check
	if !IsValidEvidenceRole(e.Role) {
		return ErrInvalidRole
	}

	// Title check
	if e.Title == "" {
		return ErrEmptyTitle
	}

	// Relative path check (if set)
	if e.RelativePath != "" {
		if err := ValidateRelativePath(e.RelativePath); err != nil {
			return ErrInvalidRelativePathType
		}
	}

	// Metadata keys and values must be strings (map already enforces this)

	return nil
}
