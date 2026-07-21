// Package gate provides subject identity collection for metrics.
package gate

import (
	"errors"
	"fmt"
	"os"
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
