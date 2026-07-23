package closure

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	PolicyProfileLeamasActV1  = "leamas-act-v1"
	PolicyProfileIndeepActV1  = "indeep-act-v1"
	PolicyProfileCircusActV1  = "circus-act-v1"
	PolicyProfileClinemmActV1 = "clinemm-act-v1"
	RunnerBindingTrustedClean = "trusted_clean"
	RunnerBindingSubjectExact = "subject_exact"
)

type PlanFreeze struct {
	CommitOID string `json:"commit_oid"`
	BlobOID   string `json:"blob_oid"`
}

type RequiredCheck struct {
	ID   string   `json:"id"`
	Argv []string `json:"argv"`
}

type policyProfile struct {
	Name           string
	RequiredChecks []RequiredCheck
}

var policyProfiles = map[string]policyProfile{
	PolicyProfileLeamasActV1: {
		Name: PolicyProfileLeamasActV1,
		RequiredChecks: []RequiredCheck{
			{ID: "focused-count-1", Argv: []string{"go", "test", "-count=1", "./internal/factory/closure/..."}},
			{ID: "vet", Argv: []string{"go", "vet", "./internal/factory/closure/..."}},
			{ID: "build", Argv: []string{"go", "build", "-buildvcs=true", "-trimpath", "-o", "/tmp/leamas-closure-protocol-v1-self", "./cmd/leamas"}},
			{ID: "gate-fast", Argv: []string{"make", "gate-fast"}},
			{ID: "diff-check", Argv: []string{"git", "diff", "--check"}},
		},
	},
	PolicyProfileIndeepActV1: {
		Name: PolicyProfileIndeepActV1,
		RequiredChecks: []RequiredCheck{
			{ID: "focused-count-1", Argv: []string{"go", "test", "-count=1", "./..."}},
			{ID: "diff-check", Argv: []string{"git", "diff", "--check"}},
		},
	},
	PolicyProfileCircusActV1: {
		Name: PolicyProfileCircusActV1,
		RequiredChecks: []RequiredCheck{
			{ID: "focused-count-1", Argv: []string{"go", "test", "-count=1", "./..."}},
			{ID: "diff-check", Argv: []string{"git", "diff", "--check"}},
		},
	},
	PolicyProfileClinemmActV1: {
		Name: PolicyProfileClinemmActV1,
		RequiredChecks: []RequiredCheck{
			{ID: "focused-count-1", Argv: []string{"go", "test", "-count=1", "./..."}},
			{ID: "diff-check", Argv: []string{"git", "diff", "--check"}},
		},
	},
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
	if err := validateOID("freeze.commit_oid", plan.Freeze.CommitOID); err != nil {
		return fmt.Errorf("freeze.commit_oid is missing or invalid: %w", err)
	}
	if err := validateSHA256("freeze.blob_oid", plan.Freeze.BlobOID); err != nil {
		return fmt.Errorf("freeze.blob_oid is missing or invalid: %w", err)
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
	if len(actual) < len(required) {
		return false
	}
	for index, want := range required {
		if argvArgMatches(actual[index], want) {
			continue
		}
		if argvPathSlot(actual[index], want) {
			continue
		}
		if argvTestPath(actual[index], want) {
			continue
		}
		return false
	}
	return true
}

func argvArgMatches(actual, required string) bool {
	if actual == required {
		return true
	}
	if strings.HasPrefix(required, "./") && strings.HasSuffix(actual, required) {
		return true
	}
	return false
}

var pathPrefixes = []string{"/tmp/leamas-closure-protocol-v1", "/tmp/leamas-closure-correction01", "/tmp/leamas-closure-self"}

func argvPathSlot(actual, required string) bool {
	if !strings.HasPrefix(required, "/") {
		return false
	}
	if !strings.HasPrefix(actual, "/") {
		return false
	}
	if !strings.HasSuffix(actual, "/cmd/leamas") {
		return false
	}
	for _, prefix := range pathPrefixes {
		if strings.HasPrefix(actual, prefix) {
			return true
		}
	}
	return false
}

func argvTestPath(actual, required string) bool {
	if !strings.HasPrefix(required, "./") {
		return false
	}
	if !strings.HasPrefix(actual, "./") {
		return false
	}
	return strings.HasSuffix(actual, required)
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
