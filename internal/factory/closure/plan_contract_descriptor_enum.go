package closure

import "sort"

// plan_contract_descriptor_enum.go centralises every closed-set enum
// authority the v1 descriptor depends on. Each helper reads its
// authoritative runtime source (supportedExecutionModes,
// policyProfiles, the ArtifactRole / CheckMode /
// RunnerAuthorityMode constants) so the descriptor and the runtime
// can never disagree.
// enumAuthorityExecutionMode is the closed, ordered enum authority
// for the execution mode. It is the descriptor's single source of
// truth and is kept in lock-step with supportedExecutionModes.
func enumAuthorityExecutionMode() []string {
	out := make([]string, 0, len(supportedExecutionModes))
	for _, m := range supportedExecutionModes {
		out = append(out, string(m))
	}
	return out
}

// enumAuthorityCheckMode is the closed, ordered enum authority for
// check modes. It mirrors the CheckModeRun and CheckModeExclude
// constants in model.go.
func enumAuthorityCheckMode() []string {
	return []string{CheckModeRun, CheckModeExclude}
}

// enumAuthorityArtifactRole is the closed, ordered enum authority
// for artifact roles. It mirrors the ArtifactRole* constants in
// artifact_roles.go and is sorted by declaration order so the
// diagnostic output is deterministic.
func enumAuthorityArtifactRole() []string {
	return []string{
		string(ArtifactRoleInput),
		string(ArtifactRoleGeneratedOutput),
		string(ArtifactRoleNotRequired),
		string(ArtifactRoleFailureErratum),
		string(ArtifactRolePostCommitEvidence),
	}
}

// enumAuthorityPolicyProfile is the closed, ordered enum authority
// for policy profiles. It enumerates every enabled profile; the
// descriptor is the single source of truth and is kept in lock-step
// with policyProfiles.
func enumAuthorityPolicyProfile() []string {
	out := []string{}
	names := make([]string, 0, len(policyProfiles))
	for name := range policyProfiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if profile, ok := policyProfiles[name]; ok && profile.Enabled {
			out = append(out, profile.Name)
		}
	}
	return out
}

// enumAuthorityRunnerBinding is the closed, ordered enum authority
// for the runner_binding field. The empty value (omitted) is
// normalised to RunnerBindingTrustedClean by VerifyRunnerBinding
// but is not a literal accepted value in JSON.
func enumAuthorityRunnerBinding() []string {
	return []string{RunnerBindingTrustedClean, RunnerBindingSubjectExact}
}

// enumAuthorityRunnerAuthorityMode is the closed, ordered enum
// authority for runner_authority.mode. It mirrors the
// RunnerAuthoritySubjectExact and RunnerAuthorityToolReleaseExact
// constants in model.go.
func enumAuthorityRunnerAuthorityMode() []string {
	return []string{
		string(RunnerAuthoritySubjectExact),
		string(RunnerAuthorityToolReleaseExact),
	}
}
