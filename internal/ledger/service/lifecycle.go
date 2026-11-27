package service

import (
	"fmt"
	"strings"

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
		// The checkpoint path reuses the SAME lifecycle-transition
		// authority as the generic update path: a terminal transition may
		// not conceal an unfinished engineering obligation.
		//
		// A record that already carries a preserved historical terminal
		// disposition is tolerated: its qualification may be unknown, but
		// the disposition is historical fact and is conserved by the
		// transition.
		if err := checkLifecycleTransition(item, opts.Lifecycle, true); err != nil {
			return err
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
//
// A claim NEVER replaces an existing owner. Replacing an owner is an
// explicit, bounded release or transfer operation (see
// TransferOwnership). A claim that names a different owner than the stored
// one is rejected.
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
		if item.Owner != "" && item.Owner != opts.Owner {
			return fmt.Errorf("%w: %s is owned by %q; use an explicit ownership transfer",
				ErrOwnershipTransferRequired, opts.ScopedID.String(), item.Owner)
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
	// It is a REQUEST, never authority: it cannot establish verification.
	EvidenceQuality domain.EvidenceQuality
	// AcceptedBy is the explicit engineering acceptance: the scoped actor
	// that accepts the work item. It is REQUIRED. Leamas never
	// autonomously determines whether a live experiment actually
	// occurred: the engineering judgment is the caller's.
	AcceptedBy string
	// Basis records the engineering justification of the decision.
	Basis string
	// SourceHead binds the acceptance decision to the repository/source
	// revision it was made against. It is REQUIRED when the item declares
	// that its acceptance requires a source binding.
	SourceHead string
}

// Accept records an engineering acceptance decision on a work item.
//
// The acceptance is only permitted when the linked evidence is admissible:
// the artifact identity was actually verified, the evidence is explicitly
// classified, and the evidence is associated with the criterion it claims
// to support. A caller-supplied quality label never establishes
// verification.
//
// The decision is checked against the SINGLE acceptance authority
// (domain.EvaluateAcceptance). The canonical validator consults the same
// authority: there is no second, competing acceptance rule.
//
// A mechanical artifact verification is not autonomous verification of an
// engineering claim: the acceptance decision remains an explicit, scoped
// act of the caller, and the caller must name itself.
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
		if strings.TrimSpace(opts.AcceptedBy) == "" {
			return fmt.Errorf("%w: an explicit acceptance requires the accepting actor",
				ErrAcceptanceRejected)
		}
		authority := domain.EvaluateAcceptance(l, item)
		if !authority.Admissible {
			return fmt.Errorf("%w: %s: %s",
				ErrAcceptanceRejected, opts.ScopedID.String(), authority.Reason)
		}
		// FRESHNESS BOUNDARY. The acceptance boundary revalidates the
		// SUPPORTING artifacts of this work item: an artifact whose bytes
		// changed after its verification no longer establishes the
		// acceptance. The revalidation is bounded to the artifacts this
		// decision relies on: an ordinary read never triggers a full
		// historical artifact scan.
		if err := s.revalidateSupportingArtifacts(l, item); err != nil {
			return fmt.Errorf("%w: %s: %s",
				ErrAcceptanceRejected, opts.ScopedID.String(), err.Error())
		}
		if opts.EvidenceQuality != "" {
			item.Quality = opts.EvidenceQuality
		}
		item.Acceptance = domain.AcceptanceAccepted
		item.Lifecycle = domain.LifecycleClosed
		item.AcceptedBy = strings.TrimSpace(opts.AcceptedBy)
		item.AcceptanceBasis = opts.Basis
		item.AcceptanceSourceHead = opts.SourceHead
		item.Revision++
		updated = item
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}
