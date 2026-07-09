// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// DefaultPolicyMinLines is the expected minimum lines for a duplicate block.
const DefaultPolicyMinLines = 40

// DefaultPolicyMinTokens is the expected minimum tokens for a duplicate block.
const DefaultPolicyMinTokens = 400

// BaselinePolicy defines the policy requirements for a valid baseline.
type BaselinePolicy struct {
	Path      string
	MinLines  int
	MinTokens int
}

// DefaultBaselinePolicy returns the default baseline policy.
func DefaultBaselinePolicy() BaselinePolicy {
	return BaselinePolicy{
		Path:      ".factory/dupcode-baseline.json",
		MinLines:  DefaultPolicyMinLines,
		MinTokens: DefaultPolicyMinTokens,
	}
}

// BaselineValidationFinding represents a single baseline validation issue.
type BaselineValidationFinding struct {
	Path     string
	Kind     string
	Message  string
	Severity string
}

// BaselineValidationResult contains the results of baseline validation.
type BaselineValidationResult struct {
	Findings []BaselineValidationFinding
}

// NormalizeOccurrencePath normalizes a path to be repo-relative with forward slashes.
func NormalizeOccurrencePath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(p)
}

// VerifyBaseline verifies the baseline artifact against policy requirements.
func VerifyBaseline(root string, policy BaselinePolicy) ([]checks.Finding, error) {
	var result []checks.Finding

	// Separate FS path (absolute/relative to root) from Git path (always repo-relative)
	baselineFSPath := policy.Path
	baselineGitPath := policy.Path
	if root != "" && root != "." {
		baselineFSPath = filepath.Join(root, policy.Path)
	}

	// 1. Check baseline presence
	if _, err := os.Stat(baselineFSPath); os.IsNotExist(err) {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "missing_dupcode_baseline",
			Message:  "baseline file not found; run 'make dupcode-baseline' to create",
			Severity: checks.SeverityError,
		})
		return result, nil
	}

	// 2. Check git tracking using repo-relative path
	if err := CheckBaselineTracked(root, baselineGitPath); err != nil {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "untracked_dupcode_baseline",
			Message:  err.Error(),
			Severity: checks.SeverityError,
		})
		return result, nil
	}

	// 3. Load baseline
	baseline, err := LoadBaseline(baselineFSPath)
	if err != nil {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "invalid_dupcode_baseline",
			Message:  fmt.Sprintf("failed to load baseline: %v", err),
			Severity: checks.SeverityError,
		})
		return result, nil
	}

	// 4. Validate schema version
	if baseline.SchemaVersion != 1 {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     "unsupported_schema_version",
			Message:  fmt.Sprintf("schema version %d not supported; expected 1", baseline.SchemaVersion),
			Severity: checks.SeverityError,
		})
	}

	// 5. Validate threshold policy
	if baseline.Thresholds.MinLines != policy.MinLines || baseline.Thresholds.MinTokens != policy.MinTokens {
		result = append(result, checks.Finding{
			Path: baselineFSPath,
			Kind: "threshold_policy_mismatch",
			Message: fmt.Sprintf("baseline thresholds %d/%d do not match policy %d/%d",
				baseline.Thresholds.MinLines, baseline.Thresholds.MinTokens,
				policy.MinLines, policy.MinTokens),
			Severity: checks.SeverityError,
		})
	}

	// 6. Validate paths
	pathFindings := ValidateBaselinePaths(baseline)
	for _, f := range pathFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		})
	}

	// 7. Validate fingerprints
	fpFindings := ValidateBaselineFingerprints(baseline)
	for _, f := range fpFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		})
	}

	// 8. Validate ordering
	orderFindings := ValidateBaselineOrdering(baseline)
	for _, f := range orderFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		})
	}

	// 9. Check for drift
	driftFindings := CheckBaselineDrift(root, baseline, policy)
	for _, f := range driftFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		})
	}

	return result, nil
}

// CheckBaselineTracked verifies the baseline is tracked by git.
func CheckBaselineTracked(root, baselinePath string) error {
	// Determine the working directory for the git command
	workDir := "."
	if root != "" && root != "." {
		workDir = root
	}

	cmd := exec.Command("git", "ls-files", "--error-unmatch", baselinePath)
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("baseline not tracked by git; ensure %s is committed", baselinePath)
	}
	return nil
}

