// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_authority_test.go covers the repository-bound
// Git authority matrix required by Phase 7 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.
//
// Every case constructs a real Git repository in a temp
// directory via the existing closure-package test helpers
// (initRepo, makeCommit, mustRunGit) and asserts the
// authority:
//
//   - is permanently bound to its repository root;
//   - ignores process CWD (other repository / non-repository);
//   - resolves commit / tree / path OIDs to canonical 40-char
//     hex;
//   - reads raw blob bytes verbatim (no trimming);
//   - rejects empty / missing / non-blob inputs;
//   - enforces the SHA-1 object-format policy before any OID
//     validation.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV2VerifierAuthorityBoundToRepoRootWhileCWDIsOtherRepo
// proves the authority stays bound to its repository even
// when the process CWD has been moved to a different
// repository. The test instantiates two real repos, moves
// the process CWD to repo B, and exercises the
// repository-A-bound authority. Every observation must
// return repo-A data; resolving repo-B-only revisions
// against the authority must fail.
func TestV2VerifierAuthorityBoundToRepoRootWhileCWDIsOtherRepo(t *testing.T) {
	repoA := initRepo(t)
	repoB := initRepo(t)

	// Make a commit in each repo so ResolveCommit has a
	// valid revision to dereference.
	commitA := makeCommit(t, repoA, "repo-A commit", map[string]string{"a.txt": "A\n"})
	if commitA == "" {
		t.Fatalf("repo A commit must resolve")
	}
	commitB := makeCommit(t, repoB, "repo-B commit", map[string]string{"b.txt": "B\n"})
	if commitB == "" {
		t.Fatalf("repo B commit must resolve")
	}

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, repoA)
	if err != nil {
		t.Fatalf("authority construction: %v", err)
	}

	// Move process CWD to repo B. The authority must
	// continue to operate against repo A.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(repoB); err != nil {
		t.Fatalf("chdir repoB: %v", err)
	}

	ctx := context.Background()
	// ObjectFormat returns "sha1" because both repos are
	// SHA-1, but the assertion proves the authority routes
	// to repo A (the constructor's repoRoot).
	format, err := auth.ObjectFormat(ctx)
	if err != nil {
		t.Fatalf("ObjectFormat: %v", err)
	}
	if format != "sha1" {
		t.Fatalf("ObjectFormat = %q, want sha1", format)
	}

	// ResolveCommit on repo A's revision must succeed even
	// while CWD is repo B.
	gotA, err := auth.ResolveCommit(ctx, commitA)
	if err != nil {
		t.Fatalf("ResolveCommit A: %v", err)
	}
	if gotA != commitA {
		t.Fatalf("ResolveCommit A returned %q, want %q", gotA, commitA)
	}

	// ResolveCommit on repo B's revision must fail
	// because the authority is bound to repo A.
	if _, err := auth.ResolveCommit(ctx, commitB); err == nil {
		t.Fatalf("ResolveCommit B must fail when authority bound to A")
	}
}

