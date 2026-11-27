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
		for _, ref := range item.DependsOn {
			target := ref
			if !strings.Contains(target, "/") && item.EpicID != "" {
				target = item.EpicID + "/" + target
			}
			if imported[target] || known[target] {
				p.Items[idx].Dependencies = append(p.Items[idx].Dependencies,
					domain.Dependency{
						Kind: domain.DependencyRequires,
						Ref:  domain.ScopedItemID(target),
					})
				continue
			}
			p.Items[idx].Dependencies = append(p.Items[idx].Dependencies,
				domain.Dependency{
					Kind: domain.DependencyBlocker,
					Ref:  domain.ScopedItemID(target),
					Note: "unresolved source dependency",
				})
			p.Unresolved = append(p.Unresolved, UnresolvedReference{
				SourceID: item.SourceID,
				ScopedID: scoped.String(),
				Ref:      ref,
				Kind:     "dependency",
			})
		}
	}
}
