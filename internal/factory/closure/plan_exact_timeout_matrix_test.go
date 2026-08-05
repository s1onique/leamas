package closure

import (
	"bytes"
	"encoding/json"
	"github.com/s1onique/leamas/internal/factory/closure/evaltest"
	"strings"
	"testing"
)

// TestExactTimeoutMatrixRoundTripped asserts the literal-JSON exact
// timeout matrix from the Closure Protocol v1 contract under
// the actual schema emitted by JSONSchema() (round-tripped through
// marshal + generic decode with UseNumber).
//
//	ACCEPT 1, 1.0, 1e0, 599, 600, 600.0, 6e2
//	REJECT every value below 1, every value above 600,
//	       every nonintegral number, every non-number,
//	       null, absent
func TestExactTimeoutMatrixRoundTripped(t *testing.T) {
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema(): %v", err)
	}
	// Round trip through marshal + UseNumber to ensure the wire
	// shape drives the evaluator.
	raw, _ := json.Marshal(schema)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var roundTripped map[string]any
	if err := dec.Decode(&roundTripped); err != nil {
		t.Fatalf("decode: %v", err)
	}

	type tc struct {
		name    string
		timeout string
		want    bool
	}
	cases := []tc{
		{"below_0_neg_big", "-9223372036854775809", false},
		{"below_0_neg_1", "-1", false},
		{"zero", "0", false},
		{"zero_float", "0.0", false},
		{"lower_bound_1", "1", true},
		{"lower_bound_1.0", "1.0", true},
		{"lower_bound_1e0", "1e0", true},
		{"nonintegral_1.5", "1.5", false},
		{"nonintegral_1e-1", "1e-1", false},
		{"mid_599", "599", true},
		{"upper_600", "600", true},
		{"upper_600.0", "600.0", true},
		{"upper_6e2", "6e2", true},
		{"above_601", "601", false},
		{"above_601.0", "601.0", false},
		{"above_float64_max", "9007199254740991", false},
		{"above_float64_max_p1", "9007199254740992", false},
		{"above_float64_max_p2", "9007199254740993", false},
		{"int64_max", "9223372036854775807", false},
		{"int64_max_p1", "9223372036854775808", false},
		{"extreme_exp", "1e1000", false},
		{"string_quoted", "\"60\"", false},
		{"null", "null", false},
	}
	for _, c := range cases {
		tc := c
		t.Run(tc.name, func(t *testing.T) {
			// Build a minimal plan with timeout_seconds set to the literal JSON.
			body := `{
				"contract_version": 1,
				"act_id": "ACT-LEAMAS-EXACT-TIMEOUT",
				"baseline": {
					"commit_oid": "1111111111111111111111111111111111111111",
					"tree_oid": "2222222222222222222222222222222222222222"
				},
				"execution": {"mode": "serial_fail_fast"},
				"checks": [{
					"id": "x",
					"mode": "run",
					"argv": ["true"],
					"working_directory": ".",
					"timeout_seconds": ` + tc.timeout + `,
					"environment": {}
				}],
				"artifacts": [],
				"policy": {
					"require_clean_before": true,
					"require_clean_after": true,
					"forbid_tracked_full_digests": true,
					"require_diff_check": true
				}
			}`
			if strings.TrimSpace(tc.timeout) == "absent" {
				body = strings.Replace(body, `"timeout_seconds": `+tc.timeout+`,`, ``, 1)
			}
			dec := json.NewDecoder(bytes.NewReader([]byte(body)))
			dec.UseNumber()
			var rootMap map[string]any
			if err := dec.Decode(&rootMap); err != nil {
				t.Fatalf("decode: %v", err)
			}
			got := evaltest.EvaluateWithSchemaExtensionAware(roundTripped, rootMap)
			if got.Accept != tc.want {
				t.Fatalf("timeout_seconds=%s extension=%v want %v (issues=%v)",
					tc.timeout, got.Accept, tc.want, got.Issues)
			}
		})
	}
}
