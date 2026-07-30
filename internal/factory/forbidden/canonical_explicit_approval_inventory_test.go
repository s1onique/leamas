// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// frozenNormalizationOracleHash is the SHA-256 oracle produced by the
// caller census for the production approval set. The explicit-inventory
// migration must preserve this exact value.
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

// TestProductionApprovalsRemainNormalizationInvariant asserts each
// production approval's explicit form equals the effective form
// produced by normalizeApproval. This proves the migration introduced
// no behavioral change: every record already carries the value the
// runtime would have filled in.
func TestProductionApprovalsRemainNormalizationInvariant(t *testing.T) {
	for _, source := range productionApprovalInventorySources() {
		for index, approval := range source.records {
			effective := normalizeApproval(approval)
			if !reflect.DeepEqual(effective, approval) {
				t.Errorf("%s[%d] %s.%s explicit != effective:\n  explicit = %+v\n  effective = %+v",
					source.name, index, approval.PackagePath, approval.Function,
					approval, effective)
			}
		}
	}
}

// TestExplicitApprovalInventoryPreservesOracleHash asserts the frozen
// normalized oracle hash is unchanged after the explicit-inventory
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
