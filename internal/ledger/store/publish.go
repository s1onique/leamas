package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// PublishPartial publishes the journal and the FIRST staged entry, and then
// returns.
//
// It exists for the process-termination witness: it creates the exact
// on-disk state an interrupted publication leaves behind — a prepared
// journal and a partially published destination — so the witness can
// terminate a REAL process in that state.
//
// It is NOT a supported production operation. A caller that uses it is
// deliberately leaving the ledger in an interrupted state, which the
// recovery must repair.
func (t *Transaction) PublishPartial() error {
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
	if len(t.entries) == 0 {
		return nil
	}
	e := t.entries[0]
	abs := filepath.Join(root, e.RelPath)
	if e.Delete {
		if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", e.RelPath, err)
		}
		return nil
	}
	return WriteFileAtomic(abs, e.Data)
}
