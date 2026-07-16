// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// This file groups the determinism contract. It maintains TWO independent
// views for each run:
//
//  1. raw publication projection (publication order, no normalization);
//  2. canonical geometry multiset (occurrences canonicalized per finding,
//     findings sorted by a total test-owned projection comparator).
//
// Each run is compared against the FIRST run's two views so that two
// distinct regressions are caught:
//
//   - geometry/grouping changed (multiset view differs);
//   - publication order changed (raw view differs).
//
// Index-based slice comparison is NOT used as a multiset comparison.
// Diagnostic iteration is sorted for deterministic output.
//
// The internal grouped projection is also compared across runs so
// internal token-span determinism is exercised separately from the
// public projection.
//
// Sibling files in this contract group:
//
//   - v4_exact_geometry_support_test.go (projection types, helpers)
//   - v4_exact_geometry_bodies_test.go (body-separation contracts)
//   - v4_exact_geometry_internal_test.go (internal token-span tests)
//   - v4_exact_geometry_ordering_test.go (CanonicalFindingOrdering,
//     CanonicalOccurrenceOrdering)
package dupcode

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// determinismRunCount is the number of repeated detector invocations
// required by the geometry determinism contract. It is large enough to
// expose map-order instability without turning the test into a performance
// benchmark.
const determinismRunCount = 5

// canonicalGeometryMultiset canonicalizes a finding projection into a
// multiset view:
//
//  1. canonicalize occurrences inside each finding;
//  2. encode the complete finding projection as a string;
//  3. sort findings by a total test-owned projection comparator;
//  4. preserve duplicate finding multiplicity.
//
// The test-owned comparator is canonicalFindingKey (token_count + sorted
// occurrence geometry). Sorting by this comparator makes the multiset
// order independent of production's publication order, so any production
// ordering defect is exposed by the raw view, not by the multiset view.
func canonicalGeometryMultiset(findings []exactFindingGeometry) []exactFindingGeometry {
	canonical := make([]exactFindingGeometry, len(findings))
	for i, f := range findings {
		canonical[i] = exactFindingGeometry{
			TokenCount:  f.TokenCount,
			Occurrences: canonicalizeOccurrences(f.Occurrences),
		}
	}
	sort.Slice(canonical, func(i, j int) bool {
		return canonicalFindingKey(canonical[i]) < canonicalFindingKey(canonical[j])
	})
	return canonical
}

// canonicalInternalGeometryMultiset canonicalizes a grouped internal
// finding projection into a multiset view. Same contract as
// canonicalGeometryMultiset but for internal positions.
func canonicalInternalGeometryMultiset(findings []exactInternalFindingGeometry) []exactInternalFindingGeometry {
	canonical := make([]exactInternalFindingGeometry, len(findings))
	for i, f := range findings {
		canonical[i] = exactInternalFindingGeometry{
			TokenCount:  f.TokenCount,
			Occurrences: canonicalizeInternalOccurrences(f.Occurrences),
		}
	}
	sort.Slice(canonical, func(i, j int) bool {
		return canonicalInternalFindingKey(canonical[i]) < canonicalInternalFindingKey(canonical[j])
	})
	return canonical
}

// TestV4ExactGeometry_Determinism verifies the EXACT geometry determinism
// contract over repeated executions of CheckRepo on the same fixture.
// The test exercises both the public projection (Path / StartLine /
// EndLine / TokenCount) and the publication-order stability.
func TestV4ExactGeometry_Determinism(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "det_a.go")
	fileB := filepath.Join(tmpDir, "det_b.go")

	cloneCounter = 0
	cloneA1 := makeCloneFunc("DetA1", 150)
	cloneA2 := makeCloneFunc("DetA2", 150)
	cloneB1 := makeCloneFunc("DetB1", 150)

	contentA := cloneA1 + cloneA2
	contentB := cloneB1

	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}

	var firstCanonical []exactFindingGeometry
	var firstRaw []exactFindingGeometry

	for i := 0; i < determinismRunCount; i++ {
		findings, err := CheckRepo(tmpDir, cfg)
		if err != nil {
			t.Fatalf("CheckRepo run %d failed: %v", i, err)
		}

		// Raw (publication-order) projection preserved verbatim so any
		// publication-order regression is detected by compareRawRuns.
		raw := make([]exactFindingGeometry, len(findings))
		for j, f := range findings {
			raw[j] = projectFindingGeometry(t, f, tmpDir)
		}

		// Canonical multiset: occurrences canonicalized per finding,
		// findings sorted by total test-owned comparator. Index-based
		// comparison is intentionally avoided so this view is a real
		// multiset comparison, not an index-based slice comparison.
		canonical := canonicalGeometryMultiset(raw)

		if i == 0 {
			firstCanonical = canonical
			firstRaw = raw
			continue
		}

		// Multiset comparison: report any change in the canonical geometry
		// projection. Use a true multiset equality (canonical key -> count)
		// rather than index-based slice equality.
		compareCanonicalMultisets(t, i, firstCanonical, canonical)

		// Publication-order comparison: raw projection slice compared
		// element-wise so any change in the actual published order is
		// caught.
		compareRawRuns(t, i, firstRaw, raw)
	}
}

