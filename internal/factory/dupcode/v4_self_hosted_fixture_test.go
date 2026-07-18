// Package dupcode provides the stable synthetic fixture used by
// the V4 forensic and pipeline tests after the self-hosted
// remediation.
//
// The ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01 refactor
// removed the canonical 504-token claim/evidence duplicate from the
// production tree. The semantic and geometry tests that used to
// read that production source as convenient test data must now read
// from a stable synthetic fixture so they no longer depend on
// production line numbers.
//
// The fixture is generated dynamically inside t.TempDir() so it is
// excluded from the live repository scan (which excludes testdata
// and every directory it walks). The rebased public paths are
// pinned to `testdata/self-hosted-remediation/...` so every test
// that observes the canonical occurrence pair can assert on a
// deterministic, repo-relative path rather than the temp-dir
// path. The rebased path is a fiction maintained by the fixture
// helper; it does not need to exist as a tracked file because the
// scan never reads it through the live walk.
//
// The canonical body uses `makeCloneFunc(name, 80)`, which yields
// exactly 491 normalized tokens per the closed-form derivation in
// v4_exact_geometry_support_test.go:
//
//	TokenCount = 7 (function wrapper) + 4 (n := 0) + 6 * 80 = 491
//
// The fixture is large enough (491 >= 400 token threshold) to
// produce exactly one canonical finding per the production
// pipeline.
package dupcode

import (
	"path/filepath"
	"testing"
)

const (
	// selfHostedFixtureLeftRelPath is the rebased public path
	// used by the self-hosted fixture's left file.
	selfHostedFixtureLeftRelPath = "testdata/self-hosted-remediation/claim_commands.go"

	// selfHostedFixtureRightRelPath is the rebased public path
	// used by the self-hosted fixture's right file.
	selfHostedFixtureRightRelPath = "testdata/self-hosted-remediation/evidence_commands.go"

	// selfHostedFixtureCanonicalTokenCount is the frozen
	// canonical token count produced by the fixture pair. It is
	// derived from `makeCloneFunc(name, 80)` per the closed-form
	// formula documented above.
	selfHostedFixtureCanonicalTokenCount = 491

	// selfHostedFixtureCanonicalLineCount is the frozen canonical
	// line count. The fixture body spans 1 line per statement,
	// plus the `func` line and closing brace line.
	selfHostedFixtureCanonicalLineCount = 83
)

// writeSelfHostedFixture writes the synthetic claim/evidence
// duplicate pair into a temp dir and returns the absolute paths.
// The pair's content is deterministic: both files contain an
// identical clone body produced by `makeCloneFunc(name, 80)`,
// guaranteeing the same canonical token count and the same
// normalized content key on every run.
//
// The fixture pads the canonical function body with non-clone
// tokens on both sides so tests that probe one-token
// extensions (e.g. CORRECTION04 maximality audits) do not run
// out of file before they run out of extension candidates. The
// padding is identical on both sides and contributes nothing to
// the canonical content key (the canonical body is still the
// entire makeCloneFunc output), so the canonical token count
// remains the closed-form 7 + 4 + 6 * 80 = 491.
// selfHostedFixtureLeftPadding and selfHostedFixtureRightPadding
// provide non-clone Go statements that flank the canonical
// function so extension-probe tests have token room to extend
// in either direction without hitting end-of-file.
//
// Each padding set is namespaced to its file (left vs right)
// because writeTestFile writes both files into the same
// `package test` namespace; identical top-level declarations
// in two files would fail the fixture type check.
func selfHostedFixtureLeftPadding() string {
	return "var leftPadA0 = 0\nvar leftPadA1 = 1\nvar leftPadA2 = 2\n"
}

func selfHostedFixtureLeftSuffix() string {
	return "var leftPadB0 = 0\nvar leftPadB1 = 1\nvar leftPadB2 = 2\n"
}

func selfHostedFixtureRightPadding() string {
	return "var rightPadA0 = 0\nvar rightPadA1 = 1\nvar rightPadA2 = 2\n"
}

func selfHostedFixtureRightSuffix() string {
	return "var rightPadB0 = 0\nvar rightPadB1 = 1\nvar rightPadB2 = 2\n"
}

func writeSelfHostedFixture(t *testing.T) (leftAbs, rightAbs string) {
	t.Helper()
	root := t.TempDir()
	leftAbs = filepath.Join(root, "claim_commands.go")
	rightAbs = filepath.Join(root, "evidence_commands.go")
	cloneCounter = 0
	leftBody := selfHostedFixtureLeftPadding() + makeCloneFunc("ClaimShowClone", 80) + selfHostedFixtureLeftSuffix()
	rightBody := selfHostedFixtureRightPadding() + makeCloneFunc("EvidenceShowClone", 80) + selfHostedFixtureRightSuffix()
	writeTestFile(t, leftAbs, leftBody)
	writeTestFile(t, rightAbs, rightBody)
	verifyFixturesTypeCheck(t, leftAbs, rightAbs)
	return leftAbs, rightAbs
}

