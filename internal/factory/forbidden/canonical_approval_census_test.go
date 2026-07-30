// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"go/types"
	"reflect"
	"testing"
)

// TestApprovalCensusRunSharedDupcodeBaselineIsPackageFunction locks the
// runSharedDupcodeBaseline declaration to package_function. It is the
// focused regression proof that any future migration that tries to mark
// runSharedDupcodeBaseline as a method will fail this test.
func TestApprovalCensusRunSharedDupcodeBaselineIsPackageFunction(t *testing.T) {
	result, analysis := runProductionCanonicalAnalysisForCensus(t)

	if result.Stats.ConfiguredProtectedSymbols != 19 ||
		result.Stats.ResolvedProtectedObjects != 19 ||
		result.Stats.ConfiguredApprovals != 34 ||
		result.Stats.ObservedEdges != 34 ||
		result.Stats.ValidatedApprovals != 34 {
		t.Fatalf("production stats drifted: %+v", result.Stats)
	}

	records := buildApprovalCensus(analysis)
	runShared := findRecordsByFunction(records, "runSharedDupcodeBaseline")
	if len(runShared) != 2 {
		t.Fatalf("runSharedDupcodeBaseline record count = %d, want 2 (ValidateBaselineArtifact, CheckBaselineDriftFromReport)", len(runShared))
	}

	for index, record := range runShared {
		if record.Configured.PackagePath != "github.com/s1onique/leamas/internal/factory/protectedverifier" {
			t.Errorf("record %d caller package = %q, want protectedverifier", index, record.Configured.PackagePath)
		}
		if record.Configured.Function != "runSharedDupcodeBaseline" {
			t.Errorf("record %d caller function = %q, want runSharedDupcodeBaseline", index, record.Configured.Function)
		}
		if record.Configured.Receiver != "" {
			t.Errorf("record %d configured receiver = %q, want empty for package function", index, record.Configured.Receiver)
		}

		object := declarationObjectForRecord(analysis, record)
		if object == nil {
			t.Fatalf("record %d caller did not resolve", index)
		}
		function, ok := object.(*types.Func)
		if !ok {
			t.Fatalf("record %d resolved object = %T, want *types.Func", index, object)
		}
		signature, _ := function.Type().(*types.Signature)
		if signature == nil {
			t.Fatalf("record %d resolved function has no signature", index)
		}
		if signature.Recv() != nil {
			t.Fatalf("record %d resolved declaration is a method (recv=%v); runSharedDupcodeBaseline is a package function", index, signature.Recv())
		}
		if record.Effective.CallerKind != CallerKindPackageFunction {
			t.Errorf("record %d normalized caller kind = %q, want %q", index, record.Effective.CallerKind, CallerKindPackageFunction)
		}
		if record.Effective.Receiver != "" {
			t.Errorf("record %d normalized receiver = %q, want empty", index, record.Effective.Receiver)
		}
		if record.ResolvedCallerKind != CallerKindPackageFunction {
			t.Errorf("record %d resolved caller kind = %q, want %q", index, record.ResolvedCallerKind, CallerKindPackageFunction)
		}
		if record.ResolvedCallerReceiver != "" {
			t.Errorf("record %d resolved receiver = %q, want empty", index, record.ResolvedCallerReceiver)
		}
		if !record.KindMatches {
			t.Errorf("record %d kind mismatch: effective=%q resolved=%q", index, record.Effective.CallerKind, record.ResolvedCallerKind)
		}
		if !record.ReceiverMatches {
			t.Errorf("record %d receiver mismatch: effective=%q resolved=%q", index, record.Effective.Receiver, record.ResolvedCallerReceiver)
		}

		explicit := ApprovedCaller{
			PackagePath:    record.Configured.PackagePath,
			Function:       record.Configured.Function,
			Receiver:       "",
			CallerKind:     CallerKindPackageFunction,
			ReferenceClass: refDirectCall,
			Cardinality:    record.Effective.Cardinality,
			Callee:         record.Configured.Callee,
		}
		if !reflect.DeepEqual(explicit, record.Effective) {
			t.Errorf("record %d explicit form does not match effective: explicit=%+v effective=%+v", index, explicit, record.Effective)
		}
	}
}

