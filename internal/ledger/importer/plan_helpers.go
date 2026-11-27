package importer

import (
	"sort"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

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
