// Package importer implements the dry-run import and reconciliation
// primitive for existing Factory boards.
//
// The importer is deliberately conservative:
//
//   - it preserves the source record identity and its provenance;
//   - it reports ambiguities and unresolved references explicitly;
//   - it never promotes a conversational claim into a repository-verified
//     fact;
//   - it is idempotent: importing the same source twice produces no
//     unintended canonical delta.
package importer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// SourceStatus is a lifecycle status observed in a source board.
type SourceStatus string

// Source statuses recognised by the importer.
const (
	SourceStatusOpen               SourceStatus = "open"
	SourceStatusInProgress         SourceStatus = "in_progress"
	SourceStatusBlocked            SourceStatus = "blocked"
	SourceStatusClosed             SourceStatus = "closed"
	SourceStatusClosedInsufficient SourceStatus = "closed_insufficient"
	SourceStatusDone               SourceStatus = "done"
	SourceStatusUnfinished         SourceStatus = "unfinished"
)

// SourceItem is one source work-item record.
type SourceItem struct {
	// SourceID is the identity of the record in the source board. It is
	// preserved verbatim through the import.
	SourceID string `json:"source_id"`
	// EpicID is the source workstream the record belongs to.
	EpicID string `json:"epic_id"`
	// Title is the source title.
	Title string `json:"title"`
	// Status is the observed source status. It is conserved.
	Status SourceStatus `json:"status"`
	// Priority is the observed source priority.
	Priority string `json:"priority,omitempty"`
	// Owner is the observed source owner.
	Owner string `json:"owner,omitempty"`
	// AcceptanceCriteria are the source acceptance criteria.
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	// ExitCriteria are the source exit criteria.
	ExitCriteria []string `json:"exit_criteria,omitempty"`
	// DependsOn lists the source dependency references.
	DependsOn []string `json:"depends_on,omitempty"`
	// SourceConfidence records whether the source record is a
	// repository-verified fact or a conversational claim.
	SourceConfidence domain.ProvenanceConfidence `json:"source_confidence,omitempty"`
}

// SourceBoard is the parsed source board.
type SourceBoard struct {
	// Source identifies the board the records came from.
	Source string `json:"source"`
	// EpicTitles maps a workstream identity to its title.
	EpicTitles map[string]string `json:"epic_titles,omitempty"`
	// Items are the source work-item records.
	Items []SourceItem `json:"items"`
}

// Ambiguity is an unresolved source ambiguity.
type Ambiguity struct {
	// SourceID is the source record the ambiguity belongs to.
	SourceID string `json:"source_id"`
	// Kind classifies the ambiguity.
	Kind string `json:"kind"`
	// Detail explains the ambiguity.
	Detail string `json:"detail"`
}

// UnresolvedReference is a source reference that did not resolve.
type UnresolvedReference struct {
	// SourceID is the source record carrying the reference.
	SourceID string `json:"source_id"`
	// Ref is the unresolved reference.
	Ref string `json:"ref"`
	// Kind classifies the reference.
	Kind string `json:"kind"`
}

// Plan is the deterministic import plan.
type Plan struct {
	// Epics are the epic records to create.
	Epics []domain.Epic `json:"epics"`
	// Items are the work-item records to create.
	Items []domain.WorkItem `json:"items"`
	// Ambiguities are the explicit ambiguity reports.
	Ambiguities []Ambiguity `json:"ambiguities"`
	// Unresolved are the explicit unresolved-reference reports.
	Unresolved []UnresolvedReference `json:"unresolved"`
	// Skipped lists the source records that already exist and therefore
	// produce no canonical delta.
	Skipped []string `json:"skipped"`
}

