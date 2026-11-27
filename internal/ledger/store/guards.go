package store

import (
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// checkRevisionGuards rejects a staged write that would overwrite a newer
// stored revision.
//
// The guard compares the stored revision against the staged revision.
//
//   - a staged revision LOWER than the stored revision is a stale write
//     and is always rejected;
//   - a staged revision EQUAL to the stored revision is rejected only when
//     the staged content differs from the stored content. An identical
//     replay is idempotent and is allowed through.
//
// This is what makes two writers that share an expected revision safe:
// the first writer advances the stored revision, and the second writer's
// equal-but-different content is rejected as a stale write.
func (m *Mutation) checkRevisionGuards(candidate domain.Ledger) error {
	stored, err := m.store.loadUnlocked()
	if err != nil {
		return err
	}
	for _, staged := range candidate.Items {
		prior, ok := stored.ItemByScoped(staged.Scoped())
		if !ok {
			continue
		}
		if staged.Revision < prior.Revision {
			return fmt.Errorf("%w: %s has stored revision %d, staged revision %d",
				ErrStaleRevision, staged.Scoped().String(), prior.Revision, staged.Revision)
		}
		if staged.Revision == prior.Revision &&
			!domain.SameWorkItemContent(prior, staged) {
			return fmt.Errorf("%w: %s has stored revision %d, staged revision %d with different content",
				ErrStaleRevision, staged.Scoped().String(), prior.Revision, staged.Revision)
		}
	}
	for _, staged := range candidate.Evidence {
		prior, ok := stored.EvidenceByID(staged.EvidenceID)
		if !ok {
			continue
		}
		if staged.Revision < prior.Revision {
			return fmt.Errorf("%w: evidence %s has stored revision %d, staged revision %d",
				ErrStaleRevision, string(staged.EvidenceID), prior.Revision, staged.Revision)
		}
		if staged.Revision == prior.Revision && !domain.SameEvidenceContent(prior, staged) {
			return fmt.Errorf("%w: evidence %s has stored revision %d, staged revision %d with different content",
				ErrStaleRevision, string(staged.EvidenceID), prior.Revision, staged.Revision)
		}
	}
	for _, staged := range candidate.Acts {
		prior, ok := stored.ActByID(staged.ActID)
		if !ok {
			continue
		}
		if staged.Revision < prior.Revision {
			return fmt.Errorf("%w: act %s has stored revision %d, staged revision %d",
				ErrStaleRevision, string(staged.ActID), prior.Revision, staged.Revision)
		}
		if staged.Revision == prior.Revision && !domain.SameActContent(prior, staged) {
			return fmt.Errorf("%w: act %s has stored revision %d, staged revision %d with different content",
				ErrStaleRevision, string(staged.ActID), prior.Revision, staged.Revision)
		}
	}
	return nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
