// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_barrier.go owns the
// in-memory publication barrier PrepareClosureEvidenceForPublication.
//
// The barrier is the single authority that may emit final
// publication bytes. It derives completeness, validates the
// canonical struct, marshals the bytes exactly once, and
// returns the typed PublicationCandidate. The barrier does
// NOT write to the filesystem; that step belongs to B3.
//
// Splitting the file at the barrier boundary keeps the
// production B2 surface under the LLM-friendly 400-line
// threshold while preserving the single closure over the
// descriptor that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01
// requires.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// PublicationCandidate is the typed output of the publication
// barrier. The struct is the only object the barrier emits.
// It pairs the canonical evidence with the exact bytes the
// barrier produced and the SHA-256 derived from those bytes.
// The SHA-256 is external metadata and is NEVER embedded in
// the JSON document.
type PublicationCandidate struct {
	// Evidence is the canonical evidence document.
	Evidence ClosureEvidence
	// Bytes is the exact JSON document the barrier produced.
	// A second JSON decode of these bytes returns io.EOF.
	Bytes []byte
	// SHA256 is the lowercase hex SHA-256 of Bytes.
	SHA256 string
}

// ErrIncompleteEvidence is the typed error the barrier returns
// when the canonical predicate derives EvidenceIncomplete. The
// barrier never returns a partial PublicationCandidate.
var ErrIncompleteEvidence = errors.New("evidence: candidate is incomplete; cannot cross publication barrier")

// PrepareClosureEvidenceForPublication is the single barrier
// entry point. It re-derives completeness, validates the
// canonical struct, marshals the final bytes exactly once, and
// returns the typed PublicationCandidate. Any failure path
// returns a zero PublicationCandidate and a typed error.
//
// The barrier never writes to the filesystem. The caller (B3)
// owns that step.
func PrepareClosureEvidenceForPublication(candidate ClosureEvidence) (PublicationCandidate, error) {
	if DeriveClosureEvidenceCompleteness(candidate) != EvidenceComplete {
		return PublicationCandidate{}, ErrIncompleteEvidence
	}
	if err := ValidateClosureEvidence(candidate); err != nil {
		return PublicationCandidate{}, fmt.Errorf("evidence: validation failed: %w", err)
	}
	bytes, err := MarshalEvidence(candidate)
	if err != nil {
		return PublicationCandidate{}, fmt.Errorf("evidence: marshal failed: %w", err)
	}
	sum := sha256.Sum256(bytes)
	return PublicationCandidate{
		Evidence: candidate,
		Bytes:    bytes,
		SHA256:   hex.EncodeToString(sum[:]),
	}, nil
}

// MarshalEvidence is the canonical JSON encoder for the
// evidence document. It uses json.Marshal (no trailing newline)
// so the barrier's output is exactly one JSON document: a second
// Decoder.Decode on the same bytes returns io.EOF. Test
// TestClosureEvidenceCanonicalSerialization asserts this.
func MarshalEvidence(candidate ClosureEvidence) ([]byte, error) {
	return json.Marshal(candidate)
}

// ComputeEvidenceSHA256 is the external-metadata helper. The
// function returns SHA-256 over the supplied bytes. The barrier
// uses it as a courtesy; the canonical hash is the one stored
// on PublicationCandidate.SHA256.
func ComputeEvidenceSHA256(bytes []byte) string {
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}
