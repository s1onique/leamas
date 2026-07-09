// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"strings"
)

// RenderDependencyDelta renders a DependencyDelta as a string.
func RenderDependencyDelta(delta *DependencyDelta) string {
	var sb strings.Builder
	sb.WriteString("## DEPENDENCY_DELTA\n")
	sb.WriteString(fmt.Sprintf("ecosystem=%s\n", delta.Ecosystem))
	sb.WriteString(fmt.Sprintf("source_status=%s\n", delta.SourceStatus))
	sb.WriteString(fmt.Sprintf("go_mod_changed=%t\n", delta.GoModChanged))
	sb.WriteString(fmt.Sprintf("go_sum_changed=%t\n", delta.GoSumChanged))
	sb.WriteString(fmt.Sprintf("module_path_changed=%t\n", delta.ModulePathChanged))
	sb.WriteString(fmt.Sprintf("go_version_changed=%t\n", delta.GoVersionChanged))
	sb.WriteString(fmt.Sprintf("toolchain_changed=%t\n", delta.ToolchainChanged))
	sb.WriteString(fmt.Sprintf("requires_added=%d\n", delta.RequiresAdded))
	sb.WriteString(fmt.Sprintf("requires_removed=%d\n", delta.RequiresRemoved))
	sb.WriteString(fmt.Sprintf("requires_modified=%d\n", delta.RequiresModified))
	sb.WriteString(fmt.Sprintf("replaces_added=%d\n", delta.ReplacesAdded))
	sb.WriteString(fmt.Sprintf("replaces_removed=%d\n", delta.ReplacesRemoved))
	sb.WriteString(fmt.Sprintf("replaces_modified=%d\n", delta.ReplacesModified))

	sb.WriteString("\nmodule:\n")
	sb.WriteString(fmt.Sprintf("  before=%s\n", delta.Module.Before))
	sb.WriteString(fmt.Sprintf("  after=%s\n", delta.Module.After))

	sb.WriteString("\ngo_version:\n")
	sb.WriteString(fmt.Sprintf("  before=%s\n", delta.GoVersion.Before))
	sb.WriteString(fmt.Sprintf("  after=%s\n", delta.GoVersion.After))

	sb.WriteString("\ntoolchain:\n")
	sb.WriteString(fmt.Sprintf("  before=%s\n", delta.Toolchain.Before))
	sb.WriteString(fmt.Sprintf("  after=%s\n", delta.Toolchain.After))

	sb.WriteString("\nrequires_added:\n")
	if len(delta.RequiresAddedList) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, r := range delta.RequiresAddedList {
			sb.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	sb.WriteString("\nrequires_removed:\n")
	if len(delta.RequiresRemovedList) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, r := range delta.RequiresRemovedList {
			sb.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	sb.WriteString("\nrequires_modified:\n")
	if len(delta.RequiresModifiedList) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, r := range delta.RequiresModifiedList {
			sb.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	sb.WriteString("\nreplaces_added:\n")
	if len(delta.ReplacesAddedList) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, r := range delta.ReplacesAddedList {
			sb.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	sb.WriteString("\nreplaces_removed:\n")
	if len(delta.ReplacesRemovedList) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, r := range delta.ReplacesRemovedList {
			sb.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	sb.WriteString("\nreplaces_modified:\n")
	if len(delta.ReplacesModifiedList) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, r := range delta.ReplacesModifiedList {
			sb.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	return sb.String()
}

// RenderEmptyDependencyDelta renders an empty/no-change delta.
func RenderEmptyDependencyDelta() string {
	return RenderDependencyDelta(&DependencyDelta{
		Ecosystem:         "go",
		SourceStatus:      "absent",
		GoModChanged:      false,
		GoSumChanged:      false,
		ModulePathChanged: false,
		GoVersionChanged:  false,
		ToolchainChanged:  false,
	})
}
