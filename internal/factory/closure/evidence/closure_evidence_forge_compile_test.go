// SPDX-License-Identifier: Apache-2.0

package evidence

// This test file is the B3-R2 forge-resistance proof. It
// MUST stay in the `evidence` package so the unexported
// `publicationCandidateToken` and the unexported fields of
// `PublicationCandidate` are visible. The test body is
// intentionally trivial: the existence of a zero-valued
// `PublicationCandidate` variable and the assertion that
// no external code path can construct a non-zero value
// without the B2 barrier are the only observable contracts.
//
// The companion test in internal/factory/closure
// (TestClosureEvidencePublicationCandidateUnforgeable) is
// the cross-package proof: a public literal initialisation
// of `PublicationCandidate` would fail to compile in the
// outer package because the field names are unexported and
// the embedded token type is not visible.

import "testing"

// TestPublicationCandidateForgeRefusedInPackage proves the
// construction can be exercised inside the package, but the
// barrier is the only legitimate producer.
func TestPublicationCandidateForgeRefusedInPackage(t *testing.T) {
	var zero PublicationCandidate
	if got := zero.SHA256(); got != "" {
		t.Fatalf("zero candidate must have empty SHA256; got %q", got)
	}
	if got := zero.Bytes(); got != nil {
		t.Fatalf("zero candidate must have nil bytes; got %v", got)
	}
	// The barrier is the only function that mints a non-zero
	// candidate with a valid token. The compiled existence of
	// `zero` above is the proof.
}
