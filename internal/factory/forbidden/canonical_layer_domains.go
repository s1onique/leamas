// SPDX-License-Identifier: Apache-2.0

package forbidden

import "sort"

// authorityLayerDomains is an invocation-local authority layer policy. It
// stores a set of allowed package paths per layer. Construction deep-copies
// the input slices and deduplicates + sorts each layer; no caller-owned
// backing array is retained.
type authorityLayerDomains struct {
	raw     []string
	adapter []string
	gate    []string
}

// newAuthorityLayerDomains constructs a policy from caller-supplied slices
// without retaining their backing arrays. Each layer is normalized to a
// sorted, deduplicated slice.
func newAuthorityLayerDomains(raw, adapter, gate []string) authorityLayerDomains {
	return authorityLayerDomains{
		raw:     normalizeLayerPackages(raw),
		adapter: normalizeLayerPackages(adapter),
		gate:    normalizeLayerPackages(gate),
	}
}

// normalizeLayerPackages copies the input, deduplicates entries, and
// returns the sorted, immutable-by-convention slice.
func normalizeLayerPackages(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, path := range in {
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// allows reports whether the package is permitted for the given layer. An
// unknown or empty layer returns false.
func (d authorityLayerDomains) allows(layer AuthorityLayer, packagePath string) bool {
	if packagePath == "" {
		return false
	}
	switch layer {
	case AuthorityLayerRaw:
		return containsString(d.raw, packagePath)
	case AuthorityLayerAdapter:
		return containsString(d.adapter, packagePath)
	case AuthorityLayerGate:
		return containsString(d.gate, packagePath)
	default:
		return false
	}
}

// containsString reports whether a sorted unique slice contains the value.
func containsString(haystack []string, needle string) bool {
	index := sort.SearchStrings(haystack, needle)
	return index < len(haystack) && haystack[index] == needle
}

// isEmpty reports whether the policy is entirely empty. A policy that has
// at least one allowed package in any layer is non-empty.
func (d authorityLayerDomains) isEmpty() bool {
	return len(d.raw) == 0 && len(d.adapter) == 0 && len(d.gate) == 0
}

// productionAuthorityLayerDomains returns the real production authority
// layer policy. Both production and fixture analyses start from this; tests
// derive a separate compatibility policy from each fixture.
func productionAuthorityLayerDomains() authorityLayerDomains {
	return newAuthorityLayerDomains(
		[]string{"github.com/s1onique/leamas/internal/factory/dupcode"},
		[]string{"github.com/s1onique/leamas/internal/factory/protectedverifier"},
		[]string{"github.com/s1onique/leamas/internal/factory/gate"},
	)
}

// fixtureAuthorityLayerDomains derives a compatibility layer policy from
// the protected-symbol inventory of a fixture. It permits every package
// path of the symbol under the symbol's declared layer, allowing the
// existing test fixtures that place symbols of multiple conceptual layers
// in the same synthetic package to remain in scope. Tests that need
// strict cross-layer rejection must use a custom policy via
// fixture.runWithLayerDomains.
func fixtureAuthorityLayerDomains(symbols []ProtectedSymbol) authorityLayerDomains {
	var raw, adapter, gate []string
	for _, symbol := range symbols {
		switch symbol.Layer {
		case AuthorityLayerRaw:
			raw = append(raw, symbol.PackagePath)
		case AuthorityLayerAdapter:
			adapter = append(adapter, symbol.PackagePath)
		case AuthorityLayerGate:
			gate = append(gate, symbol.PackagePath)
		}
	}
	return newAuthorityLayerDomains(raw, adapter, gate)
}
