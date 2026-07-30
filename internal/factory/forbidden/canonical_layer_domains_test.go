// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// TestCanonicalLayerDomainConstructorIsolatesCallerSlices proves
// that mutating the input slices after construction does not affect the
// resulting authorityLayerDomains value.
func TestCanonicalLayerDomainConstructorIsolatesCallerSlices(t *testing.T) {
	raw := []string{"example.test/policy/raw"}
	adapter := []string{"example.test/policy/adapter"}
	gate := []string{"example.test/policy/gate"}

	policy := newAuthorityLayerDomains(raw, adapter, gate)

	raw[0] = "tampered.example.test/policy/raw"
	adapter[0] = "tampered.example.test/policy/adapter"
	gate[0] = "tampered.example.test/policy/gate"

	if !policy.allows(AuthorityLayerRaw, "example.test/policy/raw") {
		t.Errorf("raw layer did not retain original package after caller mutation: %+v", policy)
	}
	if !policy.allows(AuthorityLayerAdapter, "example.test/policy/adapter") {
		t.Errorf("adapter layer did not retain original package after caller mutation: %+v", policy)
	}
	if !policy.allows(AuthorityLayerGate, "example.test/policy/gate") {
		t.Errorf("gate layer did not retain original package after caller mutation: %+v", policy)
	}
	if policy.allows(AuthorityLayerRaw, "tampered.example.test/policy/raw") {
		t.Errorf("raw layer accepted tampered caller package: %+v", policy)
	}
}

// TestCanonicalLayerDomainDeduplicatesAndSorts proves each layer slice
// is sorted and deduplicated regardless of input order.
func TestCanonicalLayerDomainDeduplicatesAndSorts(t *testing.T) {
	policy := newAuthorityLayerDomains(
		[]string{"x.example.test/policy/c", "a.example.test/policy/c", "b.example.test/policy/c", "a.example.test/policy/c"},
		[]string{"b.adapter", "a.adapter"},
		[]string{"z.gate", "a.gate"},
	)
	expectedRaw := []string{"a.example.test/policy/c", "b.example.test/policy/c", "x.example.test/policy/c"}
	if !reflect.DeepEqual([]string(policy.raw), expectedRaw) {
		t.Errorf("raw deduplication/sort failed: got %+v want %+v", policy.raw, expectedRaw)
	}
	if !reflect.DeepEqual([]string(policy.adapter), []string{"a.adapter", "b.adapter"}) {
		t.Errorf("adapter deduplication/sort failed: %+v", policy.adapter)
	}
	if !reflect.DeepEqual([]string(policy.gate), []string{"a.gate", "z.gate"}) {
		t.Errorf("gate deduplication/sort failed: %+v", policy.gate)
	}
}

// TestCanonicalLayerDomainValidDomains proves each production-style symbol
// resolves when its package matches the configured layer in a real
// fixture. The fixture's runWithLayerDomains() wraps the canonical
// pipeline so we exercise the same seam without depending on the
// production repository root.
func TestCanonicalLayerDomainValidDomains(t *testing.T) {
	fixture := newCanonicalFixture(t)
	fixture.write("rawcap/rawcap.go", "package rawcap\nfunc Cap() {}\n")
	fixture.write("adaptercap/adaptercap.go", "package adaptercap\ntype Runner struct{}\nfunc (*Runner) Run() {}\n")
	fixture.write("gatecap/gatecap.go", "package gatecap\nfunc Wrap() {}\n")
	fixture.write("caller/caller.go", "package caller\n")
	policy := newAuthorityLayerDomains(
		[]string{fixture.packagePath("rawcap")},
		[]string{fixture.packagePath("adaptercap")},
		[]string{fixture.packagePath("gatecap")},
	)
	symbols := []ProtectedSymbol{
		fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("rawcap"), "Cap", ProtectedPackageFunction, ""),
		fixtureSymbol(AuthorityLayerAdapter, fixture.packagePath("adaptercap"), "Run", ProtectedMethod, "Runner"),
		fixtureSymbol(AuthorityLayerGate, fixture.packagePath("gatecap"), "Wrap", ProtectedPackageFunction, ""),
	}
	result := fixture.runWithLayerDomains(symbols, nil, policy)
	if result.Stats.ResolvedProtectedObjects != 3 {
		t.Fatalf("resolved objects = %d, want 3", result.Stats.ResolvedProtectedObjects)
	}
}

