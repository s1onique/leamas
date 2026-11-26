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
func (s Store) Load() (domain.Ledger, error) {
	var l domain.Ledger
	if !s.Exists() {
		return l, fmt.Errorf("%w: %s", ErrLedgerNotFound, s.Root)
	}
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
