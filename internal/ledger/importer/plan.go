package importer

import (
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// BuildPlan builds the deterministic import plan.
//
// The plan never mutates anything: it is the dry-run result.
//
// CONSERVATION. Every source identity is accounted for in exactly one
// reconciliation record. The plan compares the INCOMING NORMALIZED SOURCE
// RECORD with the PRESERVED CANONICAL RECORD — its identity AND its
// content — BEFORE it filters an existing identity out. The comparison
// distinguishes:
//
//	created            the identity did not exist in the canonical state
//	unchanged          the replay produced no canonical delta
//	source_changed     the source record changed; the canonical record is
//	                   NOT overwritten, the change is reported
//	conflict           the source records of one identity disagree
//	unresolved         the identity, status or dependency is unresolved
//
// A changed historical source NEVER overwrites the canonical work: the
// change is an explicit reconciliation, not a silent mutation.
//
// UNKNOWN STATUSES. A source status the importer does not understand is
// conserved verbatim and the imported record stays DURABLY NON-EXECUTABLE
// until it is explicitly reconciled.
func BuildPlan(source SourceBoard, existing domain.Ledger) Plan {
	var p Plan

	confidence := domain.ProvenanceConversational
	sourceName := source.Source
	if sourceName == "" {
		sourceName = "unknown-source"
	}

	known := map[string]bool{}
	existingByScoped := map[string]domain.WorkItem{}
	for _, it := range existing.Items {
		known[it.Scoped().String()] = true
		existingByScoped[it.Scoped().String()] = it
	}

	// Resolve the source identity of every record. A source identity that
	// appears more than once is an explicit ambiguity: the COMPLETE identity
	// group is quarantined and NO record of that group is imported. Input
	// order must never select a winner.
	//
	// The identity group is keyed by the SCOPED identity: the same source
	// identity in two different epics are two different records, not a
	// conflict.
	grouped := map[string][]SourceItem{}
	var groupOrder []string
	for _, item := range source.Items {
		if item.SourceID == "" {
			p.Ambiguities = append(p.Ambiguities, Ambiguity{
				SourceID: item.SourceID,
				Kind:     "missing_source_id",
				Detail:   "source record has no source identity",
			})
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID: item.SourceID,
				Ref:      item.EpicID,
				Kind:     "missing_source_id",
			})
			continue
		}
		key := scopedKey(item)
		if _, seen := grouped[key]; !seen {
			groupOrder = append(groupOrder, key)
		}
		grouped[key] = append(grouped[key], item)
	}

	// A duplicated source identity is a conflict only when the records
	// actually disagree. Identical duplicates are a benign repetition and
	// import exactly once.
	conflicting := map[string]bool{}
	for _, key := range groupOrder {
		group := grouped[key]
		if len(group) < 2 {
			continue
		}
		if !groupConflicts(group) {
			continue
		}
		conflicting[key] = true
		records := make([]string, 0, len(group))
		for _, rec := range group {
			records = append(records, rec.Title)
		}
		p.Quarantined = append(p.Quarantined, Quarantine{
			SourceID: key,
			Reason:   "conflicting source records share one scoped source identity",
			Records:  records,
		})
		p.Ambiguities = append(p.Ambiguities, Ambiguity{
			SourceID: key,
			Kind:     "duplicate_source_id",
			Detail:   "source identity appears more than once with conflicting content",
		})
		p.Reconciliations = append(p.Reconciliations, Reconciliation{
			ScopedID: key,
			Outcome:  ReconciliationConflict,
			Detail:   "the source records of this identity disagree; the identity group is quarantined",
		})
	}

	// Build the epic set from the observed workstreams.
	//
	// An epic that already exists in the canonical state is NOT
	// overwritten: the existing metadata wins and the import does not
	// plan a write for it.
	existingEpics := map[domain.EpicID]bool{}
	for _, e := range existing.Epics {
		existingEpics[e.EpicID] = true
	}
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
		if existingEpics[domain.EpicID(id)] {
			// The epic already exists. Its canonical metadata is not
			// silently overwritten.
			p.SkippedEpics = append(p.SkippedEpics, id)
			continue
		}
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
		key := scopedKey(item)
		if conflicting[key] {
			// The complete identity group is quarantined. No record of
			// it acquires authority.
			continue
		}
		if importedIDs[key] {
			// A benign duplicate (identical content) is imported once.
			continue
		}
		importedIDs[key] = true
		scoped := domain.NewScopedItemID(domain.EpicID(item.EpicID), domain.ItemID(item.SourceID))

		itemConfidence := item.SourceConfidence
		if itemConfidence == "" {
			itemConfidence = confidence
		}

		// The status is resolved BEFORE the existing-identity filter. An
		// unknown status is a durable non-executable state, not an
		// identity that may be skipped silently.
		acceptance := domain.AcceptancePending
		lifecycle := domain.LifecycleOpen
		knownStatus := IsKnownSourceStatus(item.Status)
		switch NormalizeSourceStatus(item.Status) {
		case SourceStatusClosed, SourceStatusDone:
			lifecycle = domain.LifecycleClosed
		case SourceStatusClosedInsufficient:
			// A CLOSED_INSUFFICIENT record is a DISPOSITION, not a
			// completion: the investigation was closed without
			// establishing the obligation. The disposition is conserved
			// verbatim, but the work stays unfinished and is never
			// accepted.
			lifecycle = domain.LifecycleOpen
		case SourceStatusBlocked:
			lifecycle = domain.LifecycleBlocked
		case SourceStatusInProgress:
			lifecycle = domain.LifecycleInProgress
		case SourceStatusDeferred:
			lifecycle = domain.LifecycleOpen
			acceptance = domain.AcceptanceDeferred
		case SourceStatusUnfinished, SourceStatusPartial, SourceStatusQueued,
			SourceStatusInvestigate, SourceStatusHypothesis,
			SourceStatusLiveWitness, SourceStatusFinalPreflight:
			lifecycle = domain.LifecycleOpen
		default:
			// An UNKNOWN source status remains explicitly unresolved. It
			// is never coerced into a known lifecycle state: the record
			// keeps the exact source status and is reported as
			// unresolved so it cannot silently acquire authority.
			lifecycle = domain.LifecycleOpen
			acceptance = domain.AcceptanceDeferred
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID: item.SourceID,
				Ref:      string(item.Status),
				Kind:     "unknown_source_status",
			})
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
			// The exact source status is conserved verbatim, and the
			// original row provenance is preserved.
			SourceStatus: string(item.Status),
			SourceRow:    sourceRowOf(item),
			SourceLine:   item.SourceLine,
			SourceDetail: item.SourceDetail,
			Provenance: &domain.Provenance{
				Source:     sourceName,
				SourceID:   item.SourceID,
				Confidence: string(itemConfidence),
			},
		}
		if lifecycle == domain.LifecycleClosed {
			record.TerminalDisposition = string(item.Status)
		}
		if NormalizeSourceStatus(item.Status) == SourceStatusClosedInsufficient {
			// The disposition is conserved verbatim even though the work
			// itself stays unfinished.
			record.TerminalDisposition = string(item.Status)
		}

		// CONSERVATION COMPARISON. The incoming normalized source record is
		// compared with the PRESERVED canonical record — identity AND
		// content — before the existing identity is filtered out.
		prior, exists := existingByScoped[scoped.String()]
		switch {
		case !exists:
			p.Reconciliations = append(p.Reconciliations, Reconciliation{
				ScopedID: scoped.String(),
				Outcome:  ReconciliationCreated,
				Detail:   "the identity did not exist in the canonical state",
			})
		case sameSourceContent(prior, record):
			// EXACT REPLAY: the source produced no canonical delta.
			p.Reconciliations = append(p.Reconciliations, Reconciliation{
				ScopedID: scoped.String(),
				Outcome:  ReconciliationUnchanged,
				Detail:   "the replay produced no canonical delta",
			})
			p.Skipped = append(p.Skipped, scoped.String())
			continue
		case !knownStatus:
			// An UNKNOWN status can never be applied to an existing
			// identity: the record stays as it is and the unresolved
			// status is reported.
			p.Reconciliations = append(p.Reconciliations, Reconciliation{
				ScopedID: scoped.String(),
				Outcome:  ReconciliationUnresolved,
				Detail:   "the source status is unknown; the canonical record is conserved unchanged",
			})
			p.Skipped = append(p.Skipped, scoped.String())
			continue
		default:
			// CHANGED HISTORICAL SOURCE. The canonical work is NOT
			// overwritten merely because the historical source changed.
			// The change is an explicit reconciliation.
			p.Reconciliations = append(p.Reconciliations, Reconciliation{
				ScopedID: scoped.String(),
				Outcome:  ReconciliationSourceChanged,
				Detail:   "the source record changed; the canonical record is conserved and the change requires explicit reconciliation",
			})
			p.Skipped = append(p.Skipped, scoped.String())
			continue
		}
		p.Items = append(p.Items, record)
	}

	// Resolve dependencies against the imported identity set.
	//
	// Every resolved dependency becomes a canonical typed edge on the
	// importing work item. Every UNRESOLVED blocking dependency becomes a
	// durable blocker: it is recorded as a typed edge that points at an
	// identity that does not exist yet, so the importing item can never
	// become executable while the blocker is outstanding.
	imported := map[string]bool{}
	for _, item := range p.Items {
		imported[item.Scoped().String()] = true
	}
	itemIndex := map[string]int{}
	for i, item := range p.Items {
		itemIndex[item.Scoped().String()] = i
	}

	for _, item := range source.Items {
		if item.SourceID == "" || item.EpicID == "" {
			continue
		}
		if conflicting[scopedKey(item)] {
			continue
		}
		scoped := domain.NewScopedItemID(domain.EpicID(item.EpicID), domain.ItemID(item.SourceID))
		idx, ok := itemIndex[scoped.String()]
		if !ok {
			continue
		}
		for _, ref := range item.DependsOn {
			target := ref
			if !strings.Contains(target, "/") && item.EpicID != "" {
				target = item.EpicID + "/" + ref
			}
			if imported[target] || known[target] {
				// The dependency resolved: it becomes a canonical typed
				// edge. The edge is conserved, never dropped.
				p.Items[idx].Dependencies = append(p.Items[idx].Dependencies,
					domain.Dependency{
						Kind: domain.DependencyRequires,
						Ref:  domain.ScopedItemID(target),
					})
				continue
			}
			// The dependency did not resolve. It is a durable blocker:
			// the typed edge is recorded so the item stays
			// non-executable until the blocker is resolved.
			p.Items[idx].Dependencies = append(p.Items[idx].Dependencies,
				domain.Dependency{
					Kind: domain.DependencyBlocker,
					Ref:  domain.ScopedItemID(target),
					Note: "unresolved source dependency",
				})
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID: item.SourceID,
				Ref:      ref,
				Kind:     "dependency",
			})
		}
	}

	// Sort the dependency edges of every imported record deterministically.
	for i := range p.Items {
		sortDependencies(p.Items[i].Dependencies)
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
	sort.Slice(p.Reconciliations, func(i, j int) bool {
		return p.Reconciliations[i].ScopedID < p.Reconciliations[j].ScopedID
	})
	return p
}
