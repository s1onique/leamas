// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_cli_tag_test.go covers Phase 4 of
// ACT-LEAMAS-FACTORY-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01:
//
// the optional annotated-tag assertion that ties a verifier
// invocation to a closure-protocol tag targeting C.
//
// The matrix exercises the five classification outcomes
// the resolver publishes:
//
//   Expected empty  -> Expected=false (no Git observation)
//   Missing ref     -> closure_tag_missing
//   Lightweight tag -> closure_tag_lightweight
//   Annotated but
//     target != C   -> closure_tag_target_mismatch
//   Annotated and
//     target == C   -> Accepted (Expected=true Found=true
//                       Annotated=true Target=C Diagnostics=nil)

import (
	"context"
	"testing"
)

// TestV2VerifierTagMatrix exercises the closed
// classification set of the optional annotated-tag
// assertion path. Each hermetic repository is constructed
// from scratch so the matrix is independent of the host
// working tree and of any prior tests.
func TestV2VerifierTagMatrix(t *testing.T) {
	t.Run("empty_expected_returns_zero_value", func(t *testing.T) {
		dir := initRepo(t)
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "", "deadbeef")
		if got.Expected || got.Found || got.Annotated || got.Target != "" || len(got.Diagnostics) != 0 {
			t.Fatalf("expected zero-valued verdict, got %+v", got)
		}
	})

	t.Run("missing_tag_returns_closure_tag_missing", func(t *testing.T) {
		dir := initRepo(t)
		makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "v2-missing-tag", "deadbeef")
		if !got.Expected {
			t.Fatalf("Expected must be true, got %+v", got)
		}
		if !got.Diagnostics.HasCode(V2VerifierClosureTagMissing) {
			t.Fatalf("diagnostics must include closure_tag_missing, got %v", got.Diagnostics.Codes())
		}
	})

	t.Run("lightweight_tag_returns_closure_tag_lightweight", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		mustRunGit(t, dir, "tag", "v2-light-tag", subject)
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "v2-light-tag", subject)
		if !got.Expected || !got.Found || got.Annotated {
			t.Fatalf("expected Found=true Annotated=false, got %+v", got)
		}
		if !got.Diagnostics.HasCode(V2VerifierClosureTagLightweight) {
			t.Fatalf("diagnostics must include closure_tag_lightweight, got %v", got.Diagnostics.Codes())
		}
	})

	t.Run("annotated_tag_target_mismatch_returns_target_mismatch", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		other := makeCommit(t, dir, "other", map[string]string{"y": "z"})
		// Create an annotated tag pointing at `other`.
		mustRunGit(t, dir, "tag", "-a", "-m", "v2 annotated tag", "v2-anno-tag", other)
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "v2-anno-tag", subject)
		if !got.Expected || !got.Found || !got.Annotated {
			t.Fatalf("Expected/Found/Annotated must be true, got %+v", got)
		}
		if !got.Diagnostics.HasCode(V2VerifierClosureTagTargetMismatch) {
			t.Fatalf("diagnostics must include closure_tag_target_mismatch, got %v", got.Diagnostics.Codes())
		}
		if got.Target != other {
			t.Fatalf("Target must be %s, got %s", other, got.Target)
		}
	})

	t.Run("annotated_tag_target_match_returns_clean", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		mustRunGit(t, dir, "tag", "-a", "-m", "v2 annotated tag", "v2-anno-good", subject)
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "v2-anno-good", subject)
		if !got.Expected || !got.Found || !got.Annotated || got.Target != subject {
			t.Fatalf("expected Found=true Annotated=true Target=%s, got %+v", subject, got)
		}
		if len(got.Diagnostics) != 0 {
			t.Fatalf("expected no diagnostics, got %v", got.Diagnostics.Codes())
		}
	})
}
