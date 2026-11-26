package service

import (
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// RecordDecisionOptions describes a durable decision registration.
type RecordDecisionOptions struct {
	DecisionID domain.DecisionID
	Title      string
	Rationale  string
	ItemID     domain.ScopedItemID
	Supersedes []domain.DecisionID
	Provenance *domain.Provenance
}

// RecordDecision registers a durable recorded decision.
//
// A decision is a validated record, not a note. It is bound to the
// canonical state and is discoverable through the historical lookup.
func (s *Service) RecordDecision(txID string, opts RecordDecisionOptions) (domain.Decision, error) {
	var created domain.Decision
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		if err := domain.ValidateDecisionID(opts.DecisionID); err != nil {
			return err
		}
		if opts.Title == "" {
			return fmt.Errorf("decision requires a non-empty title")
		}
		if opts.ItemID != "" {
			if _, ok := l.ItemByScoped(opts.ItemID); !ok {
				return fmt.Errorf("%w: %s", ErrNotFound, opts.ItemID.String())
			}
		}
		created = domain.Decision{
			SchemaVersion: domain.LedgerSchemaVersion,
			DecisionID:    opts.DecisionID,
			Title:         opts.Title,
			Rationale:     opts.Rationale,
			ItemID:        opts.ItemID,
			Supersedes:    opts.Supersedes,
			Revision:      1,
			Provenance:    opts.Provenance,
		}
		return m.PutDecision(created)
	})
	if err != nil {
		return domain.Decision{}, err
	}
	return created, nil
}

// RecordKnowledgeOptions describes a knowledge reference registration.
type RecordKnowledgeOptions struct {
	KnowledgeID  domain.KnowledgeID
	Title        string
	Summary      string
	RelativePath string
	ItemID       domain.ScopedItemID
	Provenance   *domain.Provenance
}

// RecordKnowledge registers a reusable knowledge reference.
func (s *Service) RecordKnowledge(txID string, opts RecordKnowledgeOptions) (domain.Knowledge, error) {
	var created domain.Knowledge
	err := s.mutate(txID, func(l domain.Ledger, m *store.Mutation) error {
		if err := domain.ValidateKnowledgeID(opts.KnowledgeID); err != nil {
			return err
		}
		if opts.Title == "" {
			return fmt.Errorf("knowledge requires a non-empty title")
		}
		if opts.RelativePath != "" {
			if err := domain.ValidateRelativePath(opts.RelativePath); err != nil {
				return err
			}
		}
		if opts.ItemID != "" {
			if _, ok := l.ItemByScoped(opts.ItemID); !ok {
				return fmt.Errorf("%w: %s", ErrNotFound, opts.ItemID.String())
			}
		}
		created = domain.Knowledge{
			SchemaVersion: domain.LedgerSchemaVersion,
			KnowledgeID:   opts.KnowledgeID,
			Title:         opts.Title,
			Summary:       opts.Summary,
			RelativePath:  opts.RelativePath,
			ItemID:        opts.ItemID,
			Revision:      1,
			Provenance:    opts.Provenance,
		}
		return m.PutKnowledge(created)
	})
	if err != nil {
		return domain.Knowledge{}, err
	}
	return created, nil
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
