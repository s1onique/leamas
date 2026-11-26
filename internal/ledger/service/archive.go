package service

import (
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// ArchiveOptions describes a work-item archival.
type ArchiveOptions struct {
	ScopedID    domain.ScopedItemID
	ExpectedRev int
	// ReopenConditions are the applicable reopen conditions of the record.
	ReopenConditions []domain.ReopenCondition
}

// Archive moves a work item into the archived state.
//
// Archival preserves the stable identity, the original terminal
// disposition, the evidence references, the dependency relationships, the
// required successor obligations and the applicable reopen conditions.
// The record stays discoverable.
func (s *Service) Archive(txID string, opts ArchiveOptions) (domain.WorkItem, error) {
	var archived domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		item, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if item.Archived {
			return fmt.Errorf("%w: %s is already archived", ErrAlreadyExists, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
			return err
		}
		if item.TerminalDisposition == "" {
			item.TerminalDisposition = string(item.Lifecycle) + "/" + string(item.Acceptance)
		}
		item.Lifecycle = domain.LifecycleArchived
		item.Archived = true
		if len(opts.ReopenConditions) > 0 {
			item.Reopen = append(item.Reopen, opts.ReopenConditions...)
		}
		item.Revision++
		archived = item
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return archived, nil
}

// RestoreOptions describes a work-item restoration.
type RestoreOptions struct {
	ScopedID    domain.ScopedItemID
	ExpectedRev int
}

// Restore brings an archived work item back into the live state.
//
// The stable identity is preserved. The restoration refuses to reuse an
// identity that is already live.
func (s *Service) Restore(txID string, opts RestoreOptions) (domain.WorkItem, error) {
	var restored domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		item, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if !item.Archived {
			return fmt.Errorf("%w: %s is not archived", ErrAlreadyExists, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
			return err
		}
		item.Archived = false
		item.Lifecycle = domain.LifecycleOpen
		item.Revision++
		restored = item
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return restored, nil
}

// CloseActOptions describes the recording of a terminal ACT verdict.
type CloseActOptions struct {
	ActID       domain.ActID
	ItemID      domain.ScopedItemID
	Title       string
	Verdict     domain.ActVerdict
	Note        string
	ExpectedRev int
	// TerminalDisposition is the preserved original disposition.
	TerminalDisposition string
	Evidence            []domain.EvidenceRef
}

// CloseAct records a terminal ACT verdict bound to a work item.
//
// The terminal verdict is recorded on the ACT execution record. It never
// closes the associated engineering obligation: the work item keeps its
// own acceptance state.
func (s *Service) CloseAct(txID string, opts CloseActOptions) (domain.ActExecution, error) {
	var closed domain.ActExecution
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		if err := domain.ValidateActID(opts.ActID); err != nil {
			return err
		}
		if !domain.IsValidActVerdict(opts.Verdict) {
			return fmt.Errorf("invalid act verdict %q", string(opts.Verdict))
		}
		if !domain.IsTerminalActVerdict(opts.Verdict) {
			return fmt.Errorf("act verdict %q is not terminal", string(opts.Verdict))
		}
		disposition := opts.TerminalDisposition
		if disposition == "" {
			disposition = string(opts.Verdict)
		}
		retained := ""
		if opts.ItemID != "" {
			item, ok := l.ItemByScoped(opts.ItemID)
			if !ok {
				return fmt.Errorf("%w: %s", ErrNotFound, opts.ItemID.String())
			}
			if item.Acceptance != domain.AcceptanceAccepted {
				// The terminal verdict does NOT retire the engineering
				// obligation. The retention is recorded explicitly so the
				// obligation survives the ACT.
				retained = string(item.Acceptance)
			}
		}
		closed = domain.ActExecution{
			SchemaVersion:       domain.LedgerSchemaVersion,
			ActID:               opts.ActID,
			ItemID:              opts.ItemID,
			Title:               opts.Title,
			Verdict:             opts.Verdict,
			TerminalDisposition: disposition,
			RetainedObligation:  retained,
			Note:                opts.Note,
			Evidence:            opts.Evidence,
			Revision:            1,
		}
		return m.PutAct(closed)
	})
	if err != nil {
		return domain.ActExecution{}, err
	}
	return closed, nil
}
