// Package claim provides typed domain models for claims and evidence
// in Leamas verification witness artifacts.
package claim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// Errors for store operations.
var (
	ErrClaimNotFound      = errors.New("claim not found")
	ErrEvidenceNotFound   = errors.New("evidence not found")
	ErrRunIDMismatch      = errors.New("run ID mismatch between artifact and bundle")
	ErrInvalidClaim       = errors.New("claim failed validation")
	ErrInvalidEvidence    = errors.New("evidence failed validation")
	ErrClaimIDMismatch    = errors.New("claim ID mismatch between filename and contents")
	ErrEvidenceIDMismatch = errors.New("evidence ID mismatch between filename and contents")
)

// Store provides filesystem-based storage for claims and evidence
// within an existing run bundle.
type Store struct {
	Bundle runbundle.Bundle
}

// NewStore creates a new Store for the given run bundle.
func NewStore(bundle runbundle.Bundle) Store {
	return Store{Bundle: bundle}
}

// claimPath returns the filesystem path for a claim.
func claimPath(bundle runbundle.Bundle, id ClaimID) string {
	return filepath.Join(bundle.Path, "claims", string(id)+".json")
}

// evidencePath returns the filesystem path for evidence.
func evidencePath(bundle runbundle.Bundle, id EvidenceID) string {
	return filepath.Join(bundle.Path, "evidence", string(id)+".json")
}

// WriteClaim writes a claim to the bundle's claims directory.
// It validates the claim before writing.
func (s Store) WriteClaim(c Claim) error {
	// Validate the claim
	if err := c.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidClaim, err)
	}

	// Check run ID matches bundle
	if c.RunID != s.Bundle.ID {
		return ErrRunIDMismatch
	}

	// Marshal to JSON with indentation
	data, err := MarshalClaimJSON(c)
	if err != nil {
		return fmt.Errorf("failed to marshal claim: %w", err)
	}

	// Write the file
	path := claimPath(s.Bundle, c.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write claim: %w", err)
	}

	return nil
}

// ReadClaim reads a claim from the bundle's claims directory.
// It strictly decodes and validates the claim.
func (s Store) ReadClaim(id ClaimID) (Claim, error) {
	// Validate the claim ID
	if err := ValidateClaimID(id); err != nil {
		return Claim{}, err
	}

	// Read the file
	path := claimPath(s.Bundle, id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Claim{}, ErrClaimNotFound
		}
		return Claim{}, fmt.Errorf("failed to read claim: %w", err)
	}

	// Strict decode
	c, err := StrictDecodeClaim(data)
	if err != nil {
		return Claim{}, fmt.Errorf("failed to decode claim: %w", err)
	}

	// Verify decoded ID matches requested ID
	if c.ID != id {
		return Claim{}, ErrClaimIDMismatch
	}

	// Verify run ID matches bundle
	if c.RunID != s.Bundle.ID {
		return Claim{}, ErrRunIDMismatch
	}

	// Validate
	if err := c.Validate(); err != nil {
		return Claim{}, fmt.Errorf("%w: %v", ErrInvalidClaim, err)
	}

	return *c, nil
}

// WriteEvidence writes evidence to the bundle's evidence directory.
// It validates the evidence before writing.
func (s Store) WriteEvidence(e Evidence) error {
	// Validate the evidence
	if err := e.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEvidence, err)
	}

	// Check run ID matches bundle
	if e.RunID != s.Bundle.ID {
		return ErrRunIDMismatch
	}

	// Marshal to JSON with indentation
	data, err := MarshalEvidenceJSON(e)
	if err != nil {
		return fmt.Errorf("failed to marshal evidence: %w", err)
	}

	// Write the file
	path := evidencePath(s.Bundle, e.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write evidence: %w", err)
	}

	return nil
}

// ReadEvidence reads evidence from the bundle's evidence directory.
// It strictly decodes and validates the evidence.
func (s Store) ReadEvidence(id EvidenceID) (Evidence, error) {
	// Validate the evidence ID
	if err := ValidateEvidenceID(id); err != nil {
		return Evidence{}, err
	}

	// Read the file
	path := evidencePath(s.Bundle, id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Evidence{}, ErrEvidenceNotFound
		}
		return Evidence{}, fmt.Errorf("failed to read evidence: %w", err)
	}

	// Strict decode
	e, err := StrictDecodeEvidence(data)
	if err != nil {
		return Evidence{}, fmt.Errorf("failed to decode evidence: %w", err)
	}

	// Verify decoded ID matches requested ID
	if e.ID != id {
		return Evidence{}, ErrEvidenceIDMismatch
	}

	// Verify run ID matches bundle
	if e.RunID != s.Bundle.ID {
		return Evidence{}, ErrRunIDMismatch
	}

	// Validate
	if err := e.Validate(); err != nil {
		return Evidence{}, fmt.Errorf("%w: %v", ErrInvalidEvidence, err)
	}

	return *e, nil
}

// AddEvidenceToClaim adds an evidence ID to a claim.
// It reads the claim, validates the evidence ID, appends if missing,
// updates updated_at, and writes the claim back.
// This operation is idempotent for duplicate evidence IDs.
func (s Store) AddEvidenceToClaim(claimID ClaimID, evidenceID EvidenceID, now func() time.Time) (Claim, error) {
	// Validate claim ID
	if err := ValidateClaimID(claimID); err != nil {
		return Claim{}, err
	}

	// Validate evidence ID
	if err := ValidateEvidenceID(evidenceID); err != nil {
		return Claim{}, err
	}

	// Safe nil clock guard
	if now == nil {
		now = time.Now
	}

	// Read existing claim
	claim, err := s.ReadClaim(claimID)
	if err != nil {
		return Claim{}, err
	}

	// Check if evidence already linked
	for _, eid := range claim.EvidenceIDs {
		if eid == evidenceID {
			// Already linked, return as-is (idempotent)
			return claim, nil
		}
	}

	// Add evidence ID
	claim.EvidenceIDs = append(claim.EvidenceIDs, evidenceID)

	// Update timestamp
	claim.UpdatedAt = now()

	// Write back
	if err := s.WriteClaim(claim); err != nil {
		return Claim{}, err
	}

	return claim, nil
}
