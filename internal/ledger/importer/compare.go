package importer

import (
	"github.com/s1onique/leamas/internal/ledger/domain"
)

// sourceRowOf returns the preserved row provenance of a source record.
//
// The original row text is preserved verbatim when the source carried one.
// The source locator is preserved as a fallback, so the provenance of the
// row is never lost.
func sourceRowOf(item SourceItem) string {
	if item.SourceRow != "" {
		return item.SourceRow
	}
	return item.SourceID
}

// sameSourceContent reports whether the preserved canonical record carries
// the same normalized source content as the incoming record.
//
// It compares the SOURCE-PRESERVED fields: the title, the exact source
// status, the priority, the workstream, the owner, the criteria and the
// preserved row provenance. The canonical lifecycle and acceptance state
// are NOT part of the comparison: they are the operator's decisions, not
// the source's content.
func sameSourceContent(prior domain.WorkItem, incoming domain.WorkItem) bool {
	if prior.Title != incoming.Title {
		return false
	}
	if prior.SourceStatus != incoming.SourceStatus {
		return false
	}
	if prior.Priority != incoming.Priority {
		return false
	}
	if prior.EpicID != incoming.EpicID || prior.ItemID != incoming.ItemID {
		return false
	}
	if prior.Owner != incoming.Owner {
		return false
	}
	if prior.SourceRow != incoming.SourceRow {
		return false
	}
	if prior.SourceLine != incoming.SourceLine {
		return false
	}
	if prior.SourceDetail != incoming.SourceDetail {
		return false
	}
	if !equalStringSlices(prior.AcceptanceCriteria, incoming.AcceptanceCriteria) {
		return false
	}
	if !equalStringSlices(prior.ExitCriteria, incoming.ExitCriteria) {
		return false
	}
	return true
}
