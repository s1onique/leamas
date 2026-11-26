package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// newTestService returns a service backed by a temporary ledger root.
func newTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger")))
	if err := svc.Init(); err != nil {
		t.Fatalf("init ledger: %v", err)
	}
	return svc
}

// createItem is a small helper that creates a work item.
func createItem(t *testing.T, svc *Service, epic, item, title string) domain.WorkItem {
	t.Helper()
	created, err := svc.CreateItem("tx-"+item, CreateItemOptions{
		EpicID: domain.EpicID(epic),
		ItemID: domain.ItemID(item),
		Title:  title,
	})
	if err != nil {
		t.Fatalf("create %s/%s: %v", epic, item, err)
	}
	return created
}

// currentRevision returns the stored revision of a work item.
//
// The externally initiated mutation contract requires a POSITIVE expected
// revision: the caller must state which version of the record it observed.
// The helper reads that version from the canonical state.
func currentRevision(t *testing.T, svc *Service, id domain.ScopedItemID) int {
	t.Helper()
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	it, ok := l.ItemByScoped(id)
	if !ok {
		t.Fatalf("work item %s not found", id.String())
	}
	return it.Revision
}

// TestServiceCreateItemRejectsDuplicateIdentity witnesses adversarial
// row 1 at the service seam: a duplicate work-item identity is rejected.
func TestServiceCreateItemRejectsDuplicateIdentity(t *testing.T) {
	svc := newTestService(t)
	createItem(t, svc, "EPIC-A", "ITEM-1", "First")

	_, err := svc.CreateItem("tx-dup", CreateItemOptions{
		EpicID: domain.EpicID("EPIC-A"),
		ItemID: domain.ItemID("ITEM-1"),
		Title:  "Duplicate",
	})
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

// TestServiceUpdateRejectsStaleRevision witnesses adversarial row 9: a
// stale revision cannot overwrite accepted newer state.
func TestServiceUpdateRejectsStaleRevision(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")

	newTitle := "Second"
	if _, err := svc.UpdateItem("tx-a", UpdateItemOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
		Title:       &newTitle,
	}); err != nil {
		t.Fatalf("first update: %v", err)
	}

	staleTitle := "Stale"
	_, err := svc.UpdateItem("tx-b", UpdateItemOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
		Title:       &staleTitle,
	})
	if !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("expected ErrStaleRevision, got %v", err)
	}

	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	if item.Title != "Second" {
		t.Fatalf("stale write leaked: title is %q", item.Title)
	}
}

// TestServiceStaleRevisionDoesNotMutateState proves the rejected write
// left the canonical state byte-identical.
func TestServiceStaleRevisionDoesNotMutateState(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	before, err := os.ReadFile(svc.Store.Root + "/items/EPIC-A__ITEM-1.json")
	if err != nil {
		t.Fatalf("read item file: %v", err)
	}
	stale := "Stale"
	if _, err := svc.UpdateItem("tx-stale", UpdateItemOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision + 99,
		Title:       &stale,
	}); err == nil {
		t.Fatal("expected stale revision rejection")
	}
	after, err := os.ReadFile(svc.Store.Root + "/items/EPIC-A__ITEM-1.json")
	if err != nil {
		t.Fatalf("read item file: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("rejected mutation changed the canonical state")
	}
}

// TestServiceLinkRejectsDanglingDependency witnesses adversarial row 2 at
// the service seam: a dependency to an unknown item is rejected.
func TestServiceLinkRejectsDanglingDependency(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")

	_, err := svc.Link("tx-link", LinkOptions{
		From:        created.Scoped(),
		To:          "EPIC-A/ITEM-MISSING",
		Kind:        domain.DependencyBlocks,
		ExpectedRev: created.Revision,
	})
	if err == nil {
		t.Fatal("expected dangling dependency rejection")
	}
}

// TestServiceLinkRejectsBlockingCycle witnesses adversarial row 3 at the
// service seam: a blocking dependency cycle is rejected.
func TestServiceLinkRejectsBlockingCycle(t *testing.T) {
	svc := newTestService(t)
	a := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	b := createItem(t, svc, "EPIC-A", "ITEM-2", "Second")

	if _, err := svc.Link("tx-link-1", LinkOptions{
		From: a.Scoped(), To: b.Scoped(), Kind: domain.DependencyBlocks,
		ExpectedRev: a.Revision,
	}); err != nil {
		t.Fatalf("link a->b: %v", err)
	}
	_, err := svc.Link("tx-link-2", LinkOptions{
		From: b.Scoped(), To: a.Scoped(), Kind: domain.DependencyBlocks,
		ExpectedRev: b.Revision,
	})
	if err == nil {
		t.Fatal("expected blocking cycle rejection")
	}
}

