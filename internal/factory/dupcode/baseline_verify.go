// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"

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

// NormalizeOccurrencePath normalizes a path to be repo-relative with forward
// slashes. The result is slash-normalized and safe to embed in baseline JSON.
//
// Containment uses filepath.IsLocal, which lexically guarantees the relative
// path is nonempty, non-absolute, and contains no parent-directory
// components. This is a true path-component check; a prefix test against
// ".." would incorrectly reject legitimate local names such as
// "..generated.go" that happen to start with ".." but are not escapes.
//
// When the path is outside root, Rel returns a "../..." form which IsLocal
// rejects. The function then returns the slash-normalized original as a
// fallback so callers can still record the absolute location.
func NormalizeOccurrencePath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err == nil && filepath.IsLocal(rel) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(p)
}

// VerifyBaseline verifies the baseline artifact against policy requirements.
// This composes ValidateBaselineArtifact for static validation and
// CheckBaselineDrift for dynamic drift checking.
func VerifyBaseline(root string, policy BaselinePolicy) ([]checks.Finding, error) {
	var result []checks.Finding

	// Compute root-aware FS path for finding paths
	baselineFSPath := policy.Path
	if root != "" && root != "." {
		baselineFSPath = filepath.Join(root, policy.Path)
	}

	// 1. Static validation: presence, tracking, loading, schema, algorithm, thresholds, paths, fingerprints, ordering
	validation, err := ValidateBaselineArtifact(root, policy)
	result = append(result, validation.Findings...)
	if err != nil {
		return result, err
	}

	// Only run drift check if baseline is usable for drift comparison
	// Terminal failures (missing, untracked, symlink, non-regular, stat error, malformed)
	// skip the drift check to avoid unnecessary repository scanning.
	if !validation.UsableForDrift {
		return result, nil
	}

	// 2. Dynamic drift check: run scanner and compare against committed baseline
	driftFindings := CheckBaselineDrift(root, validation.Baseline, policy)
	for _, f := range driftFindings {
		result = append(result, checks.Finding{
			Path:     baselineFSPath, // Use root-aware path
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
// This is a thin wrapper that runs the scanner and delegates to CheckBaselineDriftFromReport
// for the actual drift comparison, ensuring a single drift-comparison authority.
func CheckBaselineDrift(root string, committedBaseline Baseline, policy BaselinePolicy) []BaselineValidationFinding {
	var findings []BaselineValidationFinding

	// Run current scanner with policy thresholds
	cfg := DefaultConfig()
	cfg.Root = root
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

	// Delegate drift comparison to the single authority
	return CheckBaselineDriftFromReport(root, committedBaseline, currentReport, policy)
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
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "1970-01-01T00:00:00Z", // Deterministic timestamp
		Tool:             "leamas dupcode",
		Thresholds:       report.Thresholds,
		Findings:         findings,
	}
}

// baselinesEqual compares two baselines for equality (ignoring generated_at).
func baselinesEqual(a, b Baseline) bool {
	if a.SchemaVersion != b.SchemaVersion {
		return false
	}
	if a.AlgorithmVersion != b.AlgorithmVersion {
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
