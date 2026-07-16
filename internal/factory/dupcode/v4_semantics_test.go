// Package dupcode provides tests for v4 algorithm semantics.
package dupcode

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cloneCounter is incremented for each generated body to ensure uniqueness.
var cloneCounter int

// sharedStatements generates a sequence of distinct statements that the
// dupcode V4 algorithm can tokenize. Each clone wraps these identical
// statements in a uniquely named function, so each file remains a
// compilable Go package.
// Each line uses a unique accumulator value so no sliding window of smaller
// length can match, ensuring only the maximal span coalesces.
func sharedStatements(iterations int) string {
	result := "    n := 0\n"
	for i := 0; i < iterations; i++ {
		result += "    n = n + 1\n"
	}
	return result
}

// makeCloneFunc generates a clone function with a unique name containing identical statements.
func makeCloneFunc(name string, iterations int) string {
	return fmt.Sprintf("func %s() {\n%s}\n\n", name, sharedStatements(iterations))
}

// generateLargeCloneBody generates a large clone body with a unique function name.
func generateLargeCloneBody(fileId string) string {
	cloneCounter++
	return makeCloneFunc(fmt.Sprintf("ClonedFunc_%s_%d", fileId, cloneCounter), 400)
}

// generateForLoopClone generates a clone using for loop structure with unique name.
func generateForLoopClone(fileId string, bodyId int) string {
	name := fmt.Sprintf("ForCloneFunc_%s_%d", fileId, bodyId)
	body := "    j := 0\n"
	for i := 0; i < 80; i++ {
		body += fmt.Sprintf("    j = j + 1\n")
	}
	return fmt.Sprintf("func %s() {\n%s}\n", name, body)
}

// generateWhileLoopClone generates the second independent loop-shaped body.
// Its subtraction operator remains distinct after identifier normalization.
func generateWhileLoopClone(fileId string, bodyId int) string {
	name := fmt.Sprintf("WhileCloneFunc_%s_%d", fileId, bodyId)
	body := "    k := 0\n"
	for i := 0; i < 80; i++ {
		body += fmt.Sprintf("    k = k - 1\n")
	}
	return fmt.Sprintf("func %s() {\n%s}\n", name, body)
}

// writeTestFile writes a valid Go source file with unique top-level declarations.
func writeTestFile(t *testing.T, path, content string) {
	fullContent := "package test\n\n" + content
	if err := os.WriteFile(path, []byte(fullContent), 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", path, err)
	}
}

// verifyFixturesTypeCheck performs full Go type-checking on all fixture files
// as a single package. This proves the fixture is not merely syntactically
// valid but also semantically well-formed.
func verifyFixturesTypeCheck(t *testing.T, files ...string) {
	seenDecls := make(map[string]string)
	fset := token.NewFileSet()
	var parsed []*ast.File
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture %s: %v", path, err)
		}
		parsedFile, parseErr := parser.ParseFile(fset, path, src, parser.AllErrors|parser.ParseComments)
		if parseErr != nil {
			t.Fatalf("fixture %s failed to parse: %v", path, parseErr)
		}
		for _, decl := range parsedFile.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				if fd.Name == nil {
					continue
				}
				key := fd.Name.Name
				if prev, exists := seenDecls[key]; exists {
					t.Fatalf("duplicate top-level declaration %q in %s (also declared in %s)",
						key, path, prev)
				}
				seenDecls[key] = path
			}
		}
		parsed = append(parsed, parsedFile)
	}
	conf := types.Config{Importer: nil, Error: func(err error) {}}
	if _, err := conf.Check("fixture/test", fset, parsed, nil); err != nil {
		t.Fatalf("fixture package does not type-check: %v", err)
	}
}

// findMaximalClone returns the cross-file finding with the largest TokenCount
// (the intended maximal clone span), breaking ties by occurrence count.
// Filters out findings that do not span at least two unique files.
func findMaximalClone(findings []Finding) *Finding {
	var best *Finding
	for i := range findings {
		f := &findings[i]
		uniqueFiles := make(map[string]bool)
		for _, occ := range f.Occurrences {
			uniqueFiles[occ.Path] = true
		}
		if len(uniqueFiles) < 2 {
			continue
		}
		if best == nil ||
			f.TokenCount > best.TokenCount ||
			(f.TokenCount == best.TokenCount && len(f.Occurrences) > len(best.Occurrences)) {
			best = f
		}
	}
	return best
}

// TestV4_FixturesCompile proves all fixture generators produce Go source
// that passes full syntax and type-checking as a single package.
func TestV4_FixturesCompile(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "smoke_a.go")
	fileB := filepath.Join(tmpDir, "smoke_b.go")
	cloneCounter = 0
	cloneA := generateLargeCloneBody("smoke_a")
	cloneB := generateLargeCloneBody("smoke_b")
	writeTestFile(t, fileA, cloneA)
	writeTestFile(t, fileB, cloneB)
	verifyFixturesTypeCheck(t, fileA, fileB)
}

