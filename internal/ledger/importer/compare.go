package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

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

// sourceFingerprint computes the immutable normalized fingerprint of a
// source record.
//
// The fingerprint covers the normalized source CONTENT AND the normalized
// source RELATIONSHIP GRAPH. The relationships are normalized BEFORE the
// comparison: a source dependency that names a local identity is expanded
// to its scoped identity, and the expanded set is sorted and deduplicated.
// A source record whose dependency graph changed therefore produces a
// different fingerprint even when no textual field changed.
//
// The fingerprint describes the SOURCE, never the canonical record. A
// legitimate canonical evolution does not change it: only a changed
// historical source does.
func sourceFingerprint(item SourceItem) string {
	parts := []string{
		"source_id=" + item.SourceID,
		"epic_id=" + item.EpicID,
		"workstream=" + item.Workstream,
		"title=" + item.Title,
		"status=" + string(item.Status),
		"priority=" + item.Priority,
		"owner=" + item.Owner,
		"detail=" + item.SourceDetail,
		"acceptance_criteria=" + strings.Join(item.AcceptanceCriteria, "\x1f"),
		"exit_criteria=" + strings.Join(item.ExitCriteria, "\x1f"),
		"depends_on=" + strings.Join(normalizedSourceRefs(item), "\x1f"),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1e")))
	return hex.EncodeToString(sum[:])
}

// normalizedSourceRefs returns the normalized source dependency references
// of a source record.
//
// A reference that names a local identity is expanded to its scoped
// identity. The expanded set is sorted and deduplicated so the order of
// the source rows never changes the fingerprint.
func normalizedSourceRefs(item SourceItem) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, ref := range item.DependsOn {
		normalized := ref
		if !strings.Contains(normalized, "/") && item.EpicID != "" {
			normalized = item.EpicID + "/" + normalized
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

// sameSourceContent reports whether the preserved canonical record carries
// the same normalized source content as the incoming record.
//
// It compares the SOURCE-PRESERVED fields: the title, the exact source
// status, the priority, the workstream, the owner, the criteria and the
// preserved row provenance. The canonical lifecycle and acceptance state
// are NOT part of the comparison: they are the operator's decisions, not
// the source's content.
//
// The comparison is a FALLBACK for canonical records that carry no source
// fingerprint. A record that carries a fingerprint is compared by the
// fingerprint, which also covers the normalized source relationship graph.
func sameSourceContent(prior domain.WorkItem, incoming domain.WorkItem) bool {
	// The fingerprint is the PRIMARY comparison when the canonical record
	// carries one: it covers the normalized source content AND the
	// normalized source relationship graph. A dependency-only change of
	// the source is therefore detected, while a legitimate canonical
	// evolution of the record never is.
	if prior.SourceFingerprint != "" && incoming.SourceFingerprint != "" {
		return prior.SourceFingerprint == incoming.SourceFingerprint
	}
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
