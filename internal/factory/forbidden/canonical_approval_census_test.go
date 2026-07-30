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
		if record.Configured.CallerKind != CallerKindPackageFunction {
			t.Errorf("record %d configured caller kind = %q, want %q", index, record.Configured.CallerKind, CallerKindPackageFunction)
		}
		if record.Configured.Receiver != "" {
			t.Errorf("record %d configured receiver = %q, want empty", index, record.Configured.Receiver)
		}
		if record.ResolvedCallerKind != CallerKindPackageFunction {
			t.Errorf("record %d resolved caller kind = %q, want %q", index, record.ResolvedCallerKind, CallerKindPackageFunction)
		}
		if record.ResolvedCallerReceiver != "" {
			t.Errorf("record %d resolved receiver = %q, want empty", index, record.ResolvedCallerReceiver)
		}
		if !record.KindMatches {
			t.Errorf("record %d kind mismatch: configured=%q resolved=%q", index, record.Configured.CallerKind, record.ResolvedCallerKind)
		}
		if !record.ReceiverMatches {
			t.Errorf("record %d receiver mismatch: configured=%q resolved=%q", index, record.Configured.Receiver, record.ResolvedCallerReceiver)
		}

		explicit := ApprovedCaller{
			PackagePath:    record.Configured.PackagePath,
			Function:       record.Configured.Function,
			Receiver:       "",
			CallerKind:     CallerKindPackageFunction,
			ReferenceClass: refDirectCall,
			Cardinality:    record.Configured.Cardinality,
			Callee:         record.Configured.Callee,
		}
		if !reflect.DeepEqual(explicit, record.Configured) {
			t.Errorf("record %d explicit form does not match configured: explicit=%+v configured=%+v", index, explicit, record.Configured)
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

	for _, record := range records {
		if !record.CallerResolved {
			t.Errorf("index %d caller did not resolve: %+v", record.Index, record.Configured)
			continue
		}
		if !record.KindMatches {
			t.Errorf("index %d kind mismatch: configured=%q resolved=%q receiver=%q",
				record.Index, record.Configured.CallerKind, record.ResolvedCallerKind, record.Configured.Receiver)
		}
		if !record.ReceiverMatches {
			t.Errorf("index %d receiver mismatch: configured=%q resolved=%q",
				record.Index, record.Configured.Receiver, record.ResolvedCallerReceiver)
		}
		if record.Configured.Cardinality <= 0 {
			t.Errorf("index %d configured cardinality = %d, want positive", record.Index, record.Configured.Cardinality)
		}
		if !validApprovalReferenceClass(record.Configured.ReferenceClass) {
			t.Errorf("index %d reference class %q is not approvable", record.Index, record.Configured.ReferenceClass)
		}
	}

	configuredPackageFunctions := countByKind(records, configuredCallerKind, CallerKindPackageFunction)
	configuredMethods := countByKind(records, configuredCallerKind, CallerKindMethod)
	configuredVariableInits := countByKind(records, configuredCallerKind, CallerKindVariableInitializer)
	configuredPackageInits := countByKind(records, configuredCallerKind, CallerKindPackageInit)
	if configuredPackageFunctions+configuredMethods+configuredVariableInits+configuredPackageInits != 34 {
		t.Fatalf("configured kind total = %d, want 34",
			configuredPackageFunctions+configuredMethods+configuredVariableInits+configuredPackageInits)
	}

	resolvedPackageFunctions := countByKind(records, resolvedCallerKind, CallerKindPackageFunction)
	resolvedMethods := countByKind(records, resolvedCallerKind, CallerKindMethod)
	resolvedVariableInits := countByKind(records, resolvedCallerKind, CallerKindVariableInitializer)
	resolvedPackageInits := countByKind(records, resolvedCallerKind, CallerKindPackageInit)
	if resolvedPackageFunctions+resolvedMethods+resolvedVariableInits+resolvedPackageInits != 34 {
		t.Fatalf("resolved kind total = %d, want 34",
			resolvedPackageFunctions+resolvedMethods+resolvedVariableInits+resolvedPackageInits)
	}

	if configuredPackageFunctions != resolvedPackageFunctions ||
		configuredMethods != resolvedMethods ||
		configuredVariableInits != resolvedVariableInits ||
		configuredPackageInits != resolvedPackageInits {
		t.Errorf("configured/resolved mismatch: pkg=%d/%d method=%d/%d var=%d/%d init=%d/%d",
			configuredPackageFunctions, resolvedPackageFunctions,
			configuredMethods, resolvedMethods,
			configuredVariableInits, resolvedVariableInits,
			configuredPackageInits, resolvedPackageInits)
	}

	t.Logf("census: configured pkg=%d method=%d var=%d init=%d",
		configuredPackageFunctions, configuredMethods, configuredVariableInits, configuredPackageInits)
	t.Logf("census: resolved   pkg=%d method=%d var=%d init=%d",
		resolvedPackageFunctions, resolvedMethods, resolvedVariableInits, resolvedPackageInits)
}

// TestApprovalCensusExplicitOracleHashStable verifies the deterministic
// explicit oracle hash is reproducible across repeated runs and counts
// exactly 34 records, all equal. The hash is computed directly from the
// configured records, which the strict schema guarantees are
// already-explicit.
func TestApprovalCensusExplicitOracleHashStable(t *testing.T) {
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

// TestExplicitApprovalRecordEquivalence asserts the configured approval
// already carries the value validateApprovalSchema would require, so the
// configured form is structurally identical to the value the runtime
// uses. The strict schema is the migration target: any record whose
// configured form would fail the schema is a migration bug.
func TestExplicitApprovalRecordEquivalence(t *testing.T) {
	_, analysis := runProductionCanonicalAnalysisForCensus(t)
	records := buildApprovalCensus(analysis)

	categories := map[string]int{}
	for index, record := range records {
		if record.Configured.Function == "runSharedDupcodeBaseline" {
			verifyExplicitEquivalence(t, index, record)
			categories["package_function"]++
			continue
		}
		switch record.Configured.CallerKind {
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
	approval := record.Configured
	if approval.CallerKind == CallerKindMethod && approval.Receiver == "" {
		t.Errorf("index %d method has empty receiver", index)
	}
	if approval.CallerKind != CallerKindMethod && approval.Receiver != "" {
		t.Errorf("index %d non-method has non-empty receiver %q", index, approval.Receiver)
	}
	if len(validateApprovalSchema(approval)) != 0 {
		t.Errorf("index %d configured record fails strict schema: %+v", index, approval)
	}
}
