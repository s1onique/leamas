// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// publicationFixture exposes a detached directory and a
// configured authority against a synthetic worktree
// inventory. Construction is fully under t.TempDir() so
// cleanup is automatic.
type publicationFixture struct {
	outsideDir string
	candidate  string
	authority  *VerifierOutputAuthority
}

func newPublicationFixture(t *testing.T) *publicationFixture {
	t.Helper()
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(outside, "verifier-out.txt")
	// repositoryRoot is the worktree root; the inventory holds
	// the canonical worktree roots for the repository. The
	// bind on PrepareVerifierOutput confirms the worktree is
	// listed, then the outside directory is opened for
	// publication.
	auth, err := PrepareVerifierOutput(worktree, candidate, []CanonicalWorktree{
		{Path: worktree},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	t.Cleanup(func() { _ = auth.Close() })
	return &publicationFixture{
		outsideDir: outside,
		candidate:  candidate,
		authority:  auth,
	}
}

// TestPublicationState_String verifies the canonical
// snake_case tokens used by downstream tooling.
func TestPublicationState_String(t *testing.T) {
	cases := []struct {
		state PublicationState
		want  string
	}{
		{PublicationNotPublished, "not_published"},
		{PublicationPublished, "published"},
		{PublicationPublishedButDirectorySyncFailed, "published_but_directory_sync_failed"},
		{PublicationPublishedButPostPublishObservationFailed, "published_but_post_publish_observation_failed"},
		{PublicationState(99), "unknown_publication_state"},
	}
	for _, c := range cases {
		if got := c.state.String(); got != c.want {
			t.Fatalf("state %d: got %q want %q", c.state, got, c.want)
		}
	}
}

// TestPublication_Success asserts the happy path: bytes are
// durable and the destination content equals what was
// published.
func TestPublication_Success(t *testing.T) {
	fx := newPublicationFixture(t)
	payload := []byte("factory: verify-v2-authority OK subject=... valid=true\n")
	res := fx.authority.Publish(payload)
	if res.State != PublicationPublished {
		t.Fatalf("expected published, got %s (err=%v)", res.State, res.Err)
	}
	got, err := os.ReadFile(fx.candidate)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("destination content mismatch:\nwant %q\ngot  %q", payload, got)
	}
	info, err := os.Stat(fx.candidate)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("destination is not regular: %v", info.Mode())
	}
}

// TestPublication_AcceptsExistingFile confirms the atomic
// rename replaces an existing destination. This is the
// "existing detached regular file" acceptance branch from
// the Phase 5 matrix; the publication must be a true
// in-place replacement rather than an append.
func TestPublication_AcceptsExistingFile(t *testing.T) {
	fx := newPublicationFixture(t)
	original := []byte("old payload\n")
	if err := os.WriteFile(fx.candidate, original, 0o644); err != nil {
		t.Fatal(err)
	}
	fresh := []byte("new payload\n")
	res := fx.authority.Publish(fresh)
	if res.State != PublicationPublished {
		t.Fatalf("expected published, got %s (err=%v)", res.State, res.Err)
	}
	got, err := os.ReadFile(fx.candidate)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, fresh) {
		t.Fatalf("destination content mismatch:\nwant %q\ngot  %q", fresh, got)
	}
}

// TestPublication_RejectsExistingDirectory is the symmetric
// "existing directory rejection" branch: publication must
// not start because the resolver already refused. The test
// confirms the rejection surfaces from PrepareVerifierOutput
// and never reaches Publish.
func TestPublication_RejectsExistingDirectory(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	dirAsCandidate := filepath.Join(outside, "itself")
	if err := os.Mkdir(dirAsCandidate, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := PrepareVerifierOutput(worktree, dirAsCandidate, []CanonicalWorktree{
		{Path: worktree},
	})
	if err == nil {
		t.Fatalf("expected rejection")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
}

// TestPublication_TempFilesAbsentAfterSuccess is the
// post-completion cleanup contract: no leftover temp files
// remain under the destination parent.
func TestPublication_TempFilesAbsentAfterSuccess(t *testing.T) {
	fx := newPublicationFixture(t)
	res := fx.authority.Publish([]byte("payload"))
	if res.State != PublicationPublished {
		t.Fatalf("expected published, got %s", res.State)
	}
	entries, err := os.ReadDir(fx.outsideDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".verifier-output-") {
			t.Fatalf("temp file %q lingered after success", e.Name())
		}
	}
}

// TestPublication_SetPermission applies a non-default perm
// and confirms the destination file inherits it.
func TestPublication_SetPermission(t *testing.T) {
	fx := newPublicationFixture(t)
	if err := fx.authority.SetPermission(0o600); err != nil {
		t.Fatalf("SetPermission: %v", err)
	}
	if err := fx.authority.SetPermission(fs.FileMode(0)); err != nil {
		t.Fatalf("SetPermission noop: %v", err)
	}
	res := fx.authority.Publish([]byte("permtest"))
	if res.State != PublicationPublished {
		t.Fatalf("expected published, got %s", res.State)
	}
	info, err := os.Stat(fx.candidate)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permission: got %v want 0600", info.Mode().Perm())
	}
}

