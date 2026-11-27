package importer

import (
	"sort"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// CoverageComplete is the source coverage that declares a COMPLETE recovery
// of a collection.
const CoverageComplete = "COMPLETE"

// incompleteCoverage reports whether a source coverage declares an
// INCOMPLETE recovery of a collection.
//
// A COMPLETE coverage is a verified account of the collection: an empty
// collection with a COMPLETE coverage is a VERIFIED ABSENCE of members. Any
// other declared coverage means the collection was NOT fully recovered, and
// the importer reports it instead of treating it as an absence.
//
// An EMPTY coverage means the source declared no coverage at all: the source
// made no statement, and the importer does not make one either.
func incompleteCoverage(coverage string) bool {
	return coverage != "" && coverage != CoverageComplete
}

// reopenConditionsOf returns the reopen conditions of a source record.
//
// The conditions are conserved verbatim: the exact text the source stated
// survives the import, and the KIND records that the condition came from the
// source rather than from a canonical decision.
func reopenConditionsOf(item SourceItem) []domain.ReopenCondition {
	if len(item.ReopenConditions) == 0 {
		return nil
	}
	out := make([]domain.ReopenCondition, 0, len(item.ReopenConditions))
	for _, condition := range item.ReopenConditions {
		out = append(out, domain.ReopenCondition{
			Kind:   "source_reopen_condition",
			Detail: condition,
		})
	}
	return out
}

// epicExitCriteriaOf returns the exit criteria that belong to the epic
// scope.
//
// The per-epic declaration wins. A board that declares a single epic scope
// owns its board-scoped exit criteria: they belong to that scope. A board
// with several epic scopes never assigns its board-scoped criteria to an
// arbitrary one of them: the criteria would be attributed by accident.
func epicExitCriteriaOf(source SourceBoard, epicID string) []string {
	if criteria, ok := source.EpicExitCriteria[epicID]; ok {
		return criteria
	}
	if len(distinctEpicScopes(source)) == 1 {
		return source.ExitCriteria
	}
	return nil
}

// distinctEpicScopes returns the distinct epic scopes of a source board.
func distinctEpicScopes(source SourceBoard) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range source.Items {
		if item.EpicID == "" || seen[item.EpicID] {
			continue
		}
		seen[item.EpicID] = true
		out = append(out, item.EpicID)
	}
	sort.Strings(out)
	return out
}

// epicCarriesCriteria reports whether the canonical epic record carries
// every one of the given exit criteria.
func epicCarriesCriteria(existing domain.Ledger, epicID domain.EpicID, criteria []string) bool {
	for _, epic := range existing.Epics {
		if epic.EpicID != epicID {
			continue
		}
		for _, want := range criteria {
			found := false
			for _, have := range epic.ExitCriteria {
				if have == want {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	return false
}
