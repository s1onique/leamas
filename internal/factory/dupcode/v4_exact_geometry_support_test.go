// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// This file groups the shared helpers, projection types, and frozen
// expected constants for the exact-geometry specification. The geometry
// tests assert the exact public finding/occurrence geometry and the
// exact internal token-span preservation required by
// ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01 and correction
// ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02.
//
// Sibling files in this contract group:
//
//   - v4_exact_geometry_bodies_test.go       (OneMaximalClone, RepeatedMultiplicity,
//     NWayClone, TwoIndependentBodies,
//     NoShadowSubFindings)
//   - v4_exact_geometry_internal_test.go    (internal token-span tests)
//   - v4_exact_geometry_internal_helpers_test.go (v4PipelineInternal orchestrator)
//   - v4_exact_geometry_determinism_test.go  (Determinism)
//   - v4_exact_geometry_ordering_test.go     (CanonicalFindingOrdering,
//     CanonicalOccurrenceOrdering)
//   - v4_exact_geometry_path_test.go         (TestNormalizeFixturePath_Contract)
//   - v4_exact_geometry_diagnostics_test.go  (multiplicity diagnostics helpers)
//
// The geometry tests are the red specification. They are expected to fail
// against the current production V4 detector because production emits
// threshold-sized sub-findings, coalesces same-file occurrences, and
// otherwise does not match the exact geometry contract. The subsequent
// production ACT owns making both the semantic and geometry contracts
// green.
package dupcode

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ----------------------------------------------------------------------------
// Exact public projection types. Normative fields: Finding.TokenCount and
// Occurrence.{Path, StartLine, EndLine}. Fingerprint, StartPos, EndPos, and
// temporary-directory paths are intentionally excluded.
// ----------------------------------------------------------------------------

type exactOccurrenceGeometry struct {
	Path      string
	StartLine int
	EndLine   int
}

type exactFindingGeometry struct {
	TokenCount  int
	Occurrences []exactOccurrenceGeometry
}

// ----------------------------------------------------------------------------
// Exact grouped internal projection types.
//
// The grouped form preserves finding membership. A flat collection of spans
// is insufficient because it cannot detect association with the wrong
// finding. The grouped projection must fail RED if occurrences are attached
// to the wrong finding.
//
// The internal token-span projection is an implementation invariant: the
// public Finding/Occurrence types do NOT carry StartPos/EndPos. They are
// not part of the public JSON / baseline contract.
// ----------------------------------------------------------------------------

