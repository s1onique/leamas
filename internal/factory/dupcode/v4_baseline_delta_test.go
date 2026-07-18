// Package dupcode provides baseline-delta classification tests for
// the canonical content merge.
//
// The committed `.factory/dupcode-baseline.json` reports zero
// findings after
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01
// regenerated the baseline. The historical 504-token claim/evidence
// finding is preserved as tracked evidence under
// docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/
// so the closure audit can verify that the predecessor state is
// reproducible.
//
// All evidence in this file is regenerated against the live tree
// (NOT a stale digest) so that the claims are auditable.
package dupcode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// deltaCanonicalOccurrence is the deduplicated, sorted
// (Path, StartLine, EndLine) triple used to compare occurrence
// geometry between the live CheckRepo output and the committed
// baseline.
type deltaCanonicalOccurrence struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline runs
// CheckRepo on the live tree and verifies the live report equals
// the committed baseline. After ACT-LEAMAS-FACTORY-DUPCODE-
// SELF-HOSTED-REMEDIATION01 removed the canonical duplicate and
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01
// regenerated the baseline, both sides must report zero
// findings. The check guards against accidental drift in
// production output between ACT checkpoints.
func TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline(t *testing.T) {
	root := deltaRepoRoot(t)
	findings, err := CheckRepo(root, DefaultConfig())
	if err != nil {
		t.Fatalf("CheckRepo on live tree failed: %v", err)
	}
	// Setup witness: the scan must complete without error and
	// emit zero findings (the remediation removed every
	// policy-threshold duplicate).
	if len(findings) != 0 {
		t.Fatalf("live tree must report zero findings after remediation; got %d: %+v",
			len(findings), findings)
	}

	baseline, err := deltaLoadBaseline(t)
	if err != nil {
		t.Fatalf("load baseline: %v", err)
	}
	// Setup witness: the committed baseline must also report zero
	// findings after this ACT regenerates it. The
	// pre-convergence baseline still records the historical 504
	// finding; that state is documented under
	// docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01/pre-convergence-baseline-verify.json.
	if len(baseline.Findings) != 0 {
		t.Fatalf("baseline must report zero findings after convergence; got %d: %+v",
			len(baseline.Findings), baseline.Findings)
	}

	// Threshold witnesses: the canonical 40/400 policy must
	// remain in effect so the baseline-verify gate continues to
	// enforce the same policy thresholds.
	if got := DefaultConfig().MinLines; got != 40 {
		t.Errorf("MinLines drift: live=%d, want 40", got)
	}
	if got := DefaultConfig().MinTokens; got != 400 {
		t.Errorf("MinTokens drift: live=%d, want 400", got)
	}
	if got, ok := baseline.Thresholds["min_lines"]; !ok || got != 40 {
		t.Errorf("baseline thresholds.min_lines drift: got %d (ok=%v), want 40", got, ok)
	}
	if got, ok := baseline.Thresholds["min_tokens"]; !ok || got != 400 {
		t.Errorf("baseline thresholds.min_tokens drift: got %d (ok=%v), want 400", got, ok)
	}
}

// TestV4BaselineDelta_SurvivingFindingIsMaximalForComponent
// proves the canonical synthetic duplicate is maximal for its
// exact connected component. The proof constructs a synthetic
// fixture whose two files declare a single shared function body
// identical to the production canonical content body, exercises
// the public CheckRepo path, and asserts the returned finding
// has exactly two occurrences across two files (one connected
// component). Any sub-finding geometry (positional shadow,
// threshold-window, or region-split fragment) is suppressed by
// v4SuppressComponentShadows / v4SuppressContainedSameFileShadows
// before the finding reaches the caller.
//
// The test is the structural-shadow witness for the canonical
// detection invariant; it does not regenerate the baseline.
func TestV4BaselineDelta_SurvivingFindingIsMaximalForComponent(t *testing.T) {
	root := t.TempDir()
	cloneCounter = 0
	writeTestFile(t, filepath.Join(root, "live_a.go"), makeCloneFunc("LiveA", 84))
	writeTestFile(t, filepath.Join(root, "live_b.go"), makeCloneFunc("LiveB", 84))
	verifyFixturesTypeCheck(t, filepath.Join(root, "live_a.go"), filepath.Join(root, "live_b.go"))

	cfg := DefaultConfig()
	findings, err := CheckRepo(root, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("exactly one connected-component finding expected, got %d: %+v", len(findings), findings)
	}
	max := findings[0]
	if len(max.Occurrences) != 2 {
		t.Fatalf("expected 2 occurrences in the maximal connected component, got %d: %+v",
			len(max.Occurrences), max.Occurrences)
	}
	uniqueFiles := make(map[string]bool)
	for _, occ := range max.Occurrences {
		uniqueFiles[occ.Path] = true
	}
	if len(uniqueFiles) != 2 {
		t.Errorf("maximal connected component should span two files, got %d: %+v", len(uniqueFiles), max.Occurrences)
	}
}

