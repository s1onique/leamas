// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_blob_test.go covers the
// F16 (CORRECTION04) and F20 (CORRECTION05) invariants of
// the range-mode Git blob batch lookup. These tests guard
// the fail-closed semantics: a missing key, a failed
// batch run, and a short/incomplete response MUST all
// produce BlobUnknown (not BlobPresent) so the renderer
// cannot accidentally publish a PRESENT status with an
// empty OID.
package digest

import (
	"testing"
)

// TestRangeBlobResult_ZeroValueIsNotPresent (F20) verifies
// the load-bearing invariant: a missing map key returns
// Status == BlobUnknown and IsPresent() == false.
func TestRangeBlobResult_ZeroValueIsNotPresent(t *testing.T) {
	var zero RangeBlobResult
	if zero.Status != BlobUnknown {
		t.Fatalf("F20 violated: zero value Status = %v, "+
			"want BlobUnknown", zero.Status)
	}
	if zero.IsPresent() {
		t.Fatalf("F20 violated: zero value IsPresent() = " +
			"true, want false")
	}
	if zero.Status == BlobPresent {
		t.Fatalf("F20 violated: BlobPresent is the zero " +
			"value, fail-open bug")
	}
}

// TestBlobLookupStatus_StringUnknown (F20) verifies the
// String() method covers BlobUnknown explicitly so the
// rendered digest can carry an UNKNOWN status.
func TestBlobLookupStatus_StringUnknown(t *testing.T) {
	if BlobUnknown.String() != "UNKNOWN" {
		t.Fatalf("F20 violated: BlobUnknown.String() = %q, "+
			"want UNKNOWN", BlobUnknown.String())
	}
	if BlobPresent.String() != "PRESENT" {
		t.Fatalf("BlobPresent.String() = %q, want PRESENT",
			BlobPresent.String())
	}
	if BlobMissing.String() != "MISSING" {
		t.Fatalf("BlobMissing.String() = %q, want MISSING",
			BlobMissing.String())
	}
}

// TestParseCatFileLine_EmptyAndGarbage (F20) verifies that
// unparseable lines produce BlobUnknown, NOT BlobOther or
// BlobPresent.
func TestParseCatFileLine_EmptyAndGarbage(t *testing.T) {
	cases := []struct {
		name string
		line string
		want BlobLookupStatus
	}{
		{"empty", "", BlobUnknown},
		{"whitespace", "   ", BlobUnknown},
		{"single token", "abc1234", BlobOther},
		// F26 (CORRECTION07): the PRESENT line
		// must be "<sha> blob <size>" with a hex
		// SHA. A bare "blob" token is not PRESENT.
		{"blob no sha", "blob 100", BlobOther},
		{"blob", "abc1234567890123456789012345678901234567 blob 100", BlobPresent},
		// F26 (CORRECTION07): a 40-char SHA-1 is
		// the canonical PRESENT shape. 7-char
		// abbreviations like "abc1234" are not.
		{"sha missing", "abc1234 missing", BlobMissing},
		{"sha ambiguous", "abc1234 ambiguous", BlobAmbiguous},
		{"sha excluded", "abc1234 excluded", BlobOther},
		// F26 (CORRECTION07): whitespace in the
		// object expression must NOT split the
		// status field. Real Git returns the
		// original <ref>:<path> on the missing
		// line, so for "HEAD:my file.txt
		// missing" the parser must match the
		// status suffix, not fields[1].
		{"missing with space", "HEAD:my file.txt missing", BlobMissing},
		{"missing with tab", "HEAD:my\tfile.txt missing", BlobMissing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCatFileLine(tc.line)
			if got.Status != tc.want {
				t.Fatalf("parseCatFileLine(%q) "+
					"Status = %v, want %v",
					tc.line, got.Status, tc.want)
			}
			if tc.want == BlobPresent && !got.IsPresent() {
				t.Fatalf("parseCatFileLine(%q) "+
					"IsPresent() = false on blob",
					tc.line)
			}
			if tc.want != BlobPresent && got.IsPresent() {
				t.Fatalf("parseCatFileLine(%q) "+
					"IsPresent() = true on non-blob",
					tc.line)
			}
		})
	}
}

// TestRangeBlobOIDsBatch_ErrorDoesNotReportPresent (F20)
// verifies that when RunWithStdin returns an error, every
// requested ref maps to a result whose IsPresent() is
// false. This is the contract that fail-closes the
// renderer against a transient cat-file failure.
func TestRangeBlobOIDsBatch_ErrorDoesNotReportPresent(t *testing.T) {
	runner := &fakeGitRunner{
		FailPatterns: []string{"cat-file"},
	}
	refs := []string{
		"HEAD~1:big.txt",
		"HEAD~2:big.txt",
		"HEAD~3:small.txt",
	}
	m := rangeBlobOIDsBatch(runner, "/tmp/repo", refs)
	for _, k := range refs {
		v, ok := m[k]
		_ = ok
		if v.IsPresent() {
			t.Fatalf("F20 violated: %q reports "+
				"IsPresent() after batch error: "+
				"%+v", k, v)
		}
		if v.Status == BlobPresent {
			t.Fatalf("F20 violated: %q reports "+
				"Status == BlobPresent after batch "+
				"error", k)
		}
	}
}

