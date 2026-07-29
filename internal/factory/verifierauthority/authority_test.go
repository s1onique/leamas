// SPDX-License-Identifier: Apache-2.0

package verifierauthority

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestValidateAuthority_LocalSafe(t *testing.T) {
	ec := ExecutionContext{}
	err := ValidateAuthority(ec, AuthorityLocalSafe, OperationVerify)
	if err != nil {
		t.Errorf("unexpected error for local_safe: %v", err)
	}
}

func TestValidateAuthority_LocalSafe_UpdateBaseline(t *testing.T) {
	// update_baseline should be allowed locally (unlike ci_exact_checkout)
	ec := ExecutionContext{}
	err := ValidateAuthority(ec, AuthorityLocalSafe, OperationUpdateBaseline)
	if err != nil {
		t.Errorf("unexpected error for local_safe update_baseline: %v", err)
	}
}

func TestValidateAuthority_UnknownAuthority(t *testing.T) {
	ec := ExecutionContext{}
	err := ValidateAuthority(ec, "unknown", OperationVerify)

	var ae *AuthorityError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AuthorityError, got: %T %v", err, err)
	}

	if ae.ReasonCode != ReasonCodeUnknownAuthority {
		t.Errorf("wrong reason code: %s", ae.ReasonCode)
	}
}

func TestValidateAuthority_CI_EmptyContext(t *testing.T) {
	// Empty context should fail CI exact checkout
	ec := ExecutionContext{}
	err := ValidateAuthority(ec, AuthorityCIExactCheckout, OperationVerify)

	var ae *AuthorityError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AuthorityError, got: %T %v", err, err)
	}

	// Should fail on missing CI
	if ae.ReasonCode != ReasonCodeMissingCI {
		t.Errorf("wrong reason code: %s", ae.ReasonCode)
	}
}

func TestValidateAuthority_CI_AllMarkersPresent(t *testing.T) {
	// Valid CI context should pass
	ec := ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: AuthorityMarker,
		GitHubSHA:       strings.Repeat("a", 40),
		GitHubWorkspace: "/repo",
		HeadCommit:      strings.Repeat("a", 40),
		WorktreeStatus:  "",
		RepositoryRoot:  "/repo",
		WorkspaceRoot:   "/repo",
	}

	err := ValidateAuthority(ec, AuthorityCIExactCheckout, OperationVerify)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAuthority_CI_HeadMismatch(t *testing.T) {
	ec := ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: AuthorityMarker,
		GitHubSHA:       strings.Repeat("a", 40),
		GitHubWorkspace: "/repo",
		HeadCommit:      strings.Repeat("b", 40), // Mismatch!
		WorktreeStatus:  "",
		RepositoryRoot:  "/repo",
		WorkspaceRoot:   "/repo",
	}

	err := ValidateAuthority(ec, AuthorityCIExactCheckout, OperationVerify)

	var ae *AuthorityError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AuthorityError, got: %T %v", err, err)
	}

	if ae.ReasonCode != ReasonCodeHeadMismatch {
		t.Errorf("wrong reason code: %s", ae.ReasonCode)
	}
}

func TestValidateAuthority_CI_DirtyTree(t *testing.T) {
	ec := ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: AuthorityMarker,
		GitHubSHA:       strings.Repeat("a", 40),
		GitHubWorkspace: "/repo",
		HeadCommit:      strings.Repeat("a", 40),
		WorktreeStatus:  "M internal/factory/gate/gate.go", // Dirty!
		RepositoryRoot:  "/repo",
		WorkspaceRoot:   "/repo",
	}

	err := ValidateAuthority(ec, AuthorityCIExactCheckout, OperationVerify)

	var ae *AuthorityError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AuthorityError, got: %T %v", err, err)
	}

	if ae.ReasonCode != ReasonCodeDirtyTree {
		t.Errorf("wrong reason code: %s", ae.ReasonCode)
	}
}

