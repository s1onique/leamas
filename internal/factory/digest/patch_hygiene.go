// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
)

// RunPatchHygiene runs git diff --check and returns parsed results.
func RunPatchHygiene(repoRoot, rangeSpec string) PatchHygiene {
	var ph PatchHygiene

	// Run git diff --check
	args := []string{"diff", "--check"}
	if rangeSpec != "" {
		args = append(args, rangeSpec)
	}

	output, exitCode := RunGitWithExitCode(repoRoot, args)

	if exitCode == -1 {
		// Command unavailable
		ph.GitDiffCheck = PatchHygieneUnavailable
		if output != "" {
			ph.Diagnostics = []string{"git diff --check unavailable: " + truncateLine(output, MaxDiagnosticLineLength)}
			ph.DiagnosticLines = 1
		}
		return ph
	}

	// Parse diagnostics
	ph.Diagnostics = ParsePatchHygieneDiagnostics(output, repoRoot)
	ph.DiagnosticLines = len(ph.Diagnostics)

	// Count errors by type
	for _, diag := range ph.Diagnostics {
		if isConflictMarker(diag) {
			ph.ConflictMarkers++
		} else {
			ph.WhitespaceErrors++
		}
	}

	if exitCode != 0 || ph.DiagnosticLines > 0 {
		ph.GitDiffCheck = PatchHygieneFail
	} else {
		ph.GitDiffCheck = PatchHygienePass
	}

	return ph
}

// RunPatchHygieneDirty runs hygiene checks for dirty (unstaged + staged) patches.
func RunPatchHygieneDirty(repoRoot string) PatchHygiene {
	// Check staged changes
	stagedResult := RunPatchHygiene(repoRoot, "--cached")

	// Check unstaged changes
	unstagedResult := RunPatchHygiene(repoRoot, "")

	// Merge results deterministically
	return mergePatchHygiene(stagedResult, unstagedResult)
}

// mergePatchHygiene combines two hygiene results.
func mergePatchHygiene(a, b PatchHygiene) PatchHygiene {
	var result PatchHygiene

	// Determine overall status
	if a.GitDiffCheck == PatchHygieneFail || b.GitDiffCheck == PatchHygieneFail {
		result.GitDiffCheck = PatchHygieneFail
	} else if a.GitDiffCheck == PatchHygieneUnavailable || b.GitDiffCheck == PatchHygieneUnavailable {
		result.GitDiffCheck = PatchHygieneUnavailable
	} else {
		result.GitDiffCheck = PatchHygienePass
	}

	result.WhitespaceErrors = a.WhitespaceErrors + b.WhitespaceErrors
	result.ConflictMarkers = a.ConflictMarkers + b.ConflictMarkers

	// Merge diagnostics (deterministic: staged first, then unstaged)
	result.Diagnostics = append(a.Diagnostics, b.Diagnostics...)
	result.DiagnosticLines = len(result.Diagnostics)

	// Bound diagnostics
	if result.DiagnosticLines > MaxPatchHygieneDiagnostics {
		result.Diagnostics = result.Diagnostics[:MaxPatchHygieneDiagnostics]
		result.DiagnosticLines = len(result.Diagnostics)
	}

	return result
}

// ParsePatchHygieneDiagnostics parses git diff --check output into lines.
func ParsePatchHygieneDiagnostics(output, repoRoot string) []string {
	if output == "" {
		return nil
	}

	var diagnostics []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Normalize repo root in paths
		line = normalizeRepoRoot(line, repoRoot)

		// Truncate long lines
		line = truncateLine(line, MaxDiagnosticLineLength)

		diagnostics = append(diagnostics, line)

		// Bound total lines
		if len(diagnostics) >= MaxPatchHygieneDiagnostics {
			break
		}
	}

	return diagnostics
}

// isConflictMarker determines if a diagnostic is a conflict marker.
func isConflictMarker(diagnostic string) bool {
	return strings.Contains(diagnostic, "conflict marker")
}

// normalizeRepoRoot replaces absolute repo path with placeholder.
func normalizeRepoRoot(line, repoRoot string) string {
	if repoRoot == "" {
		return line
	}
	// Replace absolute path with relative
	idx := strings.Index(line, repoRoot)
	if idx >= 0 {
		return strings.Replace(line, repoRoot, "<repo>", 1)
	}
	return line
}

// truncateLine truncates a line to maxLen characters.
func truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen] + "..."
}

// RenderPatchHygiene renders the PATCH_HYGIENE section.
func RenderPatchHygiene(ph PatchHygiene) string {
	var sb strings.Builder
	sb.WriteString("## PATCH_HYGIENE\n")
	sb.WriteString("git_diff_check=")
	sb.WriteString(ph.GitDiffCheck)
	sb.WriteString("\nwhitespace_errors=")
	sb.WriteString(intToString(ph.WhitespaceErrors))
	sb.WriteString("\nconflict_markers=")
	sb.WriteString(intToString(ph.ConflictMarkers))
	sb.WriteString("\ndiagnostic_lines=")
	sb.WriteString(intToString(ph.DiagnosticLines))

	if ph.DiagnosticLines > 0 {
		sb.WriteString("\ndiagnostics:")
		for _, diag := range ph.Diagnostics {
			sb.WriteString("\n  - ")
			sb.WriteString(diag)
		}
	}

	sb.WriteString("\n")
	return sb.String()
}
