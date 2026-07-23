package closure

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/s1onique/leamas/internal/execution"
)

type gitCommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

type gitClient interface {
	Run(context.Context, string, ...string) gitCommandResult
}

type realGitClient struct{}

func (realGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	result, err := execution.RunGit(ctx, directory, args...)
	return gitCommandResult{Stdout: result.Stdout, Stderr: result.Stderr, ExitCode: result.ExitCode, Err: err}
}

type subjectSnapshot struct {
	RepositoryRoot      string
	SubjectCommitOID    string
	SubjectTreeOID      string
	HeadCommitOID       string
	HeadTreeOID         string
	Branch              string
	RemoteURL           string
	OriginMainCommitOID string
	AheadBy             *int
	BehindBy            *int
	Clean               bool
}

func snapshotSubject(ctx context.Context, git gitClient, directory, requestedSubject string) (subjectSnapshot, error) {
	if strings.TrimSpace(requestedSubject) == "" || strings.HasPrefix(requestedSubject, "-") || containsClosurePlaceholder(requestedSubject) {
		return subjectSnapshot{}, fmt.Errorf("requested subject is invalid")
	}
	root, err := runGitValue(ctx, git, directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return subjectSnapshot{}, fmt.Errorf("find repository root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return subjectSnapshot{}, fmt.Errorf("resolve repository root: %w", err)
	}
	subjectCommit, err := resolveGitObject(ctx, git, root, requestedSubject+"^{commit}")
	if err != nil {
		return subjectSnapshot{}, fmt.Errorf("resolve requested subject commit: %w", err)
	}
	subjectTree, err := resolveGitObject(ctx, git, root, requestedSubject+"^{tree}")
	if err != nil {
		return subjectSnapshot{}, fmt.Errorf("resolve requested subject tree: %w", err)
	}
	headCommit, err := resolveGitObject(ctx, git, root, "HEAD^{commit}")
	if err != nil {
		return subjectSnapshot{}, fmt.Errorf("resolve HEAD commit: %w", err)
	}
	headTree, err := resolveGitObject(ctx, git, root, "HEAD^{tree}")
	if err != nil {
		return subjectSnapshot{}, fmt.Errorf("resolve HEAD tree: %w", err)
	}
	if subjectCommit != headCommit {
		return subjectSnapshot{}, fmt.Errorf("HEAD %s does not equal requested subject %s", headCommit, subjectCommit)
	}
	if subjectTree != headTree {
		return subjectSnapshot{}, fmt.Errorf("HEAD tree %s does not equal requested subject tree %s", headTree, subjectTree)
	}
	status, err := runGitValue(ctx, git, root, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return subjectSnapshot{}, fmt.Errorf("inspect working tree: %w", err)
	}
	if status != "" {
		return subjectSnapshot{}, fmt.Errorf("subject worktree is dirty")
	}
	branch, err := runGitValue(ctx, git, root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || branch == "" {
		return subjectSnapshot{}, fmt.Errorf("subject HEAD must be attached to a branch")
	}
	snapshot := subjectSnapshot{
		RepositoryRoot:   root,
		SubjectCommitOID: subjectCommit,
		SubjectTreeOID:   subjectTree,
		HeadCommitOID:    headCommit,
		HeadTreeOID:      headTree,
		Branch:           branch,
		Clean:            true,
	}
	populateOptionalRemoteIdentity(ctx, git, &snapshot)
	return snapshot, nil
}

func resolveGitObject(ctx context.Context, git gitClient, root, expression string) (string, error) {
	value, err := runGitValue(ctx, git, root, "rev-parse", "--verify", "--end-of-options", expression)
	if err != nil {
		return "", err
	}
	if err := validateOID("Git object", value); err != nil {
		return "", err
	}
	return value, nil
}

func runGitValue(ctx context.Context, git gitClient, directory string, args ...string) (string, error) {
	result := git.Run(ctx, directory, args...)
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return "", fmt.Errorf("git %s failed (exit %d): %s", strings.Join(args, " "), result.ExitCode, sanitizeDiagnostic(detail))
	}
	return strings.TrimSpace(string(result.Stdout)), nil
}

func populateOptionalRemoteIdentity(ctx context.Context, git gitClient, snapshot *subjectSnapshot) {
	remote, err := runGitValue(ctx, git, snapshot.RepositoryRoot, "remote", "get-url", "origin")
	if err == nil {
		snapshot.RemoteURL = sanitizedRemoteURL(remote)
	}
	originMain, err := resolveGitObject(ctx, git, snapshot.RepositoryRoot, "origin/main^{commit}")
	if err != nil {
		return
	}
	snapshot.OriginMainCommitOID = originMain
	counts, err := runGitValue(ctx, git, snapshot.RepositoryRoot, "rev-list", "--left-right", "--count", "HEAD...origin/main")
	if err != nil {
		return
	}
	fields := strings.Fields(counts)
	if len(fields) != 2 {
		return
	}
	ahead, aheadErr := strconv.Atoi(fields[0])
	behind, behindErr := strconv.Atoi(fields[1])
	if aheadErr == nil && behindErr == nil && ahead >= 0 && behind >= 0 {
		snapshot.AheadBy = &ahead
		snapshot.BehindBy = &behind
	}
}

func sanitizedRemoteURL(raw string) string {
	if !strings.Contains(raw, "://") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.User = nil
	return parsed.String()
}

func workingTreeClean(ctx context.Context, git gitClient, root string) (bool, error) {
	status, err := runGitValue(ctx, git, root, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return false, err
	}
	return status == "", nil
}
