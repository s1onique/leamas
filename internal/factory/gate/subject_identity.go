// Package gate provides subject identity collection for metrics.
package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

// SubjectIdentity represents the measured subject's git state.
type SubjectIdentity struct {
	HeadOID            string
	TreeOID            string
	WorktreeState      string
	SubjectInputDigest string
}

// CollectSubjectIdentity computes identity from the repository at root.
func CollectSubjectIdentity(root string) (*SubjectIdentity, error) {
	headOID, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	headOID = strings.TrimSpace(headOID)
	if headOID == "" {
		return nil, fmt.Errorf("HEAD OID is empty")
	}

	treeOID, err := runGit(root, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return nil, fmt.Errorf("git rev-parse HEAD^{tree}: %w", err)
	}
	treeOID = strings.TrimSpace(treeOID)
	if treeOID == "" {
		return nil, fmt.Errorf("tree OID is empty")
	}

	worktreeState, err := classifyWorktree(root)
	if err != nil {
		return nil, fmt.Errorf("classify worktree: %w", err)
	}

	digest := computeSubjectDigest(headOID, treeOID, worktreeState)

	return &SubjectIdentity{
		HeadOID:            headOID,
		TreeOID:            treeOID,
		WorktreeState:      worktreeState,
		SubjectInputDigest: digest,
	}, nil
}

// classifyWorktree determines the current worktree state.
func classifyWorktree(root string) (string, error) {
	// Check for untracked files
	statusOut, err := runGit(root, "status", "--porcelain")
	if err != nil {
		return "", err
	}

	hasUntracked := false
	for _, line := range strings.Split(statusOut, "\n") {
		if len(line) >= 2 && line[0] == '?' && line[1] == '?' {
			hasUntracked = true
			break
		}
	}

	// Check for staged changes
	stagedOut, err := runGit(root, "diff-index", "--cached", "HEAD")
	if err != nil {
		return "", err
	}
	hasStaged := strings.TrimSpace(stagedOut) != ""

	// Check for unstaged changes
	unstagedOut, err := runGit(root, "diff-files")
	if err != nil {
		return "", err
	}
	hasUnstaged := strings.TrimSpace(unstagedOut) != ""

	if hasUntracked {
		return "untracked", nil
	}
	if hasStaged {
		return "staged", nil
	}
	if hasUnstaged {
		return "modified", nil
	}
	return "clean", nil
}

// computeSubjectDigest creates a SHA-256 digest of the subject identity.
func computeSubjectDigest(headOID, treeOID, worktreeState string) string {
	h := sha256.New()
	h.Write([]byte("subject-v1"))
	h.Write([]byte{0})
	h.Write([]byte(headOID))
	h.Write([]byte{0})
	h.Write([]byte(treeOID))
	h.Write([]byte{0})
	h.Write([]byte(worktreeState))
	return hex.EncodeToString(h.Sum(nil))
}

// runGit executes a git command in the specified directory.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ValidateSubjectIdentity checks that subject identity is complete.
func ValidateSubjectIdentity(id *SubjectIdentity) error {
	if id == nil {
		return fmt.Errorf("subject identity is nil")
	}
	if id.HeadOID == "" {
		return fmt.Errorf("head OID is empty")
	}
	if id.TreeOID == "" {
		return fmt.Errorf("tree OID is empty")
	}
	if id.WorktreeState == "" {
		return fmt.Errorf("worktree state is empty")
	}
	if id.SubjectInputDigest == "" {
		return fmt.Errorf("subject digest is empty")
	}
	if len(id.SubjectInputDigest) != 64 {
		return fmt.Errorf("subject digest must be 64 hex chars, got %d", len(id.SubjectInputDigest))
	}
	return nil
}
