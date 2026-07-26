// SPDX-License-Identifier: Apache-2.0

package authority

import (
	"testing"
)

// TestPortableRunnerCapabilityContract verifies that the
// closure_protocol_v2_portable_runner_authority capability is properly declared.
func TestPortableRunnerCapabilityContract(t *testing.T) {
	// Verify the capability constant is defined
	if CapClosureProtocolV2PortableRunnerAuthority != "closure_protocol_v2_portable_runner_authority" {
		t.Fatalf("unexpected capability name: %s", CapClosureProtocolV2PortableRunnerAuthority)
	}

	// Verify it's in the capabilities map
	embedded := Embedded()
	level, ok := embedded[CapClosureProtocolV2PortableRunnerAuthority]
	if !ok {
		t.Fatalf("capability %s not found in embedded capabilities", CapClosureProtocolV2PortableRunnerAuthority)
	}

	if level < 1 {
		t.Fatalf("capability level should be >= 1, got %d", level)
	}

	// Verify SetEmbedded works for testing
	originalLevel := level
	SetEmbedded(CapClosureProtocolV2PortableRunnerAuthority, 2)
	newEmbedded := Embedded()
	newLevel := newEmbedded[CapClosureProtocolV2PortableRunnerAuthority]
	if newLevel != 2 {
		t.Fatalf("SetEmbedded failed: expected 2, got %d", newLevel)
	}

	// Restore original level
	SetEmbedded(CapClosureProtocolV2PortableRunnerAuthority, originalLevel)
}
