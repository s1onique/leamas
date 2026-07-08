// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/factory/redact"
)

// Options configures digest generation.
type Options struct {
	// RepoRoot is the absolute path to the Git repository root.
	RepoRoot string
	// Mode determines which changes to include.
	Mode Mode
	// Output is the path to write the digest file.
	Output string
	// Range is the commit range for ModeRange (e.g., "HEAD~1..HEAD").
	Range string
}

// ChangedFile represents a file with changes.
type ChangedFile struct {
	Path            string
	Tracked         bool
	StagedPresent   bool
	UnstagedPresent bool
	Untracked       bool
}

// Generate creates a targeted digest and returns it as a string.
func Generate(opts Options) (string, error) {
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		var err error
		repoRoot, err = DetectRepoRoot()
		if err != nil {
			return "", fmt.Errorf("failed to detect repo root: %w", err)
		}
	}

	mode := opts.Mode
	if mode == "" {
		mode = ModeAuto // default to auto mode
	}

	// Handle auto mode
	if mode == ModeAuto {
		resolved, err := ResolveAutoMode(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve auto mode: %w", err)
		}

		if resolved.Mode == ModeDirty {
			files, err := GetDirtyFiles(repoRoot)
			if err != nil {
				return "", fmt.Errorf("failed to get dirty files: %w", err)
			}
			return RenderDigestWithResolved(ModeDirty, repoRoot, files, resolved, false)
		}

		// Clean working tree: use range mode with HEAD~1..HEAD
		files, err := GetRangeFiles(repoRoot, resolved.Range)
		if err != nil {
			return "", fmt.Errorf("failed to get range files: %w", err)
		}
		return RenderRangeDigestWithResolved(repoRoot, files, resolved)
	}

	// Handle explicit modes
	switch mode {
	case ModeDirty:
		files, err := GetDirtyFiles(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to get dirty files: %w", err)
		}
		return RenderDigest(mode, repoRoot, files)
	case ModeStaged:
		files, err := GetStagedFiles(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to get staged files: %w", err)
		}
		return RenderDigest(mode, repoRoot, files)
	case ModeRange:
		if opts.Range == "" {
			return "", fmt.Errorf("ModeRange requires --range option")
		}
		files, err := GetRangeFiles(repoRoot, opts.Range)
		if err != nil {
			return "", fmt.Errorf("failed to get range files: %w", err)
		}
		return RenderRangeDigest(repoRoot, files, opts.Range)
	default:
		return "", fmt.Errorf("unsupported mode: %s", mode)
	}
}

// Write generates a digest and writes it to the output file.
// The digest content is redacted before writing to prevent secret exposure.
func Write(opts Options) error {
	content, err := Generate(opts)
	if err != nil {
		return err
	}

	// Redact secrets from digest output before writing
	content = redact.RedactDigest(content)

	// Create parent directory if needed
	dir := filepath.Dir(opts.Output)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write the digest
	if err := os.WriteFile(opts.Output, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write digest: %w", err)
	}

	return nil
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

// RenderDigest creates the markdown digest content.
func RenderDigest(mode Mode, repoRoot string, files []ChangedFile) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("# Targeted digest\n\n")
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Repo: %s\n", repoRoot))
	sb.WriteString(fmt.Sprintf("Mode: %s\n", mode))
	sb.WriteString("\n")

	// Changed files section
	sb.WriteString("## Changed files\n")
	if len(files) == 0 {
		sb.WriteString("No changed files found.\n")
	} else {
		for _, f := range files {
			if f.Tracked {
				stagedStr := "no"
				if f.StagedPresent {
					stagedStr = "yes"
				}
				unstagedStr := "no"
				if f.UnstagedPresent {
					unstagedStr = "yes"
				}
				sb.WriteString(fmt.Sprintf("%s  [tracked, staged present: %s, unstaged present: %s]\n",
					f.Path, stagedStr, unstagedStr))
			} else {
				sb.WriteString(fmt.Sprintf("%s  [untracked, staged present: no, unstaged present: yes]\n",
					f.Path))
			}
		}
	}
	sb.WriteString("\n")

	// Diffs section
	sb.WriteString("## Diffs\n")
	if len(files) == 0 {
		sb.WriteString("No diffs to show.\n")
	} else {
		for _, f := range files {
			fullPath := filepath.Join(repoRoot, f.Path)
			sb.WriteString(fmt.Sprintf("\n=== %s ===\n", f.Path))

			// Metadata
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

			// Content based on file type
			if f.Untracked {
				sb.WriteString("--- untracked file preview ---\n")
				preview, isBinary := PreviewFile(fullPath, MaxPreviewBytes, MaxPreviewLines)
				if isBinary {
					sb.WriteString("(binary file)\n")
				} else {
					sb.WriteString(preview)
				}
			} else {
				// Tracked file with staged changes
				if f.StagedPresent {
					sb.WriteString("--- staged diff ---\n")
					diff, err := RunGit(repoRoot, []string{"diff", "--cached", "--", f.Path})
					if err == nil && diff != "" {
						sb.WriteString(diff)
					}
					sb.WriteString("\n")
				}

				// Tracked file with unstaged changes
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

	sb.WriteString("\n## Workflow anchors\n")
	sb.WriteString("No workflow anchors configured.\n")

	return sb.String(), nil
}