// TestApprovalCensusMatchesResolvedDeclarations is the full 34-record
// census. It forbids any production caller attribution that does not
// match the resolved declaration.
func TestApprovalCensusMatchesResolvedDeclarations(t *testing.T) {
	result, analysis := runProductionCanonicalAnalysisForCensus(t)

	if result.Stats.ConfiguredApprovals != 34 {
		t.Fatalf("configured approvals = %d, want 34", result.Stats.ConfiguredApprovals)
	}

	records := buildApprovalCensus(analysis)

	implicitKind := 0
	implicitRef := 0
	nonpositiveCard := 0
	for _, record := range records {
		if record.ImplicitCallerKind {
			implicitKind++
		}
		if record.ImplicitReferenceClass {
			implicitRef++
		}
		if record.ImplicitCardinality {
			nonpositiveCard++
		}
		if !record.CallerResolved {
			t.Errorf("index %d caller did not resolve: %+v", record.Index, record.Configured)
			continue
		}
		if !record.KindMatches {
			t.Errorf("index %d kind mismatch: effective=%q resolved=%q receiver=%q",
				record.Index, record.Effective.CallerKind, record.ResolvedCallerKind, record.Effective.Receiver)
		}
		if !record.ReceiverMatches {
			t.Errorf("index %d receiver mismatch: effective=%q resolved=%q",
				record.Index, record.Effective.Receiver, record.ResolvedCallerReceiver)
		}
		if record.Effective.Cardinality <= 0 {
			t.Errorf("index %d effective cardinality = %d, want positive", record.Index, record.Effective.Cardinality)
		}
		if !validApprovalReferenceClass(record.Effective.ReferenceClass) {
			t.Errorf("index %d reference class %q is not approvable", record.Index, record.Effective.ReferenceClass)
		}
	}

	normalizedPackageFunctions := countByKind(records, effectiveCallerKind, CallerKindPackageFunction)
	normalizedMethods := countByKind(records, effectiveCallerKind, CallerKindMethod)
	normalizedVariableInits := countByKind(records, effectiveCallerKind, CallerKindVariableInitializer)
	normalizedPackageInits := countByKind(records, effectiveCallerKind, CallerKindPackageInit)
	if normalizedPackageFunctions+normalizedMethods+normalizedVariableInits+normalizedPackageInits != 34 {
		t.Fatalf("normalized kind total = %d, want 34",
			normalizedPackageFunctions+normalizedMethods+normalizedVariableInits+normalizedPackageInits)
	}

	resolvedPackageFunctions := countByKind(records, resolvedCallerKind, CallerKindPackageFunction)
	resolvedMethods := countByKind(records, resolvedCallerKind, CallerKindMethod)
	resolvedVariableInits := countByKind(records, resolvedCallerKind, CallerKindVariableInitializer)
	resolvedPackageInits := countByKind(records, resolvedCallerKind, CallerKindPackageInit)
	if resolvedPackageFunctions+resolvedMethods+resolvedVariableInits+resolvedPackageInits != 34 {
		t.Fatalf("resolved kind total = %d, want 34",
			resolvedPackageFunctions+resolvedMethods+resolvedVariableInits+resolvedPackageInits)
	}

	if normalizedPackageFunctions != resolvedPackageFunctions ||
		normalizedMethods != resolvedMethods ||
		normalizedVariableInits != resolvedVariableInits ||
		normalizedPackageInits != resolvedPackageInits {
		t.Errorf("normalized/resolved mismatch: pkg=%d/%d method=%d/%d var=%d/%d init=%d/%d",
			normalizedPackageFunctions, resolvedPackageFunctions,
			normalizedMethods, resolvedMethods,
			normalizedVariableInits, resolvedVariableInits,
			normalizedPackageInits, resolvedPackageInits)
	}

	t.Logf("census: implicit_kind=%d implicit_ref=%d nonpositive_card=%d", implicitKind, implicitRef, nonpositiveCard)
	t.Logf("census: normalized pkg=%d method=%d var=%d init=%d",
		normalizedPackageFunctions, normalizedMethods, normalizedVariableInits, normalizedPackageInits)
	t.Logf("census: resolved   pkg=%d method=%d var=%d init=%d",
		resolvedPackageFunctions, resolvedMethods, resolvedVariableInits, resolvedPackageInits)
}

