package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
		"description=" + item.Description,
		"status=" + string(item.Status),
		"priority=" + item.Priority,
		"owner=" + item.Owner,
		"detail=" + item.SourceDetail,
		"acceptance_criteria=" + strings.Join(item.AcceptanceCriteria, "\x1f"),
		"exit_criteria=" + strings.Join(item.ExitCriteria, "\x1f"),
		"depends_on=" + strings.Join(normalizedSourceRefs(item), "\x1f"),
		"relations=" + strings.Join(normalizedSourceRelations(item), "\x1f"),
		"source_lifecycle=" + item.SourceLifecycle,
		"source_qualification=" + item.SourceQualification,
		"criteria_coverage=" + item.CriteriaCoverage,
		"dependency_coverage=" + item.DependencyCoverage,
		"blockers=" + strings.Join(item.Blockers, "\x1f"),
		"current_act=" + item.CurrentACT,
		"related_acts=" + strings.Join(item.RelatedACTs, "\x1f"),
		"evidence_locators=" + strings.Join(item.EvidenceLocators, "\x1f"),
		"reopen_conditions=" + strings.Join(item.ReopenConditions, "\x1f"),
		"source_refs=" + strings.Join(normalizedSourceProvenance(item), "\x1f"),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1e")))
	return hex.EncodeToString(sum[:])
}

// epicSourceFingerprint computes the immutable normalized fingerprint of a
// declared source epic record.
//
// The fingerprint covers the declared metadata, the exit criteria and the
// byte-exact source provenance of the scope. It describes the SOURCE, never
// the canonical record: a legitimate canonical evolution of the scope never
// changes it, a changed historical source always does.
func epicSourceFingerprint(epic SourceEpic) string {
	parts := []string{
		"epic_id=" + epic.EpicID,
		"title=" + epic.Title,
		"lifecycle=" + epic.Lifecycle,
		"priority=" + epic.Priority,
		"original_status=" + epic.OriginalStatus,
		"exit_criteria=" + strings.Join(epic.ExitCriteria, "\x1f"),
	}
	for _, ref := range epic.SourceRefs {
		parts = append(parts, "source_ref="+fmt.Sprintf("%s:%d-%d:%s:%s",
			ref.Path, ref.StartLine, ref.EndLine, ref.SHA256, ref.SourceHead))
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
	for _, dep := range dependencyEdgesOf(item) {
		normalized := dep.Target
		if !strings.Contains(normalized, "/") && item.EpicID != "" {
			normalized = item.EpicID + "/" + normalized
		}
		key := normalized + "|" + dep.RequiredOutcome + "|" +
			strings.Join(normalizedRefProvenance(dep.SourceRefs), "\x1f")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// normalizedRefProvenance returns the normalized byte-exact provenance of a
// set of source references.
func normalizedRefProvenance(refs []domain.SourceRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, fmt.Sprintf("%s:%d-%d:%s:%s",
			ref.Path, ref.StartLine, ref.EndLine, ref.SHA256, ref.SourceHead))
	}
	sort.Strings(out)
	return out
}

// normalizedSourceRelations returns the normalized non-blocking relationship
// references of a source record.
//
// A reference that names a local identity is expanded to its scoped
// identity. The expanded set is sorted and deduplicated so the order of the
// source rows never changes the fingerprint.
func normalizedSourceRelations(item SourceItem) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, rel := range item.Relations {
		normalized := rel.Target
		if !strings.Contains(normalized, "/") && item.EpicID != "" {
			normalized = item.EpicID + "/" + normalized
		}
		key := rel.Kind + ":" + normalized + "|" + rel.RequiredOutcome + "|" +
			strings.Join(normalizedRefProvenance(rel.SourceRefs), "\x1f")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// normalizedSourceProvenance returns the normalized byte-exact source
// provenance of a source record.
//
// The EFFECTIVE source HEAD of every reference participates: a reference that
// names its own revision is a different binding than one that inherits the
// manifest head.
func normalizedSourceProvenance(item SourceItem) []string {
	out := make([]string, 0, len(item.SourceRefs))
	for _, ref := range item.SourceRefs {
		out = append(out, fmt.Sprintf("%s:%d-%d:%s:%s",
			ref.Path, ref.StartLine, ref.EndLine, ref.SHA256, ref.SourceHead))
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