// TestCanonicalLayerDomainCrossLayerRejection proves each cross-layer
// pairing fails closed. The symbol's declared layer must match the
// configured layer package; cross-package pairings are rejected.
func TestCanonicalLayerDomainCrossLayerRejection(t *testing.T) {
	fixture := newCanonicalFixture(t)
	fixture.write("rawcap/rawcap.go", "package rawcap\nfunc Cap() {}\n")
	fixture.write("adaptercap/adaptercap.go", "package adaptercap\ntype Runner struct{}\nfunc (*Runner) Run() {}\n")
	fixture.write("gatecap/gatecap.go", "package gatecap\nfunc Wrap() {}\n")
	fixture.write("caller/caller.go", "package caller\n")
	policy := newAuthorityLayerDomains(
		[]string{fixture.packagePath("rawcap")},
		[]string{fixture.packagePath("adaptercap")},
		[]string{fixture.packagePath("gatecap")},
	)
	cases := []struct {
		name   string
		symbol ProtectedSymbol
	}{
		{
			name:   "raw_in_adapter",
			symbol: fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("adaptercap"), "Cap", ProtectedPackageFunction, ""),
		},
		{
			name:   "raw_in_gate",
			symbol: fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("gatecap"), "Cap", ProtectedPackageFunction, ""),
		},
		{
			name:   "adapter_in_raw",
			symbol: fixtureSymbol(AuthorityLayerAdapter, fixture.packagePath("rawcap"), "Run", ProtectedMethod, "Runner"),
		},
		{
			name:   "adapter_in_gate",
			symbol: fixtureSymbol(AuthorityLayerAdapter, fixture.packagePath("gatecap"), "Run", ProtectedMethod, "Runner"),
		},
		{
			name:   "gate_in_raw",
			symbol: fixtureSymbol(AuthorityLayerGate, fixture.packagePath("rawcap"), "Wrap", ProtectedPackageFunction, ""),
		},
		{
			name:   "gate_in_adapter",
			symbol: fixtureSymbol(AuthorityLayerGate, fixture.packagePath("adaptercap"), "Wrap", ProtectedPackageFunction, ""),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixture.runWithLayerDomains([]ProtectedSymbol{tc.symbol}, nil, policy)
			if result.Stats.ResolvedProtectedObjects != 0 {
				t.Fatalf("resolved objects = %d, want 0", result.Stats.ResolvedProtectedObjects)
			}
			requireFindingKind(t, result.Findings, "authority_policy_layer_mismatch")
		})
	}
}

// TestCanonicalLayerDomainMissingPolicyFailsClosed proves a non-empty
// protected-symbol inventory combined with an empty domain policy
// fails closed with authority_policy_layer_policy_missing and registers
// zero objects.
func TestCanonicalLayerDomainMissingPolicyFailsClosed(t *testing.T) {
	fixture := newCanonicalFixture(t)
	fixture.write("rawcap/rawcap.go", "package rawcap\nfunc Cap() {}\n")
	fixture.write("caller/caller.go", "package caller\n")
	symbol := fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("rawcap"), "Cap", ProtectedPackageFunction, "")
	result := fixture.runWithLayerDomains([]ProtectedSymbol{symbol}, nil, authorityLayerDomains{})
	if result.Stats.ResolvedProtectedObjects != 0 {
		t.Fatalf("resolved objects = %d, want 0", result.Stats.ResolvedProtectedObjects)
	}
	requireFindingKind(t, result.Findings, "authority_policy_layer_policy_missing")
}

// TestCanonicalLayerDomainLegacyFixtureIsCompatible proves that an
// existing fixture run with the compatibility policy derived from the
// symbol inventory still resolves. This is the seam's explicit
// compatibility contract for the established tests.
func TestCanonicalLayerDomainLegacyFixtureIsCompatible(t *testing.T) {
	fixture := newCanonicalFixture(t)
	fixture.write("dupcode/dupcode.go", "package dupcode\nfunc Cap() {}\n")
	fixture.write("verifier/verifier.go", "package verifier\nimport p \"example.test/policy/dupcode\"\nfunc Allowed() { p.Cap() }\n")
	symbols := []ProtectedSymbol{
		fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("dupcode"), "Cap", ProtectedPackageFunction, ""),
	}
	result := fixture.runWithLayerDomains(symbols, nil, fixtureAuthorityLayerDomains(symbols))
	if result.Stats.ResolvedProtectedObjects != 1 {
		t.Fatalf("resolved objects = %d, want 1", result.Stats.ResolvedProtectedObjects)
	}
}

// TestCanonicalLayerDomainAllowsUnknownLayerReturnsFalse proves the
// allows query is fail-closed for unknown layer values.
func TestCanonicalLayerDomainAllowsUnknownLayerReturnsFalse(t *testing.T) {
	policy := productionAuthorityLayerDomains()
	if policy.allows(AuthorityLayer(""), "example.test/policy/raw") {
		t.Errorf("empty layer should not be allowed")
	}
	if policy.allows(AuthorityLayer("unknown"), "example.test/policy/raw") {
		t.Errorf("unknown layer should not be allowed")
	}
}

// TestCanonicalLayerDomainProductionMatchesReality proves the
// configured production policy matches the established real package
// domains. If the production package paths ever change, this test
// fails closed.
func TestCanonicalLayerDomainProductionMatchesReality(t *testing.T) {
	policy := productionAuthorityLayerDomains()
	if !policy.allows(AuthorityLayerRaw, "github.com/s1onique/leamas/internal/factory/dupcode") {
		t.Errorf("production policy missing dupcode package")
	}
	if !policy.allows(AuthorityLayerAdapter, "github.com/s1onique/leamas/internal/factory/protectedverifier") {
		t.Errorf("production policy missing protectedverifier package")
	}
	if !policy.allows(AuthorityLayerGate, "github.com/s1onique/leamas/internal/factory/gate") {
		t.Errorf("production policy missing gate package")
	}
}
