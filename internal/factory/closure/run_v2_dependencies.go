// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"crypto/sha256"
	"fmt"
)

type v2CheckRunner func(
	context.Context,
	Plan,
	v2Dependencies,
	string,
	string,
	string,
) ([]CheckResult, []EvidenceRecord, error)

// v2PatchPolicyEvaluator runs the subject-only patch hygiene policy
// on the F..S range. The freezeCommit parameter is REQUIRED so the
// implementation cannot accidentally fall back to plan.baseline; see
// PolicyRangeDecision in run_v2_policy.go and ProvenanceTopology in
// run_v2_authority.go for the explicit naming.
type v2PatchPolicyEvaluator func(
	context.Context,
	gitClient,
	string,
	Plan,
	string, // freezeCommit (F)
	string, // subject (S)
) (policyEvaluation[PatchHygiene], error)

type v2ClosurePolicyEvaluator func(
	context.Context,
	gitClient,
	string,
	Plan,
	string,
) (policyEvaluation[ClosurePolicyResult], error)

type v2Finalizer func(context.Context, v2FinalizeInput) (*TransactionResult, error)

type v2EvidencePublisher func(string, string, v2QualifiedEvidence) (v2QualifiedEvidence, error)

type v2FinalizeInput struct {
	Dependencies       v2Dependencies
	RepositoryRoot     string
	ObjectFormat       ObjectFormat
	Plan               Plan
	CanonicalPlanPath  string
	AuthoritativeBytes []byte
	PlanBlobOID        string
	FreezeCommit       string
	FreezeTree         string
	SubjectCommit      string
	SubjectTree        string
	Branch             string
	EvidenceDirectory  string
	Runner             RunnerIdentity
	RunnerAuthority    *RunnerAuthority
	Checks             []CheckResult
	CheckEvidence      []EvidenceRecord
	Patch              policyEvaluation[PatchHygiene]
	Closure            policyEvaluation[ClosurePolicyResult]
}

// defaultV2FinalizeNew builds C, E, and T in the staging directory.
// It does NOT publish the final evidence directory and does NOT
// update any refs. The orchestrator owns the unique publish step.
func defaultV2FinalizeNew(ctx context.Context, input v2FinalizeInput) (*TransactionResult, error) {
	artifacts, err := generateV2CanonicalArtifacts(v2CanonicalArtifactInput{
		ActID: input.Plan.ActID, PlanPath: input.CanonicalPlanPath,
		PlanSHA256: fmt.Sprintf("%x", sha256.Sum256(input.AuthoritativeBytes)), PlanBlobOID: input.PlanBlobOID,
		FreezeCommit: input.FreezeCommit, FreezeTree: input.FreezeTree,
		SubjectCommit: input.SubjectCommit, SubjectTree: input.SubjectTree,
		Branch: input.Branch, Runner: input.Runner, RunnerAuthority: input.RunnerAuthority, Checks: input.Checks,
		PatchHygiene: input.Patch.Value, ClosurePolicy: input.Closure.Value,
	})
	if err != nil {
		return nil, fmt.Errorf("generate canonical artifacts: %w", err)
	}
	objects, err := buildV2ClosureObjects(ctx, input.Dependencies.Git, input.RepositoryRoot,
		input.ObjectFormat, input.SubjectCommit, input.SubjectTree, input.Plan.ActID, artifacts)
	if err != nil {
		return nil, fmt.Errorf("build closure objects: %w", err)
	}
	if artifacts.Verdict != VerdictPass {
		return nil, fmt.Errorf("canonical closure verdict is fail")
	}
	if err := writeV2RuntimeEvidence(input); err != nil {
		return nil, err
	}
	evidence, err := buildV2EvidenceIndex(input.EvidenceDirectory)
	if err != nil {
		return nil, fmt.Errorf("qualify evidence index: %w", err)
	}
	tag, err := createV2TagObject(ctx, input.Dependencies.Git, input.RepositoryRoot, input.ObjectFormat, v2TagInput{
		ActID: input.Plan.ActID, ClosureCommit: objects.CommitOID, ClosureTree: objects.TreeOID,
		FreezeCommit: input.FreezeCommit, FreezeTree: input.FreezeTree,
		SubjectCommit: input.SubjectCommit, SubjectTree: input.SubjectTree,
		PlanBlobOID: input.PlanBlobOID, PlanSHA256: artifactsPlanSHA(input.AuthoritativeBytes),
		ManifestBlobOID: objects.ManifestBlobOID, ManifestSHA256: artifacts.ManifestSHA256,
		ReportBlobOID: objects.ReportBlobOID, ReportSHA256: artifacts.ReportSHA256,
		EvidenceSHA256: evidence.IndexSHA256, RunnerRevision: input.Runner.VCSRevision,
		RunnerBinarySHA: input.Runner.BinarySHA256,
	})
	if err != nil {
		return nil, fmt.Errorf("create tag object: %w", err)
	}
	return nil, &PublicationIncompleteError{
		ClosureCommit: objects.CommitOID, TagObject: tag.OID, EvidenceHash: evidence.IndexSHA256,
	}
}

func artifactsPlanSHA(planBytes []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(planBytes))
}
