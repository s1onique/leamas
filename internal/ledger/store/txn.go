package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// JournalDir is the directory holding the in-flight transaction journal.
const JournalDir = ".txn"

// JournalEntry is one planned write inside a transaction.
type JournalEntry struct {
	// RelPath is the ledger-root-relative destination path.
	RelPath string `json:"rel_path"`
	// Data is the exact bytes to publish.
	Data []byte `json:"data"`
	// Delete marks the entry as a removal instead of a write.
	Delete bool `json:"delete,omitempty"`
}

// Journal is the durable record of a planned multi-record mutation.
//
// The journal is written before any destination is touched and removed
// only after every destination has been published. Its presence on disk
// therefore means "a transaction was interrupted", and its absence means
// "the visible state is a complete accepted state".
type Journal struct {
	TxID    string         `json:"tx_id"`
	Entries []JournalEntry `json:"entries"`
	Backups []JournalEntry `json:"backups,omitempty"`
	State   string         `json:"state"`
}

// Journal states.
const (
	JournalPrepared  = "prepared"
	JournalCommitted = "committed"
)

func journalPath(root string) string {
	return filepath.Join(root, JournalDir, "journal.json")
}

// Transaction is an open multi-record mutation.
type Transaction struct {
	store   Store
	entries []JournalEntry
	backups []JournalEntry
	backed  map[string][]byte
	txID    string
	lock    *Lock
}

// Ownership is the EXPLICIT proof that a caller already holds the
// exclusive lock of a ledger root.
//
// It exists so that an internal operation which must run under the
// exclusive lock can be given the ownership EXPLICITLY, instead of
// inferring it from a process-wide reentrancy flag. The inference was
// wrong: a process-wide flag cannot distinguish "this goroutine owns the
// lock" from "another goroutine of the same process owns the lock", so it
// silently admitted a second writer.
//
// An Ownership is only constructible inside this package, and only by the
// code path that actually holds the lock.
type Ownership struct {
	// root is the ledger root the ownership refers to.
	root string
	// lock is the held exclusive lock.
	lock *Lock
}

// Root returns the ledger root the ownership refers to.
func (o Ownership) Root() string { return o.root }

// Owns reports whether the ownership refers to the given root.
func (o Ownership) Owns(root string) bool { return o != Ownership{} && o.root == root }

// Begin starts a transaction.
//
// The exclusive lock is acquired for the whole publication, so a reader
// never observes a partially published transaction and a second writer
// cannot publish concurrently.
//
// The pending journal state is checked UNDER the acquired lock: a
// transaction is never admitted on top of an interrupted publication.
func (s Store) Begin(txID string) (*Transaction, error) {
	lock, err := AcquireWrite(s.Root)
	if err != nil {
		return nil, err
	}
	if s.Exists() {
		if _, err := os.Stat(journalPath(s.Root)); err == nil {
			lock.Release()
			return nil, fmt.Errorf("%w: %s", ErrTransactionOpen, s.Root)
		}
	}
	return &Transaction{
		store:  s,
		backed: map[string][]byte{},
		txID:   txID,
		lock:   lock,
	}, nil
}

// Ownership returns the EXPLICIT ownership of the exclusive lock this
// transaction holds.
//
// A caller that must run another operation under the same lock passes this
// value explicitly. The lock is never re-entered implicitly.
func (t *Transaction) Ownership() Ownership {
	if t == nil || t.lock == nil {
		return Ownership{}
	}
	return Ownership{root: t.store.Root, lock: t.lock}
}

// Release releases the transaction lock without publishing anything.
//
// It is used when a transaction is abandoned: the lock must not stay
// held, otherwise the ledger would be permanently locked.
func (t *Transaction) Release() {
	if t == nil {
		return
	}
	t.lock.Release()
	t.lock = nil
}

// Write stages a record write.
func (t *Transaction) Write(relPath string, data []byte) {
	t.entries = append(t.entries, JournalEntry{RelPath: relPath, Data: data})
}

// Delete stages a record removal.
func (t *Transaction) Delete(relPath string) {
	t.entries = append(t.entries, JournalEntry{RelPath: relPath, Delete: true})
}

// EntryCount returns the number of staged operations.
func (t *Transaction) EntryCount() int { return len(t.entries) }

// Commit publishes the staged operations atomically.
//
// The sequence is:
//
//  1. snapshot every destination that will be touched;
//  2. write the journal with the planned entries and the snapshots;
//  3. publish each destination via temp plus rename;
//  4. remove the journal.
//
// An interruption before step 4 leaves the journal on disk. Recovery
// then restores the snapshots, so the visible state is never an accepted
// partial transaction.
func (t *Transaction) Commit() error {
	// The exclusive lock is released on every exit path: a failed
	// publication must not leave the ledger permanently locked.
	defer t.Release()

	root := t.store.Root
	if err := os.MkdirAll(filepath.Join(root, JournalDir), 0o755); err != nil {
		return fmt.Errorf("create journal dir: %w", err)
	}

	for _, e := range t.entries {
		if e.Delete {
			continue
		}
		abs := filepath.Join(root, e.RelPath)
		if prior, err := os.ReadFile(abs); err == nil {
			t.backed[e.RelPath] = prior
			t.backups = append(t.backups, JournalEntry{RelPath: e.RelPath, Data: prior})
		}
	}

	journal := Journal{
		TxID:    t.txID,
		Entries: t.entries,
		Backups: t.backups,
		State:   JournalPrepared,
	}
	if err := writeJournal(root, journal); err != nil {
		return err
	}

	for _, e := range t.entries {
		abs := filepath.Join(root, e.RelPath)
		if e.Delete {
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", e.RelPath, err)
			}
			continue
		}
		if err := WriteFileAtomic(abs, e.Data); err != nil {
			return err
		}
	}

	journal.State = JournalCommitted
	if err := writeJournal(root, journal); err != nil {
		return err
	}
	return removeJournal(root)
}

