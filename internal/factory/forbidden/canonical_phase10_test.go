// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"strconv"
	"strings"
	"testing"
)

// TestImportLiteralDecoder_DoubleQuotedDecodes proves strconv.Unquote handles
// double-quoted import path literals correctly per the Go spec.
func TestImportLiteralDecoder_DoubleQuotedDecodes(t *testing.T) {
	got, err := strconv.Unquote(`"github.com/foo/bar"`)
	if err != nil {
		t.Fatalf("Unquote double-quoted: %v", err)
	}
	if got != "github.com/foo/bar" {
		t.Errorf("Unquote = %q, want %q", got, "github.com/foo/bar")
	}
}

// TestImportLiteralDecoder_BackquotedDecodes proves strconv.Unquote handles
// backquoted (raw-string) import path literals per the Go spec.
func TestImportLiteralDecoder_BackquotedDecodes(t *testing.T) {
	got, err := strconv.Unquote("`github.com/foo/bar`")
	if err != nil {
		t.Fatalf("Unquote backquoted: %v", err)
	}
	if got != "github.com/foo/bar" {
		t.Errorf("Unquote = %q, want %q", got, "github.com/foo/bar")
	}
}

// TestImportLiteralDecoder_EscapedValidImportDecodedCorrectly proves that
// well-formed import literals with escape sequences are decoded correctly.
func TestImportLiteralDecoder_EscapedValidImportDecodedCorrectly(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`"github.com/foo/bar"`, "github.com/foo/bar"},
		{"`github.com/foo/bar`", "github.com/foo/bar"},
		{`"github.com/foo\u002fbar"`, "github.com/foo/bar"},
		{`"foo\u005cbar"`, `foo\bar`},
	}
	for _, c := range cases {
		got, err := strconv.Unquote(c.in)
		if err != nil {
			t.Errorf("Unquote(%q) err=%v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Unquote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestImportLiteralDecoder_UnterminatedLiteralFailsClosed proves that an
// unparseable import literal returns an error (the production policy uses
// this as the failure signal).
func TestImportLiteralDecoder_UnterminatedLiteralFailsClosed(t *testing.T) {
	// Unterminated double-quoted literal is invalid.
	if _, err := strconv.Unquote(`"unterminated`); err == nil {
		t.Error("expected Unquote to fail on unterminated literal")
	}
	// Unterminated backquoted literal is invalid.
	if _, err := strconv.Unquote("`unterminated"); err == nil {
		t.Error("expected Unquote to fail on unterminated backquoted literal")
	}
	// Mismatched quote (double-quote opened, backquote closer).
	if _, err := strconv.Unquote(`"mismatch` + "`"); err == nil {
		t.Error("expected Unquote to fail on mismatched quotes")
	}
}

// TestProtectedPackageDetection_MatchesConfiguredPrefixes proves the
// isProtectedPackage helper matches every DupcodeProtectedPrefix.
func TestProtectedPackageDetection_MatchesConfiguredPrefixes(t *testing.T) {
	for _, prefix := range DupcodeProtectedPrefixes {
		if !isProtectedPackage(prefix) {
			t.Errorf("isProtectedPackage(%q) = false, want true", prefix)
		}
		if !isProtectedPackage(prefix + "/sub") {
			t.Errorf("isProtectedPackage(%q/sub) = false, want true", prefix)
		}
	}
}

// TestProtectedPackageDetection_NonProtectedReturnsFalse proves isProtectedPackage
// does not match unrelated packages.
func TestProtectedPackageDetection_NonProtectedReturnsFalse(t *testing.T) {
	if isProtectedPackage("github.com/s1onique/leamas/internal/factory/llmfriendly") {
		t.Error("isProtectedPackage(llmfriendly) = true, want false")
	}
	if isProtectedPackage("fmt") {
		t.Error("isProtectedPackage(fmt) = true, want false")
	}
}

// TestAdapterProtectedPackageDetection proves the adapter package is recognized.
func TestAdapterProtectedPackageDetection(t *testing.T) {
	for _, prefix := range AdapterProtectedPrefixes {
		if !isAdapterProtectedPackage(prefix) {
			t.Errorf("isAdapterProtectedPackage(%q) = false, want true", prefix)
		}
	}
}

// TestProtectedKindValidation_AdapterMethodKindRejected proves that the
// configured adapter symbol LoadBaseline is a method (not package function).
func TestProtectedKindValidation_AdapterMethodKindRejected(t *testing.T) {
	var found bool
	for _, sym := range AdapterProtectedSymbols {
		if sym.Name == "LoadBaseline" {
			found = true
			if sym.Kind != ProtectedMethod {
				t.Errorf("LoadBaseline.Kind = %q, want %q", sym.Kind, ProtectedMethod)
			}
			if sym.Receiver != "DupcodeRunner" {
				t.Errorf("LoadBaseline.Receiver = %q, want DupcodeRunner", sym.Receiver)
			}
		}
	}
	if !found {
		t.Error("LoadBaseline missing from AdapterProtectedSymbols")
	}
}

// TestAdapterApprovedCallers_ContainsObservedEdges proves that the configured
// inventory contains real source edges rather than dynamic-interface fiction.
func TestAdapterApprovedCallers_ContainsObservedEdges(t *testing.T) {
	expected := []ApprovedCaller{
		{
			PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
			Function:    "newProtectedDupcodeRunner",
			Callee: ProtectedSymbol{
				Layer:       AuthorityLayerAdapter,
				PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
				Name:        "NewDupcodeRunner",
				Kind:        ProtectedPackageFunction,
			},
		},
		{
			PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
			Function:    "dupCodeVerifier",
			Receiver:    "",
			Callee: ProtectedSymbol{
				Layer:       AuthorityLayerAdapter,
				PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
				Name:        "NewDupcodeRunner",
				Kind:        ProtectedPackageFunction,
			},
		},
		{
			PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
			Function:    "VerifyBaseline",
			Receiver:    "protectedDupcodeRunnerAdapter",
			Callee: ProtectedSymbol{
				Layer:       AuthorityLayerAdapter,
				PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
				Name:        "VerifyBaseline",
				Kind:        ProtectedMethod,
				Receiver:    "DupcodeRunner",
			},
		},
	}
	for _, want := range expected {
		found := false
		for _, got := range AdapterApprovedCallers {
			if got.PackagePath == want.PackagePath &&
				got.Function == want.Function &&
				got.Receiver == want.Receiver &&
				got.Callee.PackagePath == want.Callee.PackagePath &&
				got.Callee.Name == want.Callee.Name &&
				got.Callee.Layer == want.Callee.Layer &&
				got.Callee.Kind == want.Callee.Kind &&
				got.Callee.Receiver == want.Callee.Receiver {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing approval edge: %+v", want)
		}
	}
}

// TestAdapterApprovedCallers_NoWildcardApprovers proves the AdapterApprovedCallers
// list does NOT contain any forbidden patterns.
func TestAdapterApprovedCallers_NoWildcardApprovers(t *testing.T) {
	for _, ac := range AdapterApprovedCallers {
		if ac.Function == "" {
			t.Errorf("empty Function in approval: %+v", ac)
		}
		if strings.Contains(ac.Function, "*") {
			t.Errorf("wildcard in Function: %+v", ac)
		}
		if strings.Contains(ac.Function, "@") {
			t.Errorf("anonymous literal in Function: %+v", ac)
		}
	}
}

// TestApprovedCallers_NoWildcardApprovers proves the raw-layer ApprovedCallers
// list does NOT contain any forbidden patterns.
func TestApprovedCallers_NoWildcardApprovers(t *testing.T) {
	for _, ac := range ApprovedCallers {
		if ac.Function == "" {
			t.Errorf("empty Function in approval: %+v", ac)
		}
		if strings.Contains(ac.Function, "*") {
			t.Errorf("wildcard in Function: %+v", ac)
		}
		if strings.Contains(ac.Function, "@") {
			t.Errorf("anonymous literal in Function: %+v", ac)
		}
	}
}

// TestReferenceClassConstants_AllPresent verifies all referenceClass values
// are defined and unique.
func TestReferenceClassConstants_AllPresent(t *testing.T) {
	classes := map[referenceClass]bool{
		refDirectCall:       true,
		refFunctionValue:    true,
		refMethodValue:      true,
		refMethodExpression: true,
		refPackageVariable:  true,
		refDeclaration:      true,
	}
	if len(classes) != 6 {
		t.Errorf("expected 6 referenceClass constants, got %d", len(classes))
	}
}

// TestProtectedSymbolKindConstants verifies all kind constants are defined.
func TestProtectedSymbolKindConstants(t *testing.T) {
	kinds := map[ProtectedSymbolKind]bool{
		ProtectedPackageFunction: true,
		ProtectedMethod:          true,
		ProtectedPackageVariable: true,
	}
	if len(kinds) != 3 {
		t.Errorf("expected 3 kind constants, got %d", len(kinds))
	}
}

// TestAuthorityLayerConstants verifies both authority layers are defined.
func TestAuthorityLayerConstants(t *testing.T) {
	if AuthorityLayerRaw == "" {
		t.Error("AuthorityLayerRaw must not be empty")
	}
	if AuthorityLayerAdapter == "" {
		t.Error("AuthorityLayerAdapter must not be empty")
	}
	if AuthorityLayerRaw == AuthorityLayerAdapter {
		t.Error("AuthorityLayerRaw and AuthorityLayerAdapter must differ")
	}
}

// TestIsApprovedCaller_RejectsWrongKind verifies IsApprovedCaller rejects a
// caller/callee pair whose declared kind does not match the configured kind.
func TestIsApprovedCaller_RejectsWrongKind(t *testing.T) {
	caller := CallerIdentity{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupCodeVerifier",
		Receiver:    "",
		Kind:        "package_function",
	}
	wrong := ProtectedSymbol{
		Layer:       AuthorityLayerAdapter,
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Name:        "NewDupcodeRunner",
		Kind:        ProtectedMethod,
	}
	if IsApprovedCaller(caller, wrong) {
		t.Error("IsApprovedCaller should reject mismatched kind")
	}
	right := ProtectedSymbol{
		Layer:       AuthorityLayerAdapter,
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Name:        "NewDupcodeRunner",
		Kind:        ProtectedPackageFunction,
	}
	if !IsApprovedCaller(caller, right) {
		t.Error("IsApprovedCaller should accept matching kind")
	}
}

// TestIsApprovedCaller_RejectsWrongReceiver verifies receiver mismatch is rejected.
func TestIsApprovedCaller_RejectsWrongReceiver(t *testing.T) {
	caller := CallerIdentity{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "LoadBaseline",
		Receiver:    "protectedDupcodeRunnerAdapter",
	}
	wrong := ProtectedSymbol{
		Layer:       AuthorityLayerAdapter,
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Name:        "LoadBaseline",
		Kind:        ProtectedMethod,
		Receiver:    "WrongRunner",
	}
	if IsApprovedCaller(caller, wrong) {
		t.Error("IsApprovedCaller should reject wrong receiver")
	}
	right := ProtectedSymbol{
		Layer:       AuthorityLayerAdapter,
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Name:        "LoadBaseline",
		Kind:        ProtectedMethod,
		Receiver:    "DupcodeRunner",
	}
	if !IsApprovedCaller(caller, right) {
		t.Error("IsApprovedCaller should accept right receiver")
	}
}

// TestIsApprovedCaller_RejectsWrongPackage verifies package mismatch is rejected.
func TestIsApprovedCaller_RejectsWrongPackage(t *testing.T) {
	caller := CallerIdentity{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupCodeVerifier",
	}
	sym := ProtectedSymbol{
		Layer:       AuthorityLayerAdapter,
		PackagePath: "github.com/some/other/package",
		Name:        "NewDupcodeRunner",
		Kind:        ProtectedPackageFunction,
	}
	if IsApprovedCaller(caller, sym) {
		t.Error("IsApprovedCaller should reject wrong package")
	}
}

// TestIsApprovedCaller_RejectsEmptyFunction verifies empty Function is rejected.
func TestIsApprovedCaller_RejectsEmptyFunction(t *testing.T) {
	caller := CallerIdentity{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "",
	}
	sym := ProtectedSymbol{
		Layer:       AuthorityLayerAdapter,
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Name:        "NewDupcodeRunner",
		Kind:        ProtectedPackageFunction,
	}
	if IsApprovedCaller(caller, sym) {
		t.Error("IsApprovedCaller should reject empty Function")
	}
}

// TestRecvTypeNameFromSig_NilReceiverReturnsEmpty verifies that the receiver
// type name helper returns empty for a nil receiver (non-method function).
func TestRecvTypeNameFromSig_NilReceiverReturnsEmpty(t *testing.T) {
	if got := recvTypeNameFromSig(nil); got != "" {
		t.Errorf("recvTypeNameFromSig(nil) = %q, want \"\"", got)
	}
}
