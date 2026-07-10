// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// PublicSurfaceDelta represents changes to the public Go API surface.
type PublicSurfaceDelta struct {
	Language         string
	SourceStatus     string
	PackagesChanged  int
	SymbolsAdded     int
	SymbolsRemoved   int
	SymbolsModified  int
	CLICommandsDelta int
	Packages         []string
	Added            []string
	Removed          []string
	Modified         []string
	CLICommands      []string
}

// getRangeModeInfo returns base and head for range mode.
func getRangeModeInfo(revRange string) (base, head string) {
	parts := strings.Split(revRange, "..")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], parts[len(parts)-1]
}

// extractGoFiles returns non-test .go files from changed files.
func extractGoFiles(paths []string) []string {
	var goFiles []string
	for _, p := range paths {
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			goFiles = append(goFiles, p)
		}
	}
	return goFiles
}

// extractCLIFiles returns files under cmd/leamas.
func extractCLIFiles(paths []string) []string {
	var cliFiles []string
	for _, p := range paths {
		if strings.HasPrefix(p, "cmd/leamas/") {
			cliFiles = append(cliFiles, p)
		}
	}
	return cliFiles
}

// CollectPublicSurfaceDelta computes the public surface delta for given mode.
func CollectPublicSurfaceDelta(mode Mode, repoRoot string, files []ChangedFile) (*PublicSurfaceDelta, error) {
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	return collectPublicSurfaceDeltaInternal(mode, repoRoot, nil, "", "", paths)
}

// CollectRangePublicSurfaceDelta computes the public surface delta for range mode.
func CollectRangePublicSurfaceDelta(repoRoot string, rangeFiles []RangeFile, revRange string) (*PublicSurfaceDelta, error) {
	var paths []string
	for _, f := range rangeFiles {
		paths = append(paths, f.Path)
	}
	base, head := getRangeModeInfo(revRange)
	return collectPublicSurfaceDeltaInternal(ModeRange, repoRoot, rangeFiles, base, head, paths)
}

// mergeExports merges exported symbols into the package's symbol map.
// This ensures all files in a package contribute to the same package-level symbol set.
func mergeExports(pkgExports map[symbolKey]symbolInfo, newExports map[symbolKey]symbolInfo) {
	for key, info := range newExports {
		// Only add if not already present (first occurrence wins)
		if _, exists := pkgExports[key]; !exists {
			pkgExports[key] = info
		}
	}
}

// isDeletedFile checks if a file is marked as deleted in range mode.
func isDeletedFile(paths []string, rangeFiles []RangeFile, filePath string) bool {
	if len(rangeFiles) == 0 {
		return false
	}
	for _, rf := range rangeFiles {
		if rf.Path == filePath && rf.Status == "deleted" {
			return true
		}
	}
	return false
}

