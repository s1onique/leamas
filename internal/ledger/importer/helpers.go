package importer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// groupConflicts reports whether the records of one source identity group
// disagree with each other.
//
// Only a genuine disagreement is a conflict. Identical repetitions of the
// same record are a benign duplication and do not quarantine the group.
//
// The comparison covers every conserved SEMANTIC field: the title, the
// exact source status, the priority, the owner, the workstream, the
// criteria, the dependencies and the source confidence.
//
// The source LOCATOR (the row text, the line number and the free-form
// detail) is deliberately EXCLUDED: the same historical record may be
// reported by several source sections, each with its own locator. A
// difference of locator alone is a DUPLICATE REPRESENTATION, not a
// conflict. Excluding the locator is what keeps a repeated historical
// representation from being mistaken for a conflicting one.
func groupConflicts(group []SourceItem) bool {
	if len(group) < 2 {
		return false
	}
	first := group[0]
	for _, other := range group[1:] {
		if other.Title != first.Title ||
			other.EpicID != first.EpicID ||
			other.Status != first.Status ||
			other.Priority != first.Priority ||
			other.Owner != first.Owner ||
			other.Workstream != first.Workstream ||
			other.SourceConfidence != first.SourceConfidence ||
			!equalStringSlices(other.AcceptanceCriteria, first.AcceptanceCriteria) ||
			!equalStringSlices(other.ExitCriteria, first.ExitCriteria) ||
			!equalStringSlices(other.DependsOn, first.DependsOn) {
			return true
		}
	}
	return false
}

// sameACTContent reports whether an indexed historical ACT record carries
// the same content as the preserved canonical record.
//
// The comparison covers every conserved SEMANTIC field of the historical
// index entry: the original verdict, the evidence locators, the source
// references and the reconciliation state.
func sameACTContent(a, b domain.ActExecution) bool {
	if a.OriginalVerdict != b.OriginalVerdict ||
		a.Reconciliation != b.Reconciliation ||
		a.Archived != b.Archived ||
		!equalStringSlices(a.EvidenceLocators, b.EvidenceLocators) {
		return false
	}
	if len(a.SourceRefs) != len(b.SourceRefs) {
		return false
	}
	for i := range a.SourceRefs {
		if a.SourceRefs[i] != b.SourceRefs[i] {
			return false
		}
	}
	if len(a.ItemIDs) != len(b.ItemIDs) {
		return false
	}
	for i := range a.ItemIDs {
		if a.ItemIDs[i] != b.ItemIDs[i] {
			return false
		}
	}
	if len(a.UnresolvedItemIDs) != len(b.UnresolvedItemIDs) {
		return false
	}
	for i := range a.UnresolvedItemIDs {
		if a.UnresolvedItemIDs[i] != b.UnresolvedItemIDs[i] {
			return false
		}
	}
	return true
}

// equalStringSlices reports whether two string slices are identical.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sortDependencies orders dependency edges deterministically.
func sortDependencies(deps []domain.Dependency) {
	sort.SliceStable(deps, func(i, j int) bool {
		if deps[i].Kind != deps[j].Kind {
			return deps[i].Kind < deps[j].Kind
		}
		return deps[i].Ref < deps[j].Ref
	})
}

// Summary renders a deterministic human-readable summary of a plan.
func (p Plan) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "epics: %d\n", len(p.Epics))
	fmt.Fprintf(&b, "items: %d\n", len(p.Items))
	fmt.Fprintf(&b, "skipped: %d\n", len(p.Skipped))
	fmt.Fprintf(&b, "ambiguities: %d\n", len(p.Ambiguities))
	fmt.Fprintf(&b, "unresolved references: %d\n", len(p.Unresolved))
	return b.String()
}

// scopedKey returns the scoped identity key of a source record.
//
// The scoped key is "<EPIC-ID>/<SOURCE-ID>": the same source identity in
// two different epics are two distinct records, not a conflict.
func scopedKey(item SourceItem) string {
	return item.EpicID + "/" + item.SourceID
}
