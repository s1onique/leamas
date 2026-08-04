// SPDX-License-Identifier: Apache-2.0

package forbidden

import "testing"

// frozenNormalizationOracleHash is the SHA-256 oracle produced by the
// caller census for the production approval set. The strict-schema
// migration must preserve this exact value because the configured
// records are already explicit.
const frozenNormalizationOracleHash = "dc21de26d1a0abfdaaf60523ae551056a66c508a12de73a647d60b8cdb2b2b25"

// approvalInventorySource ties an inventory name to its configured
// records so test failures can identify which slice a record came from.
type approvalInventorySource struct {
	name    string
	records []ApprovedCaller
}

// productionApprovalInventorySources returns the four production
// approval inventories in their canonical order.
func productionApprovalInventorySources() []approvalInventorySource {
	return []approvalInventorySource{
		{name: "ApprovedCallers", records: ApprovedCallers},
		{name: "dupcodeInternalApprovedCallers", records: dupcodeInternalApprovedCallers},
		{name: "AdapterApprovedCallers", records: AdapterApprovedCallers},
		{name: "GateApprovedCallers", records: GateApprovedCallers},
	}
}

// TestProductionApprovalInventoryIsExplicit asserts every configured
// approval record in every production inventory carries an explicit
// CallerKind, ReferenceClass, and Cardinality. No implicit fields
// remain. The total record count must remain 34.
func TestProductionApprovalInventoryIsExplicit(t *testing.T) {
	total := 0
	for _, source := range productionApprovalInventorySources() {
		for index, approval := range source.records {
			total++
			if approval.CallerKind == "" {
				t.Errorf("%s[%d] %s.%s CallerKind is empty", source.name, index, approval.PackagePath, approval.Function)
			}
			if approval.ReferenceClass == "" {
				t.Errorf("%s[%d] %s.%s ReferenceClass is empty", source.name, index, approval.PackagePath, approval.Function)
			}
			if !validApprovalReferenceClass(approval.ReferenceClass) {
				t.Errorf("%s[%d] %s.%s ReferenceClass %q is not approvable",
					source.name, index, approval.PackagePath, approval.Function, approval.ReferenceClass)
			}
			if approval.Cardinality <= 0 {
				t.Errorf("%s[%d] %s.%s Cardinality = %d, want positive",
					source.name, index, approval.PackagePath, approval.Function, approval.Cardinality)
			}
		}
	}
	if total != 34 {
		t.Fatalf("total approval count = %d, want 34", total)
	}
}

// TestProductionApprovalInventoryPassesStrictSchema asserts every
// configured approval in every production inventory passes
// validateApprovalSchema with zero issues. This is the strict-schema
// regression gate: any record whose configured form would fail the
// schema is a migration bug and breaks the 34/34 production truth.
func TestProductionApprovalInventoryPassesStrictSchema(t *testing.T) {
	total := 0
	for _, source := range productionApprovalInventorySources() {
		for index, approval := range source.records {
			total++
			issues := validateApprovalSchema(approval)
			if len(issues) != 0 {
				for _, issue := range issues {
					t.Errorf("%s[%d] %s.%s schema issue field=%s kind=%s message=%s",
						source.name, index,
						approval.PackagePath, approval.Function,
						issue.Field, issue.Kind, issue.Message)
				}
			}
		}
	}
	if total != 34 {
		t.Fatalf("total approval count = %d, want 34", total)
	}
}

// TestExplicitApprovalInventoryPreservesOracleHash asserts the frozen
// strict-schema oracle hash is unchanged after the schema validation
// migration. Any drift in record order, field values, or record count
// breaks the hash and fails this test.
func TestExplicitApprovalInventoryPreservesOracleHash(t *testing.T) {
	_, analysis := runProductionCanonicalAnalysisForCensus(t)
	records := buildApprovalCensus(analysis)
	if len(records) != 34 {
		t.Fatalf("census record count = %d, want 34", len(records))
	}
	current := hashCensusRecords(records)
	if current != frozenNormalizationOracleHash {
		t.Fatalf("oracle hash drifted: got %s want %s", current, frozenNormalizationOracleHash)
	}
	t.Logf("oracle hash matches frozen value: %s", current)
}
