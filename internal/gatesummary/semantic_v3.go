package gatesummary

import (
	"fmt"
	"sort"
)

// validateV3Counts checks that the v3 top-level `counts` aggregate is
// consistent with the per-check statuses and with len(checks).
// The v3 schema only enforces per-field constraints; this validator
// closes the cross-field invariants:
//
//   - counts.total == len(checks)
//   - counts.pass + counts.fail + counts.timeout + counts.skip
//     + counts.unavailable == counts.total
//
// A mismatch produces GS_TEST_TOTAL_MISMATCH so consumers that
// trust the aggregate can refuse to interpret the document.
//
// The validator operates on the normalized Summary model so it does
// not depend on the wire-format field names.
func validateV3Counts(s Summary) []Diagnostic {
	var diags []Diagnostic

	wantTotal := len(s.Checks)
	if s.Counts.Total != wantTotal {
		diags = append(diags, Diagnostic{
			Code:     CodeTestTotalMismatch,
			Path:     "/counts/total",
			Expected: fmt.Sprintf("%d", wantTotal),
			Observed: fmt.Sprintf("%d", s.Counts.Total),
			Message: fmt.Sprintf("counts.total=%d disagrees with len(checks)=%d",
				s.Counts.Total, wantTotal),
		})
	}

	wantSum := s.Counts.Pass + s.Counts.Fail + s.Counts.Timeout +
		s.Counts.Skip + s.Counts.Unavailable
	if wantSum != s.Counts.Total {
		diags = append(diags, Diagnostic{
			Code:     CodeTestTotalMismatch,
			Path:     "/counts/total",
			Expected: fmt.Sprintf("%d", wantSum),
			Observed: fmt.Sprintf("%d", s.Counts.Total),
			Message: fmt.Sprintf(
				"counts.total=%d disagrees with pass+fail+timeout+skip+unavailable=%d",
				s.Counts.Total, wantSum),
		})
	}

	return diags
}

// validateV3NameLists checks that each of the four parallel name
// arrays (failed_names, timeout_names, skipped_names,
// unavailable_names) exactly matches the names of checks whose
// status is the corresponding value. The names must also be sorted
// because the v3 schema does not constrain ordering and the
// producer's contract is sorted-by-name to keep the digest
// deterministic.
//
// A mismatch produces GS_DUPLICATE_CHECK_NAME (closest existing
// code) for first-time mismatches. The diagnostic is precise: it
// identifies which name was missing or extra.
func validateV3NameLists(s Summary) []Diagnostic {
	var diags []Diagnostic

	failed := collectNamesByStatus(s.Checks, GateFail)
	timeout := collectNamesByStatus(s.Checks, GateTimeout)
	skipped := collectNamesByStatus(s.Checks, GateSkip)
	unavailable := collectNamesByStatus(s.Checks, GateUnavailable)

	diags = append(diags, compareNameList(s.FailedNames, failed,
		"/failed_names", "fail")...)
	diags = append(diags, compareNameList(s.TimeoutNames, timeout,
		"/timeout_names", "timeout")...)
	diags = append(diags, compareNameList(s.SkippedNames, skipped,
		"/skipped_names", "skip")...)
	diags = append(diags, compareNameList(s.UnavailableNames, unavailable,
		"/unavailable_names", "unavailable")...)

	return diags
}

// collectNamesByStatus returns the sorted set of check names whose
// status matches want.
func collectNamesByStatus(checks []Check, want GateStatus) []string {
	out := make([]string, 0)
	for _, c := range checks {
		if c.Status == want && c.Name != "" {
			out = append(out, c.Name)
		}
	}
	sort.Strings(out)
	return out
}

// compareNameList diffs the recorded list against the derived list
// and produces at most one diagnostic per side of the diff so the
// reviewer can locate the offending names without scanning.
func compareNameList(recorded, derived []string, path, statusLabel string) []Diagnostic {
	var diags []Diagnostic

	recSet := toSortedSet(recorded)
	derSet := toSortedSet(derived)

	// Detect duplicates inside recorded: the v3 schema requires
	// uniqueItems; the decoder enforces it but a malformed
	// producer path could still surface here.
	seen := make(map[string]int, len(recorded))
	for _, n := range recorded {
		seen[n]++
	}
	for n, c := range seen {
		if c > 1 {
			diags = append(diags, Diagnostic{
				Code:    CodeDuplicateCheckName,
				Path:    path,
				Message: fmt.Sprintf("%s name %q appears %d times", statusLabel, n, c),
			})
		}
	}

	// Names recorded but not derived.
	for _, n := range recorded {
		if !derSet[n] {
			diags = append(diags, Diagnostic{
				Code:    CodeDuplicateCheckName,
				Path:    path,
				Message: fmt.Sprintf("%s_name %q in %s but no check with status %q",
					statusLabel, n, path, statusLabel),
			})
		}
	}
	// Names derived but not recorded.
	for _, n := range derived {
		if !recSet[n] {
			diags = append(diags, Diagnostic{
				Code:    CodeDuplicateCheckName,
				Path:    path,
				Message: fmt.Sprintf("check %q has status %q but is missing from %s",
					n, statusLabel, path),
			})
		}
	}

	return diags
}

// toSortedSet returns a map[string]bool built from a sorted list.
// The input is assumed to be sorted; callers always pass
// collectNamesByStatus output which sorts internally.
func toSortedSet(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for _, n := range in {
		out[n] = true
	}
	return out
}
