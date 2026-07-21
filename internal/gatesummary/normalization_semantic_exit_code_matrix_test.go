package gatesummary

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

// exitCodeMatrixCase is one row of the exit-code semantic matrix.
// Wire is the constructed JSON wire fragment for a single check.
// WantDiagnostics, WantExitPresent, WantExitText carry the
// expected contract outcome.
type exitCodeMatrixCase struct {
	ID              string
	Wire            string
	WantDiagnostics []diagnosticProjection
	WantExitPresent bool
	WantExitText    string
}

// exitCodeSemanticMatrix is the frozen exit-code matrix.
// All rows decode successfully (the JSON is valid); the rows that
// fail normalization assert exact diagnostic code/path/ordering.
// Wire failure and semantic failure remain distinct.
//
// Rows where the recorded overall_status already matches the
// derived check-status (e.g., recorded=fail with one fail check)
// produce only the per-check exit-code diagnostic. Rows where
// recorded overall diverges from the derived check status produce
// both the per-check diagnostic and a GS_OVERALL_STATUS_MISMATCH,
// ordered by (precedence, path, encounter).
var exitCodeSemanticMatrix = []exitCodeMatrixCase{
	// pass with exit_code 0 and overall=pass → success
	{
		ID:              "EX-001",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("p", "pass", "0"), "pass"),
		WantExitPresent: true,
		WantExitText:    "0",
	},
	// pass with exit_code 1; overall=fail — both per-check and
	// overall mismatch diagnostics fire (pass→fail derivation).
	{
		ID: "EX-002",
		Wire: exitCodeMatrixBody(
			checkJSONForMatrix("p1", "pass", "1"), "fail"),
		WantDiagnostics: []diagnosticProjection{
			{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
			{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
		},
		WantExitPresent: true,
		WantExitText:    "1",
	},
	// pass with exit_code -1; overall=fail
	{
		ID: "EX-003",
		Wire: exitCodeMatrixBody(
			checkJSONForMatrix("pm", "pass", "-1"), "fail"),
		WantDiagnostics: []diagnosticProjection{
			{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
			{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
		},
		WantExitPresent: true,
		WantExitText:    "-1",
	},
	// pass with exit_code null; overall=fail
	{
		ID: "EX-004",
		Wire: exitCodeMatrixBody(
			checkJSONForMatrix("pnull", "pass", "null"), "fail"),
		WantDiagnostics: []diagnosticProjection{
			{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
			{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
		},
		WantExitPresent: false,
	},

	// fail with exit_code 1; overall=fail → success
	{
		ID:              "EX-005",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("f1", "fail", "1"), "fail"),
		WantExitPresent: true,
		WantExitText:    "1",
	},
	// fail with exit_code 2; overall=fail → success
	{
		ID:              "EX-006",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("f2", "fail", "2"), "fail"),
		WantExitPresent: true,
		WantExitText:    "2",
	},
	// fail with exit_code null; overall=fail → success
	{
		ID:              "EX-007",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("fnull", "fail", "null"), "fail"),
		WantExitPresent: false,
	},
	// fail with exit_code 0; overall=fail → GS_FAIL_EXIT_CODE_MISMATCH only
	{
		ID: "EX-008",
		Wire: exitCodeMatrixBody(
			checkJSONForMatrix("f0", "fail", "0"), "fail"),
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeFailExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		},
		WantExitPresent: true,
		WantExitText:    "0",
	},

	// skip with exit_code null; overall=unavailable → success
	{
		ID:              "EX-009",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("sn", "skip", "null"), "unavailable"),
		WantExitPresent: false,
	},
	// skip with exit_code 0; overall=unavailable → GS_SKIP_EXIT_CODE_MISMATCH only
	{
		ID: "EX-010",
		Wire: exitCodeMatrixBody(
			checkJSONForMatrix("s0", "skip", "0"), "unavailable"),
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeSkipExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		},
		WantExitPresent: true,
		WantExitText:    "0",
	},

	// unavailable with exit_code null; overall=unavailable → success
	{
		ID:              "EX-011",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("un", "unavailable", "null"), "unavailable"),
		WantExitPresent: false,
	},
	// unavailable with exit_code 1; overall=unavailable → diagnostic only
	{
		ID: "EX-012",
		Wire: exitCodeMatrixBody(
			checkJSONForMatrix("u0", "unavailable", "1"), "unavailable"),
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeUnavailExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		},
		WantExitPresent: true,
		WantExitText:    "1",
	},

	// pass with huge non-zero exit_code; overall=fail → both
	{
		ID: "EX-013",
		Wire: exitCodeMatrixBody(
			checkJSONForMatrix("huge", "pass", largePositiveRaw), "fail"),
		WantDiagnostics: []diagnosticProjection{
			{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
			{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
		},
		WantExitPresent: true,
		WantExitText:    largePositiveRaw,
	},
	// fail with huge positive exit_code; overall=fail → success
	{
		ID:              "EX-014",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("fhuge", "fail", largePositiveRaw), "fail"),
		WantExitPresent: true,
		WantExitText:    largePositiveRaw,
	},
	// fail with huge negative exit_code; overall=fail → success
	{
		ID:              "EX-015",
		Wire:            exitCodeMatrixBody(checkJSONForMatrix("fhugeN", "fail", largeNegativeRaw), "fail"),
		WantExitPresent: true,
		WantExitText:    largeNegativeRaw,
	},
}

