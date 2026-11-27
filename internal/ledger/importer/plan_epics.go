package importer

import (
	"sort"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// buildEpics builds the epic set from the DECLARED epic records of the
// source.
//
// The epic INVENTORY comes from the declared scopes, not from the observed
// work items: a source epic that carries no outstanding work item is still a
// scope of the board and must stay discoverable.
//
// An epic that already exists in the canonical state is NOT overwritten:
// the existing metadata wins and the import does not plan a write for it.
// A CHANGED epic source is detected through the immutable source
// fingerprint and reported as an explicit reconciliation: a legitimate
// canonical evolution of the scope never changes the fingerprint, a changed
// historical source always does.
func buildEpics(p *Plan, source SourceBoard, existing domain.Ledger, sourceName string) {
	existingEpics := map[domain.EpicID]bool{}
	for _, e := range existing.Epics {
		existingEpics[e.EpicID] = true
	}
	declared := declaredEpics(source)
	for _, item := range source.Items {
		if item.EpicID != "" {
			continue
		}
		p.Ambiguities = append(p.Ambiguities, Ambiguity{
			SourceID: item.SourceID,
			Kind:     "missing_workstream",
			Detail:   "source record has no workstream identity",
		})
	}
	epicNames := make([]string, 0, len(declared))
	for id := range declared {
		epicNames = append(epicNames, id)
	}
	sort.Strings(epicNames)
	for _, id := range epicNames {
		meta := declared[id]
		if existingEpics[domain.EpicID(id)] {
			// The epic already exists. Its canonical metadata is not
			// silently overwritten.
			p.SkippedEpics = append(p.SkippedEpics, id)
			// The exit criteria of an EXISTING epic are never lost
			// silently: a scope that declares criteria the canonical
			// epic does not carry is reported explicitly.
			criteria := epicExitCriteriaOf(source, id)
			if len(criteria) > 0 && !epicCarriesCriteria(existing, domain.EpicID(id), criteria) {
				p.Ambiguities = append(p.Ambiguities, Ambiguity{
					SourceID: id,
					Kind:     "epic_exit_criteria_not_applied",
					Detail: "the canonical epic does not carry the source exit criteria; " +
						"they require an explicit reconciliation",
				})
			}
			// A CHANGED epic source is an explicit reconciliation. The
			// canonical scope is conserved unchanged.
			if prior, ok := existingEpic(existing, domain.EpicID(id)); ok &&
				prior.SourceFingerprint != "" && meta.fingerprint != "" &&
				prior.SourceFingerprint != meta.fingerprint {
				p.Reconciliations = append(p.Reconciliations, Reconciliation{
					ScopedID: id,
					Outcome:  ReconciliationSourceChanged,
					Detail: "the source epic record changed; the canonical scope is conserved " +
						"and the change requires explicit reconciliation",
				})
			}
			continue
		}
		title := meta.title
		if title == "" {
			title = id
		}
		priority := domain.Priority(meta.priority)
		if !domain.IsValidPriority(priority) {
			priority = domain.PriorityP2
		}
		epic := domain.Epic{
			SchemaVersion: domain.LedgerSchemaVersion,
			EpicID:        domain.EpicID(id),
			Title:         title,
			// The EXACT original disposition of the scope is conserved
			// verbatim: it is historical fact, distinct from the
			// canonical state of the scope.
			Status:       meta.originalStatus,
			Priority:     priority,
			Revision:     1,
			ExitCriteria: epicExitCriteriaOf(source, id),
			// The immutable source fingerprint of the declared scope
			// makes a changed historical epic source detectable.
			SourceFingerprint: meta.fingerprint,
			Provenance: &domain.Provenance{
				Source:     sourceName,
				Confidence: string(domain.ProvenanceImported),
			},
		}
		if meta.lifecycle != "" {
			epic.SourceLifecycle = meta.lifecycle
		}
		if len(meta.sourceRefs) > 0 {
			epic.SourceRefs = meta.sourceRefs
		}
		p.Epics = append(p.Epics, epic)
	}
}

// declaredEpic is the declared metadata of one epic scope.
type declaredEpic struct {
	title          string
	lifecycle      string
	priority       string
	originalStatus string
	sourceRefs     []domain.SourceRef
	fingerprint    string
}

// declaredEpics returns the declared epic scopes of a source board.
//
// The DECLARED records are the inventory. A scope that the board only
// reveals through its work items is added as a bare scope: it carries no
// declared metadata, and the importer never invents one.
func declaredEpics(source SourceBoard) map[string]declaredEpic {
	out := map[string]declaredEpic{}
	for _, epic := range source.Epics {
		if epic.EpicID == "" {
			continue
		}
		out[epic.EpicID] = declaredEpic{
			title:          epic.Title,
			lifecycle:      epic.Lifecycle,
			priority:       epic.Priority,
			originalStatus: epic.OriginalStatus,
			sourceRefs:     epic.SourceRefs,
			fingerprint:    epicSourceFingerprint(epic),
		}
	}
	for _, item := range source.Items {
		if item.EpicID == "" {
			continue
		}
		if _, ok := out[item.EpicID]; ok {
			continue
		}
		// A scope the board reveals only through its work items keeps the
		// title and the metadata the board declared for it. The importer
		// never invents metadata it was not given.
		meta := declaredEpic{title: source.EpicTitles[item.EpicID]}
		if boardMeta, ok := source.EpicMeta[item.EpicID]; ok {
			meta.lifecycle = boardMeta.Lifecycle
			meta.sourceRefs = boardMeta.SourceRefs
		}
		out[item.EpicID] = meta
	}
	return out
}

// existingEpic returns the canonical epic record with the given identity.
func existingEpic(existing domain.Ledger, id domain.EpicID) (domain.Epic, bool) {
	for _, epic := range existing.Epics {
		if epic.EpicID == id {
			return epic, true
		}
	}
	return domain.Epic{}, false
}
