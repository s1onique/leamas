// Package claim provides typed domain models for claims and evidence
// in Leamas verification witness artifacts.
package claim

import (
	"errors"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// Schema version for claims.
const ClaimSchemaVersion = "leamas.claim.v1"

// ClaimStatus represents the status of a claim.
type ClaimStatus string

// Claim status constants.
const (
	ClaimStatusOpen      ClaimStatus = "open"
	ClaimStatusSupported ClaimStatus = "supported"
	ClaimStatusRejected  ClaimStatus = "rejected"
	ClaimStatusUnknown   ClaimStatus = "unknown"
)

// Verdict represents the verification verdict of a claim.
type Verdict string

// Verdict constants.
const (
	VerdictUnreviewed Verdict = "unreviewed"
	VerdictPass       Verdict = "pass"
	VerdictFail       Verdict = "fail"
	VerdictMixed      Verdict = "mixed"
)

// Claim represents a verification claim with evidence support.
type Claim struct {
	SchemaVersion string          `json:"schema_version"`
	ID            ClaimID         `json:"id"`
	RunID         runbundle.RunID `json:"run_id"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Statement     string          `json:"statement"`
	Status        ClaimStatus     `json:"status"`
	Verdict       Verdict         `json:"verdict"`
	EvidenceIDs   []EvidenceID    `json:"evidence_ids"`
	Notes         string          `json:"notes,omitempty"`
}

// Errors for claim validation.
var (
	ErrEmptyStatement        = errors.New("claim statement must be non-empty")
	ErrInvalidStatus         = errors.New("claim status must be one of the declared ClaimStatus values")
	ErrInvalidVerdict        = errors.New("claim verdict must be one of the declared Verdict values")
	ErrInvalidEvidenceID     = errors.New("evidence ID must pass ValidateEvidenceID")
	ErrInvalidClaimID        = errors.New("claim ID must pass ValidateClaimID")
	ErrInvalidRunID          = errors.New("run ID must pass runbundle.ValidateRunID")
	ErrUpdatedBeforeCreated  = errors.New("updated_at must not be before created_at")
	ErrSchemaVersionMismatch = errors.New("schema_version must be exactly leamas.claim.v1")
)

// IsValidClaimStatus checks if the status is valid.
func IsValidClaimStatus(status ClaimStatus) bool {
	switch status {
	case ClaimStatusOpen, ClaimStatusSupported, ClaimStatusRejected, ClaimStatusUnknown:
		return true
	}
	return false
}

// IsValidVerdict checks if the verdict is valid.
func IsValidVerdict(verdict Verdict) bool {
	switch verdict {
	case VerdictUnreviewed, VerdictPass, VerdictFail, VerdictMixed:
		return true
	}
	return false
}

// NewClaim creates a new Claim with default values.
func NewClaim(id ClaimID, runID runbundle.RunID, statement string, now time.Time) (Claim, error) {
	// Validate claim ID
	if err := ValidateClaimID(id); err != nil {
		return Claim{}, err
	}

	// Validate run ID
	if err := runbundle.ValidateRunID(runID); err != nil {
		return Claim{}, ErrInvalidRunID
	}

	// Validate statement
	if statement == "" {
		return Claim{}, ErrEmptyStatement
	}

	return Claim{
		SchemaVersion: ClaimSchemaVersion,
		ID:            id,
		RunID:         runID,
		CreatedAt:     now,
		UpdatedAt:     now,
		Statement:     statement,
		Status:        ClaimStatusOpen,
		Verdict:       VerdictUnreviewed,
		EvidenceIDs:   []EvidenceID{},
	}, nil
}

// Validate validates a claim and returns an error if invalid.
func (c Claim) Validate() error {
	// Schema version check
	if c.SchemaVersion != ClaimSchemaVersion {
		return ErrSchemaVersionMismatch
	}

	// ID check
	if err := ValidateClaimID(c.ID); err != nil {
		return ErrInvalidClaimID
	}

	// Run ID check
	if err := runbundle.ValidateRunID(c.RunID); err != nil {
		return ErrInvalidRunID
	}

	// Statement check
	if c.Statement == "" {
		return ErrEmptyStatement
	}

	// Status check
	if !IsValidClaimStatus(c.Status) {
		return ErrInvalidStatus
	}

	// Verdict check
	if !IsValidVerdict(c.Verdict) {
		return ErrInvalidVerdict
	}

	// Evidence IDs check
	for _, eid := range c.EvidenceIDs {
		if err := ValidateEvidenceID(eid); err != nil {
			return ErrInvalidEvidenceID
		}
	}

	// Time check
	if c.UpdatedAt.Before(c.CreatedAt) {
		return ErrUpdatedBeforeCreated
	}

	return nil
}

// AddEvidence adds an evidence ID to the claim if not already present.
// Returns true if the evidence was added, false if already present.
func (c *Claim) AddEvidence(evidenceID EvidenceID) (bool, error) {
	// Validate evidence ID
	if err := ValidateEvidenceID(evidenceID); err != nil {
		return false, err
	}

	// Check if already present
	for _, existing := range c.EvidenceIDs {
		if existing == evidenceID {
			return false, nil
		}
	}

	c.EvidenceIDs = append(c.EvidenceIDs, evidenceID)
	return true, nil
}

// HasEvidence checks if the claim has the given evidence ID.
func (c Claim) HasEvidence(evidenceID EvidenceID) bool {
	for _, eid := range c.EvidenceIDs {
		if eid == evidenceID {
			return true
		}
	}
	return false
}
