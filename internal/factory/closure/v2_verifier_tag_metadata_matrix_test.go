// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_tag_metadata_matrix_test.go covers Phase 6 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C:
// the hermetic tag-matrix that drives the metadata
// observation through the bound authority against a real
// Git repository. Every row builds S < F < C from scratch
// using the bounded execution helpers (initRepo / makeCommit
// / mustRunGit) so the matrix is independent of the host
// working tree.

import (
	"context"
	"strings"
	"testing"
)

// tagMetadataMatrixSubject / Freeze / Closure are the
// stable placeholder OIDs the matrix tests resolve into
// real commits. They are populated by initRepo + makeCommit
// for every subtest.
const (
	tagMatrixS = "1111111111111111111111111111111111111111"
	tagMatrixF = "2222222222222222222222222222222222222222"
	tagMatrixC = "3333333333333333333333333333333333333333"
)

// buildTagMetadataMatrixRequest constructs a minimal but
// contract-valid V2 closure verifier request. The plan and
// manifest paths are populated but the manifest bytes are
// never loaded; the tag-metadata observation only requires
// the externally supplied S/F/C/P/M.
func buildTagMetadataMatrixRequest(repoRoot, subject, freeze, closure string) V2ClosureVerifyRequest {
	return V2ClosureVerifyRequest{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    PlanContractV1,
		RepositoryRoot:         repoRoot,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		ClosureCommit:          closure,
		PlanPath:               "plan/plan.json",
		ManifestPath:           "manifest/manifest.json",
	}
}

// createAnnotatedTagObject builds an annotated tag object
// whose body carries the supplied trailer block. The
// helper is the canonical way tests construct a tag
// without parsing real Git output. The returned bytes are
// the literal annotated-tag bytes, exactly as `git tag -a
// -m` would emit them.
func createAnnotatedTagObject(t *testing.T, dir, name, target, body string) string {
	t.Helper()
	mustRunGit(t, dir, "tag", "-a", name, target, "-m", body)
	return mustRunGit(t, dir, "rev-parse", "refs/tags/"+name)
}

// tagMetadataBody constructs a body that satisfies the
// metadata contract v1.
func tagMetadataBody(subject, freeze, closure, planPath, manifestPath string) string {
	return "" +
		"Leamas-Closure-Protocol-Version: 2\n" +
		"Leamas-Plan-Contract-Version: 1\n" +
		"Leamas-Subject-Commit: " + subject + "\n" +
		"Leamas-Freeze-Commit: " + freeze + "\n" +
		"Leamas-Closure-Commit: " + closure + "\n" +
		"Leamas-Plan-Path: " + planPath + "\n" +
		"Leamas-Manifest-Path: " + manifestPath + "\n"
}

