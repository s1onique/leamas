// Package store implements the canonical on-disk representation of the
// Factory Work Ledger.
//
// Storage layout (repository relative):
//
//	.leamas/ledger/meta.json              ledger metadata
//	.leamas/ledger/epics/<EPIC-ID>.json    epic records
//	.leamas/ledger/items/<EPIC>__<ITEM>.json
//	.leamas/ledger/acts/<ACT-ID>.json      ACT execution records
//	.leamas/ledger/evidence/<ID>.json     evidence references
//	.leamas/ledger/decisions/<ID>.json    decisions
//	.leamas/ledger/knowledge/<ID>.json    knowledge references
//
// The JSON records are the single authoritative representation. Markdown
// output is a derived projection.
//
// Every mutation is a multi-record transaction. A transaction is prepared
// in a staging directory, committed by an atomic rename of a single
// manifest pointer, and recovered deterministically: an interrupted
// transaction is either rolled forward or rolled back, never exposed as an
// accepted partial state.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// Errors returned by the store.
var (
	ErrLedgerNotFound  = errors.New("ledger not initialized")
	ErrLedgerExists    = errors.New("ledger already initialized")
	ErrRecordNotFound  = errors.New("record not found")
	ErrDuplicateRecord = errors.New("duplicate record identity")
	ErrStaleRevision   = errors.New("stale revision")
	ErrTransactionOpen = errors.New("transaction already open")
	// ErrOwnershipMismatch is returned when an explicit lock ownership does
	// not cover the ledger root of the operation.
	ErrOwnershipMismatch = errors.New("lock ownership mismatch")
)

// DefaultRoot is the repository-relative ledger root.
const DefaultRoot = ".leamas/ledger"

// Meta is the ledger metadata record.
type Meta struct {
	SchemaVersion int    `json:"schema_version"`
	CreatedAt     string `json:"created_at,omitempty"`
}

// Store is the canonical ledger storage.
type Store struct {
	Root string
}

// New returns a store rooted at the given path.
func New(root string) Store {
	if root == "" {
		root = DefaultRoot
	}
	return Store{Root: root}
}

// Default returns a store rooted at DefaultRoot.
func Default() Store { return New(DefaultRoot) }

// Exists reports whether the ledger root is present.
func (s Store) Exists() bool {
	info, err := os.Stat(filepath.Join(s.Root, "meta.json"))
	return err == nil && !info.IsDir()
}

// Init creates the empty ledger.
func (s Store) Init() error {
	if s.Exists() {
		return fmt.Errorf("%w: %s", ErrLedgerExists, s.Root)
	}
	for _, dir := range s.recordDirs() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create ledger directory %s: %w", dir, err)
		}
	}
	meta := Meta{SchemaVersion: domain.LedgerSchemaVersion}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ledger meta: %w", err)
	}
	if err := WriteFileAtomic(filepath.Join(s.Root, "meta.json"), append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s Store) recordDirs() []string {
	return []string{
		s.Root,
		filepath.Join(s.Root, "epics"),
		filepath.Join(s.Root, "items"),
		filepath.Join(s.Root, "acts"),
		filepath.Join(s.Root, "evidence"),
		filepath.Join(s.Root, "decisions"),
		filepath.Join(s.Root, "knowledge"),
	}
}

// itemFileName returns the deterministic file name of a work item.
func itemFileName(id domain.ScopedItemID) string {
	return strings.ReplaceAll(string(id), "/", "__") + ".json"
}

// Load reads the whole canonical ledger state.
//
// The read holds the shared lock, so a reader never observes a partially
// published multi-record transaction.
//
// The pending journal state is checked UNDER the acquired lock. A canonical
// read is not admitted while an interrupted publication is pending: the
// visible state would be a partial transaction, and it must not be read as
// if it were an accepted state. The recovery operation must run first.
func (s Store) Load() (domain.Ledger, error) {
	var l domain.Ledger
	if !s.Exists() {
		return l, fmt.Errorf("%w: %s", ErrLedgerNotFound, s.Root)
	}
	lock, err := AcquireRead(s.Root)
	if err != nil {
		return l, err
	}
	defer lock.Release()
	if pending, err := s.PendingJournal(); err != nil {
		return l, err
	} else if pending {
		return l, fmt.Errorf("%w: %s has an interrupted transaction; run recovery before reading the canonical state",
			ErrTransactionOpen, s.Root)
	}
	return s.loadUnlocked()
}

// loadUnlocked reads the canonical state WITHOUT taking the lock.
//
// It is used by a writer that already holds the exclusive lock: taking
// the shared lock again inside the exclusive lock would deadlock.
func (s Store) loadUnlocked() (domain.Ledger, error) {
	var l domain.Ledger
	l.SchemaVersion = domain.LedgerSchemaVersion

	epics, err := loadEpics(filepath.Join(s.Root, "epics"))
	if err != nil {
		return l, err
	}
	l.Epics = epics

	items, err := loadItems(filepath.Join(s.Root, "items"))
	if err != nil {
		return l, err
	}
	l.Items = items

	acts, err := loadActs(filepath.Join(s.Root, "acts"))
	if err != nil {
		return l, err
	}
	l.Acts = acts

	evidence, err := loadEvidence(filepath.Join(s.Root, "evidence"))
	if err != nil {
		return l, err
	}
	l.Evidence = evidence

	decisions, err := loadDecisions(filepath.Join(s.Root, "decisions"))
	if err != nil {
		return l, err
	}
	l.Decisions = decisions

	knowledge, err := loadKnowledge(filepath.Join(s.Root, "knowledge"))
	if err != nil {
		return l, err
	}
	l.Knowledge = knowledge

	return l, nil
}