// BuildPlan builds the deterministic import plan.
//
// The plan never mutates anything: it is the dry-run result.
func BuildPlan(source SourceBoard, existing domain.Ledger) Plan {
	var p Plan

	confidence := domain.ProvenanceConversational
	sourceName := source.Source
	if sourceName == "" {
		sourceName = "unknown-source"
	}

	known := map[string]bool{}
	for _, it := range existing.Items {
		known[it.Scoped().String()] = true
	}

	// Resolve the source identity of every record. A source identity that
	// appears more than once is an explicit ambiguity: it is NOT resolved
	// by picking a winner.
	resolved := map[string]bool{}
	for _, item := range source.Items {
		if item.SourceID == "" {
			p.Ambiguities = append(p.Ambiguities, Ambiguity{
				SourceID: item.SourceID,
				Kind:     "missing_source_id",
				Detail:   "source record has no source identity",
			})
			continue
		}
		if resolved[item.SourceID] {
			p.Ambiguities = append(p.Ambiguities, Ambiguity{
				SourceID: item.SourceID,
				Kind:     "duplicate_source_id",
				Detail:   "source identity appears more than once",
			})
			continue
		}
		resolved[item.SourceID] = true
	}

	// Build the epic set from the observed workstreams.
	epicIDs := map[string]bool{}
	for _, item := range source.Items {
		if item.EpicID == "" {
			p.Ambiguities = append(p.Ambiguities, Ambiguity{
				SourceID: item.SourceID,
				Kind:     "missing_workstream",
				Detail:   "source record has no workstream identity",
			})
			continue
		}
		epicIDs[item.EpicID] = true
	}
	epicNames := make([]string, 0, len(epicIDs))
	for id := range epicIDs {
		epicNames = append(epicNames, id)
	}
	sort.Strings(epicNames)
	for _, id := range epicNames {
		title := source.EpicTitles[id]
		if title == "" {
			title = id
		}
		p.Epics = append(p.Epics, domain.Epic{
			SchemaVersion: domain.LedgerSchemaVersion,
			EpicID:        domain.EpicID(id),
			Title:         title,
			Priority:      domain.PriorityP2,
			Revision:      1,
			Provenance: &domain.Provenance{
				Source:     sourceName,
				Confidence: string(domain.ProvenanceImported),
			},
		})
	}

	// Build the work-item records.
	importedIDs := map[string]bool{}
	for _, item := range source.Items {
		if item.SourceID == "" || item.EpicID == "" {
			continue
		}
		if importedIDs[item.SourceID] {
			// A duplicate source identity stays unresolved: it is
			// reported as an ambiguity and never imported twice.
			continue
		}
		importedIDs[item.SourceID] = true
		scoped := domain.NewScopedItemID(domain.EpicID(item.EpicID), domain.ItemID(item.SourceID))
		if known[scoped.String()] {
			p.Skipped = append(p.Skipped, scoped.String())
			continue
		}

		itemConfidence := item.SourceConfidence
		if itemConfidence == "" {
			itemConfidence = confidence
		}
		// A conversational claim is never promoted: the imported record
		// keeps the conversational confidence and never starts accepted.
		acceptance := domain.AcceptancePending
		lifecycle := domain.LifecycleOpen
		switch item.Status {
		case SourceStatusClosed, SourceStatusDone:
			lifecycle = domain.LifecycleClosed
		case SourceStatusClosedInsufficient:
			lifecycle = domain.LifecycleClosed
		case SourceStatusBlocked:
			lifecycle = domain.LifecycleBlocked
		case SourceStatusInProgress:
			lifecycle = domain.LifecycleInProgress
		case SourceStatusUnfinished:
			lifecycle = domain.LifecycleOpen
		}
		if item.Status == SourceStatusClosed || item.Status == SourceStatusDone {
			// A closed source record keeps its disposition, but the
			// acceptance remains unverified until evidence supports it.
			acceptance = domain.AcceptancePending
		}

		priority := domain.Priority(item.Priority)
		if !domain.IsValidPriority(priority) {
			priority = domain.PriorityP2
		}

		record := domain.WorkItem{
			SchemaVersion:      domain.LedgerSchemaVersion,
			EpicID:             domain.EpicID(item.EpicID),
			ItemID:             domain.ItemID(item.SourceID),
			Title:              item.Title,
			Kind:               domain.ItemKindTask,
			Lifecycle:          lifecycle,
			Acceptance:         acceptance,
			Priority:           priority,
			Quality:            domain.EvidenceQualityNone,
			Owner:              item.Owner,
			Revision:           1,
			AcceptanceCriteria: item.AcceptanceCriteria,
			ExitCriteria:       item.ExitCriteria,
			Provenance: &domain.Provenance{
				Source:     sourceName,
				SourceID:   item.SourceID,
				Confidence: string(itemConfidence),
			},
		}
		if lifecycle == domain.LifecycleClosed {
			record.TerminalDisposition = string(item.Status)
		}
		p.Items = append(p.Items, record)
	}

	// Resolve dependencies against the imported identity set.
	imported := map[string]bool{}
	for _, item := range p.Items {
		imported[item.Scoped().String()] = true
	}
	for _, item := range source.Items {
		if item.SourceID == "" {
			continue
		}
		for _, ref := range item.DependsOn {
			target := ref
			if !strings.Contains(target, "/") && item.EpicID != "" {
				target = item.EpicID + "/" + ref
			}
			if imported[target] || known[target] {
				continue
			}
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID: item.SourceID,
				Ref:      ref,
				Kind:     "dependency",
			})
		}
	}

	sort.Slice(p.Items, func(i, j int) bool {
		return p.Items[i].Scoped() < p.Items[j].Scoped()
	})
	sort.Strings(p.Skipped)
	sort.Slice(p.Ambiguities, func(i, j int) bool {
		if p.Ambiguities[i].SourceID != p.Ambiguities[j].SourceID {
			return p.Ambiguities[i].SourceID < p.Ambiguities[j].SourceID
		}
		return p.Ambiguities[i].Kind < p.Ambiguities[j].Kind
	})
	sort.Slice(p.Unresolved, func(i, j int) bool {
		if p.Unresolved[i].SourceID != p.Unresolved[j].SourceID {
			return p.Unresolved[i].SourceID < p.Unresolved[j].SourceID
		}
		return p.Unresolved[i].Ref < p.Unresolved[j].Ref
	})
	return p
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
