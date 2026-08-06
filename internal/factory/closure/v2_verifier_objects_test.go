// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_objects_test.go covers the byte authority
// matrix required by Phases 4-6 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-TOPOLOGY-OBJECTS01.
//
// Every case constructs a real Git repository in a temp
// directory via the existing closure-package test helpers
// and asserts:
//
//   - F:P blob bytes are read verbatim (trailing newline,
//     leading whitespace, trailing spaces, non-ASCII UTF-8)
//   - F:P SHA-256 matches SHA-256 of the literal bytes
//   - F:P blob OID matches `git rev-parse F:P`
//   - C:M blob bytes are read verbatim
//   - C:M blob OID matches `git rev-parse C:M`
//   - missing F:P / C:M / non-blob paths reject with typed
//     diagnostics
//   - optional manifest assertion: match / mismatch /
//     empty authority / missing bytes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV2VerifierFrozenPlanAuthorityRoundTrip proves the
// verifier resolves F:P bytes verbatim and surfaces the
// correct OID + SHA-256. The test writes a plan file with a
// trailing newline in F and asserts:
//
//   - BlobOID matches `git rev-parse F:PATH`
//   - BlobSHA256 matches SHA-256 of the literal bytes
//   - RawBytes preserves the trailing newline
func TestV2VerifierFrozenPlanAuthorityRoundTrip(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject implementation\n",
	})
	planContent := `{"contract_version":1,"checks":[]}` + "\n"
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/PLAN.json": planContent,
	})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	authority, err := ResolveV2FrozenPlanAuthority(
		context.Background(), auth, freeze,
		"docs/closure-plans/PLAN.json")
	if err != nil {
		t.Fatalf("ResolveV2FrozenPlanAuthority: %v", err)
	}
	if len(authority.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", authority.Diagnostics.Codes())
	}
	if authority.Path != "docs/closure-plans/PLAN.json" {
		t.Fatalf("Path = %q, want docs/closure-plans/PLAN.json", authority.Path)
	}
	wantOID := mustRunGit(t, dir, "rev-parse", freeze+":docs/closure-plans/PLAN.json")
	if authority.BlobOID != wantOID {
		t.Fatalf("BlobOID = %q, want %q", authority.BlobOID, wantOID)
	}
	wantSum := sha256.Sum256([]byte(planContent))
	wantSHA := hex.EncodeToString(wantSum[:])
	if authority.BlobSHA256 != wantSHA {
		t.Fatalf("BlobSHA256 = %q, want %q", authority.BlobSHA256, wantSHA)
	}
	if !bytes.Equal(authority.RawBytes, []byte(planContent)) {
		t.Fatalf("RawBytes = %q, want %q", authority.RawBytes, planContent)
	}
	if !bytes.HasSuffix(authority.RawBytes, []byte{'\n'}) {
		t.Fatalf("RawBytes lost trailing newline: %q", authority.RawBytes)
	}
	_ = subject
}

