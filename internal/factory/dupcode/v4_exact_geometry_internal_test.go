// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// This file groups the internal token-span contract tests. It exercises
// the lower-level V4 stages (tokenization, seed discovery, chaining,
// occurrence extraction, and N-way merge) directly so the exact
// StartPos and EndPos of every occurrence is asserted as part of the
// red specification.
//
// The internal projection is grouped: each finding carries its
// TokenCount together with the exact occurrences attached to it. A flat
// collection of spans is insufficient because it cannot detect
// association with the wrong finding.
//
// All tests in this file are expected to FAIL against the current
// production V4 detector, because production V4 over-emits
// threshold-sized sub-findings and coalesces same-file occurrences. The
// subsequent production ACT owns making both the semantic and geometry
// contracts green.
//
// The internal tests directly assert:
//
//   - exact StartPos for each occurrence;
//   - exact EndPos   for each occurrence;
//   - exact StartLine and EndLine for each occurrence;
//   - selection of the correct repeated region (the candidate table
//     frozen in this file is the audited literal expected internal
//     geometry);
//   - preservation of two same-file spans;
//   - finding-to-span association (grouped projection);
//   - absence of position-based collapse.
//
// The internal tests are NOT indirected through TokenCount equality;
// they directly compare StartPos and EndPos against the frozen audited
// literals in the table below.
//
// The orchestrator that drives the lower-level V4 stages lives in
// v4_exact_geometry_internal_helpers_test.go (v4PipelineInternal).
package dupcode

import (
	"path/filepath"
	"testing"
)

// TestV4ExactGeometryInternal_OneMaximalClone verifies the exact internal
// token-span preservation for a single maximal cross-file clone.
//
// Frozen audited literals:
//
//	a.go:   StartPos=3   EndPos=2413
//	b.go:   StartPos=3   EndPos=2413
//
// The fixture file is "package test\n\n" + generateLargeCloneBody(...).
// generateLargeCloneBody wraps 400 iterations of sharedStatements in a
// function. The package declaration contributes 3 tokens
// (package, test, ;). The function spans 7 + 4 + 6*400 = 2411 tokens
// starting at the func keyword (position 3 in the file) and ending at
// the inclusive position 3 + 2411 - 1 = 2413 (the auto-inserted SEMI
// after the closing brace).
//
// This test fails RED if:
//
//   - StartPos or EndPos shifts by one token;
//   - the wrong region is selected (e.g., a sub-finding);
//   - line geometry is correct but token geometry is wrong (StartLine
//     and StartPos / EndLine and EndPos must agree for the same span).
func TestV4ExactGeometryInternal_OneMaximalClone(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "a.go")
	fileB := filepath.Join(tmpDir, "b.go")

	cloneCounter = 0
	writeTestFile(t, fileA, generateLargeCloneBody("a"))
	writeTestFile(t, fileB, generateLargeCloneBody("b"))
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	actual := v4PipelineInternal(t, tmpDir, []string{fileA, fileB}, cfg)

	expected := []exactInternalFindingGeometry{
		{
			TokenCount: wantLargeCloneTokenCount,
			Occurrences: []exactInternalOccurrenceGeometry{
				{Path: "a.go", StartPos: 3, EndPos: 2413, StartLine: 3, EndLine: 405},
				{Path: "b.go", StartPos: 3, EndPos: 2413, StartLine: 3, EndLine: 405},
			},
		},
	}
	assertExactInternalFindingGeometry(t, actual, expected)
}

// TestV4ExactGeometryInternal_RepeatedMultiplicity verifies exact
// internal token-span preservation for same-file multiplicity.
//
// Frozen audited literals:
//
//	repeat_a.go, first body:  StartPos=3    EndPos=913
//	repeat_a.go, second body: StartPos=914  EndPos=1824
//	repeat_b.go:              StartPos=3    EndPos=913
//
// The 150-iteration clone spans 911 tokens (7 + 4 + 6*150). The package
// declaration contributes 3 tokens (positions 0..2), so the first body
// spans positions 3..913. After the first body and a blank line, the
// second body spans positions 914..1824. The repeat_b.go file holds
// only one body, positions 3..913.
//
// This test fails RED if:
//
//   - the second repeat_a.go span is replaced by the first;
//   - the two same-file spans collapse into one;
//   - StartPos / EndPos are off by one token.
func TestV4ExactGeometryInternal_RepeatedMultiplicity(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "repeat_a.go")
	fileB := filepath.Join(tmpDir, "repeat_b.go")

	cloneCounter = 0
	contentA := makeCloneFunc("RepeatA1", 150) + makeCloneFunc("RepeatA2", 150)
	contentB := makeCloneFunc("RepeatB1", 150)
	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	actual := v4PipelineInternal(t, tmpDir, []string{fileA, fileB}, cfg)

	expected := []exactInternalFindingGeometry{
		{
			TokenCount: wantMediumCloneTokenCount,
			Occurrences: []exactInternalOccurrenceGeometry{
				{Path: "repeat_a.go", StartPos: 3, EndPos: 913, StartLine: 3, EndLine: 155},
				{Path: "repeat_a.go", StartPos: 914, EndPos: 1824, StartLine: 157, EndLine: 309},
				{Path: "repeat_b.go", StartPos: 3, EndPos: 913, StartLine: 3, EndLine: 155},
			},
		},
	}
	assertExactInternalFindingGeometry(t, actual, expected)
}

