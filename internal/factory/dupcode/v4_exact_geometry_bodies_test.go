// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// This file groups the body-separation contracts:
//
//   - OneMaximalClone        : exactly one maximal cross-file finding
//   - RepeatedMultiplicity   : same-file multiplicity within one finding
//   - NWayClone              : one N-way cross-file finding across 3 files
//   - TwoIndependentBodies   : two distinct bodies do not collapse into one
//   - NoShadowSubFindings    : no threshold-sized prefix/suffix/interior
//     shadow finding coexists with the maximal finding
//
// The geometry tests assert the exact public projection (Path, StartLine,
// EndLine, TokenCount) plus the exact internal token-span preservation
// required by ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01 and
// ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02.
//
// All tests in this file are expected to FAIL against the current
// production V4 detector, because production V4 over-emits
// threshold-sized sub-findings and coalesces same-file occurrences. The
// subsequent production ACT owns making both the semantic and geometry
// contracts green.
//
// Internal token-span preservation is exercised in
// v4_exact_geometry_internal_test.go via the lower-level
// v4PipelineInternal orchestrator, which directly asserts exact
// StartPos and EndPos. The public projection tests in this file do NOT
// prove internal token-span preservation; TokenCount equality proves
// only span length. The internal tests directly assert StartPos,
// EndPos, StartLine, EndLine, and finding-to-span grouping.
package dupcode

import (
	"path/filepath"
	"testing"
)

// TestV4ExactGeometry_OneMaximalClone verifies the EXACT geometry contract
// for a single maximal cross-file clone: exactly one finding, exact token
// count, exact two occurrence paths, exact start and end lines, no shifted
// or extra occurrence.
//
// Internal token-span preservation is exercised separately by
// TestV4ExactGeometryInternal_OneMaximalClone in
// v4_exact_geometry_internal_test.go.
func TestV4ExactGeometry_OneMaximalClone(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "a.go")
	fileB := filepath.Join(tmpDir, "b.go")

	cloneCounter = 0
	cloneA := generateLargeCloneBody("a")
	cloneB := generateLargeCloneBody("b")

	writeTestFile(t, fileA, cloneA)
	writeTestFile(t, fileB, cloneB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	// Independent oracle: validate the frozen expected token-count constant
	// by counting tokens in the maximal clone span of file A.
	wantLargeCloneSpan := wantLargeCloneTokenCount
	gotLargeCloneSpan := countIndependentTokensForSpan(t, fileA, 3, 405)
	if gotLargeCloneSpan != wantLargeCloneSpan {
		t.Errorf("independent oracle: expected %d tokens in maximal clone span, got %d",
			wantLargeCloneSpan, gotLargeCloneSpan)
	}

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	expected := []exactFindingGeometry{
		{
			TokenCount: wantLargeCloneTokenCount,
			Occurrences: []exactOccurrenceGeometry{
				{Path: "a.go", StartLine: 3, EndLine: 405},
				{Path: "b.go", StartLine: 3, EndLine: 405},
			},
		},
	}
	assertExactFindingProjection(t, findings, expected, tmpDir)
}

// TestV4ExactGeometry_RepeatedMultiplicity verifies the EXACT geometry
// contract for same-file multiplicity: exactly one finding carrying two
// occurrences in repeat_a.go plus one occurrence in repeat_b.go, with each
// occurrence carrying distinct line ranges.
//
// Internal token-span preservation is exercised separately by
// TestV4ExactGeometryInternal_RepeatedMultiplicity in
// v4_exact_geometry_internal_test.go.
func TestV4ExactGeometry_RepeatedMultiplicity(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "repeat_a.go")
	fileB := filepath.Join(tmpDir, "repeat_b.go")

	cloneCounter = 0
	cloneA1 := makeCloneFunc("RepeatA1", 150)
	cloneA2 := makeCloneFunc("RepeatA2", 150)
	cloneB1 := makeCloneFunc("RepeatB1", 150)

	contentA := cloneA1 + cloneA2
	contentB := cloneB1

	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	// Independent oracle: validate the frozen expected token-count constant
	// for the 150-iteration clone body.
	wantMediumSpan := wantMediumCloneTokenCount
	gotMediumSpan := countIndependentTokensForSpan(t, fileA, 3, 155)
	if gotMediumSpan != wantMediumSpan {
		t.Errorf("independent oracle: expected %d tokens in repeat_a.go clone span, got %d",
			wantMediumSpan, gotMediumSpan)
	}

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	expected := []exactFindingGeometry{
		{
			TokenCount: wantMediumCloneTokenCount,
			Occurrences: []exactOccurrenceGeometry{
				{Path: "repeat_a.go", StartLine: 3, EndLine: 155},
				{Path: "repeat_a.go", StartLine: 157, EndLine: 309},
				{Path: "repeat_b.go", StartLine: 3, EndLine: 155},
			},
		},
	}
	assertExactFindingProjection(t, findings, expected, tmpDir)
}

