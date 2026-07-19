// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"sort"
	"strings"
)

// BuildManifest builds a deterministic manifest of changed files for
// dirty/staged modes.
//
// The change kind comes directly from `ChangedFile.Kind`, which was
// populated from Git's authoritative `--name-status -z` output. This
// function never infers `A`/`M`/`D`/`R`/`C` from the boolean presence
// flags; those flags describe whether staged and/or unstaged diffs
// should be rendered alongside the manifest entry and are independent
// of the manifest classification.
//
// If the kind is missing (zero value) the manifest still renders an
// empty status slot so the evidence hashes flag any regression where
// the caller forgets to populate it.
func BuildManifest(files []ChangedFile) []ReviewChangedFile {
	result := make([]ReviewChangedFile, 0, len(files))

	for _, f := range files {
		entry := ReviewChangedFile{
			Status: string(f.Kind),
			Path:   f.Path,
		}
		// Renames and copies carry both old and new paths. Preserve
		// the old path so the renderer can emit the canonical
		// `R old -> new` / `C old -> new` form.
		if f.OldPath != "" && f.OldPath != f.Path {
			entry.OldPath = f.OldPath
		}
		result = append(result, entry)
	}

	// Sort by path lexicographically
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// BuildRangeManifest builds a deterministic manifest of changed files for range mode.
func BuildRangeManifest(files []RangeFile) []ReviewChangedFile {
	var result []ReviewChangedFile

	for _, f := range files {
		status := f.Status
		switch status {
		case "added":
			status = StatusAdded
		case "modified":
			status = StatusModified
		case "deleted":
			status = StatusDeleted
		case "renamed":
			status = StatusRenamed
		case "copied":
			status = StatusCopied
		}

		rf := ReviewChangedFile{
			Status: status,
			Path:   f.Path,
		}

		if status == StatusRenamed && f.From != "" && f.From != f.Path {
			rf.OldPath = f.From
		}

		result = append(result, rf)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// RenderManifest renders the CHANGESET_MANIFEST section.
func RenderManifest(manifest []ReviewChangedFile) string {
	var sb strings.Builder
	sb.WriteString("## CHANGESET_MANIFEST\n")

	if len(manifest) == 0 {
		sb.WriteString("(no changed files)\n")
		return sb.String()
	}

	for _, f := range manifest {
		if f.OldPath != "" && f.OldPath != f.Path {
			sb.WriteString(f.Status)
			sb.WriteString("  ")
			sb.WriteString(f.OldPath)
			sb.WriteString(" -> ")
			sb.WriteString(f.Path)
			sb.WriteString("\n")
		} else {
			sb.WriteString(f.Status)
			sb.WriteString("  ")
			sb.WriteString(f.Path)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
