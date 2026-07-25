// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// v2ExpectedTransaction captures the exact identities that every successful or
// recovery path derives before destructive mutation or exit 0.
type v2ExpectedTransaction struct {
	Artifacts     v2CanonicalArtifacts
	Objects       v2ClosureObjects
	EvidenceHash  string
	FreezeCommit  string
	SubjectCommit string
	Tag           v2TagObject
	CommitObject  gitObject
	TagObject     gitObject
}

type gitObject struct {
	OID  string
	Type string
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, fragment := range []string{
		"Needed a single revision", "unknown revision", "not found",
		"not a tree object", "does not exist",
	} {
		if strings.Contains(msg, fragment) {
			return true
		}
	}
	return false
}

// classifyV2TransactionState is filesystem-free. Its evidence input is the
// invocation's already-qualified snapshot; it may inspect Git refs and status
// but must not reopen E.
func classifyV2TransactionState(ctx context.Context, git gitClient, repoRoot, subject, headCommit, branch string,
	expected v2ExpectedTransaction, evidence v2EvidenceSnapshot) (v2TransactionState, error) {
	branchOID, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "refs/heads/"+branch)
	if err != nil {
		if isNotFoundError(err) {
			branchOID = ""
		} else {
			return v2StateInvalid, fmt.Errorf("resolve branch ref: %w", err)
		}
	}

	_, tagExists, err := tagRefOID(ctx, git, repoRoot, expected.Tag.Name)
	if err != nil {
		return v2StateInvalid, fmt.Errorf("resolve tag ref: %w", err)
	}
	if branchOID == subject && headCommit == subject && !tagExists && !evidence.Present {
		return v2StateNew, nil
	}
	if branchOID == subject && headCommit == subject && !tagExists && evidence.Present {
		return v2StatePrepared, nil
	}
	if !evidence.Present || !tagExists || expected.CommitObject.OID == "" || expected.TagObject.OID == "" {
		return v2StateInvalid, nil
	}

	tagPeeled, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "refs/tags/"+expected.Tag.Name+"^{commit}")
	if err != nil {
		return v2StateInvalid, fmt.Errorf("resolve tag peeled target: %w", err)
	}
	tagObject, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "refs/tags/"+expected.Tag.Name+"^{tag}")
	if err != nil {
		return v2StateInvalid, fmt.Errorf("resolve tag object: %w", err)
	}
	if branchOID != expected.CommitObject.OID || headCommit != expected.CommitObject.OID ||
		tagPeeled != expected.CommitObject.OID || tagObject != expected.TagObject.OID {
		return v2StateInvalid, nil
	}
	clean, err := workingTreeClean(ctx, git, repoRoot)
	if err != nil {
		return v2StateInvalid, fmt.Errorf("inspect worktree: %w", err)
	}
	if clean {
		return v2StateVerified, nil
	}
	return v2StateRefsCommittedNeedsConvergence, nil
}

// classifyV2OrchestratorState reconstructs C/T from the caller's sole evidence
// snapshot, then delegates to the filesystem-free classifier.
func classifyV2OrchestratorState(ctx context.Context, git gitClient, repoRoot string, plan Plan,
	subject, headCommit, branch, freezeCommit, freezeTree, subjectTree, planBlobOID string,
	authoritativeBytes []byte, objectFormat ObjectFormat, evidence v2EvidenceSnapshot) (v2ExpectedTransaction, v2TransactionState, error) {
	expected := v2ExpectedTransaction{
		FreezeCommit:  freezeCommit,
		SubjectCommit: subject,
		Tag:           v2TagObject{Name: canonicalV2TagName(plan.ActID), ActID: plan.ActID},
	}
	if evidence.Present {
		reconstructed, err := reconstructV2ExpectedTransaction(ctx, v2ExpectedInput{
			Plan: plan, AuthoritativeBytes: authoritativeBytes, PlanBlobOID: planBlobOID,
			FreezeCommit: freezeCommit, FreezeTree: freezeTree,
			SubjectCommit: subject, SubjectTree: subjectTree,
			Runner: evidence.Runtime.Runner, Checks: evidence.Runtime.Checks,
			Patch: evidence.Runtime.PatchHygiene, Closure: evidence.Runtime.ClosurePolicy,
			EvidenceIndexHash: evidence.IndexHash, RepositoryRoot: repoRoot,
			Git: git, ObjectFormat: objectFormat,
		})
		if err != nil {
			return expected, v2StateInvalid, fmt.Errorf("reconstruct expected transaction: %w", err)
		}
		expected = reconstructed
	}
	txState, err := classifyV2TransactionState(ctx, git, repoRoot, subject, headCommit, branch, expected, evidence)
	return expected, txState, err
}

func resolveBranchFromHEAD(ctx context.Context, git gitClient, repoRoot string) (string, error) {
	branch, err := runGitValue(ctx, git, repoRoot, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	if branch == "" {
		return "", errors.New("HEAD is not attached to a branch")
	}
	return branch, nil
}
