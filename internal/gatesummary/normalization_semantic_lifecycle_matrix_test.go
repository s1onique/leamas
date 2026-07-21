package gatesummary

import (
	"strings"
	"testing"
)

// lifecycleMatrixCase is one row of the lifecycle/aggregate matrix.
// Checks carry statuses that derive the recorded overall_status.
// This isolates scope/parent/overall status preservation from
// per-check validators.
type lifecycleMatrixCase struct {
	ID              string
	ScopeStatus     string
	ParentStatus    string
	OverallStatus   string
	Checks          string
	WantNormOK      bool
	WantDiagnostics []diagnosticProjection
}

// lifecycleMatrixBody wraps the chosen checks slice with the
// requested scope/parent/overall lifecycle fields.
func lifecycleMatrixBody(scope, parent, overall, checksJSON string) string {
	if checksJSON == "" {
		checksJSON = "[]"
	}
	return `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-LIFECYCLE-MATRIX",
		"scope_status": "` + scope + `",
		"scope_disposition": "lifecycle matrix",
		"parent_act": "",
		"parent_status": "` + parent + `",
		"parent_disposition": "lifecycle matrix",
		"overall_status": "` + overall + `",
		"overall_disposition": "lifecycle matrix",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": ` + checksJSON + `
	}`
}

// distinctCheckForLC returns a check JSON with a unique name so
// the ClineMM-style multi-check rows don't trip duplicate-name.
func distinctCheckForLC(name, status, exitCodeWire string) string {
	return checkJSONForMatrix(name, status, exitCodeWire)
}

// lifecycleMatrix is the frozen scope/parent/overall matrix.
// The six closed/open combinations plus three aggregate-status
// variations cover the v2 evidence topology without inferring one
// field from another. Every row's checks derive the recorded
// overall_status so the only diagnostic sources are cleanliness
// and overall mismatch (or none).
var lifecycleMatrix = []lifecycleMatrixCase{
	{
		ID:            "LC-001",
		ScopeStatus:   "CLOSED",
		ParentStatus:  "OPEN",
		OverallStatus: "pass",
		Checks:        "[" + distinctCheckForLC("lc1", "pass", "0") + "]",
		WantNormOK:    true,
	},
	{
		ID:            "LC-002",
		ScopeStatus:   "CLOSED",
		ParentStatus:  "OPEN",
		OverallStatus: "fail",
		Checks:        "[" + distinctCheckForLC("lc2", "fail", "1") + "]",
		WantNormOK:    true,
	},
	{
		ID:            "LC-003",
		ScopeStatus:   "OPEN",
		ParentStatus:  "OPEN",
		OverallStatus: "fail",
		Checks:        "[" + distinctCheckForLC("lc3", "fail", "1") + "]",
		WantNormOK:    true,
	},
	{
		ID:            "LC-004",
		ScopeStatus:   "CLOSED",
		ParentStatus:  "CLOSED",
		OverallStatus: "pass",
		Checks:        "[" + distinctCheckForLC("lc4", "pass", "0") + "]",
		WantNormOK:    true,
	},
	{
		ID:            "LC-005",
		ScopeStatus:   "CLOSED",
		ParentStatus:  "CLOSED",
		OverallStatus: "fail",
		Checks:        "[" + distinctCheckForLC("lc5", "fail", "1") + "]",
		WantNormOK:    true,
	},
	{
		ID:            "LC-006",
		ScopeStatus:   "OPEN",
		ParentStatus:  "CLOSED",
		OverallStatus: "pass",
		Checks:        "[" + distinctCheckForLC("lc6", "pass", "0") + "]",
		WantNormOK:    true,
	},
	{
		// ClineMM µC-3 topology: closed scope, open parent, fail
		// aggregate (1 pass + 1 fail derives fail).
		ID:            "LC-007",
		ScopeStatus:   "CLOSED",
		ParentStatus:  "OPEN",
		OverallStatus: "fail",
		Checks: "[" +
			distinctCheckForLC("lc7p", "pass", "0") + "," +
			distinctCheckForLC("lc7f", "fail", "1") + "]",
		WantNormOK: true,
	},
	{
		// Partial scope with fail aggregate is valid: derived=fail,
		// recorded=fail, partial preserved.
		ID:            "LC-008",
		ScopeStatus:   "PARTIAL",
		ParentStatus:  "OPEN",
		OverallStatus: "fail",
		Checks:        "[" + distinctCheckForLC("lc8", "fail", "1") + "]",
		WantNormOK:    true,
	},
}

