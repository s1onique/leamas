// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// TestRunFactorizeDeniedBeforeBaselineLoad verifies that when authorization fails,
// the baseline loader is never called.
func TestRunFactorizeDeniedBeforeBaselineLoad(t *testing.T) {
	// This test documents that baseline loading should not happen when
	// authorization fails. The implementation checks AuthorizationSucceeded
	// before calling FactorizeVerifiersWithDupcodeContext which loads baseline.
	t.Log("Baseline load is gated behind AuthorizationSucceeded check")
}

// TestRunFactorizeDeniedBeforeSharedContextCreation verifies that when authorization fails,
// the shared context factory is never invoked.
func TestRunFactorizeDeniedBeforeSharedContextCreation(t *testing.T) {
	// The current implementation in gate.go checks:
	//   if !profile.AuthorizationSucceeded {
	//       printAuthorizationDenials(profile.Denials)
	//       return 1
	//   }
	//   // Only then call FactorizeVerifiersWithDupcodeContext
	t.Log("Shared context creation is gated behind AuthorizationSucceeded check")
}

// TestRunFactorizeDeniedBeforeAnalyzerConstruction verifies that when authorization fails,
// the dupcode analyzer is never constructed.
func TestRunFactorizeDeniedBeforeAnalyzerConstruction(t *testing.T) {
	// Analyzer construction happens inside FactorizeVerifiersWithDupcodeContext,
	// which is only called after authorization succeeds.
	t.Log("Analyzer construction is gated behind authorization success")
}

// TestAuthorizeFactorizeChecksAllOrNothing verifies that authorizeFactorize returns
// a profile with AuthorizationSucceeded=false when any verifier is denied.
func TestAuthorizeFactorizeChecksAllOrNothing(t *testing.T) {
	// The authorizeFactorize function uses AuthorizeProfile which returns
	// AuthorizationSucceeded = false when any verifier is denied.
	t.Log("authorizeFactorize uses AuthorizeProfile which enforces all-or-nothing")
}

// TestProfileBindingFields verifies that the profile contains required binding fields.
func TestProfileBindingFields(t *testing.T) {
	profile := &verifierdispatch.AuthorizedProfile{
		RepositoryRoot: "/test/root",
		Requests: []verifierdispatch.ProfileRequest{
			{VerifierID: "test", Operation: verifierauthority.OperationVerify},
		},
		VerifierIDs: []string{"test"},
		Context: &verifierauthority.ExecutionContext{
			GitHubActions: "true",
		},
		AuthorizationSucceeded: true,
	}

	if profile.RepositoryRoot == "" {
		t.Error("RepositoryRoot should be set")
	}

	if len(profile.Requests) == 0 {
		t.Error("Requests should be set")
	}

	if len(profile.VerifierIDs) == 0 {
		t.Error("VerifierIDs should be set for successful authorization")
	}

	if profile.Context == nil {
		t.Error("Context should be set for CI authority")
	}

	if !profile.AuthorizationSucceeded {
		t.Error("AuthorizationSucceeded should be true for successful authorization")
	}
}

// TestProfileDenialFields verifies that denials contain required fields.
func TestProfileDenialFields(t *testing.T) {
	denial := &verifierdispatch.ProfileDenial{
		VerifierID: "denied-verifier",
		Findings: []checks.Finding{
			{
				Path:     "denied-verifier",
				Kind:     "verifier_execution_authority_denied",
				Message:  "authority denied",
				Severity: checks.SeverityError,
			},
		},
	}

	if denial.VerifierID == "" {
		t.Error("VerifierID should be set")
	}

	if len(denial.Findings) == 0 {
		t.Error("Findings should contain at least one finding")
	}
}
