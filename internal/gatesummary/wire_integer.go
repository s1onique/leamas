package gatesummary

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
)

// WireInteger preserves the exact JSON number spelling selected by a
// gate-summary integer schema. Conversion is explicit so the wire boundary
// never narrows values to a machine-sized integer.
type WireInteger struct {
	raw json.Number
}

// UnmarshalJSON accepts one mathematically integral JSON number and preserves
// its source spelling exactly. The selected schema remains authoritative for
// field-level constraints such as non-negativity.
func (n *WireInteger) UnmarshalJSON(data []byte) error {
	if n == nil {
		return errors.New("gatesummary: unmarshal WireInteger into nil receiver")
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("gatesummary: decode wire integer: %w", err)
	}
	number, ok := token.(json.Number)
	if !ok {
		return fmt.Errorf("gatesummary: wire integer requires a JSON number")
	}
	if _, ok := exactBigInt(number); !ok {
		return fmt.Errorf("gatesummary: wire integer is not mathematically integral")
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("gatesummary: extra JSON after wire integer")
		}
		return fmt.Errorf("gatesummary: decode wire integer: %w", err)
	}

	n.raw = number
	return nil
}

// String returns the exact JSON number spelling. The zero value returns an
// empty string because it does not represent a decoded wire integer.
func (n WireInteger) String() string {
	return n.raw.String()
}

// BigInt converts the exact mathematical value without machine-width
// narrowing. The returned integer is owned by the caller.
func (n WireInteger) BigInt() (*big.Int, bool) {
	if n.raw == "" {
		return nil, false
	}
	return exactBigInt(n.raw)
}

// Int64 converts the value only when it is representable as int64.
func (n WireInteger) Int64() (int64, bool) {
	value, ok := n.BigInt()
	if !ok || !value.IsInt64() {
		return 0, false
	}
	return value.Int64(), true
}

func exactBigInt(number json.Number) (*big.Int, bool) {
	value, ok := new(big.Rat).SetString(number.String())
	if !ok || !value.IsInt() {
		return nil, false
	}
	return new(big.Int).Set(value.Num()), true
}
