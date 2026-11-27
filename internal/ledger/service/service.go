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
	"path/filepath"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// Errors returned by the service.
var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrStaleRevision = errors.New("stale revision")

	// ErrAcceptanceRejected is returned when an acceptance decision is not
	// backed by sufficient, correctly classified and correctly associated
	// evidence. A caller-supplied quality label never establishes
	// verification.
	ErrAcceptanceRejected = errors.New("acceptance rejected: insufficient or inadmissible evidence")

	// ErrUnfinishedObligation is returned when an archival would retire an
	// unfinished engineering obligation without an explicitly authorized
	// and conserved obligation transfer.
	ErrUnfinishedObligation = errors.New("unfinished obligation may not be archived")

	// ErrOwnershipTransferRequired is returned when a mutation would
	// silently replace the owner of an existing record. An explicit bounded
	// release or transfer operation is required.
	ErrOwnershipTransferRequired = errors.New("ownership transfer requires an explicit operation")

	// ErrEvidenceVerificationFailed is returned when the actual artifact
	// identity could not be verified against the repository. A missing,
	// incorrect or unavailable artifact never acquires verified integrity.
	ErrEvidenceVerificationFailed = errors.New("evidence artifact verification failed")
)

// Service performs ledger operations against a store.
type Service struct {
	Store store.Store
	// RepoRootOverride is the repository root the evidence artifacts are
	// resolved against. When it is empty, RepoRoot derives the root from
	// the store layout.
	RepoRootOverride string
}

// New returns a service backed by the given store.
func New(s store.Store) *Service { return &Service{Store: s} }

// WithRepoRoot returns a service whose evidence artifacts are resolved
// against the given repository root.
func (s *Service) WithRepoRoot(root string) *Service {
	s.RepoRootOverride = root
	return s
}

// RepoRoot returns the repository root the evidence artifacts are resolved
// against.
//
// The ledger root is repository-relative (".leamas/ledger"), so the
// repository root is the parent of the ledger root's parent. When the
// ledger root does not follow that layout, the ledger root itself is used
// as the artifact resolution root.
func (s *Service) RepoRoot() string {
	if s.RepoRootOverride != "" {
		return s.RepoRootOverride
	}
	root := s.Store.Root
	if root == "" {
		root = store.DefaultRoot
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	// <repo>/.leamas/ledger -> <repo>
	if filepath.Base(abs) == "ledger" && filepath.Base(filepath.Dir(abs)) == ".leamas" {
		return filepath.Dir(filepath.Dir(abs))
	}
	return abs
}

// Load reads the canonical ledger state.
func (s *Service) Load() (domain.Ledger, error) { return s.Store.Load() }

// Init creates the empty ledger.
func (s *Service) Init() error { return s.Store.Init() }

// Recover repairs an interrupted transaction.
func (s *Service) Recover() (store.RecoveryStatus, error) { return s.Store.Recover() }

// checkRevision enforces optimistic concurrency.
//
// A mutation of an EXISTING record must declare the expected revision, and
// the declared revision must match the stored revision exactly.
//
// A serialized internal transaction does not substitute for the caller's
// expected revision: the caller must state which version of the record it
// observed. A zero or negative expected revision is therefore REJECTED for
// an externally initiated mutation of an existing record, not treated as
// "no check".
func checkRevision(expected, actual int) error {
	if expected <= 0 {
		return fmt.Errorf("%w: an externally initiated mutation of an existing record requires a positive expected revision",
			ErrStaleRevision)
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
	// A rejected candidate must not leave the ledger locked: the
	// transaction is abandoned without publishing anything.
	committed := false
	defer func() {
		if !committed {
			txn.Release()
		}
	}()
	current, err := txn.LoadCurrent()
	if err != nil {
		return err
	}
	if err := apply(current, txn); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
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
	ScopedID    domain.ScopedItemID
	ExpectedRev int
	Title       *string
	Summary     *string
	Kind        *domain.ItemKind
	Lifecycle   *domain.LifecycleState
	// Acceptance is a REQUEST. It is never honoured by the generic update
	// path: the acceptance authority is exercised by the explicit Accept
	// operation only.
	Acceptance   *domain.AcceptanceState
	Priority     *domain.Priority
	Quality      *domain.EvidenceQuality
	Owner        *string
	Criteria     []string
	ExitCriteria []string
	RequiresLive *bool
}

// UpdateItem mutates a work item under an optimistic concurrency check.
//
// Three rules constrain the mutation:
//
//   - an existing owner is never silently replaced. The transfer must be an
//     explicit, bounded operation (see TransferOwnership);
//   - a semantic no-op does not increment the revision and does not
//     rewrite canonical state;
//   - a generic lifecycle update may never HIDE an unresolved obligation.
//     A work item whose acceptance obligation is still outstanding cannot
//     be moved into the closed or archived state through this path: the
//     generic update is not the operation that discharges an obligation.
//     Closing or archiving unfinished work goes through the explicit
//     acceptance or the explicit archival operation.
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
		if opts.Owner != nil && *opts.Owner != it.Owner && it.Owner != "" {
			return fmt.Errorf("%w: %s is owned by %q; use an explicit ownership transfer",
				ErrOwnershipTransferRequired, opts.ScopedID.String(), it.Owner)
		}
		before := it

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
			if err := checkLifecycleDoesNotHideObligation(it, *opts.Lifecycle); err != nil {
				return err
			}
			it.Lifecycle = *opts.Lifecycle
		}
		if opts.Acceptance != nil && *opts.Acceptance != it.Acceptance {
			return fmt.Errorf("%w: acceptance cannot be set by a generic update; use the explicit acceptance operation",
				ErrAcceptanceRejected)
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
		if domain.SameWorkItemContent(before, it) {
			// A semantic no-op rewrites nothing: the revision stays and
			// the canonical bytes are not touched.
			updated = it
			return nil
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

// TransferOwnershipOptions describes an explicit ownership transfer.
type TransferOwnershipOptions struct {
	ScopedID    domain.ScopedItemID
	ExpectedRev int
	// Owner is the new owner.
	Owner string
	// FromOwner is the expected current owner. It must match the stored
	// owner exactly, so a transfer cannot be applied to a record whose
	// ownership changed in the meantime.
	FromOwner string
}

// TransferOwnership transfers the ownership of a work item.
//
// This is the explicit, bounded release or transfer operation. It is the
// only way to replace an existing owner.
func (s *Service) TransferOwnership(txID string, opts TransferOwnershipOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		it, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, it.Revision); err != nil {
			return err
		}
		if opts.Owner == "" {
			return fmt.Errorf("transfer requires a non-empty owner")
		}
		if it.Owner != opts.FromOwner {
			return fmt.Errorf("%w: %s is owned by %q, not %q",
				ErrOwnershipTransferRequired, opts.ScopedID.String(), it.Owner, opts.FromOwner)
		}
		it.Owner = opts.Owner
		it.Revision++
		updated = it
		return m.PutItem(it)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}
