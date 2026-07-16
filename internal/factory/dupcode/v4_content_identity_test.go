package dupcode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestV4ExactContent_SameBodyDifferentPaths(t *testing.T) {
	left := analyzedContentFixture(t, "left.go", "package p\nfunc left() { x := 1; _ = x }\n")
	right := analyzedContentFixture(t, "right.go", "package p\nfunc right() { x := 1; _ = x }\n")
	leftOcc := firstDeclarationOccurrence(t, left)
	rightOcc := firstDeclarationOccurrence(t, right)
	leftKey := exactKeyForTestOccurrence(t, left, leftOcc)
	rightKey := exactKeyForTestOccurrence(t, right, rightOcc)
	if leftKey != rightKey {
		t.Fatalf("same normalized body keys differ: left=%+v right=%+v", leftKey, rightKey)
	}
}

func TestV4ExactContent_ReversedPairOrientation(t *testing.T) {
	left := analyzedContentFixture(t, "left.go", "package p\nfunc f() { x := 1; _ = x }\n")
	right := analyzedContentFixture(t, "right.go", "package p\nfunc f() { x := 1; _ = x }\n")
	leftOcc := firstDeclarationOccurrence(t, left)
	rightOcc := firstDeclarationOccurrence(t, right)
	forward := cloneChainForTest(leftOcc, rightOcc)
	reverse := cloneChainForTest(rightOcc, leftOcc)
	forwardEvidence, err := v4PairEvidenceFromChain(forward, map[string]*v4AnalyzedFile{
		"left.go": &left, "right.go": &right,
	})
	if err != nil {
		t.Fatal(err)
	}
	reverseEvidence, err := v4PairEvidenceFromChain(reverse, map[string]*v4AnalyzedFile{
		"left.go": &left, "right.go": &right,
	})
	if err != nil {
		t.Fatal(err)
	}
	if forwardEvidence.ContentKey != reverseEvidence.ContentKey {
		t.Fatalf("orientation changed content key: forward=%+v reverse=%+v", forwardEvidence.ContentKey, reverseEvidence.ContentKey)
	}
}

func TestV4ExactContent_ShiftedSourceLines(t *testing.T) {
	left := analyzedContentFixture(t, "left.go", "package p\nfunc f() { x := 1; _ = x }\n")
	right := analyzedContentFixture(t, "right.go", "package p\n\n\n// shifted\nfunc f() { x := 1; _ = x }\n")
	leftKey := exactKeyForTestOccurrence(t, left, firstDeclarationOccurrence(t, left))
	rightKey := exactKeyForTestOccurrence(t, right, firstDeclarationOccurrence(t, right))
	if leftKey != rightKey {
		t.Fatalf("source-line shift changed content key: left=%+v right=%+v", leftKey, rightKey)
	}
}

func TestV4ExactContent_AdditionAndSubtractionDiffer(t *testing.T) {
	addition := analyzedContentFixture(t, "addition.go", "package p\nfunc f() { x := 1; x = x + 1; _ = x }\n")
	subtraction := analyzedContentFixture(t, "subtraction.go", "package p\nfunc f() { x := 1; x = x - 1; _ = x }\n")
	left := exactKeyForTestOccurrence(t, addition, firstDeclarationOccurrence(t, addition))
	right := exactKeyForTestOccurrence(t, subtraction, firstDeclarationOccurrence(t, subtraction))
	if left == right || left.Digest == right.Digest {
		t.Fatalf("addition and subtraction unexpectedly share exact key: %+v", left)
	}
}

func TestV4ExactContent_StrictPrefixDiffers(t *testing.T) {
	short := analyzedContentFixture(t, "short.go", "package p\nfunc f() { x := 1; _ = x }\n")
	long := analyzedContentFixture(t, "long.go", "package p\nfunc f() { x := 1; _ = x; x = x + 1 }\n")
	left := exactKeyForTestOccurrence(t, short, firstDeclarationOccurrence(t, short))
	right := exactKeyForTestOccurrence(t, long, firstDeclarationOccurrence(t, long))
	if left == right || left.TokenCount == right.TokenCount {
		t.Fatalf("strict prefix unexpectedly shares exact identity: short=%+v long=%+v", left, right)
	}
}

func TestV4ExactContent_FrozenIndependentBodyFingerprints(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		want   string
	}{
		{"addition", generateForLoopClone("identity", 1), wantForLoopStableFingerprint},
		{"subtraction", generateWhileLoopClone("identity", 2), wantWhileLoopStableFingerprint},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := analyzedContentFixture(t, test.name+".go", "package p\n\n"+test.source)
			key := exactKeyForTestOccurrence(t, file, firstDeclarationOccurrence(t, file))
			if got := v4StableFingerprintForContentKey(key); got != test.want {
				t.Fatalf("stable fingerprint=%s, want frozen %s", got, test.want)
			}
		})
	}
}

func analyzedContentFixture(t *testing.T, name, source string) v4AnalyzedFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := writeFileForContentTest(path, source); err != nil {
		t.Fatal(err)
	}
	file, err := analyzeV4AnalyzedFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rebaseV4AnalyzedFilePath(&file, name)
	return file
}

func writeFileForContentTest(path, source string) error {
	return os.WriteFile(path, []byte(source), 0o600)
}

func firstDeclarationOccurrence(t *testing.T, file v4AnalyzedFile) maximalOccurrence {
	t.Helper()
	for _, region := range file.Analysis.Regions {
		if region.Kind == v4FunctionDeclarationRegion {
			return maximalOccurrence{
				Path: file.FileTokens.path, StartPos: region.StartPos, EndPos: region.EndPos,
				StartLine: region.StartLine, EndLine: region.EndLine,
			}
		}
	}
	t.Fatal("no declaration region")
	return maximalOccurrence{}
}

func exactKeyForTestOccurrence(t *testing.T, file v4AnalyzedFile, occurrence maximalOccurrence) v4ExactContentKey {
	t.Helper()
	key, err := v4ExactContentKeyForOccurrence(file, occurrence)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func cloneChainForTest(left, right maximalOccurrence) cloneChain {
	return cloneChain{
		Matches:    []seedMatch{{Left: rawWindow{Path: left.Path, StartPos: left.StartPos, EndPos: left.EndPos}, Right: rawWindow{Path: right.Path, StartPos: right.StartPos, EndPos: right.EndPos}}},
		LeftRange:  tokenRange{StartPos: left.StartPos, EndPos: left.EndPos},
		RightRange: tokenRange{StartPos: right.StartPos, EndPos: right.EndPos},
	}
}
