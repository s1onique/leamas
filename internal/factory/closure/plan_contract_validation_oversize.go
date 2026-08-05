package closure

import (
	"encoding/json"
	"math/big"
)

// plan_contract_validation_oversize.go centralises the
// CORRECTION16 helpers used when the supplied integer
// exceeds the int64 range. Splitting these helpers from
// plan_contract_validation_fields.go keeps every file under
// the LLM-friendly 400-line threshold while preserving the
// single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// validateOversizeInteger classifies an integral JSON value that
// does not fit in int64. CORRECTION16: the maximum comparison
// must occur on the rational form so 9223372036854775808
// (int64_max+1) and 1e1000 classify as numeric_above_maximum
// rather than invalid_type.
//
// The function rejects on below_minimum / above_maximum when
// the descriptor declares a bound. When no bound is declared
// it surfaces an invalid_type diagnostic so the absence of a
// bound never silently accepts an oversize integer.
func validateOversizeInteger(field planFieldDescriptor, path string, value any) []PlanValidationError {
	var diagnostics []PlanValidationError
	var rat *big.Rat
	switch v := value.(type) {
	case json.Number:
		authority := NewExactNumberAuthority()
		r, ok := authority.ParseExactNumber(string(v))
		if !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be an integer, got " + typeNameOf(value),
				RejectedValue: value,
				PropertyName:  field.JSONName,
			})
			return diagnostics
		}
		rat = r
	case float64:
		r := new(big.Rat).SetFloat64(v)
		rat = r
	default:
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:  path,
			SchemaPath:    path,
			Code:          PlanCodeInvalidType,
			Keyword:       KeywordType,
			Message:       "property \"" + field.JSONName + "\" must be an integer, got " + typeNameOf(value),
			RejectedValue: value,
			PropertyName:  field.JSONName,
		})
		return diagnostics
	}
	if field.Minimum != nil && rat.Cmp(big.NewRat(*field.Minimum, 1)) < 0 {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:   path,
			SchemaPath:     path,
			Code:           PlanCodeNumericBelowMinimum,
			Keyword:        KeywordMinimum,
			Message:        "property \"" + field.JSONName + "\" value " + rat.String() + " is below minimum " + itoa64(*field.Minimum),
			RejectedValue:  value,
			PropertyName:   field.JSONName,
			AcceptedValues: integerBoundRange(field),
		})
		return diagnostics
	}
	if field.Maximum != nil && rat.Cmp(big.NewRat(*field.Maximum, 1)) > 0 {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:   path,
			SchemaPath:     path,
			Code:           PlanCodeNumericAboveMaximum,
			Keyword:        KeywordMaximum,
			Message:        "property \"" + field.JSONName + "\" value " + rat.String() + " exceeds maximum " + itoa64(*field.Maximum),
			RejectedValue:  value,
			PropertyName:   field.JSONName,
			AcceptedValues: integerBoundRange(field),
		})
		return diagnostics
	}
	diagnostics = append(diagnostics, PlanValidationError{
		InstancePath:  path,
		SchemaPath:    path,
		Code:          PlanCodeInvalidType,
		Keyword:       KeywordType,
		Message:       "property \"" + field.JSONName + "\" integer " + rat.String() + " does not fit in int64",
		RejectedValue: value,
		PropertyName:  field.JSONName,
	})
	return diagnostics
}

// integerBoundRange renders the inclusive [min, max] bound range
// for integer diagnostics. Missing bounds render as "*" so the
// canonical encoding is unambiguous on both sides.
func integerBoundRange(field planFieldDescriptor) []string {
	minStr := "*"
	if field.Minimum != nil {
		minStr = itoa64(*field.Minimum)
	}
	maxStr := "*"
	if field.Maximum != nil {
		maxStr = itoa64(*field.Maximum)
	}
	return []string{"[" + minStr + ", " + maxStr + "]"}
}

// itoa64 renders an int64 as a base-10 string.
func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	var buf [20]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
