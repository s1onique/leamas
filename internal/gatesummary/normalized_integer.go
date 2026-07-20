package gatesummary

import (
	"errors"
	"fmt"
	"math/big"
)

// errInvalidInteger is returned when a WireInteger contains an invalid value.
var errInvalidInteger = errors.New("gatesummary: invalid integer value")

// Integer preserves the exact JSON number spelling from the wire document.
// Unlike WireInteger, this type exposes only immutable, owned accessors
// and does not alias any decoder state.
type Integer struct {
	raw string
}

// newIntegerFromWire constructs an Integer from a decoder WireInteger.
// Returns an error if the wire value is empty or not a valid decimal integer.
func newIntegerFromWire(w WireInteger) (Integer, error) {
	raw := w.String()
	if raw == "" {
		return Integer{}, fmt.Errorf("%w: empty wire integer", errInvalidInteger)
	}

	// Validate that the entire string is a valid decimal integer.
	// SetString returns false if the string is not a valid number in the given base.
	if _, ok := new(big.Int).SetString(raw, 10); !ok {
		return Integer{}, fmt.Errorf("%w: %q", errInvalidInteger, raw)
	}

	return Integer{raw: raw}, nil
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