// TestV2VerifierFrozenPlanAuthorityRawBytesPreservation
// proves the byte-authority contract for trailing newlines,
// leading whitespace, trailing spaces, and non-ASCII UTF-8.
// The verifier MUST NOT trim or normalise the bytes so
// SHA-256 equals the manifest binding.
func TestV2VerifierFrozenPlanAuthorityRawBytesPreservation(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{
			name:    "trailing newline",
			content: "trailing\n",
		},
		{
			name:    "leading whitespace",
			content: "  leading prefix\n",
		},
		{
			name:    "trailing spaces",
			content: "trailing suffix   \n",
		},
		{
			name:    "non-ASCII UTF-8",
			content: `{"name":"\u00e9\u00e8\u00ea"}` + "\n",
		},
		{
			name:    "all four combined",
			content: " \t  unicode: \u00e9\u00e8\u00ea \n trailing spaces   \n",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := initRepo(t)
			makeCommit(t, dir, "subject", map[string]string{
				"subject.txt": "subject\n",
			})
			freeze := makeCommit(t, dir, "freeze", map[string]string{
				"plan.json": tc.content,
			})
			auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
			if err != nil {
				t.Fatalf("authority: %v", err)
			}
			authority, err := ResolveV2FrozenPlanAuthority(
				context.Background(), auth, freeze, "plan.json")
			if err != nil {
				t.Fatalf("ResolveV2FrozenPlanAuthority: %v", err)
			}
			if len(authority.Diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %v", authority.Diagnostics.Codes())
			}
			if !bytes.Equal(authority.RawBytes, []byte(tc.content)) {
				t.Fatalf("RawBytes drift: got %q, want %q",
					authority.RawBytes, tc.content)
			}
			gotSum := sha256.Sum256(authority.RawBytes)
			wantSum := sha256.Sum256([]byte(tc.content))
			if gotSum != wantSum {
				t.Fatalf("SHA-256 drift")
			}
			if hex.EncodeToString(gotSum[:]) != authority.BlobSHA256 {
				t.Fatalf("BlobSHA256 = %q, want %q",
					authority.BlobSHA256, hex.EncodeToString(gotSum[:]))
			}
		})
	}
}

// TestV2VerifierFrozenPlanAuthorityMissingPath rejects
// ResolveV2FrozenPlanAuthority when the plan path does not
// exist at F. The diagnostic is typed
// frozen_plan_missing.
func TestV2VerifierFrozenPlanAuthorityMissingPath(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
	freeze := makeCommit(t, dir, "freeze", map[string]string{"b.txt": "B\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	authority, err := ResolveV2FrozenPlanAuthority(
		context.Background(), auth, freeze, "missing/plan.json")
	if err != nil {
		t.Fatalf("ResolveV2FrozenPlanAuthority: %v", err)
	}
	if len(authority.Diagnostics) == 0 {
		t.Fatalf("missing plan path must emit diagnostics")
	}
	if !authority.Diagnostics.HasCode(V2VerifierFrozenPlanMissing) {
		t.Fatalf("diagnostic codes = %v, want frozen_plan_missing",
			authority.Diagnostics.Codes())
	}
	if len(authority.RawBytes) != 0 {
		t.Fatalf("missing plan path must not return raw bytes")
	}
}

// TestV2VerifierFrozenPlanAuthorityNonBlobPath rejects
// ResolveV2FrozenPlanAuthority when the path resolves to a
// tree (a directory) instead of a blob. The diagnostic is
// typed frozen_plan_not_blob.
func TestV2VerifierFrozenPlanAuthorityNonBlobPath(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/PLAN.json": "{}",
	})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	// Path "docs" is a tree at F, not a blob.
	authority, err := ResolveV2FrozenPlanAuthority(
		context.Background(), auth, freeze, "docs")
	if err != nil {
		t.Fatalf("ResolveV2FrozenPlanAuthority: %v", err)
	}
	if !authority.Diagnostics.HasCode(V2VerifierFrozenPlanNotBlob) {
		t.Fatalf("diagnostic codes = %v, want frozen_plan_not_blob",
			authority.Diagnostics.Codes())
	}
	if len(authority.RawBytes) != 0 {
		t.Fatalf("non-blob path must not return raw bytes")
	}
}

// TestV2VerifierFrozenPlanAuthorityEmptyInputs rejects
// ResolveV2FrozenPlanAuthority when the freeze commit or
// the plan path is empty. The diagnostic is typed
// frozen_plan_missing.
func TestV2VerifierFrozenPlanAuthorityEmptyInputs(t *testing.T) {
	dir := initRepo(t)
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}

	_, err = ResolveV2FrozenPlanAuthority(
		context.Background(), auth, "", "plan.json")
	if err == nil {
		t.Fatalf("empty freeze commit must fail")
	}
	if !strings.Contains(err.Error(), "frozen_plan_missing") {
		t.Fatalf("error = %q, want frozen_plan_missing code", err.Error())
	}

	_, err = ResolveV2FrozenPlanAuthority(
		context.Background(), auth, "abc123", "")
	if err == nil {
		t.Fatalf("empty plan path must fail")
	}
	if !strings.Contains(err.Error(), "frozen_plan_missing") {
		t.Fatalf("error = %q, want frozen_plan_missing code", err.Error())
	}
}

