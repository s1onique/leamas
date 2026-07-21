package gatesummary

import (
	"reflect"
	"strings"
	"testing"
)

// totalsMatrixCase is one row of the totals matrix.
type totalsMatrixCase struct {
	ID              string
	Total           *string
	Pass            *string
	Fail            *string
	Skip            *string
	Unavail         *string
	WantDecodeOK    bool
	WantNormOK      bool
	WantDiagnostics []diagnosticProjection
}

// totalsMatrixBody builds a v2 document with a single pass check
// carrying the requested totals wire values. overall_status is
// "pass" because the only check is pass with a valid exit_code.
func totalsMatrixBody(total, pass, fail, skip, unavail *string) string {
	tWire := "null"
	if total != nil {
		tWire = *total
	}
	pWire := "null"
	if pass != nil {
		pWire = *pass
	}
	fWire := "null"
	if fail != nil {
		fWire = *fail
	}
	sWire := "null"
	if skip != nil {
		sWire = *skip
	}
	uWire := "null"
	if unavail != nil {
		uWire = *unavail
	}
	return `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-TOTALS-MATRIX",
		"scope_status": "OPEN",
		"scope_disposition": "totals matrix",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "pass",
		"overall_disposition": "totals matrix",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "t",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				},
				"total": ` + tWire + `,
				"pass_count": ` + pWire + `,
				"fail_count": ` + fWire + `,
				"skip_count": ` + sWire + `,
				"unavailable_count": ` + uWire + `
			}
		]
	}`
}

func sptr(v string) *string { return &v }

// totalsSemanticMatrix is the frozen totals matrix. Zero, single,
// mixed, mismatched, partial, oversized, and beyond-int64 cases
// are all covered. Multiple simultaneous mismatches appear as a
// single GS_TEST_TOTAL_MISMATCH per check, frozen by precedence.
var totalsSemanticMatrix = []totalsMatrixCase{
	// zero totals — total=0, components all 0
	{
		ID:           "TT-001",
		Total:        sptr("0"),
		Pass:         sptr("0"),
		Fail:         sptr("0"),
		Skip:         sptr("0"),
		Unavail:      sptr("0"),
		WantDecodeOK: true,
		WantNormOK:   true,
	},
	// single pass: total=1, pass=1, others 0
	{
		ID:           "TT-002",
		Total:        sptr("1"),
		Pass:         sptr("1"),
		Fail:         sptr("0"),
		Skip:         sptr("0"),
		Unavail:      sptr("0"),
		WantDecodeOK: true,
		WantNormOK:   true,
	},
	// single fail: total=3, fail=3
	{
		ID:           "TT-003",
		Total:        sptr("3"),
		Pass:         sptr("0"),
		Fail:         sptr("3"),
		Skip:         sptr("0"),
		Unavail:      sptr("0"),
		WantDecodeOK: true,
		WantNormOK:   true,
	},
	// single skip
	{
		ID:           "TT-004",
		Total:        sptr("5"),
		Pass:         sptr("0"),
		Fail:         sptr("0"),
		Skip:         sptr("5"),
		Unavail:      sptr("0"),
		WantDecodeOK: true,
		WantNormOK:   true,
	},
	// single unavailable
	{
		ID:           "TT-005",
		Total:        sptr("7"),
		Pass:         sptr("0"),
		Fail:         sptr("0"),
		Skip:         sptr("0"),
		Unavail:      sptr("7"),
		WantDecodeOK: true,
		WantNormOK:   true,
	},
	// mixed
	{
		ID:           "TT-006",
		Total:        sptr("10"),
		Pass:         sptr("4"),
		Fail:         sptr("3"),
		Skip:         sptr("2"),
		Unavail:      sptr("1"),
		WantDecodeOK: true,
		WantNormOK:   true,
	},
	// total smaller than sum
	{
		ID:           "TT-007",
		Total:        sptr("9"),
		Pass:         sptr("4"),
		Fail:         sptr("3"),
		Skip:         sptr("2"),
		Unavail:      sptr("1"),
		WantDecodeOK: true,
		WantNormOK:   false,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		},
	},
	// total larger than sum
	{
		ID:           "TT-008",
		Total:        sptr("11"),
		Pass:         sptr("4"),
		Fail:         sptr("3"),
		Skip:         sptr("2"),
		Unavail:      sptr("1"),
		WantDecodeOK: true,
		WantNormOK:   false,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		},
	},
	// multiple simultaneous mismatches still produce one diagnostic
	{
		ID:           "TT-009",
		Total:        sptr("99"),
		Pass:         sptr("1"),
		Fail:         sptr("1"),
		Skip:         sptr("1"),
		Unavail:      sptr("1"),
		WantDecodeOK: true,
		WantNormOK:   false,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		},
	},
	// partial totals (only total, no components) — schema rejects
	{
		ID:           "TT-010-partial-decode-only",
		Total:        sptr("1"),
		WantDecodeOK: false,
	},
	// arbitrary-precision total above MaxInt64
	{
		ID:           "TT-011",
		Total:        sptr(largePositiveRaw),
		Pass:         sptr(largePositiveRaw),
		Fail:         sptr("0"),
		Skip:         sptr("0"),
		Unavail:      sptr("0"),
		WantDecodeOK: true,
		WantNormOK:   true,
	},
	// arbitrary-precision above MaxInt64 with mismatch
	{
		ID:           "TT-012",
		Total:        sptr(largePositiveRaw),
		Pass:         sptr("1"),
		Fail:         sptr("0"),
		Skip:         sptr("0"),
		Unavail:      sptr("0"),
		WantDecodeOK: true,
		WantNormOK:   false,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		},
	},
	// forbidden negative total — schema rejects
	{
		ID:           "TT-013-negative-decode-only",
		Total:        sptr("-1"),
		Pass:         sptr("0"),
		Fail:         sptr("0"),
		Skip:         sptr("0"),
		Unavail:      sptr("0"),
		WantDecodeOK: false,
	},
}

