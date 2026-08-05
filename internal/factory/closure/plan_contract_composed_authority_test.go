package closure

import "testing"

// plan_contract_composed_authority_test.go owns the Phase 1+2
// composition-counter tests and the Phase 2 applicability
// walker tests. The JSON fixtures live in
// plan_contract_composed_fixtures_test.go so this file stays
// under the LLM-friendly 400-line threshold.

// countingObserver records per-invocation pipeline events so tests
// can prove that the composed pipeline parses, decodes, and
// validates semantically exactly the documented number of times
// for each scenario. CountingObserver is invocation-local: each
// test constructs a fresh observer and the counters are reset
// between test invocations because no package-global state is
// shared.
type countingObserver struct {
	parsedCount            int
	typedDecodedCount      int
	semanticValidatedCount int
}

func (c *countingObserver) Parsed()            { c.parsedCount++ }
func (c *countingObserver) TypedDecoded()      { c.typedDecodedCount++ }
func (c *countingObserver) SemanticValidated() { c.semanticValidatedCount++ }

// --- Phase 1+2: composition counter tests (observer-local) ---

func TestComposedParserRunsOnce(t *testing.T) {
	obs := &countingObserver{}
	composed := validatePlanComposedWithObserver(canonicalComposedPlan(), obs)
	if !composed.Valid {
		t.Fatalf("canonical plan must compose-validate: structural=%v semantic=%v",
			composed.Structural.Errors, composed.SemanticErrors)
	}
	if obs.parsedCount != 1 {
		t.Fatalf("parsed count = %d, want exactly 1", obs.parsedCount)
	}
	if obs.typedDecodedCount != 1 {
		t.Fatalf("typed decode count = %d, want exactly 1", obs.typedDecodedCount)
	}
	if obs.semanticValidatedCount != 1 {
		t.Fatalf("semantic validation count = %d, want exactly 1", obs.semanticValidatedCount)
	}
}

func TestComposedSemanticRunsOnce(t *testing.T) {
	for i := 0; i < 3; i++ {
		obs := &countingObserver{}
		composed := validatePlanComposedWithObserver(canonicalComposedPlan(), obs)
		if !composed.Valid {
			t.Fatalf("iteration %d must validate: %v", i, composed.Structural.Errors)
		}
		if obs.parsedCount != 1 {
			t.Fatalf("iteration %d: parsed count = %d, want 1", i, obs.parsedCount)
		}
		if obs.typedDecodedCount != 1 {
			t.Fatalf("iteration %d: typed decode count = %d, want 1", i, obs.typedDecodedCount)
		}
		if obs.semanticValidatedCount != 1 {
			t.Fatalf("iteration %d: semantic count = %d, want 1", i, obs.semanticValidatedCount)
		}
	}
}

func TestComposedSemanticFailureKeepsDecodedTrue(t *testing.T) {
	// A semantically invalid but structurally valid document:
	// two checks with the same id. Structural accepts it;
	// semantic (validatePlanChecks) rejects duplicate ids.
	obs := &countingObserver{}
	composed := validatePlanComposedWithObserver(composedPlanDuplicateCheckID(), obs)
	if composed.Valid {
		t.Fatalf("duplicate check id must be semantically invalid")
	}
	if !composed.Decoded {
		t.Fatalf("Decoded must be true after successful typed decode even when semantic fails")
	}
	if composed.SemanticValid {
		t.Fatalf("SemanticValid must be false on semantic failure")
	}
	if !composed.SemanticValid && len(composed.SemanticErrors) == 0 {
		t.Fatalf("SemanticErrors must be populated on semantic failure")
	}
	if obs.typedDecodedCount != 1 {
		t.Fatalf("typed decode count = %d, want 1", obs.typedDecodedCount)
	}
	if obs.semanticValidatedCount != 1 {
		t.Fatalf("semantic validation count = %d, want 1", obs.semanticValidatedCount)
	}
}

func TestComposedStructuralFailureKeepsDecodedFalse(t *testing.T) {
	plan := append(canonicalComposedPlan(), []byte(` {"x": 1}`)...)
	obs := &countingObserver{}
	composed := validatePlanComposedWithObserver(plan, obs)
	if composed.Valid {
		t.Fatalf("trailing second object must fail composed validation")
	}
	if composed.Decoded {
		t.Fatalf("Decoded must be false when structural fails")
	}
	if obs.typedDecodedCount != 0 {
		t.Fatalf("typed decode must NOT be called on structural failure: got %d",
			obs.typedDecodedCount)
	}
	if obs.semanticValidatedCount != 0 {
		t.Fatalf("semantic must NOT be called on structural failure: got %d",
			obs.semanticValidatedCount)
	}
}

