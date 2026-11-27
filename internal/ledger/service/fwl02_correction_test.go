package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// This file qualifies the FWL02 service-layer corrections against the
// REAL production seams.
//
// F3 — checkpoint lifecycle bypass.
// F4 — stale artifact acceptance.

// TestF3CheckpointCannotCloseAPendingObligation witnesses F3.
//
// The contract: a pending work item may NOT be closed through Checkpoint.
// The checkpoint path reuses the SAME lifecycle-transition authority as
// the generic update path.
func TestF3CheckpointCannotCloseAPendingObligation(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "pending work")
	closed := domain.LifecycleClosed

	_, err := svc.Checkpoint("tx-f3", CheckpointOptions{
		ScopedID: created.Scoped(), ExpectedRev: created.Revision,
		Lifecycle: closed, Summary: "closing it through the checkpoint",
	})
	if err == nil {
		t.Fatal("the checkpoint closed a pending obligation")
	}
	if !errors.Is(err, ErrUnfinishedObligation) {
		t.Fatalf("expected ErrUnfinishedObligation, got %v", err)
	}

	state, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := state.ItemByScoped(created.Scoped())
	if item.Lifecycle == domain.LifecycleClosed {
		t.Fatal("the pending obligation was closed")
	}
}

// TestF3CheckpointCannotArchiveAPendingObligation witnesses F3 for the
// archived state.
//
// The contract: the archived state is not reachable through the
// checkpoint at all. The archival operation is the only path into it.
func TestF3CheckpointCannotArchiveAPendingObligation(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "pending work")
	archived := domain.LifecycleArchived

	_, err := svc.Checkpoint("tx-f3-arc", CheckpointOptions{
		ScopedID: created.Scoped(), ExpectedRev: created.Revision,
		Lifecycle: archived,
	})
	if err == nil {
		t.Fatal("the checkpoint archived a pending obligation")
	}
	state, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := state.ItemByScoped(created.Scoped())
	if item.Lifecycle == domain.LifecycleArchived || item.Archived {
		t.Fatal("the pending obligation was archived")
	}
}

// TestF3CheckpointDoesNotRetireAnUnfinishedObligationWithAHistoricalDisposition
// witnesses F3.
//
// The contract: a preserved HISTORICAL terminal disposition is HISTORICAL
// FACT, never an authority. It stays conserved on the record and it stays
// visible there, but it may NOT retire the current unfinished obligation:
// the checkpoint may not move the record into a terminal state while its
// acceptance obligation is still outstanding.
func TestF3CheckpointDoesNotRetireAnUnfinishedObligationWithAHistoricalDisposition(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "historical record")

	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	historical := item
	historical.TerminalDisposition = "CLOSED_INSUFFICIENT"
	historical.SourceStatus = "CLOSED_INSUFFICIENT"
	historical.Revision++
	txn, err := svc.Store.BeginMutation("tx-f3-hist")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := txn.PutItem(historical); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := txn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	closed := domain.LifecycleClosed
	_, err = svc.Checkpoint("tx-f3-hist-cp", CheckpointOptions{
		ScopedID: created.Scoped(), ExpectedRev: historical.Revision,
		Lifecycle: closed, Summary: "the historical disposition is conserved",
	})
	if err == nil {
		t.Fatal("the historical disposition retired an unfinished obligation")
	}
	if !errors.Is(err, ErrUnfinishedObligation) {
		t.Fatalf("expected ErrUnfinishedObligation, got %v", err)
	}

	// The historical disposition remains conserved as historical fact.
	after, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	stored, _ := after.ItemByScoped(created.Scoped())
	if stored.TerminalDisposition != "CLOSED_INSUFFICIENT" {
		t.Fatalf("the historical disposition was lost: %q", stored.TerminalDisposition)
	}
	if stored.Lifecycle == domain.LifecycleClosed {
		t.Fatal("the unfinished obligation was retired")
	}
}

// TestF4AcceptanceRejectsAStaleArtifact witnesses F4.
//
// The contract: an artifact whose bytes changed after its verification no
// longer establishes the acceptance. The acceptance boundary revalidates
// the supporting artifacts.
func TestF4AcceptanceRejectsAStaleArtifact(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	artifact := filepath.Join(dir, "a.txt")
	payload := []byte("evidence\n")
	if err := os.WriteFile(artifact, payload, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	created, err := svc.CreateItem("tx", CreateItemOptions{
		EpicID: "EPIC-A", ItemID: "ITEM-1", Title: "work",
		Criteria: []string{"criterion"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID: "EV-A", Kind: "structural", Title: "evidence",
		RelativePath: "a.txt", SHA256: digest, ItemID: created.Scoped(),
		Quality: domain.EvidenceQualityVerified, Class: domain.EvidenceClassStructural,
		Criterion: "criterion", ExpectedRev: created.Revision,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}

	// The artifact bytes change AFTER the verification.
	if err := os.WriteFile(artifact, []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	state, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := state.ItemByScoped(created.Scoped())
	_, err = svc.Accept("tx-accept", AcceptOptions{
		ScopedID: item.Scoped(), ExpectedRev: item.Revision, AcceptedBy: "acceptor",
	})
	if err == nil {
		t.Fatal("the stale artifact established the acceptance")
	}
	if !errors.Is(err, ErrAcceptanceRejected) {
		t.Fatalf("expected ErrAcceptanceRejected, got %v", err)
	}

	after, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	afterItem, _ := after.ItemByScoped(created.Scoped())
	if afterItem.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("the stale artifact was accepted")
	}
}

// TestF4AcceptanceAcceptsAFreshArtifact witnesses F4.
//
// The contract: the freshness boundary does not break a legitimate
// acceptance: an unchanged artifact still establishes the acceptance.
func TestF4AcceptanceAcceptsAFreshArtifact(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	artifact := filepath.Join(dir, "a.txt")
	payload := []byte("evidence\n")
	if err := os.WriteFile(artifact, payload, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	created, err := svc.CreateItem("tx", CreateItemOptions{
		EpicID: "EPIC-A", ItemID: "ITEM-1", Title: "work",
		Criteria: []string{"criterion"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID: "EV-A", Kind: "structural", Title: "evidence",
		RelativePath: "a.txt", SHA256: digest, ItemID: created.Scoped(),
		Quality: domain.EvidenceQualityVerified, Class: domain.EvidenceClassStructural,
		Criterion: "criterion", ExpectedRev: created.Revision,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	state, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := state.ItemByScoped(created.Scoped())
	accepted, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID: item.Scoped(), ExpectedRev: item.Revision, AcceptedBy: "acceptor",
	})
	if err != nil {
		t.Fatalf("the fresh artifact did not establish the acceptance: %v", err)
	}
	if accepted.Acceptance != domain.AcceptanceAccepted {
		t.Fatalf("the acceptance was not recorded: %q", accepted.Acceptance)
	}
}

// TestF3CheckpointAcceptsANonTerminalTransition witnesses F3.
//
// The contract: the guard does not break a legitimate progress
// checkpoint.
func TestF3CheckpointAcceptsANonTerminalTransition(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "pending work")
	inProgress := domain.LifecycleInProgress

	updated, err := svc.Checkpoint("tx-f3-ok", CheckpointOptions{
		ScopedID: created.Scoped(), ExpectedRev: created.Revision,
		Lifecycle: inProgress, Summary: "work started",
	})
	if err != nil {
		t.Fatalf("the progress checkpoint was rejected: %v", err)
	}
	if updated.Lifecycle != inProgress {
		t.Fatalf("the progress checkpoint was not persisted: %q", updated.Lifecycle)
	}
}
