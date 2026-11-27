package store

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// recordKind identifies which record family a staged write belongs to.
type recordKind int

const (
	kindEpic recordKind = iota
	kindItem
	kindAct
	kindEvidence
	kindDecision
	kindKnowledge
	kindMigration
)

func (k recordKind) dir() string {
	switch k {
	case kindEpic:
		return "epics"
	case kindItem:
		return "items"
	case kindAct:
		return "acts"
	case kindEvidence:
		return "evidence"
	case kindDecision:
		return "decisions"
	case kindKnowledge:
		return "knowledge"
	case kindMigration:
		return "migrations"
	}
	return ""
}

func encodeRecord(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode record: %w", err)
	}
	return append(data, '\n'), nil
}

// Mutation accumulates record writes for a validated candidate state.
type Mutation struct {
	store   Store
	txn     *Transaction
	writes  map[string][]byte
	deletes map[string]bool
}

// BeginMutation starts a mutation against the current canonical state.
func (s Store) BeginMutation(txID string) (*Mutation, error) {
	txn, err := s.Begin(txID)
	if err != nil {
		return nil, err
	}
	return &Mutation{store: s, txn: txn, writes: map[string][]byte{}, deletes: map[string]bool{}}, nil
}

// LoadCurrent reads the canonical state while the caller holds the
// exclusive transaction lock.
//
// It is the writer-side counterpart of Load. It never takes the shared
// lock: the caller already holds the exclusive one.
func (m *Mutation) LoadCurrent() (domain.Ledger, error) {
	return m.store.loadUnlocked()
}

// stage encodes and stages one record write.
func (m *Mutation) stage(kind recordKind, name string, v any) error {
	data, err := encodeRecord(v)
	if err != nil {
		return err
	}
	rel := RelPathFor(kind.dir(), name)
	m.writes[rel] = data
	return nil
}

// PutEpic stages an epic record.
func (m *Mutation) PutEpic(e domain.Epic) error {
	return m.stage(kindEpic, string(e.EpicID)+".json", e)
}

// PutItem stages a work item record.
func (m *Mutation) PutItem(it domain.WorkItem) error {
	return m.stage(kindItem, itemFileName(it.Scoped()), it)
}

// PutAct stages an ACT execution record.
func (m *Mutation) PutAct(a domain.ActExecution) error {
	return m.stage(kindAct, string(a.ActID)+".json", a)
}

// PutEvidence stages an evidence reference record.
func (m *Mutation) PutEvidence(e domain.EvidenceRefRecord) error {
	return m.stage(kindEvidence, string(e.EvidenceID)+".json", e)
}

// PutDecision stages a decision record.
func (m *Mutation) PutDecision(d domain.Decision) error {
	return m.stage(kindDecision, string(d.DecisionID)+".json", d)
}

// PutKnowledge stages a knowledge reference record.
func (m *Mutation) PutKnowledge(k domain.Knowledge) error {
	return m.stage(kindKnowledge, string(k.KnowledgeID)+".json", k)
}

// PutMigration stages a durable migration reconciliation record.
func (m *Mutation) PutMigration(rec domain.MigrationRecord) error {
	return m.stage(kindMigration, rec.MigrationID+".json", rec)
}

// RemoveItem stages the removal of a work item record.
func (m *Mutation) RemoveItem(id domain.ScopedItemID) {
	rel := RelPathFor(kindItem.dir(), itemFileName(id))
	m.deletes[rel] = true
	delete(m.writes, rel)
}

// RemoveAct stages the removal of an ACT execution record.
func (m *Mutation) RemoveAct(id domain.ActID) {
	rel := RelPathFor(kindAct.dir(), string(id)+".json")
	m.deletes[rel] = true
	delete(m.writes, rel)
}

