// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"sort"
	"strings"
)

// BuildManifest builds a deterministic manifest of changed files for dirty/staged modes.
func BuildManifest(files []ChangedFile) []ReviewChangedFile {
	var result []ReviewChangedFile

	for _, f := range files {
		status := StatusModified
		if f.Untracked {
			status = StatusUntracked
		} else if f.Tracked {
			if f.StagedPresent && !f.UnstagedPresent {
				status = StatusAdded
			} else if !f.StagedPresent && f.UnstagedPresent {
				status = StatusModified
			} else if f.StagedPresent && f.UnstagedPresent {
				status = StatusModified
			}
		}

		result = append(result, ReviewChangedFile{
			Status: status,
			Path:   f.Path,
		})
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
