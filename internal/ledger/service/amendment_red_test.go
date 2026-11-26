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

// This file records the RED reproduction of the counterexamples that the
// FWL01 amendment review reproduced independently. Each test drives the real
// production seam (service + store + domain), never a private
// implementation detail.
//
// The tests are written as executable contracts: they assert the REQUIRED
// behaviour, not the behaviour that happens to exist today.

// TestAmendmentEvidenceFreeAcceptanceRejected reproduces counterexample 1:
// live acceptance with no evidence records.
//
// The contract: a work item that declares acceptance criteria cannot be
// accepted when no evidence record is linked to it. A caller-supplied
// quality label must not establish verification.
func TestAmendmentEvidenceFreeAcceptanceRejected(t *testing.T) {
	svc := newTestService(t)
	created, err := svc.CreateItem("tx-create", CreateItemOptions{
		EpicID:   domain.EpicID("EPIC-A"),
		ItemID:   domain.ItemID("ITEM-1"),
		Title:    "Needs evidence",
		Criteria: []string{"the criterion must hold"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// The caller claims a strong quality without any evidence record.
	_, err = svc.Accept("tx-accept", AcceptOptions{
		ScopedID:        created.Scoped(),
		ExpectedRev:     created.Revision,
		EvidenceQuality: domain.EvidenceQualityVerified,
		AcceptedBy:      "test-acceptor",
	})
	if err == nil {
		t.Fatal("evidence-free acceptance was accepted: want rejection")
	}
	if !errors.Is(err, ErrAcceptanceRejected) {
		t.Fatalf("expected ErrAcceptanceRejected, got %v", err)
	}
}

// TestAmendmentStructuralEvidenceCannotSatisfyLiveGate reproduces
// counterexample 1b: a verified structural or synthetic artifact must not
// satisfy a live qualification criterion.
func TestAmendmentStructuralEvidenceCannotSatisfyLiveGate(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "ledger")
	svc := New(store.New(root)).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	artifact := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(artifact, []byte("structural output\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	created, err := svc.CreateItem("tx-create", CreateItemOptions{
		EpicID:       domain.EpicID("EPIC-A"),
		ItemID:       domain.ItemID("ITEM-1"),
		Title:        "Needs live qualification",
		Criteria:     []string{"live qualification must pass"},
		RequiresLive: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Structural evidence: a genuinely verified artifact, but of the
	// structural class. Verification alone must not satisfy a live
	// qualification.
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID:   domain.EvidenceID("EV-STRUCTURAL"),
		Kind:         "structural",
		Title:        "structural artifact",
		RelativePath: "artifact.txt",
		SHA256:       sha256OfFile(t, artifact),
		ItemID:       created.Scoped(),
		Quality:      domain.EvidenceQualityVerified,
		Class:        domain.EvidenceClassStructural,
		Criterion:    "live qualification must pass",
		ExpectedRev:  created.Revision,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}

	cur, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := cur.ItemByScoped(created.Scoped())
	if item.Quality != domain.EvidenceQualityVerified {
		t.Fatalf("precondition: expected promoted verified quality, got %q", item.Quality)
	}

	_, err = svc.Accept("tx-accept", AcceptOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: item.Revision,
		AcceptedBy:  "test-acceptor",
	})
	if err == nil {
		t.Fatal("structural evidence satisfied a live qualification: want rejection")
	}
	if !errors.Is(err, ErrAcceptanceRejected) {
		t.Fatalf("expected ErrAcceptanceRejected, got %v", err)
	}
}

// TestAmendmentAcceptanceRequiresLinkedEvidencePositive is the positive
// control for the evidence-authority contract: a properly bound, verified
// artifact with the required evidence classification supports an explicitly
// authorized scoped acceptance.
func TestAmendmentAcceptanceRequiresLinkedEvidencePositive(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "ledger")
	svc := New(store.New(root)).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	artifact := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(artifact, []byte("live qualification output\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	created, err := svc.CreateItem("tx-create", CreateItemOptions{
		EpicID:       domain.EpicID("EPIC-A"),
		ItemID:       domain.ItemID("ITEM-1"),
		Title:        "Live qualification",
		Criteria:     []string{"live qualification must pass"},
		RequiresLive: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	ev, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID:   domain.EvidenceID("EV-LIVE"),
		Kind:         "live_qualification",
		Title:        "live qualification output",
		RelativePath: "artifact.txt",
		SHA256:       sha256OfFile(t, artifact),
		ItemID:       created.Scoped(),
		Quality:      domain.EvidenceQualityVerified,
		Class:        domain.EvidenceClassLiveQualification,
		Criterion:    "live qualification must pass",
		ExpectedRev:  created.Revision,
	})
	if err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	if ev.Quality != domain.EvidenceQualityVerified {
		t.Fatalf("expected verified evidence, got %q", ev.Quality)
	}
	if !ev.Verified {
		t.Fatal("expected a verified artifact identity")
	}

	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())

	accepted, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID:    item.Scoped(),
		ExpectedRev: item.Revision,
		AcceptedBy:  "test-acceptor",
	})
	if err != nil {
		t.Fatalf("positive control: live-qualified acceptance was rejected: %v", err)
	}
	if accepted.Acceptance != domain.AcceptanceAccepted {
		t.Fatalf("expected accepted, got %q", accepted.Acceptance)
	}
}

