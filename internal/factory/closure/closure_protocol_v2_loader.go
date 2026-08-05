// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_loader.go provides the bounded
// repository-bound frozen-plan loader. The loader rejects
// caller-supplied paths that are not repository-relative,
// never reads plan bytes from mutable disk, and records the
// blob OID and SHA-256 of the exact bytes it returns.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// V2FrozenPlanBytes is the authoritative plan content bound to
// the freeze commit F. The blob OID and SHA-256 are recorded
// alongside the bytes so downstream manifests and verifiers
// can prove the bytes came from the object database.
type V2FrozenPlanBytes struct {
	Path         string
	Bytes        []byte
	BlobOID      string
	SHA256       string
	ByteCount    int
	FreezeCommit string
}

// V2FrozenPlanLoader loads exact frozen plan bytes from a
// freeze commit F via bounded git operations.
type V2FrozenPlanLoader interface {
	LoadFrozenPlan(ctx context.Context, repoRoot, freezeCommit, repoRelativePath string) (V2FrozenPlanBytes, error)
}

// GitV2FrozenPlanLoader is the production loader that uses
// the existing gitClient to talk to the object database.
type GitV2FrozenPlanLoader struct {
	Git gitClient
}

// NewGitV2FrozenPlanLoader constructs a loader that defaults
// to RealGit when no client is supplied.
func NewGitV2FrozenPlanLoader(g gitClient) *GitV2FrozenPlanLoader {
	if g == nil {
		g = RealGit{}
	}
	return &GitV2FrozenPlanLoader{Git: g}
}

// LoadFrozenPlan resolves F:PATH in the object database and
// returns the exact blob bytes. The function rejects:
//
//   - empty repository root
//   - empty freeze commit
//   - empty / absolute / traversal paths
//   - control characters, NUL, backslashes in the path
//   - revisions that do not resolve to a commit
//   - path resolutions that are not blobs
//   - blob OID / SHA-256 mismatches
//
// On success the bytes match the blob exactly and the SHA-256
// digest is computed from those bytes. Mutable disk bytes are
// never treated as authority: the loader never reads P from
// the working tree.
func (l *GitV2FrozenPlanLoader) LoadFrozenPlan(ctx context.Context, repoRoot, freezeCommit, repoRelativePath string) (V2FrozenPlanBytes, error) {
	if err := validateLoaderInputs(repoRoot, freezeCommit, repoRelativePath); err != nil {
		return V2FrozenPlanBytes{}, err
	}
	canonicalPath := filepath.ToSlash(filepath.Clean(repoRelativePath))
	canonicalPath = strings.TrimPrefix(canonicalPath, "./")
	if canonicalPath == "" || canonicalPath == "." {
		return V2FrozenPlanBytes{}, NewV2ErrorWith(V2CodeInvalidPlanPath,
			"plan path is empty after normalisation",
			"plan_path", repoRelativePath)
	}
	commitResolved, err := runGitValue(ctx, l.Git, repoRoot, "rev-parse", "--verify", "--end-of-options", freezeCommit+"^{commit}")
	if err != nil {
		return V2FrozenPlanBytes{}, NewV2ErrorWith(V2CodeFreezeCommitNotFound,
			fmt.Sprintf("freeze commit %q did not resolve: %s", freezeCommit, err.Error()),
			"freeze_commit", err.Error())
	}
	objectSpec := commitResolved + ":" + canonicalPath
	blobOID, err := runGitValue(ctx, l.Git, repoRoot, "rev-parse", "--verify", "--end-of-options", objectSpec)
	if err != nil {
		return V2FrozenPlanBytes{}, NewV2ErrorWith(V2CodeFrozenPlanPathMissing,
			fmt.Sprintf("plan path %q missing at freeze %s: %s", canonicalPath, commitResolved, err.Error()),
			"plan_path", err.Error())
	}
	objectType, err := runGitValue(ctx, l.Git, repoRoot, "cat-file", "-t", blobOID)
	if err != nil {
		return V2FrozenPlanBytes{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("git cat-file -t %s failed: %s", blobOID, err.Error()),
			"plan_path", err.Error())
	}
	if strings.TrimSpace(objectType) != "blob" {
		return V2FrozenPlanBytes{}, NewV2ErrorWith(V2CodeFrozenPlanNotBlob,
			fmt.Sprintf("plan path %q resolved to object type %q, expected blob", canonicalPath, objectType),
			"plan_path", objectType)
	}
	catResult := l.Git.Run(ctx, repoRoot, "cat-file", "blob", blobOID)
	if catResult.Err != nil || catResult.ExitCode != 0 {
		detail := strings.TrimSpace(string(catResult.Stderr))
		if detail == "" && catResult.Err != nil {
			detail = catResult.Err.Error()
		}
		return V2FrozenPlanBytes{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("git cat-file blob %s failed: %s", blobOID, detail),
			"plan_path", detail)
	}
	bytes := append([]byte(nil), catResult.Stdout...)
	sum := sha256.Sum256(bytes)
	sha := hex.EncodeToString(sum[:])
	return V2FrozenPlanBytes{
		Path:         canonicalPath,
		Bytes:        bytes,
		BlobOID:      blobOID,
		SHA256:       sha,
		ByteCount:    len(bytes),
		FreezeCommit: commitResolved,
	}, nil
}

