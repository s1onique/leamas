// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
)

// compareRequires compares require directives between two go.mod versions.
func compareRequires(baseMod, headMod *goModWithToolchain) (added, removed, modified []string) {
	baseReqs := make(map[string]string)
	headReqs := make(map[string]string)

	if baseMod != nil {
		for _, r := range baseMod.Require {
			baseReqs[r.Mod.Path] = r.Mod.Version
		}
	}
	if headMod != nil {
		for _, r := range headMod.Require {
			headReqs[r.Mod.Path] = r.Mod.Version
		}
	}

	for path, headVer := range headReqs {
		if baseVer, exists := baseReqs[path]; !exists {
			added = append(added, path+" "+headVer)
		} else if baseVer != headVer {
			modified = append(modified, fmt.Sprintf("%s %s -> %s", path, baseVer, headVer))
		}
	}

	for path := range baseReqs {
		if _, exists := headReqs[path]; !exists {
			removed = append(removed, path+" "+baseReqs[path])
		}
	}

	return
}

// compareReplaces compares replace directives between two go.mod versions.
// Keys on old module path+version, values are new module path+version.
func compareReplaces(baseMod, headMod *goModWithToolchain) (added, removed, modified []string) {
	// Map old module path+version -> new module path+version
	baseRepls := make(map[string]string)
	headRepls := make(map[string]string)

	if baseMod != nil {
		for _, r := range baseMod.Replace {
			oldKey := r.Old.Path + " " + r.Old.Version
			newVal := r.New.Path + " " + r.New.Version
			baseRepls[oldKey] = newVal
		}
	}
	if headMod != nil {
		for _, r := range headMod.Replace {
			oldKey := r.Old.Path + " " + r.Old.Version
			newVal := r.New.Path + " " + r.New.Version
			headRepls[oldKey] = newVal
		}
	}

	// Find added (in head but not in base)
	for oldKey, newVal := range headRepls {
		if _, exists := baseRepls[oldKey]; !exists {
			added = append(added, oldKey+" => "+newVal)
		} else if baseRepls[oldKey] != newVal {
			modified = append(modified, oldKey+" => "+baseRepls[oldKey]+" -> "+newVal)
		}
	}

	// Find removed (in base but not in head)
	for oldKey := range baseRepls {
		if _, exists := headRepls[oldKey]; !exists {
			removed = append(removed, oldKey+" => "+baseRepls[oldKey])
		}
	}

	return
}

// compareGoSum returns go.sum entries added and removed.
func compareGoSum(base, head map[string]string) (added, removed []string) {
	for key := range head {
		if _, exists := base[key]; !exists {
			added = append(added, key)
		}
	}
	for key := range base {
		if _, exists := head[key]; !exists {
			removed = append(removed, key)
		}
	}
	return
}
