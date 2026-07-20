package gatesummary

import (
	"math/big"
)

// Integer preserves the exact JSON number spelling from the wire document.
// Unlike WireInteger, this type exposes only immutable, owned accessors
// and does not alias any decoder state.
type Integer struct {
	raw string
}

// newIntegerFromWire constructs an Integer from a decoder WireInteger.
// Returns an error if the wire value cannot be converted.
func newIntegerFromWire(w WireInteger) (Integer, error) {
	s := w.String()
	if s == "" {
		return Integer{}, nil
	}
	// WireInteger already validated the value during decoding.
	// Just preserve the exact string representation.
	return Integer{raw: s}, nil
}

// String returns the exact JSON number spelling.
func (i Integer) String() string {
	return i.raw
}

// BigInt converts to math/big.Int. Returns nil, false for zero value.
func (i Integer) BigInt() (*big.Int, bool) {
	if i.raw == "" {
		return nil, false
	}
	val, ok := new(big.Rat).SetString(i.raw)
	if !ok || !val.IsInt() {
		return nil, false
	}
	return new(big.Int).Set(val.Num()), true
}

// Int64 converts only when the value fits in int64.
func (i Integer) Int64() (int64, bool) {
	bi, ok := i.BigInt()
	if !ok || !bi.IsInt64() {
		return 0, false
	}
	return bi.Int64(), true
}

// Sign returns -1, 0, or +1 using arbitrary-precision arithmetic.
func (i Integer) Sign() int {
	bi, ok := i.BigInt()
	if !ok {
		return 0
	}
	return bi.Sign()
}

// IsZero returns true for zero value (empty or "0").
func (i Integer) IsZero() bool {
	return i.Sign() == 0
}
