package gatesummary

import (
	"math/big"
	"testing"
)

func TestIntegerString(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		expect string
	}{
		{"zero", "0", "0"},
		{"one", "1", "1"},
		{"negative", "-42", "-42"},
		{"large", "123456789012345678901234567890", "123456789012345678901234567890"},
		{"float_style", "42", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Integer{raw: tt.raw}
			if got := i.String(); got != tt.expect {
				t.Errorf("String() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestIntegerBigInt(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		wantOK bool
	}{
		{"zero", "0", true},
		{"one", "1", true},
		{"negative", "-42", true},
		{"large beyond int64", "9223372036854775808", true},
		{"large positive", "9223372036854775807", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Integer{raw: tt.raw}
			bi, ok := i.BigInt()
			if ok != tt.wantOK {
				t.Errorf("BigInt() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok {
				// Verify the value is parsed correctly
				expected, _ := new(big.Int).SetString(tt.raw, 10)
				if bi.Cmp(expected) != 0 {
					t.Errorf("BigInt() = %s, want %s", bi.String(), expected.String())
				}
			}
		})
	}
}

func TestIntegerInt64(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantVal int64
		wantOK  bool
	}{
		{"zero", "0", 0, true},
		{"one", "1", 1, true},
		{"negative", "-42", -42, true},
		{"beyond int64", "9223372036854775808", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Integer{raw: tt.raw}
			got, ok := i.Int64()
			if ok != tt.wantOK {
				t.Errorf("Int64() ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantOK && got != tt.wantVal {
				t.Errorf("Int64() = %d, want %d", got, tt.wantVal)
			}
		})
	}
}

func TestIntegerSign(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{"positive", "42", 1},
		{"zero", "0", 0},
		{"negative", "-42", -1},
		{"large positive", "123456789012345678901234567890", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Integer{raw: tt.raw}
			if got := i.Sign(); got != tt.want {
				t.Errorf("Sign() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIntegerIsZero(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"zero", "0", true},
		{"one", "1", false},
		{"negative", "-42", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Integer{raw: tt.raw}
			if got := i.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegerArbitraryPrecision(t *testing.T) {
	// Test with a very large integer that exceeds int64
	largeStr := "99999999999999999999999999999999999999999999999999"
	i := Integer{raw: largeStr}

	bi, ok := i.BigInt()
	if !ok {
		t.Fatal("BigInt() failed for large integer")
	}

	// Verify it's the correct value
	expected := new(big.Int)
	expected.SetString(largeStr, 10)
	if bi.Cmp(expected) != 0 {
		t.Errorf("BigInt() = %s, want %s", bi.String(), expected.String())
	}

	// Int64 should fail
	if _, ok := i.Int64(); ok {
		t.Error("Int64() should fail for large integer")
	}
}

func TestNewIntegerFromWire(t *testing.T) {
	tests := []struct {
		name    string
		wire    WireInteger
		wantOK  bool
		wantVal string
	}{
		{"zero", mustNewWireInteger("0"), true, "0"},
		{"one", mustNewWireInteger("1"), true, "1"},
		{"negative", mustNewWireInteger("-42"), true, "-42"},
		{"large", mustNewWireInteger("123456789012345678901234567890"), true, "123456789012345678901234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := newIntegerFromWire(tt.wire)
			if (err == nil) != tt.wantOK {
				t.Errorf("newIntegerFromWire() error = %v, wantOK %v", err, tt.wantOK)
			}
			if i.String() != tt.wantVal {
				t.Errorf("newIntegerFromWire() = %q, want %q", i.String(), tt.wantVal)
			}
		})
	}
}

func mustNewWireInteger(s string) WireInteger {
	var w WireInteger
	// Use JSON unmarshaling for testing
	_ = w.UnmarshalJSON([]byte(s))
	return w
}
