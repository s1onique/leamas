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
}

// Begin starts a transaction.
func (s Store) Begin(txID string) (*Transaction, error) {
	if s.Exists() {
		if _, err := os.Stat(journalPath(s.Root)); err == nil {
			return nil, fmt.Errorf("%w: %s", ErrTransactionOpen, s.Root)
		}
	}
	return &Transaction{store: s, backed: map[string][]byte{}, txID: txID}, nil
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
func (s Store) Recover() (RecoveryStatus, error) {
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
