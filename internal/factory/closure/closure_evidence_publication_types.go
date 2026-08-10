// SPDX-License-Identifier: Apache-2.0

package closure

// EvidencePublicationState is the typed result of a Publish call.
// The state machine is monotonic in visibility: the only legal
// transitions are not_published -> json_visible (the transient
// state when JSON link succeeded but sidecar did not) ->
// pair_visible -> pair_durable (or pair_visible_durability_unconfirmed).
type EvidencePublicationState int

const (
	// EvidencePublicationNotPublished means the publication
	// sequence did not make either artifact visible. The
	// destination is guaranteed untouched and no temp files
	// remain on disk.
	EvidencePublicationNotPublished EvidencePublicationState = iota

	// EvidencePublicationJSONVisible means the JSON was made
	// visible but the sidecar was not. The state never
	// collapses back to not_published.
	EvidencePublicationJSONVisible

	// EvidencePublicationPairVisible means both files are
	// visible but the parent directory fsync has not been
	// proven.
	EvidencePublicationPairVisible

	// EvidencePublicationPairDurable means both files are
	// visible AND the parent directory was successfully fsynced.
	EvidencePublicationPairDurable

	// EvidencePublicationPairVisibleDurabilityUnconfirmed
	// means both files are visible but the parent directory
	// fsync (or the post-publish observation) failed. The
	// bytes are on disk; durability is unconfirmed.
	EvidencePublicationPairVisibleDurabilityUnconfirmed
)

// String returns the canonical snake_case token.
func (s EvidencePublicationState) String() string {
	switch s {
	case EvidencePublicationNotPublished:
		return "not_published"
	case EvidencePublicationJSONVisible:
		return "json_visible"
	case EvidencePublicationPairVisible:
		return "pair_visible"
	case EvidencePublicationPairDurable:
		return "pair_durable"
	case EvidencePublicationPairVisibleDurabilityUnconfirmed:
		return "pair_visible_durability_unconfirmed"
	}
	return "unknown_publication_state"
}

// EvidencePublicationResult is the structured outcome of a
// Publish call. State is authoritative.
type EvidencePublicationResult struct {
	State         EvidencePublicationState
	CanonicalJSON string
	CanonicalSide string
	Err           error
}