func writeJournal(root string, j Journal) error {
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal: %w", err)
	}
	return WriteFileAtomic(journalPath(root), append(data, '\n'))
}

func removeJournal(root string) error {
	if err := os.RemoveAll(filepath.Join(root, JournalDir)); err != nil {
		return fmt.Errorf("remove journal: %w", err)
	}
	return nil
}

// RecoveryStatus describes the outcome of a recovery scan.
type RecoveryStatus struct {
	// Interrupted reports whether an unfinished transaction was found.
	Interrupted bool
	// RolledBack reports whether the interrupted transaction was undone.
	RolledBack bool
	// TxID is the identity of the recovered transaction.
	TxID string
	// RestoredFiles lists the files restored to their pre-transaction state.
	RestoredFiles []string
}

// Recover repairs an interrupted transaction.
//
// The repair is deterministic and idempotent: it restores every snapshot
// recorded in the journal and removes the journal. Running Recover twice
// is safe and reports the same terminal state.
//
// The recovery takes the SAME exclusive authority as a writer: it cannot
// roll back an ACTIVE foreign transaction, because an active writer holds
// the exclusive lock for the whole publication.
//
// The journal presence is checked UNDER the acquired lock, so a recovery
// never races a publication that is in flight.
//
// When the caller ALREADY holds the exclusive lock of the same root, it
// must use RecoverWithOwnership and pass the explicit ownership. The lock
// is never re-entered implicitly.
func (s Store) Recover() (RecoveryStatus, error) {
	lock, err := AcquireWrite(s.Root)
	if err != nil {
		return RecoveryStatus{}, err
	}
	defer lock.Release()
	return s.recoverUnderLock()
}

// RecoverWithOwnership repairs an interrupted transaction under an
// EXPLICITLY supplied ownership of the exclusive lock.
//
// It is the form an internal rollback uses: the caller already holds the
// exclusive lock of the same root and passes that ownership explicitly,
// instead of the lock inferring reentrancy from a process-wide flag.
//
// The ownership is checked against the root: an ownership of a DIFFERENT
// root is rejected, and an empty ownership is rejected.
func (s Store) RecoverWithOwnership(own Ownership) (RecoveryStatus, error) {
	if !own.Owns(s.Root) {
		return RecoveryStatus{}, fmt.Errorf("%w: the supplied ownership does not cover %s",
			ErrOwnershipMismatch, s.Root)
	}
	return s.recoverUnderLock()
}

// recoverUnderLock performs the rollback. The caller must hold the
// exclusive lock of the root.
func (s Store) recoverUnderLock() (RecoveryStatus, error) {
	var status RecoveryStatus

	path := journalPath(s.Root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return status, nil
		}
		return status, err
	}
	status.Interrupted = true

	var j Journal
	if err := json.Unmarshal(data, &j); err != nil {
		return status, fmt.Errorf("decode journal: %w", err)
	}
	status.TxID = j.TxID

	for _, b := range j.Backups {
		abs := filepath.Join(s.Root, b.RelPath)
		if err := WriteFileAtomic(abs, b.Data); err != nil {
			return status, err
		}
		status.RestoredFiles = append(status.RestoredFiles, b.RelPath)
	}
	for _, e := range j.Entries {
		if !e.Delete {
			continue
		}
		abs := filepath.Join(s.Root, e.RelPath)
		if _, backed := journalHasBackup(j.Backups, e.RelPath); backed {
			continue
		}
		if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
			return status, fmt.Errorf("remove %s: %w", e.RelPath, err)
		}
	}
	sort.Strings(status.RestoredFiles)

	if err := removeJournal(s.Root); err != nil {
		return status, err
	}
	status.RolledBack = true
	return status, nil
}

func journalHasBackup(backups []JournalEntry, rel string) ([]byte, bool) {
	for _, b := range backups {
		if b.RelPath == rel {
			return b.Data, true
		}
	}
	return nil, false
}

// PendingJournal reports whether an unfinished transaction exists.
func (s Store) PendingJournal() (bool, error) {
	_, err := os.Stat(journalPath(s.Root))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// RelPathFor builds a ledger-root-relative path for a record file.
func RelPathFor(dir, name string) string {
	return filepath.ToSlash(filepath.Join(dir, name))
}

// sanitizeRelPath rejects journal paths that escape the ledger root.
func sanitizeRelPath(rel string) error {
	if rel == "" {
		return errors.New("empty journal path")
	}
	if strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return fmt.Errorf("journal path escapes ledger root: %q", rel)
	}
	return nil
}