// TestAmendmentArchiveOfUnfinishedObligationRejected reproduces
// counterexample 2: archival of an unfinished obligation.
//
// The contract: an unfinished work item cannot be archived unless an
// explicitly authorized and conserved obligation transfer exists.
func TestAmendmentArchiveOfUnfinishedObligationRejected(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "Unfinished work")

	_, err := svc.Archive("tx-archive", ArchiveOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
	})
	if err == nil {
		t.Fatal("unfinished work item was archived: want rejection")
	}
	if !errors.Is(err, ErrUnfinishedObligation) {
		t.Fatalf("expected ErrUnfinishedObligation, got %v", err)
	}

	// The obligation must still be live and discoverable.
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, ok := l.ItemByScoped(created.Scoped())
	if !ok {
		t.Fatal("work item disappeared from the ledger")
	}
	if item.Archived {
		t.Fatal("work item was archived despite the rejected archival")
	}
}

// TestAmendmentOwnershipReplacementRequiresExplicitTransfer reproduces the
// ownership half of W4: silent ownership replacement.
//
// The contract: an update must not silently replace an existing owner. An
// explicit bounded release or transfer operation is required.
func TestAmendmentOwnershipReplacementRequiresExplicitTransfer(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "Owned work")
	if _, err := svc.Claim("tx-claim", ClaimOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
		Owner:       "owner-a",
	}); err != nil {
		t.Fatalf("claim: %v", err)
	}

	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	owned, _ := l.ItemByScoped(created.Scoped())

	newOwner := "owner-b"
	_, err = svc.UpdateItem("tx-update", UpdateItemOptions{
		ScopedID:    owned.Scoped(),
		ExpectedRev: owned.Revision,
		Owner:       &newOwner,
	})
	if err == nil {
		t.Fatal("silent ownership replacement was accepted: want rejection")
	}
	if !errors.Is(err, ErrOwnershipTransferRequired) {
		t.Fatalf("expected ErrOwnershipTransferRequired, got %v", err)
	}
}

// TestAmendmentSemanticNoOpUpdateDoesNotBumpRevision reproduces the
// W4 requirement that a semantic no-op update must not increment the
// record revision or rewrite canonical state.
func TestAmendmentSemanticNoOpUpdateDoesNotBumpRevision(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "Stable work")

	sameTitle := "Stable work"
	updated, err := svc.UpdateItem("tx-update", UpdateItemOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
		Title:       &sameTitle,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Revision != created.Revision {
		t.Fatalf("semantic no-op update bumped revision: %d -> %d",
			created.Revision, updated.Revision)
	}
}

// sha256OfFile returns the lowercase hexadecimal SHA-256 of a file.
func sha256OfFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestAmendmentActMayAddressMultipleWorkItems witnesses the W2
// multi-item ACT requirement.
//
// The contract: an ACT may address several work items. Each keeps its own
// acceptance state, and the terminal ACT verdict never implies acceptance
// of the associated work items.
func TestAmendmentActMayAddressMultipleWorkItems(t *testing.T) {
	svc := newTestService(t)
	first := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	second := createItem(t, svc, "EPIC-A", "ITEM-2", "Second")

	closed, err := svc.CloseAct("tx-close", CloseActOptions{
		ActID:   "ACT-MULTI",
		ItemID:  first.Scoped(),
		ItemIDs: []domain.ScopedItemID{second.Scoped()},
		Verdict: domain.ActVerdictClosed,
	})
	if err != nil {
		t.Fatalf("close act: %v", err)
	}
	if len(closed.ItemIDs) != 2 {
		t.Fatalf("expected the ACT to address 2 work items, got %+v", closed.ItemIDs)
	}

	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, id := range []domain.ScopedItemID{first.Scoped(), second.Scoped()} {
		item, ok := l.ItemByScoped(id)
		if !ok {
			t.Fatalf("work item %s disappeared", id.String())
		}
		if item.Acceptance == domain.AcceptanceAccepted {
			t.Fatalf("terminal ACT implied acceptance of %s", id.String())
		}
		if item.Lifecycle == domain.LifecycleClosed {
			t.Fatalf("terminal ACT closed the lifecycle of %s", id.String())
		}
	}
}

// TestAmendmentActRejectsUnknownWorkItem proves the multi-item binding is
// validated: an ACT may not name a work item that does not exist.
func TestAmendmentActRejectsUnknownWorkItem(t *testing.T) {
	svc := newTestService(t)
	createItem(t, svc, "EPIC-A", "ITEM-1", "First")

	_, err := svc.CloseAct("tx-close", CloseActOptions{
		ActID:   "ACT-BAD",
		ItemID:  "EPIC-A/ITEM-1",
		ItemIDs: []domain.ScopedItemID{"EPIC-A/ITEM-MISSING"},
		Verdict: domain.ActVerdictClosed,
	})
	if err == nil {
		t.Fatal("an ACT naming an unknown work item was accepted")
	}
}

// TestAmendmentExplicitOwnershipTransferIsBounded witnesses the W4
// ownership requirement: the transfer is explicit and bounded by the
// expected current owner.
func TestAmendmentExplicitOwnershipTransferIsBounded(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "Owned work")
	if _, err := svc.Claim("tx-claim", ClaimOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
		Owner:       "owner-a",
	}); err != nil {
		t.Fatalf("claim: %v", err)
	}

	// A transfer that does not name the current owner is refused.
	if _, err := svc.TransferOwnership("tx-transfer", TransferOwnershipOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision + 1,
		Owner:       "owner-b",
		FromOwner:   "owner-wrong",
	}); err == nil {
		t.Fatal("a transfer with the wrong expected owner was accepted")
	}

	// The correct transfer succeeds.
	updated, err := svc.TransferOwnership("tx-transfer", TransferOwnershipOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision + 1,
		Owner:       "owner-b",
		FromOwner:   "owner-a",
	})
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if updated.Owner != "owner-b" {
		t.Fatalf("expected owner-b, got %q", updated.Owner)
	}
}
