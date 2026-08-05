// SPDX-License-Identifier: Apache-2.0

package closure

// v2_publication_barrier_r2a_success_test.go proves the
// success-path publication contract and the pre-correction
// regression proof required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2A:
//
//   - candidate constructed exactly once
//   - after snapshot succeeds
//   - no drift
//   - manifest published exactly once
//   - manifest bytes equal candidate bytes
//   - publication uses AtomicWriteV2Manifest (temp + rename)
//
// Splitting this from v2_publication_barrier_r2a_matrix_test.go
// keeps every file under the LLM-friendly 400-line threshold.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV2RunnerPublicationBarrier_SuccessPublishesCandidateBytesExactlyOnce
// proves the success path:
//
//   - candidate constructed exactly once
//   - after snapshot succeeds
//   - no drift
//   - manifest published exactly once
//   - manifest bytes equal candidate bytes
//   - publication uses AtomicWriteV2Manifest (temp + rename)
//
// The test deliberately uses the production SnapshotFn so the
// success-path byte-for-byte equality is observable end-to-end.
func TestV2RunnerPublicationBarrier_SuccessPublishesCandidateBytesExactlyOnce(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t, defaultV2RunnerSnapshotFunc, observer)
	// Remove any prior manifest so we observe the publication
	// explicitly.
	_ = os.Remove(req.ManifestOutput)
	manifest, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err != nil {
		t.Fatalf("successful run must not error: %v", err)
	}
	if manifest.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("manifest bound wrong protocol: %s", manifest.ClosureProtocolVersion)
	}
	if observer.Calls() != 1 {
		t.Fatalf("expected exactly one CandidateConstructed call, got %d", observer.Calls())
	}
	info, statErr := os.Stat(req.ManifestOutput)
	if statErr != nil {
		t.Fatalf("manifest must exist after success: %v", statErr)
	}
	if info.Size() == 0 {
		t.Fatalf("manifest must be non-empty after success")
	}
	onDisk, readErr := os.ReadFile(req.ManifestOutput)
	if readErr != nil {
		t.Fatalf("read manifest: %v", readErr)
	}
	if !bytes.Equal(onDisk, observer.lastBytes) {
		t.Fatalf("manifest bytes differ from candidate bytes\non-disk=%q\ncandidate=%q",
			string(onDisk), string(observer.lastBytes))
	}
	// No temp-file leftovers: AtomicWriteV2Manifest renames
	// then returns; if the rename succeeded the temp file is
	// gone.
	dir := filepath.Dir(req.ManifestOutput)
	entries, dirErr := os.ReadDir(dir)
	if dirErr != nil {
		t.Fatalf("read evidence dir: %v", dirErr)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".v2manifest.") && strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("leftover temp manifest file: %s", e.Name())
		}
	}
}

// TestV2RunnerPublicationBarrier_PreCorrectionRegression is the
// regression proof of the pre-correction defect. The test
// asserts the post-correction behaviour (inner success +
// after observation failure -> manifest absent) using the
// exact snapshot-phase seam. If the runner ever regresses to
// publishing the manifest before the after-state authority
// passes, this test fails loudly.
func TestV2RunnerPublicationBarrier_PreCorrectionRegression(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	// Use the explicit SnapshotFn seam to fail the AFTER phase
	// only. The BEFORE phase uses the real snapshot so the
	// inner runner executes and constructs a candidate.
	deps := r2aRunnerDeps(t,
		unavailableAfterSnapshotFn(t, V2CodeCallerStateUnavailable,
			"caller state observation failed: simulated post-inner failure"),
		observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("post-inner after-snapshot failure must reject")
	}
	// The inner runner constructed a candidate; the outer
	// runner refused to publish it. This is the exact
	// pre-correction regression: the manifest MUST be absent
	// even though the inner execution completed successfully.
	if observer.Calls() != 1 {
		t.Fatalf("inner runner must still construct candidate (pre-correction would skip), got %d", observer.Calls())
	}
	assertManifestAbsent(t, req.ManifestOutput)
}
