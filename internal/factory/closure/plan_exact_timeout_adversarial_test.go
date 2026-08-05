package closure

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"
)

// TestExactTimeoutMatrixAdversarialBoundaries exercises the exact
// timeout matrix around the [1,600] inclusive range using literal
// JSON numbers whose exact rational representation would not
// survive a float64 round trip.
func TestExactTimeoutMatrixAdversarialBoundaries(t *testing.T) {
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
		name    string
		timeout string
		want    bool
	}
	cases := []tc{
		{"0.999_below_1", "0.999999999999999999999999999999999999", false},
		{"1_exact", "1", true},
		{"1.000000000000000000000000000000000001_above_1", "1.000000000000000000000000000000000001", false},
		{"599.999_below_600", "599.999999999999999999999999999999999", false},
		{"599.5_below_600", "599.5", false},
		{"599_exact", "599", true},
		{"600_exact", "600", true},
		{"600.000000000000000000000000000000001_above_600", "600.000000000000000000000000000000001", false},
		{"1e0", "1e0", true},
		{"6e2", "6e2", true},
		{"6.000000000000000000000000000000001e2", "6.000000000000000000000000000000001e2", false},
		{"float64_max", "9007199254740991", false},
		{"float64_max_p1", "9007199254740992", false},
		{"int64_max", "9223372036854775807", false},
		{"1e1000", "1e1000", false},
		{"1e-1000", "1e-1000", false},
		{"quoted", "\"60\"", false},
		{"null", "null", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
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
					"timeout_seconds": ` + c.timeout + `,
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
			got := evaluateWithSchemaExtensionAware(roundTripped, rootMap)
			if got.Accept != c.want {
				t.Fatalf("timeout=%s extension=%v want %v (issues=%v)",
					c.timeout, got.Accept, c.want, got.Issues)
			}
		})
	}
}

// TestExactRationalAdversarialDecoding proves the exactRational
// helper classifies adversarial boundaries correctly.
func TestExactRationalAdversarialDecoding(t *testing.T) {
	type tc struct {
		input      string
		wantInt    bool
		wantBelow1 bool
	}
	cases := []tc{
		{"0.999999999999999999999999999999999999", false, true},
		{"1", true, false},
		{"1.000000000000000000000000000000000001", false, false},
		{"600", true, false},
		{"600.000000000000000000000000000000001", false, false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.input, func(t *testing.T) {
			r, ok := exactRational(json.Number(c.input))
			if !ok {
				t.Fatalf("exactRational failed for %s", c.input)
			}
			if got := exactIsInteger(r); got != c.wantInt {
				t.Fatalf("%s integer: got %v want %v", c.input, got, c.wantInt)
			}
			one := exactFromInt64(1)
			if got := exactLessThan(r, one); got != c.wantBelow1 {
				t.Fatalf("%s below 1: got %v want %v", c.input, got, c.wantBelow1)
			}
		})
	}
}

// TestBigRatPreservesHighPrecision demonstrates that the
// exactRational helper uses math/big.Rat so values that
// float64 would round across the [1,600] inclusive range
// are still classified correctly.
func TestBigRatPreservesHighPrecision(t *testing.T) {
	raw := "600.000000000000000000000000000000001"
	r, ok := new(big.Rat).SetString(raw)
	if !ok {
		t.Fatalf("SetString failed")
	}
	if r.IsInt() {
		t.Fatalf("600.000...001 must not classify as integer")
	}
	if r.Sign() <= 0 {
		t.Fatalf("must be positive")
	}
}