// TestSemanticLifecycleMatrix walks every lifecycle matrix row.
// It asserts that scope/parent/aggregate statuses are independent
// and never inferred from one another.
func TestSemanticLifecycleMatrix(t *testing.T) {
	if got := len(lifecycleMatrix); got != 8 {
		t.Fatalf("lifecycle matrix has %d rows, want 8", got)
	}
	for _, c := range lifecycleMatrix {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			wire := lifecycleMatrixBody(c.ScopeStatus, c.ParentStatus, c.OverallStatus, c.Checks)
			dec := Decode(strings.NewReader(wire))
			if !dec.Success() {
				t.Fatalf("%s: decode failed: %v", c.ID, dec.Diagnostics)
			}
			norm := Normalize(dec.Document)
			got := projectDiagnostics(norm.Diagnostics)
			if !compareProjections(got, c.WantDiagnostics) {
				t.Fatalf("%s: diagnostics = %#v, want %#v",
					c.ID, got, c.WantDiagnostics)
			}
			if norm.Success() != c.WantNormOK {
				t.Fatalf("%s: normalize success = %v, want %v",
					c.ID, norm.Success(), c.WantNormOK)
			}
			if norm.Success() {
				if norm.Summary.Scope == nil {
					t.Fatalf("%s: missing normalized scope", c.ID)
				}
				if got := string(norm.Summary.Scope.Status); got != strings.ToLower(c.ScopeStatus) {
					t.Fatalf("%s: scope status = %q, want %q",
						c.ID, got, strings.ToLower(c.ScopeStatus))
				}
				if norm.Summary.Parent == nil {
					t.Fatalf("%s: missing normalized parent", c.ID)
				}
				if got := string(norm.Summary.Parent.Status); got != strings.ToLower(c.ParentStatus) {
					t.Fatalf("%s: parent status = %q, want %q",
						c.ID, got, strings.ToLower(c.ParentStatus))
				}
				if got := string(norm.Summary.Overall.Status); got != c.OverallStatus {
					t.Fatalf("%s: overall status = %q, want %q",
						c.ID, got, c.OverallStatus)
				}
			}
		})
	}
}

// TestNormalizeV2_PreservesClosedScopeOpenParentFailedAggregate
// is the named downstream regression. It asserts the canonical
// ClineMM CORRECTION21 µC-3 evidence topology is preserved
// exactly. The three fields must be recorded independently.
func TestNormalizeV2_PreservesClosedScopeOpenParentFailedAggregate(t *testing.T) {
	data := []byte(`{
		"schema_version": 2,
		"generated_at": "2026-07-19T08:43:26.649Z",
		"scope_id": "ACT-CLINEMM-FORK-BASELINE01-CORRECTION21-MICROC3",
		"scope_status": "CLOSED",
		"scope_disposition": "µC-3 reader authority closed",
		"parent_act": "ACT-CLINEMM-FORK-BASELINE01-CORRECTION21",
		"parent_status": "OPEN",
		"parent_disposition": "production bundle absent",
		"overall_status": "fail",
		"overall_disposition": "child closed; parent open",
		"execution_head_oid": "4bd2c5646f7a4d9c8e1f0123456789abcdef0123",
		"execution_tree_oid": "984aaf36abc12345def67890abcdef0123456789",
		"subject_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "strict_typecheck",
				"scope": "MICROC3",
				"status": "pass",
				"evidence": "factory/scripts/tsconfig.json",
				"detail": "strict TypeScript check passed",
				"extras": {
					"argv": ["bunx", "tsc", "--project", "factory/scripts/tsconfig.json"],
					"exit_code": 0,
					"duration_ms": 1287,
					"stdout_sha256": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				},
				"total": 177,
				"pass_count": 177,
				"fail_count": 0,
				"skip_count": 0,
				"unavailable_count": 0
			},
			{
				"name": "parent_production_bundle",
				"scope": "ACT-CLINEMM-FORK-BASELINE01-CORRECTION21",
				"status": "fail",
				"evidence": "parent production bundle",
				"detail": "required parent production bundle is absent",
				"extras": {
					"argv": [],
					"exit_code": null,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`)
	dec := Decode(strings.NewReader(string(data)))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if !norm.Success() {
		t.Fatalf("normalize failed: %v", norm.Diagnostics)
	}
	if norm.Summary.Scope == nil {
		t.Fatal("scope is nil")
	}
	if got := string(norm.Summary.Scope.Status); got != "closed" {
		t.Fatalf("scope_status = %q, want %q", got, "closed")
	}
	if norm.Summary.Parent == nil {
		t.Fatal("parent is nil")
	}
	if got := string(norm.Summary.Parent.Status); got != "open" {
		t.Fatalf("parent_status = %q, want %q", got, "open")
	}
	if got := string(norm.Summary.Overall.Status); got != "fail" {
		t.Fatalf("overall_status = %q, want %q", got, "fail")
	}
	if norm.Summary.Scope.Disposition != "µC-3 reader authority closed" {
		t.Fatalf("scope disposition not preserved: %q", norm.Summary.Scope.Disposition)
	}
	if norm.Summary.Parent.Disposition != "production bundle absent" {
		t.Fatalf("parent disposition not preserved: %q", norm.Summary.Parent.Disposition)
	}
	if norm.Summary.Parent.Act != "ACT-CLINEMM-FORK-BASELINE01-CORRECTION21" {
		t.Fatalf("parent act not preserved: %q", norm.Summary.Parent.Act)
	}
	if norm.Summary.Overall.Disposition == nil ||
		*norm.Summary.Overall.Disposition != "child closed; parent open" {
		t.Fatalf("overall disposition not preserved: %v", norm.Summary.Overall.Disposition)
	}
}
