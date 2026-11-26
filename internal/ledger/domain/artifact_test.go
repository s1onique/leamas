package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestAmendmentArtifactVerificationRejectsMissingArtifact witnesses the
// FWL01 amendment negative witness: a missing artifact must not acquire
// verified integrity.
func TestAmendmentArtifactVerificationRejectsMissingArtifact(t *testing.T) {
	dir := t.TempDir()
	const digest = "0000000000000000000000000000000000000000000000000000000000000000"
	_, err := VerifyArtifact(dir, "missing.txt", digest, 0)
	if err == nil {
		t.Fatal("a missing artifact was verified: want rejection")
	}
}

// TestAmendmentArtifactVerificationRejectsWrongDigest witnesses the
// FWL01 amendment negative witness: a wrong but syntactically valid
// SHA-256 must not acquire verified integrity.
func TestAmendmentArtifactVerificationRejectsWrongDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(path, []byte("payload\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	const wrong = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := VerifyArtifact(dir, "artifact.txt", wrong, 0); err == nil {
		t.Fatal("a wrong digest was verified: want rejection")
	}
}

// TestAmendmentArtifactVerificationRejectsSizeMismatch witnesses the byte
// size verification requirement.
func TestAmendmentArtifactVerificationRejectsSizeMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(path, []byte("payload\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	const digest = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := VerifyArtifact(dir, "artifact.txt", digest, 999); err == nil {
		t.Fatal("a size mismatch was verified: want rejection")
	}
}

// TestAmendmentArtifactVerificationRejectsSymlinkEscape witnesses the
// FWL01 amendment negative witness: a path escape through a symbolic link
// must be rejected.
//
// The artifact is reachable only through a symlink that points outside the
// repository root. It must not be readable through a safe
// repository-relative access.
func TestAmendmentArtifactVerificationRejectsSymlinkEscape(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("outside the repository\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	link := filepath.Join(repo, "escape.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks are not supported on this platform: %v", err)
	}
	const digest = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := VerifyArtifact(repo, "escape.txt", digest, 0); err == nil {
		t.Fatal("a symlink escape was verified: want rejection")
	}
}

// TestAmendmentArtifactVerificationAcceptsMatchingArtifact is the positive
// control for the artifact verification contract.
func TestAmendmentArtifactVerificationAcceptsMatchingArtifact(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("live qualification output\n")
	if err := os.WriteFile(filepath.Join(dir, "artifact.txt"), payload, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	digest := sha256Hex(payload)
	identity, err := VerifyArtifact(dir, "artifact.txt", digest, int64(len(payload)))
	if err != nil {
		t.Fatalf("a matching artifact was rejected: %v", err)
	}
	if identity.SHA256 != digest {
		t.Fatalf("expected digest %q, got %q", digest, identity.SHA256)
	}
	if identity.ByteSize != int64(len(payload)) {
		t.Fatalf("expected size %d, got %d", len(payload), identity.ByteSize)
	}
}

// sha256Hex returns the lowercase hexadecimal SHA-256 of a payload.
func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