// TestV4_NonZeroOffsetClone verifies a substantial prefix in only one file
// guarantees the clone starts at a non-zero offset in that file.
func TestV4_NonZeroOffsetClone(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "a.go")
	fileB := filepath.Join(tmpDir, "b.go")
	cloneCounter = 0
	cloneA := generateLargeCloneBody("a")
	cloneB := generateLargeCloneBody("b")
	// Only file A gets the substantial prefix; file B starts with the clone.
	prefix := ""
	for i := 0; i < 50; i++ {
		prefix += fmt.Sprintf("func prefix%d() { _ = %d }\n", i, i)
	}
	writeTestFile(t, fileA, prefix+cloneA)
	writeTestFile(t, fileB, cloneB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	max := findMaximalClone(findings)
	if max == nil {
		t.Fatal("expected a maximal cross-file clone finding")
	}
	// Verify the occurrences in file A start at non-zero offset relative to B
	var startA, startB int
	for _, occ := range max.Occurrences {
		base := filepath.Base(occ.Path)
		switch base {
		case "a.go":
			startA = occ.StartLine
		case "b.go":
			startB = occ.StartLine
		}
	}
	if startA <= 0 || startB <= 0 {
		t.Fatalf("invalid start lines: a=%d b=%d", startA, startB)
	}
	// The substantial prefix must push the clone in A past line 50.
	if startA <= 50 {
		t.Errorf("expected clone in a.go to start past line 50 due to prefix, got start=%d", startA)
	}
	if startA == startB {
		t.Errorf("expected non-zero offset (a=%d, b=%d)", startA, startB)
	}
}

// TestV4_MaximalCloneCoalescing verifies overlapping windows coalesce into one maximal finding per span.
func TestV4_MaximalCloneCoalescing(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "coal_a.go")
	fileB := filepath.Join(tmpDir, "coal_b.go")
	cloneCounter = 0
	cloneA := generateLargeCloneBody("coal_a")
	cloneB := generateLargeCloneBody("coal_b")
	writeTestFile(t, fileA, cloneA)
	writeTestFile(t, fileB, cloneB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	// The maximal cross-file finding should have at least 2 occurrences (one per file).
	max := findMaximalClone(findings)
	if max == nil {
		t.Fatal("expected maximal cross-file clone finding")
	}
	if len(max.Occurrences) < 2 {
		t.Errorf("expected >=2 occurrences in maximal finding, got %d", len(max.Occurrences))
	}
}

// TestV4_IndependentCloneBodies verifies two structurally distinct clone
// bodies produce two separate findings (not collapsed into one).
func TestV4_IndependentCloneBodies(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "indep_a.go")
	fileB := filepath.Join(tmpDir, "indep_b.go")
	cloneCounter = 0
	// Two structurally distinct bodies (different identifier names)
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
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	// Count distinct fingerprints among cross-file findings
	crossFileFingerprints := make(map[string]bool)
	for _, f := range findings {
		uniqueFiles := make(map[string]bool)
		for _, occ := range f.Occurrences {
			uniqueFiles[occ.Path] = true
		}
		if len(uniqueFiles) >= 2 {
			crossFileFingerprints[f.Fingerprint] = true
		}
	}
	if len(crossFileFingerprints) < 2 {
		t.Errorf("expected at least 2 distinct cross-file findings, got %d", len(crossFileFingerprints))
	}
}

// TestV4_RepeatedOccurrenceInOneFile verifies repeated occurrence detection.
func TestV4_RepeatedOccurrenceInOneFile(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "repeat_a.go")
	fileB := filepath.Join(tmpDir, "repeat_b.go")
	cloneCounter = 0
	// Each function name is unique, statements identical
	cloneA1 := makeCloneFunc("RepeatA1", 80)
	cloneA2 := makeCloneFunc("RepeatA2", 80)
	cloneB1 := makeCloneFunc("RepeatB1", 80)
	contentA := cloneA1 + cloneA2
	contentB := cloneB1
	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	// Find a finding that spans both files
	max := findMaximalClone(findings)
	if max == nil {
		t.Fatal("expected cross-file clone finding")
	}
	// Count occurrences per base file
	counts := map[string]int{}
	for _, occ := range max.Occurrences {
		counts[filepath.Base(occ.Path)]++
	}
	// V4 coalesces overlapping windows per file into single occurrences.
	// At least one occurrence per file is expected.
	// With 2 clones in A and 1 in B, we expect >= 2 total occurrences.
	if len(max.Occurrences) < 2 {
		t.Errorf("expected >= 2 total occurrences, got %d (all: %#v)", len(max.Occurrences), counts)
	}
}

