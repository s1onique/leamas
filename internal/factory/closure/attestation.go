package closure

import (
	"context"
	"fmt"
)

// AttestationRequest contains parameters for attestation generation.
type AttestationRequest struct {
	RepoRoot      string
	Git           gitClient
	Manifest      Manifest
	ChainResult   ChainValidationResult
	ClosureCommit string
}

// GenerateAttestation generates a post-closure attestation from chain validation results.
func GenerateAttestation(ctx context.Context, req AttestationRequest) (Attestation, error) {
	var attest Attestation

	// Validate prerequisites
	if req.RepoRoot == "" {
		return attest, fmt.Errorf("repository root is required")
	}
	if req.Git == nil {
		return attest, fmt.Errorf("git client is required")
	}
	if req.ClosureCommit == "" {
		return attest, fmt.Errorf("closure commit is required")
	}

	// Get closure commit and tree (distinct from subject)
	cCommit := req.ClosureCommit
	cTree, err := getTree(ctx, req.Git, req.RepoRoot, cCommit)
	if err != nil {
		return attest, fmt.Errorf("get closure tree: %w", err)
	}

	// Get freeze commit and tree
	fCommit := req.Manifest.PlanFreeze.FreezeCommit
	fTree, err := getTree(ctx, req.Git, req.RepoRoot, fCommit)
	if err != nil {
		return attest, fmt.Errorf("get freeze tree: %w", err)
	}

	// Get freeze plan blob OID
	planPath := req.Manifest.Plan.Path
	fPlanBlob, err := runGitValue(ctx, req.Git, req.RepoRoot, "rev-parse", fCommit+":"+planPath)
	if err != nil {
		return attest, fmt.Errorf("get freeze plan blob: %w", err)
	}

	// Get subject commit and tree
	sCommit := req.Manifest.Subject.CommitOID
	sTree, err := getTree(ctx, req.Git, req.RepoRoot, sCommit)
	if err != nil {
		return attest, fmt.Errorf("get subject tree: %w", err)
	}

	// Get tag info
	tagName := req.Manifest.Tag
	tagObjOID, peeledTarget, isAnnotated, err := getTagInfo(ctx, req.Git, req.RepoRoot, tagName)
	if err != nil {
		return attest, fmt.Errorf("get tag info: %w", err)
	}

	// Build attestation
	attest.AttestationVersion = 1
	attest.ActID = req.Manifest.ActID
	attest.ProtocolVersion = fmt.Sprintf("v%d", req.Manifest.ContractVersion)
	attest.Description = "Closure Protocol V1 attestation"

	attest.ClosureReference = ClosureReference{
		ClosureCommit: cCommit,
		ClosureTree:   cTree,
	}

	attest.TagIdentity = TagIdentity{
		TagName:      tagName,
		TagObjectOID: tagObjOID,
		TagType:      "annotated",
		PeeledTarget: peeledTarget,
	}

	attest.FreezeReference = FreezeReference{
		FreezeCommit: fCommit,
		FreezeTree:   fTree,
		PlanBlobOID:  fPlanBlob,
	}

	attest.SubjectReference = SubjectReference{
		SubjectCommit: sCommit,
		SubjectTree:   sTree,
	}

	// Chain validity from result
	attest.ChainValidity = ChainValidity{
		FNotEqualS:                 req.ChainResult.FNotEqualS,
		FIsAncestorOfS:             req.ChainResult.FIsAncestorOfS,
		PlanBytesFEqualsPlanBytesS: req.ChainResult.PlanBytesFEqualsPlanBytesS,
		ManifestFMatchesActualF:    req.ChainResult.ManifestFMatchesActualF,
		ManifestFTreeMatchesFTree:  req.ChainResult.ManifestFTreeMatchesFTree,
		ManifestSMatchesActualS:    req.ChainResult.ManifestSMatchesActualS,
		ManifestSTreeMatchesSTree:  req.ChainResult.ManifestSTreeMatchesSTree,
		TagPeeledTargetMatchesC:    req.ChainResult.TagPeeledTargetMatchesC,
	}

	// Plan self-reference check
	noSelf, _ := CheckPlanNoSelfReferenceFromBytes([]byte(fPlanBlob))
	attest.NoSelfReference = noSelf

	_ = isAnnotated // Already validated as annotated in getTagInfo

	return attest, nil
}
