// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
)

func verifierRunPointer(run func(string) []checks.Finding) uintptr {
	if run == nil {
		return 0
	}
	return reflect.ValueOf(run).Pointer()
}

func verifierWithoutRun(verifier registry.Verifier) registry.Verifier {
	verifier.Run = nil
	return verifier
}

func TestFactorizeMetadataPreservesCanonicalRegistry(t *testing.T) {
	counters := &factorizeDupcodeCounters{}
	actual := factorizeLifecycleRegistry(t, counters)
	canonical := AllVerifiers()
	if len(actual) != len(canonical) {
		t.Fatalf("factorize registry length = %d, canonical = %d", len(actual), len(canonical))
	}

	for i := range canonical {
		if !reflect.DeepEqual(verifierWithoutRun(actual[i]), verifierWithoutRun(canonical[i])) {
			t.Errorf("metadata at index %d differs:\nactual=%#v\ncanonical=%#v", i, actual[i], canonical[i])
		}
		actualRun := verifierRunPointer(actual[i].Run)
		canonicalRun := verifierRunPointer(canonical[i].Run)
		switch canonical[i].Name {
		case "dupcode", "dupcode-baseline":
			if actualRun == 0 || actualRun == canonicalRun {
				t.Errorf("%s Run was not replaced", canonical[i].Name)
			}
		default:
			if actualRun != canonicalRun {
				t.Errorf("%s Run changed unexpectedly", canonical[i].Name)
			}
		}
	}
	counters.assert(t, factorizeDupcodeTotals{})
}

func TestFactorizeMetadataMissingVerifierFailsClosed(t *testing.T) {
	canonical := AllVerifiers()
	filtered := make([]registry.Verifier, 0, len(canonical)-1)
	for _, verifier := range canonical {
		if verifier.Name != "dupcode" {
			filtered = append(filtered, verifier)
		}
	}
	original := append([]registry.Verifier(nil), filtered...)
	out, err := replaceDupcodeVerifierRuns(filtered, func(string) []checks.Finding { return nil }, func(string) []checks.Finding { return nil })
	if err == nil {
		t.Fatal("missing dupcode verifier replacement unexpectedly succeeded")
	}
	if out != nil {
		t.Fatalf("missing replacement output = %#v, want nil", out)
	}
	for i := range filtered {
		if verifierRunPointer(filtered[i].Run) != verifierRunPointer(original[i].Run) {
			t.Fatalf("input verifier %d mutated after missing-entry failure", i)
		}
	}
}

func TestFactorizeMetadataDuplicateVerifierFailsClosed(t *testing.T) {
	canonical := AllVerifiers()
	duplicate := factorizeDupcodeVerifier(t, canonical, "dupcode")
	input := append(append([]registry.Verifier(nil), canonical...), duplicate)
	originalPointers := make([]uintptr, len(input))
	for i := range input {
		originalPointers[i] = verifierRunPointer(input[i].Run)
	}

	out, err := replaceDupcodeVerifierRuns(input, func(string) []checks.Finding { return nil }, func(string) []checks.Finding { return nil })
	if err == nil {
		t.Fatal("duplicate dupcode verifier replacement unexpectedly succeeded")
	}
	if out != nil {
		t.Fatalf("duplicate replacement output = %#v, want nil", out)
	}
	for i := range input {
		if verifierRunPointer(input[i].Run) != originalPointers[i] {
			t.Fatalf("input verifier %d mutated after duplicate-entry failure", i)
		}
	}
}

func TestFactorizeConstructionProtectedSetupOnlyInInitialize(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "factorize_dupcode.go", nil, 0)
	if err != nil {
		t.Fatalf("parse factorize_dupcode.go: %v", err)
	}
	protectedSetupCaller := map[string]string{
		"ReadBaselineThresholds":        "readFactorizeDupcodeThresholds",
		"NewAnalyzerFromAdapter":        "newFactorizeDupcodeAnalyzer",
		"ReadThresholds":                "initialize",
		"NewAnalyzer":                   "initialize",
		"NewProvider":                   "initialize",
		"NewDupcodeAnalysisContext":     "initialize",
		"NewDupcodeVerifierFactory":     "initialize",
		"SharedDupCodeVerifier":         "initialize",
		"SharedDupcodeBaselineVerifier": "initialize",
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			wantCaller, protected := protectedSetupCaller[selector.Sel.Name]
			if !protected {
				return true
			}
			if function.Name.Name != wantCaller {
				t.Errorf("protected setup call %s occurs in %s, want %s", selector.Sel.Name, function.Name.Name, wantCaller)
			}
			return true
		})
	}
}

func TestFactorizeIsolationHasNoPackageGlobalLifecycleState(t *testing.T) {
	path := "factorize_dupcode.go"
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			var rendered bytes.Buffer
			if err := format.Node(&rendered, fileSet, value); err != nil {
				t.Fatalf("render package variable: %v", err)
			}
			text := strings.ToLower(rendered.String())
			for _, forbidden := range []string{"lifecycle", "analysisprovider", "factorizedupcodecache"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("package-level factorize state is forbidden: %s", rendered.String())
				}
			}
		}
	}
}
