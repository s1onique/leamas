// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// sha256HexRegex matches valid SHA256 hex strings.
var sha256HexRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ValidateBaselinePaths validates all file paths in the baseline.
func ValidateBaselinePaths(baseline Baseline) []BaselineValidationFinding {
	var findings []BaselineValidationFinding

	for i, finding := range baseline.Findings {
		for j, occ := range finding.Occurrences {
			path := occ.Path

			// Check for absolute path
			if filepath.IsAbs(path) {
				findings = append(findings, BaselineValidationFinding{
					Path:     path,
					Kind:     "absolute_path_in_baseline",
					Message:  fmt.Sprintf("finding[%d].occurrences[%d]: absolute paths not allowed: %s", i, j, path),
					Severity: "error",
				})
			}

			// Check for backslashes (Windows-style paths)
			if strings.Contains(path, "\\") {
				findings = append(findings, BaselineValidationFinding{
					Path:     path,
					Kind:     "os_specific_path_in_baseline",
					Message:  fmt.Sprintf("finding[%d].occurrences[%d]: OS-specific backslashes not allowed: %s", i, j, path),
					Severity: "error",
				})
			}

			// Check for parent traversal
			parts := strings.Split(filepath.ToSlash(path), "/")
			for _, part := range parts {
				if part == ".." {
					findings = append(findings, BaselineValidationFinding{
						Path:    path,
						Kind:    "path_escapes_repo_root",
						Message: fmt.Sprintf("finding[%d].occurrences[%d]: path escapes repo root via '..': %s", i, j, path),
					})
					break
				}
			}

			// Check for empty path
			if path == "" {
				findings = append(findings, BaselineValidationFinding{
					Path:     path,
					Kind:     "empty_path_in_baseline",
					Message:  fmt.Sprintf("finding[%d].occurrences[%d]: empty path not allowed", i, j),
					Severity: "error",
				})
			}

			// Check for invalid line numbers
			if occ.StartLine <= 0 {
				findings = append(findings, BaselineValidationFinding{
					Path:     path,
					Kind:     "invalid_start_line",
					Message:  fmt.Sprintf("finding[%d].occurrences[%d]: start_line must be > 0, got %d", i, j, occ.StartLine),
					Severity: "error",
				})
			}

			if occ.EndLine <= 0 {
				findings = append(findings, BaselineValidationFinding{
					Path:     path,
					Kind:     "invalid_end_line",
					Message:  fmt.Sprintf("finding[%d].occurrences[%d]: end_line must be > 0, got %d", i, j, occ.EndLine),
					Severity: "error",
				})
			}

			if occ.EndLine < occ.StartLine {
				findings = append(findings, BaselineValidationFinding{
					Path:     path,
					Kind:     "end_line_before_start_line",
					Message:  fmt.Sprintf("finding[%d].occurrences[%d]: end_line (%d) < start_line (%d)", i, j, occ.EndLine, occ.StartLine),
					Severity: "error",
				})
			}
		}
	}

	return findings
}

// ValidateBaselineFingerprints validates all fingerprints in the baseline.
func ValidateBaselineFingerprints(baseline Baseline) []BaselineValidationFinding {
	var findings []BaselineValidationFinding
	seenFingerprints := make(map[string]int) // fingerprint -> first index

	for i, finding := range baseline.Findings {
		fp := finding.Fingerprint

		// Check for empty fingerprint
		if fp == "" {
			findings = append(findings, BaselineValidationFinding{
				Path:     "",
				Kind:     "empty_fingerprint",
				Message:  fmt.Sprintf("finding[%d]: fingerprint is empty", i),
				Severity: "error",
			})
			continue
		}

		// Check for valid SHA256 format
		if !sha256HexRegex.MatchString(fp) {
			findings = append(findings, BaselineValidationFinding{
				Path:     "",
				Kind:     "invalid_fingerprint_format",
				Message:  fmt.Sprintf("finding[%d]: fingerprint must be SHA256 hex (64 chars), got %q (len=%d)", i, fp, len(fp)),
				Severity: "error",
			})
		}

		// Check for duplicate fingerprints
		if prevIdx, exists := seenFingerprints[fp]; exists {
			findings = append(findings, BaselineValidationFinding{
				Path:     "",
				Kind:     "duplicate_fingerprint",
				Message:  fmt.Sprintf("finding[%d]: duplicate fingerprint %q (first seen at index %d)", i, fp, prevIdx),
				Severity: "error",
			})
		} else {
			seenFingerprints[fp] = i
		}
	}

	return findings
}

// ValidateBaselineOrdering validates the sorting order of findings and occurrences.
func ValidateBaselineOrdering(baseline Baseline) []BaselineValidationFinding {
	var findings []BaselineValidationFinding

	// Check findings are sorted by fingerprint
	for i := 1; i < len(baseline.Findings); i++ {
		if baseline.Findings[i].Fingerprint <= baseline.Findings[i-1].Fingerprint {
			findings = append(findings, BaselineValidationFinding{
				Path: "",
				Kind: "findings_not_sorted",
				Message: fmt.Sprintf("findings not sorted by fingerprint: finding[%d] (%q) <= finding[%d] (%q)",
					i, baseline.Findings[i].Fingerprint, i-1, baseline.Findings[i-1].Fingerprint),
				Severity: "error",
			})
			break // Report only first sorting issue
		}
	}

	// Check occurrences within each finding are sorted by path, then start_line, then end_line
	for i, finding := range baseline.Findings {
		for j := 1; j < len(finding.Occurrences); j++ {
			prev := finding.Occurrences[j-1]
			curr := finding.Occurrences[j]

			if curr.Path < prev.Path ||
				(curr.Path == prev.Path && curr.StartLine < prev.StartLine) ||
				(curr.Path == prev.Path && curr.StartLine == prev.StartLine && curr.EndLine < prev.EndLine) {
				findings = append(findings, BaselineValidationFinding{
					Path: prev.Path,
					Kind: "occurrences_not_sorted",
					Message: fmt.Sprintf("finding[%d].occurrences not sorted: occurrences[%d] (%s:%d-%d) should come before occurrences[%d] (%s:%d-%d)",
						i, j, curr.Path, curr.StartLine, curr.EndLine, j-1, prev.Path, prev.StartLine, prev.EndLine),
				})
				break
			}
		}
	}

	return findings
}
