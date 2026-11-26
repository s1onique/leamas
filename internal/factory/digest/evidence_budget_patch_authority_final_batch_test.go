// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_final_batch_test.go
// holds the G2 historical-batch-framing witnesses for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 FINAL AUTHORITY CORRECTION.
//
// INDEPENDENTLY REPRODUCED DEFECT: `git cat-file --batch` emits, for
// every resolved object, a header line followed by the object's FULL
// CONTENT and a trailing delimiter:
//
//	<oid> <type> <size>\n
//	<content bytes>\n
//
// The predecessor parser read the header and, when the type was not
// "blob", CONTINUED WITHOUT CONSUMING THE CONTENT. The next record was
// therefore read from the middle of the previous object's content, so
// every subsequent record was mis-attributed.
//
// The companion files hold the A1/A2, A3/A4 and resource witnesses.
package digest

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// G2: historical batch framing.
// ---------------------------------------------------------------------------

// TestAmendment_G2_MixedBatchKeepsRecordAlignment verifies that the
// `git cat-file --batch` parser keeps RECORD ALIGNMENT when a batch
// mixes object types.
//
// The witness uses a REAL tree/blob mix, because a synthetic frame
// would not prove that the parser matches Git's actual wire format.
func TestAmendment_G2_MixedBatchKeepsRecordAlignment(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A file whose content contains a newline, so a mis-framed record
	// cannot accidentally land on a record boundary.
	writeFilePath(t, filepath.Join(dir, "alpha.txt"),
		"first line\nsecond line\nthird line\n")
	patchBudgetCommitAll(t, dir, "add alpha")

	runner := realGitRunner{}
	headOut, err := runner.Run(dir, []string{"rev-parse", "HEAD"})
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	head := strings.TrimSpace(headOut)

	queries := []string{head + "^{tree}", head + ":alpha.txt"}
	raw, err := runner.RunWithStdin(dir, []string{
		"cat-file", "--batch"}, strings.Join(queries, "\n"))
	if err != nil {
		t.Fatalf("cat-file --batch: %v", err)
	}

	// PRECONDITION: the batch really does mix object types.
	if !strings.Contains(raw, " tree ") {
		t.Fatalf("fixture setup: the batch must contain a tree record. "+
			"Raw batch:\n%q", raw)
	}
	if !strings.Contains(raw, " blob ") {
		t.Fatalf("fixture setup: the batch must contain a blob record. "+
			"Raw batch:\n%q", raw)
	}

	// POSTCONDITION: the blob that FOLLOWS the tree in the wire order
	// must be classified correctly. If the parser lost alignment, the
	// blob record is never reached and its data is absent.
	out := map[string]HistoricalClassifierData{}
	parseHistoricalBatch(raw, []string{"tree", "alpha.txt"}, out)

	blob, ok := out["alpha.txt"]
	if !ok {
		t.Fatalf("record alignment lost: the blob record that follows "+
			"the tree record was never classified. Parsed: %v. "+
			"Raw batch:\n%q", out, raw)
	}
	if !blob.OK {
		t.Fatalf("record alignment lost: the blob record that follows " +
			"the tree record was NOT classified as available")
	}
	if !strings.Contains(string(blob.Body), "second line") {
		t.Fatalf("record alignment lost: the blob body was read from "+
			"the wrong offset, got %q", string(blob.Body))
	}
	// The tree is NOT a blob: it must not be classified as one.
	if tree := out["tree"]; tree.OK {
		t.Fatalf("only appropriate object types may be classified: " +
			"the tree record was classified as an available blob")
	}
}

// TestAmendment_G2_TruncatedRecordIsNotMarkedOK verifies that an
// incomplete or malformed record is NEVER reported as a successful
// classification.
//
// A truncated capture (the capture limit was hit, the process was
// killed mid-write) leaves a record whose declared content is shorter
// than the bytes actually present. Reporting such a record as OK
// would classify a HALF-READ blob as complete evidence.
func TestAmendment_G2_TruncatedRecordIsNotMarkedOK(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{
			// The header declares 10 bytes but only 3 are present.
			name: "truncated_content",
			raw:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa blob 10\nabc",
		},
		{
			// The header itself is cut off.
			name: "truncated_header",
			raw:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa blob",
		},
		{
			// A non-numeric size.
			name: "malformed_size",
			raw:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa blob xyz\nabc",
		},
		{
			// A negative size.
			name: "negative_size",
			raw:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa blob -1\nabc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := map[string]HistoricalClassifierData{}
			parseHistoricalBatch(tc.raw, []string{"p.txt"}, out)
			got := out["p.txt"]
			if got.OK {
				t.Fatalf("a truncated or malformed record must NOT be "+
					"marked OK: %+v", got)
			}
			if len(got.Body) != 0 {
				t.Fatalf("a truncated or malformed record must not "+
					"retain a partial body: %q", string(got.Body))
			}
		})
	}
}

// TestAmendment_G2_RetainedBodyIsDetachedFromTheBatch verifies that the
// retained classifier body does NOT alias the batch backing array.
//
// The parser slices the batch buffer to produce the retained body. A
// slice of the batch array keeps the WHOLE batch alive (the capture
// limit's worth of bytes) for as long as the body is retained, and any
// later write into the batch array is observable through the slice.
//
// The retained body must therefore be an independent COPY.
func TestAmendment_G2_RetainedBodyIsDetachedFromTheBatch(t *testing.T) {
	content := "line one\nline two\n"
	raw := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa blob " +
		strconv.Itoa(len(content)) + "\n" + content + "\n"

	out := map[string]HistoricalClassifierData{}
	parseHistoricalBatch(raw, []string{"p.txt"}, out)

	got := out["p.txt"]
	if !got.OK {
		t.Fatalf("fixture setup: the record must be classified")
	}
	if string(got.Body) != content {
		t.Fatalf("fixture setup: the body must be %q, got %q",
			content, string(got.Body))
	}

	// Mutate the batch backing array. If the retained body aliases it,
	// the mutation becomes observable through the retained body.
	batch := []byte(raw)
	parseHistoricalBatch(string(batch), []string{"p.txt"}, out)
	got = out["p.txt"]
	if !got.OK {
		t.Fatalf("fixture setup: the record must be classified")
	}
	for i := range batch {
		batch[i] = 'X'
	}
	if string(got.Body) != content {
		t.Fatalf("the retained classifier body aliases the batch "+
			"backing array: mutating the batch changed the retained "+
			"body to %q", string(got.Body))
	}
}
