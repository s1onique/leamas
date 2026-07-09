// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"sort"
)

// CollectDependencyDelta computes the dependency delta for given mode.
func CollectDependencyDelta(mode Mode, repoRoot string, files []ChangedFile) (*DependencyDelta, error) {
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}

	if !hasGoModuleFiles(paths) {
		return &DependencyDelta{
			Ecosystem:    "go",
			SourceStatus: "absent",
			GoModChanged: false,
		}, nil
	}

	return collectDependencyDeltaInternal(mode, repoRoot, nil, "", "")
}

// CollectRangeDependencyDelta computes the dependency delta for range mode.
func CollectRangeDependencyDelta(repoRoot string, rangeFiles []RangeFile, revRange string) (*DependencyDelta, error) {
	var paths []string
	for _, f := range rangeFiles {
		paths = append(paths, f.Path)
	}

	if !hasGoModuleFiles(paths) {
		return &DependencyDelta{
			Ecosystem:    "go",
			SourceStatus: "absent",
			GoModChanged: false,
		}, nil
	}

	base, head := getRangeModeInfo(revRange)
	return collectDependencyDeltaInternal(ModeRange, repoRoot, rangeFiles, base, head)
}

// collectDependencyDeltaInternal is the internal collector.
func collectDependencyDeltaInternal(mode Mode, repoRoot string, rangeFiles []RangeFile, base, head string) (*DependencyDelta, error) {
	var baseMod, headMod *goModWithToolchain
	var baseSum, headSum map[string]string

	switch mode {
	case ModeRange:
		baseMod, _ = getGoModAtCommit(repoRoot, base)
		headMod, _ = getGoModAtCommit(repoRoot, head)
		if baseMod != nil {
			baseSum, _ = getGoSumAtCommit(repoRoot, base)
		}
		if headMod != nil {
			headSum, _ = getGoSumAtCommit(repoRoot, head)
		}
	case ModeDirty, ModeStaged:
		baseMod, _ = getGoModAtCommit(repoRoot, "HEAD")
		headMod, _ = getWorktreeGoMod(repoRoot)
		if baseMod != nil {
			baseSum, _ = getGoSumAtCommit(repoRoot, "HEAD")
		}
		if headMod != nil {
			headSum, _ = getWorktreeGoSum(repoRoot)
		}
	default:
		baseMod, _ = getGoModAtCommit(repoRoot, "HEAD")
		headMod, _ = getWorktreeGoMod(repoRoot)
		if baseMod != nil {
			baseSum, _ = getGoSumAtCommit(repoRoot, "HEAD")
		}
		if headMod != nil {
			headSum, _ = getWorktreeGoSum(repoRoot)
		}
	}

	if baseSum == nil {
		baseSum = make(map[string]string)
	}
	if headSum == nil {
		headSum = make(map[string]string)
	}

	modulePathChanged := false
	goVersionChanged := false
	toolchainChanged := false

	if baseMod != nil && headMod != nil {
		if baseMod.Module.Mod.Path != headMod.Module.Mod.Path {
			modulePathChanged = true
		}
		baseGo, headGo := "", ""
		if baseMod.Go != nil {
			baseGo = baseMod.Go.Version
		}
		if headMod.Go != nil {
			headGo = headMod.Go.Version
		}
		if baseGo != headGo {
			goVersionChanged = true
		}
		if baseMod.ToolchainName != headMod.ToolchainName {
			toolchainChanged = true
		}
	}

	requiresAdded, requiresRemoved, requiresModified := compareRequires(baseMod, headMod)
	replacesAdded, replacesRemoved, replacesModified := compareReplaces(baseMod, headMod)

	sort.Strings(requiresAdded)
	sort.Strings(requiresRemoved)
	sort.Strings(requiresModified)
	sort.Strings(replacesAdded)
	sort.Strings(replacesRemoved)
	sort.Strings(replacesModified)

	_ = compareGoSum(baseSum, headSum)

	moduleInfo := ModuleInfo{}
	goVersionInfo := VersionInfo{}
	toolchainInfo := VersionInfo{}

	if baseMod != nil && baseMod.Module != nil {
		moduleInfo.Before = baseMod.Module.Mod.Path
	}
	if headMod != nil && headMod.Module != nil {
		moduleInfo.After = headMod.Module.Mod.Path
	}
	if baseMod != nil && baseMod.Go != nil {
		goVersionInfo.Before = baseMod.Go.Version
	}
	if headMod != nil && headMod.Go != nil {
		goVersionInfo.After = headMod.Go.Version
	}
	if baseMod != nil {
		toolchainInfo.Before = baseMod.ToolchainName
	}
	if headMod != nil {
		toolchainInfo.After = headMod.ToolchainName
	}

	return &DependencyDelta{
		Ecosystem:    "go",
		SourceStatus: "present",
		GoModChanged: modulePathChanged || goVersionChanged || toolchainChanged ||
			len(requiresAdded) > 0 || len(requiresRemoved) > 0 || len(requiresModified) > 0,
		GoSumChanged:         !mapsEqual(baseSum, headSum),
		ModulePathChanged:    modulePathChanged,
		GoVersionChanged:     goVersionChanged,
		ToolchainChanged:     toolchainChanged,
		RequiresAdded:        len(requiresAdded),
		RequiresRemoved:      len(requiresRemoved),
		RequiresModified:     len(requiresModified),
		ReplacesAdded:        len(replacesAdded),
		ReplacesRemoved:      len(replacesRemoved),
		ReplacesModified:     len(replacesModified),
		Module:               moduleInfo,
		GoVersion:            goVersionInfo,
		Toolchain:            toolchainInfo,
		RequiresAddedList:    requiresAdded,
		RequiresRemovedList:  requiresRemoved,
		RequiresModifiedList: requiresModified,
		ReplacesAddedList:    replacesAdded,
		ReplacesRemovedList:  replacesRemoved,
		ReplacesModifiedList: replacesModified,
	}, nil
}
