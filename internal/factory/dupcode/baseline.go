// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Policy thresholds for duplicate code detection.
// These define the expected gate policy.
const (
	// PolicyMinLines is the expected minimum lines for a duplicate block.
	PolicyMinLines = 40
	// PolicyMinTokens is the expected minimum tokens for a duplicate block.
	PolicyMinTokens = 400
)

// ErrBaselinePolicyMismatch indicates the baseline thresholds don't match policy.
var ErrBaselinePolicyMismatch = errors.New("baseline thresholds do not match policy (40/400)")

// Baseline represents a committed baseline of duplicate code findings.
type Baseline struct {
	SchemaVersion    int                `json:"schema_version"`
	AlgorithmVersion int                `json:"algorithm_version,omitempty"`
	GeneratedAt      string             `json:"generated_at"`
	Tool             string             `json:"tool"`
	Thresholds       BaselineThresholds `json:"thresholds"`
	Findings         []BaselineFinding  `json:"findings"`
}

// BaselineThresholds records the thresholds used when generating the baseline.
type BaselineThresholds struct {
	MinLines  int `json:"min_lines"`
	MinTokens int `json:"min_tokens"`
}

// BaselineFinding represents a single duplicate block in the baseline.
type BaselineFinding struct {
	Fingerprint string               `json:"fingerprint"`
	TokenCount  int                  `json:"token_count"`
	LineCount   int                  `json:"line_count"`
	Occurrences []BaselineOccurrence `json:"occurrences"`
}

// BaselineOccurrence represents a location of a baseline finding.
type BaselineOccurrence struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// CompareResult contains the comparison between current findings and baseline.
type CompareResult struct {
	NewFindings      []NewFinding      `json:"new_findings,omitempty"`
	WorsenedFindings []WorsenedFinding `json:"worsened_findings,omitempty"`
	HasChanges       bool              `json:"has_changes"`
}

// NewFinding represents a duplicate fingerprint that is new (not in baseline).
type NewFinding struct {
	Fingerprint string               `json:"fingerprint"`
	TokenCount  int                  `json:"token_count"`
	LineCount   int                  `json:"line_count"`
	Occurrences []BaselineOccurrence `json:"occurrences"`
}

// WorsenedFinding represents a baseline fingerprint that has new occurrence locations.
type WorsenedFinding struct {
	Fingerprint         string               `json:"fingerprint"`
	BaselineOccurrences []BaselineOccurrence `json:"baseline_occurrences"`
	NewOccurrences      []BaselineOccurrence `json:"new_occurrences"`
	TotalNow            int                  `json:"total_now"`
}

// Report contains the results of a duplicate code scan.
type Report struct {
	Findings   []Finding          `json:"findings"`
	Thresholds BaselineThresholds `json:"thresholds"`
	Root       string             `json:"root,omitempty"`
}

// ErrInvalidBaseline is returned when baseline validation fails.
type ErrInvalidBaseline struct {
	Reason error
}

func (e *ErrInvalidBaseline) Error() string {
	return fmt.Sprintf("invalid baseline: %v", e.Reason)
}

func (e *ErrInvalidBaseline) Unwrap() error {
	return e.Reason
}

// LoadBaseline loads a baseline from a JSON file.
func LoadBaseline(path string) (Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Baseline{}, fmt.Errorf("loading baseline: %w", err)
	}

	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return Baseline{}, &ErrInvalidBaseline{fmt.Errorf("parse error: %w", err)}
	}

	if baseline.SchemaVersion != 1 {
		return Baseline{}, &ErrInvalidBaseline{fmt.Errorf("unsupported schema version: %d", baseline.SchemaVersion)}
	}

	// Validate algorithm version
	if baseline.AlgorithmVersion == 0 {
		return Baseline{}, &ErrInvalidBaseline{
			fmt.Errorf("missing algorithm_version: old baseline format not supported; regenerate with 'make dupcode-baseline'"),
		}
	}
	if baseline.AlgorithmVersion != AlgorithmVersion {
		return Baseline{}, &ErrInvalidBaseline{
			fmt.Errorf("algorithm_version mismatch: baseline has %d, current detector uses %d; regenerate with 'make dupcode-baseline'",
				baseline.AlgorithmVersion, AlgorithmVersion),
		}
	}

	// Validate thresholds match policy
	if baseline.Thresholds.MinLines != PolicyMinLines || baseline.Thresholds.MinTokens != PolicyMinTokens {
		return Baseline{}, &ErrInvalidBaseline{
			fmt.Errorf("threshold mismatch: got %d/%d, expected %d/%d",
				baseline.Thresholds.MinLines, baseline.Thresholds.MinTokens,
				PolicyMinLines, PolicyMinTokens),
		}
	}

	return baseline, nil
}

// BaselineWriter allows injecting time for deterministic testing.
type BaselineWriter struct {
	Now func() time.Time
}

// NewBaselineWriter creates a BaselineWriter with the current time.
func NewBaselineWriter() *BaselineWriter {
	return &BaselineWriter{Now: time.Now}
}

// WriteBaseline writes a report to a baseline JSON file.
func WriteBaseline(path string, report Report) error {
	bw := &BaselineWriter{Now: time.Now}
	return bw.Write(path, report)
}