// CompareToWorkingPlan compares the frozen bytes against an
// optional working-plan assertion. The assertion is supplied
// as raw bytes; an empty assertion disables the comparison.
//
// When the assertion does not match the frozen bytes the
// function returns a typed V2Error carrying
// V2CodeWorkingPlanMismatch.
func CompareToWorkingPlan(frozen V2FrozenPlanBytes, workingBytes []byte) error {
	if len(workingBytes) == 0 {
		return nil
	}
	if len(workingBytes) != frozen.ByteCount {
		return NewV2ErrorWith(V2CodeWorkingPlanMismatch,
			fmt.Sprintf("working plan byte count %d does not match frozen plan byte count %d",
				len(workingBytes), frozen.ByteCount),
			"plan_path", "")
	}
	workingSum := sha256.Sum256(workingBytes)
	workingSHA := hex.EncodeToString(workingSum[:])
	if workingSHA != frozen.SHA256 {
		return NewV2ErrorWith(V2CodeWorkingPlanMismatch,
			fmt.Sprintf("working plan SHA256 %s does not match frozen plan SHA256 %s",
				workingSHA, frozen.SHA256),
			"plan_path", "")
	}
	return nil
}

// ReadOptionalWorkingPlan reads the working plan bytes from
// the filesystem for use with CompareToWorkingPlan. An empty
// path disables the comparison and returns (nil, nil).
//
// The function deliberately reads mutable disk bytes so
// callers can compare them against frozen authority.
func ReadOptionalWorkingPlan(workingPath string) ([]byte, error) {
	if strings.TrimSpace(workingPath) == "" {
		return nil, nil
	}
	if !filepath.IsAbs(workingPath) {
		return nil, fmt.Errorf("working plan path must be absolute: %q", workingPath)
	}
	cleaned := filepath.Clean(workingPath)
	data, err := os.ReadFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("read working plan %s: %w", cleaned, err)
	}
	return data, nil
}

// validateLoaderInputs rejects empty / absolute / traversal
// / control-character paths before any git work runs.
func validateLoaderInputs(repoRoot, freezeCommit, repoRelativePath string) error {
	if strings.TrimSpace(repoRoot) == "" {
		return NewV2ErrorWith(V2CodeInvalidPlanPath, "repository root is empty", "repository_root", "")
	}
	if strings.TrimSpace(freezeCommit) == "" {
		return NewV2ErrorWith(V2CodeFreezeCommitNotFound, "freeze commit is empty", "freeze_commit", "")
	}
	if strings.TrimSpace(repoRelativePath) == "" {
		return NewV2ErrorWith(V2CodeInvalidPlanPath, "plan path is empty", "plan_path", "")
	}
	if filepath.IsAbs(repoRelativePath) {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("plan path %q must be repository-relative", repoRelativePath),
			"plan_path", repoRelativePath)
	}
	cleaned := filepath.ToSlash(filepath.Clean(repoRelativePath))
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." || strings.Contains(cleaned, "/../") {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("plan path %q escapes the repository", repoRelativePath),
			"plan_path", repoRelativePath)
	}
	if strings.ContainsRune(repoRelativePath, '\\') {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("plan path %q contains backslash", repoRelativePath),
			"plan_path", repoRelativePath)
	}
	if containsControl(repoRelativePath) {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("plan path %q contains control character", repoRelativePath),
			"plan_path", repoRelativePath)
	}
	if strings.HasPrefix(repoRelativePath, "-") {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("plan path %q must not start with '-'", repoRelativePath),
			"plan_path", repoRelativePath)
	}
	return nil
}

// containsControl returns true when the string contains any
// byte less than 0x20 or equals 0x7F. NUL bytes are rejected
// because they would split argv in downstream consumers.
func containsControl(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == 0x7F {
			return true
		}
	}
	return false
}
