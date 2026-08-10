// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/gatesummary"
)

// copyAndSortChecksV3 creates a sorted copy for v3 rendering.
// Uses the v2 canonical sorting key: name, scope, status, duration-present,
// duration-value, exit-code-present, exit-code-value, sanitized-evidence.
// V3-specific evidence identifiers (id/order/execution_class) are not
// part of the sort key; v3 documents are produced by a single bounded
// runner whose check ordering is already deterministic.
func copyAndSortChecksV3(checks []gatesummary.Check) []gatesummary.Check {
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
		if ci.Name != cj.Name {
			return ci.Name < cj.Name
		}
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
		if ci.Status != cj.Status {
			return ci.Status < cj.Status
		}
		hasDurI := ci.DurationMs != nil
		hasDurJ := cj.DurationMs != nil
		if hasDurI != hasDurJ {
			return !hasDurI
		}
		if hasDurI {
			cmp := compareIntegers(ci.DurationMs, cj.DurationMs)
			if cmp != 0 {
				return cmp < 0
			}
		}
		hasExitI := ci.Execution != nil && ci.Execution.ExitCode != nil
		hasExitJ := cj.Execution != nil && cj.Execution.ExitCode != nil
		if hasExitI != hasExitJ {
			return !hasExitI
		}
		if hasExitI {
			cmp := compareIntegers(ci.Execution.ExitCode, cj.Execution.ExitCode)
			if cmp != 0 {
				return cmp < 0
			}
		}
		evI := ""
		evJ := ""
		if ci.Evidence != nil {
			evI = sanitizeLine(*ci.Evidence)
		}
		if cj.Evidence != nil {
			evJ = sanitizeLine(*cj.Evidence)
		}
		if evI != evJ {
			return evI < evJ
		}
		return indexed[i].index < indexed[j].index
	})
	result := make([]gatesummary.Check, len(checks))
	for i, ic := range indexed {
		result[i] = ic.check
	}
	return result
}

