// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"sort"
)

// ChangedFile represents a file with changes.
type ChangedFile struct {
	Path            string
	Tracked         bool
	StagedPresent   bool
	UnstagedPresent bool
	Untracked       bool
}

// GetDirtyFiles returns all changed files for dirty mode.
func GetDirtyFiles(repoRoot string) ([]ChangedFile, error) {
	// Get staged files using NUL delimiter
	stagedOutput, err := RunGit(repoRoot, []string{"diff", "--cached", "--name-only", "-z"})
	if err != nil {
		return nil, err
	}
	stagedFiles := splitNULList(stagedOutput)

	// Get unstaged files using NUL delimiter
	unstagedOutput, err := RunGit(repoRoot, []string{"diff", "--name-only", "-z"})
	if err != nil {
		return nil, err
	}
	unstagedFiles := splitNULList(unstagedOutput)

	// Get untracked files using NUL delimiter
	untrackedOutput, err := RunGit(repoRoot, []string{"ls-files", "--others", "--exclude-standard", "-z"})
	if err != nil {
		return nil, err
	}
	untrackedFiles := splitNULList(untrackedOutput)

	// Build a map of all files with their status
	fileMap := make(map[string]*ChangedFile)

	// Process staged files
	for _, path := range stagedFiles {
		if path == "" {
			continue
		}
		if f, exists := fileMap[path]; exists {
			f.StagedPresent = true
		} else {
			fileMap[path] = &ChangedFile{
				Path:          path,
				Tracked:       true,
				StagedPresent: true,
			}
		}
	}

	// Process unstaged files
	for _, path := range unstagedFiles {
		if path == "" {
			continue
		}
		if f, exists := fileMap[path]; exists {
			f.UnstagedPresent = true
		} else {
			fileMap[path] = &ChangedFile{
				Path:            path,
				Tracked:         true,
				UnstagedPresent: true,
			}
		}
	}

	// Process untracked files
	for _, path := range untrackedFiles {
		if path == "" {
			continue
		}
		if _, exists := fileMap[path]; !exists {
			fileMap[path] = &ChangedFile{
				Path:            path,
				Untracked:       true,
				StagedPresent:   false,
				UnstagedPresent: true,
			}
		}
	}

	// Convert to slice and sort
	result := make([]ChangedFile, 0, len(fileMap))
	for _, f := range fileMap {
		result = append(result, *f)
	}

	// Sort: tracked first, then untracked, both alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].Tracked != result[j].Tracked {
			return result[i].Tracked
		}
		return result[i].Path < result[j].Path
	})

	return result, nil
}

// GetStagedFiles returns only staged changed files.
func GetStagedFiles(repoRoot string) ([]ChangedFile, error) {
	// Get staged files using NUL delimiter
	stagedOutput, err := RunGit(repoRoot, []string{"diff", "--cached", "--name-only", "-z"})
	if err != nil {
		return nil, err
	}

	stagedFiles := splitNULList(stagedOutput)
	result := make([]ChangedFile, 0, len(stagedFiles))

	for _, path := range stagedFiles {
		if path == "" {
			continue
		}
		result = append(result, ChangedFile{
			Path:          path,
			Tracked:       true,
			StagedPresent: true,
		})
	}

	// Sort alphabetically
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result, nil
}