// TestV4ExactGeometryInternal_Determinism verifies the EXACT internal
// token-span determinism contract. The internal grouped projection must
// be identical across repeated executions.
//
// Like the public determinism test, this uses two independent views:
// the raw publication-order internal projection and the canonical
// internal geometry multiset. Index-based comparison is avoided.
func TestV4ExactGeometryInternal_Determinism(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "det_a.go")
	fileB := filepath.Join(tmpDir, "det_b.go")

	cloneCounter = 0
	cloneA1 := makeCloneFunc("DetA1", 150)
	cloneA2 := makeCloneFunc("DetA2", 150)
	cloneB1 := makeCloneFunc("DetB1", 150)

	contentA := cloneA1 + cloneA2
	contentB := cloneB1

	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}

	var firstCanonical []exactInternalFindingGeometry
	var firstRaw []exactInternalFindingGeometry

	for i := 0; i < determinismRunCount; i++ {
		raw := v4PipelineInternal(t, tmpDir, []string{fileA, fileB}, cfg)
		canonical := canonicalInternalGeometryMultiset(raw)

		if i == 0 {
			firstCanonical = canonical
			firstRaw = raw
			continue
		}

		compareInternalCanonicalMultisets(t, i, firstCanonical, canonical)
		compareInternalRawRuns(t, i, firstRaw, raw)
	}
}

// compareCanonicalMultisets compares two canonicalized finding
// projection multisets using canonical-key count maps. It reports every
// multiplicity delta with explicit expected N and actual M and the
// projection. Diagnostic iteration is sorted for deterministic output.
func compareCanonicalMultisets(t *testing.T, run int, want, got []exactFindingGeometry) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("run %d: canonical multiset cardinality changed from %d to %d",
			run, len(want), len(got))
		// Continue reporting deltas; the cardinality message alone does not
		// localize the change.
	}
	reportMultiplicityDiffs(t,
		fmt.Sprintf("run %d canonical multiset", run),
		want, got)
}

// compareRawRuns compares two raw (publication-order) projection slices
// element-wise. Each element's Occurrences are compared in their actual
// order so any change in the publication order of either findings or
// occurrences is caught.
func compareRawRuns(t *testing.T, run int, want, got []exactFindingGeometry) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("run %d: published finding cardinality changed from %d to %d",
			run, len(want), len(got))
	}
	compareCount := len(want)
	if len(got) < compareCount {
		compareCount = len(got)
	}
	for i := 0; i < compareCount; i++ {
		if want[i].TokenCount != got[i].TokenCount {
			t.Errorf("run %d: published finding[%d] token_count changed: %d -> %d",
				run, i, want[i].TokenCount, got[i].TokenCount)
		}
		if len(want[i].Occurrences) != len(got[i].Occurrences) {
			t.Errorf("run %d: published finding[%d] occurrence count changed: %d -> %d",
				run, i, len(want[i].Occurrences), len(got[i].Occurrences))
			continue
		}
		for j := range want[i].Occurrences {
			if want[i].Occurrences[j] != got[i].Occurrences[j] {
				t.Errorf("run %d: published finding[%d].occurrence[%d] changed: %+v -> %+v",
					run, i, j, want[i].Occurrences[j], got[i].Occurrences[j])
			}
		}
	}
}

// compareInternalCanonicalMultisets compares two canonicalized internal
// grouped-projection multisets. Same contract as
// compareCanonicalMultisets but for the internal projection.
func compareInternalCanonicalMultisets(t *testing.T, run int, want, got []exactInternalFindingGeometry) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("run %d: internal canonical multiset cardinality changed from %d to %d",
			run, len(want), len(got))
	}
	reportInternalMultiplicityDiffs(t,
		fmt.Sprintf("run %d internal canonical multiset", run),
		want, got)
}

// compareInternalRawRuns compares two raw internal grouped-projection
// slices element-wise. Each element's Occurrences are compared in their
// actual order so any change in the publication order of either
// internal findings or internal occurrences is caught.
func compareInternalRawRuns(t *testing.T, run int, want, got []exactInternalFindingGeometry) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("run %d: internal published finding cardinality changed from %d to %d",
			run, len(want), len(got))
	}
	compareCount := len(want)
	if len(got) < compareCount {
		compareCount = len(got)
	}
	for i := 0; i < compareCount; i++ {
		if want[i].TokenCount != got[i].TokenCount {
			t.Errorf("run %d: internal published finding[%d] token_count changed: %d -> %d",
				run, i, want[i].TokenCount, got[i].TokenCount)
		}
		if len(want[i].Occurrences) != len(got[i].Occurrences) {
			t.Errorf("run %d: internal published finding[%d] occurrence count changed: %d -> %d",
				run, i, len(want[i].Occurrences), len(got[i].Occurrences))
			continue
		}
		for j := range want[i].Occurrences {
			if want[i].Occurrences[j] != got[i].Occurrences[j] {
				t.Errorf("run %d: internal published finding[%d].occurrence[%d] changed: %+v -> %+v",
					run, i, j, want[i].Occurrences[j], got[i].Occurrences[j])
			}
		}
	}
}
