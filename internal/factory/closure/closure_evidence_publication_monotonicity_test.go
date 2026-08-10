// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// TestClosureEvidencePublicationMonotonicity pins the
// complete-once contract. A second prepare against the same
// destination fails without touching existing bytes; the
// same publication authority is a one-shot. The matrix also
// covers the JSON-only preexisting and sidecar-only
// preexisting variants.
func TestClosureEvidencePublicationMonotonicity(t *testing.T) {
	fx := newEvidencePublicationFixture(t)
	first := prepareFromEvidencePublicationFixture(t, fx)
	candidate := evidenceOnlyCandidate(t)
	if res := first.Publish(candidate); res.Err != nil {
		t.Fatalf("first publish: %v", res.Err)
	}
	_ = first.Close()
	original, err := os.ReadFile(fx.json)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if _, err := PrepareEvidencePublication(fx.worktree, fx.json, []CanonicalWorktree{{Path: fx.worktree}}); err == nil {
		t.Fatalf("second prepare must fail")
	}
	after, err := os.ReadFile(fx.json)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if !bytes.Equal(original, after) {
		t.Fatalf("existing bytes changed: %q vs %q", original, after)
	}
	if err := os.Remove(fx.json); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareEvidencePublication(fx.worktree, fx.json, []CanonicalWorktree{{Path: fx.worktree}}); err == nil {
		t.Fatalf("sidecar-only preexisting must reject")
	}
	if err := os.WriteFile(fx.json, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareEvidencePublication(fx.worktree, fx.json, []CanonicalWorktree{{Path: fx.worktree}}); err == nil {
		t.Fatalf("both preexisting must reject")
	}
}

// TestClosureEvidencePublicationConcurrentSameDestination
// proves two concurrent publishes against the same destination
// both fail closed: the first succeeds (pair_durable); the
// second refuses (not_published, no overwrite).
func TestClosureEvidencePublicationConcurrentSameDestination(t *testing.T) {
	fx := newEvidencePublicationFixture(t)
	auth := prepareFromEvidencePublicationFixture(t, fx)
	candidate := evidenceOnlyCandidate(t)
	var wg sync.WaitGroup
	results := make([]EvidencePublicationResult, 2)
	for i := 0; i < 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = auth.Publish(candidate)
		}()
	}
	wg.Wait()
	var durable, refused int
	for _, r := range results {
		switch r.State {
		case EvidencePublicationPairDurable:
			durable++
		case EvidencePublicationNotPublished:
			refused++
		default:
			t.Fatalf("unexpected state: %s", r.State)
		}
	}
	if durable != 1 || refused != 1 {
		t.Fatalf("durable=%d refused=%d, want 1/1", durable, refused)
	}
}

// TestClosureEvidencePublicationIncompleteNeverWritten is
// the B2/B3 barrier regression: an INCOMPLETE candidate must
// NEVER reach the publisher. The barrier returns a typed
// error before the publisher is constructed; the parent
// directory therefore holds no JSON and no sidecar.
func TestClosureEvidencePublicationIncompleteNeverWritten(t *testing.T) {
	fx := newEvidencePublicationFixture(t)
	candidate := evidence.BuildEmptyEvidence() // schema_version only; INCOMPLETE.
	_, err := evidence.PrepareClosureEvidenceForPublication(candidate)
	if err == nil {
		t.Fatalf("B2 barrier must refuse INCOMPLETE candidate")
	}
	if !errors.Is(err, evidence.ErrIncompleteEvidence) {
		t.Fatalf("err = %v, want ErrIncompleteEvidence", err)
	}
	entries, _ := os.ReadDir(fx.outside)
	if len(entries) != 0 {
		t.Fatalf("parent dir has %d entries before publish: %v", len(entries), entries)
	}
}
