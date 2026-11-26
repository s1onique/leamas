package service

import (
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// LinkOptions describes a typed relationship between two work items.
type LinkOptions struct {
	// From is the source work item.
	From domain.ScopedItemID
	// To is the target work item.
	To domain.ScopedItemID
	// Kind is the typed relation kind.
	Kind domain.DependencyKind
	// ExpectedRev is the expected revision of the source work item.
	ExpectedRev int
	// Note is an optional operator note. It is never authority.
	Note string
}

// Link adds a typed dependency between two work items.
func (s *Service) Link(txID string, opts LinkOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		if !domain.IsValidDependencyKind(opts.Kind) {
			return fmt.Errorf("invalid dependency kind %q", string(opts.Kind))
		}
		from, ok := l.ItemByScoped(opts.From)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.From.String())
		}
		if _, ok := l.ItemByScoped(opts.To); !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.To.String())
		}
		if opts.From == opts.To {
			return fmt.Errorf("work item %q cannot depend on itself", opts.From.String())
		}
		if err := checkRevision(opts.ExpectedRev, from.Revision); err != nil {
			return err
		}
		for _, dep := range from.Dependencies {
			if dep.Ref == opts.To && dep.Kind == opts.Kind {
				return fmt.Errorf("%w: dependency %s %s already exists",
					ErrAlreadyExists, opts.Kind, opts.To.String())
			}
		}
		from.Dependencies = append(from.Dependencies, domain.Dependency{
			Kind: opts.Kind,
			Ref:  opts.To,
			Note: opts.Note,
		})
		sortDependencies(from.Dependencies)
		from.Revision++
		updated = from
		return m.PutItem(from)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}

func sortDependencies(deps []domain.Dependency) {
	for i := 1; i < len(deps); i++ {
		for j := i; j > 0; j-- {
			left, right := deps[j-1], deps[j]
			if left.Kind < right.Kind ||
				(left.Kind == right.Kind && left.Ref <= right.Ref) {
				break
			}
			deps[j-1], deps[j] = deps[j], deps[j-1]
		}
	}
}

// AddEvidenceOptions describes a new evidence reference.
type AddEvidenceOptions struct {
	EvidenceID   domain.EvidenceID
	Kind         string
	Title        string
	RelativePath string
	SHA256       string
	ItemID       domain.ScopedItemID
	Quality      domain.EvidenceQuality
	ExpectedRev  int
	Provenance   *domain.Provenance
}

// AddEvidence creates an evidence reference and binds it to a work item.
//
// The evidence reference records the artifact identity (path and digest)
// and its quality. It never decides acceptance: the acceptance state of
// the bound work item is not changed by this operation.
func (s *Service) AddEvidence(txID string, opts AddEvidenceOptions) (domain.EvidenceRefRecord, error) {
	var created domain.EvidenceRefRecord
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		if err := domain.ValidateEvidenceID(opts.EvidenceID); err != nil {
			return err
		}
		if _, exists := l.EvidenceByID(opts.EvidenceID); exists {
			return fmt.Errorf("%w: %s", ErrAlreadyExists, string(opts.EvidenceID))
		}
		if opts.RelativePath != "" {
			if err := domain.ValidateRelativePath(opts.RelativePath); err != nil {
				return err
			}
		}
		if opts.SHA256 != "" {
			if err := domain.ValidateSHA256(opts.SHA256); err != nil {
				return err
			}
		}
		quality := opts.Quality
		if quality == "" {
			quality = domain.EvidenceQualityDeclared
		}
		if !domain.IsValidEvidenceQuality(quality) {
			return fmt.Errorf("invalid evidence quality %q", string(quality))
		}

		itemID := opts.ItemID
		if itemID != "" {
			item, ok := l.ItemByScoped(itemID)
			if !ok {
				return fmt.Errorf("%w: %s", ErrNotFound, itemID.String())
			}
			if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
				return err
			}
		}

		created = domain.EvidenceRefRecord{
			SchemaVersion: domain.LedgerSchemaVersion,
			EvidenceID:    opts.EvidenceID,
			Kind:          opts.Kind,
			Title:         opts.Title,
			RelativePath:  opts.RelativePath,
			SHA256:        opts.SHA256,
			ItemID:        itemID,
			Quality:       quality,
			Revision:      1,
			Provenance:    opts.Provenance,
		}
		if err := m.PutEvidence(created); err != nil {
			return err
		}
		if itemID == "" {
			return nil
		}

		item, _ := l.ItemByScoped(itemID)
		for _, existing := range item.Evidence {
			if existing.EvidenceID == opts.EvidenceID {
				return fmt.Errorf("%w: evidence %s already bound to %s",
					ErrAlreadyExists, string(opts.EvidenceID), itemID.String())
			}
		}
		item.Evidence = append(item.Evidence, domain.EvidenceRef{
			EvidenceID: opts.EvidenceID,
			Quality:    quality,
		})
		item.Quality = promoteQuality(item.Quality, quality)
		item.Revision++
		return m.PutItem(item)
	})
	if err != nil {
		return domain.EvidenceRefRecord{}, err
	}
	return created, nil
}

// SetSuccessorOptions describes a successor obligation assignment.
type SetSuccessorOptions struct {
	// ScopedID is the work item that hands the obligation over.
	ScopedID domain.ScopedItemID
	// Successor is the work item that takes the obligation over.
	Successor domain.ScopedItemID
	// ExpectedRev is the expected revision of the source work item.
	ExpectedRev int
}

// SetSuccessor records a successor obligation on a work item.
//
// The successor relation is typed and validated: the successor must exist
// and must not be the work item itself.
func (s *Service) SetSuccessor(txID string, opts SetSuccessorOptions) (domain.WorkItem, error) {
	var updated domain.WorkItem
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		item, ok := l.ItemByScoped(opts.ScopedID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.ScopedID.String())
		}
		if _, ok := l.ItemByScoped(opts.Successor); !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, opts.Successor.String())
		}
		if opts.ScopedID == opts.Successor {
			return fmt.Errorf("work item %q cannot be its own successor", opts.ScopedID.String())
		}
		if err := checkRevision(opts.ExpectedRev, item.Revision); err != nil {
			return err
		}
		for _, s := range item.Successors {
			if s == opts.Successor {
				return fmt.Errorf("%w: successor %s already recorded",
					ErrAlreadyExists, opts.Successor.String())
			}
		}
		item.Successors = append(item.Successors, opts.Successor)
		item.Revision++
		updated = item
		return m.PutItem(item)
	})
	if err != nil {
		return domain.WorkItem{}, err
	}
	return updated, nil
}

// promoteQuality returns the stronger of two evidence qualities.
func promoteQuality(current, added domain.EvidenceQuality) domain.EvidenceQuality {
	rank := func(q domain.EvidenceQuality) int {
		switch q {
		case domain.EvidenceQualityVerified:
			return 3
		case domain.EvidenceQualityReferenced:
			return 2
		case domain.EvidenceQualityDeclared:
			return 1
		case domain.EvidenceQualityNone:
			return 0
		}
		return 0
	}
	if rank(added) > rank(current) {
		return added
	}
	return current
}
