package fwmi

import (
	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/importer"
)

// adaptItem converts one reconciled work item into a source record.
//
// The lifecycle mapping is EXPLICIT: an UNRESOLVED source lifecycle becomes
// an unknown source status, which the importer keeps durably non-executable
// and distinctly reported. It is never silently a deferral.
func adaptItem(epic EpicRecord, item WorkItem, sourceHead string) (importer.SourceItem, int, int) {
	src := importer.SourceItem{
		SourceID:            item.ItemID,
		EpicID:              epic.EpicID,
		Title:               item.Title,
		Description:         item.Description,
		Status:              sourceStatusOf(item),
		Priority:            derefString(item.Priority),
		AcceptanceCriteria:  append([]string{}, item.Criteria...),
		SourceConfidence:    domain.ProvenanceImported,
		SourceLifecycle:     item.Lifecycle,
		SourceQualification: item.Qualification,
		CriteriaCoverage:    item.CriteriaCoverage,
		DependencyCoverage:  item.DependencyCoverage,
		SourceRefs:          convertRefsWithHead(item.SourceRefs, sourceHead),
	}
	if item.Workstream != nil {
		src.Workstream = *item.Workstream
	}
	if item.OriginalStatus != nil {
		// The exact source status is conserved verbatim.
		src.SourceRow = *item.OriginalStatus
	}
	// The EXPLICIT blockers, the CURRENT and RELATED ACT associations, the
	// historical evidence locators and the reopen conditions are conserved
	// verbatim: they are load-bearing facts about the obligation.
	src.Blockers = append([]string{}, item.Blockers...)
	src.CurrentACT = derefString(item.CurrentACT)
	src.RelatedACTs = append([]string{}, item.RelatedACTs...)
	src.EvidenceLocators = append([]string{}, item.EvidenceLocators...)
	src.ReopenConditions = append([]string{}, item.ReopenConditions...)

	deps := 0
	relations := 0
	for _, rel := range item.Dependencies {
		edge := importer.SourceRelation{
			Kind:            rel.Kind,
			Target:          rel.Target,
			RequiredOutcome: rel.RequiredOutcome,
			SourceRefs:      convertRefsWithHead(rel.SourceRefs, sourceHead),
		}
		if rel.Kind == RelDependsOn {
			src.DependsOn = append(src.DependsOn, rel.Target)
			// The BLOCKING edge is conserved with its full
			// load-bearing content: the outcome it requires and the
			// byte-exact source locations it was derived from.
			src.DependsOnRelations = append(src.DependsOnRelations, edge)
			deps++
			continue
		}
		src.Relations = append(src.Relations, edge)
		relations++
	}
	return src, deps, relations
}

// sourceStatusOf returns the source status of an item.
//
// An UNRESOLVED lifecycle yields an unknown status: the importer keeps such
// a record durably non-executable and reports it as unresolved. An
// intentional DEFERRED lifecycle yields the known deferral status, so the
// two remain distinguishable in the canonical state.
func sourceStatusOf(item WorkItem) importer.SourceStatus {
	switch item.Lifecycle {
	case LifecycleOpen:
		return importer.SourceStatusOpen
	case LifecycleActive:
		return importer.SourceStatusInProgress
	case LifecycleBlocked:
		return importer.SourceStatusBlocked
	case LifecycleDeferred:
		return importer.SourceStatusDeferred
	case LifecycleTerminal:
		return importer.SourceStatusClosed
	default:
		return importer.SourceStatus("unresolved")
	}
}

// convertRefs converts the interchange source references.
//
// The EFFECTIVE source HEAD of every reference is retained: a reference that
// names no revision of its own is bound to the source HEAD of the package,
// and a reference that names one keeps it. A persisted reference therefore
// always resolves against a concrete source revision.
func convertRefs(refs []SourceRef) []domain.SourceRef {
	return convertRefsWithHead(refs, "")
}

// convertRefsWithHead converts the interchange source references, binding
// every reference that names no revision of its own to the given source
// HEAD.
func convertRefsWithHead(refs []SourceRef, head string) []domain.SourceRef {
	if len(refs) == 0 {
		return nil
	}
	out := make([]domain.SourceRef, 0, len(refs))
	for _, ref := range refs {
		effective := ref.SourceHead
		if effective == "" {
			effective = head
		}
		out = append(out, domain.SourceRef{
			Path:       ref.Path,
			StartLine:  ref.StartLine,
			EndLine:    ref.EndLine,
			SHA256:     ref.SHA256,
			SourceHead: effective,
		})
	}
	return out
}

// derefString returns the value of an optional scalar.
func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
