// SPDX-License-Identifier: Apache-2.0

// Package digest: range_scope.go implements the range-targetedness
// diagnostic required by ACT-LEAMAS-TARGETED-DIGEST-RANGE-SCOPE-WARNING01.
//
// The diagnostic is advisory-only. It does not change the digest range,
// gate semantics, or process exit status. It independently measures the
// geometry of an explicit A..B range and classifies it as NORMAL, BROAD,
// or EXTREME so agents receive early warning of anomalously broad scopes.
package digest

import (
	"fmt"
	"strings"
)

// RangeTargetedness classifies the geometry of an explicit digest range.
// These are UX heuristics only; they have no protocol semantics.
type RangeTargetedness string

const (
	RangeTargetednessNormal  RangeTargetedness = "NORMAL"
	RangeTargetednessBroad   RangeTargetedness = "BROAD"
	RangeTargetednessExtreme RangeTargetedness = "EXTREME"
	RangeTargetednessUnknown RangeTargetedness = "UNKNOWN"
)

// DiagnosticStatus indicates whether the range scope diagnostic succeeded.
type DiagnosticStatus string

const (
	DiagnosticStatusAvailable   DiagnosticStatus = "available"
	DiagnosticStatusUnavailable DiagnosticStatus = "unavailable"
)

// Advisory thresholds for range geometry classification.
// These are UX constants and MUST NOT become closure-protocol invariants.
const (
	broadFileThreshold      = 500
	broadCommitThreshold    = 100
	broadMergeFileThreshold = 250
	extremeFileThreshold    = 1000
)

// Warning codes rendered into the digest.
const (
	WarningCodeNone                        = "none"
	WarningCodeDigestRangeBroad            = "DIGEST_RANGE_BROAD"
	WarningCodeDigestRangeExtreme          = "DIGEST_RANGE_EXTREME"
	WarningCodeDigestRangeScopeUnavailable = "DIGEST_RANGE_SCOPE_UNAVAILABLE"
)

// RangeScope captures the measured geometry of an explicit digest range.
type RangeScope struct {
	LeftEndpointOID  string
	RightEndpointOID string
	CommitCount      int
	MergeCommitCount int
	FilesChanged     int
	CrossesMerge     bool
	Targetedness     RangeTargetedness
	WarningCode      string
	DiagnosticStatus DiagnosticStatus
	DiagnosticError  string
	RawRangeSpec     string
}

// classifyRangeScope is a pure function that classifies range geometry.
func classifyRangeScope(filesChanged, commitCount, mergeCommitCount int) RangeTargetedness {
	if filesChanged > extremeFileThreshold {
		return RangeTargetednessExtreme
	}
	if filesChanged > broadFileThreshold {
		return RangeTargetednessBroad
	}
	if commitCount > broadCommitThreshold {
		return RangeTargetednessBroad
	}
	if filesChanged > broadMergeFileThreshold && mergeCommitCount > 0 {
		return RangeTargetednessBroad
	}
	return RangeTargetednessNormal
}

func warningCodeFor(t RangeTargetedness) string {
	switch t {
	case RangeTargetednessBroad:
		return WarningCodeDigestRangeBroad
	case RangeTargetednessExtreme:
		return WarningCodeDigestRangeExtreme
	default:
		return WarningCodeNone
	}
}

func warningProse(t RangeTargetedness) string {
	switch t {
	case RangeTargetednessBroad:
		return "The explicit range is mechanically valid but unusually broad.\nVerify that its endpoints represent the intended change authority."
	case RangeTargetednessExtreme:
		return "The explicit range spans an exceptionally large change surface.\nThe digest remains mechanically valid; verify that the selected\nendpoints represent the intended change authority."
	default:
		return ""
	}
}

// CollectRangeScope computes the range scope geometry for an explicit range.
// All collection failures are rendered as evidence states.
func CollectRangeScope(repoRoot, rangeSpec string, filesChanged int) *RangeScope {
	scope := &RangeScope{
		FilesChanged:     filesChanged,
		DiagnosticStatus: DiagnosticStatusUnavailable,
		RawRangeSpec:     rangeSpec,
	}

	leftRef, rightRef, err := parseRangeSpec(rangeSpec)
	if err != nil {
		scope.DiagnosticError = truncateError(err.Error())
		return scope
	}

	leftOID, err := resolveToFullOID(repoRoot, leftRef)
	if err != nil {
		scope.DiagnosticError = truncateError(fmt.Sprintf("resolve left endpoint %q: %v", leftRef, err))
		return scope
	}
	scope.LeftEndpointOID = leftOID

	rightOID, err := resolveToFullOID(repoRoot, rightRef)
	if err != nil {
		scope.DiagnosticError = truncateError(fmt.Sprintf("resolve right endpoint %q: %v", rightRef, err))
		return scope
	}
	scope.RightEndpointOID = rightOID

	commitCount, err := countRevList(repoRoot, leftRef, rightRef)
	if err != nil {
		scope.DiagnosticError = truncateError(fmt.Sprintf("count commits: %v", err))
		return scope
	}
	scope.CommitCount = commitCount

	mergeCount, err := countRevListMerges(repoRoot, leftRef, rightRef)
	if err != nil {
		scope.DiagnosticError = truncateError(fmt.Sprintf("count merge commits: %v", err))
		return scope
	}
	scope.MergeCommitCount = mergeCount
	scope.CrossesMerge = mergeCount > 0

	scope.Targetedness = classifyRangeScope(filesChanged, commitCount, mergeCount)
	scope.WarningCode = warningCodeFor(scope.Targetedness)
	scope.DiagnosticStatus = DiagnosticStatusAvailable

	return scope
}

