// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"sort"
	"strings"
)

// RangeFile represents a file changed in a commit range.
type RangeFile struct {
	Path   string
	From   string
	To     string
	Status string // "added", "modified", "deleted"
}

// GetRangeFiles returns files changed in the given revision range.
func GetRangeFiles(repoRoot, revRange string) ([]RangeFile, error) {
	// Get list of changed files with status using NUL delimiter
	output, err := RunGit(repoRoot, []string{"diff", "--name-status", "-z", revRange})
	if err != nil {
		return nil, err
	}

	parts := splitNULList(output)
	var files []RangeFile

	for i := 0; i < len(parts)-1; i += 2 {
		if parts[i] == "" {
			continue
		}
		status := parts[i]
		path := parts[i+1]

		var from, to string
		switch {
		case status == "A" || strings.HasPrefix(status, "A"):
			// Added: old side is /dev/null, new side is the file
			from = ""
			to = path
		case status == "D" || strings.HasPrefix(status, "D"):
			// Deleted: old side is the file, new side is /dev/null
			from = path
			to = ""
		case strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C"):
			// Renamed or copied: old name followed by new name
			if i+3 < len(parts) {
				from = path
				to = parts[i+3]
				i += 2
			}
		default:
			from = ""
			to = ""
		}

		files = append(files, RangeFile{
			Path:   path,
			From:   from,
			To:     to,
			Status: statusToHuman(status),
		})
	}

	// Deduplicate files and sort
	files = UniqueRangeFiles(files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// statusToHuman converts git status letter to human-readable string.
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
	default:
		return "modified"
	}
}