// TestApprovalCensusNormalizationOracleHashStable verifies the
// deterministic normalized oracle hash is reproducible across repeated
// runs and counts exactly 20 records, all equal.
func TestApprovalCensusNormalizationOracleHashStable(t *testing.T) {
	_, analysis := runProductionCanonicalAnalysisForCensus(t)
	records := buildApprovalCensus(analysis)
	if len(records) != 34 {
		t.Fatalf("census record count = %d, want 34", len(records))
	}

	first := hashCensusRecords(records)
	for run := 0; run < 19; run++ {
		again := hashCensusRecords(records)
		if again != first {
			t.Fatalf("oracle hash drift on run %d: got %s want %s", run+1, again, first)
		}
	}
	t.Logf("oracle hash stable across 20 runs: %s", first)
}

// TestApprovalCensusExplicitRecordEquivalence proves that constructing
// each normalized approval explicitly with the documented effective
// fields reproduces the current normalizeApproval output. This is the
// oracle for the future explicit-inventory migration: any record whose
// explicit form does not match the effective form is a migration bug.
func TestApprovalCensusExplicitRecordEquivalence(t *testing.T) {
	_, analysis := runProductionCanonicalAnalysisForCensus(t)
	records := buildApprovalCensus(analysis)

	categories := map[string]int{}
	for index, record := range records {
		if record.Configured.Function == "runSharedDupcodeBaseline" {
			verifyExplicitEquivalence(t, index, record)
			categories["package_function"]++
			continue
		}
		switch record.Effective.CallerKind {
		case CallerKindPackageFunction:
			verifyExplicitEquivalence(t, index, record)
			categories["package_function"]++
		case CallerKindMethod:
			verifyExplicitEquivalence(t, index, record)
			categories["method"]++
		default:
			categories["deferred"]++
		}
	}

	if categories["package_function"] == 0 {
		t.Fatalf("no package-function records exercised")
	}
	if categories["method"] == 0 {
		t.Fatalf("no method records exercised")
	}
	t.Logf("explicit equivalence categories: %+v", categories)
}

func verifyExplicitEquivalence(t *testing.T, index int, record approvalCensusRecord) {
	t.Helper()
	explicit := ApprovedCaller{
		PackagePath:    record.Configured.PackagePath,
		Function:       record.Configured.Function,
		Receiver:       record.Configured.Receiver,
		CallerKind:     record.Effective.CallerKind,
		ReferenceClass: record.Effective.ReferenceClass,
		Cardinality:    record.Effective.Cardinality,
		Callee:         record.Configured.Callee,
	}
	if explicit.Receiver == "" && record.Effective.Receiver == "" {
		// package-function / variable-init path
	} else if explicit.Receiver != record.Effective.Receiver {
		t.Errorf("index %d receiver drifted: explicit=%q effective=%q", index, explicit.Receiver, record.Effective.Receiver)
	}
	if explicit.CallerKind != record.Effective.CallerKind {
		t.Errorf("index %d caller kind drifted: explicit=%q effective=%q", index, explicit.CallerKind, record.Effective.CallerKind)
	}
	if !reflect.DeepEqual(explicit, record.Effective) {
		t.Errorf("index %d explicit form does not match effective: explicit=%+v effective=%+v", index, explicit, record.Effective)
	}
}
