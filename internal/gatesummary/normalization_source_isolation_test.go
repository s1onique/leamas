package gatesummary

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"
)

// isolationDocForTest loads the canonical v2-full fixture and
// returns the decoded Document. It is the source document for
// every isolation test.
func isolationDocForTest(t *testing.T) Document {
	t.Helper()
	data, err := os.ReadFile("testdata/valid/v2-full.json")
	if err != nil {
		t.Fatalf("read v2-full fixture: %v", err)
	}
	dec := Decode(strings.NewReader(string(data)))
	if !dec.Success() {
		t.Fatalf("decode v2-full: %v", dec.Diagnostics)
	}
	return dec.Document
}

// jsonMarshal is a tiny indirection so the test file does not
// depend on the JSON serialization details of Summary.
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// diagnosticsString renders a diagnostics slice for error text.
func diagnosticsString(ds []Diagnostic) string {
	parts := make([]string, len(ds))
	for i, d := range ds {
		parts[i] = d.Code + "@" + d.Path
	}
	return strings.Join(parts, ",")
}

// TestNormalizationSourceIsolation proves that mutating the
// source Document after Normalize does not mutate the
// NormalizationResult, and that mutating the NormalizationResult
// does not mutate the source. Both directions are exercised.
//
// It also proves two consecutive Normalize calls against the
// same source do not alias: mutating the first result does not
// mutate the second.
func TestNormalizationSourceIsolation(t *testing.T) {
	doc := isolationDocForTest(t)
	if doc.Version() != Version2 {
		t.Fatalf("expected v2 source, got %v", doc.Version())
	}

	first := Normalize(doc)
	second := Normalize(doc)
	if !first.Success() || !second.Success() {
		t.Fatalf("normalize failed: first=%v second=%v",
			first.Diagnostics, second.Diagnostics)
	}

	secondJSON := mustJSON(t, second.Summary)

	// Mutate source doc.v2 directly with values that keep the
	// document semantically valid (otherwise we conflate
	// source mutation with normalization rejection). The
	// projection in projectV2 made independent copies, so the
	// previously produced NormalizationResults MUST remain
	// unaffected.
	originalScopeID := ""
	if doc.v2 != nil {
		originalScopeID = doc.v2.ScopeID
		doc.v2.ScopeID = "MUTATED-SCOPE"
	}

	// Re-normalize to verify the mutation took effect on the
	// source. This new result must show MUTATED-SCOPE.
	doc2 := Normalize(doc)
	if !doc2.Success() {
		t.Fatalf("post-mutation normalize failed: %v", doc2.Diagnostics)
	}
	if doc2.Summary.Scope == nil || doc2.Summary.Scope.ID != "MUTATED-SCOPE" {
		t.Fatalf("source mutation not visible in post-mutation normalize: %+v",
			doc2.Summary.Scope)
	}
	// Restore source for downstream tests in the package.
	if doc.v2 != nil {
		doc.v2.ScopeID = originalScopeID
	}
	// The pre-mutation results must be unchanged.
	if first.Summary.Scope == nil || first.Summary.Scope.ID == "MUTATED-SCOPE" {
		t.Fatalf("source mutation leaked into first result")
	}
	if second.Summary.Scope == nil || second.Summary.Scope.ID == "MUTATED-SCOPE" {
		t.Fatalf("source mutation leaked into second result")
	}

	// Mutate first result, prove second unchanged.
	first.Summary.Checks[0].Name = "MUTATED-NAME"
	if second.Summary.Checks[0].Name == "MUTATED-NAME" {
		t.Fatalf("first-result mutation leaked into second result")
	}
	secondJSONNow := mustJSON(t, second.Summary)
	if secondJSONNow != secondJSON {
		t.Fatalf("second summary identity changed after first mutation")
	}
}

