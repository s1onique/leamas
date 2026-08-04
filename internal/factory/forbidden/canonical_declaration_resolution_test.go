// SPDX-License-Identifier: Apache-2.0

package forbidden

import "testing"

func declarationFixture(t *testing.T) (*canonicalFixture, string) {
	t.Helper()
	fixture := newCanonicalFixture(t)
	fixture.write("protected/protected.go", `package protected

func Cap() {}

type Runner struct{}
func (*Runner) Run() {}

var Capability = func() {}

func localScope() {
	var LocalCapability = func() {}
	_ = LocalCapability
}

type Left struct{}
type Right struct{}
func (*Left) Clash() {}
func (*Right) Clash() {}
`)
	return fixture, fixture.packagePath("protected")
}

func TestCanonicalProtectedDeclarationsResolveGlobally(t *testing.T) {
	fixture, protectedPkg := declarationFixture(t)
	symbols := []ProtectedSymbol{
		fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, ""),
		fixtureSymbol(AuthorityLayerAdapter, protectedPkg, "Run", ProtectedMethod, "Runner"),
		fixtureSymbol(AuthorityLayerAdapter, protectedPkg, "Capability", ProtectedPackageVariable, ""),
	}
	result := fixture.run(symbols, nil)
	for _, kind := range []string{
		"authority_policy_symbol_missing",
		"authority_policy_symbol_ambiguous",
		"authority_policy_kind_mismatch",
		"authority_policy_receiver_mismatch",
		"authority_policy_scope_mismatch",
		"authority_policy_duplicate_symbol",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	if result.Stats.ConfiguredProtectedSymbols != 3 {
		t.Fatalf("configured symbols = %d, want 3", result.Stats.ConfiguredProtectedSymbols)
	}
	if result.Stats.ResolvedProtectedObjects != 3 {
		t.Fatalf("resolved objects = %d, want 3", result.Stats.ResolvedProtectedObjects)
	}
}

func TestCanonicalProtectedDeclarationFailures(t *testing.T) {
	cases := []struct {
		name     string
		symbols  func(string) []ProtectedSymbol
		wantKind string
	}{
		{
			name: "missing symbol",
			symbols: func(pkg string) []ProtectedSymbol {
				return []ProtectedSymbol{fixtureSymbol(AuthorityLayerRaw, pkg, "Missing", ProtectedPackageFunction, "")}
			},
			wantKind: "authority_policy_symbol_missing",
		},
		{
			name: "wrong kind",
			symbols: func(pkg string) []ProtectedSymbol {
				return []ProtectedSymbol{fixtureSymbol(AuthorityLayerRaw, pkg, "Cap", ProtectedMethod, "Runner")}
			},
			wantKind: "authority_policy_kind_mismatch",
		},
		{
			name: "wrong receiver",
			symbols: func(pkg string) []ProtectedSymbol {
				return []ProtectedSymbol{fixtureSymbol(AuthorityLayerAdapter, pkg, "Run", ProtectedMethod, "WrongRunner")}
			},
			wantKind: "authority_policy_receiver_mismatch",
		},
		{
			name: "duplicate configuration",
			symbols: func(pkg string) []ProtectedSymbol {
				symbol := fixtureSymbol(AuthorityLayerRaw, pkg, "Cap", ProtectedPackageFunction, "")
				return []ProtectedSymbol{symbol, symbol}
			},
			wantKind: "authority_policy_duplicate_symbol",
		},
		{
			name: "non-package variable scope",
			symbols: func(pkg string) []ProtectedSymbol {
				return []ProtectedSymbol{fixtureSymbol(AuthorityLayerAdapter, pkg, "LocalCapability", ProtectedPackageVariable, "")}
			},
			wantKind: "authority_policy_scope_mismatch",
		},
		{
			name: "ambiguous method without receiver",
			symbols: func(pkg string) []ProtectedSymbol {
				return []ProtectedSymbol{fixtureSymbol(AuthorityLayerAdapter, pkg, "Clash", ProtectedMethod, "")}
			},
			wantKind: "authority_policy_symbol_ambiguous",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture, protectedPkg := declarationFixture(t)
			result := fixture.run(tc.symbols(protectedPkg), nil)
			requireFindingKind(t, result.Findings, tc.wantKind)
		})
	}
}

func TestCanonicalMalformedTypeInformationFailsClosed(t *testing.T) {
	fixture := newCanonicalFixture(t)
	fixture.write("broken/broken.go", "package broken\nfunc Broken( {\n")
	result := fixture.run(nil, nil)
	if len(result.Findings) == 0 {
		t.Fatal("malformed package produced no findings")
	}
	found := false
	for _, kind := range []string{
		"dupcode_package_metadata_error",
		"dupcode_type_error",
		"dupcode_type_info_error",
	} {
		for _, finding := range result.Findings {
			if finding.Kind == kind {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("malformed package findings = %v, want package/type failure", findingKinds(result.Findings))
	}
}
