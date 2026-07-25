// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// runClosureV2WithDependencies is the main v2 closure transaction runner.
//
// Order of operations (authority-first, no destructive side effects
// before verification):
//
//  1. Resolve repository root.
//  2. Load candidate plan bytes (filesystem); resolve relative paths
//     against repoRoot.
//  3. Bind exact supplied path to canonical repository path.
//  4. Derive S and F from the supplied subject.
//  5. Read authoritative plan bytes from F via git cat-file.
//  6. Compare candidate bytes with authoritative bytes; fail if different.
//  7. Parse and validate the authoritative bytes.
//  8. Resolve attached branch and HEAD.
//  9. Two-stage classification:
//     9a. Inspect branch, HEAD, tag presence, final-E presence.
//     9b. If final E is present, read qualified evidence once,
//     reconstruct exact C/T, then classify fully.
//  10. NEW → require clean worktree; enforce runner identity; create
//     evidence staging; execute checks; build C/E/T in staging;
//     publish E once; publish refs; bounded convergence; exact
//     verifier.
//  11. PREPARED → enforce runner identity; reconstruct C/T;
//     publish refs; bounded convergence; exact verifier.
//  12. REFS_COMMITTED_NEEDS_CONVERGENCE → enforce runner identity;
//     bounded convergence; exact verifier.
//  13. VERIFIED → enforce runner identity; exact verifier (no
//     mutation).
//  14. INVALID → fail.
func runClosureV2WithDependencies(ctx context.Context, options RunV2Options, deps v2Dependencies) (*TransactionResult, error) {
	repoRoot, err := runGitValue(ctx, deps.Git, options.RepoDirectory, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("find repository root: %w", err)
	}

	objectFormat, err := DetectStorageFormat(ctx, deps.Git, repoRoot)
	if err != nil {
		return nil, fmt.Errorf("detect object format: %w", err)
	}

	if options.PlanPath == "" {
		return nil, fmt.Errorf("--plan is required")
	}
	if options.Subject == "" {
		return nil, fmt.Errorf("--subject is required")
	}

	suppliedPath := options.PlanPath
	if !filepath.IsAbs(suppliedPath) {
		suppliedPath = filepath.Join(repoRoot, suppliedPath)
	}
	candidateBytes, err := os.ReadFile(suppliedPath)
	if err != nil {
		return nil, fmt.Errorf("read candidate plan: %w", err)
	}

	candidatePlan, _, err := parsePlanBytes(candidateBytes)
	if err != nil {
		return nil, fmt.Errorf("parse candidate plan: %w", err)
	}
	if !actIDPattern.MatchString(candidatePlan.ActID) || containsClosurePlaceholder(candidatePlan.ActID) {
		return nil, fmt.Errorf("candidate plan has invalid act_id %q", candidatePlan.ActID)
	}

	canonicalRelPath, err := bindExactPlanPath(repoRoot, candidatePlan.ActID, options.PlanPath)
	if err != nil {
		return nil, err
	}

	subjectCommit, err := resolveGitObjectWithFormat(ctx, deps.Git, repoRoot, options.Subject+"^{commit}", objectFormat)
	if err != nil {
		return nil, fmt.Errorf("resolve subject commit: %w", err)
	}
	if err := ValidateOIDWithFormat("subject commit", subjectCommit, objectFormat); err != nil {
		return nil, err
	}
	freezeCommit, err := verifySingleParent(ctx, deps.Git, repoRoot, subjectCommit, objectFormat)
	if err != nil {
		return nil, fmt.Errorf("subject must be a non-merge single-parent commit: %w", err)
	}
	if err := ValidateOIDWithFormat("freeze commit", freezeCommit, objectFormat); err != nil {
		return nil, err
	}

	subjectTree, err := resolveGitObjectWithFormat(ctx, deps.Git, repoRoot, subjectCommit+"^{tree}", objectFormat)
	if err != nil {
		return nil, fmt.Errorf("resolve subject tree: %w", err)
	}
	freezeTree, err := resolveGitObjectWithFormat(ctx, deps.Git, repoRoot, freezeCommit+"^{tree}", objectFormat)
	if err != nil {
		return nil, fmt.Errorf("resolve freeze tree: %w", err)
	}

	planBlobOID, err := bindExactPlanBytes(ctx, deps.Git, repoRoot, objectFormat, canonicalRelPath, freezeCommit, subjectCommit, candidateBytes)
	if err != nil {
		return nil, err
	}
	authoritativeBytes, err := readBlobBytesViaGit(ctx, deps.Git, repoRoot, planBlobOID)
	if err != nil {
		return nil, fmt.Errorf("read authoritative plan bytes: %w", err)
	}

	authoritativePlan, _, err := parsePlanBytes(authoritativeBytes)
	if err != nil {
		return nil, fmt.Errorf("parse authoritative plan: %w", err)
	}
	if err := ValidatePlan(authoritativePlan); err != nil {
		return nil, fmt.Errorf("validate authoritative plan: %w", err)
	}
	if err := validateV2Plan(authoritativePlan); err != nil {
		return nil, fmt.Errorf("authoritative plan validation: %w", err)
	}
	plan := authoritativePlan

	branch, err := resolveBranchFromHEAD(ctx, deps.Git, repoRoot)
	if err != nil {
		return nil, fmt.Errorf("subject HEAD must be attached to a branch: %w", err)
	}

	headCommit, err := runGitValue(ctx, deps.Git, repoRoot, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return nil, fmt.Errorf("get HEAD commit: %w", err)
	}

	evidenceDir := evidenceDirectoryPath(repoRoot, plan.ActID, subjectCommit)

	enforceCurrentRunner := func() (RunnerIdentity, error) {
		runnerIdentity, err := deps.Runner.Identity()
		if err != nil {
			return RunnerIdentity{}, fmt.Errorf("get runner identity: %w", err)
		}
		actualHash, err := deps.RunningBinarySHA256()
		if err != nil {
			return RunnerIdentity{}, fmt.Errorf("compute running binary SHA256: %w", err)
		}
		if err := EnforceRunnerAuthority(plan.RunnerAuthority, runnerIdentity, actualHash, subjectCommit, subjectTree); err != nil {
			return RunnerIdentity{}, err
		}
		return runnerIdentity, nil
	}

	// Capture only evidence presence first. Recovery authority is enforced
	// before E is opened or deterministic object reconstruction can write.
	evidencePresent, err := v2EvidencePresent(evidenceDir)
	if err != nil {
		return nil, err
	}
	evidence := v2EvidenceSnapshot{Present: evidencePresent}
	if evidencePresent {
		currentRunner, authorityErr := enforceCurrentRunner()
		if authorityErr != nil {
			return nil, authorityErr
		}
		evidence, err = readQualifiedV2Evidence(evidenceDir, plan.ActID, subjectCommit)
		if err != nil {
			return nil, fmt.Errorf("read qualified evidence: %w", err)
		}
		if err := validateV2EvidenceAuthority(plan, subjectTree, currentRunner, branch, evidence); err != nil {
			return nil, fmt.Errorf("authorize recovery evidence: %w", err)
		}
	}

	expected, txState, err := classifyV2OrchestratorState(ctx, deps.Git, repoRoot, plan, subjectCommit, headCommit,
		branch, freezeCommit, freezeTree, subjectTree, planBlobOID, authoritativeBytes, objectFormat, evidence)
	if err != nil {
		return nil, err
	}

	switch txState {
	case v2StateVerified, v2StateRefsCommittedNeedsConvergence:
		if txState == v2StateRefsCommittedNeedsConvergence {
			if err := boundedConverge(ctx, deps.Git, repoRoot, expected); err != nil {
				return nil, fmt.Errorf("bounded convergence: %w", err)
			}
		}
		return deps.VerifyExisting(ctx, deps.Git, repoRoot, evidenceDir, expected, evidence)

	case v2StatePrepared:
		if err := publishV2Refs(ctx, deps.Git, repoRoot, branch, expected.Tag.Name, subjectCommit,
			expected.CommitObject.OID, expected.TagObject.OID); err != nil {
			return nil, fmt.Errorf("publish refs: %w", err)
		}
		if err := boundedConverge(ctx, deps.Git, repoRoot, expected); err != nil {
			return nil, fmt.Errorf("bounded convergence: %w", err)
		}
		return deps.VerifyExisting(ctx, deps.Git, repoRoot, evidenceDir, expected, evidence)

	case v2StateInvalid:
		return nil, fmt.Errorf("HEAD is not a valid closure candidate")
	}

	// 10. NEW path.
	worktreeClean, err := workingTreeClean(ctx, deps.Git, repoRoot)
	if err != nil {
		return nil, fmt.Errorf("check worktree: %w", err)
	}
	if !worktreeClean {
		return nil, fmt.Errorf("worktree must be clean")
	}

	runnerIdentity, err := enforceCurrentRunner()
	if err != nil {
		return nil, err
	}

	stagingDir, err := createEvidenceStagingDirectory(repoRoot, plan.ActID, subjectCommit)
	if err != nil {
		return nil, fmt.Errorf("create evidence directory: %w", err)
	}

	finalizeInput := v2FinalizeInput{
		Dependencies:       deps,
		RepositoryRoot:     repoRoot,
		ObjectFormat:       objectFormat,
		Plan:               plan,
		CanonicalPlanPath:  canonicalRelPath,
		AuthoritativeBytes: authoritativeBytes,
		PlanBlobOID:        planBlobOID,
		FreezeCommit:       freezeCommit,
		FreezeTree:         freezeTree,
		SubjectCommit:      subjectCommit,
		SubjectTree:        subjectTree,
		Branch:             branch,
		EvidenceDirectory:  stagingDir,
		Runner:             runnerIdentity,
		RunnerAuthority:    plan.RunnerAuthority,
		Checks:             nil,
		CheckEvidence:      nil,
		Patch:              policyEvaluation[PatchHygiene]{},
		Closure:            policyEvaluation[ClosurePolicyResult]{},
	}

	checkResults, checkEvidence, err := deps.RunChecks(ctx, plan, deps, repoRoot, stagingDir, subjectTree)
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("execute checks: %w", err)
	}

	patchOutcome, err := deps.EvaluatePatchPolicy(ctx, deps.Git, repoRoot, plan, freezeCommit, subjectCommit)
	if err != nil {
		return nil, recordFailedPolicyDiagnostics(stagingDir, plan.ActID, patchOutcome.Diagnostics, nil, err)
	}
	if !patchOutcome.Passed {
		return nil, recordFailedPolicyDiagnostics(stagingDir, plan.ActID, patchOutcome.Diagnostics, nil,
			fmt.Errorf("required patch hygiene failed"))
	}
	closureOutcome, err := deps.EvaluateClosurePolicy(ctx, deps.Git, repoRoot, plan, subjectCommit)
	if err != nil {
		return nil, recordFailedPolicyDiagnostics(stagingDir, plan.ActID, patchOutcome.Diagnostics, closureOutcome.Diagnostics, err)
	}
	if !closureOutcome.Passed {
		return nil, recordFailedPolicyDiagnostics(stagingDir, plan.ActID, patchOutcome.Diagnostics, closureOutcome.Diagnostics,
			fmt.Errorf("required closure policy failed"))
	}

	finalizeInput.Checks = checkResults
	finalizeInput.CheckEvidence = checkEvidence
	finalizeInput.Patch = patchOutcome
	finalizeInput.Closure = closureOutcome

	_, finalizeErr := deps.FinalizeNew(ctx, finalizeInput)
	if finalizeErr == nil {
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("finalizer must return PublicationIncompleteError after building C/E/T")
	}
	var incomplete *PublicationIncompleteError
	if !errors.As(finalizeErr, &incomplete) {
		_ = os.RemoveAll(stagingDir)
		return nil, finalizeErr
	}

	evidence, err = readQualifiedV2Evidence(stagingDir, plan.ActID, subjectCommit)
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("read qualified staging evidence: %w", err)
	}
	if err := validateV2EvidenceAuthority(plan, subjectTree, runnerIdentity, branch, evidence); err != nil {
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("authorize staging evidence: %w", err)
	}
	reconstructed, err := reconstructV2ExpectedTransaction(ctx, v2ExpectedInput{
		Plan: plan, AuthoritativeBytes: authoritativeBytes, PlanBlobOID: planBlobOID,
		FreezeCommit: freezeCommit, FreezeTree: freezeTree,
		SubjectCommit: subjectCommit, SubjectTree: subjectTree,
		Runner: evidence.Runtime.Runner, Checks: evidence.Runtime.Checks,
		Patch: evidence.Runtime.PatchHygiene, Closure: evidence.Runtime.ClosurePolicy,
		EvidenceIndexHash: evidence.IndexHash, RepositoryRoot: repoRoot,
		Git: deps.Git, ObjectFormat: objectFormat,
	})
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("reconstruct expected transaction: %w", err)
	}
	if incomplete.ClosureCommit != reconstructed.CommitObject.OID ||
		incomplete.TagObject != reconstructed.TagObject.OID ||
		incomplete.EvidenceHash != evidence.IndexHash {
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("finalizer identities do not match deterministic reconstruction")
	}

	qualified := v2QualifiedEvidence{IndexBytes: evidence.IndexBytes, IndexSHA256: evidence.IndexHash}
	if _, err := deps.PublishEvidence(stagingDir, evidenceDir, qualified); err != nil {
		return nil, fmt.Errorf("publish evidence: %w", err)
	}
	// P0-2: Reverify the published E snapshot before ref publication so a
	// concurrent or escaped descendant cannot evict the bytes between rename
	// and exit zero. The rename itself is atomic for same-directory moves on
	// Unix filesystems (publishV2Evidence enforces this), but a follow-up
	// re-read from the final path proves the published directory still holds
	// the exact bytes that the finalizer sealed into E.
	finalEvidence, err := readQualifiedV2Evidence(evidenceDir, plan.ActID, subjectCommit)
	if err != nil {
		return nil, fmt.Errorf("reverify final evidence: %w", err)
	}
	if SHA256Hex(finalEvidence.IndexBytes) != evidence.IndexHash {
		return nil, fmt.Errorf("final evidence index hash diverges from staging snapshot")
	}
	if finalEvidence.Runtime.PublicationBranch != branch {
		return nil, fmt.Errorf("final evidence publication branch %q does not match attached branch %q",
			finalEvidence.Runtime.PublicationBranch, branch)
	}
	evidence = finalEvidence
	if err := publishV2Refs(ctx, deps.Git, repoRoot, branch, reconstructed.Tag.Name, subjectCommit,
		reconstructed.CommitObject.OID, reconstructed.TagObject.OID); err != nil {
		return nil, fmt.Errorf("publish refs: %w", err)
	}
	if err := boundedConverge(ctx, deps.Git, repoRoot, reconstructed); err != nil {
		return nil, fmt.Errorf("bounded convergence: %w", err)
	}
	return deps.VerifyExisting(ctx, deps.Git, repoRoot, evidenceDir, reconstructed, evidence)
}

func parsePlanBytes(b []byte) (Plan, []byte, error) {
	return LoadPlanFromBytes(b)
}
