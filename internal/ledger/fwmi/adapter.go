package fwmi

import (
	"fmt"
	"sort"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/importer"
)

// AdapterVersion is the version of the FWMI interchange adapter.
const AdapterVersion = 1

// AdaptResult is the result of adapting an interchange package.
type AdaptResult struct {
	// Board is the normalized source board of the package.
	//
	// The indexed historical ACT executions travel with the board: a
	// historical execution attempt is an execution record, never a
	// work-item obligation.
	Board importer.SourceBoard
	// Migrations are the durable reconciliation records of the package.
	Migrations []domain.MigrationRecord
	// Stats counts the adapted records.
	Stats Stats
}

// Stats counts the adapted interchange records.
type Stats struct {
	Epics                 int `json:"epics"`
	Items                 int `json:"items"`
	HistoricalActs        int `json:"historical_acts"`
	ReconciliationRecords int `json:"reconciliation_records"`
	Dependencies          int `json:"dependencies"`
	CrossLinks            int `json:"cross_links"`
	Unresolved            int `json:"unresolved"`
	// Blockers is the number of EXPLICIT blockers the package declared.
	Blockers int `json:"blockers"`
	// HistoricalAssociations is the number of historical ACT associations
	// the package carried.
	HistoricalAssociations int `json:"historical_associations"`
}

// Adapt converts the interchange package into the importer's source model.
//
// The adaptation is CONSERVATIVE. It never promotes a historical
// qualification assertion into an acceptance decision, never coerces an
// unknown source status into a deferral, and never invents a dependency.
func Adapt(pkg *Package) (AdaptResult, error) {
	var res AdaptResult
	res.Board = importer.SourceBoard{
		Source:           pkg.Manifest.Repository,
		EpicTitles:       map[string]string{},
		EpicMeta:         map[string]importer.EpicMeta{},
		EpicExitCriteria: map[string][]string{},
		Items:            []importer.SourceItem{},
		ExitCriteria:     []string{},
	}
	// The source identity of the board is the repository of the manifest.
	// It is NEVER substituted: a missing repository is a missing source
	// binding, and a placeholder would make an unbound source look bound.
	res.Board.Source = pkg.Manifest.Repository

	for _, epic := range pkg.Epics {
		res.Board.EpicTitles[epic.EpicID] = epic.Title
		res.Board.EpicMeta[epic.EpicID] = importer.EpicMeta{
			Lifecycle:  epic.Lifecycle,
			SourceRefs: convertRefsWithHead(epic.SourceRefs, pkg.Manifest.SourceHead),
		}
		if len(epic.ExitCriteria) > 0 {
			res.Board.EpicExitCriteria[epic.EpicID] = append([]string{}, epic.ExitCriteria...)
		}
		// The DECLARED epic record is the epic INVENTORY: a scope the
		// package declares stays discoverable even when it carries no
		// outstanding work item.
		res.Board.Epics = append(res.Board.Epics, importer.SourceEpic{
			EpicID:         epic.EpicID,
			Title:          epic.Title,
			Lifecycle:      epic.Lifecycle,
			Priority:       derefString(epic.Priority),
			OriginalStatus: derefString(epic.OriginalStatus),
			ExitCriteria:   append([]string{}, epic.ExitCriteria...),
			// The EFFECTIVE source HEAD of the declared scope is
			// retained: a reference that names no revision of its own
			// is bound to the source HEAD of the package.
			SourceRefs: convertRefsWithHead(epic.SourceRefs, pkg.Manifest.SourceHead),
		})
		for _, item := range epic.Items {
			src, deps, relations := adaptItem(epic, item, pkg.Manifest.SourceHead)
			res.Board.Items = append(res.Board.Items, src)
			res.Stats.Dependencies += deps
			res.Stats.CrossLinks += relations
			res.Stats.Blockers += len(src.Blockers)
			res.Stats.HistoricalAssociations += len(src.RelatedACTs)
			if src.CurrentACT != "" {
				res.Stats.HistoricalAssociations++
			}
		}
		res.Stats.Epics++
		res.Stats.Items += len(epic.Items)
	}

	for _, rec := range pkg.Historical.Records {
		if rec.ACTID == "" {
			res.Stats.Unresolved++
			continue
		}
		res.Board.Acts = append(res.Board.Acts, importer.SourceACT{
			SourceID:         rec.ACTID,
			OriginalVerdict:  derefString(rec.OriginalVerdict),
			EpicID:           derefString(rec.EpicID),
			WorkItemIDs:      append([]string{}, rec.WorkItemIDs...),
			EvidenceLocators: append([]string{}, rec.EvidenceLocators...),
			SourceRefs:       convertRefsWithHead(rec.SourceRefs, pkg.Manifest.SourceHead),
			Reconciliation:   rec.Reconciliation,
		})
		res.Stats.HistoricalActs++
	}

	seen := map[string]ReconciliationRecord{}
	for _, rec := range pkg.Reconciliation.Records {
		id := migrationIDOf(rec)
		if prior, ok := seen[id]; ok {
			if !sameReconciliationContent(prior, rec) {
				// Two DISTINCT reconciliation records claim the same
				// identity: the second one would overwrite the first.
				// The conflict is rejected BEFORE publication.
				return AdaptResult{}, fmt.Errorf("%w: %s and %s",
					ErrConflictingReconciliation, prior.Reason, rec.Reason)
			}
			// An exact repetition is a benign replay of the same record.
			continue
		}
		seen[id] = rec
		migration := migrationRecordOf(rec, pkg.Manifest)
		res.Migrations = append(res.Migrations, migration)
		res.Stats.ReconciliationRecords++
	}
	// The durable reconciliation records travel with the source board: the
	// PLAN reconciles them with the canonical migration state.
	res.Board.Migrations = res.Migrations

	sort.SliceStable(res.Board.Items, func(i, j int) bool {
		if res.Board.Items[i].EpicID != res.Board.Items[j].EpicID {
			return res.Board.Items[i].EpicID < res.Board.Items[j].EpicID
		}
		return res.Board.Items[i].SourceID < res.Board.Items[j].SourceID
	})
	return res, nil
}
