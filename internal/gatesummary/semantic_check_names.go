package gatesummary

// validateCheckNames checks for duplicate v2 check names.
// Returns diagnostics for each duplicate found (first occurrence is preserved).
// Uses map for linear expected-time detection; preserves encounter order.
func validateCheckNames(checks []Check) []Diagnostic {
	if len(checks) < 2 {
		return nil
	}
	seen := make(map[string]int, len(checks)) // name -> first index
	var diags []Diagnostic
	for i, c := range checks {
		if prev, exists := seen[c.Name]; exists {
			diags = append(diags, Diagnostic{
				Code:     CodeDuplicateCheckName,
				Path:     "/checks/" + itoa(i) + "/name",
				Expected: "unique check name",
				Observed: c.Name,
				Message:  "duplicate check name: " + c.Name,
			})
			_ = prev
		} else {
			seen[c.Name] = i
		}
	}
	return diags
}