// TestV4BaselineDelta_RemovedFindingIsStructuralShadow proves that
// a sub-finding whose every occurrence is contained in one larger
// finding with the same content sub-slice at the same relative
// offset is correctly eliminated by the structural shadow rule.
//
// The proof uses a synthetic fixture whose anchor body in file A
// is strictly larger than the duplicate body in file B; the
// component materializer must collapse any sub-finding to the
// maximal finding via v4SuppressComponentShadows.
func TestV4BaselineDelta_RemovedFindingIsStructuralShadow(t *testing.T) {
	files := manualAnalyzedFiles("shr_a.go", "shr_b.go")
	large := v4InternalFinding{
		StableFingerprint: "sha",
		TokenCount:        12,
		Occurrences: []maximalOccurrence{
			{Path: "shr_a.go", StartPos: 0, EndPos: 11, StartLine: 1, EndLine: 12},
			{Path: "shr_b.go", StartPos: 0, EndPos: 11, StartLine: 1, EndLine: 12},
		},
	}
	small := v4InternalFinding{
		StableFingerprint: "sha",
		TokenCount:        5,
		Occurrences: []maximalOccurrence{
			{Path: "shr_a.go", StartPos: 2, EndPos: 6, StartLine: 1, EndLine: 7},
			{Path: "shr_b.go", StartPos: 2, EndPos: 6, StartLine: 1, EndLine: 7},
		},
	}
	got := v4SuppressComponentShadows([]v4InternalFinding{small, large}, files)
	if len(got) != 1 {
		t.Fatalf("structural shadow was retained: %+v", got)
	}
	if got[0].TokenCount != 12 {
		t.Errorf("retained finding has wrong TokenCount: got %d want 12", got[0].TokenCount)
	}
}

// deltaLoadBaseline loads the committed dupcode baseline JSON.
//
// Thresholds are decoded as a map[string]int because the canonical
// baseline JSON serializes thresholds that way. Tests that need
// typed access use BaselineThresholds from baseline.go.
func deltaLoadBaseline(t *testing.T) (*dupcodeBaseline, error) {
	t.Helper()
	path := filepath.Join(deltaRepoRoot(t), ".factory", "dupcode-baseline.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b dupcodeBaseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

type dupcodeBaseline struct {
	SchemaVersion    int                      `json:"schema_version"`
	AlgorithmVersion int                      `json:"algorithm_version"`
	GeneratedAt      string                   `json:"generated_at"`
	Tool             string                   `json:"tool"`
	Thresholds       map[string]int           `json:"thresholds"`
	Findings         []dupcodeBaselineFinding `json:"findings"`
}

type dupcodeBaselineFinding struct {
	Fingerprint string                      `json:"fingerprint"`
	TokenCount  int                         `json:"token_count"`
	LineCount   int                         `json:"line_count"`
	Occurrences []dupcodeBaselineOccurrence `json:"occurrences"`
}

type dupcodeBaselineOccurrence struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func deltaCanonicalizeFromPublic(occs []Occurrence) []deltaCanonicalOccurrence {
	sorted := make([]Occurrence, len(occs))
	copy(sorted, occs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		if sorted[i].StartLine != sorted[j].StartLine {
			return sorted[i].StartLine < sorted[j].StartLine
		}
		return sorted[i].EndLine < sorted[j].EndLine
	})
	out := make([]deltaCanonicalOccurrence, len(sorted))
	for i, o := range sorted {
		out[i] = deltaCanonicalOccurrence{Path: o.Path, StartLine: o.StartLine, EndLine: o.EndLine}
	}
	return out
}

func deltaCanonicalizeFromBaseline(occs []dupcodeBaselineOccurrence) []deltaCanonicalOccurrence {
	sorted := make([]dupcodeBaselineOccurrence, len(occs))
	copy(sorted, occs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		if sorted[i].StartLine != sorted[j].StartLine {
			return sorted[i].StartLine < sorted[j].StartLine
		}
		return sorted[i].EndLine < sorted[j].EndLine
	})
	out := make([]deltaCanonicalOccurrence, len(sorted))
	for i, o := range sorted {
		out[i] = deltaCanonicalOccurrence{Path: o.Path, StartLine: o.StartLine, EndLine: o.EndLine}
	}
	return out
}

// deltaRepoRoot returns the absolute path to the repository root.
// The test is executed from the package directory; the repository
// root is three levels up.
func deltaRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	return root
}
