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

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// publicationCandidateToken is an unexported capability held
// by every PublicationCandidate. Only PrepareClosureEvidenceForPublication
// (same package) can mint a token, so the type is unforgeable
// outside the B2 barrier.
type publicationCandidateToken struct{}

// PublicationCandidate is the typed output of the publication
// barrier. The struct is the only object the barrier emits.
// It pairs the canonical evidence with the exact bytes the
// barrier produced and the SHA-256 derived from those bytes.
// The SHA-256 is external metadata and is NEVER embedded in
// the JSON document.
//
// All fields are unexported. Outside the `evidence` package,
// the only way to obtain a PublicationCandidate is to call
// PrepareClosureEvidenceForPublication; the unexported token
// type prevents any external code from constructing one by
// literal initialization. B3 reads the bytes / digest through
// the accessor methods only.
type PublicationCandidate struct {
	_       publicationCandidateToken
	evidence ClosureEvidence
	bytes    []byte
	sha256   string
}

// Bytes returns the exact JSON document the barrier produced.
func (p PublicationCandidate) Bytes() []byte { return p.bytes }

// SHA256 returns the lowercase hex SHA-256 of Bytes.
func (p PublicationCandidate) SHA256() string { return p.sha256 }

// Document returns the canonical ClosureEvidence struct.
func (p PublicationCandidate) Document() ClosureEvidence { return p.evidence }

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
		evidence: candidate,
		bytes:    bytes,
		sha256:   hex.EncodeToString(sum[:]),
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

// UnmarshalClosureEvidence is the strict single-document
// decoder for the canonical evidence record. B2-R1 added the
// entry point because the previous B2 implementation decoded
// the evidence document with json.Unmarshal, which silently
// discards unknown object keys. For an authority document a
// caller that injects "completeness":"COMPLETE" or other
// authority-looking fields must be rejected, not absorbed.
//
// B2-R2 fixes:
//
//  1. DisallowUnknownFields() rejects unknown object keys.
//  2. Decoder.More() is the wrong primitive for top-level
//     EOF; the decoder uses a second Decode that MUST
//     return io.EOF.
//
// B2-R3 fix:
//
//  3. encoding/json v1 silently accepts duplicate object
//     member names. The plancontract leaf already rejects
//     duplicate keys via its strict scanner; B2-R3 routes
//     through plancontract.StrictDecode so the closure
//     evidence record and the Plan Contract documents share
//     the same duplicate-field rejection contract.
//
// The decoder in this order:
//
//  1. DisallowUnknownFields() rejects unknown object keys.
//  2. plancontract.StrictDecode scans and rejects duplicate
//     keys at every depth.
//  3. Second Decode MUST return io.EOF; any other return
//     rejects.
func UnmarshalClosureEvidence(data []byte) (ClosureEvidence, error) {
	var out ClosureEvidence
	if err := plancontract.StrictDecode(data, &out); err != nil {
		return ClosureEvidence{}, fmt.Errorf("evidence: strict decode failed: %w", err)
	}
	return out, nil
}

// ComputeEvidenceSHA256 is the external-metadata helper. The
// function returns SHA-256 over the supplied bytes. The barrier
// uses it as a courtesy; the canonical hash is the one stored
// on PublicationCandidate.SHA256()().
func ComputeEvidenceSHA256(bytes []byte) string {
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}