// TestSemanticExitCodeMatrix walks every exit-code matrix row.
// Wire-failure rows are NOT in this matrix — they belong to
// the corpus. Wire syntax failure is distinct from semantic
// status/exit-code mismatch.
func TestSemanticExitCodeMatrix(t *testing.T) {
	if got := len(exitCodeSemanticMatrix); got != 15 {
		t.Fatalf("exit-code matrix has %d rows, want 15", got)
	}
	for _, c := range exitCodeSemanticMatrix {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			dec := Decode(strings.NewReader(c.Wire))
			if !dec.Success() {
				t.Fatalf("%s: decode failed unexpectedly: %v", c.ID, dec.Diagnostics)
			}
			norm := Normalize(dec.Document)
			got := projectDiagnostics(norm.Diagnostics)
			if !compareProjections(got, c.WantDiagnostics) {
				t.Fatalf("%s: diagnostics = %#v, want %#v",
					c.ID, got, c.WantDiagnostics)
			}
			// On rejected rows, the normalized Summary is the
			// zero value (no checks published). Only inspect
			// normalized fields on successful rows.
			if len(c.WantDiagnostics) == 0 {
				if len(norm.Summary.Checks) != 1 {
					t.Fatalf("%s: expected 1 normalized check, got %d",
						c.ID, len(norm.Summary.Checks))
				}
				ec := norm.Summary.Checks[0].Execution
				if ec == nil {
					t.Fatalf("%s: missing Execution on normalized check", c.ID)
				}
				if (ec.ExitCode != nil) != c.WantExitPresent {
					t.Fatalf("%s: exit-code presence = %v, want %v",
						c.ID, ec.ExitCode != nil, c.WantExitPresent)
				}
				if ec.ExitCode != nil && ec.ExitCode.String() != c.WantExitText {
					t.Fatalf("%s: exit-code text = %q, want %q",
						c.ID, ec.ExitCode.String(), c.WantExitText)
				}
			}
		})
	}
}

// TestExitCodeArbitraryPrecisionExact exercises pass + huge and
// fail + huge with arbitrary precision. The asserted exit-code
// string preserves the exact wire spelling. The normalized
// Summary is the zero value on reject; we inspect the diagnostic
// alone.
func TestExitCodeArbitraryPrecisionExact(t *testing.T) {
	wire := exitCodeMatrixBody(
		checkJSONForMatrix("big", "pass", largePositiveRaw), "fail")
	dec := Decode(strings.NewReader(wire))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if norm.Success() {
		t.Fatal("expected normalize failure")
	}
	var passDiag *Diagnostic
	for i := range norm.Diagnostics {
		if norm.Diagnostics[i].Code == CodePassExitCodeMismatch {
			passDiag = &norm.Diagnostics[i]
			break
		}
	}
	if passDiag == nil {
		t.Fatalf("missing GS_PASS_EXIT_CODE_MISMATCH diagnostic: %v",
			norm.Diagnostics)
	}
	if passDiag.Path != "/checks/0/extras/exit_code" {
		t.Fatalf("path = %q, want %q", passDiag.Path, "/checks/0/extras/exit_code")
	}
	// Exercise the arbitrary-precision integer contract directly.
	i, err := newIntegerFromWire(wireIntegerForTest(largePositiveRaw))
	if err != nil {
		t.Fatalf("newIntegerFromWire failed: %v", err)
	}
	bi, ok := i.BigInt()
	if !ok {
		t.Fatal("BigInt failed")
	}
	want := new(big.Int)
	want.SetString(largePositiveRaw, 10)
	if bi.Cmp(want) != 0 {
		t.Fatalf("BigInt = %s, want %s", bi.String(), want.String())
	}
	if i.String() != largePositiveRaw {
		t.Fatalf("String() = %q, want %q", i.String(), largePositiveRaw)
	}
	if _, ok := i.Int64(); ok {
		t.Fatal("Int64 should fail for value beyond math.MaxInt64")
	}
	if bi.Cmp(big.NewInt(math.MaxInt64)) <= 0 {
		t.Fatal("test value should exceed math.MaxInt64")
	}
}

// TestExitCodeWireFailureNotInMatrix asserts that malformed wire
// exit_code values are rejected at decode time, not normalize time.
func TestExitCodeWireFailureNotInMatrix(t *testing.T) {
	wire := exitCodeMatrixBody(
		checkJSONForMatrix("bad", "pass", `"abc"`), "pass")
	dec := Decode(strings.NewReader(wire))
	if dec.Success() {
		t.Fatalf("malformed wire exit_code should reject at decode")
	}
	if len(dec.Diagnostics) == 0 {
		t.Fatal("decode rejection produced zero diagnostics")
	}
}
