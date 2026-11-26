package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// acceptItem accepts a work item through the explicit acceptance operation.
//
// The acceptance needs admissible evidence: a verified artifact bound to
// the item's criterion.
func acceptItem(t *testing.T, svc *Service, dir string, id domain.ScopedItemID) domain.WorkItem {
	t.Helper()
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, ok := l.ItemByScoped(id)
	if !ok {
		t.Fatalf("work item %s not found", id.String())
	}
	artifact := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(artifact, []byte("acceptance evidence\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	criterion := ""
	if len(item.AcceptanceCriteria) == 1 {
		criterion = item.AcceptanceCriteria[0]
	}
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID:   domain.EvidenceID("EV-ACCEPT-" + item.ItemID),
		Kind:         "live_qualification",
		Title:        "acceptance evidence",
		RelativePath: "artifact.txt",
		SHA256:       sha256OfFile(t, artifact),
		ItemID:       item.Scoped(),
		Quality:      domain.EvidenceQualityVerified,
		Class:        domain.EvidenceClassLiveQualification,
		Criterion:    criterion,
		ExpectedRev:  item.Revision,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	l, err = svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ = l.ItemByScoped(id)
	accepted, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID:    item.Scoped(),
		ExpectedRev: item.Revision,
		AcceptedBy:  "test-acceptor",
	})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	return accepted
}

// TestServiceArchivePreservesEvidenceAndSuccessors witnesses adversarial
// row 7 at the service seam: archive preserves evidence and successor
// references.
//
// The FWL01 amendment qualifies this row: an unfinished work item is not
// archivable at all. The archival is therefore performed on an ACCEPTED
// work item, which carries no outstanding obligation.
func TestServiceArchivePreservesEvidenceAndSuccessors(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	a := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	b := createItem(t, svc, "EPIC-A", "ITEM-2", "Second")

	if _, err := svc.Link("tx-link", LinkOptions{
		From:        a.Scoped(),
		To:          b.Scoped(),
		Kind:        domain.DependencyRelates,
		ExpectedRev: a.Revision,
	}); err != nil {
		t.Fatalf("link: %v", err)
	}
	if _, err := svc.SetSuccessor("tx-succ", SetSuccessorOptions{
		ScopedID:    a.Scoped(),
		Successor:   b.Scoped(),
		ExpectedRev: currentRevision(t, svc, a.Scoped()),
	}); err != nil {
		t.Fatalf("set successor: %v", err)
	}
	acceptItem(t, svc, dir, a.Scoped())

	current, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := current.ItemByScoped(a.Scoped())
	if len(item.Successors) != 1 {
		t.Fatalf("expected one successor, got %+v", item.Successors)
	}
	if len(item.Evidence) != 1 {
		t.Fatalf("expected one evidence reference, got %+v", item.Evidence)
	}

	archived, err := svc.Archive("tx-archive", ArchiveOptions{
		ScopedID:         a.Scoped(),
		ExpectedRev:      item.Revision,
		ReopenConditions: []domain.ReopenCondition{{Kind: "evidence_invalidated"}},
	})
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if !archived.Archived {
		t.Fatal("expected archived record")
	}
	if len(archived.Evidence) != 1 || archived.Evidence[0].EvidenceID != "EV-ACCEPT-ITEM-1" {
		t.Fatalf("archive lost the evidence reference: %+v", archived.Evidence)
	}
	if len(archived.Dependencies) != 1 || archived.Dependencies[0].Ref != b.Scoped() {
		t.Fatalf("archive lost the dependency relationship: %+v", archived.Dependencies)
	}
	if len(archived.Successors) != 1 {
		t.Fatalf("archive lost the successor obligation: %+v", archived.Successors)
	}
	if len(archived.Reopen) != 1 {
		t.Fatalf("archive lost the reopen condition: %+v", archived.Reopen)
	}
	if archived.TerminalDisposition == "" {
		t.Fatal("archive lost the original terminal disposition")
	}
}

// TestServiceArchiveOfUnfinishedWorkRequiresExplicitTransfer witnesses the
// FWL01 amendment obligation-conservation rule: an unfinished work item
// cannot disappear from the live board through archival.
//
// UNFINISHED-WORK ARCHIVAL TRANSFERS ARE DISABLED in this release. A
// complete conservation of a transferred obligation cannot be established
// with a small, bounded implementation, so the transfer is REJECTED
// outright rather than partially honoured.
func TestServiceArchiveOfUnfinishedWorkRequiresExplicitTransfer(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "Unfinished")
	successor := createItem(t, svc, "EPIC-A", "ITEM-2", "Successor")

	// Without a transfer the archival is refused.
	if _, err := svc.Archive("tx-archive", ArchiveOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
	}); err == nil {
		t.Fatal("expected the archival of unfinished work to be refused")
	}

	// With a transfer the archival is ALSO refused: the transfer is
	// disabled, not silently honoured.
	if _, err := svc.Archive("tx-archive", ArchiveOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
		TransferTo:  successor.Scoped(),
	}); err == nil {
		t.Fatal("expected the unfinished-work archival transfer to be refused")
	} else if !errors.Is(err, ErrUnfinishedObligation) {
		t.Fatalf("expected ErrUnfinishedObligation, got %v", err)
	}

	// The obligation is still live and discoverable.
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	live, ok := l.ItemByScoped(created.Scoped())
	if !ok {
		t.Fatal("the unfinished work item disappeared")
	}
	if live.Archived {
		t.Fatal("the unfinished work item was archived despite the rejection")
	}
	if live.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("the unfinished work item was accepted by the rejected archival")
	}
}

// TestServiceRestorePreservesStableIdentity witnesses adversarial row 8
// at the service seam: restore preserves the stable identity.
func TestServiceRestorePreservesStableIdentity(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	accepted := acceptItem(t, svc, dir, created.Scoped())

	archived, err := svc.Archive("tx-archive", ArchiveOptions{
		ScopedID:    accepted.Scoped(),
		ExpectedRev: accepted.Revision,
	})
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	restored, err := svc.Restore("tx-restore", RestoreOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: archived.Revision,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Scoped() != archived.Scoped() {
		t.Fatalf("restore changed the identity: %s -> %s", archived.Scoped(), restored.Scoped())
	}
	if restored.Archived {
		t.Fatal("restored record is still archived")
	}
}

// TestServiceRestoreRefusesIdentityReuse proves the restore rule is not
// vacuous: restoring a live record is refused.
func TestServiceRestoreRefusesIdentityReuse(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	if _, err := svc.Restore("tx-restore", RestoreOptions{
		ScopedID:    created.Scoped(),
		ExpectedRev: created.Revision,
	}); err == nil {
		t.Fatal("expected restore refusal for a live record")
	}
}

// TestServiceArchivedRecordsRemainDiscoverable proves that archival never
// hides a record from the canonical state.
func TestServiceArchivedRecordsRemainDiscoverable(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "First")
	accepted := acceptItem(t, svc, dir, created.Scoped())
	if _, err := svc.Archive("tx-archive", ArchiveOptions{
		ScopedID:    accepted.Scoped(),
		ExpectedRev: accepted.Revision,
	}); err != nil {
		t.Fatalf("archive: %v", err)
	}
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := l.ItemByScoped(created.Scoped()); !ok {
		t.Fatal("archived record is not discoverable")
	}
}

// boolPtr returns a pointer to the given bool.
func boolPtr(v bool) *bool { return &v }
