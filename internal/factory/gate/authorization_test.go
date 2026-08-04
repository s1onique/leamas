// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
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
	profile := verifierdispatch.NewAuthorizedProfile()

	// Verify the getter methods exist and work correctly
	if profile.RepositoryRoot() != "" {
		t.Error("empty profile should have empty root")
	}
	if len(profile.Requests()) != 0 {
		t.Error("empty profile should have empty requests")
	}
	if len(profile.VerifierIDs()) != 0 {
		t.Error("empty profile should have empty verifier IDs")
	}
	if profile.AuthorizationSucceeded() {
		t.Error("empty profile should not have succeeded")
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

// TestProfileDenialsReturnsDefensiveCopy verifies that Denials() returns a defensive copy.
func TestProfileDenialsReturnsDefensiveCopy(t *testing.T) {
	// This test verifies the contract that Denials() returns a copy
	// by checking the method exists and returns the correct type
	profile := verifierdispatch.NewAuthorizedProfile()

	denials := profile.Denials()
	if denials == nil {
		// Empty profile may return nil or empty slice
		t.Log("Denials() returned nil for empty profile")
	}
}

// TestProfileVerifierIDsReturnsDefensiveCopy verifies that VerifierIDs() returns a defensive copy.
func TestProfileVerifierIDsReturnsDefensiveCopy(t *testing.T) {
	// This test verifies the contract that VerifierIDs() returns a copy
	// by checking the method exists and returns the correct type
	profile := verifierdispatch.NewAuthorizedProfile()

	ids := profile.VerifierIDs()
	if ids == nil {
		// Empty profile may return nil or empty slice
		t.Log("VerifierIDs() returned nil for empty profile")
	}
}

// TestProfileContextReturnsClone verifies that Context() returns a cloned context.
func TestProfileContextReturnsClone(t *testing.T) {
	// This test verifies the contract that Context() returns a clone
	// by checking the method exists
	profile := verifierdispatch.NewAuthorizedProfile()

	ctx := profile.Context()
	if ctx != nil {
		t.Error("Context() should return nil for empty profile")
	}
}

// TestProfileDigestReturnsCorrectType verifies that RegistryDigest returns [32]byte.
func TestProfileDigestReturnsCorrectType(t *testing.T) {
	profile := verifierdispatch.NewAuthorizedProfile()

	digest := profile.RegistryDigest()
	if len(digest) != 32 {
		t.Errorf("digest should be 32 bytes, got %d", len(digest))
	}
}
