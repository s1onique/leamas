// Package dupcode provides the fixture constructors and shared
// helpers for the R1 cross-region regression proof owned by
// ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02.
//
// The CORRECTION01 corpus relied on an asymmetric fixture whose
// right side silently re-used the left side's path. The two sides
// then resolved to the same production syntax region, the alignment
// guard was never consulted, and the regression proof passed without
// ever exercising the cross-region all-pairs fallback.
//
// The fixtures in this file repair the defect: every fixture uses
// the path-aware makeRawWindows constructor and writes the path of
// every window explicitly. The helpers below are the shared seam
// the R2-R6 tests use to drive the production candidate
// generator and assert canonical output.
package dupcode

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// expectedAsymmetricRightStarts / expectedAsymmetricLeftStarts are
// the canonical start-position sequences asserted by the R2
// fixture-contract test. Keeping them as named constants keeps the
// assertion legible and prevents the test from drifting away from
// the documented shape.
var (
	expectedAsymmetricRightStarts = []int{0, 1, 2}
	expectedAsymmetricLeftStarts  = []int{50, 100, 101, 102}
)

// asymmetricRightFixture is the corrected asymmetric right-side-extra
// fixture. It must always build the canonical alpha.go/beta.go shape
// via the path-aware makeRawWindows helper so the production
// pipeline actually crosses two distinct regions.
func asymmetricRightFixture() v4PerfFixture {
	return v4PerfFixture{
		Name:          "AsymmetricLeadingExtraRight",
		WindowSize:    3,
		LeftWindows:   makeRawWindows("alpha.go", expectedAsymmetricRightStarts),
		RightWindows:  makeRawWindows("beta.go", expectedAsymmetricLeftStarts),
		PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
	}
}

// asymmetricLeftFixture mirrors asymmetricRightFixture: the left side
// has the extra leading occurrence. The guard must still return
// false and the maximal off-index chain must survive.
func asymmetricLeftFixture() v4PerfFixture {
	return v4PerfFixture{
		Name:          "AsymmetricLeadingExtraLeft",
		WindowSize:    3,
		LeftWindows:   makeRawWindows("alpha.go", expectedAsymmetricLeftStarts),
		RightWindows:  makeRawWindows("beta.go", expectedAsymmetricRightStarts),
		PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
	}
}

// alignedDistinctFixture pins the aligned fast path: equal cardinality
// and identical relative positions across two distinct regions. The
// guard must return true; the diagonal must fire.
func alignedDistinctFixture() v4PerfFixture {
	return v4PerfFixture{
		Name:         "AlignedDistinctRegions",
		WindowSize:   3,
		LeftWindows:  makeRawWindows("alpha.go", []int{0, 1, 2}),
		RightWindows: makeRawWindows("beta.go", []int{100, 101, 102}),
		PerPathLength: map[string]int{
			"alpha.go": 200, "beta.go": 200,
		},
	}
}

