// Package service implements the Factory Work Ledger operations.
//
// Every operation follows the same shape:
//
//  1. load the canonical state;
//  2. build the candidate state;
//  3. validate the candidate;
//  4. commit the candidate as one atomic transaction.
//
// A rejected candidate leaves the ledger byte-identical to its previous
// state. Revision checks reject stale writers instead of silently
// applying a last-writer-wins update.
package service

import (
	"errors"
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// Errors returned by the service.
var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrStaleRevision = errors.New("stale revision")
)

// Service performs ledger operations against a store.
type Service struct {
	Store store.Store
}

// New returns a service backed by the given store.
func New(s store.Store) *Service { return &Service{Store: s} }

// Load reads the canonical ledger state.
func (s *Service) Load() (domain.Ledger, error) { return s.Store.Load() }

// Init creates the empty ledger.
func (s *Service) Init() error { return s.Store.Init() }

// Recover repairs an interrupted transaction.
func (s *Service) Recover() (store.RecoveryStatus, error) { return s.Store.Recover() }

// checkRevision enforces optimistic concurrency.
//
// A mutation that declares an expected revision must match the stored
// revision exactly. A mismatch is a stale write and is rejected.
func checkRevision(expected, actual int) error {
	if expected <= 0 {
		return nil
	}
	if expected != actual {
		return fmt.Errorf("%w: expected revision %d, stored revision is %d",
			ErrStaleRevision, expected, actual)
	}
	return nil
}

// mutate applies a mutation to the canonical state under one transaction.
func (s *Service) mutate(txID string, apply func(l domain.Ledger, m *store.Mutation) error) error {
	txn, err := s.Store.BeginMutation(txID)
	if err != nil {
		return err
	}
	current, err := s.Store.Load()
	if err != nil {
		return err
	}
	if err := apply(current, txn); err != nil {
		return err
	}
	return txn.Commit()
}

// CreateItemOptions describes a new work item.
type CreateItemOptions struct {
	EpicID       domain.EpicID
	ItemID       domain.ItemID
	Title        string
	Summary      string
	Kind         domain.ItemKind
	Priority     domain.Priority
	Owner        string
	Criteria     []string
	ExitCriteria []string
	RequiresLive bool
	Provenance   *domain.Provenance
}

// CreateItem creates a new work item.
func (s *Service) CreateItem(txID string, opts CreateItemOptions) (domain.WorkItem, error) {
	var created domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		if err := domain.ValidateEpicID(opts.EpicID); err != nil {
			return err
		}
		if err := domain.ValidateItemID(opts.ItemID); err != nil {
			return err
		}
		scoped := domain.NewScopedItemID(opts.EpicID, opts.ItemID)
		if _, exists := l.ItemByScoped(scoped); exists {
			return fmt.Errorf("%w: %s", ErrAlreadyExists, scoped.String())
		}
		kind := opts.Kind
		if kind == "" {
			kind = domain.ItemKindTask
		}
		created = domain.WorkItem{
			SchemaVersion:      domain.LedgerSchemaVersion,
			EpicID:             opts.EpicID,
			ItemID:             opts.ItemID,
			Title:              opts.Title,
			Summary:            opts.Summary,
			Kind:               kind,
			Lifecycle:          domain.LifecycleOpen,
			Acceptance:         domain.AcceptancePending,
			Priority:           opts.Priority,
			Quality:            domain.EvidenceQualityNone,
			Owner:              opts.Owner,
			Revision:           1,
			AcceptanceCriteria: opts.Criteria,
			ExitCriteria:       opts.ExitCriteria,
			RequiresLiveGate:   opts.RequiresLive,
			Provenance:         opts.Provenance,
		}
		return m.PutItem(created)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return created, nil
}

// UpdateItemOptions describes a work-item mutation.
type UpdateItemOptions struct {
	ScopedID     domain.ScopedItemID
	ExpectedRev  int
	Title        *string
	Summary      *string
	Kind         *domain.ItemKind
	Lifecycle    *domain.LifecycleState
	Acceptance   *domain.AcceptanceState
	Priority     *domain.Priority
	Quality      *domain.EvidenceQuality
	Owner        *string
	Criteria     []string
	ExitCriteria []string
	RequiresLive *bool
}

// UpdateItem mutates a work item under an optimistic concurrency check.
func (s *Service) UpdateItem(txID string, opts UpdateItemOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		it, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, it.Revision); err != nil {
			return err
		}
		if opts.Title != nil {
			it.Title = *opts.Title
		}
		if opts.Summary != nil {
			it.Summary = *opts.Summary
		}
		if opts.Kind != nil {
			it.Kind = *opts.Kind
		}
		if opts.Lifecycle != nil {
			it.Lifecycle = *opts.Lifecycle
		}
		if opts.Acceptance != nil {
			it.Acceptance = *opts.Acceptance
		}
		if opts.Priority != nil {
			it.Priority = *opts.Priority
		}
		if opts.Quality != nil {
			it.Quality = *opts.Quality
		}
		if opts.Owner != nil {
			it.Owner = *opts.Owner
		}
		if opts.Criteria != nil {
			it.AcceptanceCriteria = opts.Criteria
		}
		if opts.ExitCriteria != nil {
			it.ExitCriteria = opts.ExitCriteria
		}
		if opts.RequiresLive != nil {
			it.RequiresLiveGate = *opts.RequiresLive
		}
		it.Revision++
		updated = it
		return m.PutItem(it)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}
