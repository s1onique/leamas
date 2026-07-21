package gatesummary

import (
	"strings"
	"testing"
)

// cleanlinessMatrixCase is one row of the cleanliness matrix.
type cleanlinessMatrixCase struct {
	ID              string
	ScopeStatus     string
	CleanBefore     bool
	CleanAfter      bool
	WantNormOK      bool
	WantDiagnostics []diagnosticProjection
}

// cleanlinessMatrixBody builds a v2 document with the requested
// cleanliness fields. overall_status=unavailable matches the
// empty-checks derivation so overall mismatch is silent.
func cleanlinessMatrixBody(scope string, before, after bool) string {
	return `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-CLEAN-MATRIX",
		"scope_status": "` + scope + `",
		"scope_disposition": "cleanliness matrix",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "unavailable",
		"overall_disposition": "cleanliness matrix",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": ` + boolJSON(before) + `,
		"worktree_clean_after": ` + boolJSON(after) + `,
		"checks": []
	}`
}

func boolJSON(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// cleanlinessMatrix covers the four clean-before/clean-after
// combinations against the frozen closed-scope invariant, plus
// the negative open/partial-scope cases that must NOT trigger
// cleanliness violations regardless of cleanliness values.
//
// Ordering note: the diagnosticSet sorts by (precedence, path)
// with "/worktree_clean_after" sorting before
// "/worktree_clean_before" alphabetically.
var cleanlinessMatrix = []cleanlinessMatrixCase{
	{
		ID:          "CL-001",
		ScopeStatus: "CLOSED",
		CleanBefore: true,
		CleanAfter:  true,
		WantNormOK:  true,
	},
	{
		ID:          "CL-002",
		ScopeStatus: "CLOSED",
		CleanBefore: false,
		CleanAfter:  true,
		WantNormOK:  false,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeScopeClosedDirtyWorktree, Path: "/worktree_clean_before"},
		},
	},
	{
		ID:          "CL-003",
		ScopeStatus: "CLOSED",
		CleanBefore: true,
		CleanAfter:  false,
		WantNormOK:  false,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeScopeClosedDirtyWorktree, Path: "/worktree_clean_after"},
		},
	},
	{
		ID:          "CL-004",
		ScopeStatus: "CLOSED",
		CleanBefore: false,
		CleanAfter:  false,
		WantNormOK:  false,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeScopeClosedDirtyWorktree, Path: "/worktree_clean_after"},
			{Code: CodeScopeClosedDirtyWorktree, Path: "/worktree_clean_before"},
		},
	},
	// Open scope: cleanliness is independent and not enforced.
	{
		ID:          "CL-005",
		ScopeStatus: "OPEN",
		CleanBefore: true,
		CleanAfter:  true,
		WantNormOK:  true,
	},
	{
		ID:          "CL-006",
		ScopeStatus: "OPEN",
		CleanBefore: false,
		CleanAfter:  true,
		WantNormOK:  true,
	},
	{
		ID:          "CL-007",
		ScopeStatus: "OPEN",
		CleanBefore: true,
		CleanAfter:  false,
		WantNormOK:  true,
	},
	{
		ID:          "CL-008",
		ScopeStatus: "OPEN",
		CleanBefore: false,
		CleanAfter:  false,
		WantNormOK:  true,
	},
	// Partial scope: cleanliness not enforced.
	{
		ID:          "CL-009",
		ScopeStatus: "PARTIAL",
		CleanBefore: false,
		CleanAfter:  false,
		WantNormOK:  true,
	},
}

// TestSemanticCleanlinessMatrix walks every cleanliness matrix row.
// Each cleanliness field is independently validated. One field
// must not silently overwrite another.
func TestSemanticCleanlinessMatrix(t *testing.T) {
	if got := len(cleanlinessMatrix); got != 9 {
		t.Fatalf("cleanliness matrix has %d rows, want 9", got)
	}
	for _, c := range cleanlinessMatrix {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			wire := cleanlinessMatrixBody(c.ScopeStatus, c.CleanBefore, c.CleanAfter)
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
				if norm.Summary.Worktree == nil {
					t.Fatalf("%s: missing normalized worktree", c.ID)
				}
				if norm.Summary.Worktree.CleanBefore != c.CleanBefore {
					t.Fatalf("%s: clean_before = %v, want %v",
						c.ID, norm.Summary.Worktree.CleanBefore, c.CleanBefore)
				}
				if norm.Summary.Worktree.CleanAfter != c.CleanAfter {
					t.Fatalf("%s: clean_after = %v, want %v",
						c.ID, norm.Summary.Worktree.CleanAfter, c.CleanAfter)
				}
			}
		})
	}
}