// renderGateSummaryV3 renders a valid v3 summary into the canonical
// digest section. The format is a strict superset of the v2 digest
// section: scope/parent/overall/execution/worktree fields are emitted
// the same way, and v3-only evidence fields (registry_sha256,
// events_sha256, transcript_sha256, counts, failed_names, timeout_names,
// skipped_names, unavailable_names) are appended after the per-check
// block. Per-check rows also surface the v3-only stdout/stderr byte
// counts, truncation flags, raw exit code, termination signal, and
// deadline-exceeded flag.
//
// The rendered block intentionally re-uses the v2 check-row geometry
// (name / scope / status / duration_ms / exit_code / evidence) so that
// downstream digest consumers that already understand v2 continue to
// extract the canonical fields. V3-only per-check evidence appears as
// additional fields on the same row, prefixed to keep the v2 surface
// stable.
func renderGateSummaryV3(sourcePath string, summary gatesummary.Summary) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=present\n")
	sb.WriteString(fmt.Sprintf("schema_version=3\n"))
	sb.WriteString(fmt.Sprintf("generated_at=%s\n", sanitizeLine(summary.GeneratedAt)))

	// Scope fields (identical layout to v2)
	if summary.Scope != nil {
		sb.WriteString(fmt.Sprintf("scope_id=%s\n", sanitizeLine(summary.Scope.ID)))
		sb.WriteString(fmt.Sprintf("scope_status=%s\n", summary.Scope.Status))
		sb.WriteString(fmt.Sprintf("scope_disposition=%s\n", sanitizeLine(summary.Scope.Disposition)))
	} else {
		sb.WriteString("scope_id=\n")
		sb.WriteString("scope_status=\n")
		sb.WriteString("scope_disposition=\n")
	}

	// Parent fields (identical layout to v2)
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

	// Overall fields (identical layout to v2)
	sb.WriteString(fmt.Sprintf("overall_status=%s\n", summary.Overall.Status))
	if summary.Overall.Disposition != nil {
		sb.WriteString(fmt.Sprintf("overall_disposition=%s\n", sanitizeLine(*summary.Overall.Disposition)))
	} else {
		sb.WriteString("overall_disposition=\n")
	}

	// Execution binding (identical layout to v2)
	if summary.Execution != nil {
		sb.WriteString(fmt.Sprintf("execution_head_oid=%s\n", sanitizeLine(summary.Execution.HeadOID)))
		sb.WriteString(fmt.Sprintf("execution_tree_oid=%s\n", sanitizeLine(summary.Execution.TreeOID)))
		sb.WriteString(fmt.Sprintf("subject_tree_oid=%s\n", sanitizeLine(summary.Execution.SubjectOID)))
	} else {
		sb.WriteString("execution_head_oid=\n")
		sb.WriteString("execution_tree_oid=\n")
		sb.WriteString("subject_tree_oid=\n")
	}

	// Worktree cleanliness (identical layout to v2)
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

	// Counts (v3-only authoritative aggregate block)
	c := summary.Counts
	sb.WriteString(fmt.Sprintf("counts_total=%d\n", c.Total))
	sb.WriteString(fmt.Sprintf("counts_pass=%d\n", c.Pass))
	sb.WriteString(fmt.Sprintf("counts_fail=%d\n", c.Fail))
	sb.WriteString(fmt.Sprintf("counts_timeout=%d\n", c.Timeout))
	sb.WriteString(fmt.Sprintf("counts_skip=%d\n", c.Skip))
	sb.WriteString(fmt.Sprintf("counts_unavailable=%d\n", c.Unavailable))

	// Evidence hashes (v3-only top-level binding)
	if summary.EvidenceHashes != nil {
		sb.WriteString(fmt.Sprintf("registry_sha256=%s\n", sanitizeLine(summary.EvidenceHashes.RegistrySHA256)))
		sb.WriteString(fmt.Sprintf("events_sha256=%s\n", sanitizeLine(summary.EvidenceHashes.EventsSHA256)))
		sb.WriteString(fmt.Sprintf("transcript_sha256=%s\n", sanitizeLine(summary.EvidenceHashes.TranscriptSHA256)))
	} else {
		sb.WriteString("registry_sha256=\n")
		sb.WriteString("events_sha256=\n")
		sb.WriteString("transcript_sha256=\n")
	}

	// Per-check counts (use the same per-status counters as v2 so the
	// existing digest surface stays comparable).
	totals := countChecks(summary.Checks)
	sb.WriteString(fmt.Sprintf("checks_total=%d\n", totals.total))
	sb.WriteString(fmt.Sprintf("checks_passed=%d\n", totals.passed))
	sb.WriteString(fmt.Sprintf("checks_failed=%d\n", totals.failed))
	sb.WriteString(fmt.Sprintf("checks_skipped=%d\n", totals.skipped))
	sb.WriteString(fmt.Sprintf("checks_unavailable=%d\n", totals.unavailable))

	// Render checks - sort deterministically without mutating original.
	if len(summary.Checks) > 0 {
		checks := copyAndSortChecksV3(summary.Checks)
		sb.WriteString("checks:\n")
		for _, c := range checks {
			scopeStr := ""
			if c.Scope != nil {
				scopeStr = sanitizeLine(*c.Scope)
			}
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
			// V3-only per-check evidence surfaced as trailing fields.
			idStr := ""
			if c.ID != nil {
				idStr = *c.ID
			}
			orderStr := ""
			if c.Order != nil {
				orderStr = c.Order.String()
			}
			execClassStr := ""
			if c.ExecutionClass != nil {
				execClassStr = *c.ExecutionClass
			}
			stdoutBytes := ""
			stderrBytes := ""
			stdoutTrunc := "false"
			stderrTrunc := "false"
			deadlineExceeded := "false"
			rawExitStr := ""
			signalStr := ""
			implementation := ""
			if c.Execution != nil {
				stdoutBytes = fmt.Sprintf("%d", c.Execution.StdoutBytes)
				stderrBytes = fmt.Sprintf("%d", c.Execution.StderrBytes)
				if c.Execution.StdoutTruncated {
					stdoutTrunc = "true"
				}
				if c.Execution.StderrTruncated {
					stderrTrunc = "true"
				}
				if c.Execution.DeadlineExceeded {
					deadlineExceeded = "true"
				}
				if c.Execution.RawExitCode != nil {
					rawExitStr = c.Execution.RawExitCode.String()
				}
				if c.Execution.TerminationSignal != nil {
					signalStr = sanitizeLine(*c.Execution.TerminationSignal)
				}
				implementation = sanitizeLine(c.Execution.Implementation)
			}
			sb.WriteString(fmt.Sprintf(
				"  - name=%s scope=%s status=%s duration_ms=%s exit_code=%s evidence=%s id=%s order=%s execution_class=%s stdout_bytes=%s stderr_bytes=%s stdout_truncated=%s stderr_truncated=%s deadline_exceeded=%s raw_exit_code=%s termination_signal=%s implementation=%s\n",
				sanitizeLine(c.Name), scopeStr, c.Status, durationStr, exitCodeStr, evidence,
				idStr, orderStr, execClassStr,
				stdoutBytes, stderrBytes, stdoutTrunc, stderrTrunc,
				deadlineExceeded, rawExitStr, signalStr, implementation,
			))
		}
	}

	return sb.String()
}
