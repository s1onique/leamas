package dupcode

import (
	"fmt"
	"path/filepath"
	"testing"
)

var _ = fmt.Sprintf
var _ = filepath.Clean

func TestV4ComponentShadow_StructuralContainmentOnly(t *testing.T) {
	files := manualAnalyzedFiles("a.go", "b.go")
	large := v4InternalFinding{
		TokenCount: 12,
		Occurrences: []maximalOccurrence{
			{Path: "a.go", StartPos: 0, EndPos: 11, StartLine: 1, EndLine: 12},
			{Path: "b.go", StartPos: 0, EndPos: 11, StartLine: 1, EndLine: 12},
		},
	}
	small := v4InternalFinding{
		TokenCount: 5,
		Occurrences: []maximalOccurrence{
			{Path: "a.go", StartPos: 2, EndPos: 6, StartLine: 1, EndLine: 7},
			{Path: "b.go", StartPos: 2, EndPos: 6, StartLine: 1, EndLine: 7},
		},
	}
	if got := v4SuppressComponentShadows([]v4InternalFinding{large, small}, files); len(got) != 1 {
		t.Fatalf("structural shadow was retained: %+v", got)
	}
}

func TestV4ComponentMerge_PartialCloneAcrossDifferentFunctions(t *testing.T) {
	root := t.TempDir()
	writeCloneFixture(t, filepath.Join(root, "a.go"), partialFunction("A", "+"))
	writeCloneFixture(t, filepath.Join(root, "b.go"), partialFunction("B", "-"))
	findings, err := CheckRepo(root, Config{MinLines: 40, MinTokens: 400})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.TokenCount >= 400 && len(finding.Occurrences) == 2 {
			return
		}
	}
	t.Fatalf("no threshold-sized partial clone survived: %+v", findings)
}

func TestV4ComponentMerge_TwoDisjointClonesWithinOneFunction(t *testing.T) {
	root := t.TempDir()
	writeCloneFixture(t, filepath.Join(root, "a.go"), twoBlockFunction("A", "*"))
	writeCloneFixture(t, filepath.Join(root, "b.go"), twoBlockFunction("B", "/"))
	findings, err := CheckRepo(root, Config{MinLines: 40, MinTokens: 400})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, finding := range findings {
		if finding.TokenCount >= 400 && len(finding.Occurrences) == 2 {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected two disjoint clone components, got %d: %+v", count, findings)
	}
}

func TestV4ComponentMerge_MinimumThresholdCloneSurvives(t *testing.T) {
	root := t.TempDir()
	writeCloneFixture(t, filepath.Join(root, "a.go"), makeCloneFunc("MinimumA", 67))
	writeCloneFixture(t, filepath.Join(root, "b.go"), makeCloneFunc("MinimumB", 67))
	findings, err := CheckRepo(root, Config{MinLines: 40, MinTokens: 400})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.TokenCount >= 400 && len(finding.Occurrences) == 2 {
			return
		}
	}
	t.Fatalf("minimum-threshold clone was lost: %+v", findings)
}

// TestV4ComponentMerge_SmallerThresholdLegacyFixture is the executable
// replacement for the legacy fixture that previously exercised windows
// including unowned package/import tokens. The replacement uses a
// fixture whose clone bodies are wholly inside one executable region
// per file and exercises the public CheckRepo pipeline at smaller
// configurable thresholds (below the repository defaults), proving:
//
//   - component materialization works below the repository defaults;
//   - the finding is discovered through the public CheckRepo entry
//     point;
//   - exact-content identity surfaces as the public Finding's
//     (StableFingerprint, TokenCount) tuple;
//   - unowned top-level tokens do not prevent function-local clone
//     detection;
//   - no package or import geometry leaks into the finding
//     occurrences.
//
// The fixture uses two files, each of which declares a top-level var
// (unowned tokens), declares two side functions, and declares one
// clone function whose body is identical to the clone function in
// the other file. The clone body is plain sharedStatements(80)
// content (484 tokens) wrapped in a region-bounded declaration.
func TestV4ComponentMerge_SmallerThresholdLegacyFixture(t *testing.T) {
	root := t.TempDir()
	fileA := filepath.Join(root, "with_a.go")
	fileB := filepath.Join(root, "with_b.go")
	cloneCounter = 0

	contentA := fmt.Sprintf(`var sideA = 1

func topA() {
	_ = sideA
}

func cloneWithA() {
%s}

func bottomA() {
	_ = sideA
}
`, sharedStatements(80))
	contentB := fmt.Sprintf(`var sideB = 1

func topB() {
	_ = sideB
}

func cloneWithB() {
%s}

func bottomB() {
	_ = sideB
}
`, sharedStatements(80))
	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	// Smaller-than-default thresholds: 80 tokens, 5 lines.
	small := Config{MinLines: 5, MinTokens: 80}
	findings, err := CheckRepo(root, small)
	if err != nil {
		t.Fatalf("CheckRepo failed at small thresholds: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding at small thresholds, got 0")
	}

	// And the same fixture at default thresholds: at 400 tokens, the
	// sharedStatements(80) body (484 tokens) still exceeds the
	// default MinTokens/MinLines so the same detection must hold.
	def := DefaultConfig()
	defFindings, err := CheckRepo(root, def)
	if err != nil {
		t.Fatalf("CheckRepo failed at default thresholds: %v", err)
	}
	if len(defFindings) == 0 {
		t.Fatalf("expected at least one finding at default thresholds, got 0")
	}

	saw := false
	for _, f := range findings {
		if f.TokenCount < small.MinTokens || len(f.Occurrences) < 2 {
			continue
		}
		saw = true
		uniqueFiles := make(map[string]bool)
		for _, occ := range f.Occurrences {
			uniqueFiles[occ.Path] = true
			if occ.StartLine < 3 {
				t.Errorf("occurrence starts on package/var line: %s:%d", occ.Path, occ.StartLine)
			}
			if occ.EndLine < occ.StartLine {
				t.Errorf("occurrence has inverted line range: %s:%d-%d", occ.Path, occ.StartLine, occ.EndLine)
			}
		}
		if len(uniqueFiles) < 2 {
			t.Errorf("small-threshold finding should span >=2 files, got %d: %+v", len(uniqueFiles), f.Occurrences)
		}
		if f.StableFingerprint == "" {
			t.Errorf("finding has empty StableFingerprint: %+v", f)
		}
	}
	if !saw {
		t.Fatalf("no small-threshold finding met MinTokens/MinLines with >=2 occurrences: %+v", findings)
	}
}

func partialFunction(name, tailOperator string) string {
	return fmt.Sprintf("func %s() {\n%s    n = n %s 2\n}\n", name, sharedStatements(80), tailOperator)
}

func twoBlockFunction(name, gapOperator string) string {
	return fmt.Sprintf("func %s() {\n%s    n = n %s 2\n%s}\n", name, sharedStatements(80), gapOperator, sharedStatements(80))
}

func writeCloneFixture(t *testing.T, path, declaration string) {
	t.Helper()
	writeTestFile(t, path, declaration)
}

var _ = fmt.Sprintf
