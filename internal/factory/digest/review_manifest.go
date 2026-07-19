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
// Paths are written through `PathEscape` so a single manifest entry
// can never split across visual lines, even if the underlying
// filename contains a tab, newline, carriage return, backslash, or
// a control byte. Callers that want the original filename can parse
// the rendered line back with `ParseEscapedPath`.
func BuildManifest(files []ChangedFile) []ReviewChangedFile {
	result := make([]ReviewChangedFile, 0, len(files))

	for _, f := range files {
		entry := ReviewChangedFile{
			Status:  string(f.Kind),
			Path:    PathEscape(f.Path),
			OldPath: PathEscape(f.OldPath),
		}
		// Renames and copies carry both old and new paths.
		// The escaped form above produces the canonical printed
		// path; OldPath is set only when it differs from Path so
		// the renderer can emit the `R old -> new` form.
		if f.OldPath != "" && f.OldPath != f.Path {
			// Keep entry as constructed; entry.OldPath already set.
		} else {
			entry.OldPath = ""
		}
		result = append(result, entry)
	}

	// Sort by (escaped) path lexicographically. `PathEscape` is a pure
	// formatter that does not reorder bytes inside a path, so sorting
	// the escaped strings gives the same order as sorting the raw
	// paths.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// BuildRangeManifest builds a deterministic manifest of changed files
// for range mode.
//
// The change kind is sourced from the RangeFile's `Status` field,
// which is already a single-letter Git status set by the structured
// `--name-status -z` parser. Paths are escaped on render so unusual
// filenames survive the digest unchanged.
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
		case "type_changed":
			status = StatusTypeChanged
		case "unmerged":
			status = StatusUnmerged
		case "unknown":
			status = StatusUnknown
		case "broken_pair":
			status = StatusBrokenPair
		}

		entry := ReviewChangedFile{
			Status:  status,
			Path:    PathEscape(f.Path),
			OldPath: PathEscape(f.From),
		}
		if status == StatusRenamed && f.From != "" && f.From != f.Path {
			// OldPath already set above.
		} else {
			entry.OldPath = ""
		}
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// RenderManifest renders the CHANGESET_MANIFEST section.
//
// Manifest lines are emitted using the canonical escaped path form.
// Callers that want to recover the original filename must call
// `ParseEscapedPath` on the printed path.
func RenderManifest(manifest []ReviewChangedFile) string {
	var sb strings.Builder
	sb.WriteString("## CHANGESET_MANIFEST\n")

	if len(manifest) == 0 {
		sb.WriteString("(no changed files)\n")
		return sb.String()
	}

	for _, f := range manifest {
		if f.OldPath != "" {
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