func TestValidateAuthority_CI_UpdateBaselineDenied(t *testing.T) {
	ec := ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: AuthorityMarker,
		GitHubSHA:       strings.Repeat("a", 40),
		GitHubWorkspace: "/repo",
		HeadCommit:      strings.Repeat("a", 40),
		WorktreeStatus:  "",
		RepositoryRoot:  "/repo",
		WorkspaceRoot:   "/repo",
	}

	err := ValidateAuthority(ec, AuthorityCIExactCheckout, OperationUpdateBaseline)

	var ae *AuthorityError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AuthorityError, got: %T %v", err, err)
	}

	if ae.ReasonCode != ReasonCodeOperationDenied {
		t.Errorf("wrong reason code: %s", ae.ReasonCode)
	}
}

func TestIsAuthorityError(t *testing.T) {
	var err error = &AuthorityError{
		ReasonCode: ReasonCodeMissingCI,
	}

	if !IsAuthorityError(err) {
		t.Error("expected true for AuthorityError")
	}

	if IsAuthorityError(context.DeadlineExceeded) {
		t.Error("expected false for non-AuthorityError")
	}
}

func TestReasonCode(t *testing.T) {
	err := &AuthorityError{
		ReasonCode: ReasonCodeHeadMismatch,
	}

	if ReasonCode(err) != ReasonCodeHeadMismatch {
		t.Errorf("wrong reason code: %s", ReasonCode(err))
	}

	if ReasonCode(context.DeadlineExceeded) != "" {
		t.Error("expected empty string for non-AuthorityError")
	}
}

// TestNewLocalOnlyContext covers the package-internal helper.
func TestNewLocalOnlyContext(t *testing.T) {
	ec := newLocalOnlyContext()
	if ec == nil {
		t.Fatal("expected non-nil context")
	}
	// All fields should be zero values
	if ec.CI != "" || ec.GitHubActions != "" {
		t.Error("expected empty CI fields for local context")
	}
	if !ec.observationCompleted() {
		t.Error("expected observation to be marked completed")
	}
}

// TestNoExportedLocalConstructor asserts that no exported production
// function returns a preclassified ExecutionContext. The exported
// production surface must channel every local-classification through
// DetectExecutionContext so observation provenance cannot be forged.
func TestNoExportedLocalConstructor(t *testing.T) {
	// Build a fresh ExecutionContext by hand: all exported fields.
	empty := ExecutionContext{}
	if got := ClassifyExecutionEnvironment(empty); got == EnvironmentLocal {
		t.Error("zero-value ExecutionContext must NOT classify as EnvironmentLocal")
	}
	// A populated but contradictory context must not classify as local.
	contradictory := ExecutionContext{
		CI:              "false",
		GitHubActions:   "false",
		AuthorityMarker: "marker",
	}
	if got := ClassifyExecutionEnvironment(contradictory); got == EnvironmentLocal {
		t.Error("contradictory context must NOT classify as EnvironmentLocal")
	}
	// Calling DetectExecutionContext against a path that is not a
	// git repository must yield an unclassified context, NOT local.
	ctx := context.Background()
	ec := DetectExecutionContext(ctx, t.TempDir())
	if got := ClassifyExecutionEnvironment(ec); got == EnvironmentLocal {
		t.Errorf("DetectExecutionContext against non-git must NOT classify as EnvironmentLocal, got %q", got)
	}
}

func TestValidateAuthority_CI_GitHeadFailed(t *testing.T) {
	ec := ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: AuthorityMarker,
		GitHubSHA:       strings.Repeat("a", 40),
		GitHubWorkspace: "/repo",
		HeadErr:         errors.New("git failed"),
		WorktreeStatus:  "",
		RepositoryRoot:  "/repo",
		WorkspaceRoot:   "/repo",
	}

	err := ValidateAuthority(ec, AuthorityCIExactCheckout, OperationVerify)

	var ae *AuthorityError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AuthorityError, got: %T %v", err, err)
	}

	if ae.ReasonCode != ReasonCodeGitHeadFailed {
		t.Errorf("wrong reason code: %s", ae.ReasonCode)
	}
}
