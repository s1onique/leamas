// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/gatesummary"
)

// compareIntegers compares two Integer values using arbitrary-precision arithmetic.
// Returns -1, 0, or +1.
func compareIntegers(left, right *gatesummary.Integer) int {
	leftBig, leftOK := left.BigInt()
	rightBig, rightOK := right.BigInt()
	if !leftOK || !rightOK {
		panic("normalized Integer must be valid")
	}
	return leftBig.Cmp(rightBig)
}

// renderGateSummaryV1 renders a valid v1 summary.
func renderGateSummaryV1(sourcePath string, summary gatesummary.Summary) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=present\n")
	sb.WriteString(fmt.Sprintf("schema_version=1\n"))
	sb.WriteString(fmt.Sprintf("generated_at=%s\n", sanitizeLine(summary.GeneratedAt)))
	sb.WriteString(fmt.Sprintf("overall_status=%s\n", summary.Overall.Status))

	// Count checks
	totals := countChecks(summary.Checks)

	sb.WriteString(fmt.Sprintf("checks_total=%d\n", totals.total))
	sb.WriteString(fmt.Sprintf("checks_passed=%d\n", totals.passed))
	sb.WriteString(fmt.Sprintf("checks_failed=%d\n", totals.failed))
	sb.WriteString(fmt.Sprintf("checks_skipped=%d\n", totals.skipped))
	sb.WriteString(fmt.Sprintf("checks_unavailable=%d\n", totals.unavailable))

	// Render checks - sort deterministically without mutating original
	if len(summary.Checks) > 0 {
		checks := copyAndSortChecksV1(summary.Checks)
		sb.WriteString("checks:\n")
		for _, c := range checks {
			// V1: absent duration omitted; zero is also omitted (legacy behavior)
			durationStr := ""
			if c.DurationMs != nil && !c.DurationMs.IsZero() {
				durationStr = fmt.Sprintf(" duration_ms=%s", c.DurationMs.String())
			}
			// V1: empty evidence falls back to check name
			evidence := ""
			if c.Evidence != nil && *c.Evidence != "" {
				evidence = sanitizeLine(*c.Evidence)
			} else {
				evidence = sanitizeLine(c.Name)
			}
			sb.WriteString(fmt.Sprintf("  - name=%s status=%s%s evidence=%s\n",
				sanitizeLine(c.Name), c.Status, durationStr, evidence))
		}
	}

	return sb.String()
}

