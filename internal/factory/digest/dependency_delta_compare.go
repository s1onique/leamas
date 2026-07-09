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
func compareReplaces(baseMod, headMod *goModWithToolchain) (added, removed, modified []string) {
	baseRepls := make(map[string]string)
	headRepls := make(map[string]string)

	if baseMod != nil {
		for _, r := range baseMod.Replace {
			baseRepls[r.New.Path] = r.New.Version
		}
	}
	if headMod != nil {
		for _, r := range headMod.Replace {
			headRepls[r.New.Path] = r.New.Version
		}
	}

	for path, headVer := range headRepls {
		if baseVer, exists := baseRepls[path]; !exists {
			added = append(added, path+" "+headVer)
		} else if baseVer != headVer {
			modified = append(modified, fmt.Sprintf("%s %s -> %s", path, baseVer, headVer))
		}
	}

	for path := range baseRepls {
		if _, exists := headRepls[path]; !exists {
			removed = append(removed, path+" "+baseRepls[path])
		}
	}

	return
}

// compareGoSum returns entries added to go.sum.
func compareGoSum(base, head map[string]string) []string {
	var added []string
	for key := range head {
		if _, exists := base[key]; !exists {
			added = append(added, key)
		}
	}
	return added
}
