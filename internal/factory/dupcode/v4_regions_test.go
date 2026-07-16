package dupcode

import (
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestV4AnalysisPathRebase_UpdatesAllEmbeddedIDs(t *testing.T) {
	analysis := analyzeV4Source(t, "rebase.go", "package p\nfunc outer() { _ = func() {} }\n")
	rebaseV4AnalysisPath(&analysis, "nested/rebase.go")

	if analysis.Path != "nested/rebase.go" {
		t.Fatalf("analysis path = %q", analysis.Path)
	}
	for _, region := range analysis.Regions {
		if region.Path != analysis.Path {
			t.Fatalf("region path = %q, want %q", region.Path, analysis.Path)
		}
	}
	for i, owner := range analysis.TokenOwner {
		if owner.Path != "" && owner.Path != analysis.Path {
			t.Fatalf("owner[%d] path = %q, want %q", i, owner.Path, analysis.Path)
		}
	}
}

func TestV4AnalysisPathRebase_NoAbsolutePathsRemain(t *testing.T) {
	analysis := analyzeV4Source(t, "absolute.go", "package p\nfunc f() {}\n")
	if !filepath.IsAbs(analysis.Path) {
		t.Fatalf("fixture analysis path %q is not absolute", analysis.Path)
	}
	rebaseV4AnalysisPath(&analysis, "absolute.go")

	if filepath.IsAbs(analysis.Path) {
		t.Fatalf("analysis path remains absolute: %q", analysis.Path)
	}
	for _, region := range analysis.Regions {
		if filepath.IsAbs(region.Path) {
			t.Fatalf("region path remains absolute: %q", region.Path)
		}
	}
	for _, owner := range analysis.TokenOwner {
		if owner.Path != "" && filepath.IsAbs(owner.Path) {
			t.Fatalf("owner path remains absolute: %q", owner.Path)
		}
	}
}

func TestV4AnalysisPathRebase_PublicAndInternalUseSameIDs(t *testing.T) {
	const source = "package p\nfunc outer() { _ = func() { println(1) } }\n"
	left := analyzeV4Source(t, "root-a.go", source)
	right := analyzeV4Source(t, "root-b.go", source)
	rebaseV4AnalysisPath(&left, "same.go")
	rebaseV4AnalysisPath(&right, "same.go")

	if !reflect.DeepEqual(left.Regions, right.Regions) {
		t.Fatalf("rebased regions differ:\nleft=%+v\nright=%+v", left.Regions, right.Regions)
	}
	if !reflect.DeepEqual(left.TokenOwner, right.TokenOwner) {
		t.Fatalf("rebased token owners differ")
	}
}

func TestV4AnalysisPathRebase_DistinctRootsCanonicalOutput(t *testing.T) {
	makeRoot := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		cloneCounter = 0
		writeTestFile(t, filepath.Join(root, "a.go"), makeCloneFunc("RootA", 67))
		writeTestFile(t, filepath.Join(root, "b.go"), makeCloneFunc("RootB", 67))
		return root
	}
	leftRoot := makeRoot(t)
	rightRoot := makeRoot(t)
	cfg := Config{MinLines: 40, MinTokens: 400}
	left, err := CheckRepo(leftRoot, cfg)
	if err != nil {
		t.Fatal(err)
	}
	right, err := CheckRepo(rightRoot, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("canonical public output differs across roots:\nleft=%+v\nright=%+v", left, right)
	}
	internal := v4PipelineInternal(t, leftRoot, []string{filepath.Join(leftRoot, "a.go"), filepath.Join(leftRoot, "b.go")}, cfg)
	if len(left) != len(internal) {
		t.Fatalf("public/internal finding count differs: public=%d internal=%d", len(left), len(internal))
	}
	for i := range left {
		if left[i].TokenCount != internal[i].TokenCount || len(left[i].Occurrences) != len(internal[i].Occurrences) {
			t.Fatalf("public/internal geometry differs at %d: public=%+v internal=%+v", i, left[i], internal[i])
		}
	}
}