// TestV4ExactGeometry_NWayClone verifies the EXACT geometry contract for an
// N-way clone across three files: exactly one finding with three occurrences,
// exact token count, exact fixture-relative paths, exact line ranges, no
// pairwise shadow finding.
//
// Internal token-span preservation is exercised separately by
// TestV4ExactGeometryInternal_NWayClone in
// v4_exact_geometry_internal_test.go.
func TestV4ExactGeometry_NWayClone(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{
		filepath.Join(tmpDir, "nw_a.go"),
		filepath.Join(tmpDir, "nw_b.go"),
		filepath.Join(tmpDir, "nw_c.go"),
	}

	cloneCounter = 0
	for i, f := range files {
		writeTestFile(t, f, generateLargeCloneBody([]string{"a", "b", "c"}[i]))
	}
	verifyFixturesTypeCheck(t, files...)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	expected := []exactFindingGeometry{
		{
			TokenCount: wantLargeCloneTokenCount,
			Occurrences: []exactOccurrenceGeometry{
				{Path: "nw_a.go", StartLine: 3, EndLine: 405},
				{Path: "nw_b.go", StartLine: 3, EndLine: 405},
				{Path: "nw_c.go", StartLine: 3, EndLine: 405},
			},
		},
	}
	assertExactFindingProjection(t, findings, expected, tmpDir)
}

// TestV4ExactGeometry_TwoIndependentBodies verifies the EXACT geometry
// contract for two structurally distinct bodies: exactly two findings, each
// carrying its own two occurrences (one per file), with no cross-association
// between the two bodies. The test remains correct even if the two
// findings are returned in the opposite slice order; canonical ordering is
// tested separately in v4_exact_geometry_ordering_test.go.
//
// Internal token-span preservation is exercised separately by
// TestV4ExactGeometryInternal_TwoIndependentBodies in
// v4_exact_geometry_internal_test.go.
func TestV4ExactGeometry_TwoIndependentBodies(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "ind_a.go")
	fileB := filepath.Join(tmpDir, "ind_b.go")

	cloneCounter = 0
	clone1 := generateForLoopClone("a", 1)
	clone2 := generateWhileLoopClone("a", 2)
	clone1B := generateForLoopClone("b", 1)
	clone2B := generateWhileLoopClone("b", 2)
	contentA := clone1 + "\n" + clone2
	contentB := clone1B + "\n" + clone2B

	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	// Independent oracle: validate the frozen expected token-count constant
	// for the 80-iteration loop body (used for both ForLoop and WhileLoop).
	wantLoopSpan := wantLoopCloneTokenCount
	gotForSpan := countIndependentTokensForSpan(t, fileA, 3, 85)
	if gotForSpan != wantLoopSpan {
		t.Errorf("independent oracle: expected %d tokens in ForLoop span, got %d",
			wantLoopSpan, gotForSpan)
	}

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	expected := []exactFindingGeometry{
		{
			TokenCount: wantLoopCloneTokenCount,
			Occurrences: []exactOccurrenceGeometry{
				{Path: "ind_a.go", StartLine: 3, EndLine: 85},
				{Path: "ind_b.go", StartLine: 3, EndLine: 85},
			},
		},
		{
			TokenCount: wantLoopCloneTokenCount,
			Occurrences: []exactOccurrenceGeometry{
				{Path: "ind_a.go", StartLine: 87, EndLine: 169},
				{Path: "ind_b.go", StartLine: 87, EndLine: 169},
			},
		},
	}
	assertExactFindingProjection(t, findings, expected, tmpDir)
}

// TestV4ExactGeometry_NoShadowSubFindings verifies the EXACT geometry
// contract that the maximal clone geometry is present and that no
// threshold-sized prefix, suffix, or interior shadow finding coexists
// with it. The test asserts the complete expected geometry multiset, not
// merely the finding count.
//
// Internal token-span preservation is exercised separately by
// TestV4ExactGeometryInternal_NoShadowSubFindings in
// v4_exact_geometry_internal_test.go.
func TestV4ExactGeometry_NoShadowSubFindings(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "shadow_a.go")
	fileB := filepath.Join(tmpDir, "shadow_b.go")

	cloneCounter = 0
	cloneA := generateLargeCloneBody("shadow_a")
	cloneB := generateLargeCloneBody("shadow_b")

	writeTestFile(t, fileA, cloneA)
	writeTestFile(t, fileB, cloneB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	// The maximal geometry multiset MUST contain the cross-file maximal
	// clone and MUST NOT contain any sub-finding.
	expected := []exactFindingGeometry{
		{
			TokenCount: wantLargeCloneTokenCount,
			Occurrences: []exactOccurrenceGeometry{
				{Path: "shadow_a.go", StartLine: 3, EndLine: 405},
				{Path: "shadow_b.go", StartLine: 3, EndLine: 405},
			},
		},
	}
	assertExactFindingProjection(t, findings, expected, tmpDir)
}
