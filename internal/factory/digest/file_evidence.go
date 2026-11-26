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

// MaxGitCaptureBytes bounds how much stdout a single Git invocation
// may materialise.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 AMENDMENT-01 (A6):
// moving patch generation AHEAD of the size suppression means a
// subprocess now runs before the per-file budget is applied. A
// pathological range (for example the deletion of a 500 MB text
// file) would otherwise materialise the whole patch in memory
// before the 64 KiB evidence limit ever applied.
//
// The digest package executes Git through raw os/exec rather than the
// bounded internal/execution gateway, so the bound is applied HERE,
// at the digest's own runner boundary. No new generic execution
// framework is introduced.
//
// The limit is deliberately far above the per-file evidence budget
// (64 KiB) so an ordinary patch is never affected, while a patch that
// is already known to be unusable is refused instead of materialised.
//
// A capture that hits the limit is reported as an ERROR, never as a
// truncated patch. A partial patch must never be mistaken for the
// complete one.
const MaxGitCaptureBytes = 16 * 1024 * 1024

// realGitRunner implements gitRunner using the actual git binary.
type realGitRunner struct{}

// runGitBounded executes git with a bounded stdout capture.
//
// ACHIEVED CONTRACT: BOUNDED_CAPTURE.
//
// The RETAINED stdout payload is bounded by MaxGitCaptureBytes and the
// RETAINED stderr payload is bounded by 1 MiB. An oversized result is
// detected and reported as an ERROR, never as a truncated success.
//
// NOT BOUNDED (deliberately out of scope for this ACT):
//
//   - the subprocess OUTPUT GENERATION is not bounded: after the limit
//     is reached the writer keeps draining and discarding the
//     remainder, so Git still produces its full output;
//   - the subprocess EXECUTION TIME is not bounded by this code: there
//     is no deadline and no cancellation;
//   - the subprocess is NOT terminated on overflow.
//
// The predecessor's comment claimed that an oversized result causes
// the process to be terminated. That was never true of this
// implementation and was corrected here.
//
// Migrating the digest's Git execution onto the bounded
// internal/execution gateway is a separate architectural task and is
// NOT performed by this ACT.
func runGitBounded(repoRoot string, args []string,
	stdin string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out limitedBuffer
	out.limit = MaxGitCaptureBytes
	cmd.Stdout = &out
	var errBuf limitedBuffer
	errBuf.limit = 1 * 1024 * 1024
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if out.exceeded {
		return "", fmt.Errorf(
			"git %s exceeded the %d byte capture limit; "+
				"the output is NOT a complete result",
			strings.Join(args, " "), MaxGitCaptureBytes)
	}
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return out.String(), fmt.Errorf("git %s failed: %s",
				strings.Join(args, " "), errBuf.String())
		}
		return out.String(), fmt.Errorf("git %s failed: %w",
			strings.Join(args, " "), err)
	}
	return out.String(), nil
}

// limitedBuffer is a bounded stdout/stderr sink.
//
// It records whether the limit was exceeded so the caller can report
// an overflow instead of a truncated success.
type limitedBuffer struct {
	sb       strings.Builder
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.exceeded {
		// Discard the remainder, but keep reporting success so the
		// process is not killed by a broken pipe. The overflow flag
		// is what the caller acts on.
		return len(p), nil
	}
	remaining := b.limit - b.sb.Len()
	if remaining <= 0 {
		b.exceeded = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.sb.Write(p[:remaining])
		b.exceeded = true
		return len(p), nil
	}
	b.sb.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) String() string { return b.sb.String() }

func (realGitRunner) Run(repoRoot string, args []string) (string, error) {
	return runGitBounded(repoRoot, args, "")
}

func (realGitRunner) RunWithStdin(repoRoot string, args []string,
	input string) (string, error) {
	return runGitBounded(repoRoot, args, input)
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
