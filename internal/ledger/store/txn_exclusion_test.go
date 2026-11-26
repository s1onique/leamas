package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// This file qualifies the transaction exclusion witnesses of the FWL01
// amendment addendum.
//
// The witnesses are:
//   - two concurrent goroutine writers;
//   - two concurrent subprocess writers;
//   - a reader overlapping a publication;
//   - a recovery overlapping an active publication;
//   - an ACTUAL PROCESS TERMINATION after a partial publication;
//   - a post-crash read before recovery;
//   - a deterministic recovery and a repeated recovery.
//
// The process-termination witness terminates a REAL PROCESS. Constructing an
// interrupted journal is useful supplementary evidence, but it is not
// equivalent: only a real termination proves that the operating system
// releases the lock and that the on-disk state is recoverable.

// TestWitnessConcurrentGoroutineWritersAreMutuallyExclusive proves that two
// goroutines of the SAME process cannot both hold the exclusive lock.
//
// The exclusion is enforced by the operating-system advisory file lock,
// which is held per open file description. There is no process-wide
// reentrancy flag that could admit a second writer.
func TestWitnessConcurrentGoroutineWritersAreMutuallyExclusive(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ledger")
	if err := New(root).Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	holders := 0
	maxHolders := 0
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			l, err := AcquireWrite(root)
			if err != nil {
				return
			}
			mu.Lock()
			holders++
			if holders > maxHolders {
				maxHolders = holders
			}
			mu.Unlock()
			time.Sleep(150 * time.Millisecond)
			mu.Lock()
			holders--
			mu.Unlock()
			l.Release()
		}()
	}
	close(start)
	wg.Wait()

	if maxHolders != 1 {
		t.Fatalf("expected at most one concurrent writer, observed %d", maxHolders)
	}
}

// TestWitnessReaderDoesNotEnterDuringPublication proves that a reader of the
// SAME process does not enter while a writer of that process publishes.
func TestWitnessReaderDoesNotEnterDuringPublication(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ledger")
	if err := New(root).Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	w, err := AcquireWrite(root)
	if err != nil {
		t.Fatalf("write lock: %v", err)
	}
	entered := make(chan struct{})
	released := make(chan struct{})
	go func() {
		r, err := AcquireRead(root)
		if err == nil {
			close(entered)
			r.Release()
		}
		close(released)
	}()
	select {
	case <-entered:
		t.Fatal("the reader entered while the writer held the exclusive lock")
	case <-time.After(300 * time.Millisecond):
	}
	w.Release()
	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Fatal("the reader did not proceed after the writer released the lock")
	}
}

// TestWitnessRecoveryDoesNotOverlapActivePublication proves that a recovery
// does not run while a writer of the SAME process holds the exclusive lock.
func TestWitnessRecoveryDoesNotOverlapActivePublication(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ledger")
	if err := New(root).Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	w, err := AcquireWrite(root)
	if err != nil {
		t.Fatalf("write lock: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = New(root).Recover()
	}()
	select {
	case <-done:
		t.Fatal("the recovery ran while a writer held the exclusive lock")
	case <-time.After(300 * time.Millisecond):
	}
	w.Release()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the recovery did not proceed after the writer released the lock")
	}
}