// CheckBaselineDrift checks if the committed baseline is stale compared to current scanner output.
func CheckBaselineDrift(root string, committedBaseline Baseline, policy BaselinePolicy) []BaselineValidationFinding {
	var findings []BaselineValidationFinding

	// Run current scanner with policy thresholds
	cfg := DefaultConfig()
	cfg.MinLines = policy.MinLines
	cfg.MinTokens = policy.MinTokens

	currentReport, err := CheckReport(root, cfg)
	if err != nil {
		findings = append(findings, BaselineValidationFinding{
			Path:     policy.Path,
			Kind:     "drift_check_scanner_error",
			Message:  fmt.Sprintf("failed to run scanner for drift check: %v", err),
			Severity: "error",
		})
		return findings
	}

	// Generate a canonical baseline with deterministic timestamp
	deterministicBaseline := GenerateCanonicalBaseline(root, currentReport)

	// Compare findings (ignoring generated_at)
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

// GenerateCanonicalBaseline creates a baseline with a deterministic timestamp for comparison.
func GenerateCanonicalBaseline(root string, report Report) Baseline {
	// Normalize root to "." if empty for consistent path handling
	normRoot := "."
	if root != "" && root != "." {
		normRoot = root
	}

	findings := make([]BaselineFinding, 0, len(report.Findings))
	for _, f := range report.Findings {
		occurrences := make([]BaselineOccurrence, 0, len(f.Occurrences))
		for _, occ := range f.Occurrences {
			occurrences = append(occurrences, BaselineOccurrence{
				Path:      NormalizeOccurrencePath(normRoot, occ.Path),
				StartLine: occ.StartLine,
				EndLine:   occ.EndLine,
			})
		}

		fp := f.Fingerprint
		if f.StableFingerprint != "" {
			fp = f.StableFingerprint
		}

		findings = append(findings, BaselineFinding{
			Fingerprint: fp,
			TokenCount:  f.TokenCount,
			LineCount:   f.LineCount,
			Occurrences: occurrences,
		})
	}

	// Sort findings by fingerprint
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Fingerprint < findings[j].Fingerprint
	})

	// Sort occurrences within each finding by path, start_line, end_line
	for i := range findings {
		sort.Slice(findings[i].Occurrences, func(a, b int) bool {
			occA := findings[i].Occurrences[a]
			occB := findings[i].Occurrences[b]
			if occA.Path != occB.Path {
				return occA.Path < occB.Path
			}
			if occA.StartLine != occB.StartLine {
				return occA.StartLine < occB.StartLine
			}
			return occA.EndLine < occB.EndLine
		})
	}

	return Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "1970-01-01T00:00:00Z", // Deterministic timestamp
		Tool:          "leamas dupcode",
		Thresholds:    report.Thresholds,
		Findings:      findings,
	}
}

// baselinesEqual compares two baselines for equality (ignoring generated_at).
func baselinesEqual(a, b Baseline) bool {
	if a.SchemaVersion != b.SchemaVersion {
		return false
	}
	if a.Tool != b.Tool {
		return false
	}
	if a.Thresholds.MinLines != b.Thresholds.MinLines || a.Thresholds.MinTokens != b.Thresholds.MinTokens {
		return false
	}
	if len(a.Findings) != len(b.Findings) {
		return false
	}
	for i := range a.Findings {
		if !findingEqual(a.Findings[i], b.Findings[i]) {
			return false
		}
	}
	return true
}

// findingEqual compares two baseline findings for equality.
func findingEqual(a, b BaselineFinding) bool {
	if a.Fingerprint != b.Fingerprint {
		return false
	}
	if a.TokenCount != b.TokenCount {
		return false
	}
	if a.LineCount != b.LineCount {
		return false
	}
	if len(a.Occurrences) != len(b.Occurrences) {
		return false
	}
	for i := range a.Occurrences {
		if a.Occurrences[i] != b.Occurrences[i] {
			return false
		}
	}
	return true
}

// PrintBaselineVerifyResult prints the result of baseline verification.
func PrintBaselineVerifyResult(name string, findings []checks.Finding) int {
	if len(findings) == 0 {
		fmt.Printf("%s: OK\n", name)
		return 0
	}

	fmt.Printf("%s: FAILED\n", name)
	for _, f := range findings {
		fmt.Printf("  %s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
	return 1
}
