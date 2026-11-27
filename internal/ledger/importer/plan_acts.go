package importer

import (
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// buildHistoricalActs builds the indexed historical ACT execution records.
//
// A historical execution attempt is NOT a work-item obligation. It is
// indexed as an ARCHIVED record with its ORIGINAL disposition: it stays
// discoverable without ever re-entering the live execution history and
// without asserting a canonical terminal verdict of its own.
//
// An EXACT REPLAY produces no canonical delta: an identity that the
// canonical state already carries is reported as unchanged, and a changed
// historical record is reported as an explicit reconciliation.
func buildHistoricalActs(p *Plan, source SourceBoard, existingByAct map[string]domain.ActExecution,
	imported map[string]bool, known map[string]bool, sourceName string) {
	for _, act := range source.Acts {
		if act.SourceID == "" {
			p.Ambiguities = append(p.Ambiguities, Ambiguity{
				SourceID: act.SourceID,
				Kind:     "missing_source_id",
				Detail:   "the historical ACT record has no source identity",
			})
			continue
		}
		record := domain.ActExecution{
			SchemaVersion: domain.LedgerSchemaVersion,
			ActID:         domain.ActID(act.SourceID),
			Title:         act.SourceID,
			// The canonical verdict of an INDEXED historical attempt is
			// pending: indexing a historical report neither re-opens it
			// nor asserts a terminal verdict of its own.
			Verdict:          domain.ActVerdictPending,
			Revision:         1,
			Archived:         true,
			OriginalVerdict:  act.OriginalVerdict,
			EvidenceLocators: act.EvidenceLocators,
			SourceRefs:       act.SourceRefs,
			Reconciliation:   act.Reconciliation,
			Provenance: &domain.Provenance{
				Source:     sourceName,
				SourceID:   act.SourceID,
				Confidence: string(domain.ProvenanceImported),
			},
		}
		for _, id := range act.WorkItemIDs {
			target := id
			if !strings.Contains(target, "/") && act.EpicID != "" {
				target = act.EpicID + "/" + target
			}
			if imported[target] || known[target] {
				record.ItemIDs = append(record.ItemIDs, domain.ScopedItemID(target))
				continue
			}
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID: act.SourceID,
				Ref:      id,
				Kind:     "historical_act_work_item",
			})
			// The UNRESOLVED association is conserved EXPLICITLY on
			// the historical record: an association the historical
			// index carried is never dropped because its target is
			// unknown.
			record.UnresolvedItemIDs = append(record.UnresolvedItemIDs, id)
		}
		if prior, exists := existingByAct[act.SourceID]; exists {
			if sameACTContent(prior, record) {
				p.Reconciliations = append(p.Reconciliations, Reconciliation{
					ScopedID: act.SourceID,
					Outcome:  ReconciliationUnchanged,
					Detail:   "the replay produced no canonical delta",
				})
			} else {
				p.Reconciliations = append(p.Reconciliations, Reconciliation{
					ScopedID: act.SourceID,
					Outcome:  ReconciliationSourceChanged,
					Detail: "the indexed historical record changed; the canonical record is conserved " +
						"and the change requires explicit reconciliation",
				})
			}
			p.Skipped = append(p.Skipped, act.SourceID)
			continue
		}
		p.Acts = append(p.Acts, record)
	}
}