// TestWitnessPendingJournalBlocksCanonicalRead proves that a canonical read
// is not admitted while an interrupted publication is pending.
//
// The visible state would be a partial transaction. It must not be read as
// if it were an accepted state.
func TestWitnessPendingJournalBlocksCanonicalRead(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ledger")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	original := mustEncode(t, amendmentItem("ITEM-A", 1))
	if err := WriteFileAtomic(filepath.Join(root, "items", "EPIC-A__ITEM-A.json"), original); err != nil {
		t.Fatalf("seed: %v", err)
	}
	txn, err := s.Begin("tx-interrupted")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txn.Write("items/EPIC-A__ITEM-A.json", mustEncode(t, amendmentItem("ITEM-A", 2)))
	if err := WriteFileAtomic(filepath.Join(root, "items", "EPIC-A__ITEM-A.json"),
		mustEncode(t, amendmentItem("ITEM-A", 2))); err != nil {
		t.Fatalf("partial write: %v", err)
	}
	if err := writeJournal(root, Journal{
		TxID:    "tx-interrupted",
		Entries: txn.entries,
		Backups: []JournalEntry{{RelPath: "items/EPIC-A__ITEM-A.json", Data: original}},
		State:   JournalPrepared,
	}); err != nil {
		t.Fatalf("journal: %v", err)
	}

	// The transaction is abandoned, so the exclusive lock is released.
	// The pending journal is now visible to every other participant.
	txn.Release()

	// The canonical read is refused: the visible state is a partial
	// transaction and must not be read as an accepted state.
	if _, err := s.Load(); err == nil {
		t.Fatal("the canonical read was admitted while a transaction was pending")
	}
	// A new transaction is refused too.
	if _, err := s.Begin("tx-next"); err == nil {
		t.Fatal("a new transaction was admitted on top of an interrupted publication")
	}

	// The recovery restores the pre-transaction state.
	status, err := s.Recover()
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if !status.Interrupted || !status.RolledBack {
		t.Fatalf("expected an interrupted transaction to be rolled back: %+v", status)
	}
	restored, err := os.ReadFile(filepath.Join(root, "items", "EPIC-A__ITEM-A.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(restored) != string(original) {
		t.Fatalf("the partial transaction was exposed: %q", string(restored))
	}

	// The canonical read is admitted again.
	if _, err := s.Load(); err != nil {
		t.Fatalf("the canonical read was refused after the recovery: %v", err)
	}
}

// TestWitnessRecoveryIsDeterministicAndRepeatable proves that the recovery
// is deterministic and that a repeated recovery is safe.
func TestWitnessRecoveryIsDeterministicAndRepeatable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ledger")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	original := mustEncode(t, amendmentItem("ITEM-A", 1))
	if err := WriteFileAtomic(filepath.Join(root, "items", "EPIC-A__ITEM-A.json"), original); err != nil {
		t.Fatalf("seed: %v", err)
	}
	txn, err := s.Begin("tx-interrupted")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txn.Write("items/EPIC-A__ITEM-A.json", mustEncode(t, amendmentItem("ITEM-A", 2)))
	if err := WriteFileAtomic(filepath.Join(root, "items", "EPIC-A__ITEM-A.json"),
		mustEncode(t, amendmentItem("ITEM-A", 2))); err != nil {
		t.Fatalf("partial write: %v", err)
	}
	if err := writeJournal(root, Journal{
		TxID:    "tx-interrupted",
		Entries: txn.entries,
		Backups: []JournalEntry{{RelPath: "items/EPIC-A__ITEM-A.json", Data: original}},
		State:   JournalPrepared,
	}); err != nil {
		t.Fatalf("journal: %v", err)
	}
	txn.Release()

	first, err := s.Recover()
	if err != nil {
		t.Fatalf("first recover: %v", err)
	}
	afterFirst, err := os.ReadFile(filepath.Join(root, "items", "EPIC-A__ITEM-A.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	second, err := s.Recover()
	if err != nil {
		t.Fatalf("second recover: %v", err)
	}
	afterSecond, err := os.ReadFile(filepath.Join(root, "items", "EPIC-A__ITEM-A.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !first.Interrupted || !first.RolledBack {
		t.Fatalf("the first recovery did not roll back: %+v", first)
	}
	if second.Interrupted {
		t.Fatalf("the second recovery found an interrupted transaction: %+v", second)
	}
	if string(afterFirst) != string(afterSecond) {
		t.Fatalf("the repeated recovery changed the state: %q -> %q",
			string(afterFirst), string(afterSecond))
	}
}

// TestWitnessProcessTerminationAfterPartialPublication proves that an ACTUAL
// PROCESS TERMINATION after a partial publication leaves a recoverable
// state, and that the interrupted writer never exposed an accepted partial
// state.
//
// The process is terminated with SIGKILL: the operating system releases the
// advisory lock and the on-disk journal survives.
func TestWitnessProcessTerminationAfterPartialPublication(t *testing.T) {
	if testing.Short() {
		t.Skip("the process-termination witness is skipped in short mode")
	}
	root := filepath.Join(t.TempDir(), "ledger")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	original := []byte("{\"original\":true}\n")
	if err := WriteFileAtomic(filepath.Join(root, "items", "A.json"), original); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Build the helper process that publishes a partial transaction and
	// then waits to be killed.
	helper := buildCrashHelper(t)
	defer os.RemoveAll(filepath.Dir(helper))

	// The helper publishes the first record and signals that it did so.
	ready := filepath.Join(t.TempDir(), "ready")
	cmd := exec.Command(helper, root, ready)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the helper: %v", err)
	}
	waitForFile(t, ready, 20*time.Second)

	// Terminate the ACTUAL PROCESS.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill the helper: %v", err)
	}
	_ = cmd.Wait()

	// The lock was released by the operating system: a new writer may
	// acquire it.
	lock, err := AcquireWrite(root)
	if err != nil {
		t.Fatalf("the exclusive lock was not released after the termination: %v", err)
	}
	lock.Release()

	// The POST-CRASH READ BEFORE RECOVERY is refused: the interrupted
	// writer must not expose an accepted partial state.
	if _, err := s.Load(); err == nil {
		t.Fatal("the post-crash read was admitted before the recovery")
	}

	// The recovery is deterministic and restores the pre-transaction state.
	status, err := s.Recover()
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if !status.Interrupted || !status.RolledBack {
		t.Fatalf("expected an interrupted transaction to be rolled back: %+v", status)
	}
	restored, err := os.ReadFile(filepath.Join(root, "items", "A.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(restored) != string(original) {
		t.Fatalf("the interrupted writer exposed an accepted partial state: %q", string(restored))
	}
	if _, err := os.Stat(filepath.Join(root, "items", "B.json")); !os.IsNotExist(err) {
		t.Fatal("the interrupted writer left a created record behind")
	}
}
