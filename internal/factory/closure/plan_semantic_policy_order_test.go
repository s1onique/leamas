package closure

import (
	"testing"
)

// TestPolicyFieldOrderDocumentsActualContractOrder verifies that PolicyFieldOrder
// returns the exact field order from the contract descriptor.
func TestPolicyFieldOrderDocumentsActualContractOrder(t *testing.T) {
	order := PolicyFieldOrder()
	// The actual order from the contract: require_clean_before, require_clean_after,
	// forbid_tracked_full_digests, require_diff_check
	want := []string{
		"require_clean_before",
		"require_clean_after",
		"forbid_tracked_full_digests",
		"require_diff_check",
	}
	if len(order) != len(want) {
		t.Fatalf("PolicyFieldOrder length = %d, want %d", len(order), len(want))
	}
	for i, w := range want {
		if order[i] != w {
			t.Errorf("PolicyFieldOrder[%d] = %q, want %q", i, order[i], w)
		}
	}
}
