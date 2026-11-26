package service

import (
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// CheckpointOptions describes a durable semantic checkpoint.
type CheckpointOptions struct {
	// ScopedID is the work item the checkpoint applies to.
	ScopedID domain.ScopedItemID
	// ExpectedRev is the expected revision of the work item.
	ExpectedRev int
	// Lifecycle is the new lifecycle state.
	Lifecycle domain.LifecycleState
	// Summary records what the checkpoint captured.
	Summary string
}

// Checkpoint records a durable semantic checkpoint on a work item.
//
// A checkpoint records progress. It never accepts the work item: the
// acceptance state is a separate concern and is only changed by an
// explicit acceptance decision backed by sufficient evidence.
func (s *Service) Checkpoint(txID string, opts CheckpointOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		item, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
			return err
		}
		if !domain.IsValidLifecycleState(opts.Lifecycle) {
			return fmt.Errorf("invalid lifecycle state %q", string(opts.Lifecycle))
		}
		if opts.Lifecycle == domain.LifecycleArchived {
			return fmt.Errorf("use archive to move a work item into the archived state")
		}
		if opts.Summary != "" {
			item.Summary = opts.Summary
		}
		item.Lifecycle = opts.Lifecycle
		item.Revision++
		updated = item
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}

// ClaimOptions describes a work-item ownership claim.
type ClaimOptions struct {
	ScopedID    domain.ScopedItemID
	ExpectedRev int
	Owner       string
}

// Claim assigns an owner to a work item.
func (s *Service) Claim(txID string, opts ClaimOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		item, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
			return err
		}
		if opts.Owner == "" {
			return fmt.Errorf("claim requires a non-empty owner")
		}
		item.Owner = opts.Owner
		if item.Lifecycle == domain.LifecycleOpen {
			item.Lifecycle = domain.LifecycleInProgress
		}
		item.Revision++
		updated = item
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}

// AcceptOptions describes an explicit acceptance decision.
type AcceptOptions struct {
	ScopedID    domain.ScopedItemID
	ExpectedRev int
	// EvidenceQuality is the quality of the evidence backing the decision.
	EvidenceQuality domain.EvidenceQuality
}

// Accept records an engineering acceptance decision on a work item.
//
// The acceptance is only permitted when the recorded evidence quality is
// sufficient. Insufficient evidence never becomes successful acceptance:
// the candidate state is rejected by the validator before it is published.
func (s *Service) Accept(txID string, opts AcceptOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		item, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
			return err
		}
		quality := opts.EvidenceQuality
		if quality == "" {
			quality = item.Quality
		}
		item.Quality = quality
		item.Acceptance = domain.AcceptanceAccepted
		item.Lifecycle = domain.LifecycleClosed
		item.Revision++
		updated = item
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}