// traceForSelfHostedFixture builds the v4PipelineTrace for the
// self-hosted fixture pair. The fixture is generated in a temp
// dir and rebased to the stable testdata paths above.
//
// The function asserts (a) the scan succeeds without error, (b)
// the trace produces a non-empty ComponentsBeforeShadow, and (c)
// the canonical final finding has the frozen token count and
// exactly two occurrences. These assertions are not the test
// invariants themselves; they are setup witnesses so any failure
// surfaced by a downstream test points at the downstream test
// rather than at scan setup.
func traceForSelfHostedFixture(t *testing.T) (
	leftFile, rightFile *v4AnalyzedFile,
	trace v4PipelineTrace,
	finals []v4InternalFinding,
) {
	t.Helper()
	leftAbs, rightAbs := writeSelfHostedFixture(t)

	leftVal, err := analyzeV4AnalyzedFile(leftAbs)
	if err != nil {
		t.Fatalf("analyze %s: %v", leftAbs, err)
	}
	rightVal, err := analyzeV4AnalyzedFile(rightAbs)
	if err != nil {
		t.Fatalf("analyze %s: %v", rightAbs, err)
	}
	rebaseV4AnalyzedFilePath(&leftVal, selfHostedFixtureLeftRelPath)
	rebaseV4AnalyzedFilePath(&rightVal, selfHostedFixtureRightRelPath)
	leftFile = &leftVal
	rightFile = &rightVal

	filesMap := map[string]*v4AnalyzedFile{
		selfHostedFixtureLeftRelPath:  leftFile,
		selfHostedFixtureRightRelPath: rightFile,
	}
	analysesMap := map[string]*v4FileAnalysis{
		selfHostedFixtureLeftRelPath:  &leftVal.Analysis,
		selfHostedFixtureRightRelPath: &rightVal.Analysis,
	}

	windowMap := make(map[string][]rawWindow)
	fingerprintTokens := make(map[string]int)
	for i, ft1 := range []fileTokens{leftVal.FileTokens, rightVal.FileTokens} {
		if len(ft1.tokens) < DefaultConfig().MinTokens {
			continue
		}
		for j := i + 1; j < 2; j++ {
			ft2 := []fileTokens{leftVal.FileTokens, rightVal.FileTokens}[j]
			if len(ft2.tokens) < DefaultConfig().MinTokens {
				continue
			}
			findCommonWindows(ft1, ft2, DefaultConfig(), windowMap, fingerprintTokens)
		}
	}

	finals, trace, err = v4BuildInternalFindingsTrace(windowMap, analysesMap, filesMap)
	if err != nil {
		t.Fatalf("v4BuildInternalFindingsTrace: %v", err)
	}
	if len(trace.ComponentsBeforeShadow) == 0 {
		t.Fatalf("self-hosted fixture setup invariant: ComponentsBeforeShadow is empty (fixture must produce at least one component)")
	}
	if len(finals) != 1 {
		t.Fatalf("self-hosted fixture setup invariant: trace must emit exactly one final finding, got %d", len(finals))
	}
	got := finals[0]
	if got.TokenCount != selfHostedFixtureCanonicalTokenCount {
		t.Fatalf("self-hosted fixture setup invariant: canonical TokenCount=%d, want %d",
			got.TokenCount, selfHostedFixtureCanonicalTokenCount)
	}
	if len(got.Occurrences) != 2 {
		t.Fatalf("self-hosted fixture setup invariant: canonical occurrence count=%d, want 2",
			len(got.Occurrences))
	}
	return leftFile, rightFile, trace, finals
}

// canonicalSelfHostedFinding returns the single canonical finding
// from the self-hosted fixture trace, asserting exactly one
// finding with the frozen token count and exactly two occurrences.
// Tests that need the canonical finding should call this helper
// instead of writing their own TokenCount assertion.
func canonicalSelfHostedFinding(t *testing.T, finals []v4InternalFinding) v4InternalFinding {
	t.Helper()
	if len(finals) != 1 {
		t.Fatalf("self-hosted fixture must emit exactly one final finding, got %d", len(finals))
	}
	got := finals[0]
	if got.TokenCount != selfHostedFixtureCanonicalTokenCount {
		t.Fatalf("canonical self-hosted finding must have TokenCount=%d, got %d",
			selfHostedFixtureCanonicalTokenCount, got.TokenCount)
	}
	if len(got.Occurrences) != 2 {
		t.Fatalf("canonical self-hosted finding must have 2 occurrences, got %d",
			len(got.Occurrences))
	}
	return got
}

// selfHostedFixtureLeftFile / selfHostedFixtureRightFile are
// convenient accessors used by tests that already have a trace
// but want to assert geometry on the analyzed files.
func selfHostedFixtureLeftFile(t *testing.T) *v4AnalyzedFile {
	t.Helper()
	left, right, _, _ := traceForSelfHostedFixture(t)
	_ = right
	return left
}

func selfHostedFixtureRightFile(t *testing.T) *v4AnalyzedFile {
	t.Helper()
	left, right, _, _ := traceForSelfHostedFixture(t)
	_ = left
	return right
}
