// Package gate provides subject identity collection for metrics.
package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// SubjectIdentity represents the measured subject's git state.
type SubjectIdentity struct {
	HeadOID            string
	TreeOID            string
	WorktreeState      string
	SubjectInputDigest string
}

// fileEntry represents one entry in the subject inventory.
type fileEntry struct {
	path       string
	entryType  string // "blob", "tree", "commit", "deleted"
	mode       string // file mode or ""
	target     string // symlink target or ""
	contentSHA string // SHA-256 of content (or empty for deleted)
}

// CollectSubjectIdentity computes identity from the repository at root.
// It produces a content-bound digest over tracked and nonignored untracked files.
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

	// Build content-bound inventory
	inventory, err := buildSubjectInventory(root)
	if err != nil {
		return nil, fmt.Errorf("build subject inventory: %w", err)
	}

	// Compute content-bound digest
	digest := computeContentBoundDigest(headOID, treeOID, inventory)

	return &SubjectIdentity{
		HeadOID:            headOID,
		TreeOID:            treeOID,
		WorktreeState:      "content-bound",
		SubjectInputDigest: digest,
	}, nil
}

// buildSubjectInventory constructs a sorted inventory of all nonignored files.
func buildSubjectInventory(root string) ([]fileEntry, error) {
	var entries []fileEntry

	// Get list of tracked files from HEAD
	trackedOut, err := runGit(root, "ls-tree", "-r", "--name-only", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("ls-tree HEAD: %w", err)
	}

	trackedFiles := strings.Split(strings.TrimSpace(trackedOut), "\n")

	// Process tracked files
	for _, path := range trackedFiles {
		if path == "" {
			continue
		}

		// Check if file is deleted in worktree
		deleted, err := isDeleted(root, path)
		if err != nil {
			return nil, err
		}

		if deleted {
			entries = append(entries, fileEntry{
				path:      path,
				entryType: "deleted",
			})
			continue
		}

		// Get file mode
		mode, err := getFileMode(root, path)
		if err != nil {
			return nil, err
		}

		// Check if file is staged
		stagedSHA, err := getStagedBlobSHA(root, path)
		if err != nil {
			return nil, err
		}

		// Check if file is modified in worktree
		worktreeSHA, err := getWorktreeBlobSHA(root, path)
		if err != nil {
			return nil, err
		}

		// Determine which content to hash
		contentSHA := stagedSHA
		if contentSHA == "" {
			contentSHA = worktreeSHA
		}

		// Check if symlink
		target := ""
		if mode == "120000" {
			target, _ = os.Readlink(filepath.Join(root, path))
		}

		entries = append(entries, fileEntry{
			path:       path,
			entryType:  "blob",
			mode:       mode,
			target:     target,
			contentSHA: contentSHA,
		})
	}

	// Process nonignored untracked files
	untrackedOut, err := runGit(root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	for _, line := range strings.Split(untrackedOut, "\n") {
		if len(line) < 4 {
			continue
		}
		// Untracked files have "??" in first two columns
		if line[0] == '?' && line[1] == '?' {
			path := strings.TrimSpace(line[3:])
			if path == "" || strings.Contains(path, " -> ") {
				continue
			}

			// Check if ignored
			ignored, err := isIgnored(root, path)
			if err != nil {
				return nil, err
			}
			if ignored {
				continue
			}

			// Hash untracked file content
			contentSHA, err := hashFileContent(root, path)
			if err != nil {
				return nil, err
			}

			entries = append(entries, fileEntry{
				path:       path,
				entryType:  "blob",
				mode:       "100644",
				contentSHA: contentSHA,
			})
		}
	}

	// Sort by path for deterministic ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	return entries, nil
}

// isDeleted checks if a tracked file is deleted in the worktree.
func isDeleted(root, path string) (bool, error) {
	out, err := runGit(root, "diff-files", "--name-status", "--", path)
	if err != nil {
		return false, err
	}
	trimmed := strings.TrimSpace(out)
	return trimmed != "" && strings.HasPrefix(trimmed, "D\t"), nil
}

// getFileMode returns the file mode from HEAD.
func getFileMode(root, path string) (string, error) {
	out, err := runGit(root, "ls-tree", "--format=%(objecttype) %(filemode)", "HEAD", "--", path)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(out)
	parts := strings.Split(trimmed, " ")
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return "100644", nil
}

// getStagedBlobSHA returns the SHA of staged content for a file.
func getStagedBlobSHA(root, path string) (string, error) {
	// Check if file is staged
	out, err := runGit(root, "diff-index", "--cached", "HEAD", "--", path)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return "", nil // not staged
	}

	// Parse the diff-index output to get staged blob SHA
	// Format: <old mode> <old sha> <new mode> <new sha> <status>\t<path>
	parts := strings.Split(trimmed, "\t")
	if len(parts) < 2 {
		return "", nil
	}
	headerParts := strings.Fields(parts[0])
	if len(headerParts) < 4 {
		return "", nil
	}
	sha := headerParts[3]
	if sha == "0000000000000000000000000000000000000000" {
		return "", nil
	}
	return sha, nil
}

// getWorktreeBlobSHA returns the SHA of worktree content for a file.
func getWorktreeBlobSHA(root, path string) (string, error) {
	// Use git hash-object to get SHA of current file content
	out, err := runGit(root, "hash-object", filepath.Join(root, path))
	if err != nil {
		// File may not exist or be inaccessible
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// hashFileContent computes SHA-256 of a file's content.
func hashFileContent(root, path string) (string, error) {
	fullPath := filepath.Join(root, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// isIgnored checks if a path is gitignored.
func isIgnored(root, path string) (bool, error) {
	out, err := runGit(root, "check-ignore", "-q", "--", path)
	// Exit code 0 means ignored, 1 means not ignored, >1 means error
	return err == nil && strings.TrimSpace(out) == path, nil
}

// computeContentBoundDigest creates a domain-separated SHA-256 digest over
// the complete subject inventory including file contents.
func computeContentBoundDigest(headOID, treeOID string, inventory []fileEntry) string {
	h := sha256.New()
	h.Write([]byte("factorize-subject-v1"))
	h.Write([]byte{0})
	h.Write([]byte(headOID))
	h.Write([]byte{0})
	h.Write([]byte(treeOID))
	h.Write([]byte{0})
	h.Write([]byte(fmt.Sprintf("%d", len(inventory))))
	h.Write([]byte{0})

	for _, entry := range inventory {
		h.Write([]byte(entry.path))
		h.Write([]byte{0})
		h.Write([]byte(entry.entryType))
		h.Write([]byte{0})
		h.Write([]byte(entry.mode))
		h.Write([]byte{0})
		h.Write([]byte(entry.target))
		h.Write([]byte{0})
		h.Write([]byte(entry.contentSHA))
		h.Write([]byte{0})
	}

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

// ComputeSubjectDigestForTest computes a content-bound digest for testing.
// This allows tests to verify the digest changes with different content.
func ComputeSubjectDigestForTest(headOID, treeOID string, contentChanges map[string]string) string {
	var entries []fileEntry
	for path, content := range contentChanges {
		entries = append(entries, fileEntry{
			path:       path,
			entryType:  "blob",
			mode:       "100644",
			contentSHA: content,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})
	return computeContentBoundDigest(headOID, treeOID, entries)
}