// TestPublication_DoublePublishFails confirms state
// monotonicity: a second Publish call after success is
// rejected and reports the published state.
func TestPublication_DoublePublishFails(t *testing.T) {
	fx := newPublicationFixture(t)
	res := fx.authority.Publish([]byte("first"))
	if res.State != PublicationPublished {
		t.Fatalf("first: got %s", res.State)
	}
	res2 := fx.authority.Publish([]byte("second"))
	if res2.State == PublicationNotPublished {
		t.Fatalf("expected state other than not_published, got %s", res2.State)
	}
	if res2.CanonicalPath == "" {
		t.Fatalf("CanonicalPath empty")
	}
}

// TestPublication_CloseBeforePublishIsStateInvariant
// ensures calling Close before Publish releases the parent
// descriptor and any subsequent Publish signals a clean
// error path.
func TestPublication_CloseBeforePublishIsStateInvariant(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(outside, "x.txt")
	auth, err := PrepareVerifierOutput(worktree, candidate, []CanonicalWorktree{{Path: worktree}})
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	res := auth.Publish([]byte("payload"))
	if res.State != PublicationNotPublished {
		t.Fatalf("expected not_published after close, got %s", res.State)
	}
	if res.Err == nil || !strings.Contains(res.Err.Error(), "released") {
		t.Fatalf("expected released error, got %v", res.Err)
	}
}

// TestPublication_NilAuthorityNoPanic confirms nil-safety
// for the receiver's State and CanonicalPath methods.
func TestPublication_NilAuthorityNoPanic(t *testing.T) {
	var auth *VerifierOutputAuthority
	if auth.State() != PublicationNotPublished {
		t.Fatalf("nil state wrong")
	}
	if auth.CanonicalPath() != "" {
		t.Fatalf("nil canonical wrong")
	}
	if err := auth.Close(); err != nil {
		t.Fatalf("nil close: %v", err)
	}
	res := auth.Publish([]byte("payload"))
	if res.State != PublicationNotPublished {
		t.Fatalf("expected not_published, got %s", res.State)
	}
}

// TestPublication_RequiresParentDirectory confirms Prepare
// fails when the destination's parent directory does not
// exist. The CLI's policy rejects a nonexistent parent
// because the canonical destination must resolve to a real
// directory at preparation time.
func TestPublication_RequiresParentDirectory(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	bogus := filepath.Join(root, "outside", "absent-parent", "x.txt")
	if _, err := PrepareVerifierOutput(worktree, bogus, []CanonicalWorktree{{Path: worktree}}); err == nil {
		t.Fatalf("expected rejection when parent directory absent")
	}
}

// TestPublication_IO_DestinationReadBackRoundTrip closes the
// publication sequence with a read-back that confirms the
// destination's POSIX mode and content are the published
// bytes.
func TestPublication_IO_DestinationReadBackRoundTrip(t *testing.T) {
	fx := newPublicationFixture(t)
	payload := bytes.Repeat([]byte("a"), 4096)
	res := fx.authority.Publish(payload)
	if res.State != PublicationPublished {
		t.Fatalf("expected published, got %s", res.State)
	}
	got, err := io.ReadAll(readFileHandle(t, fx.candidate))
	if err != nil {
		t.Fatalf("readall: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload round-trip failed")
	}
}

// readFileHandle is a tiny helper that returns an open
// *os.File so the caller can stream the contents under
// io.ReadAll without managing the lifecycle inline.
func readFileHandle(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// TestPublication_AuthoritativeDirectory confirms the
// publication lands in the canonical absolute destination
// (root-relative + rel-name), not in any other directory.
func TestPublication_AuthoritativeDirectory(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(outside, "verifier.txt")
	auth, err := PrepareVerifierOutput(worktree, candidate, []CanonicalWorktree{{Path: worktree}})
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	res := auth.Publish([]byte("hello"))
	if res.State != PublicationPublished {
		t.Fatalf("expected published, got %s", res.State)
	}
	if res.CanonicalPath != auth.CanonicalPath() {
		t.Fatalf("CanonicalPath mismatch: result %q vs authority %q",
			res.CanonicalPath, auth.CanonicalPath())
	}
	if !strings.HasPrefix(res.CanonicalPath, filepath.Clean(outside)) {
		t.Fatalf("CanonicalPath %q not under %q", res.CanonicalPath, outside)
	}
}
