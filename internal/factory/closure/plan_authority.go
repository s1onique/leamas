package closure

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	PolicyProfileLeamasActV1  = "leamas-act-v1"
	PolicyProfileUnsupported  = "unsupported"
	RunnerBindingTrustedClean = "trusted_clean"
	RunnerBindingSubjectExact = "subject_exact"
)

type RequiredCheck struct {
	ID   string   `json:"id"`
	Argv []string `json:"argv"`
}

type policyProfile struct {
	Name           string
	RequiredChecks []RequiredCheck
	Enabled        bool
}

var policyProfiles = map[string]policyProfile{
	PolicyProfileLeamasActV1: {
		Name:    PolicyProfileLeamasActV1,
		Enabled: true,
		RequiredChecks: []RequiredCheck{
			{ID: "focused-count-1", Argv: []string{"go", "test", "-count=1", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "focused-count-20", Argv: []string{"go", "test", "-count=20", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "focused-race-5", Argv: []string{"go", "test", "-race", "-count=5", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "vet", Argv: []string{"go", "vet", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "build", Argv: []string{"go", "build", "-buildvcs=true", "-trimpath", "-o", "/tmp/leamas-closure-protocol-v1-self", "./cmd/leamas"}},
			{ID: "gate-fast", Argv: []string{"make", "gate-fast"}},
			{ID: "diff-check", Argv: []string{"git", "diff", "--check"}},
		},
	},
	PolicyProfileUnsupported: {
		Name:    PolicyProfileUnsupported,
		Enabled: false,
	},
	"indeep-act-v1":  {Name: "indeep-act-v1", Enabled: false},
	"circus-act-v1":  {Name: "circus-act-v1", Enabled: false},
	"clinemm-act-v1": {Name: "clinemm-act-v1", Enabled: false},
}

func jsonMarshalPlan(plan Plan) ([]byte, error) {
	return json.MarshalIndent(plan, "", "  ")
}

func validatePlanAuthority(plan Plan) error {
	if plan.PolicyProfile == "" {
		return nil
	}
	profile, ok := policyProfiles[plan.PolicyProfile]
	if !ok {
		return fmt.Errorf("policy_profile %q is unknown", plan.PolicyProfile)
	}
	if !profile.Enabled {
		return fmt.Errorf("policy_profile %q is not yet implemented for this repository", plan.PolicyProfile)
	}
	for _, required := range profile.RequiredChecks {
		if !planHasCheck(plan, required) {
			return fmt.Errorf("plan does not satisfy policy profile %q: missing or non-matching check %q", profile.Name, required.ID)
		}
	}
	return nil
}

func planHasCheck(plan Plan, required RequiredCheck) bool {
	for _, check := range plan.Checks {
		if check.ID != required.ID {
			continue
		}
		if !argvSatisfies(check.Argv, required.Argv) {
			return false
		}
		return true
	}
	return false
}

func argvSatisfies(actual []string, required []string) bool {
	if len(actual) != len(required) {
		return false
	}
	for index, want := range required {
		if actual[index] != want {
			return false
		}
	}
	return true
}

func VerifyRunnerBinding(binding string, manifest Manifest) error {
	switch binding {
	case "", RunnerBindingTrustedClean:
		return nil
	case RunnerBindingSubjectExact:
		if manifest.Runner.VCSRevision != manifest.Subject.CommitOID {
			return fmt.Errorf("subject_exact runner binding requires runner.vcs_revision (%s) to equal subject.commit_oid (%s)", manifest.Runner.VCSRevision, manifest.Subject.CommitOID)
		}
		if manifest.Runner.VCSModified {
			return fmt.Errorf("subject_exact runner binding forbids a modified runner")
		}
		return nil
	default:
		return fmt.Errorf("unknown runner_binding %q", binding)
	}
}

func isSupportedProfile(name string) bool {
	profile, ok := policyProfiles[name]
	return ok && profile.Enabled
}

func supportedProfiles() []string {
	out := make([]string, 0, len(policyProfiles))
	for name, profile := range policyProfiles {
		if profile.Enabled {
			out = append(out, name)
		}
	}
	return out
}

func _placeholderUseStrings() {
	_ = strings.HasPrefix
}