// exactInternalOccurrenceGeometry is the grouped internal occurrence
// projection. StartPos and EndPos are 0-based token offsets. EndPos is
// INCLUSIVE (the same convention as rawWindow.EndPos in coalesce.go,
// computed as startPos + MinTokens - 1).
type exactInternalOccurrenceGeometry struct {
	Path      string
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

// exactInternalFindingGeometry is the grouped internal finding projection.
// It associates each occurrence with a single finding's TokenCount, so
// span-to-finding mismatches are detected.
type exactInternalFindingGeometry struct {
	TokenCount  int
	Occurrences []exactInternalOccurrenceGeometry
}

// ----------------------------------------------------------------------------
// Frozen expected token-count constants.
//
// Derivation: sharedStatements (in v4_semantics_test.go) emits
//
//	"    n := 0\n" + iterations * "    n = n + 1\n"
//
// go/scanner automatically inserts a SEMI (";") at the end of each
// statement, so per line the non-comment token counts are:
//
//	"n := 0"    -> IDENT, ":=", INT(0), ";"            = 4 tokens
//	"n = n + 1" -> IDENT, "=", IDENT, "+", INT(1), ";" = 6 tokens
//
// Wrapping function header adds "func", IDENT(name), "(", ")", "{" (5
// tokens), and the closing "}" with its own SEMI (2 tokens), for a total
// of 7 wrapper tokens around the body.
//
// TokenCount = 7 + 4 + 6 * iterations
//
// generateLargeCloneBody uses iterations=400                  -> 2411
// makeCloneFunc with iterations=150                          ->  911
// generateForLoopClone / generateWhileLoopClone iterations=80 -> 491
// ----------------------------------------------------------------------------

const (
	wantLargeCloneTokenCount  = 2411 // generateLargeCloneBody (iterations=400)
	wantMediumCloneTokenCount = 911  // makeCloneFunc       (iterations=150)
	wantLoopCloneTokenCount   = 491  // generateForLoopClone / WhileLoopClone (iterations=80)
)

// ----------------------------------------------------------------------------
// Stable fixture-root-relative path projection.
//
// Production CheckRepo normalizes paths via NormalizePathForBaseline. The
// geometry tests do NOT reuse that helper as the independent oracle (a
// shared defect would mask incorrect geometry). The projector below:
//
//  1. Joins a relative occurrence path with the fixture root before
//     applying filepath.Rel so mixed relative/absolute inputs are handled
//     identically.
//  2. Rejects errors from filepath.Rel.
//  3. Rejects results that fail filepath.IsLocal. The "strings.HasPrefix(rel,
//     \"..\")" pattern is forbidden by the geometry ACT (would incorrectly
//     reject legitimate local names like "..generated.go").
//  4. Normalizes separators via filepath.ToSlash.
//  5. Preserves the complete fixture-root-relative path; the basename alone
//     is never accepted.
// ----------------------------------------------------------------------------

func normalizeFixturePath(fixtureRoot, occurrencePath string) (string, error) {
	if fixtureRoot == "" {
		return "", fmt.Errorf("empty fixture root")
	}
	if occurrencePath == "" {
		return "", fmt.Errorf("empty occurrence path")
	}
	absPath := occurrencePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(fixtureRoot, occurrencePath)
	}
	rel, err := filepath.Rel(fixtureRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("make fixture-relative path: %w", err)
	}
	if rel == "" {
		return "", fmt.Errorf("empty fixture-relative path for %q", occurrencePath)
	}
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("occurrence path escapes fixture root: %q", rel)
	}
	return filepath.ToSlash(rel), nil
}

func canonicalizeOccurrences(occs []exactOccurrenceGeometry) []exactOccurrenceGeometry {
	sorted := make([]exactOccurrenceGeometry, len(occs))
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
	return sorted
}

func canonicalizeInternalOccurrences(occs []exactInternalOccurrenceGeometry) []exactInternalOccurrenceGeometry {
	sorted := make([]exactInternalOccurrenceGeometry, len(occs))
	copy(sorted, occs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		if sorted[i].StartPos != sorted[j].StartPos {
			return sorted[i].StartPos < sorted[j].StartPos
		}
		if sorted[i].EndPos != sorted[j].EndPos {
			return sorted[i].EndPos < sorted[j].EndPos
		}
		if sorted[i].StartLine != sorted[j].StartLine {
			return sorted[i].StartLine < sorted[j].StartLine
		}
		return sorted[i].EndLine < sorted[j].EndLine
	})
	return sorted
}

func projectFindingGeometry(t *testing.T, f Finding, fixtureRoot string) exactFindingGeometry {
	t.Helper()
	occs := make([]exactOccurrenceGeometry, len(f.Occurrences))
	for i, occ := range f.Occurrences {
		proj, err := normalizeFixturePath(fixtureRoot, occ.Path)
		if err != nil {
			t.Fatalf("project occurrence path %q (root=%q): %v",
				occ.Path, fixtureRoot, err)
		}
		occs[i] = exactOccurrenceGeometry{
			Path:      proj,
			StartLine: occ.StartLine,
			EndLine:   occ.EndLine,
		}
	}
	return exactFindingGeometry{
		TokenCount:  f.TokenCount,
		Occurrences: occs,
	}
}

func canonicalFindingKey(f exactFindingGeometry) string {
	parts := make([]string, 0, 1+len(f.Occurrences))
	parts = append(parts, fmt.Sprintf("tc=%d", f.TokenCount))
	for _, o := range f.Occurrences {
		parts = append(parts, fmt.Sprintf("%s:%d-%d", o.Path, o.StartLine, o.EndLine))
	}
	return strings.Join(parts, "|")
}

