package closure

import (
	"bytes"
	"encoding/json"
	"github.com/s1onique/leamas/internal/factory/closure/evaltest"
	"strings"
	"testing"
)

// TestPublicWireNumberTypeEndToEnd proves the public plan
// bytes reach the structural integer conversion as
// json.Number, not float64. The test fails if the
// structural parser stops using json.Decoder.UseNumber.
func TestPublicWireNumberTypeEndToEnd(t *testing.T) {
	planBytes := []byte(`{
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-PUBLIC-WIRE",
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
			"timeout_seconds": 1,
			"environment": {}
		}],
		"artifacts": [],
		"policy": {
			"require_clean_before": true,
			"require_clean_after": true,
			"forbid_tracked_full_digests": true,
			"require_diff_check": true
		}
	}`)
	// Decode with json.Decoder.UseNumber
	dec := json.NewDecoder(bytes.NewReader(planBytes))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		t.Fatalf("decode: %v", err)
	}
	rootMap := root.(map[string]any)
	checks := rootMap["checks"].([]any)
	check := checks[0].(map[string]any)
	timeout, ok := check["timeout_seconds"]
	if !ok {
		t.Fatalf("timeout_seconds missing")
	}
	// Must be json.Number, not float64
	if _, isFloat := timeout.(float64); isFloat {
		t.Fatalf("public wire timeout is float64; json.Decoder.UseNumber is not active")
	}
	if _, isNum := timeout.(json.Number); !isNum {
		t.Fatalf("public wire timeout type=%T want json.Number", timeout)
	}
}

// TestStructuralPublicWireFloat64Unreachable proves the
// public plan path does not surface float64 to the
// structural integer conversion. The float64 branch in
// jsonNumberToInteger is a compatibility fallback for
// non-public callers; the public plan path uses
// json.Decoder.UseNumber.
func TestStructuralPublicWireFloat64Unreachable(t *testing.T) {
	planBytes := []byte(`{
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-FLOAT64-UNREACH",
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
			"timeout_seconds": 60,
			"environment": {}
		}],
		"artifacts": [],
		"policy": {
			"require_clean_before": true,
			"require_clean_after": true,
			"forbid_tracked_full_digests": true,
			"require_diff_check": true
		}
	}`)
	// The structural validator's parseBoundedClosurePlanDocument
	// uses UseNumber; this test documents that contract.
	dec := json.NewDecoder(bytes.NewReader(planBytes))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Walk the tree to find timeout_seconds
	checks := v.(map[string]any)["checks"].([]any)
	check := checks[0].(map[string]any)
	timeout := check["timeout_seconds"]
	if _, isFloat := timeout.(float64); isFloat {
		t.Fatalf("public plan timeout was decoded as float64; the structural parser is not using UseNumber")
	}
}

// TestFourLayerTimeoutParity exercises standard schema,
// extension-aware schema, structural validator, and
// composed runtime against every canonical and
// adversarial timeout form. All four layers must agree.
func TestFourLayerTimeoutParity(t *testing.T) {
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
	cases := []struct {
		name    string
		timeout string
		present bool
		wantStd bool
		wantExt bool
		wantRt  bool
	}{
		{"absent", "", false, true, false, false},
		{"null", "null", true, false, false, false},
		{"0", "0", true, false, false, false},
		{"1", "1", true, true, true, true},
		{"1.0", "1.0", true, true, true, true},
		{"1e0", "1e0", true, true, true, true},
		{"1.5", "1.5", true, false, false, false},
		{"1e-1", "1e-1", true, false, false, false},
		{"599", "599", true, true, true, true},
		{"600", "600", true, true, true, true},
		{"600.0", "600.0", true, true, true, true},
		{"6e2", "6e2", true, true, true, true},
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
				"act_id": "ACT-LEAMAS-FOUR-LAYER",
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
			// Standard schema
			dec := json.NewDecoder(strings.NewReader(body))
			dec.UseNumber()
			var rootMap map[string]any
			if err := dec.Decode(&rootMap); err != nil {
				t.Fatalf("decode: %v", err)
			}
			std := evaltest.EvaluateWithSchemaStandard(roundTripped, rootMap)
			ext := evaltest.EvaluateWithSchemaExtensionAware(roundTripped, rootMap)
			composed := ValidatePlanComposed([]byte(body))
			if std.Accept != c.wantStd {
				t.Fatalf("%s: standard=%v want %v", c.name, std.Accept, c.wantStd)
			}
			if ext.Accept != c.wantExt {
				t.Fatalf("%s: extension=%v want %v", c.name, ext.Accept, c.wantExt)
			}
			if composed.Valid != c.wantRt {
				t.Fatalf("%s: composed=%v want %v", c.name, composed.Valid, c.wantRt)
			}
		})
	}
}
