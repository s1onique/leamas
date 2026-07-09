// Package claim provides typed domain models for claims and evidence
// in Leamas verification witness artifacts.
package claim

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictDecodeClaim decodes JSON into a Claim, rejecting unknown fields.
func StrictDecodeClaim(data []byte) (*Claim, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var c Claim
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("failed to decode claim JSON: %w", err)
	}
	return &c, nil
}

// StrictDecodeEvidence decodes JSON into Evidence, rejecting unknown fields.
func StrictDecodeEvidence(data []byte) (*Evidence, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var e Evidence
	if err := dec.Decode(&e); err != nil {
		return nil, fmt.Errorf("failed to decode evidence JSON: %w", err)
	}
	return &e, nil
}

// MarshalClaimJSON marshals a Claim to JSON with indentation.
func MarshalClaimJSON(c Claim) ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// MarshalEvidenceJSON marshals Evidence to JSON with indentation.
func MarshalEvidenceJSON(e Evidence) ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}
