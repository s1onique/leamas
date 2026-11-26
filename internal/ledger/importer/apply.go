package importer

import (
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// ApplyResult reports the outcome of an applied import.
type ApplyResult struct {
	// CreatedEpics is the number of epic records created.
	CreatedEpics int `json:"created_epics"`
	// CreatedItems is the number of work-item records created.
	CreatedItems int `json:"created_items"`
	// Skipped is the number of source records that already existed.
	Skipped int `json:"skipped"`
	// Ambiguities is the number of ambiguity reports.
	Ambiguities int `json:"ambiguities"`
	// Unresolved is the number of unresolved-reference reports.
	Unresolved int `json:"unresolved"`
}

// Receipt is the conservation receipt of an import.
//
// It reports what the import conserved: the nodes, the relationships, the
// blockers, the original dispositions, the criteria and the reconciliation
// records. The receipt is the evidence that the import did not lose
// obligations.
//
// An imported work-item COUNT is not a conservation proof: the receipt
// reports the semantic content that was carried over.
type Receipt struct {
	// Nodes is the number of imported work-item records.
	Nodes int `json:"nodes"`
	// Epics is the number of imported epic records.
	Epics int `json:"epics"`
	// Relationships is the number of conserved dependency edges.
	Relationships int `json:"relationships"`
	// Blockers is the number of durable unresolved blockers.
	Blockers int `json:"blockers"`
	// OriginalDispositions is the number of records whose exact source
	// status was conserved.
	OriginalDispositions int `json:"original_dispositions"`
	// Criteria is the number of conserved acceptance criteria.
	Criteria int `json:"criteria"`
	// ExitCriteria is the number of conserved exit criteria.
	ExitCriteria int `json:"exit_criteria"`
	// Reconciliations lists the explicit reconciliation records.
	Reconciliations []Reconciliation `json:"reconciliations"`
	// Quarantined lists the quarantined identity groups.
	Quarantined []Quarantine `json:"quarantined"`
}

// BuildReceipt computes the conservation receipt of an applied import.
//
// The receipt is computed against the state BEFORE the import and the plan
// that was applied, so it reports the conservation rather than restating
// the plan.
//
// The receipt compares the ORIGINAL SOURCE, the NORMALIZED RECORDS and the
// ACTUAL PERSISTED STATE. A planned-record count is not a conservation
// proof: the receipt reports the semantic content that was carried over.
func BuildReceipt(before domain.Ledger, p Plan) Receipt {
	r := Receipt{
		Nodes:           len(p.Items),
		Epics:           len(p.Epics),
		Reconciliations: []Reconciliation{},
		Quarantined:     p.Quarantined,
	}
	if r.Quarantined == nil {
		r.Quarantined = []Quarantine{}
	}
	for _, item := range p.Items {
		r.OriginalDispositions++
		r.Criteria += len(item.AcceptanceCriteria)
		r.ExitCriteria += len(item.ExitCriteria)
		for _, dep := range item.Dependencies {
			if dep.Kind == domain.DependencyBlocker {
				r.Blockers++
				continue
			}
			r.Relationships++
		}
	}
	// The reconciliation records are the plan's own explicit outcomes. The
	// plan compares the incoming normalized source record with the
	// preserved canonical record, so the receipt restates a comparison
	// that was actually made, not a count.
	r.Reconciliations = append(r.Reconciliations, p.Reconciliations...)
	return r
}

// Apply publishes an import plan into the canonical ledger.
//
// The import is idempotent: a source record that already exists is
// skipped, so replaying an import produces no canonical delta.
//
// The import FAILS CLOSED on unresolved load-bearing ambiguity: a
// quarantined identity group or an unresolved blocking dependency prevents
// the authoritative import unless the ambiguity is durably isolated from
// authoritative scheduling and acceptance.
func Apply(s store.Store, p Plan, txID string) (ApplyResult, error) {
	var res ApplyResult
	if len(p.Quarantined) > 0 {
		return res, fmt.Errorf("%w: %d conflicting identity group(s) are quarantined; the import fails closed",
			ErrAmbiguousImport, len(p.Quarantined))
	}
	txn, err := s.BeginMutation(txID)
	if err != nil {
		return res, err
	}
	for _, e := range p.Epics {
		if err := txn.PutEpic(e); err != nil {
			return res, err
		}
	}
	for _, it := range p.Items {
		if err := txn.PutItem(it); err != nil {
			return res, err
		}
	}
	if err := txn.Commit(); err != nil {
		return res, err
	}
	res.CreatedEpics = len(p.Epics)
	res.CreatedItems = len(p.Items)
	res.Skipped = len(p.Skipped)
	res.Ambiguities = len(p.Ambiguities)
	res.Unresolved = len(p.Unresolved)
	return res, nil
}

// Delta describes the canonical difference between two ledger states.
type Delta struct {
	// AddedItems lists the work items present only in the new state.
	AddedItems []string `json:"added_items"`
	// RemovedItems lists the work items present only in the old state.
	RemovedItems []string `json:"removed_items"`
	// ChangedItems lists the work items whose content changed.
	ChangedItems []string `json:"changed_items"`
	// AddedEvidence lists the evidence present only in the new state.
	AddedEvidence []string `json:"added_evidence"`
}

// Empty reports whether the delta contains no change at all.
func (d Delta) Empty() bool {
	return len(d.AddedItems) == 0 && len(d.RemovedItems) == 0 &&
		len(d.ChangedItems) == 0 && len(d.AddedEvidence) == 0
}

// ComputeDelta computes the canonical difference between two states.
func ComputeDelta(old, updated domain.Ledger) Delta {
	var d Delta

	oldItems := map[string]domain.WorkItem{}
	for _, it := range old.Items {
		oldItems[it.Scoped().String()] = it
	}
	newItems := map[string]domain.WorkItem{}
	for _, it := range updated.Items {
		newItems[it.Scoped().String()] = it
	}
	for id := range newItems {
		prior, ok := oldItems[id]
		if !ok {
			d.AddedItems = append(d.AddedItems, id)
			continue
		}
		if itemContentChanged(prior, newItems[id]) {
			d.ChangedItems = append(d.ChangedItems, id)
		}
	}
	for id := range oldItems {
		if _, ok := newItems[id]; !ok {
			d.RemovedItems = append(d.RemovedItems, id)
		}
	}
	oldEvidence := map[string]bool{}
	for _, e := range old.Evidence {
		oldEvidence[string(e.EvidenceID)] = true
	}
	for _, e := range updated.Evidence {
		if !oldEvidence[string(e.EvidenceID)] {
			d.AddedEvidence = append(d.AddedEvidence, string(e.EvidenceID))
		}
	}
	sortStrings(d.AddedItems)
	sortStrings(d.RemovedItems)
	sortStrings(d.ChangedItems)
	sortStrings(d.AddedEvidence)
	return d
}

// itemContentChanged reports whether the content of a work item differs.
//
// The comparison is field-wise because the record carries slices that are
// not comparable with the equality operator.
func itemContentChanged(a, b domain.WorkItem) bool {
	if a.Title != b.Title || a.Summary != b.Summary || a.Kind != b.Kind ||
		a.Lifecycle != b.Lifecycle || a.Acceptance != b.Acceptance ||
		a.Priority != b.Priority || a.Quality != b.Quality ||
		a.Owner != b.Owner || a.Revision != b.Revision ||
		a.Archived != b.Archived || a.RequiresLiveGate != b.RequiresLiveGate ||
		a.TerminalDisposition != b.TerminalDisposition {
		return true
	}
	if !equalStrings(a.AcceptanceCriteria, b.AcceptanceCriteria) {
		return true
	}
	if !equalStrings(a.ExitCriteria, b.ExitCriteria) {
		return true
	}
	if len(a.Dependencies) != len(b.Dependencies) {
		return true
	}
	for i := range a.Dependencies {
		if a.Dependencies[i] != b.Dependencies[i] {
			return true
		}
	}
	if len(a.Evidence) != len(b.Evidence) {
		return true
	}
	for i := range a.Evidence {
		if a.Evidence[i] != b.Evidence[i] {
			return true
		}
	}
	if len(a.Successors) != len(b.Successors) {
		return true
	}
	for i := range a.Successors {
		if a.Successors[i] != b.Successors[i] {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
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

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// Describe renders a deterministic human-readable delta.
func (d Delta) Describe() string {
	return fmt.Sprintf("added: %d, removed: %d, changed: %d, evidence added: %d",
		len(d.AddedItems), len(d.RemovedItems), len(d.ChangedItems), len(d.AddedEvidence))
}
