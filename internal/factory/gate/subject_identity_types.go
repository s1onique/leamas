// Package gate provides subject identity collection for metrics.
package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// classifyWorktreeState determines if worktree is clean or dirty.
func classifyWorktreeState(inventory map[string]*inventoryEntry) string {
	for _, entry := range inventory {
		if entry.worktreeExists && entry.indexExists {
			if len(entry.worktreeContent) > 0 {
				h := sha256.Sum256(entry.worktreeContent)
				worktreeHash := hex.EncodeToString(h[:])
				if worktreeHash != entry.indexContentHash {
					return "dirty"
				}
			}
			if entry.worktreeMode != entry.indexMode {
				return "dirty"
			}
		}
		if entry.indexExists && !entry.worktreeExists {
			return "dirty"
		}
		if !entry.indexExists && entry.worktreeExists {
			return "dirty"
		}
	}
	return "clean"
}
