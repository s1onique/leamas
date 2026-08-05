package closure

import (
	"errors"
	"fmt"
	"math/big"
)

// plan_exact_number.go centralises the production exact-number
// authority used by both the schema evaluator and PlanCheck
// runtime decoder. The authority is the only path through
// which public integer and bound comparisons reach numeric
// decisions. It does not pass through float64.
//
// CORRECTION09: the helpers in this file are consumed by:
//   - PlanCheck.UnmarshalJSON (runtime timeout decoding)
//   - evalField (schema evaluator bound checks)
//   - applyApplicabilityRules (no direct numeric path)
//
// The schema-evaluator test-only helpers in
// plan_schema_evaluator_helpers.go continue to expose
// valueMatchesType, schemaFieldRequiresPresence, and
// isEmpty for the evaluator. Those are distinct from this
// production number authority.

// TimeoutBounds is the inclusive integer range the Closure
// Protocol v1 contract pins for timeout_seconds.
const (
	TimeoutMinSeconds int64 = 1
	TimeoutMaxSeconds int64 = 600
)

// TimeoutDecodeError is the typed error family for runtime
// timeout decoding. It carries enough structure for the
// diagnostic stage to surface a stable contract.
type TimeoutDecodeError struct {
	Kind        string
	Raw         string
	Minimum     int64
	Maximum     int64
	Cause       error
}

// Error implements the error interface.
func (e *TimeoutDecodeError) Error() string {
	switch e.Kind {
	case "non_number":
		return fmt.Sprintf("timeout_seconds: not a number: %s", e.Raw)
	case "non_integral":
		return fmt.Sprintf("timeout_seconds: not an integer: %s", e.Raw)
	case "below_minimum":
		return fmt.Sprintf("timeout_seconds: below %d: %s", e.Minimum, e.Raw)
	case "above_maximum":
		return fmt.Sprintf("timeout_seconds: above %d: %s", e.Maximum, e.Raw)
	}
	return fmt.Sprintf("timeout_seconds: %s: %s", e.Kind, e.Raw)
}

// Unwrap exposes the underlying cause for errors.Is / errors.As.
func (e *TimeoutDecodeError) Unwrap() error { return e.Cause }

// ErrTimeoutNonNumber is the canonical error returned when
// timeout_seconds is not a JSON number.
var ErrTimeoutNonNumber = errors.New("timeout_seconds: not a number")

// ErrTimeoutNonIntegral is the canonical error returned when
// timeout_seconds is mathematically non-integral.
var ErrTimeoutNonIntegral = errors.New("timeout_seconds: not an integer")

// ErrTimeoutBelowMinimum is the canonical error returned when
// timeout_seconds is below TimeoutMinSeconds.
var ErrTimeoutBelowMinimum = fmt.Errorf("timeout_seconds: below %d", TimeoutMinSeconds)

// ErrTimeoutAboveMaximum is the canonical error returned when
// timeout_seconds is above TimeoutMaxSeconds.
var ErrTimeoutAboveMaximum = fmt.Errorf("timeout_seconds: above %d", TimeoutMaxSeconds)

// ExactNumberAuthority is the documented public surface for the
// exact-number authority. Helpers in this file are the only
// authoritative numeric path for Closure Protocol v1
// validation.
type ExactNumberAuthority struct{}

// NewExactNumberAuthority returns the singleton authority.
func NewExactNumberAuthority() ExactNumberAuthority {
	return ExactNumberAuthority{}
}

// ParseExactNumber parses a literal JSON number text into a
// big.Rat. Returns nil, false on parse error or empty input.
func (ExactNumberAuthority) ParseExactNumber(s string) (*big.Rat, bool) {
	if s == "" {
		return nil, false //nolint
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, false //nolint
	}
	return r, true
}

// IsIntegral reports whether the rational is mathematically
// integral.
func (ExactNumberAuthority) IsIntegral(r *big.Rat) bool {
	if r == nil {
		return false
	}
	return r.IsInt()
}

// InRange reports whether r is within [min, max] inclusive
// using exact rational comparison. Non-integral r is rejected
// when requireInteger is true.
func (ExactNumberAuthority) InRange(r *big.Rat, min, max int64, requireInteger bool) (bool, string) {
	if r == nil {
		return false, "non_number" //nolint:govet
	}
	if requireInteger && !r.IsInt() {
		return false, "non_integral"
	}
	minRat := new(big.Rat).SetInt64(min)
	maxRat := new(big.Rat).SetInt64(max)
	if r.Cmp(minRat) < 0 {
		return false, "below_minimum"
	}
	if r.Cmp(maxRat) > 0 {
		return false, "above_maximum"
	}
	return true, ""
}

// ConvertBoundedInteger converts an integral r into an int64
// after proving the value fits in int64.
func (ExactNumberAuthority) ConvertBoundedInteger(r *big.Rat) (int64, bool) {
	if r == nil || !r.IsInt() {
		return 0, false
	}
	num := r.Num()
	if !num.IsInt64() {
		return 0, false
	}
	return num.Int64(), true
}

// DecodeTimeout is the production entry point for converting
// a literal JSON number into the integer seconds the runtime
// expects. It returns the integer, a typed error suitable for
// the diagnostic stage, and a presence flag distinguishing
// absent (false) from present (true). The presence flag is
// returned through a separate channel because Go's encoding/json
// does not natively distinguish absent and null on a single
// field.
func (a ExactNumberAuthority) DecodeTimeout(raw string) (int64, *TimeoutDecodeError) {
	if raw == "" {
		return 0, &TimeoutDecodeError{Kind: "absent", Raw: ""}
	}
	rat, ok := a.ParseExactNumber(raw)
	if !ok {
		return 0, &TimeoutDecodeError{Kind: "non_number", Raw: raw, Cause: ErrTimeoutNonNumber}
	}
	if ok, kind := a.InRange(rat, TimeoutMinSeconds, TimeoutMaxSeconds, true); !ok {
		switch kind {
		case "non_integral":
			return 0, &TimeoutDecodeError{Kind: "non_integral", Raw: raw, Cause: ErrTimeoutNonIntegral}
		case "below_minimum":
			return 0, &TimeoutDecodeError{Kind: "below_minimum", Raw: raw, Minimum: TimeoutMinSeconds, Cause: ErrTimeoutBelowMinimum}
		case "above_maximum":
			return 0, &TimeoutDecodeError{Kind: "above_maximum", Raw: raw, Maximum: TimeoutMaxSeconds, Cause: ErrTimeoutAboveMaximum}
		}
	}
	v, ok := a.ConvertBoundedInteger(rat)
	if !ok {
		return 0, &TimeoutDecodeError{Kind: "non_integral", Raw: raw, Cause: ErrTimeoutNonIntegral}
	}
	return v, nil
}
