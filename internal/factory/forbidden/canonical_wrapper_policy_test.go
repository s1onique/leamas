// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"strings"
	"testing"
)

func TestCanonicalWrapperCapabilityRejectsLaundering(t *testing.T) {
	fixture := newCanonicalFixture(t)
	gatePkg := fixture.packagePath("gate")
	fixture.write("gate/wrappers.go", `package gate
func Wrapper() {}
func Production() { _ = Wrapper }
func EvilPreAuthoritySetup() { Wrapper() }
`)
	wrapper := fixtureSymbol(AuthorityLayerGate, gatePkg, "Wrapper", ProtectedPackageFunction, "")
	approval := fixtureApproval(
		gatePkg,
		"Production",
		"",
		CallerKindPackageFunction,
		wrapper,
		refFunctionValue,
	)
	result := fixture.run([]ProtectedSymbol{wrapper}, []ApprovedCaller{approval})
	if result.Stats.ValidatedApprovals != 1 {
		t.Fatalf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
	finding := requireFindingKind(t, result.Findings, "dupcode_gate_bypass")
	if !strings.Contains(finding.Message, "EvilPreAuthoritySetup") {
		t.Fatalf("laundering finding = %q, want evil caller identity", finding.Message)
	}
}

func TestCanonicalFactorizeWrappersAreProtectedCapabilities(t *testing.T) {
	wantSymbols := map[string]bool{
		"newFactorizeDupcodeAnalyzer":    false,
		"readFactorizeDupcodeThresholds": false,
	}
	for _, symbol := range GateProtectedSymbols {
		if _, ok := wantSymbols[symbol.Name]; ok {
			wantSymbols[symbol.Name] = true
			if symbol.Layer != AuthorityLayerGate || symbol.Kind != ProtectedPackageFunction {
				t.Errorf("wrapper symbol = %+v, want gate package function", symbol)
			}
		}
	}
	for name, found := range wantSymbols {
		if !found {
			t.Errorf("factorize wrapper %s is not protected", name)
		}
	}

	for name := range wantSymbols {
		found := false
		for _, approval := range GateApprovedCallers {
			if approval.Function == "productionFactorizeDupcodeDeps" &&
				approval.Callee.Name == name &&
				approval.ReferenceClass == refFunctionValue {
				found = true
			}
		}
		if !found {
			t.Errorf("missing sole production wrapper approval for %s", name)
		}
	}
}

func TestCanonicalProductionPolicyAndApprovalTruth(t *testing.T) {
	result := runCanonicalAnalysis(repoRoot(t), testModulePath, productionCanonicalConfig())
	for _, finding := range result.Findings {
		if strings.HasPrefix(finding.Kind, "authority_policy_") ||
			finding.Kind == "dupcode_bypass" ||
			finding.Kind == "dupcode_adapter_bypass" ||
			finding.Kind == "dupcode_gate_bypass" ||
			strings.Contains(finding.Kind, "protected_function_value") ||
			strings.Contains(finding.Kind, "adapter_function_value") {
			t.Errorf("production authority-policy finding: %#v", finding)
		}
	}

	wrapperEdges := 0
	for _, edge := range result.ObservedEdges {
		if edge.Callee.Layer != AuthorityLayerGate {
			continue
		}
		wrapperEdges++
		if edge.Caller.Function != "productionFactorizeDupcodeDeps" ||
			edge.Caller.Receiver != "" ||
			edge.ReferenceClass != refFunctionValue {
			t.Errorf("unexpected factorize wrapper edge: %#v", edge)
		}
	}
	if wrapperEdges != 2 {
		t.Fatalf("factorize wrapper edges = %d, want 2", wrapperEdges)
	}
	if result.Stats.ConfiguredProtectedSymbols != result.Stats.ResolvedProtectedObjects {
		t.Fatalf("protected resolution stats = %+v", result.Stats)
	}
	if result.Stats.ConfiguredApprovals != result.Stats.ValidatedApprovals {
		t.Fatalf("approval validation stats = %+v", result.Stats)
	}
	t.Logf("canonical stats=%+v normalized_finding_hash=%s", result.Stats, result.NormalizedFindingHash)
}
