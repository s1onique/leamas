// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	v2ClosureFileMode = "100644"
	v2ClosureName     = "Leamas Closure"
	v2ClosureEmail    = "closure@leamas.local"
)

type v2ClosureObjects struct {
	ManifestBlobOID string
	ReportBlobOID   string
	TreeOID         string
	CommitOID       string
}

func canonicalV2ManifestPath(actID string) string {
	return "docs/closure-manifests/" + actID + ".json"
}

func canonicalV2ReportPath(actID string) string {
	return "docs/close-reports/" + actID + ".md"
}

func buildV2ClosureObjects(ctx context.Context, git gitClient, repoRoot string, objectFormat ObjectFormat,
	subjectCommit, subjectTree, actID string, artifacts v2CanonicalArtifacts) (v2ClosureObjects, error) {
	manifestPath := canonicalV2ManifestPath(actID)
	reportPath := canonicalV2ReportPath(actID)
	if err := rejectExistingTreePaths(ctx, git, repoRoot, subjectTree, manifestPath, reportPath); err != nil {
		return v2ClosureObjects{}, err
	}
	manifestBlob, err := hashBlob(ctx, git, repoRoot, artifacts.ManifestBytes)
	if err != nil {
		return v2ClosureObjects{}, fmt.Errorf("hash manifest: %w", err)
	}
	reportBlob, err := hashBlob(ctx, git, repoRoot, artifacts.ReportBytes)
	if err != nil {
		return v2ClosureObjects{}, fmt.Errorf("hash report: %w", err)
	}
	if err := ValidateOIDWithFormat("manifest blob", manifestBlob, objectFormat); err != nil {
		return v2ClosureObjects{}, err
	}
	if err := ValidateOIDWithFormat("report blob", reportBlob, objectFormat); err != nil {
		return v2ClosureObjects{}, err
	}
	treeOID, err := buildV2ClosureTree(ctx, git, repoRoot, subjectTree, manifestPath, manifestBlob, reportPath, reportBlob)
	if err != nil {
		return v2ClosureObjects{}, err
	}
	if err := verifyV2ClosureTree(ctx, git, repoRoot, objectFormat, subjectTree, treeOID, manifestPath, manifestBlob, reportPath, reportBlob); err != nil {
		return v2ClosureObjects{}, err
	}
	commitOID, err := buildDeterministicClosureCommit(ctx, git, repoRoot, treeOID, subjectCommit, actID)
	if err != nil {
		return v2ClosureObjects{}, err
	}
	if err := ValidateOIDWithFormat("closure commit", commitOID, objectFormat); err != nil {
		return v2ClosureObjects{}, err
	}
	return v2ClosureObjects{ManifestBlobOID: manifestBlob, ReportBlobOID: reportBlob, TreeOID: treeOID, CommitOID: commitOID}, nil
}

func rejectExistingTreePaths(ctx context.Context, git gitClient, repoRoot, treeOID string, paths ...string) error {
	for _, path := range paths {
		result := git.Run(ctx, repoRoot, "ls-tree", treeOID, "--", path)
		if result.Err != nil || result.ExitCode != 0 {
			return fmt.Errorf("inspect subject path %s: %s", path, gitFailureDetail(result))
		}
		if len(result.Stdout) != 0 {
			return fmt.Errorf("canonical closure path already exists in subject tree: %s", path)
		}
	}
	return nil
}

func buildV2ClosureTree(ctx context.Context, git gitClient, repoRoot, subjectTree,
	manifestPath, manifestBlob, reportPath, reportBlob string) (treeOID string, err error) {
	indexFile, err := os.CreateTemp("", "leamas-closure-index-*")
	if err != nil {
		return "", fmt.Errorf("create alternate index: %w", err)
	}
	indexPath := indexFile.Name()
	if closeErr := indexFile.Close(); closeErr != nil {
		_ = os.Remove(indexPath)
		return "", fmt.Errorf("close alternate index: %w", closeErr)
	}
	if err := os.Remove(indexPath); err != nil {
		return "", fmt.Errorf("prepare alternate index: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(indexPath); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
			err = fmt.Errorf("remove alternate index: %w", removeErr)
			treeOID = ""
		}
	}()
	env := []string{"GIT_INDEX_FILE=" + filepath.Clean(indexPath)}
	if err := runGitWithEnvOK(ctx, git, repoRoot, env, "read-tree", subjectTree); err != nil {
		return "", err
	}
	if err := runGitWithEnvOK(ctx, git, repoRoot, env, "update-index", "--add", "--cacheinfo", v2ClosureFileMode, manifestBlob, manifestPath); err != nil {
		return "", err
	}
	if err := runGitWithEnvOK(ctx, git, repoRoot, env, "update-index", "--add", "--cacheinfo", v2ClosureFileMode, reportBlob, reportPath); err != nil {
		return "", err
	}
	result := git.RunWithEnv(ctx, repoRoot, env, "write-tree")
	if result.Err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("git write-tree failed: %s", gitFailureDetail(result))
	}
	return strings.TrimSpace(string(result.Stdout)), nil
}