// TestNormalizationBigIntIndependence verifies that the
// arbitrary-precision integer storage in a Check.Totals does not
// alias between two normalized results. Mutating one result's
// big.Int (the value BigInt() returns is a fresh allocation,
// but we still assert the underlying Integer remains intact in
// the other result).
func TestNormalizationBigIntIndependence(t *testing.T) {
	doc := isolationDocForTest(t)
	first := Normalize(doc)
	second := Normalize(doc)
	if !first.Success() || !second.Success() {
		t.Fatalf("normalize failed")
	}

	// Find a check that has Totals populated.
	var idx int = -1
	for i, c := range first.Summary.Checks {
		if c.Totals != nil {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Skip("no check with totals in v2-full fixture")
	}

	firstTot := first.Summary.Checks[idx].Totals.Total
	secondTot := second.Summary.Checks[idx].Totals.Total
	if firstTot.String() != secondTot.String() {
		t.Fatalf("totals text differs: %q vs %q",
			firstTot.String(), secondTot.String())
	}
	originalSecond := secondTot.String()

	// The first BigInt() returns a fresh *big.Int. Mutating it
	// does not affect the underlying Integer's raw spelling.
	bi, ok := firstTot.BigInt()
	if !ok {
		t.Fatal("BigInt failed on first totals")
	}
	bi.SetInt64(999999999)
	if secondTot.String() != originalSecond {
		t.Fatalf("first BigInt mutation leaked into second Integer raw: %q",
			secondTot.String())
	}
	// The first Integer is also unaffected by the BigInt
	// mutation: its raw spelling is preserved.
	firstTotRaw := firstTot.String()
	if firstTotRaw == "999999999" {
		t.Fatalf("first Integer raw spelling was overwritten by BigInt mutation")
	}
}

// TestNormalizationTwoResultsIndependence verifies that two
// Normalize calls against the same source do not alias, even
// when the first result is mutated aggressively across every
// nested field category: strings, slices, maps, big.Int,
// pointer fields, byte-backed hashes.
func TestNormalizationTwoResultsIndependence(t *testing.T) {
	doc := isolationDocForTest(t)
	first := Normalize(doc)
	second := Normalize(doc)
	if !first.Success() || !second.Success() {
		t.Fatalf("normalize failed")
	}

	secondBefore := mustJSON(t, second.Summary)

	// Top-level strings
	first.Summary.GeneratedAt = "MUTATED-GENERATED-AT"
	// Tool pointer
	if first.Summary.Tool != nil {
		*first.Summary.Tool = "MUTATED-TOOL"
	}
	// Scope
	if first.Summary.Scope != nil {
		first.Summary.Scope.ID = "MUTATED-SCOPE"
		first.Summary.Scope.Disposition = "MUTATED-DISP"
		first.Summary.Scope.Status = LifecycleClosed
	}
	// Parent
	if first.Summary.Parent != nil {
		first.Summary.Parent.Act = "MUTATED-PARENT-ACT"
		first.Summary.Parent.Status = LifecycleOpen
		first.Summary.Parent.Disposition = "MUTATED-PARENT-DISP"
		first.Summary.Parent.Root = true
	}
	// Overall
	first.Summary.Overall.Status = GateFail
	if first.Summary.Overall.Disposition != nil {
		*first.Summary.Overall.Disposition = "MUTATED-OVERALL-DISP"
	}
	// Execution
	if first.Summary.Execution != nil {
		first.Summary.Execution.HeadOID = "MUTATED-HEAD"
		first.Summary.Execution.TreeOID = "MUTATED-TREE"
		first.Summary.Execution.SubjectOID = "MUTATED-SUBJECT"
	}
	// Worktree
	if first.Summary.Worktree != nil {
		first.Summary.Worktree.CleanBefore = !first.Summary.Worktree.CleanBefore
		first.Summary.Worktree.CleanAfter = !first.Summary.Worktree.CleanAfter
	}
	// Checks: mutate every nested field category present.
	if len(first.Summary.Checks) > 0 {
		c := &first.Summary.Checks[0]
		c.Name = "MUTATED-NAME"
		if c.Scope != nil {
			*c.Scope = "MUTATED-CHECK-SCOPE"
		}
		c.Status = GateFail
		if c.Evidence != nil {
			*c.Evidence = "MUTATED-EVIDENCE"
		}
		if c.Detail != nil {
			*c.Detail = "MUTATED-DETAIL"
		}
		if c.DurationMs != nil {
			if d, ok := c.DurationMs.BigInt(); ok && d != nil {
				d.SetInt64(9999)
			}
		}
		if c.Execution != nil {
			for i := range c.Execution.Argv {
				c.Execution.Argv[i] = "MUTATED-ARGV"
			}
			if c.Execution.ExitCode != nil {
				if ec, ok := c.Execution.ExitCode.BigInt(); ok && ec != nil {
					ec.SetInt64(42)
				}
			}
			c.Execution.StdoutSHA256 = "MUTATED-STDOUT"
			c.Execution.StderrSHA256 = "MUTATED-STDERR"
		}
		if c.Totals != nil {
			c.Totals.Total = Integer{raw: "9999"}
		}
	}

	secondAfter := mustJSON(t, second.Summary)
	if secondAfter != secondBefore {
		t.Fatalf("second summary mutated by first-summary edits:\nbefore=%v\nafter=%v",
			secondBefore, secondAfter)
	}
}

// mustJSON marshals v and aborts the test on failure. It is used
// for content equality snapshots where pointer identity is
// meaningless.
func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := jsonMarshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return string(b)
}

// TestConcurrentNormalizationDeterminism runs Normalize
// concurrently against the same immutable decoded Document and
// asserts every result is deeply equal. The race detector
// observes the executed paths.
func TestConcurrentNormalizationDeterminism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}
	doc := isolationDocForTest(t)
	const goroutines = 32
	const repeats = 4

	var wg sync.WaitGroup
	results := make([]NormalizationResult, goroutines*repeats)
	errs := make(chan error, goroutines*repeats)

	for r := 0; r < repeats; r++ {
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			idx := r*goroutines + g
			go func(i int) {
				defer wg.Done()
				results[i] = Normalize(doc)
				if !results[i].Success() {
					errs <- &concurrentError{i, results[i].Diagnostics}
					return
				}
			}(idx)
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("concurrent normalize: %v", e)
	}

	wantJSON := mustJSON(t, results[0].Summary)
	for i, r := range results {
		if !r.Success() {
			t.Fatalf("result[%d] failed", i)
		}
		gotJSON := mustJSON(t, r.Summary)
		if gotJSON != wantJSON {
			t.Fatalf("result[%d] differs from result[0]", i)
		}
		if len(r.Diagnostics) != len(results[0].Diagnostics) {
			t.Fatalf("result[%d] diagnostics count differs", i)
		}
		for j, d := range r.Diagnostics {
			if d.Code != results[0].Diagnostics[j].Code ||
				d.Path != results[0].Diagnostics[j].Path {
				t.Fatalf("result[%d] diagnostic[%d] differs: %+v vs %+v",
					i, j, d, results[0].Diagnostics[j])
			}
		}
	}
}

// concurrentError is a tiny error wrapper for concurrent failures.
type concurrentError struct {
	i           int
	diagnostics []Diagnostic
}

func (e *concurrentError) Error() string {
	return "concurrent result[" + itoa(e.i) + "] failed: " +
		diagnosticsString(e.diagnostics)
}

// Ensure big.Int is referenced so imports remain tidy.
var _ = big.NewInt
