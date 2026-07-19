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
// Paths are preserved verbatim: `ReviewChangedFile.Path` and
// `OldPath` carry the **raw** repository-relative paths so that
// filesystem inspection (`ComputeStats` classification, generated /
// binary detection, review-map construction, diff lookup) still
// addresses the on-disk path. Renderers that emit the manifest into
// line-oriented Markdown apply `PathEscape` only at the rendering
// boundary, never here.
//
// OldPath is populated only for renames and copies, only when it
// differs from Path.
func BuildManifest(files []ChangedFile) []ReviewChangedFile {
	result := make([]ReviewChangedFile, 0, len(files))

	for _, f := range files {
		entry := ReviewChangedFile{
			Status:  string(f.Kind),
			Path:    f.Path,
			OldPath: f.OldPath,
		}
		// Renames and copies carry both old and new paths.
		if f.OldPath == "" || f.OldPath == f.Path {
			entry.OldPath = ""
		}
		result = append(result, entry)
	}

	// Sort by raw path lexicographically. Sort does not reorder
	// bytes inside a path, so sorting the raw paths produces the
	// same order as sorting the escaped paths.
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
// `--name-status -z` parser. Paths are preserved verbatim for the
// same reasons as `BuildManifest` — escaping is a render concern, not
// a semantic concern.
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
			Path:    f.Path,
			OldPath: f.From,
		}
		// Renames AND copies carry both old and new paths; the
		// manifest renderer emits the `R old -> new` / `C old -> new`
		// form whenever OldPath is set and differs from Path.
		if status != StatusRenamed && status != StatusCopied {
			entry.OldPath = ""
		}
		if entry.OldPath == "" || entry.OldPath == entry.Path {
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
// Paths are escaped at the rendering boundary via `PathEscape` so a
// single manifest entry can never split across visual lines, even
// if the underlying filename contains a tab, newline, carriage
// return, backslash, or a control byte. Callers that want the
// original filename can parse the rendered line back with
// `ParseEscapedPath`.
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
			sb.WriteString(PathEscape(f.OldPath))
			sb.WriteString(" -> ")
			sb.WriteString(PathEscape(f.Path))
			sb.WriteString("\n")
		} else {
			sb.WriteString(f.Status)
			sb.WriteString("  ")
			sb.WriteString(PathEscape(f.Path))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