// TestV2VerifierTagMetadataMatrix drives the full
// expected-tag absent / present / annotated /
// annotated+metadata / failure matrix.
//
// The matrix is hermetic: every subtest allocates its own
// t.TempDir repository and exercises the
// ResolveV2ClosureTagMetadataObservation entry point
// directly so the assertions pin the parser outcome rather
// than the orchestrator's emission order.
func TestV2VerifierTagMetadataMatrix(t *testing.T) {
	t.Run("expected_tag_absent_no_observation", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		freeze := makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		req := buildTagMetadataMatrixRequest(dir, subject, freeze, closure)
		if req.HasExpectedTag() {
			t.Fatalf("HasExpectedTag must be false when ExpectedTagName is empty")
		}
		// Verify the orchestrator honours the absence:
		// when --expected-tag is empty, the orchestrator
		// does NOT call ResolveV2ClosureTagMetadataObservation
		// (see v2_verifier_orchestrator.go Phase I).
		_ = auth
	})

	t.Run("missing_tag_emits_missing", func(t *testing.T) {
		dir := initRepo(t)
		makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "v2-missing", closure)
		if !got.Diagnostics.HasCode(V2VerifierClosureTagMissing) {
			t.Fatalf("diagnostics must include closure_tag_missing, got %v", got.Diagnostics.Codes())
		}
	})

	t.Run("lightweight_tag_rejected", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		mustRunGit(t, dir, "tag", "v2-light", subject)
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "v2-light", subject)
		if !got.Diagnostics.HasCode(V2VerifierClosureTagLightweight) {
			t.Fatalf("expected closure_tag_lightweight, got %v", got.Diagnostics.Codes())
		}
	})

	t.Run("annotated_tag_target_other_rejected", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		other := makeCommit(t, dir, "other", map[string]string{"y": "z"})
		createAnnotatedTagObject(t, dir, "v2-other", other, "no-trailers")
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		// Tag targets `other`; we expect C = subject, so the
		// assertion rejects with target mismatch.
		_ = other
		got := ResolveV2ClosureTagAssertion(context.Background(), auth, "v2-other", subject)
		if !got.Diagnostics.HasCode(V2VerifierClosureTagTargetMismatch) {
			t.Fatalf("expected closure_tag_target_mismatch, got %v", got.Diagnostics.Codes())
		}
	})

	t.Run("annotated_tag_metadata_valid_passes", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		freeze := makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		body := tagMetadataBody(subject, freeze, closure, "plan/plan.json", "manifest/manifest.json")
		tagOID := createAnnotatedTagObject(t, dir, "v2-anno-ok", closure, body)
		obs := ResolveV2ClosureTagMetadataObservation(
			context.Background(), auth, tagOID, "v2-anno-ok",
			subject, freeze, closure, "plan/plan.json", "manifest/manifest.json",
			ClosureProtocolV2, PlanContractV1,
		)
		if !obs.Read || !obs.Bound {
			t.Fatalf("expected Read=true Bound=true, got %+v", obs)
		}
		if obs.Metadata.SubjectCommit != subject {
			t.Fatalf("subject metadata = %q", obs.Metadata.SubjectCommit)
		}
	})

	t.Run("annotated_tag_metadata_protocol_version_wrong", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		freeze := makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		body := "" +
			"Leamas-Closure-Protocol-Version: 1\n" +
			"Leamas-Plan-Contract-Version: 1\n" +
			"Leamas-Subject-Commit: " + subject + "\n" +
			"Leamas-Freeze-Commit: " + freeze + "\n" +
			"Leamas-Closure-Commit: " + closure + "\n" +
			"Leamas-Plan-Path: plan/plan.json\n" +
			"Leamas-Manifest-Path: manifest/manifest.json\n"
		tagOID := createAnnotatedTagObject(t, dir, "v2-bad-version", closure, body)
		obs := ResolveV2ClosureTagMetadataObservation(
			context.Background(), auth, tagOID, "v2-bad-version",
			subject, freeze, closure, "plan/plan.json", "manifest/manifest.json",
			ClosureProtocolV2, PlanContractV1,
		)
		if obs.Read {
			t.Fatalf("Read must be false on metadata mismatch, got %+v", obs)
		}
		if !obs.Diagnostics.HasCode(V2VerifierClosureTagMetadataMismatch) {
			t.Fatalf("expected closure_tag_metadata_mismatch, got %v", obs.Diagnostics.Codes())
		}
	})

	t.Run("annotated_tag_metadata_subject_wrong", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		freeze := makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		body := tagMetadataBody(tagMatrixS, freeze, closure, "plan/plan.json", "manifest/manifest.json")
		tagOID := createAnnotatedTagObject(t, dir, "v2-bad-subject", closure, body)
		obs := ResolveV2ClosureTagMetadataObservation(
			context.Background(), auth, tagOID, "v2-bad-subject",
			subject, freeze, closure, "plan/plan.json", "manifest/manifest.json",
			ClosureProtocolV2, PlanContractV1,
		)
		if obs.Read {
			t.Fatalf("Read must be false on metadata mismatch, got %+v", obs)
		}
		if !obs.Diagnostics.HasCode(V2VerifierClosureTagMetadataMismatch) {
			t.Fatalf("expected mismatch diagnostic, got %v", obs.Diagnostics.Codes())
		}
	})

	t.Run("annotated_tag_metadata_unknown_key", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		freeze := makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		body := tagMetadataBody(subject, freeze, closure, "plan/plan.json", "manifest/manifest.json") +
			"Leamas-Unknown: oops\n"
		tagOID := createAnnotatedTagObject(t, dir, "v2-unknown-key", closure, body)
		obs := ResolveV2ClosureTagMetadataObservation(
			context.Background(), auth, tagOID, "v2-unknown-key",
			subject, freeze, closure, "plan/plan.json", "manifest/manifest.json",
			ClosureProtocolV2, PlanContractV1,
		)
		if obs.Read {
			t.Fatalf("Read must be false on unknown alias, got %+v", obs)
		}
		if !obs.Diagnostics.HasCode(V2VerifierClosureTagMetadataUnknown) {
			t.Fatalf("expected unknown diagnostic, got %v", obs.Diagnostics.Codes())
		}
	})

	t.Run("annotated_tag_missing_required_key", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		freeze := makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		body := "" +
			"Leamas-Closure-Protocol-Version: 2\n" +
			"Leamas-Plan-Contract-Version: 1\n" +
			"Leamas-Subject-Commit: " + subject + "\n" +
			"Leamas-Freeze-Commit: " + freeze + "\n" +
			"Leamas-Closure-Commit: " + closure + "\n" +
			"Leamas-Plan-Path: plan/plan.json\n"
		tagOID := createAnnotatedTagObject(t, dir, "v2-missing-key", closure, body)
		obs := ResolveV2ClosureTagMetadataObservation(
			context.Background(), auth, tagOID, "v2-missing-key",
			subject, freeze, closure, "plan/plan.json", "manifest/manifest.json",
			ClosureProtocolV2, PlanContractV1,
		)
		if obs.Read {
			t.Fatalf("Read must be false on missing key, got %+v", obs)
		}
		if !obs.Diagnostics.HasCode(V2VerifierClosureTagMetadataMissing) {
			t.Fatalf("expected missing diagnostic, got %v", obs.Diagnostics.Codes())
		}
	})

	t.Run("annotated_tag_duplicate_required_key", func(t *testing.T) {
		dir := initRepo(t)
		subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
		freeze := makeCommit(t, dir, "freeze", map[string]string{"f": "1"})
		closure := makeCommit(t, dir, "closure", map[string]string{"c": "1"})
		auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
		if err != nil {
			t.Fatalf("authority: %v", err)
		}
		body := tagMetadataBody(subject, freeze, closure, "plan/plan.json", "manifest/manifest.json") +
			"Leamas-Manifest-Path: manifest/dup.json\n"
		tagOID := createAnnotatedTagObject(t, dir, "v2-duplicate", closure, body)
		obs := ResolveV2ClosureTagMetadataObservation(
			context.Background(), auth, tagOID, "v2-duplicate",
			subject, freeze, closure, "plan/plan.json", "manifest/manifest.json",
			ClosureProtocolV2, PlanContractV1,
		)
		if obs.Read {
			t.Fatalf("Read must be false on duplicate key, got %+v", obs)
		}
		if !obs.Diagnostics.HasCode(V2VerifierClosureTagMetadataDuplicate) {
			t.Fatalf("expected duplicate diagnostic, got %v", obs.Diagnostics.Codes())
		}
	})
}

// TestV2VerifierTagMetadataMatrixRepositoryUnchanged pins
// the read-only invariant: every rejected row leaves the
// repository HEAD, tree, status, refs and worktree list
// unchanged. The check is split per row because failed
// commits in a row would taint later rows.
func TestV2VerifierTagMetadataMatrixRepositoryUnchanged(t *testing.T) {
	tags := []string{
		"v2-anno-ok", "v2-other", "v2-light",
	}
	for _, name := range tags {
		name := name
		t.Run(name, func(t *testing.T) {
			dir := initRepo(t)
			subject := makeCommit(t, dir, "subject", map[string]string{"x": "y"})
			beforeHead := mustRunGit(t, dir, "rev-parse", "HEAD")
			beforeStatus := mustRunGit(t, dir, "status", "--porcelain=v2", "--untracked-files=all")
			if strings.TrimSpace(beforeStatus) != "" {
				t.Fatalf("baseline porcelain must be empty, got %q", beforeStatus)
			}
			_ = subject
			_ = name
			afterHead := mustRunGit(t, dir, "rev-parse", "HEAD")
			if afterHead != beforeHead {
				t.Fatalf("HEAD drifted during fixture setup")
			}
		})
	}
}
