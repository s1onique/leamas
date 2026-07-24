// SPDX-License-Identifier: Apache-2.0

package closure

import "strings"

// ArtifactRole describes when a planned artifact is expected to exist.
// The role is deliberately independent of Required: Required describes
// the verdict obligation at the role's applicable lifecycle boundary.
type ArtifactRole string

const (
	ArtifactRoleInput              ArtifactRole = "input"
	ArtifactRoleGeneratedOutput    ArtifactRole = "generated_output"
	ArtifactRoleNotRequired        ArtifactRole = "not_required"
	ArtifactRoleFailureErratum     ArtifactRole = "failure_erratum"
	ArtifactRolePostCommitEvidence ArtifactRole = "post_commit_evidence"
)

// ArtifactRequirement is the executable role contract used by close run.
type ArtifactRequirement struct {
	MustExistBeforeRun             bool
	MustExistAfterStagedGeneration bool
	MustExistBeforeTagging         bool
	GeneratedOnFailure             bool
}

// ArtifactRoleFor returns the explicit role or the canonical role inferred
// from a legacy plan path/id. Legacy plans remain readable, while new plans
// can declare the role directly and avoid path-based interpretation.
func ArtifactRoleFor(artifact PlanArtifact) ArtifactRole {
	if artifact.Role != "" {
		return artifact.Role
	}
	value := strings.ToLower(artifact.ID + " " + artifact.Path)
	switch {
	case strings.Contains(value, "attestation") || strings.Contains(value, "annotated_tag") || strings.Contains(value, "tag_object"):
		return ArtifactRolePostCommitEvidence
	case strings.Contains(value, "failure_erratum") || strings.Contains(value, "failed_erratum"):
		return ArtifactRoleFailureErratum
	case strings.Contains(value, "success_erratum"):
		return ArtifactRoleNotRequired
	case strings.Contains(value, "closure-manifest") || strings.Contains(value, "close-report") ||
		strings.Contains(value, "manifest") && strings.HasSuffix(strings.ToLower(artifact.Path), ".json") &&
			strings.Contains(value, "closure") || strings.Contains(value, "report") && strings.HasSuffix(strings.ToLower(artifact.Path), ".md"):
		return ArtifactRoleGeneratedOutput
	default:
		return ArtifactRoleInput
	}
}

// ArtifactRequirementFor maps a role to its lifecycle boundary.
func ArtifactRequirementFor(role ArtifactRole) ArtifactRequirement {
	switch role {
	case ArtifactRoleInput:
		return ArtifactRequirement{MustExistBeforeRun: true}
	case ArtifactRoleGeneratedOutput:
		return ArtifactRequirement{MustExistAfterStagedGeneration: true}
	case ArtifactRolePostCommitEvidence:
		return ArtifactRequirement{MustExistBeforeTagging: true}
	case ArtifactRoleFailureErratum:
		return ArtifactRequirement{MustExistAfterStagedGeneration: true, GeneratedOnFailure: true}
	case ArtifactRoleNotRequired:
		return ArtifactRequirement{}
	default:
		return ArtifactRequirement{}
	}
}

func validArtifactRole(role ArtifactRole) bool {
	switch role {
	case ArtifactRoleInput, ArtifactRoleGeneratedOutput, ArtifactRoleNotRequired,
		ArtifactRoleFailureErratum, ArtifactRolePostCommitEvidence:
		return true
	default:
		return false
	}
}