// TestV2VerifierFrozenPlanAuthorityNilAuthority rejects
// ResolveV2FrozenPlanAuthority when the authority is nil.
// The function returns a typed V2VerifierError.
func TestV2VerifierFrozenPlanAuthorityNilAuthority(t *testing.T) {
	_, err := ResolveV2FrozenPlanAuthority(
		context.Background(), nil, "abc", "plan.json")
	if err == nil {
		t.Fatalf("nil authority must fail")
	}
	if !strings.Contains(err.Error(), "frozen_plan_missing") {
		t.Fatalf("error = %q, want frozen_plan_missing code", err.Error())
	}
}

// TestV2VerifierCommittedManifestAuthorityRoundTrip proves
// the verifier resolves C:M bytes verbatim and surfaces the
// correct OID + SHA-256.
func TestV2VerifierCommittedManifestAuthorityRoundTrip(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
	makeCommit(t, dir, "freeze", map[string]string{"b.txt": "B\n"})
	manifestContent := `{"closure_protocol_version":"2","plan_contract_version":1}` + "\n"
	closure := makeCommit(t, dir, "closure", map[string]string{
		"docs/closure-manifests/MANIFEST.json": manifestContent,
	})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	authority, err := ResolveV2CommittedManifestAuthority(
		context.Background(), auth, closure,
		"docs/closure-manifests/MANIFEST.json")
	if err != nil {
		t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
	}
	if len(authority.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", authority.Diagnostics.Codes())
	}
	wantOID := mustRunGit(t, dir, "rev-parse", closure+":docs/closure-manifests/MANIFEST.json")
	if authority.BlobOID != wantOID {
		t.Fatalf("BlobOID = %q, want %q", authority.BlobOID, wantOID)
	}
	wantSum := sha256.Sum256([]byte(manifestContent))
	if authority.BlobSHA256 != hex.EncodeToString(wantSum[:]) {
		t.Fatalf("BlobSHA256 = %q, want %q",
			authority.BlobSHA256, hex.EncodeToString(wantSum[:]))
	}
	if !bytes.Equal(authority.RawBytes, []byte(manifestContent)) {
		t.Fatalf("RawBytes drift: got %q, want %q",
			authority.RawBytes, manifestContent)
	}
}

// TestV2VerifierCommittedManifestAuthorityRawBytesPreservation
// proves the byte contract for the committed manifest
// authority path. The verifier MUST preserve the exact
// bytes so SHA-256 equals the optional disk assertion
// binding.
func TestV2VerifierCommittedManifestAuthorityRawBytesPreservation(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{name: "trailing newline", content: "manifest\n"},
		{name: "leading whitespace", content: "   leading manifest\n"},
		{name: "trailing spaces", content: "trailing spaces   \n"},
		{name: "non-ASCII UTF-8", content: `{"name":"\u00e9\u00e8\u00ea"}` + "\n"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := initRepo(t)
			makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
			makeCommit(t, dir, "freeze", map[string]string{"b.txt": "B\n"})
			closure := makeCommit(t, dir, "closure", map[string]string{
				"manifest.json": tc.content,
			})
			auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
			if err != nil {
				t.Fatalf("authority: %v", err)
			}
			authority, err := ResolveV2CommittedManifestAuthority(
				context.Background(), auth, closure, "manifest.json")
			if err != nil {
				t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
			}
			if !bytes.Equal(authority.RawBytes, []byte(tc.content)) {
				t.Fatalf("RawBytes drift: got %q, want %q",
					authority.RawBytes, tc.content)
			}
			gotSum := sha256.Sum256(authority.RawBytes)
			if hex.EncodeToString(gotSum[:]) != authority.BlobSHA256 {
				t.Fatalf("BlobSHA256 drift")
			}
		})
	}
}