// collectPublicSurfaceDeltaInternal is the internal collector.
// For dirty: base=HEAD, current=worktree
// For staged: base=HEAD, current=index
// For range: base=range-left, current=range-right
func collectPublicSurfaceDeltaInternal(mode Mode, repoRoot string, rangeFiles []RangeFile, base, head string, paths []string) (*PublicSurfaceDelta, error) {
	goFiles := extractGoFiles(paths)
	cliFiles := extractCLIFiles(paths)

	// Package-level symbol sets - symbols are merged across all files in a package.
	// This prevents false removals when a symbol is deleted from file A but
	// still exists in file B of the same package.
	baseExports := make(map[string]map[symbolKey]symbolInfo)
	headExports := make(map[string]map[symbolKey]symbolInfo)

	// Determine comparison targets based on mode
	for _, file := range goFiles {
		pkg := packageFromPath(file)

		// Ensure package maps exist
		if _, exists := baseExports[pkg]; !exists {
			baseExports[pkg] = make(map[symbolKey]symbolInfo)
		}
		if _, exists := headExports[pkg]; !exists {
			headExports[pkg] = make(map[symbolKey]symbolInfo)
		}

		// Get base exports
		var baseContent []byte
		var baseErr error
		switch mode {
		case ModeRange:
			baseContent, baseErr = getFileContentAtCommit(repoRoot, file, base)
		case ModeDirty, ModeStaged:
			// Compare against HEAD for both dirty and staged
			baseContent, baseErr = getFileContentAtCommit(repoRoot, file, "HEAD")
		default:
			// Auto mode - treat as dirty
			baseContent, baseErr = getFileContentAtCommit(repoRoot, file, "HEAD")
		}

		if baseErr == nil && len(baseContent) > 0 {
			exports, err := parseExportsFromBytes(baseContent, pkg)
			if err == nil {
				// Merge exports from this file into the package-level set
				mergeExports(baseExports[pkg], exports)
			}
		}

		// Get head exports (current state)
		// For deleted files in range mode, we still need to check if the package
		// has other files that might contain the same symbols
		var headContent []byte
		var headErr error
		fileDeleted := isDeletedFile(paths, rangeFiles, file)

		switch mode {
		case ModeRange:
			if fileDeleted {
				// File is deleted - no head content, but we still need to check
				// if the package exists in head (another file might have the symbols)
				// For now, headContent stays empty
				headErr = nil // No error, just no content
			} else {
				headContent, headErr = getFileContentAtCommit(repoRoot, file, head)
			}
		case ModeDirty:
			headContent, headErr = getWorktreeFileContent(repoRoot, file)
		case ModeStaged:
			headContent, headErr = getIndexFileContent(repoRoot, file)
			// If not staged, try worktree
			if headErr != nil {
				headContent, headErr = getWorktreeFileContent(repoRoot, file)
			}
		default:
			headContent, headErr = getWorktreeFileContent(repoRoot, file)
		}

		if headErr == nil && len(headContent) > 0 {
			exports, err := parseExportsFromBytes(headContent, pkg)
			if err == nil {
				// Merge exports from this file into the package-level set
				mergeExports(headExports[pkg], exports)
			}
		}
	}

	var added, removed, modified []string
	allPackages := make(map[string]bool)

	for pkg := range baseExports {
		allPackages[pkg] = true
	}
	for pkg := range headExports {
		allPackages[pkg] = true
	}

	for pkg := range allPackages {
		baseSymbols := baseExports[pkg]
		headSymbols := headExports[pkg]
		if baseSymbols == nil {
			baseSymbols = make(map[symbolKey]symbolInfo)
		}
		if headSymbols == nil {
			headSymbols = make(map[symbolKey]symbolInfo)
		}

		for key, headInfo := range headSymbols {
			if baseInfo, exists := baseSymbols[key]; !exists {
				added = append(added, fmt.Sprintf("%s.%s", pkg, key.String()))
			} else if baseInfo.Signature != headInfo.Signature {
				modified = append(modified, fmt.Sprintf("%s.%s", pkg, key.String()))
			}
		}
		for key := range baseSymbols {
			if _, exists := headSymbols[key]; !exists {
				removed = append(removed, fmt.Sprintf("%s.%s", pkg, key.String()))
			}
		}
	}

	var cliCommands []string
	for _, file := range cliFiles {
		fullPath := filepath.Join(repoRoot, file)
		cmds := extractCLISymbols(fullPath)
		cliCommands = append(cliCommands, cmds...)
	}
	cliCommands = deduplicateStrings(cliCommands)

	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(modified)
	sort.Strings(cliCommands)

	var packages []string
	for pkg := range allPackages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	return &PublicSurfaceDelta{
		Language:         "go",
		SourceStatus:     "present",
		PackagesChanged:  len(allPackages),
		SymbolsAdded:     len(added),
		SymbolsRemoved:   len(removed),
		SymbolsModified:  len(modified),
		CLICommandsDelta: len(cliCommands),
		Packages:         packages,
		Added:            added,
		Removed:          removed,
		Modified:         modified,
		CLICommands:      cliCommands,
	}, nil
}

// packageFromPath extracts package path from a Go file path.
func packageFromPath(filePath string) string {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "/" {
		return "main"
	}
	return strings.ReplaceAll(dir, "/", ".")
}

// RenderPublicSurfaceDelta renders a PublicSurfaceDelta as a string.
func RenderPublicSurfaceDelta(delta *PublicSurfaceDelta) string {
	var sb strings.Builder
	sb.WriteString("## PUBLIC_SURFACE_DELTA\n")
	sb.WriteString(fmt.Sprintf("language=%s\n", delta.Language))
	sb.WriteString(fmt.Sprintf("source_status=%s\n", delta.SourceStatus))
	sb.WriteString(fmt.Sprintf("packages_changed=%d\n", delta.PackagesChanged))
	sb.WriteString(fmt.Sprintf("symbols_added=%d\n", delta.SymbolsAdded))
	sb.WriteString(fmt.Sprintf("symbols_removed=%d\n", delta.SymbolsRemoved))
	sb.WriteString(fmt.Sprintf("symbols_modified=%d\n", delta.SymbolsModified))
	sb.WriteString(fmt.Sprintf("cli_commands_changed=%d\n", delta.CLICommandsDelta))

	sb.WriteString("\npackages:\n")
	if len(delta.Packages) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, pkg := range delta.Packages {
			sb.WriteString(fmt.Sprintf("  - %s\n", pkg))
		}
	}

	sb.WriteString("\nadded:\n")
	if len(delta.Added) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, sym := range delta.Added {
			sb.WriteString(fmt.Sprintf("  - %s\n", sym))
		}
	}

	sb.WriteString("\nremoved:\n")
	if len(delta.Removed) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, sym := range delta.Removed {
			sb.WriteString(fmt.Sprintf("  - %s\n", sym))
		}
	}

	sb.WriteString("\nmodified:\n")
	if len(delta.Modified) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, sym := range delta.Modified {
			sb.WriteString(fmt.Sprintf("  - %s\n", sym))
		}
	}

	sb.WriteString("\ncli_commands:\n")
	if len(delta.CLICommands) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, cmd := range delta.CLICommands {
			sb.WriteString(fmt.Sprintf("  - %s\n", cmd))
		}
	}

	return sb.String()
}

// RenderEmptyPublicSurfaceDelta renders an empty/no-change delta.
func RenderEmptyPublicSurfaceDelta() string {
	return RenderPublicSurfaceDelta(&PublicSurfaceDelta{
		Language:         "go",
		SourceStatus:     "present",
		PackagesChanged:  0,
		SymbolsAdded:     0,
		SymbolsRemoved:   0,
		SymbolsModified:  0,
		CLICommandsDelta: 0,
	})
}
