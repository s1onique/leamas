// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"sort"
	"strings"
)

// RangeFile represents a file changed in a commit range.
//
// The change kind is sourced from `git diff --name-status -z` via the
// shared NUL-delimited parser. `Status` is one of "added",
// "modified", "deleted", "renamed", or "copied" (the human-readable
// form used by BuildRangeManifest). `From` and `To` carry the pre/
// post-change paths so the manifest can render the
// `R old -> new` / `C source -> copy` form when applicable.
type RangeFile struct {
	Path   string
	From   string
	To     string
	Status string // "added", "modified", "deleted", "renamed", "copied"
}

// GetRangeFiles returns files changed in the given revision range.
//
// The change kind is sourced from `git diff --name-status -z` and
// routed through the same NUL-delimited parser used by staged and
// dirty modes, so the range results are consistent with the other
// modes for rename and copy detection. Renamed and copied entries
// retain both the old and new path on the resulting RangeFile.
func GetRangeFiles(repoRoot, revRange string) ([]RangeFile, error) {
	args := []string{"diff", "--name-status", "-z"}
	args = append(args, detectArgs()...)
	args = append(args, revRange)

	output, err := RunGit(repoRoot, args)
	if err != nil {
		return nil, err
	}

	records, err := ParseGitStatusRecords(output)
	if err != nil {
		return nil, err
	}

	var files []RangeFile
	for _, ch := range records {
		if ch.Path == "" {
			continue
		}
		files = append(files, RangeFile{
			Path:   ch.Path,
			From:   ch.OldPath,
			To:     ch.Path,
			Status: statusToHuman(string(ch.Kind)),
		})
	}

	// Deduplicate files and sort.
	files = UniqueRangeFiles(files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// statusToHuman converts git status letter to human-readable string.
//
// The renamed/copied human names only fire for those leading letters;
// other status letters (T, U, X, B) carry through as the lowercase
// form so BuildRangeManifest can map them to the corresponding single
// letter status code. The default clause only catches truly unknown
// letters and collapses them to "modified" so the rest of the digest
// pipeline remains robust.
func statusToHuman(status string) string {
	switch {
	case strings.HasPrefix(status, "A"):
		return "added"
	case strings.HasPrefix(status, "D"):
		return "deleted"
	case strings.HasPrefix(status, "M"):
		return "modified"
	case strings.HasPrefix(status, "R"):
		return "renamed"
	case strings.HasPrefix(status, "C"):
		return "copied"
	case strings.HasPrefix(status, "T"):
		return "type_changed"
	case strings.HasPrefix(status, "U"):
		return "unmerged"
	case strings.HasPrefix(status, "X"):
		return "unknown"
	case strings.HasPrefix(status, "B"):
		return "broken_pair"
	default:
		return "modified"
	}
}