// TestV4ExactGeometryInternal_NWayClone verifies exact internal
// token-span preservation for an N-way clone across three files.
//
// Frozen audited literals:
//
//	nw_a.go: StartPos=3 EndPos=2413
//	nw_b.go: StartPos=3 EndPos=2413
//	nw_c.go: StartPos=3 EndPos=2413
//
// This test fails RED if:
//
//   - any occurrence points to the wrong repeated region;
//   - any N-way occurrence is missing;
//   - StartPos / EndPos are off by one token.
func TestV4ExactGeometryInternal_NWayClone(t *testing.T) {
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
	actual := v4PipelineInternal(t, tmpDir, files, cfg)

	expected := []exactInternalFindingGeometry{
		{
			TokenCount: wantLargeCloneTokenCount,
			Occurrences: []exactInternalOccurrenceGeometry{
				{Path: "nw_a.go", StartPos: 3, EndPos: 2413, StartLine: 3, EndLine: 405},
				{Path: "nw_b.go", StartPos: 3, EndPos: 2413, StartLine: 3, EndLine: 405},
				{Path: "nw_c.go", StartPos: 3, EndPos: 2413, StartLine: 3, EndLine: 405},
			},
		},
	}
	assertExactInternalFindingGeometry(t, actual, expected)
}

// TestV4ExactGeometryInternal_TwoIndependentBodies verifies exact
// internal token-span preservation for two structurally distinct bodies
// in each file.
//
// Frozen audited literals:
//
//	ind_a.go / ind_b.go first body:  StartPos=3   EndPos=493
//	ind_a.go / ind_b.go second body: StartPos=494 EndPos=984
//
// The 80-iteration loop clone spans 491 tokens (7 + 4 + 6*80). The
// package declaration contributes 3 tokens (positions 0..2), so the
// first body spans positions 3..493. After the first body and a blank
// line, the second body spans positions 494..984.
//
// This test fails RED if:
//
//   - any occurrence is associated with the wrong finding (the
//     grouped projection catches this);
//   - StartPos / EndPos are off by one token;
//   - the two bodies collapse into a single finding.
func TestV4ExactGeometryInternal_TwoIndependentBodies(t *testing.T) {
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

	cfg := Config{MinLines: 40, MinTokens: 400}
	actual := v4PipelineInternal(t, tmpDir, []string{fileA, fileB}, cfg)

	expected := []exactInternalFindingGeometry{
		{
			TokenCount: wantLoopCloneTokenCount,
			Occurrences: []exactInternalOccurrenceGeometry{
				{Path: "ind_a.go", StartPos: 3, EndPos: 493, StartLine: 3, EndLine: 85},
				{Path: "ind_b.go", StartPos: 3, EndPos: 493, StartLine: 3, EndLine: 85},
			},
		},
		{
			TokenCount: wantLoopCloneTokenCount,
			Occurrences: []exactInternalOccurrenceGeometry{
				{Path: "ind_a.go", StartPos: 494, EndPos: 984, StartLine: 87, EndLine: 169},
				{Path: "ind_b.go", StartPos: 494, EndPos: 984, StartLine: 87, EndLine: 169},
			},
		},
	}
	assertExactInternalFindingGeometry(t, actual, expected)
}

// TestV4ExactGeometryInternal_NoShadowSubFindings verifies the maximal
// internal geometry is present and no threshold-sized prefix, suffix,
// or interior shadow finding coexists with it.
//
// Frozen audited literals:
//
//	shadow_a.go: StartPos=3 EndPos=2413
//	shadow_b.go: StartPos=3 EndPos=2413
//
// The maximal large-clone body covers positions 3..2413 in each file.
// Any shadow sub-finding with smaller StartPos/EndPos or off-by-one
// positions is a regression.
func TestV4ExactGeometryInternal_NoShadowSubFindings(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "shadow_a.go")
	fileB := filepath.Join(tmpDir, "shadow_b.go")

	cloneCounter = 0
	writeTestFile(t, fileA, generateLargeCloneBody("shadow_a"))
	writeTestFile(t, fileB, generateLargeCloneBody("shadow_b"))
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	actual := v4PipelineInternal(t, tmpDir, []string{fileA, fileB}, cfg)

	expected := []exactInternalFindingGeometry{
		{
			TokenCount: wantLargeCloneTokenCount,
			Occurrences: []exactInternalOccurrenceGeometry{
				{Path: "shadow_a.go", StartPos: 3, EndPos: 2413, StartLine: 3, EndLine: 405},
				{Path: "shadow_b.go", StartPos: 3, EndPos: 2413, StartLine: 3, EndLine: 405},
			},
		},
	}
	assertExactInternalFindingGeometry(t, actual, expected)
}
