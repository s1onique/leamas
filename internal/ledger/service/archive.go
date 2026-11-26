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
	// TransferTo names the work item that would conserve the obligation
	// when the archived record is unfinished.
	//
	// UNFINISHED-WORK ARCHIVAL TRANSFERS ARE DISABLED for this release.
	// A complete conservation of a transferred obligation — every
	// downstream edge, every retained obligation, every exit criterion and
	// every evidence reference of the transferred item — cannot be
	// established with a small, bounded implementation. Rather than
	// shipping a transfer that silently drops an obligation, the transfer
	// is rejected outright: an unfinished work item may not be archived.
	//
	// The field is retained so the shape of the request stays explicit.
	// A request that names a transfer target is REJECTED, not ignored.
	TransferTo domain.ScopedItemID
}

// Archive moves a work item into the archived state.
//
// Archival preserves the stable identity, the original terminal
// disposition, the evidence references, the dependency relationships, the
// required successor obligations and the applicable reopen conditions.
// The record stays discoverable.
//
// An UNFINISHED work item may not be archived. Retiring it would make an
// outstanding engineering obligation disappear from the live board while no
// archived record shows it. The obligation must instead be conserved: the
// item must first be accepted.
//
// A terminal ACT verdict is not an acceptance: an ACT that closed over
// unfinished work does not make that work finished.
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
		if err := s.conserveObligation(l, &item, opts.TransferTo); err != nil {
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

// conserveObligation enforces the obligation-conservation rule.
//
// An accepted work item carries no outstanding obligation and may be
// archived freely.
//
// An UNFINISHED work item may not be archived at all in this release. A
// transfer request is explicitly REJECTED rather than partially honoured:
// an unfinished-work archival transfer cannot yet guarantee that the
// transferred obligation is conserved in full.
func (s *Service) conserveObligation(l domain.Ledger, item *domain.WorkItem, transferTo domain.ScopedItemID) error {
	if domain.IsTerminalAcceptance(item.Acceptance) {
		return nil
	}
	if transferTo != "" {
		return fmt.Errorf("%w: %s has acceptance state %q; unfinished-work archival transfers are disabled in this release",
			ErrUnfinishedObligation, item.Scoped().String(), string(item.Acceptance))
	}
	return fmt.Errorf("%w: %s has acceptance state %q; archive requires the work item to be accepted first",
		ErrUnfinishedObligation, item.Scoped().String(), string(item.Acceptance))
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

// ReopenOptions describes an explicit reopening.
type ReopenOptions struct {
	ScopedID    domain.ScopedItemID
	ExpectedRev int
}

// Reopen explicitly reopens a work item.
//
// A reopen is distinct from a restore: a restore brings an ARCHIVED record
// back into the live state, while a reopen resets the lifecycle of a live
// record that was blocked or closed.
//
// A reopen never erases the historical acceptance record. It makes the work
// ACTIONABLE again:
//
//   - the lifecycle returns to the open state, so the item is executable;
//   - the acceptance state returns to pending, so the item is no longer a
//     satisfied obligation;
//   - the acceptance DECISION fields (who accepted, on what basis, against
//     which source revision) are preserved verbatim, so the historical
//     acceptance record survives the reopening.
//
// The preserved decision is not authority: the acceptance authority
// re-evaluates the item against the CURRENT evidence. A reopened item must
// acquire fresh admissible evidence before it can be accepted again.
func (s *Service) Reopen(txID string, opts ReopenOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		item, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if item.Archived {
			return fmt.Errorf("%w: %s is archived; use restore to bring it back",
				ErrAlreadyExists, opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
			return err
		}
		before := item
		item.Lifecycle = domain.LifecycleOpen
		// The reopened work is unfinished work again: the acceptance
		// state returns to pending so the obligation is live and the
		// item is no longer excluded from execution by a terminal
		// acceptance.
		item.Acceptance = domain.AcceptancePending
		// The historical acceptance DECISION is conserved: who
		// accepted, on what basis and against which source revision
		// remain on the record.
		item.Revision++
		updated = item
		if domain.SameWorkItemContent(before, item) {
			return nil
		}
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}

// CloseActOptions describes the recording of a terminal ACT verdict.
type CloseActOptions struct {
	ActID  domain.ActID
	ItemID domain.ScopedItemID
	// ItemIDs are the additional work items the ACT addresses. An ACT may
	// address several work items; each keeps its own acceptance state.
	ItemIDs     []domain.ScopedItemID
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
//
// An ACT may address SEVERAL work items. Each addressed work item keeps its
// own acceptance state and its own retained-obligation record: a terminal
// verdict over one item never implies acceptance of another.
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
		bound := append([]domain.ScopedItemID{}, opts.ItemIDs...)
		if opts.ItemID != "" {
			bound = append(bound, opts.ItemID)
		}
		// Every addressed work item is checked: an ACT may not name a
		// work item that does not exist.
		for _, id := range bound {
			if _, ok := l.ItemByScoped(id); !ok {
				return fmt.Errorf("%w: %s", ErrNotFound, id.String())
			}
		}
		closed = domain.ActExecution{
			SchemaVersion:       domain.LedgerSchemaVersion,
			ActID:               opts.ActID,
			ItemID:              opts.ItemID,
			ItemIDs:             bound,
			Title:               opts.Title,
			Verdict:             opts.Verdict,
			TerminalDisposition: disposition,
			Note:                opts.Note,
			Evidence:            opts.Evidence,
			Revision:            1,
		}
		// The retained obligation is recorded per addressed work item:
		// each keeps its own acceptance state.
		if opts.ItemID != "" {
			item, _ := l.ItemByScoped(opts.ItemID)
			if item.Acceptance != domain.AcceptanceAccepted {
				closed.RetainedObligation = string(item.Acceptance)
			}
		}
		return m.PutAct(closed)
	})
	if err != nil {
		return domain.ActExecution{}, err
	}
	return closed, nil
}