// renderGateSummaryV2 renders a valid v2 summary with fixed check row geometry.
// All fields are rendered in fixed order with empty values when absent for
// deterministic downstream parsing.
func renderGateSummaryV2(sourcePath string, summary gatesummary.Summary) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=present\n")
	sb.WriteString(fmt.Sprintf("schema_version=2\n"))
	sb.WriteString(fmt.Sprintf("generated_at=%s\n", sanitizeLine(summary.GeneratedAt)))

	// Scope fields
	if summary.Scope != nil {
		sb.WriteString(fmt.Sprintf("scope_id=%s\n", sanitizeLine(summary.Scope.ID)))
		sb.WriteString(fmt.Sprintf("scope_status=%s\n", summary.Scope.Status))
		sb.WriteString(fmt.Sprintf("scope_disposition=%s\n", sanitizeLine(summary.Scope.Disposition)))
	} else {
		sb.WriteString("scope_id=\n")
		sb.WriteString("scope_status=\n")
		sb.WriteString("scope_disposition=\n")
	}

	// Parent fields
	if summary.Parent != nil {
		sb.WriteString(fmt.Sprintf("parent_act=%s\n", sanitizeLine(summary.Parent.Act)))
		sb.WriteString(fmt.Sprintf("parent_status=%s\n", summary.Parent.Status))
		sb.WriteString(fmt.Sprintf("parent_disposition=%s\n", sanitizeLine(summary.Parent.Disposition)))
		if summary.Parent.Root {
			sb.WriteString("parent_root=true\n")
		} else {
			sb.WriteString("parent_root=false\n")
		}
	} else {
		sb.WriteString("parent_act=\n")
		sb.WriteString("parent_status=\n")
		sb.WriteString("parent_disposition=\n")
		sb.WriteString("parent_root=\n")
	}

	// Overall fields
	sb.WriteString(fmt.Sprintf("overall_status=%s\n", summary.Overall.Status))
	if summary.Overall.Disposition != nil {
		sb.WriteString(fmt.Sprintf("overall_disposition=%s\n", sanitizeLine(*summary.Overall.Disposition)))
	} else {
		sb.WriteString("overall_disposition=\n")
	}

	// Execution binding
	if summary.Execution != nil {
		sb.WriteString(fmt.Sprintf("execution_head_oid=%s\n", sanitizeLine(summary.Execution.HeadOID)))
		sb.WriteString(fmt.Sprintf("execution_tree_oid=%s\n", sanitizeLine(summary.Execution.TreeOID)))
		sb.WriteString(fmt.Sprintf("subject_tree_oid=%s\n", sanitizeLine(summary.Execution.SubjectOID)))
	} else {
		sb.WriteString("execution_head_oid=\n")
		sb.WriteString("execution_tree_oid=\n")
		sb.WriteString("subject_tree_oid=\n")
	}

	// Worktree cleanliness
	if summary.Worktree != nil {
		if summary.Worktree.CleanBefore {
			sb.WriteString("worktree_clean_before=true\n")
		} else {
			sb.WriteString("worktree_clean_before=false\n")
		}
		if summary.Worktree.CleanAfter {
			sb.WriteString("worktree_clean_after=true\n")
		} else {
			sb.WriteString("worktree_clean_after=false\n")
		}
	} else {
		sb.WriteString("worktree_clean_before=\n")
		sb.WriteString("worktree_clean_after=\n")
	}

	// Count checks
	totals := countChecks(summary.Checks)

	sb.WriteString(fmt.Sprintf("checks_total=%d\n", totals.total))
	sb.WriteString(fmt.Sprintf("checks_passed=%d\n", totals.passed))
	sb.WriteString(fmt.Sprintf("checks_failed=%d\n", totals.failed))
	sb.WriteString(fmt.Sprintf("checks_skipped=%d\n", totals.skipped))
	sb.WriteString(fmt.Sprintf("checks_unavailable=%d\n", totals.unavailable))

	// Render checks - sort deterministically without mutating original
	// Uses stable sort for deterministic ordering when keys are equal
	if len(summary.Checks) > 0 {
		checks := copyAndSortChecksV2(summary.Checks)
		sb.WriteString("checks:\n")
		for _, c := range checks {
			scopeStr := ""
			if c.Scope != nil {
				scopeStr = sanitizeLine(*c.Scope)
			}
			// V2: absent duration omitted; present zero renders as duration_ms=0
			durationStr := ""
			if c.DurationMs != nil {
				durationStr = c.DurationMs.String()
			}
			exitCodeStr := ""
			if c.Execution != nil && c.Execution.ExitCode != nil {
				exitCodeStr = c.Execution.ExitCode.String()
			}
			evidence := ""
			if c.Evidence != nil {
				evidence = sanitizeLine(*c.Evidence)
			}
			sb.WriteString(fmt.Sprintf(
				"  - name=%s scope=%s status=%s duration_ms=%s exit_code=%s evidence=%s\n",
				sanitizeLine(c.Name), scopeStr, c.Status, durationStr, exitCodeStr, evidence,
			))
		}
	}

	return sb.String()
}

// checkCounts holds normalized check totals.
type checkCounts struct {
	total       int
	passed      int
	failed      int
	skipped     int
	unavailable int
}

// countChecks computes check totals from a normalized check slice.
func countChecks(checks []gatesummary.Check) checkCounts {
	var c checkCounts
	c.total = len(checks)
	for _, check := range checks {
		switch check.Status {
		case gatesummary.GatePass:
			c.passed++
		case gatesummary.GateFail:
			c.failed++
		case gatesummary.GateSkip:
			c.skipped++
		case gatesummary.GateUnavailable:
			c.unavailable++
		}
	}
	return c
}

