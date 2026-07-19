// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RenderChangedFilesAndDiffs renders the Changed files list and diff
// content for dirty/staged modes.
//
// Each entry carries an explicit Git kind (`A` / `M` / `D` / `T` /
// `R` / `C` / `U` / `?` / `X` / `B`) sourced from the structured
// parser. Paths are written through `PathEscape` so the section
// stays one record per line even when the original filename contains
// bytes that would otherwise break the rendering (tab, newline,
// backslash, control bytes). The staged/unstaged presence flags
// remain independent metadata that the renderer uses to attach the
// right patches.
func RenderChangedFilesAndDiffs(repoRoot string, files []ChangedFile) string {
	var sb strings.Builder

	sb.WriteString("## Changed files\n")
	if len(files) == 0 {
		sb.WriteString("No changed files found.\n")
	} else {
		for _, f := range files {
			kindStr := string(f.Kind)
			if f.Untracked {
				kindStr = StatusUntracked
			}
			escapedPath := PathEscape(f.Path)
			if f.Tracked {
				stagedStr := "no"
				if f.StagedPresent {
					stagedStr = "yes"
				}
				unstagedStr := "no"
				if f.UnstagedPresent {
					unstagedStr = "yes"
				}
				if f.OldPath != "" && f.OldPath != f.Path {
					sb.WriteString(fmt.Sprintf(
						"%s  [tracked, kind: %s, staged present: %s, unstaged present: %s, old path: %s]\n",
						escapedPath, kindStr, stagedStr, unstagedStr, PathEscape(f.OldPath),
					))
				} else {
					sb.WriteString(fmt.Sprintf(
						"%s  [tracked, kind: %s, staged present: %s, unstaged present: %s]\n",
						escapedPath, kindStr, stagedStr, unstagedStr,
					))
				}
			} else {
				sb.WriteString(fmt.Sprintf(
					"%s  [untracked, staged present: no, unstaged present: yes]\n",
					escapedPath,
				))
			}
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## Diffs\n")
	if len(files) == 0 {
		sb.WriteString("No diffs to show.\n")
	} else {
		for _, f := range files {
			fullPath := filepath.Join(repoRoot, f.Path)
			sb.WriteString(fmt.Sprintf("\n=== %s ===\n", PathEscape(f.Path)))

			if f.Tracked {
				stagedStr := "yes"
				if !f.StagedPresent {
					stagedStr = "no"
				}
				unstagedStr := "yes"
				if !f.UnstagedPresent {
					unstagedStr = "no"
				}
				sb.WriteString(fmt.Sprintf("Metadata: tracked, staged present: %s, unstaged present: %s\n",
					stagedStr, unstagedStr))
			} else {
				sb.WriteString("Metadata: untracked, staged present: no, unstaged present: yes\n")
			}
			sb.WriteString("\n")

			if f.Untracked {
				sb.WriteString("--- untracked file content ---\n")
				content, isBinary := ReadFileFull(fullPath)
				if isBinary {
					sb.WriteString("(binary file)\n")
				} else {
					sb.WriteString(content)
				}
			} else {
				if f.StagedPresent {
					sb.WriteString("--- staged diff ---\n")
					diff, err := RunGit(repoRoot, []string{"diff", "--cached", "--", f.Path})
					if err == nil && diff != "" {
						sb.WriteString(diff)
					}
					sb.WriteString("\n")
				}

				if f.UnstagedPresent {
					sb.WriteString("--- unstaged diff ---\n")
					diff, err := RunGit(repoRoot, []string{"diff", "--", f.Path})
					if err == nil && diff != "" {
						sb.WriteString(diff)
					}
				}
			}
		}
	}

	return sb.String()
}

// RenderRangeFileEvidence renders the Changed files list and diffs for
// range mode. Paths are escaped on render for the same reason as in
// the dirty/staged renderer.
func RenderRangeFileEvidence(repoRoot string, files []RangeFile, rangeSpec string) string {
	var sb strings.Builder

	sb.WriteString("## Changed files\n")
	if len(files) == 0 {
		sb.WriteString("No changed files found in range.\n")
	} else {
		for _, f := range files {
			sb.WriteString(fmt.Sprintf("%s  [%s]\n", PathEscape(f.Path), f.Status))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## Diffs\n")
	if len(files) == 0 {
		sb.WriteString("No diffs to show.\n")
	} else {
		for _, f := range files {
			sb.WriteString(fmt.Sprintf("\n=== %s ===\n", PathEscape(f.Path)))
			sb.WriteString(fmt.Sprintf("Status: %s\n\n", f.Status))

			diff, err := RunGit(repoRoot, []string{"diff", "--unified=3", rangeSpec, "--", f.Path})
			if err == nil && diff != "" {
				sb.WriteString(diff)
			} else {
				diff, err = RunGit(repoRoot, []string{"diff", "--unified=3", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "HEAD", "--", f.Path})
				if err == nil && diff != "" {
					sb.WriteString(diff)
				} else {
					sb.WriteString("(no diff available)\n")
				}
			}
		}
	}

	return sb.String()
}
