package closure

import (
	"bytes"
	"encoding/json"
	"github.com/s1onique/leamas/internal/factory/closure/evaltest"
	"testing"
)

// TestPinnedRunMatrixRoundTripped proves the pinned run-presence
// parity survives JSON marshal + generic decode. The authoritative
// schema authority is the public CLI-emitted bytes, not the direct
// in-memory map from JSONSchema().
func TestPinnedRunMatrixRoundTripped(t *testing.T) {
	inMem, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema(): %v", err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(inMem); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	dec.UseNumber()
	var roundTripped map[string]any
	if err := dec.Decode(&roundTripped); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Now evaluate against the round-tripped schema.
	type result struct{ standard, extension, runtime bool }
	cases := []struct {
		name string
		ops  []string
		want result
	}{
		{"both_present", nil, result{true, true, true}},
		{"wd_absent", []string{"delete:working_directory"}, result{true, false, false}},
		{"ts_absent", []string{"delete:timeout_seconds"}, result{true, false, false}},
		{"both_absent", []string{"delete:working_directory", "delete:timeout_seconds"}, result{true, false, false}},
		{"wd_null", []string{"set:working_directory=null"}, result{false, false, false}},
		{"ts_null", []string{"set:timeout_seconds=null"}, result{false, false, false}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(canonicalRunPlanPinned)
			for _, op := range tc.ops {
				data = applyPinnedMutation(t, data, op)
			}
			var rootMap map[string]any
			if err := json.Unmarshal(data, &rootMap); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			standard := evaltest.EvaluateWithSchemaStandard(roundTripped, rootMap)
			extension := evaltest.EvaluateWithSchemaExtensionAware(roundTripped, rootMap)
			composed := ValidatePlanComposed(data)
			if standard.Accept != tc.want.standard {
				t.Fatalf("standard=%v want %v", standard.Accept, tc.want.standard)
			}
			if extension.Accept != tc.want.extension {
				t.Fatalf("extension=%v want %v", extension.Accept, tc.want.extension)
			}
			if composed.Valid != tc.want.runtime {
				t.Fatalf("runtime=%v want %v", composed.Valid, tc.want.runtime)
			}
		})
	}
}
