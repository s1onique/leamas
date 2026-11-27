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
	// Class is the explicit evidence class. It determines which kind of
	// claim the artifact may support. It is required for an artifact that
	// is to establish a live qualification.
	Class domain.EvidenceClass
	// Criterion is the acceptance criterion this evidence supports. It is
	// the explicit criterion/scope association.
	Criterion string
	// ByteSize is the declared artifact size. When it is zero the size is
	// taken from the verified artifact.
	ByteSize int64
	// SourceHead binds the evidence to the repository/source HEAD it was
	// produced against.
	SourceHead string
}

// AddEvidence creates an evidence reference and binds it to a work item.
//
// The evidence reference records the artifact identity (path and digest)
// and its quality. It never decides acceptance: the acceptance state of
// the bound work item is not changed by this operation.
//
// When the reference declares a relative path and a digest, the artifact
// identity is ACTUALLY VERIFIED against the repository: the file must
// exist at the declared relative path, its byte size must match and its
// SHA-256 must match. A missing, incorrect or unavailable artifact never
// acquires verified integrity.
//
// The caller-supplied quality is a request. It never establishes
// verification on its own: only a successful artifact verification does.
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
		class := opts.Class
		if class == "" {
			class = domain.EvidenceClassStructural
		}
		if !domain.IsValidEvidenceClass(class) {
			return fmt.Errorf("invalid evidence class %q", string(class))
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

		record := domain.EvidenceRefRecord{
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
			Class:         class,
			Criterion:     opts.Criterion,
			ByteSize:      opts.ByteSize,
			SourceHead:    opts.SourceHead,
		}

		// The artifact identity is verified against the repository. A
		// failure here rejects the registration entirely: a missing or
		// incorrect artifact never acquires verified integrity.
		if opts.RelativePath != "" && opts.SHA256 != "" {
			identity, err := domain.VerifyArtifact(
				s.RepoRoot(), opts.RelativePath, opts.SHA256, opts.ByteSize)
			if err != nil {
				return fmt.Errorf("%w: evidence %s: %v",
					ErrEvidenceVerificationFailed, string(opts.EvidenceID), err)
			}
			record.ByteSize = identity.ByteSize
			record.SHA256 = identity.SHA256
			record.Verified = true
			// A verified artifact carries verified quality. The
			// caller cannot claim a weaker or a stronger quality than
			// the verification establishes.
			record.Quality = domain.EvidenceQualityVerified
		}

		if err := m.PutEvidence(record); err != nil {
			return err
		}
		created = record
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
			Quality:    record.Quality,
			Class:      record.Class,
			Criterion:  record.Criterion,
		})
		item.Quality = promoteQuality(item.Quality, record.Quality)
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

// RevalidateEvidenceOptions describes an evidence re-verification.
type RevalidateEvidenceOptions struct {
	// EvidenceID is the evidence record to re-verify. When it is empty,
	// every evidence record is re-verified.
	EvidenceID domain.EvidenceID
}

// RevalidateEvidence re-checks the artifact identity of the evidence
// records against the repository.
//
// An evidence record whose artifact changed, was removed or was replaced
// loses its verified integrity. The record is demoted to the declared
// quality: it never keeps a verified status it can no longer prove.
//
// A demoted evidence record can no longer support an acceptance.
// revalidateSupportingArtifacts re-checks the artifact identity of the
// evidence records that support the acceptance of the given work item.
//
// It is the freshness boundary of the acceptance operation: an artifact
// whose bytes changed after its verification no longer establishes the
// acceptance, so the acceptance is refused.
//
// The scan is BOUNDED to the artifacts this decision relies on. It is
// never a full historical artifact scan: an ordinary read operation does
// not revalidate the whole ledger.
func (s *Service) revalidateSupportingArtifacts(l domain.Ledger, item domain.WorkItem) error {
	repoRoot := s.RepoRoot()
	for _, ref := range item.Evidence {
		rec, ok := l.EvidenceByID(ref.EvidenceID)
		if !ok {
			continue
		}
		if !rec.Verified {
			continue
		}
		validation := domain.RevalidateEvidence(repoRoot, rec)
		if validation.Valid {
			continue
		}
		return fmt.Errorf("evidence %q is stale: %s",
			string(rec.EvidenceID), validation.Reason)
	}
	return nil
}

func (s *Service) RevalidateEvidence(txID string, opts RevalidateEvidenceOptions) ([]domain.EvidenceValidation, error) {
	var out []domain.EvidenceValidation
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		for _, rec := range l.Evidence {
			if opts.EvidenceID != "" && rec.EvidenceID != opts.EvidenceID {
				continue
			}
			if !rec.Verified {
				continue
			}
			validation := domain.RevalidateEvidence(s.RepoRoot(), rec)
			out = append(out, validation)
			if validation.Valid {
				continue
			}
			// The artifact no longer carries the declared identity: the
			// verified status is withdrawn.
			updated := rec
			updated.Verified = false
			updated.Quality = domain.EvidenceQualityReferenced
			updated.Revision++
			if err := m.PutEvidence(updated); err != nil {
				return err
			}
			// The bound work item loses the promoted quality as well.
			if updated.ItemID != "" {
				item, ok := l.ItemByScoped(updated.ItemID)
				if !ok {
					continue
				}
				item.Quality = domain.EvidenceQualityNone
				item.Revision++
				if err := m.PutItem(item); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