// TestServiceInsufficientEvidenceCannotAccept witnesses adversarial row 4
// at the service seam: insufficient evidence cannot produce successful
// acceptance.
func TestServiceInsufficientEvidenceCannotAccept(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")

	if _, err := svc.UpdateItem("tx-crit", UpdateItemOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
		Criteria:    []string{"the qualification run passes"},
	}); err != nil {
		t.Fatalf("add criteria: %v", err)
	}

	// No evidence: acceptance must be rejected.
	if _, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision + 1,
		AcceptedBy:  "test-acceptor",
	}); err == nil {
		t.Fatal("expected acceptance rejection without evidence")
	}

	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	if item.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("insufficient evidence produced successful acceptance")
	}
}

// TestServiceStructuralPassCannotCloseLiveGate witnesses adversarial row
// 5 at the service seam: a structural pass cannot close a required live
// gate.
func TestServiceStructuralPassCannotCloseLiveGate(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")

	if _, err := svc.UpdateItem("tx-live", UpdateItemOptions{
		ScopedID:     created.Scoped(),
		ExpectedRev:  created.Revision,
		RequiresLive: boolPtr(true),
	}); err != nil {
		t.Fatalf("mark live gate: %v", err)
	}

	// A structural (referenced) evidence reference is not a live
	// qualification: acceptance must be rejected.
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID:  "EV-A",
		Kind:        "file",
		Title:       "Structural report",
		ItemID:      created.Scoped(),
		Quality:     domain.EvidenceQualityReferenced,
		ExpectedRev: created.Revision + 1,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}

	if _, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision + 2,
		AcceptedBy:  "test-acceptor",
	}); err == nil {
		t.Fatal("expected acceptance rejection for unsatisfied live gate")
	}
}

// TestServiceVerifiedEvidenceSatisfiesLiveGate proves the live gate rule
// is not vacuous: verified evidence satisfies it.
//
// The FWL01 amendment strengthens this row: the evidence must be an
// actually verified artifact of the live qualification class. A
// caller-supplied "verified" quality label no longer satisfies the gate.
func TestServiceVerifiedEvidenceSatisfiesLiveGate(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	artifact := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(artifact, []byte("live qualification output\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	if _, err := svc.UpdateItem("tx-live", UpdateItemOptions{
		ScopedID:     created.Scoped(),
		ExpectedRev:  created.Revision,
		RequiresLive: boolPtr(true),
	}); err != nil {
		t.Fatalf("mark live gate: %v", err)
	}
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID:   "EV-A",
		Kind:         "live_qualification",
		Title:        "Live qualification",
		RelativePath: "live.txt",
		SHA256:       sha256OfFile(t, artifact),
		ItemID:       created.Scoped(),
		Quality:      domain.EvidenceQualityVerified,
		Class:        domain.EvidenceClassLiveQualification,
		ExpectedRev:  created.Revision + 1,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	accepted, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision + 2,
		AcceptedBy:  "test-acceptor",
	})
	if err != nil {
		t.Fatalf("expected acceptance with verified evidence, got %v", err)
	}
	if accepted.Acceptance != domain.AcceptanceAccepted {
		t.Fatalf("expected accepted, got %q", accepted.Acceptance)
	}
}

// TestServiceTerminalActRetainsUnfinishedObligation witnesses adversarial
// row 6 at the service seam: a terminal ACT with an unfinished obligation
// retains that obligation.
func TestServiceTerminalActRetainsUnfinishedObligation(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")

	if _, err := svc.CloseAct("tx-close", CloseActOptions{
		ActID:   "ACT-A",
		ItemID:  created.Scoped(),
		Verdict: domain.ActVerdictClosed,
	}); err != nil {
		t.Fatalf("close act: %v", err)
	}

	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	if item.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("terminal ACT closed the engineering obligation")
	}
	if item.Lifecycle == domain.LifecycleClosed {
		t.Fatal("terminal ACT closed the work item lifecycle")
	}
}
