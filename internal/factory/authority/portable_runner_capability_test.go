// SPDX-License-Identifier: Apache-2.0

// Package authority: portable_runner_capability_test.go asserts the
// closure_protocol_v2_portable_runner_authority capability surface
// required by ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-
// PORTABLE-RUNNER-AUTHORITY01-CORRECTION02.
//
// The capability must be:
//   - declared in the production capability constants
//   - present in the embedded capability table
//   - present in the required-capabilities file
//
// The test exercises the production capability table and the
// required-capabilities loader; it does not mock the surface.
package authority

import (
	"testing"
)

// TestPortableRunnerCapabilityContract asserts the portable runner
// authority capability contract: the capability name must be present
// in both the embedded table and the required-capabilities file.
func TestPortableRunnerCapabilityContract(t *testing.T) {
	// 1. Embedded table must expose the capability.
	embedded := Embedded()
	if _, ok := embedded[CapClosureProtocolV2PortableRunnerAuth]; !ok {
		t.Fatalf("embedded capabilities missing %q", CapClosureProtocolV2PortableRunnerAuth)
	}
	if embedded[CapClosureProtocolV2PortableRunnerAuth] < 1 {
		t.Fatalf("embedded %s level=%d want >=1", CapClosureProtocolV2PortableRunnerAuth, embedded[CapClosureProtocolV2PortableRunnerAuth])
	}

	// 2. Required-capabilities file must declare the capability.
	repoRoot, err := FindRepositoryRoot("")
	if err != nil {
		t.Fatalf("FindRepositoryRoot: %v", err)
	}
	path := DefaultPath(repoRoot)
	required, err := LoadRequiredCanonical(path)
	if err != nil {
		t.Fatalf("LoadRequiredCanonical(%s): %v", path, err)
	}
	if _, ok := required.Raw[CapClosureProtocolV2PortableRunnerAuth]; !ok {
		t.Fatalf("required-capabilities file missing %q (got %v)", CapClosureProtocolV2PortableRunnerAuth, required.Raw)
	}
	if required.Raw[CapClosureProtocolV2PortableRunnerAuth] < 1 {
		t.Fatalf("required %s level=%d want >=1", CapClosureProtocolV2PortableRunnerAuth, required.Raw[CapClosureProtocolV2PortableRunnerAuth])
	}

	// 3. Embedded must satisfy required floor.
	if err := required.SatisfiedBy(SnapshotEmbedded()); err != nil {
		t.Fatalf("SatisfiedBy: %v", err)
	}
}

// TestPortableRunnerCapabilityStaleBinary simulates the stale-binary
// regression: a binary that lacks the portable runner authority
// capability must be rejected by the capability gate.
func TestPortableRunnerCapabilityStaleBinary(t *testing.T) {
	original := capabilities[CapClosureProtocolV2PortableRunnerAuth]
	t.Cleanup(func() { SetEmbedded(CapClosureProtocolV2PortableRunnerAuth, original) })

	SetEmbedded(CapClosureProtocolV2PortableRunnerAuth, 0)

	required := &RequiredCapabilities{Raw: map[string]int{
		CapClosureProtocolV2PortableRunnerAuth: 1,
	}}
	err := required.SatisfiedBy(SnapshotEmbedded())
	if err == nil {
		t.Fatal("expected CapabilityGap when portable_runner_authority=0")
	}
	if err.Error() == "" {
		t.Fatal("CapabilityGap.Error() must not be empty")
	}
}