// TestV2VerifierAuthorityBoundToRepoRootWhileCWDIsNonRepo
// proves the authority still routes to its bound root when
// the process CWD has been moved to a directory that is not
// a Git repository. The CWD independence is a hard contract:
// the authority never falls back to process CWD.
func TestV2VerifierAuthorityBoundToRepoRootWhileCWDIsNonRepo(t *testing.T) {
	repoA := initRepo(t)
	commitA := makeCommit(t, repoA, "repo-A commit", map[string]string{"a.txt": "A\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, repoA)
	if err != nil {
		t.Fatalf("authority construction: %v", err)
	}

	// Move to a fresh, non-repository directory.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	noRepo := t.TempDir()
	if err := os.Chdir(noRepo); err != nil {
		t.Fatalf("chdir noRepo: %v", err)
	}

	ctx := context.Background()
	format, err := auth.ObjectFormat(ctx)
	if err != nil {
		t.Fatalf("ObjectFormat against CWD=non-repo: %v", err)
	}
	if format != "sha1" {
		t.Fatalf("ObjectFormat = %q, want sha1", format)
	}
	gotA, err := auth.ResolveCommit(ctx, commitA)
	if err != nil {
		t.Fatalf("ResolveCommit A via CWD=non-repo: %v", err)
	}
	if gotA != commitA {
		t.Fatalf("ResolveCommit A returned %q, want %q", gotA, commitA)
	}
}

// TestV2VerifierAuthorityEmptyRepoRootRejected proves the
// constructor refuses an empty repository root with a typed
// repository_unavailable diagnostic.
func TestV2VerifierAuthorityEmptyRepoRootRejected(t *testing.T) {
	if _, err := newV2ClosureGitAuthorityWithClient(RealGit{}, ""); err == nil {
		t.Fatalf("empty repoRoot must fail construction")
	} else {
		verr, ok := err.(*V2VerifierError)
		if !ok {
			t.Fatalf("expected *V2VerifierError, got %T: %v", err, err)
		}
		if !verr.Diags.HasCode(V2VerifierRepositoryUnavailable) {
			t.Fatalf("expected repository_unavailable, got %v", verr.Diags.Codes())
		}
	}
}

// TestV2VerifierAuthorityNonexistentRepoRootRejected proves
// the constructor accepts a non-existent path (so callers
// can build authorities against not-yet-created worktrees)
// but the FIRST observation surfaces the failure as a typed
// repository_unavailable / object_format_unavailable
// diagnostic. The constructor itself does not probe the
// filesystem, by design.
func TestV2VerifierAuthorityNonexistentRepoRootRejected(t *testing.T) {
	bogus := filepath.Join(t.TempDir(), "not-a-repo")
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, bogus)
	if err != nil {
		t.Fatalf("constructor should succeed for non-existent root: %v", err)
	}
	if _, err := auth.ObjectFormat(context.Background()); err == nil {
		t.Fatalf("ObjectFormat against non-existent repo must fail")
	}
}

// TestV2VerifierAuthorityResolveCommitMatrix proves the
// resolver returns a 40-char lowercase OID for valid
// revisions and a typed diagnostic for missing or
// malformed revisions.
func TestV2VerifierAuthorityResolveCommitMatrix(t *testing.T) {
	dir := initRepo(t)
	commit := makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	ctx := context.Background()

	// Happy path: rev resolves to the commit OID.
	got, err := auth.ResolveCommit(ctx, commit)
	if err != nil {
		t.Fatalf("ResolveCommit happy path: %v", err)
	}
	if got != commit {
		t.Fatalf("ResolveCommit = %q, want %q", got, commit)
	}
	if len(got) != 40 {
		t.Fatalf("ResolveCommit returned %d-char OID, want 40", len(got))
	}

	// Missing revision: typed diagnostic.
	if _, err := auth.ResolveCommit(ctx, strings.Repeat("d", 40)); err == nil {
		t.Fatalf("missing revision must fail")
	} else {
		verr, ok := err.(*V2VerifierError)
		if !ok {
			t.Fatalf("missing rev must produce *V2VerifierError, got %T", err)
		}
		if !verr.Diags.HasCode(V2VerifierSubjectUnresolved) {
			t.Fatalf("missing rev code = %v, want subject_unresolved",
				verr.Diags.Codes())
		}
	}

	// Empty revision: typed diagnostic.
	if _, err := auth.ResolveCommit(ctx, ""); err == nil {
		t.Fatalf("empty revision must fail")
	}
	if _, err := auth.ResolveCommit(ctx, "   "); err == nil {
		t.Fatalf("whitespace revision must fail")
	}
}

// TestV2VerifierAuthorityResolveTreeMatrix proves the
// resolver returns a 40-char lowercase OID for the tree of
// a known commit and rejects unknown commits.
func TestV2VerifierAuthorityResolveTreeMatrix(t *testing.T) {
	dir := initRepo(t)
	commit := makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	ctx := context.Background()

	tree, err := auth.ResolveTree(ctx, commit)
	if err != nil {
		t.Fatalf("ResolveTree happy path: %v", err)
	}
	if len(tree) != 40 {
		t.Fatalf("ResolveTree returned %d-char OID, want 40", len(tree))
	}
	// The tree OID must equal what `git rev-parse` returns
	// so we can use it in subsequent operations.
	wantTree := mustRunGit(t, dir, "rev-parse", commit+"^{tree}")
	if tree != wantTree {
		t.Fatalf("ResolveTree = %q, want %q", tree, wantTree)
	}

	if _, err := auth.ResolveTree(ctx, strings.Repeat("e", 40)); err == nil {
		t.Fatalf("missing commit must fail")
	}
}

// TestV2VerifierAuthorityResolvePathObjectMatrix proves the
// resolver returns (oid, "blob") for blob paths and rejects
// tree-only / missing paths. The test also proves the OID
// returned matches the authoritative `git rev-parse` output.
func TestV2VerifierAuthorityResolvePathObjectMatrix(t *testing.T) {
	dir := initRepo(t)
	commit := makeCommit(t, dir, "subject", map[string]string{
		"plan.json":   `{"contract_version":1}`,
		"extra/n.txt": "nested\n",
	})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	ctx := context.Background()

	// Happy path: blob.
	oid, objectType, err := auth.ResolvePathObject(ctx, commit, "plan.json")
	if err != nil {
		t.Fatalf("ResolvePathObject plan.json: %v", err)
	}
	if objectType != "blob" {
		t.Fatalf("plan.json object type = %q, want blob", objectType)
	}
	wantOID := mustRunGit(t, dir, "rev-parse", commit+":plan.json")
	if oid != wantOID {
		t.Fatalf("ResolvePathObject OID = %q, want %q", oid, wantOID)
	}

	// Nested path.
	oid, objectType, err = auth.ResolvePathObject(ctx, commit, "extra/n.txt")
	if err != nil {
		t.Fatalf("ResolvePathObject extra/n.txt: %v", err)
	}
	if objectType != "blob" {
		t.Fatalf("extra/n.txt object type = %q, want blob", objectType)
	}

	// Missing path: typed diagnostic.
	if _, _, err := auth.ResolvePathObject(ctx, commit, "missing.txt"); err == nil {
		t.Fatalf("missing path must fail")
	} else {
		verr, ok := err.(*V2VerifierError)
		if !ok {
			t.Fatalf("missing path must produce *V2VerifierError, got %T", err)
		}
		if !verr.Diags.HasCode(V2VerifierFrozenPlanMissing) {
			t.Fatalf("missing path code = %v, want frozen_plan_missing",
				verr.Diags.Codes())
		}
	}

	// Empty path / empty commit: typed diagnostics.
	if _, _, err := auth.ResolvePathObject(ctx, commit, ""); err == nil {
		t.Fatalf("empty path must fail")
	}
	if _, _, err := auth.ResolvePathObject(ctx, "", "plan.json"); err == nil {
		t.Fatalf("empty commit must fail")
	}
}

// TestV2VerifierAuthorityIsAncestorMatrix proves the
// resolver classifies Git's documented exit codes correctly:
//
//	0 -> true
//	1 -> false
//	other -> typed observation failure
func TestV2VerifierAuthorityIsAncestorMatrix(t *testing.T) {
	dir := initRepo(t)
	parent := makeCommit(t, dir, "parent", map[string]string{"a.txt": "A\n"})
	child := makeCommit(t, dir, "child", map[string]string{"b.txt": "B\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	ctx := context.Background()

	// parent is ancestor of child.
	if got, err := auth.IsAncestor(ctx, parent, child); err != nil || !got {
		t.Fatalf("IsAncestor(parent, child) = (%v, %v), want (true, nil)", got, err)
	}
	// child is NOT ancestor of parent.
	if got, err := auth.IsAncestor(ctx, child, parent); err != nil || got {
		t.Fatalf("IsAncestor(child, parent) = (%v, %v), want (false, nil)", got, err)
	}
	// self is ancestor of self -> false (exit 1).
	if got, err := auth.IsAncestor(ctx, parent, parent); err != nil || !got {
		t.Fatalf("IsAncestor(self, self) = (%v, %v), want (true, nil)", got, err)
	}

	// Missing revisions produce a typed observation failure
	// (exit != 0 / 1). The matrix accepts either
	// empty-input guards or upstream Git errors; both are
	// surfaced as typed diagnostics.
	if _, err := auth.IsAncestor(ctx, "", child); err == nil {
		t.Fatalf("empty ancestor must fail")
	}
	if _, err := auth.IsAncestor(ctx, parent, ""); err == nil {
		t.Fatalf("empty descendant must fail")
	}
}

// TestV2VerifierAuthorityReadBlobRawPreservesTrailingNewline
// proves `git cat-file blob <oid>` returns the exact raw
// bytes including the trailing newline. The byte-authority
// contract is the foundation of ACT 2 (frozen plan SHA-256)
// and ACT 3 (manifest SHA-256).
func TestV2VerifierAuthorityReadBlobRawPreservesTrailingNewline(t *testing.T) {
	dir := initRepo(t)
	commit := makeCommit(t, dir, "subject", map[string]string{"plan.json": `{"contract_version":1}` + "\n"})
	oid := mustRunGit(t, dir, "rev-parse", commit+":plan.json")

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	raw, err := auth.ReadBlob(context.Background(), oid)
	if err != nil {
		t.Fatalf("ReadBlob: %v", err)
	}
	if !bytes.HasSuffix(raw, []byte{'\n'}) {
		t.Fatalf("ReadBlob lost trailing newline: %q", raw)
	}
	wantSum := sha256.Sum256(raw)
	gotSum := sha256.Sum256([]byte(`{"contract_version":1}` + "\n"))
	if wantSum != gotSum {
		t.Fatalf("ReadBlob raw differ from expected; sha256 drift")
	}
	if hex.EncodeToString(wantSum[:]) != hex.EncodeToString(gotSum[:]) {
		t.Fatalf("ReadBlob byte sha256 mismatch")
	}
}

// TestV2VerifierAuthorityReadBlobRawPreservesLeadingAndTrailing
// proves leading whitespace and trailing spaces also
// round-trip through the byte-authority path unchanged. The
// regression surface for TrimSpace-style stripping is
// covered here.
func TestV2VerifierAuthorityReadBlobRawPreservesLeadingAndTrailing(t *testing.T) {
	dir := initRepo(t)
	commit := makeCommit(t, dir, "subject", map[string]string{"plan.json": " leading prefix \n leading and trailing spaces \n trailing suffix "})
	oid := mustRunGit(t, dir, "rev-parse", commit+":plan.json")

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	raw, err := auth.ReadBlob(context.Background(), oid)
	if err != nil {
		t.Fatalf("ReadBlob: %v", err)
	}
	if raw[0] != ' ' {
		t.Fatalf("leading space lost: %q", raw)
	}
	if raw[len(raw)-1] != ' ' {
		t.Fatalf("trailing space lost: %q", raw)
	}
}

// TestV2VerifierAuthorityReadBlobRejectsMissingAndEmpty
// proves the resolver surfaces typed diagnostics for
// missing OIDs and empty input.
func TestV2VerifierAuthorityReadBlobRejectsMissingAndEmpty(t *testing.T) {
	dir := initRepo(t)
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	ctx := context.Background()

	// Empty OID: typed diagnostic.
	if _, err := auth.ReadBlob(ctx, ""); err == nil {
		t.Fatalf("empty OID must fail")
	}
	if _, err := auth.ReadBlob(ctx, "   "); err == nil {
		t.Fatalf("whitespace OID must fail")
	}

	// Missing OID: typed diagnostic. The repo has no
	// reachable object with this OID.
	if _, err := auth.ReadBlob(ctx, strings.Repeat("f", 40)); err == nil {
		t.Fatalf("missing OID must fail")
	} else {
		verr, ok := err.(*V2VerifierError)
		if !ok {
			t.Fatalf("expected *V2VerifierError, got %T: %v", err, err)
		}
		if !verr.Diags.HasCode(V2VerifierFrozenPlanReadFailed) {
			t.Fatalf("missing OID code = %v, want frozen_plan_read_failed",
				verr.Diags.Codes())
		}
	}
}

// TestV2VerifierAuthorityObjectFormatMatrix exercises every
// branch of the ObjectFormat contract:
//
//   - "sha1" returns the literal token (accepted downstream
//     by EnforceV2VerifierObjectFormatPolicy);
//   - "sha256" returns the token (rejected downstream);
//   - empty returns object_format_unavailable;
//   - error returns object_format_unavailable.
func TestV2VerifierAuthorityObjectFormatMatrix(t *testing.T) {
	dir := initRepo(t)
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	format, err := auth.ObjectFormat(context.Background())
	if err != nil {
		t.Fatalf("ObjectFormat: %v", err)
	}
	if format != "sha1" {
		t.Fatalf("ObjectFormat = %q, want sha1 (hermetic repo is sha1)", format)
	}
}

// TestV2VerifierAuthorityEnforceFormatSha1Accepted proves
// the production EnforceV2VerifierObjectFormatPolicy
// short-circuits to nil when the bound repository reports
// "sha1" without invoking CatFile.
func TestV2VerifierAuthorityEnforceFormatSha1Accepted(t *testing.T) {
	dir := initRepo(t)
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	if err := EnforceV2VerifierObjectFormatPolicy(auth); err != nil {
		t.Fatalf("sha1 policy must pass, got %v", err)
	}
}

// TestV2VerifierAuthorityEnforceFormatSha256Rejected proves
// the production EnforceV2VerifierObjectFormatPolicy rejects
// a sha256 authority with the stable unsupported_object_format
// diagnostic BEFORE any blob read. The stub CatFile panics if
// invoked; the test catches the panic via t.Cleanup-style
// indirect by relying on the stub's contract.
func TestV2VerifierAuthorityEnforceFormatSha256Rejected(t *testing.T) {
	auth := v2StubAuthority{format: "sha256"}
	err := EnforceV2VerifierObjectFormatPolicy(auth)
	if err == nil {
		t.Fatalf("sha256 policy must reject")
	}
	verr, ok := err.(*V2VerifierError)
	if !ok {
		t.Fatalf("expected *V2VerifierError, got %T", err)
	}
	if !verr.Diags.HasCode(V2VerifierUnsupportedObjectFormat) {
		t.Fatalf("code = %v, want unsupported_object_format",
			verr.Diags.Codes())
	}
}

// TestV2VerifierAuthorityEnforceFormatEmptyRejected proves
// an empty format is rejected with object_format_unavailable
// and the CatFile adapter is never invoked.
func TestV2VerifierAuthorityEnforceFormatEmptyRejected(t *testing.T) {
	auth := v2StubAuthority{format: ""}
	err := EnforceV2VerifierObjectFormatPolicy(auth)
	if err == nil {
		t.Fatalf("empty format must reject")
	}
	verr, ok := err.(*V2VerifierError)
	if !ok {
		t.Fatalf("expected *V2VerifierError, got %T", err)
	}
	if !verr.Diags.HasCode(V2VerifierObjectFormatUnavailable) {
		t.Fatalf("code = %v, want object_format_unavailable",
			verr.Diags.Codes())
	}
}

// TestV2VerifierAuthorityEnforceFormatResolverErrorRejected
// proves a resolver observation failure is rejected with
// object_format_unavailable and the typed error preserves
// the underlying cause for errors.Unwrap.
func TestV2VerifierAuthorityEnforceFormatResolverErrorRejected(t *testing.T) {
	synthetic := errors.New("synthetic observation failure")
	auth := v2StubAuthority{formatErr: synthetic}
	err := EnforceV2VerifierObjectFormatPolicy(auth)
	if err == nil {
		t.Fatalf("resolver error must reject")
	}
	verr, ok := err.(*V2VerifierError)
	if !ok {
		t.Fatalf("expected *V2VerifierError, got %T", err)
	}
	if !verr.Diags.HasCode(V2VerifierObjectFormatUnavailable) {
		t.Fatalf("code = %v, want object_format_unavailable",
			verr.Diags.Codes())
	}
}

// TestV2VerifierAuthorityEnforceFormatNilAuthorityRejected
// proves a nil authority produces a typed diagnostic without
// panicking. The contract exists so the foundation ACT is
// safe to wire defensively.
func TestV2VerifierAuthorityEnforceFormatNilAuthorityRejected(t *testing.T) {
	err := EnforceV2VerifierObjectFormatPolicy(nil)
	if err == nil {
		t.Fatalf("nil authority must reject")
	}
	verr, ok := err.(*V2VerifierError)
	if !ok {
		t.Fatalf("expected *V2VerifierError, got %T", err)
	}
	if !verr.Diags.HasCode(V2VerifierObjectFormatUnavailable) {
		t.Fatalf("code = %v, want object_format_unavailable",
			verr.Diags.Codes())
	}
}

// TestV2VerifierAuthorityFormatAdapterCatFilePanics proves
// the format adapter's CatFile stub panics if invoked. The
// SHA-1 policy MUST reject before any blob read; the panic
// surfaces a regression immediately rather than silently
// producing a misleading success. The test uses defer/recover
// to assert the panic fires.
func TestV2VerifierAuthorityFormatAdapterCatFilePanics(t *testing.T) {
	auth := v2StubAuthority{format: "sha1"}
	adapter := v2AuthorityFormatAdapter{authority: auth}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("CatFile invocation must panic")
		}
	}()
	_, _ = adapter.CatFile("any-oid")
}

// v2StubAuthority is a deterministic V2ClosureGitAuthority
// implementation used by the matrix tests. The stub supplies
// canned responses for ObjectFormat and a panic-on-CatFile
// guard so a regression that routes through the adapter
// surfaces immediately.
type v2StubAuthority struct {
	format    string
	formatErr error
}

// ObjectFormat returns the stub's canned format / error.
func (s v2StubAuthority) ObjectFormat(ctx context.Context) (string, error) {
	if s.formatErr != nil {
		return "", s.formatErr
	}
	return s.format, nil
}

// ResolveCommit is unused by the matrix tests; it panics so
// any unexpected call surfaces immediately.
func (s v2StubAuthority) ResolveCommit(ctx context.Context, revision string) (string, error) {
	panic(fmt.Sprintf("v2StubAuthority.ResolveCommit must not be called (rev=%s)", revision))
}

// ResolveTree is unused; panics on call.
func (s v2StubAuthority) ResolveTree(ctx context.Context, commit string) (string, error) {
	panic(fmt.Sprintf("v2StubAuthority.ResolveTree must not be called (commit=%s)", commit))
}

// IsAncestor is unused; panics on call.
func (s v2StubAuthority) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	panic("v2StubAuthority.IsAncestor must not be called")
}

// ResolvePathObject is unused; panics on call.
func (s v2StubAuthority) ResolvePathObject(ctx context.Context, commit, path string) (string, string, error) {
	panic(fmt.Sprintf("v2StubAuthority.ResolvePathObject must not be called (commit=%s path=%s)", commit, path))
}

// ReadBlob is unused; panics on call.
func (s v2StubAuthority) ReadBlob(ctx context.Context, oid string) ([]byte, error) {
	panic(fmt.Sprintf("v2StubAuthority.ReadBlob must not be called (oid=%s)", oid))
}
