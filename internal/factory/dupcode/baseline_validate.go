// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// sha256HexRegex matches valid SHA256 hex strings.
var sha256HexRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)

// BaselineArtifactValidation contains the results of baseline artifact validation.
type BaselineArtifactValidation struct {
	Baseline       Baseline
	Findings       []checks.Finding
	UsableForDrift bool
}

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

// ValidateBaselineArtifact performs static validation of the baseline artifact
// without scanning the repository. This is used by the factorize path where
// shared analysis is used.
//
// Returns a BaselineArtifactValidation with the baseline, findings, and drift eligibility.
// UsableForDrift is false for terminal conditions:
//   - missing, untracked, symlink, non-regular file
//   - stat/lstat error (returned as error)
//   - JSON parse error (invalid_dupcode_baseline finding)
//   - schema/algorithm/threshold mismatch (invalid_dupcode_baseline finding)
//
// An error is returned for stat/lstat failures (e.g., permission denied).
// All other validation failures are returned as findings with UsableForDrift=false.
func ValidateBaselineArtifact(root string, policy BaselinePolicy) (BaselineArtifactValidation, error) {
	var result []checks.Finding
	var baseline Baseline

	// Separate FS path (absolute/relative to root) from Git path (always repo-relative)
	baselineFSPath := policy.Path
	baselineGitPath := policy.Path
	if root != "" && root != "." {
		baselineFSPath = filepath.Join(root, policy.Path)
	}

	// 1. Check baseline presence and type using Lstat to detect symlinks
	info, err := os.Lstat(baselineFSPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "missing_dupcode_baseline",
			Message:  "baseline file not found; run 'make dupcode-baseline' to create",
			Severity: checks.SeverityError,
		})
		return BaselineArtifactValidation{Findings: result, UsableForDrift: false}, nil
	case err != nil:
		return BaselineArtifactValidation{}, fmt.Errorf("stat dupcode baseline %q: %w", baselineFSPath, err)
	case info.Mode()&os.ModeSymlink != 0:
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "symlink_not_allowed",
			Message:  "baseline path cannot be a symbolic link",
			Severity: checks.SeverityError,
		})
		return BaselineArtifactValidation{Findings: result, UsableForDrift: false}, nil
	case !info.Mode().IsRegular():
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "invalid_baseline_type",
			Message:  fmt.Sprintf("baseline path is not a regular file: %s", info.Mode().String()),
			Severity: checks.SeverityError,
		})
		return BaselineArtifactValidation{Findings: result, UsableForDrift: false}, nil
	}

	// 2. Check git tracking using repo-relative path
	if err := CheckBaselineTracked(root, baselineGitPath); err != nil {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "untracked_dupcode_baseline",
			Message:  err.Error(),
			Severity: checks.SeverityError,
		})
		return BaselineArtifactValidation{Findings: result, UsableForDrift: false}, nil
	}

	// 3. Decode baseline (no policy validation yet - we perform our own)
	baseline, err = decodeBaselineArtifact(baselineFSPath)
	if err != nil {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "invalid_dupcode_baseline",
			Message:  fmt.Sprintf("failed to load baseline: %v", err),
			Severity: checks.SeverityError,
		})
		return BaselineArtifactValidation{Findings: result, UsableForDrift: false}, nil
	}

	// 4. Validate compatibility using the shared authority (terminal - preserves historical behavior)
	if err := validateBaselineCompatibility(baseline); err != nil {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "invalid_dupcode_baseline",
			Message:  fmt.Sprintf("failed to load baseline: %v", err),
			Severity: checks.SeverityError,
		})
		return BaselineArtifactValidation{Findings: result, UsableForDrift: false}, nil
	}

	// 4a. Validate caller-supplied policy thresholds (non-terminal)
	// This preserves historical VerifyBaseline behavior: custom policy thresholds
	// produce a finding but the baseline remains usable for drift comparison.
	if baseline.Thresholds.MinLines != policy.MinLines ||
		baseline.Thresholds.MinTokens != policy.MinTokens {
		result = append(result, checks.Finding{
			Path: baselineFSPath,
			Kind: "threshold_policy_mismatch",
			Message: fmt.Sprintf("baseline thresholds %d/%d do not match policy %d/%d",
				baseline.Thresholds.MinLines, baseline.Thresholds.MinTokens,
				policy.MinLines, policy.MinTokens),
			Severity: checks.SeverityError,
		})
		// Baseline remains usable for drift - caller policy mismatch is non-terminal
	}

	// 5. Validate paths (non-terminal)
	pathFindings := ValidateBaselinePaths(baseline)
	for _, f := range pathFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		})
	}

	// 7. Validate fingerprints (non-terminal)
	fpFindings := ValidateBaselineFingerprints(baseline)
	for _, f := range fpFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		})
	}

	// 8. Validate ordering (non-terminal)
	orderFindings := ValidateBaselineOrdering(baseline)
	for _, f := range orderFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		})
	}

	// Baseline is loadable and valid for drift comparison (all terminal checks passed)
	return BaselineArtifactValidation{
		Baseline:       baseline,
		Findings:       result,
		UsableForDrift: true,
	}, nil
}

// CheckBaselineDriftFromReport checks if the committed baseline is stale compared to
// current scanner output. The caller provides the current report to avoid redundant scanning.
func CheckBaselineDriftFromReport(root string, committedBaseline Baseline, report Report, policy BaselinePolicy) []BaselineValidationFinding {
	var findings []BaselineValidationFinding

	// Generate a canonical baseline with deterministic timestamp from the provided report
	deterministicBaseline := GenerateCanonicalBaseline(root, report)

	// Compare findings (ignoring generated_at) using the package's baselinesEqual
	if !baselinesEqual(committedBaseline, deterministicBaseline) {
		findings = append(findings, BaselineValidationFinding{
			Path:     policy.Path,
			Kind:     "dupcode_baseline_drift",
			Message:  "dupcode baseline is stale; run 'make dupcode-baseline' and review the diff",
			Severity: "error",
		})
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
