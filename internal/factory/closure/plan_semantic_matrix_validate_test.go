package closure

import (
	"errors"
	"strings"
	"testing"
)

// TestValidatePlanSemanticMatrix tests every semantic error path through
// the real ValidatePlan function by mutating a valid fixture.
func TestValidatePlanSemanticMatrix(t *testing.T) {
	// Test cases: (description, mutator, wantCount, wantPath, wantCode)
	cases := []struct {
		name      string
		mutate    func(*Plan)
		wantCount int
		wantPath  string
		wantCode  PlanValidationCode
	}{
		// Baseline OID placeholders - structural, not semantic
		// ActID - uses "type" keyword in practice
		{
			name:      "act_id placeholder",
			mutate:    func(p *Plan) { p.ActID = "TODO" },
			wantCount: 1, wantPath: "/act_id",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "act_id invalid format",
			mutate:    func(p *Plan) { p.ActID = "not-valid" },
			wantCount: 1, wantPath: "/act_id",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Baseline commit OID
		{
			name:      "baseline commit_oid placeholder",
			mutate:    func(p *Plan) { p.Baseline.CommitOID = "TODO-GIT-OID" },
			wantCount: 1, wantPath: "/baseline/commit_oid",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "baseline commit_oid invalid format",
			mutate:    func(p *Plan) { p.Baseline.CommitOID = "not-valid-oid" },
			wantCount: 1, wantPath: "/baseline/commit_oid",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Baseline tree OID
		{
			name:      "baseline tree_oid placeholder",
			mutate:    func(p *Plan) { p.Baseline.TreeOID = "TODO-GIT-OID" },
			wantCount: 1, wantPath: "/baseline/tree_oid",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Execution mode
		{
			name:      "execution mode missing",
			mutate:    func(p *Plan) { p.Execution.Mode = nil },
			wantCount: 1, wantPath: "/execution/mode",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "execution mode empty",
			mutate:    func(p *Plan) { m := ExecutionMode(""); p.Execution.Mode = &m },
			wantCount: 1, wantPath: "/execution/mode",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "execution mode unknown",
			mutate:    func(p *Plan) { m := ExecutionMode("unknown"); p.Execution.Mode = &m },
			wantCount: 1, wantPath: "/execution/mode",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Checks - count
		{
			name:      "checks empty",
			mutate:    func(p *Plan) { p.Checks = []PlanCheck{} },
			wantCount: 1, wantPath: "/checks",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Checks - ID
		{
			name:      "checks 0 id placeholder",
			mutate:    func(p *Plan) { p.Checks[0].ID = "TODO" },
			wantCount: 1, wantPath: "/checks/0/id",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks 0 id invalid format",
			mutate:    func(p *Plan) { p.Checks[0].ID = "Invalid-ID" },
			wantCount: 1, wantPath: "/checks/0/id",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks duplicate id",
			mutate:    func(p *Plan) { p.Checks[1].ID = p.Checks[0].ID },
			wantCount: 1, wantPath: "/checks/1/id",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Checks - mode
		{
			name:      "checks 0 mode unknown",
			mutate:    func(p *Plan) { p.Checks[0].Mode = "unknown" },
			wantCount: 1, wantPath: "/checks/0/mode",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Checks - argv
		{
			name:      "checks 0 argv empty",
			mutate:    func(p *Plan) { p.Checks[0].Argv = []string{} },
			wantCount: 1, wantPath: "/checks/0/argv",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks 0 argv element empty",
			mutate:    func(p *Plan) { p.Checks[0].Argv[1] = "" },
			wantCount: 1, wantPath: "/checks/0/argv/1",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks 0 argv element placeholder",
			mutate:    func(p *Plan) { p.Checks[0].Argv[0] = "TODO" },
			wantCount: 1, wantPath: "/checks/0/argv/0",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Checks - working_directory
		{
			name:      "checks 0 working_directory absolute",
			mutate:    func(p *Plan) { p.Checks[0].WorkingDirectory = "/absolute" },
			wantCount: 1, wantPath: "/checks/0/working_directory",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Checks - timeout_seconds
		{
			name:      "checks 0 timeout_seconds zero",
			mutate:    func(p *Plan) { p.Checks[0].TimeoutSeconds = 0 },
			wantCount: 1, wantPath: "/checks/0/timeout_seconds",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Checks - environment
		{
			name:      "checks 0 environment nil",
			mutate:    func(p *Plan) { p.Checks[0].Environment = nil },
			wantCount: 1, wantPath: "/checks/0/environment",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Exclusion check - reason
		{
			name:      "checks 1 reason empty",
			mutate:    func(p *Plan) { p.Checks[1].Reason = "" },
			wantCount: 1, wantPath: "/checks/1/reason",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks 1 reason multiline",
			mutate:    func(p *Plan) { p.Checks[1].Reason = "line1\nline2" },
			wantCount: 1, wantPath: "/checks/1/reason",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Exclusion check - forbidden fields
		{
			name:      "checks 1 exclude with argv",
			mutate:    func(p *Plan) { p.Checks[1].Argv = []string{"forbidden"} },
			wantCount: 1, wantPath: "/checks/1/argv",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks 1 exclude with working_directory",
			mutate:    func(p *Plan) { p.Checks[1].WorkingDirectory = "forbidden" },
			wantCount: 1, wantPath: "/checks/1/working_directory",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks 1 exclude with timeout",
			mutate:    func(p *Plan) { p.Checks[1].TimeoutSeconds = 100 },
			wantCount: 1, wantPath: "/checks/1/timeout_seconds",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "checks 1 exclude with environment",
			mutate:    func(p *Plan) { p.Checks[1].Environment = map[string]string{"X": "Y"} },
			wantCount: 1, wantPath: "/checks/1/environment",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Artifacts - count
		{
			name:      "artifacts exceeds limit",
			mutate:    func(p *Plan) { p.Artifacts = make([]PlanArtifact, MaxArtifacts+1) },
			wantCount: 1, wantPath: "/artifacts",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Artifacts - ID
		{
			name:      "artifacts 0 id placeholder",
			mutate:    func(p *Plan) { p.Artifacts[0].ID = "TODO" },
			wantCount: 1, wantPath: "/artifacts/0/id",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name: "artifacts duplicate id",
			mutate: func(p *Plan) {
				p.Artifacts = append(p.Artifacts, PlanArtifact{
					ID:        p.Artifacts[0].ID,
					Path:      "other",
					Required:  boolPtr(true),
					MaxBytes:  1000,
					MediaType: "text/plain",
				})
			},
			wantCount: 1, wantPath: "/artifacts/1/id",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Artifacts - path
		{
			name:      "artifacts 0 path absolute",
			mutate:    func(p *Plan) { p.Artifacts[0].Path = "/absolute" },
			wantCount: 1, wantPath: "/artifacts/0/path",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Artifacts - required
		{
			name:      "artifacts 0 required nil",
			mutate:    func(p *Plan) { p.Artifacts[0].Required = nil },
			wantCount: 1, wantPath: "/artifacts/0/required",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Artifacts - max_bytes
		{
			name:      "artifacts 0 max_bytes zero",
			mutate:    func(p *Plan) { p.Artifacts[0].MaxBytes = 0 },
			wantCount: 1, wantPath: "/artifacts/0/max_bytes",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Artifacts - media_type
		{
			name:      "artifacts 0 media_type empty",
			mutate:    func(p *Plan) { p.Artifacts[0].MediaType = "" },
			wantCount: 1, wantPath: "/artifacts/0/media_type",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "artifacts 0 media_type placeholder",
			mutate:    func(p *Plan) { p.Artifacts[0].MediaType = "TODO" },
			wantCount: 1, wantPath: "/artifacts/0/media_type",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Policy
		{
			name:      "policy require_clean_before missing",
			mutate:    func(p *Plan) { p.Policy.RequireCleanBefore = nil },
			wantCount: 1, wantPath: "/policy/require_clean_before",
			wantCode: PlanCodeRequiredPropertyMissing,
		},
		{
			name:      "policy require_diff_check missing",
			mutate:    func(p *Plan) { p.Policy.RequireDiffCheck = nil },
			wantCount: 1, wantPath: "/policy/require_diff_check",
			wantCode: PlanCodeRequiredPropertyMissing,
		},
		{
			name:      "policy require_clean_before false",
			mutate:    func(p *Plan) { p.Policy.RequireCleanBefore = boolPtr(false) },
			wantCount: 1, wantPath: "/policy",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		{
			name:      "policy require_clean_after false",
			mutate:    func(p *Plan) { p.Policy.RequireCleanAfter = boolPtr(false) },
			wantCount: 1, wantPath: "/policy",
			wantCode: PlanCodeSemanticConstraintFailed,
		},
		// Runner authority - these return plain errors without diagnostics
		// We test them separately for diagnostics, but they do produce errors
		{
			name:      "runner_authority mode unknown",
			mutate:    func(p *Plan) { p.RunnerAuthority.Mode = "unknown" },
			wantCount: 0, wantPath: "", // Plain error, no diagnostics
			wantCode: "",
		},
		{
			name: "runner_authority tool_release_exact without tool",
			mutate: func(p *Plan) {
				p.RunnerAuthority = &RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact}
			},
			wantCount: 0, wantPath: "", // Plain error, no diagnostics
			wantCode: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := validSemanticPlanFixture(t)
			tc.mutate(&plan)

			err := ValidatePlan(plan)
			if err == nil {
				t.Fatalf("ValidatePlan() = nil, want error")
			}

			var diags []PlanValidationError
			var source planDiagnosticSource
			if errors.As(err, &source) {
				diags = source.PlanDiagnostics()
			}

			if tc.wantCount == 0 {
				// Expecting plain error without diagnostics
				if len(diags) != 0 {
					t.Errorf("got %d diagnostics, want 0 (plain error)", len(diags))
				}
				return
			}

			if len(diags) == 0 {
				t.Fatalf("no diagnostics extracted from %T", err)
			}

			if len(diags) != tc.wantCount {
				t.Errorf("got %d diagnostics, want %d", len(diags), tc.wantCount)
			}

			if diags[0].InstancePath != tc.wantPath {
				t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, tc.wantPath)
			}
			if tc.wantCode != "" && diags[0].Code != tc.wantCode {
				t.Errorf("Code = %v, want %v", diags[0].Code, tc.wantCode)
			}
		})
	}
}

// TestRunnerAuthorityDiagnosticIdentityClosed verifies the closed
// field-to-pointer mapping for runner authority diagnostics.
func TestRunnerAuthorityDiagnosticIdentityClosed(t *testing.T) {
	// Test that known fields resolve
	knownFields := []string{
		"mode", "tool", "tool.revision", "tool.tree_oid",
		"tool.binary_sha256", "tool.version", "tool.tag_name",
		"tool.tag_object_oid", "vcs.revision", "vcs.modified",
		"binary_sha256", "target.subject", "target.tree",
	}
	for _, field := range knownFields {
		t.Run("known_"+field, func(t *testing.T) {
			err := &RunnerAuthorityError{Field: field, Message: "test"}
			diags := err.PlanDiagnostics()
			if len(diags) != 1 {
				t.Fatalf("got %d diags, want 1", len(diags))
			}
			if diags[0].InstancePath == "" {
				t.Errorf("InstancePath is empty for known field %q", field)
			}
			if diags[0].Code != PlanCodeSemanticConstraintFailed {
				t.Errorf("Code = %v, want %v", diags[0].Code, PlanCodeSemanticConstraintFailed)
			}
		})
	}

	// Test that unknown fields fail closed
	unknownFields := []string{
		"unknown", "tool.revison", "vcs.revison",
		"target", "vcs", "binary",
	}
	for _, field := range unknownFields {
		t.Run("unknown_"+field, func(t *testing.T) {
			err := &RunnerAuthorityError{Field: field, Message: "test"}
			diags := err.PlanDiagnostics()
			if len(diags) != 1 {
				t.Fatalf("got %d diags, want 1", len(diags))
			}
			// Unknown fields should NOT produce a plausible plan pointer
			if strings.HasPrefix(diags[0].InstancePath, "/runner_authority/") &&
				!strings.Contains(diags[0].InstancePath, "unknown") {
				t.Errorf("unknown field %q produced path %q, want empty or /runner_authority/unknown",
					field, diags[0].InstancePath)
			}
		})
	}
}

// TestPolicyDiagnosticOrder verifies diagnostics are emitted in
// PolicyFieldOrder regardless of input order.
func TestPolicyDiagnosticOrder(t *testing.T) {
	// Build plan with missing fields in reverse order
	plan := validSemanticPlanFixture(t)
	plan.Policy.RequireCleanBefore = nil
	plan.Policy.RequireDiffCheck = nil

	err := ValidatePlan(plan)
	if err == nil {
		t.Fatal("ValidatePlan() = nil, want error")
	}

	var source planDiagnosticSource
	if !errors.As(err, &source) {
		t.Fatal("error does not implement planDiagnosticSource")
	}

	diags := source.PlanDiagnostics()
	order := PolicyFieldOrder()

	// Verify order matches PolicyFieldOrder
	wantOrder := []string{}
	for _, name := range order {
		if name == "require_clean_before" || name == "require_diff_check" {
			wantOrder = append(wantOrder, name)
		}
	}

	if len(diags) != len(wantOrder) {
		t.Fatalf("got %d diags, want %d", len(diags), len(wantOrder))
	}

	for i, want := range wantOrder {
		if diags[i].PropertyName != want {
			t.Errorf("diagnostic[%d].PropertyName = %q, want %q", i, diags[i].PropertyName, want)
		}
	}
}

// TestEnvironmentSemanticDeterminism verifies environment validation
// produces identical errors regardless of map iteration order.
func TestEnvironmentSemanticDeterminism(t *testing.T) {
	// Insert invalid key in forward order
	fwd := validSemanticPlanFixture(t)
	fwd.Checks[0].Environment = map[string]string{
		"VALID_KEY":   "value",
		"invalid/key": "value", // Contains /
	}

	// Insert invalid key in reverse order
	rev := validSemanticPlanFixture(t)
	rev.Checks[0].Environment = map[string]string{
		"invalid/key": "value",
		"VALID_KEY":   "value",
	}

	errFwd := ValidatePlan(fwd)
	errRev := ValidatePlan(rev)

	if errFwd == nil || errRev == nil {
		t.Fatal("both plans should fail validation")
	}

	// Extract diagnostics
	var srcFwd, srcRev planDiagnosticSource
	errors.As(errFwd, &srcFwd)
	errors.As(errRev, &srcRev)

	diagsFwd := srcFwd.PlanDiagnostics()
	diagsRev := srcRev.PlanDiagnostics()

	if len(diagsFwd) != len(diagsRev) {
		t.Fatalf("different diagnostic counts: fwd=%d rev=%d", len(diagsFwd), len(diagsRev))
	}

	for i := range diagsFwd {
		if diagsFwd[i].InstancePath != diagsRev[i].InstancePath {
			t.Errorf("diagnostic[%d] InstancePath differs: fwd=%q rev=%q",
				i, diagsFwd[i].InstancePath, diagsRev[i].InstancePath)
		}
		if diagsFwd[i].Code != diagsRev[i].Code {
			t.Errorf("diagnostic[%d] Code differs: fwd=%v rev=%v",
				i, diagsFwd[i].Code, diagsRev[i].Code)
		}
	}
}

// TestSemanticComposedJSON verifies the composed validation result structure.
// The test uses a plan that should fail semantic validation with the
// reason field cleared on an exclusion check.
func TestSemanticComposedJSON(t *testing.T) {
	// Build a plan with a semantic error (missing reason on exclusion check)
	plan := validSemanticPlanFixture(t)
	plan.Checks[1].Reason = ""

	// The plan should fail semantic validation directly
	err := ValidatePlan(plan)
	if err == nil {
		t.Fatal("ValidatePlan() = nil, want error")
	}

	// Verify it implements planDiagnosticSource
	var source planDiagnosticSource
	if !errors.As(err, &source) {
		t.Fatal("error does not implement planDiagnosticSource")
	}

	diags := source.PlanDiagnostics()
	if len(diags) == 0 {
		t.Fatal("no diagnostics from semantic error")
	}

	// Verify the diagnostic has expected properties
	if diags[0].InstancePath != "/checks/1/reason" {
		t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/checks/1/reason")
	}
	if diags[0].Code != PlanCodeSemanticConstraintFailed {
		t.Errorf("Code = %v, want %v", diags[0].Code, PlanCodeSemanticConstraintFailed)
	}
}
