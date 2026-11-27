package importer

import (
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// resolveDependencies resolves the source dependency graph of a plan.
//
// Every resolved dependency becomes a canonical typed edge. Every
// UNRESOLVED blocking dependency becomes a durable blocker: it is
// recorded as a typed edge that points at an identity that does not
// exist yet, so the importing item can never become executable while
// the blocker is outstanding.
func resolveDependencies(p *Plan, source SourceBoard, imported map[string]bool, known map[string]bool, conflicting map[string]bool, itemIndex map[string]int) {
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
		for _, dep := range dependencyEdgesOf(item) {
			target := dep.Target
			if !strings.Contains(target, "/") && item.EpicID != "" {
				target = item.EpicID + "/" + target
			}
			if imported[target] || known[target] {
				p.Items[idx].Dependencies = append(p.Items[idx].Dependencies,
					domain.Dependency{
						Kind:            domain.DependencyRequires,
						Ref:             domain.ScopedItemID(target),
						RequiredOutcome: dep.RequiredOutcome,
						SourceRefs:      dep.SourceRefs,
					})
				continue
			}
			p.Items[idx].Dependencies = append(p.Items[idx].Dependencies,
				domain.Dependency{
					Kind:            domain.DependencyBlocker,
					Ref:             domain.ScopedItemID(target),
					Note:            "unresolved source dependency",
					RequiredOutcome: dep.RequiredOutcome,
					SourceRefs:      dep.SourceRefs,
				})
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID:         item.SourceID,
				ScopedID:         scoped.String(),
				Ref:              dep.Target,
				Kind:             "dependency",
				RelationshipKind: dep.Kind,
				RequiredOutcome:  dep.RequiredOutcome,
				SourceRefs:       dep.SourceRefs,
			})
		}
	}
}

// dependencyEdgesOf returns the blocking dependency edges of a source
// record with their full load-bearing content.
//
// A source that declared the DEPENDS_ON edges with their required outcome and
// their byte-exact provenance is conserved through the relations. A source
// that declared only plain targets is conserved through the plain list.
func dependencyEdgesOf(item SourceItem) []SourceRelation {
	if len(item.DependsOnRelations) > 0 {
		return item.DependsOnRelations
	}
	out := make([]SourceRelation, 0, len(item.DependsOn))
	for _, target := range item.DependsOn {
		out = append(out, SourceRelation{Kind: "DEPENDS_ON", Target: target})
	}
	return out
}

// resolveRelations resolves the NON-BLOCKING known relationships of the
// source records.
//
// A cross-link or an external relationship is NOT a local execution
// prerequisite. It becomes a canonical typed "relates" edge when its target
// resolves, and it is reported as an explicit unresolved reference when it
// does not. It never becomes a blocker: only a DEPENDS_ON relationship
// establishes local execution authority.
func resolveRelations(p *Plan, source SourceBoard, imported map[string]bool, known map[string]bool, itemIndex map[string]int) {
	for _, item := range source.Items {
		if item.SourceID == "" || item.EpicID == "" {
			continue
		}
		scoped := domain.NewScopedItemID(domain.EpicID(item.EpicID), domain.ItemID(item.SourceID))
		idx, ok := itemIndex[scoped.String()]
		if !ok {
			continue
		}
		for _, rel := range item.Relations {
			target := rel.Target
			if !strings.Contains(target, "/") && item.EpicID != "" {
				target = item.EpicID + "/" + target
			}
			if imported[target] || known[target] {
				p.Items[idx].Dependencies = append(p.Items[idx].Dependencies,
					domain.Dependency{
						Kind:            domain.DependencyRelates,
						Ref:             domain.ScopedItemID(target),
						Note:            rel.Kind,
						RequiredOutcome: rel.RequiredOutcome,
						SourceRefs:      rel.SourceRefs,
					})
				continue
			}
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID:         item.SourceID,
				ScopedID:         scoped.String(),
				Ref:              rel.Target,
				Kind:             "relation",
				RelationshipKind: rel.Kind,
				RequiredOutcome:  rel.RequiredOutcome,
				SourceRefs:       rel.SourceRefs,
			})
		}
	}
}