func canonicalInternalFindingKey(f exactInternalFindingGeometry) string {
	parts := make([]string, 0, 1+len(f.Occurrences))
	parts = append(parts, fmt.Sprintf("tc=%d", f.TokenCount))
	for _, o := range f.Occurrences {
		parts = append(parts,
			fmt.Sprintf("%s:%d-%d/%d-%d", o.Path, o.StartLine, o.EndLine, o.StartPos, o.EndPos))
	}
	return strings.Join(parts, "|")
}

// assertExactFindingProjection compares actual findings against expected
// findings as canonicalized multisets and reports every missing,
// unexpected, and multiplicity-mismatched entry.
//
// The multiset key is the full canonical projection (TokenCount plus all
// occurrence geometry). Index-based comparison is avoided so canonical-
// ordering defects do not masquerade as geometry defects.
//
// Multiplicity diagnostics explicitly report expected N and actual M along
// with the projection. Missing/unexpected diagnostics may be emitted in
// addition but never replace the multiplicity report. Diagnostic key
// iteration is sorted for deterministic output.
func assertExactFindingProjection(t *testing.T, actual []Finding, expected []exactFindingGeometry, fixtureRoot string) {
	t.Helper()

	projected := make([]exactFindingGeometry, len(actual))
	for i, f := range actual {
		projected[i] = projectFindingGeometry(t, f, fixtureRoot)
		projected[i].Occurrences = canonicalizeOccurrences(projected[i].Occurrences)
	}
	for i := range expected {
		expected[i].Occurrences = canonicalizeOccurrences(expected[i].Occurrences)
	}

	if len(projected) != len(expected) {
		t.Errorf("finding cardinality mismatch: expected %d findings, got %d",
			len(expected), len(projected))
	}

	reportMultiplicityDiffs(t, "exact geometry", expected, projected)
}

// assertExactInternalFindingGeometry compares actual grouped internal
// findings against expected grouped internal findings as canonicalized
// multisets. The grouped projection preserves finding membership so
// span-to-finding mismatches are detected.
//
// Multiplicity diagnostics explicitly report expected N and actual M. Key
// iteration is sorted for deterministic output.
func assertExactInternalFindingGeometry(t *testing.T, actual []exactInternalFindingGeometry, expected []exactInternalFindingGeometry) {
	t.Helper()

	got := make([]exactInternalFindingGeometry, len(actual))
	for i := range actual {
		got[i] = exactInternalFindingGeometry{
			TokenCount:  actual[i].TokenCount,
			Occurrences: canonicalizeInternalOccurrences(actual[i].Occurrences),
		}
	}
	want := make([]exactInternalFindingGeometry, len(expected))
	for i := range expected {
		want[i] = exactInternalFindingGeometry{
			TokenCount:  expected[i].TokenCount,
			Occurrences: canonicalizeInternalOccurrences(expected[i].Occurrences),
		}
	}

	if len(got) != len(want) {
		t.Errorf("internal finding cardinality mismatch: expected %d findings, got %d",
			len(want), len(got))
	}

	reportInternalMultiplicityDiffs(t, "internal exact geometry", want, got)
}

// countIndependentTokensForSpan is a test-owned independent oracle that
// counts non-comment Go tokens in path whose source line is in
// [startLine, endLine] (inclusive). It uses go/scanner directly and does
// NOT invoke any V4 production helper. It validates the frozen expected
// token-count constants and is not a second V4 detector.
func countIndependentTokensForSpan(t *testing.T, path string, startLine, endLine int) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	fset := token.NewFileSet()
	file := fset.AddFile(path, fset.Base(), len(data))
	var s scanner.Scanner
	s.Init(file, data, nil, 0)
	count := 0
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			continue
		}
		ln := file.Line(pos)
		if ln < startLine {
			continue
		}
		if ln > endLine {
			break
		}
		count++
	}
	return count
}
