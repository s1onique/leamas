package closure

import (
	"testing"
)

// canonicalComposedPlan returns a v1 closure plan JSON document that
// passes every validator (structural, applicability, typed, semantic).
// The fixture is intentionally plain to make assertions about the
// diagnostic stream easy to construct.
func canonicalComposedPlan() []byte {
	return []byte(`{"contract_version": 1, "act_id": "ACT-LEAMAS-COMPOSED", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
	"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}], "artifacts": [], 
	"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}}`)
}

// --- Phase 1+2: composition counter tests ---

func TestComposedParserRunsOnce(t *testing.T) {
	ResetCompositionCounters()
	composed := ValidatePlanComposed(canonicalComposedPlan())
	if !composed.Valid {
		t.Fatalf("canonical plan must compose-validate: structural=%v semantic=%v",
			composed.Structural.Errors, composed.Semantic)
	}
	if PlanParserCalls() != 1 {
		t.Fatalf("parser calls = %d, want exactly 1", PlanParserCalls())
	}
	if PlanTypedDecodeCalls() != 1 {
		t.Fatalf("typed decode calls = %d, want exactly 1", PlanTypedDecodeCalls())
	}
	if PlanSemanticValidateCalls() != 1 {
		t.Fatalf("semantic validation calls = %d, want exactly 1", PlanSemanticValidateCalls())
	}
}

func TestComposedSemanticRunsOnce(t *testing.T) {
	ResetCompositionCounters()
	// Even on a duplicated invocation the counters must still add up
	// to one call per stage per composed run.
	for i := 0; i < 3; i++ {
		composed := ValidatePlanComposed(canonicalComposedPlan())
		if !composed.Valid {
			t.Fatalf("iteration %d must validate: %v", i, composed.Structural.Errors)
		}
	}
	if PlanParserCalls() != 3 {
		t.Fatalf("parser calls = %d, want 3 (one per composed run)", PlanParserCalls())
	}
	if PlanTypedDecodeCalls() != 3 {
		t.Fatalf("typed decode calls = %d, want 3", PlanTypedDecodeCalls())
	}
	if PlanSemanticValidateCalls() != 3 {
		t.Fatalf("semantic calls = %d, want 3", PlanSemanticValidateCalls())
	}
}

func TestComposedSemanticFailureKeepsDecodedTrue(t *testing.T) {
	// A semantically invalid but structurally valid document: two
	// checks with the same id. Structural accepts it; semantic
	// (validatePlanChecks) rejects duplicate ids.
	plan := `{"contract_version": 1, "act_id": "ACT-DUP-ID", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
	"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "x", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}, {"id": "x", "mode": "run", 
	"argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}], "artifacts": [], "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, 
	"require_diff_check": true}}`
	ResetCompositionCounters()
	composed := ValidatePlanComposed([]byte(plan))
	if composed.Valid {
		t.Fatalf("duplicate check id must be semantically invalid")
	}
	if !composed.Decoded {
		t.Fatalf("Decoded must be true after successful typed decode even when semantic fails")
	}
	if composed.Semantic == nil {
		t.Fatalf("Semantic error must be populated")
	}
}

func TestComposedStructuralFailureKeepsDecodedFalse(t *testing.T) {
	// Trailing second JSON object: structural fails; Decoded stays
	// false because the typed decode must never run.
	plan := append(canonicalComposedPlan(), []byte(` {"x": 1}`)...)
	ResetCompositionCounters()
	composed := ValidatePlanComposed(plan)
	if composed.Valid {
		t.Fatalf("trailing second object must fail composed validation")
	}
	if composed.Decoded {
		t.Fatalf("Decoded must be false when structural fails")
	}
	if PlanTypedDecodeCalls() != 0 {
		t.Fatalf("typed decode must NOT be called on structural failure: got %d",
			PlanTypedDecodeCalls())
	}
}

// --- Phase 4: applicability tests ---