// Candidate returns the candidate ledger state implied by the staged
// writes, so the caller can validate it before committing.
func (m *Mutation) Candidate() (domain.Ledger, error) {
	l, err := m.store.loadUnlocked()
	if err != nil {
		return l, err
	}

	items := map[domain.ScopedItemID]int{}
	for i, it := range l.Items {
		items[it.Scoped()] = i
	}
	for rel, data := range m.writes {
		switch {
		case filepath.Base(filepath.Dir(rel)) == kindItem.dir():
			var it domain.WorkItem
			if err := json.Unmarshal(data, &it); err != nil {
				return l, err
			}
			if idx, ok := items[it.Scoped()]; ok {
				l.Items[idx] = it
			} else {
				items[it.Scoped()] = len(l.Items)
				l.Items = append(l.Items, it)
			}
		case filepath.Base(filepath.Dir(rel)) == kindEpic.dir():
			var e domain.Epic
			if err := json.Unmarshal(data, &e); err != nil {
				return l, err
			}
			replaced := false
			for i := range l.Epics {
				if l.Epics[i].EpicID == e.EpicID {
					l.Epics[i] = e
					replaced = true
					break
				}
			}
			if !replaced {
				l.Epics = append(l.Epics, e)
			}
		case filepath.Base(filepath.Dir(rel)) == kindAct.dir():
			var a domain.ActExecution
			if err := json.Unmarshal(data, &a); err != nil {
				return l, err
			}
			replaced := false
			for i := range l.Acts {
				if l.Acts[i].ActID == a.ActID {
					l.Acts[i] = a
					replaced = true
					break
				}
			}
			if !replaced {
				l.Acts = append(l.Acts, a)
			}
		case filepath.Base(filepath.Dir(rel)) == kindEvidence.dir():
			var e domain.EvidenceRefRecord
			if err := json.Unmarshal(data, &e); err != nil {
				return l, err
			}
			replaced := false
			for i := range l.Evidence {
				if l.Evidence[i].EvidenceID == e.EvidenceID {
					l.Evidence[i] = e
					replaced = true
					break
				}
			}
			if !replaced {
				l.Evidence = append(l.Evidence, e)
			}
		case filepath.Base(filepath.Dir(rel)) == kindDecision.dir():
			var d domain.Decision
			if err := json.Unmarshal(data, &d); err != nil {
				return l, err
			}
			replaced := false
			for i := range l.Decisions {
				if l.Decisions[i].DecisionID == d.DecisionID {
					l.Decisions[i] = d
					replaced = true
					break
				}
			}
			if !replaced {
				l.Decisions = append(l.Decisions, d)
			}
		case filepath.Base(filepath.Dir(rel)) == kindKnowledge.dir():
			var k domain.Knowledge
			if err := json.Unmarshal(data, &k); err != nil {
				return l, err
			}
			replaced := false
			for i := range l.Knowledge {
				if l.Knowledge[i].KnowledgeID == k.KnowledgeID {
					l.Knowledge[i] = k
					replaced = true
					break
				}
			}
			if !replaced {
				l.Knowledge = append(l.Knowledge, k)
			}
		case filepath.Base(filepath.Dir(rel)) == kindMigration.dir():
			var rec domain.MigrationRecord
			if err := json.Unmarshal(data, &rec); err != nil {
				return l, err
			}
			replaced := false
			for i := range l.Migrations {
				if l.Migrations[i].MigrationID == rec.MigrationID {
					l.Migrations[i] = rec
					replaced = true
					break
				}
			}
			if !replaced {
				l.Migrations = append(l.Migrations, rec)
			}
		}
	}

	for rel := range m.deletes {
		switch filepath.Base(filepath.Dir(rel)) {
		case kindItem.dir():
			var kept []domain.WorkItem
			for _, it := range l.Items {
				if itemFileName(it.Scoped()) == filepath.Base(rel) {
					continue
				}
				kept = append(kept, it)
			}
			l.Items = kept
		case kindAct.dir():
			var kept []domain.ActExecution
			for _, a := range l.Acts {
				if string(a.ActID)+".json" == filepath.Base(rel) {
					continue
				}
				kept = append(kept, a)
			}
			l.Acts = kept
		}
	}

	domain.SortItems(l.Items)
	domain.SortActs(l.Acts)
	domain.SortEvidence(l.Evidence)
	return l, nil
}

// Release abandons the mutation and releases the transaction lock.
//
// It is used when the candidate is rejected before the publication: the
// lock must not stay held, otherwise the ledger would be permanently
// locked.
func (m *Mutation) Release() {
	if m == nil || m.txn == nil {
		return
	}
	m.txn.Release()
}

// Commit validates the candidate state and publishes the transaction.
//
// The candidate is validated BEFORE any destination is touched. A failing
// candidate leaves the ledger untouched.
//
// The publication is guarded by an optimistic revision check: a staged
// record may not overwrite a stored record whose revision is newer than the
// staged one. Two writers that both believe they hold the same expected
// revision cannot both publish: the second one is rejected instead of
// silently overwriting the first.
func (m *Mutation) Commit() error {
	defer m.Release()

	candidate, err := m.Candidate()
	if err != nil {
		return err
	}
	if res := domain.ValidateLedger(candidate); !res.OK() {
		return fmt.Errorf("candidate state rejected: %w", res)
	}
	if err := m.checkRevisionGuards(candidate); err != nil {
		return err
	}

	rels := make([]string, 0, len(m.writes))
	for rel := range m.writes {
		rels = append(rels, rel)
	}
	sortStrings(rels)
	for _, rel := range rels {
		m.txn.Write(rel, m.writes[rel])
	}
	dels := make([]string, 0, len(m.deletes))
	for rel := range m.deletes {
		dels = append(dels, rel)
	}
	sortStrings(dels)
	for _, rel := range dels {
		m.txn.Delete(rel)
	}
	return m.txn.Commit()
}