func runGitWithEnvOK(ctx context.Context, git gitClient, repoRoot string, env []string, args ...string) error {
	result := git.RunWithEnv(ctx, repoRoot, env, args...)
	if result.Err != nil || result.ExitCode != 0 {
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), gitFailureDetail(result))
	}
	return nil
}

func verifyV2ClosureTree(ctx context.Context, git gitClient, repoRoot string, format ObjectFormat,
	subjectTree, closureTree, manifestPath, manifestBlob, reportPath, reportBlob string) error {
	if err := ValidateOIDWithFormat("closure tree", closureTree, format); err != nil {
		return err
	}
	result := git.Run(ctx, repoRoot, "diff-tree", "--no-commit-id", "--name-status", "-r", subjectTree, closureTree)
	if result.Err != nil || result.ExitCode != 0 {
		return fmt.Errorf("diff subject and closure trees: %s", gitFailureDetail(result))
	}
	expected := map[string]string{manifestPath: manifestBlob, reportPath: reportBlob}
	lines := strings.Split(strings.TrimSuffix(string(result.Stdout), "\n"), "\n")
	if len(lines) != len(expected) {
		return fmt.Errorf("closure tree must contain exactly two additions, got %d changes", len(lines))
	}
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 2 || fields[0] != "A" {
			return fmt.Errorf("closure tree contains non-addition change %q", line)
		}
		blob, ok := expected[fields[1]]
		if !ok {
			return fmt.Errorf("closure tree contains unexpected path %q", fields[1])
		}
		entry, err := readV2TreeEntry(ctx, git, repoRoot, closureTree, fields[1])
		if err != nil {
			return err
		}
		if entry.mode != v2ClosureFileMode || entry.objectType != "blob" || entry.oid != blob {
			return fmt.Errorf("closure tree entry mismatch for %s", fields[1])
		}
	}
	return nil
}

type v2TreeEntry struct{ mode, objectType, oid string }

func readV2TreeEntry(ctx context.Context, git gitClient, repoRoot, treeOID, path string) (v2TreeEntry, error) {
	result := git.Run(ctx, repoRoot, "ls-tree", treeOID, "--", path)
	if result.Err != nil || result.ExitCode != 0 {
		return v2TreeEntry{}, fmt.Errorf("inspect closure tree path %s: %s", path, gitFailureDetail(result))
	}
	fields := strings.Fields(string(result.Stdout))
	if len(fields) != 4 || fields[3] != path {
		return v2TreeEntry{}, fmt.Errorf("invalid closure tree entry for %s", path)
	}
	return v2TreeEntry{mode: fields[0], objectType: fields[1], oid: fields[2]}, nil
}

func buildDeterministicClosureCommit(ctx context.Context, git gitClient, repoRoot, closureTree, subjectCommit, actID string) (string, error) {
	result := git.Run(ctx, repoRoot, "show", "-s", "--format=%ct", subjectCommit)
	if result.Err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("read subject committer epoch: %s", gitFailureDetail(result))
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(string(result.Stdout)), 10, 64)
	if err != nil || epoch == int64(^uint64(0)>>1) {
		return "", fmt.Errorf("parse subject committer epoch: %q", strings.TrimSpace(string(result.Stdout)))
	}
	date := fmt.Sprintf("%d +0000", epoch+1)
	env := []string{
		"GIT_AUTHOR_NAME=" + v2ClosureName, "GIT_AUTHOR_EMAIL=" + v2ClosureEmail, "GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + v2ClosureName, "GIT_COMMITTER_EMAIL=" + v2ClosureEmail, "GIT_COMMITTER_DATE=" + date,
	}
	message := "chore(closure): close " + actID + "\n"
	commit := git.RunWithStdinAndEnv(ctx, repoRoot, message, env,
		"commit-tree", closureTree, "-p", subjectCommit)
	if commit.Err != nil || commit.ExitCode != 0 {
		return "", fmt.Errorf("git commit-tree failed: %s", gitFailureDetail(commit))
	}
	return strings.TrimSpace(string(commit.Stdout)), nil
}

func gitFailureDetail(result gitCommandResult) string {
	detail := strings.TrimSpace(string(result.Stderr))
	if detail == "" && result.Err != nil {
		detail = result.Err.Error()
	}
	return sanitizeDiagnostic(detail)
}