// asymmetricAnnotatedInputs builds the same annotated-window and
// per-region index inputs the production candidate generator sees
// for the supplied fixture. The returned values let the R3 / R4
// tests exercise the alignment guard and the conservative all-pairs
// fallback without going through the full pipeline.
//
// The candidate set returned for `fx` is the result of running the
// production candidate generator (v4BuildRegionBoundedChainInputs)
// on the region-filtered window map. This is the "equivalent
// test-only seam" referenced by the ACT for the conservative
// cross-region all-pairs emitter.
func asymmetricAnnotatedInputs(t *testing.T, fx v4PerfFixture) (
	annotatedWindows []v4AnnotatedWindow,
	idxA, idxB []int,
	leftRegion, rightRegion v4SyntaxRegionID,
	combined []v4RegionSeedMatch,
	partitions map[v4ChainPairKey][]v4RegionSeedMatch,
) {
	t.Helper()
	wm := v4BuildAlignedWindowMap("seed", fx.LeftWindows, fx.RightWindows)
	analyses := v4MakeAlignedAnalyses(fx.PerPathLength, nil)
	combined, partitions = v4BuildRegionBoundedChainInputs(wm, analyses)
	if len(combined) == 0 {
		t.Fatalf("%s: production candidate generator returned no matches", fx.Name)
	}

	// Reconstruct the annotated windows in the order the candidate
	// generator saw them. We rebuild the slice from the raw windowMap
	// so the index sequences the guard reads are identical to the
	// production input.
	flat := make([]rawWindow, 0)
	for _, ws := range wm {
		flat = append(flat, ws...)
	}
	annotatedWindows = make([]v4AnnotatedWindow, len(flat))
	byRegion := make(map[v4SyntaxRegionID][]int)
	for i, w := range flat {
		a, ok := analyses[w.Path]
		if !ok {
			annotatedWindows[i] = v4AnnotatedWindow{w: w}
			continue
		}
		rid, ok := a.windowFitsRegion(w.StartPos, w.EndPos)
		if !ok {
			annotatedWindows[i] = v4AnnotatedWindow{w: w}
			continue
		}
		annotatedWindows[i] = v4AnnotatedWindow{w: w, region: rid}
		byRegion[rid] = append(byRegion[rid], i)
	}
	for rid := range byRegion {
		idxs := byRegion[rid]
		sort.Slice(idxs, func(i, j int) bool {
			return annotatedWindows[idxs[i]].w.StartPos < annotatedWindows[idxs[j]].w.StartPos
		})
		byRegion[rid] = idxs
	}
	regionOrder := make([]v4SyntaxRegionID, 0, len(byRegion))
	for rid := range byRegion {
		regionOrder = append(regionOrder, rid)
	}
	sort.Slice(regionOrder, func(i, j int) bool {
		if regionOrder[i].Path != regionOrder[j].Path {
			return regionOrder[i].Path < regionOrder[j].Path
		}
		return regionOrder[i].Ordinal < regionOrder[j].Ordinal
	})
	if len(regionOrder) < 2 {
		t.Fatalf("%s: expected two distinct regions, got %d", fx.Name, len(regionOrder))
	}
	leftRegion = regionOrder[0]
	rightRegion = regionOrder[1]
	idxA = byRegion[leftRegion]
	idxB = byRegion[rightRegion]
	return annotatedWindows, idxA, idxB, leftRegion, rightRegion, combined, partitions
}

// pathStarts extracts the ordered (path, start) tuples from a slice
// of raw windows. Used to format clear failure diagnostics in the
// R2 / R3 / R4 tests.
func pathStarts(ws []v4RawWindow) string {
	var b strings.Builder
	b.WriteString("[")
	for i, w := range ws {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s@%d", w.Path, w.StartPos)
	}
	b.WriteString("]")
	return b.String()
}

// pickRegion returns the region that owns the first token of the
// supplied window. It is used by the fixture-contract test to
// prove the two sides of the fixture resolve to distinct regions.
func pickRegion(t *testing.T, analyses map[string]*v4FileAnalysis, w v4RawWindow) v4SyntaxRegionID {
	t.Helper()
	a, ok := analyses[w.Path]
	if !ok {
		t.Fatalf("analysis missing for %q", w.Path)
	}
	rid, ok := a.windowFitsRegion(w.StartPos, w.EndPos)
	if !ok {
		t.Fatalf("window %s does not fit a region", pathStarts([]v4RawWindow{w}))
	}
	return rid
}

// intSliceEqual compares two int slices for exact equality.
func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// crossRegionCorpusCase is one row of the R6 three-case differential
// table. Every case carries its intended guard verdict so a future
// regression that re-orders the guard logic is caught before the
// canonical comparison runs.
type crossRegionCorpusCase struct {
	Name          string
	Fixture       v4PerfFixture
	WantGuardOK   bool
	WantMinChain  int
	WantOffset    int
	WantLeftPath  string
	WantRightPath string
}

// summarizePartitions renders a partition map for diagnostics. The
// summary uses only the key + member count so a failure message stays
// small enough to read at a glance.
func summarizePartitions(partitions map[v4ChainPairKey][]v4RegionSeedMatch) string {
	keys := make([]v4ChainPairKey, 0, len(partitions))
	for k := range partitions {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Offset != keys[j].Offset {
			return keys[i].Offset < keys[j].Offset
		}
		return keys[i].String() < keys[j].String()
	})
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, len(partitions[k])))
	}
	return strings.Join(parts, ", ")
}
