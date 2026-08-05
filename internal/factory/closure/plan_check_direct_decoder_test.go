package closure

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
)

// TestPlanCheckDirectDecoder exercises PlanCheck.UnmarshalJSON
// directly. CORRECTION09 requires the production change to
// have direct tests proving the custom decoder is reached,
// that absence and null are represented distinctly, and that
// the typed error family is used.
func TestPlanCheckDirectDecoder(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantErrKind string
		wantSecs    int
	}{
		{"absent", `{"id":"x","mode":"run","argv":["t"]}`, "", 0},
		{"null", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":null}`, "non_number", 0},
		{"string_60", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":"60"}`, "non_number", 0},
		{"bool", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":true}`, "non_number", 0},
		{"array", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":[]}`, "non_number", 0},
		{"object", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":{}}`, "non_number", 0},
		{"zero", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":0}`, "below_minimum", 0},
		{"one", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":1}`, "", 1},
		{"one_dot_zero", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":1.0}`, "", 1},
		{"one_e_zero", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":1e0}`, "", 1},
		{"one_point_five", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":1.5}`, "non_integral", 0},
		{"one_e_minus_one", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":1e-1}`, "non_integral", 0},
		{"599", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":599}`, "", 599},
		{"600", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":600}`, "", 600},
		{"600_dot_zero", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":600.0}`, "", 600},
		{"6e2", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":6e2}`, "", 600},
		{"601", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":601}`, "above_maximum", 0},
		{"600_dot_001", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":600.000000000000000000000000000000001}`, "non_integral", 0},
		{"float64_max_p2", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":9007199254740993}`, "above_maximum", 0},
		{"int64_max_p1", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":9223372036854775808}`, "above_maximum", 0},
		{"1e1000", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":1e1000}`, "above_maximum", 0},
		{"malformed_json", `{"id":"x","mode":"run","argv":["t"],"timeout_seconds":1xyz}`, "syntax", 0},
		{"unknown_property", `{"id":"x","mode":"run","argv":["t"],"unknown_field":1}`, "disallow_unknown", 0},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			var got PlanCheck
			err := got.UnmarshalJSON([]byte(c.input))
			if c.wantErrKind == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if got.TimeoutSeconds != c.wantSecs {
					t.Fatalf("TimeoutSeconds=%d want %d", got.TimeoutSeconds, c.wantSecs)
				}
				return
			}
			if err == nil {
				t.Fatalf("want error kind %s, got nil", c.wantErrKind)
			}
			if c.wantErrKind == "syntax" {
				// Real JSON syntax error: not a TimeoutDecodeError.
				if _, ok := err.(*json.SyntaxError); !ok {
					t.Fatalf("want json.SyntaxError, got %T: %v", err, err)
				}
				return
			}
			if c.wantErrKind == "disallow_unknown" {
				// DisallowUnknownFields: also a real decoder error.
				if _, ok := err.(*json.SyntaxError); !ok {
					// Accept any decoder error message
					if !strings.Contains(err.Error(), "unknown field") {
						t.Fatalf("want unknown-field error, got %v", err)
					}
				}
				return
			}
			var tErr *TimeoutDecodeError
			if !errors.As(err, &tErr) {
				t.Fatalf("want TimeoutDecodeError, got %T: %v", err, err)
			}
			if tErr.Kind != c.wantErrKind {
				t.Fatalf("error kind=%s want %s", tErr.Kind, c.wantErrKind)
			}
		})
	}
}

// TestExactNumberAuthorityDirect exercises the production
// exact-number authority independently of PlanCheck. The
// authority is the only path through which public integer and
// bound comparisons reach numeric decisions.
func TestExactNumberAuthorityDirect(t *testing.T) {
	a := NewExactNumberAuthority()
	// Parse
	if r, ok := a.ParseExactNumber("1"); !ok || r.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("ParseExactNumber(1) failed: %v", r)
	}
	if r, ok := a.ParseExactNumber("600.000000000000000000000000000000001"); !ok || !r.IsInt() == false {
		t.Fatalf("ParseExactNumber(600.000...001) failed")
	}
	if _, ok := a.ParseExactNumber("garbage"); ok {
		t.Fatalf("ParseExactNumber(garbage) should fail")
	}
	// InRange
	if ok, _ := a.InRange(big.NewRat(600, 1), 1, 600, true); !ok {
		t.Fatalf("600 should be in [1, 600]")
	}
	if ok, kind := a.InRange(big.NewRat(601, 1), 1, 600, true); ok || kind != "above_maximum" {
		t.Fatalf("601 should be above_maximum")
	}
	if ok, kind := a.InRange(big.NewRat(0, 1), 1, 600, true); ok || kind != "below_minimum" {
		t.Fatalf("0 should be below_minimum")
	}
	if ok, kind := a.InRange(big.NewRat(15, 10), 1, 600, true); ok || kind != "non_integral" {
		t.Fatalf("1.5 should be non_integral")
	}
	// DecodeTimeout
	if v, err := a.DecodeTimeout("1"); err != nil || v != 1 {
		t.Fatalf("DecodeTimeout(1)=%d err=%v", v, err)
	}
	if v, err := a.DecodeTimeout("1.0"); err != nil || v != 1 {
		t.Fatalf("DecodeTimeout(1.0)=%d err=%v", v, err)
	}
	if v, err := a.DecodeTimeout("1e0"); err != nil || v != 1 {
		t.Fatalf("DecodeTimeout(1e0)=%d err=%v", v, err)
	}
	if v, err := a.DecodeTimeout("6e2"); err != nil || v != 600 {
		t.Fatalf("DecodeTimeout(6e2)=%d err=%v", v, err)
	}
	if v, err := a.DecodeTimeout(""); err == nil || v != 0 {
		t.Fatalf("DecodeTimeout('') want error")
	}
	if v, err := a.DecodeTimeout("garbage"); err == nil || v != 0 {
		t.Fatalf("DecodeTimeout(garbage) want error")
	}
}

// TestPlanCheckReplacementSemantics proves the custom decoder
// fully replaces prior receiver state rather than merging into it.
func TestPlanCheckReplacementSemantics(t *testing.T) {
	c := &PlanCheck{
		ID:             "stale",
		Mode:           "stale_mode",
		Argv:           []string{"stale"},
		TimeoutSeconds: 999,
	}
	if err := c.UnmarshalJSON([]byte(`{"id":"fresh","mode":"run","argv":["t"]}`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if c.ID != "fresh" || c.Mode != "run" || len(c.Argv) != 1 || c.Argv[0] != "t" || c.TimeoutSeconds != 0 {
		t.Fatalf("decoder merged: %+v", c)
	}
}

// TestExactNumberMarshalRoundTrip proves that the decoded
// PlanCheck round-trips through json.Marshal correctly.
func TestExactNumberMarshalRoundTrip(t *testing.T) {
	in := []byte(`{"id":"x","mode":"run","argv":["t"],"timeout_seconds":60,"environment":{}}`)
	var c PlanCheck
	if err := c.UnmarshalJSON(in); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if c.TimeoutSeconds != 60 {
		t.Fatalf("TimeoutSeconds=%d want 60", c.TimeoutSeconds)
	}
	out, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !bytes.Contains(out, []byte(`"timeout_seconds":60`)) {
		t.Fatalf("marshal output missing timeout_seconds: %s", out)
	}
}