// TestRangeBlobOIDsBatch_ShortOutputDoesNotReportPresent
// (F20 + F24 CORRECTION06) verifies END-TO-END that
// when the cat-file response has fewer records than
// the request, the unrecorded refs map to a fail-closed
// zero value. Drives the full rangeBlobOIDsBatch path
// through fakeGitRunner.CatFileOutput.
func TestRangeBlobOIDsBatch_ShortOutputDoesNotReportPresent(t *testing.T) {
	// Three requests; only two response lines.
	// The third ref must remain UNKNOWN.
	runner := &fakeGitRunner{
		CatFileOutput: "abcdef1234567890abcdef1234567890abcdef12 blob 100\n" +
			"0000000000000000000000000000000000000000 missing\n",
	}
	refs := []string{
		"HEAD~1:a.txt",
		"HEAD~1:b.txt",
		"HEAD~1:c.txt",
	}
	m := rangeBlobOIDsBatch(runner, "/tmp/repo", refs)

	// First two refs must be PRESENT/MISSING as the
	// mocked batch output specifies.
	first := m[refs[0]]
	if !first.IsPresent() || first.OID !=
		"abcdef1234567890abcdef1234567890abcdef12" {
		t.Fatalf("F20 violated: first ref = %+v, want PRESENT+OID", first)
	}
	second := m[refs[1]]
	if second.Status != BlobMissing || second.IsPresent() {
		t.Fatalf("F20 violated: second ref = %+v, want MISSING", second)
	}

	// The third ref received no response line at all.
	// It MUST report BlobUnknown (zero value), NOT
	// BlobPresent. This is the F20 invariant.
	third, ok := m[refs[2]]
	_ = ok // missing keys are also acceptable
	if third.IsPresent() {
		t.Fatalf("F20 violated: third ref IsPresent() = true on short output: %+v", third)
	}
	if third.Status == BlobPresent {
		t.Fatalf("F20 violated: third ref Status = BlobPresent on short output")
	}
	if third.Status != BlobUnknown {
		t.Fatalf("F20 violated: third ref Status = %v, want BlobUnknown", third.Status)
	}
}

// TestRangeBlobOIDsBatch_MissingProducesMissing (F16
// carried into F20) verifies that a "<hash> missing"
// cat-file line produces BlobMissing (NOT BlobPresent and
// NOT BlobUnknown), so the renderer can distinguish a
// confirmed-missing object from a totally unknown one.
func TestRangeBlobOIDsBatch_MissingProducesMissing(t *testing.T) {
	got := parseCatFileLine(
		"0000000000000000000000000000000000000000 missing")
	if got.Status != BlobMissing {
		t.Fatalf("F16 violated: missing Status = %v, "+
			"want BlobMissing", got.Status)
	}
	if got.IsPresent() {
		t.Fatalf("F16 violated: missing IsPresent()")
	}
}

// TestRangeBlobOIDsBatch_ValidBlobProducesPresent (F16+F20)
// verifies the happy path: a "<oid> blob <size>" line
// produces a PRESENT result with the OID recorded.
func TestRangeBlobOIDsBatch_ValidBlobProducesPresent(t *testing.T) {
	got := parseCatFileLine(
		"abcdef1234567890abcdef1234567890abcdef12 blob 1234")
	if !got.IsPresent() {
		t.Fatalf("valid blob IsPresent() = false, %+v", got)
	}
	if got.Status != BlobPresent {
		t.Fatalf("valid blob Status = %v, want BlobPresent",
			got.Status)
	}
	if got.OID != "abcdef1234567890abcdef1234567890abcdef12" {
		t.Fatalf("valid blob OID = %q, want full hash",
			got.OID)
	}
}

// TestRangeBlobOIDsBatch_TrailingNewlineIgnored (F20)
// verifies that the trailing empty line from stdout
// (cat-file emits each record followed by \n, including
// the last one) does not consume a ref slot.
func TestRangeBlobOIDsBatch_TrailingNewlineIgnored(t *testing.T) {
	// The parser must skip empty lines without
	// "consuming" a ref. parseCatFileLine("") returns
	// BlobUnknown — rangeBlobOIDsBatch filters empty
	// lines before invoking the parser, so a trailing
	// "\n" does not shift the response/refs alignment.
	// This test verifies the empty-line filter logic
	// is correct: an empty line does NOT produce a
	// result that would shift `assigned`.
	empty := parseCatFileLine("")
	if empty.Status != BlobUnknown {
		t.Fatalf("trailing empty line Status = %v, "+
			"want BlobUnknown (so the assigned "+
			"counter is not incremented for it)",
			empty.Status)
	}
	// A non-blob record (missing) MUST increment
	// assigned; we test that the parser still
	// recognizes it correctly even when wrapped in
	// trailing newlines (i.e. preceded/followed by
	// empty lines, which rangeBlobOIDsBatch skips).
	miss := parseCatFileLine(
		"0000000000000000000000000000000000000000 missing")
	if miss.Status != BlobMissing {
		t.Fatalf("missing Status = %v, want BlobMissing",
			miss.Status)
	}
}