// TestV2VerifierCommittedManifestAuthorityMissingPath
// rejects ResolveV2CommittedManifestAuthority when the
// manifest path does not exist at C. The diagnostic is
// typed closure_manifest_missing.
func TestV2VerifierCommittedManifestAuthorityMissingPath(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
	makeCommit(t, dir, "freeze", map[string]string{"b.txt": "B\n"})
	closure := makeCommit(t, dir, "closure", map[string]string{"c.txt": "C\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	authority, err := ResolveV2CommittedManifestAuthority(
		context.Background(), auth, closure, "missing/manifest.json")
	if err != nil {
		t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
	}
	if !authority.Diagnostics.HasCode(V2VerifierClosureManifestMissing) {
		t.Fatalf("diagnostic codes = %v, want closure_manifest_missing",
			authority.Diagnostics.Codes())
	}
	if len(authority.RawBytes) != 0 {
		t.Fatalf("missing manifest path must not return raw bytes")
	}
}

// TestV2VerifierCommittedManifestAuthorityNonBlobPath
// rejects ResolveV2CommittedManifestAuthority when the path
// resolves to a tree. The diagnostic is typed
// closure_manifest_not_blob.
func TestV2VerifierCommittedManifestAuthorityNonBlobPath(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
	makeCommit(t, dir, "freeze", map[string]string{"b.txt": "B\n"})
	closure := makeCommit(t, dir, "closure", map[string]string{
		"docs/closure-manifests/MANIFEST.json": "{}",
	})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	authority, err := ResolveV2CommittedManifestAuthority(
		context.Background(), auth, closure, "docs")
	if err != nil {
		t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
	}
	if !authority.Diagnostics.HasCode(V2VerifierClosureManifestNotBlob) {
		t.Fatalf("diagnostic codes = %v, want closure_manifest_not_blob",
			authority.Diagnostics.Codes())
	}
}

// TestV2VerifierCommittedManifestAuthorityEmptyBlob
// rejects ResolveV2CommittedManifestAuthority when the
// manifest blob is empty. The diagnostic is typed
// closure_manifest_read_failed.
func TestV2VerifierCommittedManifestAuthorityEmptyBlob(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
	makeCommit(t, dir, "freeze", map[string]string{"b.txt": "B\n"})
	// Make a commit with an empty manifest blob.
	emptyPath := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(emptyPath, []byte{}, 0o644); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	mustRunGit(t, dir, "add", "empty.json")
	mustRunGit(t, dir, "commit", "-m", "empty manifest")
	closure := mustRunGit(t, dir, "rev-parse", "HEAD")

	// Sanity check: confirm the empty blob is reachable at
	// the closure commit so the test exercises the
	// empty-bytes path (not the missing-path path).
	emptyOID := mustRunGit(t, dir, "rev-parse", closure+":empty.json")
	if emptyOID != "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391" {
		t.Fatalf("empty.json blob OID = %q, want the well-known empty-blob OID", emptyOID)
	}

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	authority, err := ResolveV2CommittedManifestAuthority(
		context.Background(), auth, closure, "empty.json")
	if err != nil {
		t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
	}
	if !authority.Diagnostics.HasCode(V2VerifierClosureManifestReadFailed) {
		t.Fatalf("diagnostic codes = %v, want closure_manifest_read_failed",
			authority.Diagnostics.Codes())
	}
}

// TestV2VerifierOptionalManifestAssertionMatch proves the
// optional assertion accepts caller bytes that match C:M.
func TestV2VerifierOptionalManifestAssertionMatch(t *testing.T) {
	authority := V2CommittedManifestAuthority{
		Path:       "manifest.json",
		RawBytes:   []byte("manifest content\n"),
		BlobSHA256: "deadbeef",
	}
	result := AssertV2OptionalManifestAssertion(
		authority, []byte("manifest content\n"))
	if !result.Supplied || !result.Matches {
		t.Fatalf("matching assertion must report Supplied=true Matches=true, got %+v", result)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("matching assertion must emit no diagnostics, got %v",
			result.Diagnostics.Codes())
	}
}

// TestV2VerifierOptionalManifestAssertionMismatch proves
// the optional assertion rejects mismatched caller bytes
// with the typed closure_manifest_assertion_mismatch
// diagnostic.
func TestV2VerifierOptionalManifestAssertionMismatch(t *testing.T) {
	authority := V2CommittedManifestAuthority{
		Path:       "manifest.json",
		RawBytes:   []byte("committed manifest\n"),
		BlobSHA256: hex.EncodeToString(sha256.New().Sum(nil)),
	}
	result := AssertV2OptionalManifestAssertion(
		authority, []byte("different working copy\n"))
	if !result.Supplied || result.Matches {
		t.Fatalf("mismatching assertion must report Supplied=true Matches=false, got %+v", result)
	}
	if !result.Diagnostics.HasCode(V2VerifierClosureManifestAssertionMismatch) {
		t.Fatalf("diagnostic codes = %v, want closure_manifest_assertion_mismatch",
			result.Diagnostics.Codes())
	}
	// Expected and observed should be populated.
	first := result.Diagnostics[0]
	if first.Expected == "" || first.Observed == "" {
		t.Fatalf("mismatch diagnostic must populate expected/observed: %+v", first)
	}
}

// TestV2VerifierOptionalManifestAssertionNotSupplied
// proves the verifier treats a missing optional assertion as
// not supplied and never rejects.
func TestV2VerifierOptionalManifestAssertionNotSupplied(t *testing.T) {
	authority := V2CommittedManifestAuthority{
		Path:     "manifest.json",
		RawBytes: []byte("committed manifest\n"),
	}
	result := AssertV2OptionalManifestAssertion(authority, nil)
	if result.Supplied {
		t.Fatalf("nil optional bytes must report Supplied=false")
	}
	if result.Matches {
		t.Fatalf("nil optional bytes must report Matches=false")
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("nil optional bytes must emit no diagnostics, got %v",
			result.Diagnostics.Codes())
	}
}

// TestV2VerifierOptionalManifestAssertionEmptyAuthority
// proves the assertion reports a mismatch (not a panic)
// when the authority bytes are empty but optional bytes
// were supplied.
func TestV2VerifierOptionalManifestAssertionEmptyAuthority(t *testing.T) {
	authority := V2CommittedManifestAuthority{
		Path:     "manifest.json",
		RawBytes: nil,
	}
	result := AssertV2OptionalManifestAssertion(
		authority, []byte("supplied bytes\n"))
	if !result.Supplied || result.Matches {
		t.Fatalf("empty authority + supplied bytes must report mismatch, got %+v", result)
	}
	if !result.Diagnostics.HasCode(V2VerifierClosureManifestAssertionMismatch) {
		t.Fatalf("diagnostic codes = %v, want closure_manifest_assertion_mismatch",
			result.Diagnostics.Codes())
	}
}

// TestV2VerifierOptionalManifestAssertionAuthorityBytesUnchanged
// proves the verifier does NOT modify the authority bytes
// when the optional assertion matches, mismatches, or is
// absent.
func TestV2VerifierOptionalManifestAssertionAuthorityBytesUnchanged(t *testing.T) {
	original := []byte("committed manifest bytes\n")
	authority := V2CommittedManifestAuthority{
		Path:     "manifest.json",
		RawBytes: append([]byte(nil), original...),
	}

	// Match.
	_ = AssertV2OptionalManifestAssertion(authority, original)
	if !bytes.Equal(authority.RawBytes, original) {
		t.Fatalf("match path mutated authority bytes")
	}

	// Mismatch.
	_ = AssertV2OptionalManifestAssertion(authority, []byte("different\n"))
	if !bytes.Equal(authority.RawBytes, original) {
		t.Fatalf("mismatch path mutated authority bytes")
	}

	// Not supplied.
	_ = AssertV2OptionalManifestAssertion(authority, nil)
	if !bytes.Equal(authority.RawBytes, original) {
		t.Fatalf("not-supplied path mutated authority bytes")
	}
}