func truncateError(msg string) string {
	const maxLen = 100
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-3] + "..."
}

func parseRangeSpec(rangeSpec string) (left, right string, err error) {
	if strings.Contains(rangeSpec, "...") {
		return "", "", fmt.Errorf("symmetric range (A...B) is not supported by range-scope diagnostic; use A..B")
	}
	left, right, found := strings.Cut(rangeSpec, "..")
	if !found {
		return "", "", fmt.Errorf("invalid range spec %q: expected A..B", rangeSpec)
	}
	left, right = strings.TrimSpace(left), strings.TrimSpace(right)
	if left == "" || right == "" {
		return "", "", fmt.Errorf("invalid range spec %q: both endpoints required", rangeSpec)
	}
	return left, right, nil
}

func resolveToFullOID(repoRoot, ref string) (string, error) {
	out, err := runGitValueTrimmed(repoRoot, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(out)), nil
}

func countRevList(repoRoot, left, right string) (int, error) {
	out, err := runGitValueTrimmed(repoRoot, "rev-list", "--count", left+".."+right)
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(out, "%d", &count); err != nil {
		return 0, fmt.Errorf("parse rev-list count %q: %w", out, err)
	}
	return count, nil
}

func countRevListMerges(repoRoot, left, right string) (int, error) {
	out, err := runGitValueTrimmed(repoRoot, "rev-list", "--count", "--merges", left+".."+right)
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(out, "%d", &count); err != nil {
		return 0, fmt.Errorf("parse merge count %q: %w", out, err)
	}
	return count, nil
}

// RenderRangeScope renders the RangeScope as markdown.
func RenderRangeScope(scope *RangeScope) string {
	if scope == nil {
		return renderUnavailableScope("", "internal error: nil scope")
	}
	var sb strings.Builder
	sb.WriteString("## RANGE_SCOPE\n\nauthority=explicit_range\n")
	if scope.DiagnosticStatus != DiagnosticStatusAvailable {
		sb.WriteString(fmt.Sprintf("targetedness=%s\n", RangeTargetednessUnknown))
		sb.WriteString(fmt.Sprintf("warning_code=%s\n", WarningCodeDigestRangeScopeUnavailable))
		sb.WriteString(fmt.Sprintf("diagnostic_status=%s\n", scope.DiagnosticStatus))
		if scope.RawRangeSpec != "" {
			sb.WriteString(fmt.Sprintf("raw_range_spec=%s\n", scope.RawRangeSpec))
		}
		if scope.DiagnosticError != "" {
			sb.WriteString(fmt.Sprintf("diagnostic_error=%s\n", scope.DiagnosticError))
		}
		sb.WriteString("\nwarning:\n")
		sb.WriteString("  Range-scope diagnostic is unavailable.\n")
		sb.WriteString("  This advisory failure does not alter the selected digest range.\n")
		return sb.String()
	}
	sb.WriteString(fmt.Sprintf("left_endpoint=%s\n", shortSHA(scope.LeftEndpointOID)))
	sb.WriteString(fmt.Sprintf("right_endpoint=%s\n", shortSHA(scope.RightEndpointOID)))
	sb.WriteString(fmt.Sprintf("commit_count=%d\n", scope.CommitCount))
	sb.WriteString(fmt.Sprintf("merge_commit_count=%d\n", scope.MergeCommitCount))
	sb.WriteString(fmt.Sprintf("files_changed=%d\n", scope.FilesChanged))
	sb.WriteString(fmt.Sprintf("crosses_merge=%t\n", scope.CrossesMerge))
	sb.WriteString(fmt.Sprintf("targetedness=%s\n", scope.Targetedness))
	sb.WriteString(fmt.Sprintf("warning_code=%s\n", scope.WarningCode))
	sb.WriteString(fmt.Sprintf("diagnostic_status=%s\n", scope.DiagnosticStatus))
	if scope.Targetedness != RangeTargetednessNormal {
		sb.WriteString("\nwarning:\n")
		for _, line := range strings.Split(warningProse(scope.Targetedness), "\n") {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}
	return sb.String()
}

func renderUnavailableScope(rawRangeSpec, diagnosticError string) string {
	var sb strings.Builder
	sb.WriteString("## RANGE_SCOPE\n\nauthority=explicit_range\n")
	sb.WriteString(fmt.Sprintf("targetedness=%s\n", RangeTargetednessUnknown))
	sb.WriteString(fmt.Sprintf("warning_code=%s\n", WarningCodeDigestRangeScopeUnavailable))
	sb.WriteString(fmt.Sprintf("diagnostic_status=%s\n", DiagnosticStatusUnavailable))
	if rawRangeSpec != "" {
		sb.WriteString(fmt.Sprintf("raw_range_spec=%s\n", rawRangeSpec))
	}
	if diagnosticError != "" {
		sb.WriteString(fmt.Sprintf("diagnostic_error=%s\n", diagnosticError))
	}
	sb.WriteString("\nwarning:\n")
	sb.WriteString("  Range-scope diagnostic is unavailable.\n")
	sb.WriteString("  This advisory failure does not alter the selected digest range.\n")
	return sb.String()
}
