package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file records the RED reproduction of the transaction-qualification
// counterexamples against the real storage seams.

// amendmentItem is a small helper that builds a canonical work item.
func amendmentItem(id string, rev int) domain.WorkItem {
	return domain.WorkItem{
		SchemaVersion: domain.LedgerSchemaVersion,
		EpicID:        domain.EpicID("EPIC-A"),
		ItemID:        domain.ItemID(id),
		Title:         "item " + id,
		Kind:          domain.ItemKindTask,
		Lifecycle:     domain.LifecycleOpen,
		Acceptance:    domain.AcceptancePending,
		Revision:      rev,
	}
}

// snapshotTree renders the ledger root as a stable byte-level fingerprint
// of every regular file it contains.
//
// The lock file is excluded: it is a coordination artifact, not canonical
// state.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == LockFileName {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lines = append(lines, filepath.ToSlash(rel)+"="+string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// TestAmendmentConcurrentWritersCannotBothPublish reproduces witness 1:
// two writers using the same expected revision cannot both publish
// conflicting updates.
func TestAmendmentConcurrentWritersCannotBothPublish(t *testing.T) {
	s := newTestStore(t)
	base := amendmentItem("ITEM-1", 1)
	if err := WriteFileAtomic(filepath.Join(s.Root, RelPathFor("items", "EPIC-A__ITEM-1.json")), mustEncode(t, base)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Writer A publishes revision 2.
	txnA, err := s.BeginMutation("tx-a")
	if err != nil {
		t.Fatalf("begin a: %v", err)
	}
	itemA := base
	itemA.Title = "writer a"
	itemA.Revision = 2
	if err := txnA.PutItem(itemA); err != nil {
		t.Fatalf("put a: %v", err)
	}
	if err := txnA.Commit(); err != nil {
		t.Fatalf("commit a: %v", err)
	}

	// Writer B still believes revision 1 is current. It must be rejected.
	txnB, err := s.BeginMutation("tx-b")
	if err != nil {
		t.Fatalf("begin b: %v", err)
	}
	itemB := base
	itemB.Title = "writer b"
	itemB.Revision = 2
	if err := txnB.PutItem(itemB); err != nil {
		t.Fatalf("put b: %v", err)
	}
	err = txnB.Commit()
	if err == nil {
		t.Fatal("stale writer published a conflicting update: want rejection")
	}

	l, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped("EPIC-A/ITEM-1")
	if item.Title != "writer a" {
		t.Fatalf("conflicting update was published: title is %q", item.Title)
	}
}

// TestAmendmentFailedValidationLeavesStateByteIdentical reproduces
// witness 6: failed validation leaves canonical state byte-identical.
func TestAmendmentFailedValidationLeavesStateByteIdentical(t *testing.T) {
	s := newTestStore(t)
	base := amendmentItem("ITEM-1", 1)
	if err := WriteFileAtomic(filepath.Join(s.Root, RelPathFor("items", "EPIC-A__ITEM-1.json")), mustEncode(t, base)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	before := snapshotTree(t, s.Root)

	// An invalid item: empty title violates the canonical rules.
	txn, err := s.BeginMutation("tx-bad")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	bad := amendmentItem("ITEM-2", 1)
	bad.Title = ""
	if err := txn.PutItem(bad); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := txn.Commit(); err == nil {
		t.Fatal("invalid candidate was published: want rejection")
	}

	after := snapshotTree(t, s.Root)
	if before != after {
		t.Fatalf("canonical state changed after a failed validation:\nbefore=%s\nafter=%s", before, after)
	}
}

// TestAmendmentInterruptedPublicationIsRecoverable reproduces witness 3:
// interrupted publication is recoverable without obligation loss.
//
// The recovery runs under the EXPLICIT ownership of the lock the open
// transaction holds: the lock is never re-entered implicitly.
func TestAmendmentInterruptedPublicationIsRecoverable(t *testing.T) {
	s := newTestStore(t)
	base := amendmentItem("ITEM-1", 1)
	original := mustEncode(t, base)
	if err := WriteFileAtomic(filepath.Join(s.Root, RelPathFor("items", "EPIC-A__ITEM-1.json")), original); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Simulate an interruption: a journal plans two destinations, but only
	// the first one is published before the process stops.
	txn, err := s.Begin("tx-interrupt")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	interrupted := base
	interrupted.Title = "interrupted"
	interrupted.Revision = 2
	txn.Write(RelPathFor("items", "EPIC-A__ITEM-1.json"), mustEncode(t, interrupted))
	txn.Write(RelPathFor("items", "EPIC-A__ITEM-2.json"), mustEncode(t, amendmentItem("ITEM-2", 1)))

	if err := WriteFileAtomic(filepath.Join(s.Root, RelPathFor("items", "EPIC-A__ITEM-1.json")), mustEncode(t, interrupted)); err != nil {
		t.Fatalf("partial write: %v", err)
	}
	if err := writeJournal(s.Root, Journal{
		TxID:    "tx-interrupt",
		Entries: txn.entries,
		Backups: []JournalEntry{{RelPath: RelPathFor("items", "EPIC-A__ITEM-1.json"), Data: original}},
		State:   JournalPrepared,
	}); err != nil {
		t.Fatalf("journal: %v", err)
	}

	// The recovery runs under the EXPLICIT ownership of the lock the open
	// transaction already holds.
	status, err := s.RecoverWithOwnership(txn.Ownership())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if !status.Interrupted {
		t.Fatal("recovery did not report an interrupted transaction")
	}
	if !status.RolledBack {
		t.Fatal("recovery did not roll the interrupted transaction back")
	}

	// The journal is gone, so the canonical read is admitted again.
	txn.Release()
	l, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, ok := l.ItemByScoped("EPIC-A/ITEM-1")
	if !ok {
		t.Fatal("the interrupted transaction lost the obligation")
	}
	if item.Title != "item ITEM-1" {
		t.Fatalf("rollback did not restore the pre-transaction state: %q", item.Title)
	}
}

// TestAmendmentReplayOfCommittedOperationDoesNotDuplicate reproduces
// witness 5: replaying a committed operation cannot duplicate its effects.
func TestAmendmentReplayOfCommittedOperationDoesNotDuplicate(t *testing.T) {
	s := newTestStore(t)
	item := amendmentItem("ITEM-1", 1)
	if err := WriteFileAtomic(filepath.Join(s.Root, RelPathFor("items", "EPIC-A__ITEM-1.json")), mustEncode(t, item)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	publish := func() {
		txn, err := s.BeginMutation("tx-replay")
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		rec := item
		rec.Title = "replayed"
		rec.Revision = 2
		if err := txn.PutItem(rec); err != nil {
			t.Fatalf("put: %v", err)
		}
		if err := txn.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
	publish()

	before := snapshotTree(t, s.Root)
	publish()
	after := snapshotTree(t, s.Root)
	if before != after {
		t.Fatalf("replaying a committed operation changed canonical state:\nbefore=%s\nafter=%s", before, after)
	}
}

// TestAmendmentJournalPathStaysWithinStorage reproduces witness 7:
// transaction and journal paths remain within authorized storage.
func TestAmendmentJournalPathStaysWithinStorage(t *testing.T) {
	if err := sanitizeRelPath("../escape.json"); err == nil {
		t.Fatal("journal path escaped the ledger root: want rejection")
	}
	if err := sanitizeRelPath("/abs/escape.json"); err == nil {
		t.Fatal("absolute journal path was accepted: want rejection")
	}
	if err := sanitizeRelPath("items/ok.json"); err != nil {
		t.Fatalf("legitimate journal path was rejected: %v", err)
	}
}

// mustEncode encodes a record the same way the store does.
func mustEncode(t *testing.T, v any) []byte {
	t.Helper()
	data, err := encodeRecord(v)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return data
}