// TestSemanticTotalsMatrix walks every totals matrix row.
func TestSemanticTotalsMatrix(t *testing.T) {
	if got := len(totalsSemanticMatrix); got != 13 {
		t.Fatalf("totals matrix has %d rows, want 13", got)
	}
	for _, c := range totalsSemanticMatrix {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			var wire string
			if c.ID == "TT-010-partial-decode-only" {
				wire = totalsMatrixBody(sptr("1"), nil, nil, nil, nil)
			} else {
				wire = totalsMatrixBody(c.Total, c.Pass, c.Fail, c.Skip, c.Unavail)
			}
			dec := Decode(strings.NewReader(wire))
			if dec.Success() != c.WantDecodeOK {
				t.Fatalf("%s: decode success = %v, want %v (diags=%v)",
					c.ID, dec.Success(), c.WantDecodeOK, dec.Diagnostics)
			}
			if !c.WantDecodeOK {
				return
			}
			norm := Normalize(dec.Document)
			if norm.Success() != c.WantNormOK {
				t.Fatalf("%s: normalize success = %v, want %v (diags=%v)",
					c.ID, norm.Success(), c.WantNormOK, norm.Diagnostics)
			}
			got := projectDiagnostics(norm.Diagnostics)
			if !compareProjections(got, c.WantDiagnostics) {
				t.Fatalf("%s: diagnostics = %#v, want %#v",
					c.ID, got, c.WantDiagnostics)
			}
			if c.WantNormOK && len(norm.Summary.Checks) == 1 {
				tot := norm.Summary.Checks[0].Totals
				if tot == nil {
					t.Fatalf("%s: missing normalized totals", c.ID)
				}
				wantTotal := ""
				if c.Total != nil {
					wantTotal = *c.Total
				}
				if tot.Total.String() != wantTotal {
					t.Fatalf("%s: total = %q, want %q",
						c.ID, tot.Total.String(), wantTotal)
				}
			}
		})
	}
}

// TestTotalsDeterministicOrdering verifies that two checks with
// independent totals mismatches produce deterministic ordered
// diagnostics, with all per-check totals mismatches preceding any
// overall-status mismatch.
func TestTotalsDeterministicOrdering(t *testing.T) {
	wire := `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-ORDER",
		"scope_status": "OPEN",
		"scope_disposition": "ordering",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "pass",
		"overall_disposition": "ordering",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "a",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				},
				"total": 100,
				"pass_count": 1,
				"fail_count": 0,
				"skip_count": 0,
				"unavailable_count": 0
			},
			{
				"name": "b",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				},
				"total": 200,
				"pass_count": 1,
				"fail_count": 0,
				"skip_count": 0,
				"unavailable_count": 0
			}
		]
	}`
	dec := Decode(strings.NewReader(wire))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if norm.Success() {
		t.Fatal("normalize unexpectedly succeeded")
	}
	got := projectDiagnostics(norm.Diagnostics)
	want := []diagnosticProjection{
		{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		{Code: CodeTestTotalMismatch, Path: "/checks/1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering = %#v, want %#v", got, want)
	}
}
