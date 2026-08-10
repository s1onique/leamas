// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_placeholder_immutability_test.go
// is the B2-R7-R2 parity test that asserts the canonical
// placeholder set is read-only via ExactClosurePlaceholdersCopy.
//
// PLACEHOLDER_AUTHORITY_NOT_EXTERNALLY_MUTABLE: true.
// External callers cannot mutate the canonical set; the
// only public surface is ExactClosurePlaceholdersCopy
// (returns a fresh map each call) and ContainsClosurePlaceholder
// (a predicate). The test mutates the returned copy and
// asserts the canonical validation authority is unchanged.
package plancontract

import (
	"testing"
)

// TestExactClosurePlaceholdersCopyIsImmutable verifies
// that mutating the copy returned by
// ExactClosurePlaceholdersCopy does NOT affect the
// canonical validation authority. The canonical map is
// private; only the predicate ContainsClosurePlaceholder
// and the copy function expose it. External mutation must
// be a no-op against the canonical set.
func TestExactClosurePlaceholdersCopyIsImmutable(t *testing.T) {
	t.Parallel()

	// Capture the canonical probe value via the public
	// predicate; "TBD" is one of the canonical entries.
	if !ContainsClosurePlaceholder("TBD") {
		t.Fatalf("expected ContainsClosurePlaceholder(\"TBD\") to be true")
	}

	// Take a copy and mutate it.
	mutated := ExactClosurePlaceholdersCopy()
	mutated["NEW_KEY"] = struct{}{}
	delete(mutated, "TBD")

	// The canonical authority must be unchanged: the
	// predicate still detects "TBD" and rejects the new
	// (non-canonical) "NEW_KEY".
	if !ContainsClosurePlaceholder("TBD") {
		t.Fatalf("canonical placeholder set was mutated; B2-R7-R2 PLACEHOLDER_AUTHORITY_NOT_EXTERNALLY_MUTABLE invariant violated")
	}
	if ContainsClosurePlaceholder("NEW_KEY") {
		t.Fatalf("external mutation leaked into canonical validation authority")
	}

	// A fresh copy must be independent of any previous
	// mutation.
	fresh := ExactClosurePlaceholdersCopy()
	if _, found := fresh["NEW_KEY"]; found {
		t.Fatalf("fresh copy retained a previously mutated key")
	}
	if _, found := fresh["TBD"]; !found {
		t.Fatalf("fresh copy lost a canonical key after a previous mutation")
	}
}

// TestEmbeddedClosurePlaceholdersCopyIsImmutable mirrors
// the test above for the embedded-placeholder marker
// list. Mutation of the returned slice must not affect
// the canonical validation authority.
func TestEmbeddedClosurePlaceholdersCopyIsImmutable(t *testing.T) {
	t.Parallel()

	fresh := EmbeddedClosurePlaceholdersCopy()
	if len(fresh) == 0 {
		t.Fatalf("canonical embedded-placeholder list must be non-empty")
	}

	// The predicate does not cover embedded markers
	// directly, but the canonical set is consulted by the
	// validation authorities; the test mutates the copy
	// and asserts the canonical slice is unchanged.
	original := make([]string, len(fresh))
	copy(original, fresh)

	fresh[0] = "MUTATED"

	again := EmbeddedClosurePlaceholdersCopy()
	for i, v := range original {
		if again[i] != v {
			t.Fatalf("canonical embedded-placeholder list was mutated at index %d: got %q want %q",
				i, again[i], v)
		}
	}
}
