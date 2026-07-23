package closure

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var remoteNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)

func verifyRemotePublication(
	ctx context.Context,
	git gitClient,
	closure committedClosure,
	tagName string,
	remote string,
	localTagOID string,
) (bool, string, error) {
	if !remoteNamePattern.MatchString(remote) || strings.Contains(remote, "..") || strings.Contains(remote, "//") {
		return false, "", fmt.Errorf("invalid configured remote name %q", remote)
	}
	branchRef := "refs/heads/" + closure.Manifest.Repository.Branch
	tagRef := "refs/tags/" + tagName
	result := git.Run(ctx, closure.RepositoryRoot,
		"ls-remote", "--heads", "--tags", remote, branchRef, tagRef, tagRef+"^{}")
	if result.Err != nil || result.ExitCode != 0 {
		return false, "remote refs unavailable; publication pending", nil
	}
	refs := parseRemoteRefs(result.Stdout)
	remoteBranch := refs[branchRef]
	remoteTag := refs[tagRef]
	remotePeeled := refs[tagRef+"^{}"]
	if remoteTag == "" || remotePeeled == "" || remoteBranch == "" {
		return false, "remote branch or annotated tag is not advertised; publication pending", nil
	}
	if remoteTag != localTagOID {
		return false, "", fmt.Errorf("remote tag object OID does not match local immutable tag object")
	}
	if remotePeeled != closure.TargetCommitOID {
		return false, "", fmt.Errorf("remote peeled tag target does not match closure commit")
	}
	contains, known := remoteBranchContainsClosure(ctx, git, closure.RepositoryRoot, remoteBranch, closure.TargetCommitOID)
	if !known || !contains {
		return false, "remote branch containment is not proven; publication pending", nil
	}
	return true, "", nil
}

func parseRemoteRefs(data []byte) map[string]string {
	refs := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !oidPattern.MatchString(fields[0]) {
			continue
		}
		refs[fields[1]] = fields[0]
	}
	return refs
}

func remoteBranchContainsClosure(ctx context.Context, git gitClient, root, remoteBranch, closureCommit string) (bool, bool) {
	if remoteBranch == closureCommit {
		return true, true
	}
	result := git.Run(ctx, root, "merge-base", "--is-ancestor", closureCommit, remoteBranch)
	if result.Err == nil && result.ExitCode == 0 {
		return true, true
	}
	if result.ExitCode == 1 {
		return false, true
	}
	return false, false
}
