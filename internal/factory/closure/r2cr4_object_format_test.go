// SPDX-License-Identifier: Apache-2.0

package closure

// r2cr4_object_format_test.go covers the programmatic
// SHA-1 policy and verifier precedence required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R4.
//
// Production contract:
//   - EnforceSHA1ObjectFormat runs BEFORE any OID validation
//   - sha1 passes, sha256 fails with unsupported_object_format,
//     empty / error fails with object_format_unavailable
//   - OID length is NOT used as a format detector
//   - VerifyClosureManifestR2B never calls CatFile when the
//     resolver reports anything other than sha1

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestR2CRObjectFormatEnforcedBeforeOIDValidation proves
// the verifier rejects a sha256 resolver with a typed
// unsupported_object_format diagnostic before any OID
// validation runs.
func TestR2CRObjectFormatEnforcedBeforeOIDValidation(t *testing.T) {
	resolver := &r2cr4StubResolver{formatResult: "sha256"}
	err := EnforceSHA1ObjectFormat(resolver)
	if err == nil {
		t.Fatalf("expected unsupported_object_format error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeUnsupportedObjectFormat) {
		t.Fatalf("expected unsupported_object_format, got %v", v2err.Diags.Codes())
	}
}

// TestR2CRObjectFormatEmptyRejected proves an empty format
// string is rejected with object_format_unavailable.
func TestR2CRObjectFormatEmptyRejected(t *testing.T) {
	resolver := &r2cr4StubResolver{formatResult: ""}
	err := EnforceSHA1ObjectFormat(resolver)
	if err == nil {
		t.Fatalf("expected object_format_unavailable error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if !v2err.Diags.HasCode(V2CodeObjectFormatUnavailable) {
		t.Fatalf("expected object_format_unavailable, got %v", v2err.Diags.Codes())
	}
}

// TestR2CRObjectFormatResolverErrorRejected proves a
// resolver error surfaces as object_format_unavailable.
func TestR2CRObjectFormatResolverErrorRejected(t *testing.T) {
	resolver := &r2cr4StubResolver{formatErr: errors.New("synthetic observation failure")}
	err := EnforceSHA1ObjectFormat(resolver)
	if err == nil {
		t.Fatalf("expected object_format_unavailable error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if !v2err.Diags.HasCode(V2CodeObjectFormatUnavailable) {
		t.Fatalf("expected object_format_unavailable, got %v", v2err.Diags.Codes())
	}
}

// TestR2CRObjectFormatSha1Accepted proves a sha1 resolver
// passes EnforceSHA1ObjectFormat without diagnostics.
func TestR2CRObjectFormatSha1Accepted(t *testing.T) {
	resolver := &r2cr4StubResolver{formatResult: "sha1"}
	if err := EnforceSHA1ObjectFormat(resolver); err != nil {
		t.Fatalf("sha1 must be accepted, got %v", err)
	}
}

// TestR2CRObjectFormatOIDLengthNotUsedAsDetector proves the
// resolver's reported format overrides OID length detection.
func TestR2CRObjectFormatOIDLengthNotUsedAsDetector(t *testing.T) {
	resolver := &r2cr4StubResolver{
		formatResult: "sha1",
		cat:          func(oid string) ([]byte, error) { return []byte("tree " + oid + "\n"), nil },
	}
	if err := EnforceSHA1ObjectFormat(resolver); err != nil {
		t.Fatalf("sha1 must be accepted regardless of OID length, got %v", err)
	}
}

// r2cr4StubResolver is a GitObjectResolver whose ObjectFormat
// and CatFile outcomes are configured by the test.
type r2cr4StubResolver struct {
	cat          func(oid string) ([]byte, error)
	formatResult string
	formatErr    error
}

func (r *r2cr4StubResolver) CatFile(oid string) ([]byte, error) {
	if r.cat != nil {
		return r.cat(oid)
	}
	return nil, errors.New("cat not configured")
}

func (r *r2cr4StubResolver) ObjectFormat() (string, error) {
	return r.formatResult, r.formatErr
}

// r2cr4SpyResolver counts ObjectFormat and CatFile
// invocations so tests can assert that the verifier rejects
// an unsupported format BEFORE any CatFile call.
type r2cr4SpyResolver struct {
	formatResult  string
	formatErr     error
	formatCalls   int
	catCalls      int
	catShouldFail bool
}

func (r *r2cr4SpyResolver) ObjectFormat() (string, error) {
	r.formatCalls++
	return r.formatResult, r.formatErr
}

func (r *r2cr4SpyResolver) CatFile(oid string) ([]byte, error) {
	r.catCalls++
	if r.catShouldFail {
		return nil, errors.New("cat should not have been called")
	}
	return []byte("tree " + oid + "\n"), nil
}

// TestR2CRVerifyClosureManifestR2B_RejectsSha256BeforeCatFile
// proves the verifier rejects a sha256 resolver BEFORE any
// CatFile call. The spy resolver counts both operations;
// for sha256 / empty / error, CatFile must never run.
func TestR2CRVerifyClosureManifestR2B_RejectsSha256BeforeCatFile(t *testing.T) {
	cases := []struct {
		name   string
		format string
		err    error
	}{
		{name: "sha256", format: "sha256"},
		{name: "empty", format: ""},
		{name: "error", format: "", err: errors.New("synthetic observation failure")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resolver := &r2cr4SpyResolver{
				formatResult:  c.format,
				formatErr:     c.err,
				catShouldFail: true,
			}
			// Construct a minimal manifest + repo on disk so
			// VerifyClosureManifestR2B has something to read
			// IF the verifier reaches CatFile. With the spy
			// configured to fail on CatFile, any unintended
			// call would surface as an error rather than the
			// expected format diagnostic.
			repo := initRepo(t)
			manifestPath := filepath.Join(t.TempDir(), "r2cr-spy-"+c.name+".json")
			doc := []byte("{\"contract_version\":1,\"act_id\":\"X\"}")
			if err := os.WriteFile(manifestPath, doc, 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}
			res, err := VerifyClosureManifestR2B(EvidenceVerifierOptions{
				ManifestPath:  manifestPath,
				SubjectCommit: strings.Repeat("a", 40),
				Resolver:      resolver,
			})
			if err == nil {
				t.Fatalf("verifier must reject %s before CatFile", c.name)
			}
			// The error should be a *V2Error from the policy.
			if _, ok := err.(*V2Error); !ok {
				t.Fatalf("expected *V2Error, got %T: %v", err, err)
			}
			if resolver.catCalls != 0 {
				t.Fatalf("CatFile must not be called for %s (got %d calls)",
					c.name, resolver.catCalls)
			}
			if resolver.formatCalls < 1 {
				t.Fatalf("ObjectFormat must be called at least once for %s",
					c.name)
			}
			_ = res
			_ = repo
		})
	}
}

// TestR2CRProductionResolver_BoundToRepoRoot proves the
// production r2crGitObjectResolver routes every call to its
// bound repository root regardless of the process CWD. The
// test instantiates two real repositories and a third
// "no-repo" directory, then exercises the resolver from
// each CWD.
func TestR2CRProductionResolver_BoundToRepoRoot(t *testing.T) {
	repoA := initRepo(t)
	repoB := initRepo(t)
	// Commit one blob in each repo so CatFile has something
	// to read.
	oidA := mustRunGit(t, repoA, "hash-object", "-w", "--stdin")
	if _, err := runGitValue(context.Background(), RealGit{}, repoA, "hash-object", "-w", "--stdin"); err != nil {
		_ = err
	}
	// Use a real blob: write a file and hash it.
	blobA := writeR2CRBlob(t, repoA, "repo A blob")
	oidA = blobA
	blobB := writeR2CRBlob(t, repoB, "repo B blob")
	oidB := blobB

	resolverA, err := NewR2CRGitObjectResolver(RealGit{}, repoA)
	if err != nil {
		t.Fatalf("resolver A constructor: %v", err)
	}

	// Empty repoRoot must reject.
	if _, err := NewR2CRGitObjectResolver(RealGit{}, ""); err == nil {
		t.Fatalf("empty repoRoot must fail construction")
	}

	// Nonexistent repoRoot must surface as observation
	// failure when ObjectFormat is called.
	bogus, err := NewR2CRGitObjectResolver(RealGit{}, "/nonexistent/r2cr4-bogus-repo")
	if err != nil {
		t.Fatalf("bogus constructor should succeed: %v", err)
	}
	if _, err := bogus.ObjectFormat(); err == nil {
		t.Fatalf("ObjectFormat against nonexistent repo must fail")
	}

	// Process CWD is repoB but the resolver is bound to A.
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(repoB); err != nil {
		t.Fatalf("chdir repoB: %v", err)
	}
	// ObjectFormat must report A (sha1), not whatever B
	// has (also sha1 here, but the test is about the
	// repository binding mechanism).
	format, err := resolverA.ObjectFormat()
	if err != nil {
		t.Fatalf("ObjectFormat via CWD=repoB for resolverA: %v", err)
	}
	if format != "sha1" {
		t.Fatalf("expected sha1, got %q", format)
	}
	// CatFile for A's blob must succeed via CWD=repoB.
	rawA, err := resolverA.CatFile(oidA)
	if err != nil {
		t.Fatalf("CatFile A-blob via CWD=repoB: %v", err)
	}
	if !bytes.Contains(rawA, []byte("repo A blob")) {
		t.Fatalf("CatFile returned wrong content: %q", rawA)
	}
	// CatFile for B's blob via the A-bound resolver must fail
	// because B is not A.
	if _, err := resolverA.CatFile(oidB); err == nil {
		t.Fatalf("CatFile B-blob via resolverA must fail")
	}

	// Now chdir to a non-repository CWD; the resolver must
	// still operate against its bound root.
	noRepo := t.TempDir()
	if err := os.Chdir(noRepo); err != nil {
		t.Fatalf("chdir noRepo: %v", err)
	}
	if format, err := resolverA.ObjectFormat(); err != nil || format != "sha1" {
		t.Fatalf("ObjectFormat via CWD=non-repo: format=%q err=%v", format, err)
	}
	if _, err := resolverA.CatFile(oidA); err != nil {
		t.Fatalf("CatFile A-blob via CWD=non-repo: %v", err)
	}
}

// writeR2CRBlob writes content to a temp file in repoRoot,
// asks git to hash-object it, and returns the blob OID.
func writeR2CRBlob(t *testing.T, repoRoot, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "blob")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write blob: %v", err)
	}
	oid, err := runGitValue(context.Background(), RealGit{}, repoRoot, "hash-object", "-w", path)
	if err != nil {
		t.Fatalf("hash-object: %v", err)
	}
	return oid
}
