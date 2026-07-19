package gatesummary

import (
	"encoding/json"
	"testing"
)

func TestClassifyVersionV1(t *testing.T) {
	dec := classifyVersion(json.Number("1"))
	if dec.code != "" {
		t.Fatalf("expected success, got %s", dec.code)
	}
	if dec.version != Version1 {
		t.Fatalf("expected Version1, got %d", dec.version)
	}
}

func TestClassifyVersionV2(t *testing.T) {
	dec := classifyVersion(json.Number("2"))
	if dec.code != "" {
		t.Fatalf("expected success, got %s", dec.code)
	}
	if dec.version != Version2 {
		t.Fatalf("expected Version2, got %d", dec.version)
	}
}

func TestClassifyVersionUnsupported(t *testing.T) {
	for _, raw := range []string{"0", "-1", "-2", "3", "4", "99999999999999999999"} {
		dec := classifyVersion(json.Number(raw))
		if dec.code != CodeUnsupportedVersion {
			t.Errorf("raw=%s: expected %s, got %s", raw, CodeUnsupportedVersion, dec.code)
		}
	}
}

func TestClassifyVersionInvalidType(t *testing.T) {
	for _, tok := range []json.Token{"foo", true, false, nil, []any{1}, map[string]int{"a": 1}} {
		dec := classifyVersion(tok)
		if dec.code != CodeInvalidVersionType {
			t.Errorf("tok=%v: expected %s, got %s", tok, CodeInvalidVersionType, dec.code)
		}
	}
}

func TestClassifyVersionDecimal(t *testing.T) {
	for _, raw := range []string{"1.0", "2.0", "2.00", "-2.0", "1e0", "2e0", "2E0", "2e+0", "2e-0"} {
		dec := classifyVersion(json.Number(raw))
		if dec.code != CodeInvalidVersionType {
			t.Errorf("raw=%s: expected %s, got %s", raw, CodeInvalidVersionType, dec.code)
		}
	}
}

func TestClassifyVersionHugeInteger(t *testing.T) {
	// Numbers outside int64 must still classify as unsupported
	// without overflow or GS_INTERNAL.
	for _, raw := range []string{
		"99999999999999999999999999999999",
		"-99999999999999999999999999999999",
	} {
		dec := classifyVersion(json.Number(raw))
		if dec.code != CodeUnsupportedVersion {
			t.Errorf("raw=%s: expected %s, got %s", raw, CodeUnsupportedVersion, dec.code)
		}
	}
}

func TestClassifyVersionZeroAndNegative(t *testing.T) {
	for _, raw := range []string{"0", "-0", "-1", "-2"} {
		dec := classifyVersion(json.Number(raw))
		if dec.code != CodeUnsupportedVersion {
			t.Errorf("raw=%s: expected %s, got %s", raw, CodeUnsupportedVersion, dec.code)
		}
	}
}

func TestIntegerLexicalRe(t *testing.T) {
	must := []string{"0", "-0", "1", "-1", "100", "-100"}
	mustNot := []string{"01", "1.0", "1e0", "+1", "", " 1", "1 "}
	for _, s := range must {
		if !integerLexicalRe.MatchString(s) {
			t.Errorf("%q must match integer regex", s)
		}
	}
	for _, s := range mustNot {
		if integerLexicalRe.MatchString(s) {
			t.Errorf("%q must not match integer regex", s)
		}
	}
}