func TestApplicabilityMissingRunArgv(t *testing.T) {
	// /checks/0/mode=run but argv is absent.
	plan := `{"contract_version": 1, "act_id": "ACT-MISSING-ARGV", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
	"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "noop", "mode": "run", "working_directory": ".", "timeout_seconds": 60, "environment": {}}], "artifacts": [], "policy": {"require_clean_before": true, 
	"require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}}`
	result := ValidatePlanStructural([]byte(plan))
	if result.Valid {
		t.Fatalf("missing argv under run must be rejected by structural/applicability")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeRequiredPropertyMissing && e.InstancePath == "/checks/0/argv" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected required_property_missing at /checks/0/argv; got %v", result.Errors)
	}
}

func TestApplicabilityRunReasonForbidden(t *testing.T) {
	// /checks/0/mode=run with a present reason.
	plan := `{"contract_version": 1, "act_id": "ACT-RUN-REASON", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
	"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}, "reason": "noop"}], 
	"artifacts": [], "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}}`
	result := ValidatePlanStructural([]byte(plan))
	if result.Valid {
		t.Fatalf("reason under run must be rejected by applicability")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeSemanticConstraintFailed && e.InstancePath == "/checks/0/reason" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected semantic_constraint_failed at /checks/0/reason; got %v", result.Errors)
	}
}

func TestApplicabilityMissingExcludeReason(t *testing.T) {
	plan := `{"contract_version": 1, "act_id": "ACT-EXCL-NOREASON", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
	"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "x", "mode": "exclude"}], "artifacts": [], "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, 
	"require_diff_check": true}}`
	result := ValidatePlanStructural([]byte(plan))
	if result.Valid {
		t.Fatalf("exclude without reason must be rejected by applicability")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeRequiredPropertyMissing && e.InstancePath == "/checks/0/reason" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected required_property_missing at /checks/0/reason; got %v", result.Errors)
	}
}

func TestApplicabilityExcludeRunFieldsForbidden(t *testing.T) {
	// /checks/0/mode=exclude with argv present.
	plan := `{"contract_version": 1, "act_id": "ACT-EXCL-ARGV", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
	"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "x", "mode": "exclude", "reason": "noop", "argv": ["true"]}], "artifacts": [], "policy": {"require_clean_before": true, 
	"require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}}`
	result := ValidatePlanStructural([]byte(plan))
	if result.Valid {
		t.Fatalf("argv under exclude must be rejected by applicability")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeSemanticConstraintFailed && e.InstancePath == "/checks/0/argv" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected semantic_constraint_failed at /checks/0/argv; got %v", result.Errors)
	}
}

// --- Phase 5: integer lexical parity ---

func TestIntegerLexicalParityExponent(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{"timeout_seconds_1e2", `{"contract_version": 1, "act_id": "ACT-EXP", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
		"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 1e2, "environment": {}}], "artifacts": [], 
		"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}}`},
		{"max_bytes_1e0", `{"contract_version": 1, "act_id": "ACT-MB-EXP", "baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"}, 
		"execution": {"mode": "serial_fail_fast"}, "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}], "artifacts": [{"id": "summary", 
		"path": ".factory/gate-fast-summary.json", "required": true, "max_bytes": 1e0, "media_type": "application/json"}], "policy": {"require_clean_before": true, "require_clean_after": true, 
		"forbid_tracked_full_digests": true, "require_diff_check": true}}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanStructural([]byte(tc.data))
			if result.Valid {
				t.Fatalf("non-integer form must be rejected")
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == PlanCodeInvalidType {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected invalid_type diagnostic; got %v", result.Errors)
			}
		})
	}
}

func TestIntegerLexicalParityAcceptsCanonicalForms(t *testing.T) {
	// canonical form "60" must continue to pass; the lexical rule
	// accepts only what Go's int decoder accepts.
	result := ValidatePlanStructural(canonicalComposedPlan())
	if !result.Valid {
		t.Fatalf("canonical form must pass; errors=%v", result.Errors)
	}
}
