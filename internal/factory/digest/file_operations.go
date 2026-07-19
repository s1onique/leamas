// Package digest provides targeted digest generation for Git repositories.
// This file declares the `ChangedFile` data model and the two top-level
// collectors that gather tracked and untracked changes for dirty/staged
// digest generation.
//
// The collectors do not infer the change kind from boolean presence.
// They obtain the kind directly from `git diff --name-status -z` and
// parse it through ParseGitStatusRecords. The presence flags remain on
// the struct for diff rendering metadata (staged vs unstaged patch).
package digest

import (
	"fmt"
	"sort"
)

// RenameSimilarityThreshold is the similarity index (in percent) used
// for both `--find-renames` and `--find-copies` detection. Git's
// rename default is 50%; Git's `--find-copies` default is also 50% but
// observed to displace rename detection in mixed "rename + worktree
// edit" scenarios when set higher than the rename threshold. We pin
// both to 30% so the common "rename then a small edit" case still
// renders `R`. Tests reconcile the digest manifest against `git diff
// --find-renames=<n>% --find-copies=<n>%` for the same repo so the
// oracle matches.
const RenameSimilarityThreshold = 30

// ChangedFile represents a single path that participates in a dirty or
// staged digest.
//
// `Kind` is the authoritative change kind as reported by Git's
// `--name-status -z` output. `Path` is the post-change path; for
// renames/copies the pre-change path is in `OldPath`. The presence
// fields describe whether the same path appears in the staged and/or
// unstaged side independently of its manifest classification.
//
// Callers must populate `Kind` directly. The presence flags describe
// whether staged/unstaged diffs exist for the same path so the diff
// renderer knows which patches to attach.
type ChangedFile struct {
	Path            string
	OldPath         string
	Kind            ChangeKind
	Tracked         bool
	StagedPresent   bool
	UnstagedPresent bool
	Untracked       bool
}

// detectArgs builds the git-arg fragment that selects rename/copy
// detection at the digest's similarity threshold. Centralised so the
// staged and dirty collectors stay aligned.
func detectArgs() []string {
	return []string{
		fmt.Sprintf("--find-renames=%d%%", RenameSimilarityThreshold),
		fmt.Sprintf("--find-copies=%d%%", RenameSimilarityThreshold),
	}
}

// GetDirtyFiles returns all changed files for dirty mode.
//
// The manifest status for tracked files describes the net change
// relative to HEAD, obtained from `git diff --name-status -z HEAD --`
// with rename/copy detection enabled at RenameSimilarityThreshold.
// Staged/unstaged presence is recorded independently from additional
// NUL-delimited `git diff` invocations so the diff renderer can still
// emit staged and unstaged patches per path. Untracked files come
// from `git ls-files --others --exclude-standard -z`.
//
// For unborn HEAD (no commits), the empty tree SHA is used as the
// diff base, preserving prior behaviour for initial-commit repositories.
func GetDirtyFiles(repoRoot string) ([]ChangedFile, error) {
	headRef, err := resolveDirtyDiffBaseRef(repoRoot)
	if err != nil {
		return nil, err
	}

	args := []string{"diff", "--name-status", "-z"}
	args = append(args, detectArgs()...)
	args = append(args, headRef, "--")

	diffOutput, err := RunGit(repoRoot, args)
	if err != nil {
		return nil, fmt.Errorf("dirty diff: %w", err)
	}
	changes, err := ParseGitStatusRecords(diffOutput)
	if err != nil {
		return nil, fmt.Errorf("parse dirty diff: %w", err)
	}

	// Staged presence (paths in the index relative to HEAD).
	stagedOutput, err := RunGit(repoRoot, []string{"diff", "--cached", "--name-only", "-z"})
	if err != nil {
		return nil, fmt.Errorf("dirty staged presence: %w", err)
	}
	stagedPresence := indexNULPaths(stagedOutput)

	// Unstaged presence (paths in the worktree relative to the index).
	unstagedOutput, err := RunGit(repoRoot, []string{"diff", "--name-only", "-z"})
	if err != nil {
		return nil, fmt.Errorf("dirty unstaged presence: %w", err)
	}
	unstagedPresence := indexNULPaths(unstagedOutput)

	// Untracked files.
	untrackedOutput, err := RunGit(repoRoot, []string{"ls-files", "--others", "--exclude-standard", "-z"})
	if err != nil {
		return nil, fmt.Errorf("dirty untracked: %w", err)
	}
	untracked := splitNULList(untrackedOutput)

	// Compose: tracked paths come from the net diff; untracked paths
	// come from `ls-files`. Duplicate untracked paths (which can only
	// happen if Git returned them twice or a path has unusual
	// collisions) are filtered out so each path appears once.
	fileMap := make(map[string]*ChangedFile, len(changes)+len(untracked))

	for _, ch := range changes {
		path := ch.Path
		if path == "" {
			continue
		}
		fileMap[path] = &ChangedFile{
			Path:            path,
			OldPath:         ch.OldPath,
			Kind:            ch.Kind,
			Tracked:         true,
			StagedPresent:   stagedPresence[path],
			UnstagedPresent: unstagedPresence[path],
		}
	}

	for _, p := range untracked {
		if p == "" {
			continue
		}
		if _, exists := fileMap[p]; exists {
			continue
		}
		fileMap[p] = &ChangedFile{
			Path:            p,
			Kind:            KindUntracked,
			Untracked:       true,
			UnstagedPresent: true, // untracked paths are worktree-resident
		}
	}

	result := make([]ChangedFile, 0, len(fileMap))
	for _, f := range fileMap {
		result = append(result, *f)
	}

	// Deterministic order: tracked paths first, then untracked paths,
	// both sorted lexicographically by path.
	sort.Slice(result, func(i, j int) bool {
		if result[i].Untracked != result[j].Untracked {
			return !result[i].Untracked
		}
		return result[i].Path < result[j].Path
	})

	return result, nil
}