// Write writes a report to a baseline JSON file with the given timestamp.
func (bw *BaselineWriter) Write(path string, report Report) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating baseline directory: %w", err)
	}

	// Normalize and build baseline from report
	findings := make([]BaselineFinding, 0, len(report.Findings))
	for _, f := range report.Findings {
		occurrences := make([]BaselineOccurrence, 0, len(f.Occurrences))
		for _, occ := range f.Occurrences {
			// Normalize path to repo-relative with forward slashes
			normalizedPath := filepath.ToSlash(occ.Path)
			occurrences = append(occurrences, BaselineOccurrence{
				Path:      normalizedPath,
				StartLine: occ.StartLine,
				EndLine:   occ.EndLine,
			})
		}
		// Use StableFingerprint for baseline storage (not truncated display fingerprint)
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

	// Sort findings deterministically by fingerprint
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Fingerprint < findings[j].Fingerprint
	})

	baseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      bw.Now().UTC().Format(time.RFC3339),
		Tool:             "leamas dupcode",
		Thresholds:       report.Thresholds,
		Findings:         findings,
	}

	// Sort occurrences within each finding
	for i := range baseline.Findings {
		sort.Slice(baseline.Findings[i].Occurrences, func(a, b int) bool {
			if baseline.Findings[i].Occurrences[a].Path != baseline.Findings[i].Occurrences[b].Path {
				return baseline.Findings[i].Occurrences[a].Path < baseline.Findings[i].Occurrences[b].Path
			}
			return baseline.Findings[i].Occurrences[a].StartLine < baseline.Findings[i].Occurrences[b].StartLine
		})
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing baseline file: %w", err)
	}

	return nil
}

// CompareToBaseline compares current findings against a baseline.
// Fingerprint identity is the primary key. Occurrence comparison uses path+count
// to avoid false positives from line number shifts.
func CompareToBaseline(report Report, baseline Baseline) CompareResult {
	result := CompareResult{}

	// Build lookup maps from baseline - index by stable fingerprint
	baselineByFP := make(map[string]BaselineFinding)
	for _, bf := range baseline.Findings {
		baselineByFP[bf.Fingerprint] = bf
	}

	// Check each current finding
	for _, finding := range report.Findings {
		// Use stable fingerprint for matching
		matchingFP := finding.StableFingerprint
		if matchingFP == "" {
			matchingFP = finding.Fingerprint
		}

		baselineFinding, exists := baselineByFP[matchingFP]

		if !exists {
			// New fingerprint not in baseline
			occurrences := make([]BaselineOccurrence, 0, len(finding.Occurrences))
			for _, occ := range finding.Occurrences {
				occurrences = append(occurrences, BaselineOccurrence{
					Path:      filepath.ToSlash(occ.Path),
					StartLine: occ.StartLine,
					EndLine:   occ.EndLine,
				})
			}
			result.NewFindings = append(result.NewFindings, NewFinding{
				Fingerprint: matchingFP,
				TokenCount:  finding.TokenCount,
				LineCount:   finding.LineCount,
				Occurrences: occurrences,
			})
			result.HasChanges = true
		} else {
			// Fingerprint exists in baseline - check for worsened
			// Group occurrences by path for comparison
			baselineOccByPath := make(map[string][]BaselineOccurrence)
			for _, occ := range baselineFinding.Occurrences {
				path := filepath.ToSlash(occ.Path)
				baselineOccByPath[path] = append(baselineOccByPath[path], occ)
			}

			// Count baseline occurrences per path
			baselineCountByPath := make(map[string]int)
			for path, occs := range baselineOccByPath {
				baselineCountByPath[path] = len(occs)
			}

			// Count current occurrences per path
			currentOccByPath := make(map[string][]Occurrence)
			for _, occ := range finding.Occurrences {
				path := filepath.ToSlash(occ.Path)
				currentOccByPath[path] = append(currentOccByPath[path], occ)
			}

			// Find new occurrences by path (only if count increases)
			var newOccurrences []BaselineOccurrence
			for path, currentOccs := range currentOccByPath {
				baselineCount := baselineCountByPath[path]
				currentCount := len(currentOccs)

				if currentCount > baselineCount {
					// Only mark the delta as new
					delta := currentCount - baselineCount
					for i := 0; i < delta && i < len(currentOccs); i++ {
						newOccurrences = append(newOccurrences, BaselineOccurrence{
							Path:      path,
							StartLine: currentOccs[i].StartLine,
							EndLine:   currentOccs[i].EndLine,
						})
					}
				}
			}

			if len(newOccurrences) > 0 {
				result.WorsenedFindings = append(result.WorsenedFindings, WorsenedFinding{
					Fingerprint:         matchingFP,
					BaselineOccurrences: baselineFinding.Occurrences,
					NewOccurrences:      newOccurrences,
					TotalNow:            len(finding.Occurrences),
				})
				result.HasChanges = true
			}
		}
	}

	return result
}

// StableFingerprintHash returns a stable hash of a normalized fingerprint string.
func StableFingerprintHash(normalized string) string {
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// NormalizePathForBaseline converts a path to repo-relative with forward slashes.
func NormalizePathForBaseline(path, root string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	if filepath.IsAbs(rel) {
		return path
	}
	return filepath.ToSlash(rel)
}

// CheckReport scans the repository and returns a full report.
func CheckReport(root string, cfg Config) (Report, error) {
	findings, err := CheckRepo(root, cfg)
	if err != nil {
		return Report{}, err
	}

	// Paths are already normalized in CheckRepo, no need to normalize again.
	return Report{
		Findings: findings,
		Thresholds: BaselineThresholds{
			MinLines:  cfg.MinLines,
			MinTokens: cfg.MinTokens,
		},
		Root: root,
	}, nil
}
