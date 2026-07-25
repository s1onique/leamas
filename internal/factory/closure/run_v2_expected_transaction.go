// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
)

// v2ExpectedInput captures the authoritative inputs required to
// deterministically reconstruct the expected C / E / T identities.
//
// The reconstruction is a pure value-level operation: constructing the
// expected transaction performs no Git writes, no worktree mutation,
// and no index mutation. It DOES invoke Git to re-derive object OIDs
// from authoritative bytes (`commit-tree`, `mktag`, `hash-object`,
// `write-tree`). These reads are idempotent and only happen when the
// caller explicitly requests them; the struct itself can be created
// without any Git command side effects.
type v2ExpectedInput struct {
	Plan               Plan
	AuthoritativeBytes []byte
	PlanBlobOID        string
	FreezeCommit       string
	FreezeTree         string
	SubjectCommit      string
	SubjectTree        string
	Runner             RunnerIdentity
	Checks             []CheckResult
	Patch              PatchHygiene
	Closure            ClosurePolicyResult
	EvidenceIndexHash  string    // E
	RepositoryRoot     string    // for git invocations
	Git                gitClient // injected; required for OID derivation
	ObjectFormat       ObjectFormat
}

// reconstructV2ExpectedTransaction derives the exact C / E / T
// identities expected by the closed transaction. It is consumed by
// both the verifier and the recovery path so they cannot disagree.
//
// The function does NOT consult `git fsck --unreachable`. It derives
// `C` and `T` by re-running the deterministic construction pipeline
// from the authoritative plan and the qualified runtime evidence.
// Running `git commit-tree` and `mktag` again is safe: identical
// bytes reproduce identical OIDs.
func reconstructV2ExpectedTransaction(ctx context.Context, in v2ExpectedInput) (v2ExpectedTransaction, error) {
	if in.Plan.ActID == "" {
		return v2ExpectedTransaction{}, fmt.Errorf("plan.act_id is required")
	}
	if in.SubjectCommit == "" || in.SubjectTree == "" {
		return v2ExpectedTransaction{}, fmt.Errorf("subject commit and tree are required")
	}
	if in.FreezeCommit == "" || in.FreezeTree == "" {
		return v2ExpectedTransaction{}, fmt.Errorf("freeze commit and tree are required")
	}
	if in.PlanBlobOID == "" {
		return v2ExpectedTransaction{}, fmt.Errorf("plan blob OID is required")
	}
	if in.EvidenceIndexHash == "" {
		return v2ExpectedTransaction{}, fmt.Errorf("evidence index hash is required")
	}
	if in.Git == nil {
		return v2ExpectedTransaction{}, fmt.Errorf("git client is required")
	}
	if in.RepositoryRoot == "" {
		return v2ExpectedTransaction{}, fmt.Errorf("repository root is required")
	}

	planSHA := artifactsPlanSHA(in.AuthoritativeBytes)
	artifacts, err := generateV2CanonicalArtifacts(v2CanonicalArtifactInput{
		ActID:         in.Plan.ActID,
		PlanPath:      "docs/closure-plans/" + in.Plan.ActID + ".json",
		PlanSHA256:    planSHA,
		PlanBlobOID:   in.PlanBlobOID,
		FreezeCommit:  in.FreezeCommit,
		FreezeTree:    in.FreezeTree,
		SubjectCommit: in.SubjectCommit,
		SubjectTree:   in.SubjectTree,
		Branch:        "", // detached reconstruction: branch is not sealed into C
		Runner:        in.Runner,
		Checks:        in.Checks,
		PatchHygiene:  in.Patch,
		ClosurePolicy: in.Closure,
	})
	if err != nil {
		return v2ExpectedTransaction{}, fmt.Errorf("regenerate canonical artifacts: %w", err)
	}

	objects, err := buildV2ClosureObjects(ctx, in.Git, in.RepositoryRoot, in.ObjectFormat,
		in.SubjectCommit, in.SubjectTree, in.Plan.ActID, artifacts)
	if err != nil {
		return v2ExpectedTransaction{}, fmt.Errorf("rebuild closure objects: %w", err)
	}

	tag, err := createV2TagObject(ctx, in.Git, in.RepositoryRoot, in.ObjectFormat, v2TagInput{
		ActID:           in.Plan.ActID,
		ClosureCommit:   objects.CommitOID,
		ClosureTree:     objects.TreeOID,
		FreezeCommit:    in.FreezeCommit,
		FreezeTree:      in.FreezeTree,
		SubjectCommit:   in.SubjectCommit,
		SubjectTree:     in.SubjectTree,
		PlanBlobOID:     in.PlanBlobOID,
		PlanSHA256:      planSHA,
		ManifestBlobOID: objects.ManifestBlobOID,
		ManifestSHA256:  artifacts.ManifestSHA256,
		ReportBlobOID:   objects.ReportBlobOID,
		ReportSHA256:    artifacts.ReportSHA256,
		EvidenceSHA256:  in.EvidenceIndexHash,
		RunnerRevision:  in.Runner.VCSRevision,
		RunnerBinarySHA: in.Runner.BinarySHA256,
	})
	if err != nil {
		return v2ExpectedTransaction{}, fmt.Errorf("rebuild tag object: %w", err)
	}

	return v2ExpectedTransaction{
		Artifacts:     artifacts,
		Objects:       objects,
		EvidenceHash:  in.EvidenceIndexHash,
		FreezeCommit:  in.FreezeCommit,
		SubjectCommit: in.SubjectCommit,
		Tag:           tag,
		CommitObject:  gitObject{OID: objects.CommitOID, Type: "commit"},
		TagObject:     gitObject{OID: tag.OID, Type: "tag"},
	}, nil
}
