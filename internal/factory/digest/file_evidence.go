// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitRunner is the interface for executing Git commands.
type gitRunner interface {
	Run(repoRoot string, args []string) (string, error)
	// RunWithStdin is the F15 (CORRECTION02) batched primitive.
	// The default implementation provided by realGitRunner feeds
	// `input` to the git process on stdin and captures stdout. It
	// is used by `git cat-file --batch-check` to look up N blob
	// sizes in one process.
	RunWithStdin(repoRoot string, args []string,
		input string) (string, error)
}

// realGitRunner implements gitRunner using the actual git binary.
type realGitRunner struct{}

func (realGitRunner) Run(repoRoot string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), fmt.Errorf("git %s failed: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return string(output), fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

func (realGitRunner) RunWithStdin(repoRoot string, args []string,
	input string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), fmt.Errorf("git %s failed: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return string(output), fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

// RenderChangedFilesAndDiffs renders the Changed files list and diff
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
	return renderRangeFileEvidenceWithRunner(realGitRunner{}, repoRoot, files, rangeSpec)
}

// renderRangeFileEvidenceWithRunner renders range file evidence using the provided runner.
// This allows tests to inject a fake runner for controlled error injection.
func renderRangeFileEvidenceWithRunner(runner gitRunner, repoRoot string, files []RangeFile, rangeSpec string) string {
	return renderRangeFileEvidenceBoundedWithRunner(runner, repoRoot, files, rangeSpec, "")
}