func TestV4AnalyzedFile_AuthoritativeInventoryAlignment(t *testing.T) {
	cases := map[string]string{
		"comments":            "package p\n// comment\nfunc f() { /* block */ println(1) }\n",
		"inserted semicolons": "package p\nfunc f() {\nprintln(1)\n}\n",
		"strings":             "package p\nfunc f() { _, _ = `raw`, \"interpreted\" }\n",
		"generics":            "package p\nfunc f[T ~int](v T) T { return v }\n",
		"multiline signature": "package p\nfunc f(\na int,\nb string,\n) {}\n",
		"method":              "package p\ntype T int\nfunc (T) f() {}\n",
		"function literal":    "package p\nvar f = func() {}\n",
		"utf8 identifier":     "package p\nfunc привет() { мир := 1; _ = мир }\n",
		"no trailing newline": "package p\nfunc f() {}",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			analysis := analyzeV4Source(t, name+".go", source)
			if len(analysis.Tokens) != len(analysis.TokenOwner) ||
				len(analysis.Tokens) != len(analysis.NormalizedTokens) {
				t.Fatalf("inventory lengths tokens=%d owners=%d normalized=%d",
					len(analysis.Tokens), len(analysis.TokenOwner), len(analysis.NormalizedTokens))
			}
		})
	}
}

func TestV4SyntaxRegions_FunctionDeclarationInsertedSemicolon(t *testing.T) {
	analysis := analyzeV4Source(t, "decl.go", "package p\nfunc f() {}\n")
	region := onlyRegionOfKind(t, analysis, v4FunctionDeclarationRegion)
	if analysis.Tokens[region.EndPos] != token.SEMICOLON {
		t.Fatalf("declaration end token = %s, want ;", analysis.Tokens[region.EndPos])
	}
}

func TestV4SyntaxRegions_LiteralAssignedAtLineEnd(t *testing.T) {
	assertLiteralEndsAtBrace(t, "package p\nfunc outer() { f := func() {}\n_ = f }\n")
}

func TestV4SyntaxRegions_ImmediatelyInvokedLiteral(t *testing.T) {
	assertLiteralEndsAtBrace(t, "package p\nfunc outer() { func() {}() }\n")
}

func TestV4SyntaxRegions_LiteralInCompositeValue(t *testing.T) {
	assertLiteralEndsAtBrace(t, "package p\nfunc outer() { _ = []func(){func() {}} }\n")
}

func TestV4SyntaxRegions_NestedLiteral(t *testing.T) {
	analysis := analyzeV4Source(t, "nested.go", "package p\nfunc outer() { _ = func() { _ = func() {} } }\n")
	count := 0
	for _, region := range analysis.Regions {
		if region.Kind == v4FunctionLiteralRegion {
			count++
			assertRegionEndsAtBrace(t, analysis, region)
		}
	}
	if count != 2 {
		t.Fatalf("literal region count = %d, want 2", count)
	}
}

func TestV4SyntaxRegions_LiteralFollowedByComma(t *testing.T) {
	assertLiteralEndsAtBrace(t, "package p\nfunc outer() { _ = []func(){func() {}, func() {}} }\n")
}

func TestV4SyntaxRegions_LiteralFollowedBySelectorOrCall(t *testing.T) {
	assertLiteralEndsAtBrace(t, "package p\nfunc outer() { _ = func() func() { return func() {} }()() }\n")
}

func analyzeV4Source(t *testing.T, name, source string) v4FileAnalysis {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	analysis, err := analyzeV4File(path)
	if err != nil {
		t.Fatalf("analyze source: %v", err)
	}
	return analysis
}

func onlyRegionOfKind(t *testing.T, analysis v4FileAnalysis, kind v4SyntaxRegionKind) v4SyntaxRegion {
	t.Helper()
	var matches []v4SyntaxRegion
	for _, region := range analysis.Regions {
		if region.Kind == kind {
			matches = append(matches, region)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("regions of kind %s = %d, want 1", kind, len(matches))
	}
	return matches[0]
}

func assertLiteralEndsAtBrace(t *testing.T, source string) {
	t.Helper()
	analysis := analyzeV4Source(t, "literal.go", source)
	for _, region := range analysis.Regions {
		if region.Kind == v4FunctionLiteralRegion {
			assertRegionEndsAtBrace(t, analysis, region)
		}
	}
}

func assertRegionEndsAtBrace(t *testing.T, analysis v4FileAnalysis, region v4SyntaxRegion) {
	t.Helper()
	if analysis.Tokens[region.EndPos] != token.RBRACE {
		t.Fatalf("literal end token at %d = %s, want }", region.EndPos, analysis.Tokens[region.EndPos])
	}
	if region.EndPos+1 < len(analysis.TokenOwner) && analysis.TokenOwner[region.EndPos+1].Ordinal == region.Ordinal &&
		analysis.TokenOwner[region.EndPos+1].Path == region.Path {
		t.Fatalf("token after literal at %d is still owned by literal %+v", region.EndPos+1, region)
	}
}
