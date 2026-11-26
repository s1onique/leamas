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

// Apply publishes an import plan into the canonical ledger.
//
// The import is idempotent: a source record that already exists is
// skipped, so replaying an import produces no canonical delta.
func Apply(s store.Store, p Plan, txID string) (ApplyResult, error) {
	var res ApplyResult
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