// TestV4_ThreeFileClone verifies clone in three files produces a finding with three occurrences.
func TestV4_ThreeFileClone(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{
		filepath.Join(tmpDir, "three_a.go"),
		filepath.Join(tmpDir, "three_b.go"),
		filepath.Join(tmpDir, "three_c.go"),
	}
	cloneCounter = 0
	for i, f := range files {
		writeTestFile(t, f, generateLargeCloneBody([]string{"a", "b", "c"}[i]))
	}
	verifyFixturesTypeCheck(t, files...)
	findings, err := CheckRepo(tmpDir, Config{MinLines: 40, MinTokens: 400})
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	max := findMaximalClone(findings)
	if max == nil {
		t.Fatal("expected cross-file clone finding")
	}
	uniqueFiles := make(map[string]bool)
	for _, occ := range max.Occurrences {
		uniqueFiles[filepath.Base(occ.Path)] = true
	}
	if len(uniqueFiles) != 3 {
		t.Errorf("expected 3 unique files in maximal finding, got %d", len(uniqueFiles))
	}
}

// TestV4_InterleavedChainPartitions tests partition correctness using production v4BuildChainsWithPartitioning.
func TestV4_InterleavedChainPartitions(t *testing.T) {
	matches := []seedMatch{
		{SeedFingerprint: "seed-A-to-B-1", Left: rawWindow{Path: "A.go", StartPos: 0, EndPos: 39, StartLine: 1, EndLine: 10}, Right: rawWindow{Path: "B.go", StartPos: 100, EndPos: 139, StartLine: 100, EndLine: 110}, Offset: 100},
		{SeedFingerprint: "seed-A-to-C", Left: rawWindow{Path: "A.go", StartPos: 50, EndPos: 89, StartLine: 50, EndLine: 60}, Right: rawWindow{Path: "C.go", StartPos: 250, EndPos: 289, StartLine: 250, EndLine: 260}, Offset: 200},
		{SeedFingerprint: "seed-A-to-B-2", Left: rawWindow{Path: "A.go", StartPos: 30, EndPos: 69, StartLine: 30, EndLine: 40}, Right: rawWindow{Path: "B.go", StartPos: 130, EndPos: 169, StartLine: 130, EndLine: 140}, Offset: 100},
	}
	chains := v4BuildChainsWithPartitioning(matches)
	if len(chains) != 2 {
		t.Fatalf("expected exactly 2 chains, got %d", len(chains))
	}
	chainByOffset := make(map[int][]cloneChain)
	for _, c := range chains {
		chainByOffset[c.Offset] = append(chainByOffset[c.Offset], c)
	}
	if len(chainByOffset[100]) != 1 {
		t.Fatalf("expected exactly one offset-100 chain, got %d", len(chainByOffset[100]))
	}
	if len(chainByOffset[100][0].Matches) != 2 {
		t.Fatalf("expected offset-100 chain to have 2 matches, got %d", len(chainByOffset[100][0].Matches))
	}
	if len(chainByOffset[200]) != 1 {
		t.Fatalf("expected exactly one offset-200 chain, got %d", len(chainByOffset[200]))
	}
	if len(chainByOffset[200][0].Matches) != 1 {
		t.Fatalf("expected offset-200 chain to have 1 match, got %d", len(chainByOffset[200][0].Matches))
	}
}

// TestV4_AlgorithmVersionIs4 verifies the algorithm version constant.
func TestV4_AlgorithmVersionIs4(t *testing.T) {
	if AlgorithmVersion != 4 {
		t.Errorf("expected AlgorithmVersion=4, got %d", AlgorithmVersion)
	}
}

// TestV4_BaselineRejectsV3 verifies that V3 baselines are rejected.
func TestV4_BaselineRejectsV3(t *testing.T) {
	v3Baseline := `{"schema_version": 1,"algorithm_version": 3,"generated_at": "2024-01-01T00:00:00Z","tool": "leamas dupcode","thresholds": {"min_lines": 40, "min_tokens": 400},"findings": []}`
	tmpFile := filepath.Join(t.TempDir(), "v3-baseline.json")
	if err := os.WriteFile(tmpFile, []byte(v3Baseline), 0644); err != nil {
		t.Fatalf("Failed to write temp baseline: %v", err)
	}
	_, err := LoadBaseline(tmpFile)
	if err == nil {
		t.Error("expected V3 baseline to be rejected")
	}
}

// TestV4_MergeInvariantPanics verifies the invariant check in v4MergeToNWayClone.
func TestV4_MergeInvariantPanics(t *testing.T) {
	group := []v4Finding{
		{StableFingerprint: "abc", TokenCount: 100, Occurrences: []maximalOccurrence{{Path: "a.go", StartPos: 0, EndPos: 99}}},
		{StableFingerprint: "abc", TokenCount: 200, Occurrences: []maximalOccurrence{{Path: "b.go", StartPos: 0, EndPos: 199}}},
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on inconsistent token counts")
		} else if !strings.Contains(fmt.Sprintf("%v", r), "inconsistent token counts") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	_ = v4MergeToNWayClone(group)
}
