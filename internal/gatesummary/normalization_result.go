package gatesummary

// NormalizationResult is the outcome of Normalize. A successful result
// contains only a normalized Summary. Semantic-invalid input contains
// Diagnostics with nil Err. Operational failures set Err and may carry
// GS_INTERNAL.
type NormalizationResult struct {
	Summary     Summary
	Diagnostics []Diagnostic
	Err         error
}

// Success reports whether normalization produced a valid summary with no diagnostics.
func (r NormalizationResult) Success() bool {
	return r.Err == nil && len(r.Diagnostics) == 0 && r.Summary.Valid()
}