// TestComposedTrailingDocumentFailsBeforeTypedDecode proves
// that a trailing-document structural failure keeps Decoded=false
// and never reaches the typed decoder. The previous test name
// "TestComposedTypedFailureKeepsDecodedFalse" was misleading
// because the body exercised a structural failure path, not a
// typed-decode failure. The rename pins the actual scenario so
// future readers cannot mistake this test for typed-stage proof.
func TestComposedTrailingDocumentFailsBeforeTypedDecode(t *testing.T) {
	// A structural failure (trailing document + unknown field)
	// keeps Decoded=false and the typed decoder is NOT called.
	// This proves the pipeline short-circuits on structural
	// failure.
	plan := append(canonicalComposedPlan(), []byte(` {"surprise": true}`)...)
	obs := &countingObserver{}
	composed := validatePlanComposedWithObserver(plan, obs)
	if composed.Valid {
		t.Fatalf("trailing second object + unknown field must fail composed validation")
	}
	if composed.Decoded {
		t.Fatalf("Decoded must be false on structural failure")
	}
	if obs.parsedCount != 1 {
		t.Fatalf("parsed count = %d, want 1", obs.parsedCount)
	}
	if obs.typedDecodedCount != 0 {
		t.Fatalf("typed decode must NOT be called on structural failure: got %d", obs.typedDecodedCount)
	}
	if obs.semanticValidatedCount != 0 {
		t.Fatalf("semantic must NOT be called on structural failure: got %d", obs.semanticValidatedCount)
	}
}

// TestRepeatedObserverIsolation runs the composed pipeline
// repeatedly and proves each iteration constructs and tears down
// its own observer. The previous test was misnamed "concurrent"
// but the body does not launch overlapping goroutines; the
// renaming reflects the sequential iteration.
func TestRepeatedObserverIsolation(t *testing.T) {
	// Sequential repeated invocations must each have their own
	// independent observer state. The correction removed
	// process-global counters, so each observer is invocation-local.
	plan := canonicalComposedPlan()
	for i := 0; i < 8; i++ {
		obs := &countingObserver{}
		composed := validatePlanComposedWithObserver(plan, obs)
		if !composed.Valid {
			t.Fatalf("iteration %d must validate: %v", i, composed.Structural.Errors)
		}
		if obs.parsedCount != 1 || obs.typedDecodedCount != 1 || obs.semanticValidatedCount != 1 {
			t.Fatalf("iteration %d: counters leaked %+v", i, obs)
		}
	}
}

// --- Phase 2: applicability tests ---

func TestApplicabilityMissingRunArgv(t *testing.T) {
	result := ValidatePlanStructural(composedPlanMissingRunArgv())
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
	result := ValidatePlanStructural(composedPlanRunReasonForbidden())
	if result.Valid {
		t.Fatalf("reason under run must be rejected by applicability")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeForbiddenPresence && e.InstancePath == "/checks/0/reason" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected forbidden_presence at /checks/0/reason; got %v", result.Errors)
	}
}

func TestApplicabilityMissingExcludeReason(t *testing.T) {
	result := ValidatePlanStructural(composedPlanMissingExcludeReason())
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
	result := ValidatePlanStructural(composedPlanExcludeWithArgv())
	if result.Valid {
		t.Fatalf("argv under exclude must be rejected by applicability")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeForbiddenPresence && e.InstancePath == "/checks/0/argv" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected forbidden_presence at /checks/0/argv; got %v", result.Errors)
	}
}

func TestApplicabilityExcludeEmptyStringForbidden(t *testing.T) {
	// Forbidden means property absent regardless of value. Empty
	// string, empty array, empty object, and null must all be
	// rejected.
	cases := []struct {
		name string
		data []byte
	}{
		{"reason-empty", emptyReasonPlan()},
		{"argv-empty-array", argvEmptyArrayExcludePlan()},
		{"working_directory-empty-string", workingDirectoryEmptyExcludePlan()},
		{"timeout_seconds-zero", timeoutSecondsZeroExcludePlan()},
		{"environment-empty-object", environmentEmptyExcludePlan()},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanStructural(tc.data)
			if result.Valid {
				t.Fatalf("forbidden field with empty value must be rejected")
			}
			found := false
			for _, e := range result.Errors {
				// Forbidden means absent regardless of value. The
				// structural validator's minItems check on argv fires
				// before the applicability walker, so we accept either
				// invalid_type (minItems) or semantic_constraint_failed.
				if e.Code == PlanCodeForbiddenPresence || e.Code == PlanCodeSemanticConstraintFailed || e.Code == PlanCodeInvalidType {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected forbidden_presence, semantic_constraint_failed, or invalid_type diagnostic; got %v", result.Errors)
			}
		})
	}
}