// copyAndSortChecksV1 creates a sorted copy for v1 rendering.
// Uses complete canonical sorting key: name, status, duration-present, duration-value, sanitized-evidence.
func copyAndSortChecksV1(checks []gatesummary.Check) []gatesummary.Check {
	type indexedCheck struct {
		check gatesummary.Check
		index int
	}
	indexed := make([]indexedCheck, len(checks))
	for i, c := range checks {
		indexed[i] = indexedCheck{check: c, index: i}
	}
	sort.SliceStable(indexed, func(i, j int) bool {
		ci, cj := indexed[i].check, indexed[j].check
		// Canonical key: name
		if ci.Name != cj.Name {
			return ci.Name < cj.Name
		}
		// Canonical key: status
		if ci.Status != cj.Status {
			return ci.Status < cj.Status
		}
		// Canonical key: duration-present (absent < present, zero counts as absent for V1)
		hasDurI := ci.DurationMs != nil && !ci.DurationMs.IsZero()
		hasDurJ := cj.DurationMs != nil && !cj.DurationMs.IsZero()
		if hasDurI != hasDurJ {
			return !hasDurI // absent (false) comes before present (true)
		}
		// Canonical key: duration-value (only if present and non-zero)
		if hasDurI {
			cmp := compareIntegers(ci.DurationMs, cj.DurationMs)
			if cmp != 0 {
				return cmp < 0
			}
		}
		// Canonical key: sanitized-evidence
		evI := ci.Evidence
		if evI == nil || *evI == "" {
			evI = &ci.Name
		}
		evJ := cj.Evidence
		if evJ == nil || *evJ == "" {
			evJ = &cj.Name
		}
		seI := sanitizeLine(*evI)
		seJ := sanitizeLine(*evJ)
		if seI != seJ {
			return seI < seJ
		}
		// Tiebreaker: original index
		return indexed[i].index < indexed[j].index
	})
	result := make([]gatesummary.Check, len(checks))
	for i, ic := range indexed {
		result[i] = ic.check
	}
	return result
}

// copyAndSortChecksV2 creates a sorted copy for v2 rendering.
// Uses complete canonical sorting key: name, scope, status, duration-present, duration-value,
// exit-code-present, exit-code-value, sanitized-evidence.
func copyAndSortChecksV2(checks []gatesummary.Check) []gatesummary.Check {
	type indexedCheck struct {
		check gatesummary.Check
		index int
	}
	indexed := make([]indexedCheck, len(checks))
	for i, c := range checks {
		indexed[i] = indexedCheck{check: c, index: i}
	}
	sort.SliceStable(indexed, func(i, j int) bool {
		ci, cj := indexed[i].check, indexed[j].check
		// Canonical key: name
		if ci.Name != cj.Name {
			return ci.Name < cj.Name
		}
		// Canonical key: scope
		scopeI := ""
		scopeJ := ""
		if ci.Scope != nil {
			scopeI = *ci.Scope
		}
		if cj.Scope != nil {
			scopeJ = *cj.Scope
		}
		if scopeI != scopeJ {
			return scopeI < scopeJ
		}
		// Canonical key: status
		if ci.Status != cj.Status {
			return ci.Status < cj.Status
		}
		// Canonical key: duration-present (absent < present, zero counts as present for V2)
		hasDurI := ci.DurationMs != nil
		hasDurJ := cj.DurationMs != nil
		if hasDurI != hasDurJ {
			return !hasDurI
		}
		// Canonical key: duration-value (only if present)
		if hasDurI {
			cmp := compareIntegers(ci.DurationMs, cj.DurationMs)
			if cmp != 0 {
				return cmp < 0
			}
		}
		// Canonical key: exit-code-present (absent < present)
		hasECI := ci.Execution != nil && ci.Execution.ExitCode != nil
		hasECJ := cj.Execution != nil && cj.Execution.ExitCode != nil
		if hasECI != hasECJ {
			return !hasECI
		}
		// Canonical key: exit-code-value (only if present)
		if hasECI {
			cmp := compareIntegers(ci.Execution.ExitCode, cj.Execution.ExitCode)
			if cmp != 0 {
				return cmp < 0
			}
		}
		// Canonical key: sanitized-evidence
		evI := ""
		if ci.Evidence != nil {
			evI = sanitizeLine(*ci.Evidence)
		}
		evJ := ""
		if cj.Evidence != nil {
			evJ = sanitizeLine(*cj.Evidence)
		}
		if evI != evJ {
			return evI < evJ
		}
		// Tiebreaker: original index
		return indexed[i].index < indexed[j].index
	})
	result := make([]gatesummary.Check, len(checks))
	for i, ic := range indexed {
		result[i] = ic.check
	}
	return result
}
