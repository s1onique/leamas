package closure

import (
	"strings"
	"testing"
)

// plan_contract_correction_single_document_test.go contains the
// Phase 1 single-document parsing tests for
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION01-
// STRUCTURAL-PARITY-CLOSURE01. Splitting it keeps every file under
// the LLM-friendly 400-line threshold.

// --- Phase 1: exact single-document parsing ---

func TestContractSingleDocumentWithTrailingWhitespaceAccepted(t *testing.T) {
	data := []byte(canonicalValidPlan() + "\n   \n")
	result := ValidatePlanStructural(data)
	if !result.Valid {
		t.Fatalf("valid plan + whitespace should pass; errors=%v", result.Errors)
	}
}

func TestContractSingleDocumentRejectsTrailingSecondObject(t *testing.T) {
	data := []byte(canonicalValidPlan() + ` {"surprise": true}`)
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("trailing second JSON object must be rejected")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeInvalidJSON && strings.Contains(e.Message, "trailing") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid_json with 'trailing' in message: %v", result.Errors)
	}
}

func TestContractSingleDocumentRejectsTrailingGarbage(t *testing.T) {
	data := []byte(canonicalValidPlan() + " garbage")
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("trailing garbage must be rejected")
	}
}

func TestContractDuplicateRootKeyRejected(t *testing.T) {
	data := []byte(`{
		"contract_version": 1,
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-DUP-ROOT",
		"baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
		"artifacts": [],
		"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
	}`)
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("duplicate root contract_version must be rejected")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeDuplicateProperty && e.InstancePath == "/contract_version" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unknown_property at /contract_version: %v", result.Errors)
	}
	// Typed decoder must agree.
	if _, err := DecodePlan(data); err == nil {
		t.Fatalf("typed decoder accepted duplicate root contract_version")
	}
}

func TestContractDuplicateNestedKeyRejected(t *testing.T) {
	data := []byte(`{
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-DUP-NESTED",
		"baseline": {"commit_oid": "1111111111111111111111111111111111111111", "commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
		"artifacts": [],
		"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
	}`)
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("duplicate nested commit_oid must be rejected")
	}
	if _, err := DecodePlan(data); err == nil {
		t.Fatalf("typed decoder accepted duplicate commit_oid")
	}
}

func TestContractDuplicateEnvironmentKeyRejected(t *testing.T) {
	data := []byte(`{
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-DUP-ENV",
		"baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {"FOO": "bar", "FOO": "baz"}}],
		"artifacts": [],
		"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
	}`)
	// Both structural and typed paths must reject duplicate keys
	// inside the free-form string-map environment.
	if _, err := DecodePlan(data); err == nil {
		t.Fatalf("typed decoder accepted duplicate environment key")
	}
	if ValidatePlanStructural(data).Valid {
		t.Fatalf("structural validator accepted duplicate environment key")
	}
}
