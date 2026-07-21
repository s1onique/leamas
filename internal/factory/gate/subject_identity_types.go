// Package gate provides subject identity collection for metrics.
package gate

import (
	"errors"
	"fmt"
	"os"
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

// inventoryEntry represents one entry in the subject inventory.
// Each entry records what the verifier actually scans.
type inventoryEntry struct {
	path             string
	worktreeExists   bool
	worktreeContent  []byte // nil if does not exist
	worktreeMode     string
	symlinkTarget    string
	indexExists      bool
	indexContentHash string
	indexMode        string
	indexStage       int
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

// classifyWorktreeStateWithGit uses git status to determine worktree cleanliness.
func classifyWorktreeStateWithGit(root, exactExclude, tempPrefix string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}

	output := strings.TrimRight(string(out), "\x00")
	if output == "" {
		return "clean", nil
	}

	// Check if any remaining entries are not exclusions
	entries := strings.Split(output, "\x00")
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		// Format: XY path (XY is status, path may contain spaces/tabs encoded as \x00)
		if len(entry) < 3 {
			continue
		}
		parts := strings.SplitN(entry, " ", 2)
		if len(parts) < 2 {
			continue
		}
		path := parts[1]

		// Skip exclusions
		if path == exactExclude {
			continue
		}
		if tempPrefix != "" && strings.HasPrefix(path, tempPrefix) {
			continue
		}

		// Any non-excluded change means dirty
		return "dirty", nil
	}

	return "clean", nil
}

// inspectWorkingSubjectPath inspects a path for the working subject inventory.
// Returns error for any failure except non-existence.
func inspectWorkingSubjectPath(path string) (exists bool, err error) {
	_, err = os.Lstat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("inspect working-subject path %q: %w", path, err)
	}
}
