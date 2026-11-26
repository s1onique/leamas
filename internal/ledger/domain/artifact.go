package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Artifact authority errors.
var (
	// ErrArtifactMissing is returned when the declared artifact does not
	// exist at the declared repository-relative path.
	ErrArtifactMissing = errors.New("evidence artifact is missing")

	// ErrArtifactUnreadable is returned when the declared artifact cannot
	// be read safely.
	ErrArtifactUnreadable = errors.New("evidence artifact is unreadable")

	// ErrDigestMismatch is returned when the artifact content does not
	// match the declared SHA-256 digest.
	ErrDigestMismatch = errors.New("evidence artifact digest mismatch")

	// ErrSizeMismatch is returned when the artifact size does not match
	// the declared byte size.
	ErrSizeMismatch = errors.New("evidence artifact size mismatch")
)

// ArtifactIdentity is the verified identity of one repository artifact.
type ArtifactIdentity struct {
	// RelativePath is the repository-relative path of the artifact.
	RelativePath string
	// SHA256 is the computed lowercase hexadecimal digest.
	SHA256 string
	// ByteSize is the computed size in bytes.
	ByteSize int64
}

// VerifyArtifact verifies the actual identity of a repository artifact.
//
// The artifact is opened through a repository-relative path only. A path
// that escapes the repository root — directly or through a symbolic link —
// is rejected before any byte is read.
//
// The verification is mechanical: it proves that the bytes at the declared
// path are the declared bytes. It never verifies an engineering claim.
func VerifyArtifact(repoRoot, relPath, wantDigest string, wantSize int64) (ArtifactIdentity, error) {
	if err := ValidateRelativePath(relPath); err != nil {
		return ArtifactIdentity{}, err
	}
	if err := ValidateSHA256(wantDigest); err != nil {
		return ArtifactIdentity{}, err
	}

	abs, err := safeRepositoryPath(repoRoot, relPath)
	if err != nil {
		return ArtifactIdentity{}, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return ArtifactIdentity{}, fmt.Errorf("%w: %s", ErrArtifactMissing, relPath)
		}
		return ArtifactIdentity{}, fmt.Errorf("%w: %s: %v", ErrArtifactUnreadable, relPath, err)
	}
	if info.IsDir() {
		return ArtifactIdentity{}, fmt.Errorf("%w: %s is a directory", ErrArtifactUnreadable, relPath)
	}
	if !info.Mode().IsRegular() {
		return ArtifactIdentity{}, fmt.Errorf("%w: %s is not a regular file", ErrArtifactUnreadable, relPath)
	}

	if wantSize > 0 && info.Size() != wantSize {
		return ArtifactIdentity{}, fmt.Errorf("%w: %s: expected %d bytes, found %d",
			ErrSizeMismatch, relPath, wantSize, info.Size())
	}

	f, err := os.Open(abs)
	if err != nil {
		return ArtifactIdentity{}, fmt.Errorf("%w: %s: %v", ErrArtifactUnreadable, relPath, err)
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return ArtifactIdentity{}, fmt.Errorf("%w: %s: %v", ErrArtifactUnreadable, relPath, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != wantDigest {
		return ArtifactIdentity{}, fmt.Errorf("%w: %s: expected %s, computed %s",
			ErrDigestMismatch, relPath, wantDigest, got)
	}

	return ArtifactIdentity{
		RelativePath: relPath,
		SHA256:       got,
		ByteSize:     n,
	}, nil
}

// safeRepositoryPath resolves a repository-relative path and proves that
// the result stays inside the repository root.
//
// Both the lexical form and the resolved form are checked. A symbolic link
// that points outside the repository is rejected: the artifact is not
// reachable through a safe repository-relative access.
func safeRepositoryPath(repoRoot, relPath string) (string, error) {
	if repoRoot == "" {
		return "", fmt.Errorf("%w: empty repository root", ErrArtifactUnreadable)
	}
	rootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrArtifactUnreadable, err)
	}
	// Resolve the root itself: the repository root may legitimately be
	// reached through a symbolic link.
	if resolvedRoot, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootAbs = resolvedRoot
	}

	target := filepath.Join(rootAbs, filepath.FromSlash(relPath))
	if !withinRoot(rootAbs, target) {
		return "", fmt.Errorf("%w: %s escapes the repository root", ErrArtifactUnreadable, relPath)
	}

	// The deepest existing ancestor is resolved, so a symlinked directory
	// component cannot smuggle the target outside the repository.
	probe := target
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			if !withinRoot(rootAbs, resolved) {
				return "", fmt.Errorf("%w: %s escapes the repository root", ErrArtifactUnreadable, relPath)
			}
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	return target, nil
}

// withinRoot reports whether target is inside root.
func withinRoot(root, target string) bool {
	if target == root {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}
