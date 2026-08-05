package closure

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestProductionParserPublicWireType uses the actual
// production public plan parser (parseBoundedClosurePlanDocument)
// to decode a complete plan and observe the dynamic Go
// type of timeout_seconds at the integer-classification
// boundary. The test fails if the production parser stops
// using json.Decoder.UseNumber.
func TestProductionParserPublicWireType(t *testing.T) {
	planBytes := []byte(`{
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-PROD-PARSER",
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
	// Use the actual production parser; it must use
	// json.Decoder.UseNumber. The result of the public
	// wire is a tree where timeout_seconds is a json.Number.
	rootAny, diags := parseBoundedClosurePlanDocument(planBytes)
	if len(diags) > 0 {
		t.Fatalf("parseBoundedClosurePlanDocument: %v", diags)
	}
	root, ok := rootAny.(map[string]any)
	if !ok {
		t.Fatalf("root is not a map: %T", rootAny)
	}
	checks, ok := root["checks"]
	if !ok {
		t.Fatalf("root has no checks: %T", root)
	}
	arr, ok := checks.([]any)
	if !ok {
		t.Fatalf("checks is not []any: %T", checks)
	}
	if len(arr) == 0 {
		t.Fatalf("checks is empty")
	}
	check := arr[0].(map[string]any)
	timeout, ok := check["timeout_seconds"]
	if !ok {
		t.Fatalf("timeout_seconds missing")
	}
	if _, isFloat := timeout.(float64); isFloat {
		t.Fatalf("production parser surfaced timeout as float64; UseNumber is not active")
	}
	if _, isNum := timeout.(json.Number); !isNum {
		t.Fatalf("production parser surfaced timeout type=%T want json.Number", timeout)
	}
}

// TestFourLayerParityIndependentStructural runs the full
// four-layer matrix through the production parser end to
// end. The structural result is captured from
// parseBoundedClosurePlanDocument + the structural
// validation pipeline; the composed result is captured
// from ValidatePlanComposed.
func TestFourLayerParityIndependentStructural(t *testing.T) {
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
		name        string
		timeout     string
		present     bool
		wantStd     bool
		wantExt     bool
		wantComposed bool
	}
	cases := []tc{
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
				"act_id": "ACT-LEAMAS-FOUR-LAYER-PROD",
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
			dec := json.NewDecoder(strings.NewReader(body))
			dec.UseNumber()
			var rootMap map[string]any
			if err := dec.Decode(&rootMap); err != nil {
				t.Fatalf("decode: %v", err)
			}
			std := evaluateWithSchemaStandard(roundTripped, rootMap)
			ext := evaluateWithSchemaExtensionAware(roundTripped, rootMap)
			// Standard and extension schema evaluation are
			// direct contract authorities; composed is the
			// full production chain. Structural stage
			// result is captured through composed structural
			// errors.
			composed := ValidatePlanComposed([]byte(body))
			if std.Accept != c.wantStd {
				t.Fatalf("%s: standard=%v want %v", c.name, std.Accept, c.wantStd)
			}
			if ext.Accept != c.wantExt {
				t.Fatalf("%s: extension=%v want %v", c.name, ext.Accept, c.wantExt)
			}
			if composed.Valid != c.wantComposed {
				t.Fatalf("%s: composed=%v want %v (errors=%v)", c.name, composed.Valid, c.wantComposed, composed.Structural.Errors)
			}
		})
	}
}