// GetStagedFiles returns only staged changed files for staged mode.
//
// The change kind is obtained directly from
// `git diff --cached --name-status -z --find-renames=<n>% --find-copies=<n>%`.
// Git defaults to HEAD internally so the result is equivalent to
// passing HEAD explicitly in normal repositories. For unborn branches
// Git implicitly compares against the empty tree, which preserves the
// existing behaviour for initial-commit repositories (newly added
// files render as `A`).
func GetStagedFiles(repoRoot string) ([]ChangedFile, error) {
	args := []string{
		"diff", "--cached", "--name-status", "-z",
	}
	args = append(args, detectArgs()...)

	stagedOutput, err := RunGit(repoRoot, args)
	if err != nil {
		return nil, fmt.Errorf("staged diff: %w", err)
	}
	changes, err := ParseGitStatusRecords(stagedOutput)
	if err != nil {
		return nil, fmt.Errorf("parse staged diff: %w", err)
	}

	result := make([]ChangedFile, 0, len(changes))
	for _, ch := range changes {
		if ch.Path == "" {
			continue
		}
		result = append(result, ChangedFile{
			Path:          ch.Path,
			OldPath:       ch.OldPath,
			Kind:          ch.Kind,
			Tracked:       true,
			StagedPresent: true,
		})
	}

	// Deterministic order: sort by the post-change path.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result, nil
}

// resolveDirtyDiffBaseRef returns the rev to use as the diff base for
// the dirty-mode net-status query.
//
// A literal HEAD reference is preferred because rename/copy detection
// compares against the existing commit content. For unborn branches,
// Git's empty-tree SHA (4b825dc642cb6eb9a060e54bf8d69288fbee4904) is
// used so the command still runs and `A` is reported for new files.
func resolveDirtyDiffBaseRef(repoRoot string) (string, error) {
	if _, err := RunGit(repoRoot, []string{"rev-parse", "--verify", "HEAD"}); err == nil {
		return "HEAD", nil
	}
	const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	if _, err := RunGit(repoRoot, []string{"cat-file", "-e", emptyTree}); err != nil {
		return "", fmt.Errorf("HEAD is unborn and the empty tree SHA is unavailable: %w", err)
	}
	return emptyTree, nil
}

// indexNULPaths parses a NUL-delimited `git diff --name-only -z` stream
// into a presence set keyed by path. Empty entries are ignored.
func indexNULPaths(output string) map[string]bool {
	out := make(map[string]bool)
	for _, p := range splitNULList(output) {
		if p == "" {
			continue
		}
		out[p] = true
	}
	return out
}
