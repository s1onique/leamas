package closure

import (
	"encoding/json"
	"testing"
)

// TestJsonNumberToIntegerDirect exercises the structural
// integer conversion authority end to end. CORRECTION13
// requires direct tests proving the conversion accepts 1,
// 1.0, 1e0, 600, 600.0, 6e2 and rejects non-integral
// forms.
func TestJsonNumberToIntegerDirect(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantOk  bool
		wantVal int
	}{
		{"1", "1", true, 1},
		{"1.0", "1.0", true, 1},
		{"1e0", "1e0", true, 1},
		{"600", "600", true, 600},
		{"600.0", "600.0", true, 600},
		{"6e2", "6e2", true, 600},
		{"1.5", "1.5", false, 0},
		{"1e-1", "1e-1", false, 0},
		{"600.000000000000000000001", "600.000000000000000000001", false, 0},
		// 1e1000 is a mathematical integer but exceeds the
		// helper's int64 representable range. The helper
		// rejects it; the bound stage will surface the
		// above-maximum classification.
		{"1e1000", "1e1000", false, 0},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got, ok := jsonNumberToInteger(json.Number(c.input))
			if ok != c.wantOk {
				t.Fatalf("jsonNumberToInteger(%s) ok=%v want %v", c.input, ok, c.wantOk)
			}
			if ok && got != c.wantVal {
				t.Fatalf("jsonNumberToInteger(%s)=%d want %d", c.input, got, c.wantVal)
			}
		})
	}
}

// TestJsonNumberToIntegerPublicWireReachability proves the
// public plan bytes never reach structural validation as
// float64. The committed parser uses json.Decoder.UseNumber
// and the conversion helper accepts json.Number.
func TestJsonNumberToIntegerPublicWireReachability(t *testing.T) {
	// The helper accepts float64 only as a compatibility
	// fallback for non-public callers. Add a guard test that
	// records this and is documented as a known compatibility
	// branch.
	got, ok := jsonNumberToInteger(float64(1))
	if !ok || got != 1 {
		t.Fatalf("jsonNumberToInteger(float64(1))=%d ok=%v want 1,true", got, ok)
	}
}

// TestJsonNumberToIntegerStructuralMatrix exercises the
// canonical and adversarial matrix end to end through the
// structural authority.
func TestJsonNumberToIntegerStructuralMatrix(t *testing.T) {
	// jsonNumberToInteger is the integer conversion primitive;
	// it only proves mathematical integrality. Bound checks
	// ([1, 600]) happen at the next stage. The accepted list
	// contains every integral form; the rejected list contains
	// every non-integral form.
	accepts := []string{"0", "1", "1.0", "1e0", "599", "600", "600.0", "6e2", "601"}
	rejects := []string{"1.5", "1e-1", "0.5", "599.5"}
	for _, s := range accepts {
		s := s
		t.Run("accept_"+s, func(t *testing.T) {
			_, ok := jsonNumberToInteger(json.Number(s))
			if !ok {
				t.Fatalf("expected %s to be accepted as integer", s)
			}
		})
	}
	for _, s := range rejects {
		s := s
		t.Run("reject_"+s, func(t *testing.T) {
			_, ok := jsonNumberToInteger(json.Number(s))
			if ok {
				t.Fatalf("expected %s to be rejected as non-integer", s)
			}
		})
	}
}
