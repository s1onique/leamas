package closure

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestThreeLayerTimeoutParity proves the round-tripped schema
// and the runtime composed validator agree on every literal
// timeout value in the exact matrix, including the absent case.
//
//   ACCEPT 1, 1.0, 1e0, 599, 600, 600.0, 6e2
//   REJECT every value below 1, every value above 600,
//          every nonintegral number, every non-number,
//          null, absent
func TestThreeLayerTimeoutParity(t *testing.T) {
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema(): %v", err)
	}
	raw, _ := json.Marshal(schema)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var roundTripped map[string]any
	if err := dec.Decode(&roundTripped); err != nil {
		t.Fatalf("decode: %v", err)
	}

	type tc struct {
		name     string
		timeout  string
		present  bool
		wantStd  bool
		wantExt  bool
		wantRt   bool
	}
	cases := []tc{
		{"0_absent", "", false, true, false, false}, // standard mode does not require via x-applicability
		{"null", "null", true, false, false, false},
		{"0", "0", true, false, false, false},
		{"1", "1", true, true, true, true},
		{"1.0", "1.0", true, true, true, true}, // CORRECTION11: structural stage accepts 1.0
		{"1e0", "1e0", true, true, true, true},
		{"1.5", "1.5", true, false, false, false},
		{"1e-1", "1e-1", true, false, false, false},
		{"599", "599", true, true, true, true},
		{"600", "600", true, true, true, true},
		{"600.0", "600.0", true, true, true, true}, // CORRECTION11
		{"6e2", "6e2", true, true, true, true}, // CORRECTION11
		{"601", "601", true, false, false, false},
		{"1e1000", "1e1000", true, false, false, false},
		{"quoted", "\"60\"", true, false, false, false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			timeoutLine := ""
			if c.present {
				timeoutLine = `"timeout_seconds": ` + c.timeout + `,`
			}
			body := `{
				"contract_version": 1,
				"act_id": "ACT-LEAMAS-THREE-LAYER",
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
					` + timeoutLine + `
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
			dec := json.NewDecoder(bytes.NewReader([]byte(body)))
			dec.UseNumber()
			var rootMap map[string]any
			if err := dec.Decode(&rootMap); err != nil {
				t.Fatalf("decode: %v", err)
			}
			std := evaluateWithSchemaStandard(roundTripped, rootMap)
			ext := evaluateWithSchemaExtensionAware(roundTripped, rootMap)
			composed := ValidatePlanComposed([]byte(body))
			if std.Accept != c.wantStd {
				t.Fatalf("standard=%v want %v (issues=%v)", std.Accept, c.wantStd, std.Issues)
			}
			if ext.Accept != c.wantExt {
				t.Fatalf("extension=%v want %v (issues=%v)", ext.Accept, c.wantExt, ext.Issues)
			}
			if composed.Valid != c.wantRt {
				t.Fatalf("runtime=%v want %v", composed.Valid, c.wantRt)
			}
		})
	}
}
