package store

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) Store {
	t.Helper()
	s := New(filepath.Join(t.TempDir(), "ledger"))
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	return s
}

// TestTransactionRecoveryRestoresPreTransactionState witnesses adversarial
// row 10: an interrupted multi-record operation cannot expose an accepted
// partial transaction.
func TestTransactionRecoveryRestoresPreTransactionState(t *testing.T) {
	s := newTestStore(t)

	original := []byte("{\"original\":true}\n")
	if err := WriteFileAtomic(filepath.Join(s.Root, "items", "A.json"), original); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Simulate an interrupted transaction: a journal exists that plans to
	// overwrite one file and create another.
	txn, err := s.Begin("tx-interrupted")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txn.Write("items/A.json", []byte("{\"partial\":true}\n"))
	txn.Write("items/B.json", []byte("{\"created\":true}\n"))

	// Publish the first entry only, then stop: this is the interruption.
	if err := WriteFileAtomic(filepath.Join(s.Root, "items", "A.json"),
		[]byte("{\"partial\":true}\n")); err != nil {
		t.Fatalf("partial write: %v", err)
	}
	if err := writeJournal(s.Root, Journal{
		TxID:    "tx-interrupted",
		Entries: txn.entries,
		Backups: []JournalEntry{{RelPath: "items/A.json", Data: original}},
		State:   JournalPrepared,
	}); err != nil {
		t.Fatalf("journal: %v", err)
	}

	// The recovery runs under the EXPLICIT ownership of the lock the open
	// transaction already holds: the lock is never re-entered implicitly.
	status, err := s.RecoverWithOwnership(txn.Ownership())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if !status.Interrupted || !status.RolledBack {
		t.Fatalf("expected interrupted transaction to be rolled back: %+v", status)
	}

	restored, err := os.ReadFile(filepath.Join(s.Root, "items", "A.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(restored) != string(original) {
		t.Fatalf("partial transaction was exposed: %q", string(restored))
	}
	if _, err := os.Stat(filepath.Join(s.Root, "items", "B.json")); !os.IsNotExist(err) {
		t.Fatal("partial transaction left a created record behind")
	}
}

// TestTransactionRecoveryIsIdempotent proves that replaying the recovery
// does not change the state again.
func TestTransactionRecoveryIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	original := []byte("{\"original\":true}\n")
	if err := WriteFileAtomic(filepath.Join(s.Root, "items", "A.json"), original); err != nil {
		t.Fatalf("seed: %v", err)
	}
	txn, err := s.Begin("tx-interrupted")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txn.Write("items/A.json", []byte("{\"partial\":true}\n"))
	if err := WriteFileAtomic(filepath.Join(s.Root, "items", "A.json"),
		[]byte("{\"partial\":true}\n")); err != nil {
		t.Fatalf("partial write: %v", err)
	}
	if err := writeJournal(s.Root, Journal{
		TxID:    "tx-interrupted",
		Entries: txn.entries,
		Backups: []JournalEntry{{RelPath: "items/A.json", Data: original}},
		State:   JournalPrepared,
	}); err != nil {
		t.Fatalf("journal: %v", err)
	}

	// The first recovery runs under the EXPLICIT ownership of the lock the
	// open transaction holds.
	if _, err := s.RecoverWithOwnership(txn.Ownership()); err != nil {
		t.Fatalf("first recover: %v", err)
	}
	afterFirst, err := os.ReadFile(filepath.Join(s.Root, "items", "A.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// The transaction is abandoned, so the lock is released. The repeated
	// recovery then acquires the exclusive lock itself and is idempotent.
	txn.Release()
	status, err := s.Recover()
	if err != nil {
		t.Fatalf("second recover: %v", err)
	}
	if status.Interrupted {
		t.Fatal("second recovery found an interrupted transaction")
	}
	afterSecond, err := os.ReadFile(filepath.Join(s.Root, "items", "A.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(afterFirst) != string(afterSecond) {
		t.Fatal("replaying the recovery changed the state")
	}
}

// TestCommittedTransactionLeavesNoJournal proves that an accepted
// transaction leaves no recovery marker behind.
func TestCommittedTransactionLeavesNoJournal(t *testing.T) {
	s := newTestStore(t)
	txn, err := s.Begin("tx-ok")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txn.Write("items/A.json", []byte("{\"a\":1}\n"))
	if err := txn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	pending, err := s.PendingJournal()
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending {
		t.Fatal("committed transaction left a journal behind")
	}
}

// TestReplayTransactionDoesNotDuplicateRecords witnesses adversarial row
// 11: replaying a transaction does not duplicate records or evidence.
func TestReplayTransactionDoesNotDuplicateRecords(t *testing.T) {
	s := newTestStore(t)
	payload := []byte("{\"a\":1}\n")

	for i := 0; i < 3; i++ {
		txn, err := s.Begin("tx-replay")
		if err != nil {
			t.Fatalf("begin %d: %v", i, err)
		}
		txn.Write("items/A.json", payload)
		if err := txn.Commit(); err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(s.Root, "items"))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	count := 0
	for _, e := range entries {
		if e.Name() == "A.json" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("replay produced %d copies of the record", count)
	}
}

// TestJournalPathSanitizationRejectsEscape proves that a journal path
// cannot escape the ledger root.
func TestJournalPathSanitizationRejectsEscape(t *testing.T) {
	if err := sanitizeRelPath("../escape.json"); err == nil {
		t.Fatal("expected path escape rejection")
	}
	if err := sanitizeRelPath("/absolute.json"); err == nil {
		t.Fatal("expected absolute path rejection")
	}
	if err := sanitizeRelPath("items/A.json"); err != nil {
		t.Fatalf("expected relative path to be accepted, got %v", err)
	}
}
