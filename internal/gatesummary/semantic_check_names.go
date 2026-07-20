package gatesummary

import "sort"

// validateCheckNames checks for duplicate v2 check names.
// Returns diagnostics for each duplicate found (first occurrence is preserved).
// Uses map for linear expected-time detection; preserves order.
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
				Path:     "/checks/" + escapePointer(c.Name),
				Expected: "unique check name",
				Observed: c.Name,
				Message:  "duplicate check name: " + c.Name,
			})
			// Keep prev pointing to first occurrence
			_ = prev
		} else {
			seen[c.Name] = i
		}
	}
	// Sort by path for deterministic output
	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Path != diags[j].Path {
			return diags[i].Path < diags[j].Path
		}
		return diags[i].Message < diags[j].Message
	})
	return diags
}

// findDuplicateWireNames checks for duplicate names in a string slice.
func findDuplicateWireNames(names []string) []Diagnostic {
	if len(names) < 2 {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	var diags []Diagnostic
	for _, name := range names {
		if _, exists := seen[name]; exists {
			diags = append(diags, Diagnostic{
				Code:     CodeDuplicateCheckName,
				Path:     "/checks/" + escapePointer(name),
				Expected: "unique check name",
				Observed: name,
				Message:  "duplicate check name: " + name,
			})
		} else {
			seen[name] = struct{}{}
		}
	}
	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Path != diags[j].Path {
			return diags[i].Path < diags[j].Path
		}
		return diags[i].Message < diags[j].Message
	})
	return diags
}
