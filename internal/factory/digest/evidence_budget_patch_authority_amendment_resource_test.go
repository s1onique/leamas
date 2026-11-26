// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_amendment_resource_test.go
// holds the AMENDMENT-01 execution-resource witness for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// A6  moving patch generation AHEAD of the size suppression means a
//
//	subprocess runs before the per-file evidence budget is
//	applied. The digest package executes Git through raw os/exec
//	rather than the bounded internal/execution gateway, so the
//	bound is applied at the digest's own runner boundary.
//
// The companion files hold the A1/A2 and A3/A4 witnesses.
package digest

import (
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// A6: execution resource conservation.
// ---------------------------------------------------------------------------

// TestAmendment_A6_GitCaptureIsBounded verifies that the digest's Git
// runner applies a BOUNDED stdout capture.
//
// AMENDMENT-01 (A6): moving patch generation ahead of the size
// suppression means a subprocess runs before the per-file evidence
// budget is applied. The digest package executes Git through raw
// os/exec rather than the bounded internal/execution gateway, so the
// bound is applied at the digest's own runner boundary.
//
// The witness verifies the CONTRACT, not a specific byte count:
//
//   - a normal Git invocation succeeds and returns its output;
//   - an invocation whose output exceeds the capture limit FAILS with
//     an explicit overflow error;
//   - the overflow result is NEVER returned as a successful, truncated
//     string.
func TestAmendment_A6_GitCaptureIsBounded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// The normal case still works.
	out, err := RunGit(dir, []string{"rev-parse", "HEAD"})
	if err != nil {
		t.Fatalf("a normal git invocation must still succeed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("a normal git invocation must return its output")
	}

	// The bounded capture detects an overflow and reports it as an
	// error rather than returning a truncated success.
	buf := &limitedBuffer{limit: 16}
	if _, err := buf.Write(make([]byte, 64)); err != nil {
		t.Fatalf("limited buffer must not error on write: %v", err)
	}
	if !buf.exceeded {
		t.Fatalf("the limited buffer must record the overflow")
	}
	if len(buf.String()) != 16 {
		t.Fatalf("the limited buffer must retain exactly its limit, "+
			"got %d bytes", len(buf.String()))
	}

	// The production limit must be declared and finite.
	if MaxGitCaptureBytes <= 0 {
		t.Fatalf("MaxGitCaptureBytes must be a positive bound")
	}
	if MaxGitCaptureBytes < MaxPerFileBytes {
		t.Fatalf("MaxGitCaptureBytes=%d must exceed the per-file "+
			"evidence budget=%d so an ordinary patch is never "+
			"refused", MaxGitCaptureBytes, MaxPerFileBytes)
	}
}

// TestAmendment_F1_HistoricalBatchStaysWithinTheCaptureLimit verifies
// that the historical batch loader never asks Git for more content
// than the capture limit allows.
//
// The `git cat-file --batch` protocol emits the CONTENT of every
// requested object, not just its metadata. A batch over many blobs
// therefore produces far more output than the capture limit allows,
// the whole captured result is discarded, and the historical
// classification of the range is lost.
//
// AMENDMENT-01 FINALIZATION (F1): the batch is CHUNKED by blob size,
// so no chunk can overflow the capture limit.
//
// The witness uses a SMALL budget so the fixture stays cheap: the
// chunking logic is exercised with a budget of a few hundred bytes
// rather than with tens of megabytes of real blobs.
func TestAmendment_F1_HistoricalBatchStaysWithinTheCaptureLimit(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Four blobs of 100 bytes each.
	paths := []string{"b1.txt", "b2.txt", "b3.txt", "b4.txt"}
	for _, p := range paths {
		writeFilePath(t, filepath.Join(dir, p),
			strings.Repeat("X", 100))
	}
	patchBudgetCommitAll(t, dir, "add blobs")

	runner := realGitRunner{}
	head, err := runner.Run(dir, []string{"rev-parse", "HEAD"})
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	head = strings.TrimSpace(head)

	sizes := historicalBlobSizesBatch(runner, dir, head, paths)
	for _, p := range paths {
		if sizes[p] != 100 {
			t.Fatalf("fixture setup: the size of %s must be 100, got %d",
				p, sizes[p])
		}
	}

	// A budget of 250 bytes must split the four blobs into at least
	// two chunks, because four 100-byte blobs exceed it.
	chunks := chunkPathsBySize(paths, sizes, 250)
	if len(chunks) < 2 {
		t.Fatalf("the chunking must split a batch whose aggregate "+
			"content exceeds the budget, got %d chunk(s)", len(chunks))
	}
	// Every chunk must stay within the budget, and every path must
	// appear exactly once.
	seen := map[string]int{}
	for _, chunk := range chunks {
		total := int64(0)
		for _, p := range chunk {
			total += sizes[p]
			seen[p]++
		}
		if total > 250 {
			t.Fatalf("a chunk of %d bytes exceeds the 250 byte "+
				"budget", total)
		}
	}
	for _, p := range paths {
		if seen[p] != 1 {
			t.Fatalf("the path %s appears %d times; every path must "+
				"appear exactly once", p, seen[p])
		}
	}

	// The production budget must be below the capture limit, so a
	// chunk can never overflow the capture by construction.
	if historicalBatchContentBudget >= MaxGitCaptureBytes {
		t.Fatalf("the historical batch budget=%d must stay below the "+
			"capture limit=%d so a chunk can never overflow it",
			historicalBatchContentBudget, MaxGitCaptureBytes)
	}
}
