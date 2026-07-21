// Package gate provides subject identity collection for metrics.
package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// indexInfo holds git index information for a path.
type indexInfo struct {
	exists      bool
	contentHash string
	mode        string
	stage       int
}

// getIndexEntry returns index information for a specific path.
func getIndexEntry(root, path string) (*indexInfo, error) {
	out, err := runGit(root, "ls-files", "--stage", "-z", "--", path)
	if err != nil {
		return nil, err
	}
	output := strings.TrimRight(out, "\x00")
	if output == "" {
		return &indexInfo{exists: false}, nil
	}
	parts := strings.SplitN(output, "\t", 2)
	if len(parts) < 2 {
		return &indexInfo{exists: false}, nil
	}
	header := parts[0]
	filePath := parts[1]
	headerParts := strings.Fields(header)
	if len(headerParts) < 3 {
		return &indexInfo{exists: false}, nil
	}
	sha := headerParts[1]
	if sha == "0000000000000000000000000000000000000000" {
		return &indexInfo{exists: false}, nil
	}
	return &indexInfo{
		exists:      filePath == path,
		contentHash: sha,
		mode:        headerParts[0],
		stage:       0,
	}, nil
}

// metricsExclusions computes repository-relative exclusions for metrics destination.
func metricsExclusions(root, destination string) (exact string, tempPrefix string, err error) {
	if destination == "" {
		return "", "", nil
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}

	destAbs, err := filepath.Abs(destination)
	if err != nil {
		return "", "", err
	}

	rel, err := filepath.Rel(rootAbs, destAbs)
	if err != nil {
		return "", "", err
	}

	// Destination is outside the repository
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", nil
	}

	rel = filepath.ToSlash(rel)
	return rel, rel + ".tmp.", nil
}

// buildWorkingSubjectInventory constructs a complete inventory of the working subject.
func buildWorkingSubjectInventory(root, exactExclude, tempPrefix string) (map[string]*inventoryEntry, error) {
	inventory := make(map[string]*inventoryEntry)
	allPaths := make(map[string]bool)

	// Collect HEAD paths
	headPaths, err := getHEADPaths(root)
	if err != nil {
		return nil, fmt.Errorf("get HEAD paths: %w", err)
	}
	for _, p := range headPaths {
		allPaths[p] = true
	}

	// Collect index paths
	indexPaths, err := getIndexPaths(root)
	if err != nil {
		return nil, fmt.Errorf("get index paths: %w", err)
	}
	for _, p := range indexPaths {
		allPaths[p] = true
	}

	// Collect nonignored untracked paths
	untrackedPaths, err := getNonignoredUntrackedPaths(root)
	if err != nil {
		return nil, fmt.Errorf("get untracked paths: %w", err)
	}
	for _, p := range untrackedPaths {
		allPaths[p] = true
	}

	// Remove exclusions
	for path := range allPaths {
		if path == exactExclude {
			delete(allPaths, path)
			continue
		}
		if tempPrefix != "" && strings.HasPrefix(path, tempPrefix) {
			delete(allPaths, path)
		}
	}

	// Build inventory entry for each path
	for path := range allPaths {
		entry, err := buildInventoryEntry(root, path)
		if err != nil {
			return nil, fmt.Errorf("build entry for %s: %w", path, err)
		}
		inventory[path] = entry
	}

	return inventory, nil
}

func buildInventoryEntry(root, path string) (*inventoryEntry, error) {
	fullPath := filepath.Join(root, path)
	entry := &inventoryEntry{path: path, worktreeExists: false}

	// Use proper error handling for Lstat
	exists, err := inspectWorkingSubjectPath(fullPath)
	if err != nil {
		return nil, err
	}
	entry.worktreeExists = exists

	if exists {
		info, err := os.Lstat(fullPath)
		if err != nil {
			return nil, err
		}
		entry.worktreeMode = formatMode(info.Mode())
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(fullPath)
			if err != nil {
				return nil, err
			}
			entry.symlinkTarget = target
		} else if info.Mode().IsRegular() {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, err
			}
			entry.worktreeContent = content
		}
	}

	indexInfo, err := getIndexEntry(root, path)
	if err != nil {
		return nil, err
	}
	if indexInfo != nil {
		entry.indexExists = indexInfo.exists
		entry.indexContentHash = indexInfo.contentHash
		entry.indexMode = indexInfo.mode
		entry.indexStage = indexInfo.stage
	}

	return entry, nil
}

func formatMode(m os.FileMode) string {
	if m.IsRegular() {
		if m&0111 != 0 {
			return "100755"
		}
		return "100644"
	}
	if m&os.ModeSymlink != 0 {
		return "120000"
	}
	if m&os.ModeDir != 0 {
		return "040000"
	}
	return "000000"
}

func computeSubjectDigest(headOID, treeOID, worktreeState string, inventory map[string]*inventoryEntry) string {
	h := sha256.New()
	h.Write([]byte("factorize-subject-v2"))
	h.Write([]byte{0})
	h.Write([]byte(headOID))
	h.Write([]byte{0})
	h.Write([]byte(treeOID))
	h.Write([]byte{0})
	h.Write([]byte(worktreeState))
	h.Write([]byte{0})

	paths := make([]string, 0, len(inventory))
	for p := range inventory {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	h.Write([]byte(fmt.Sprintf("%d", len(paths))))
	h.Write([]byte{0})

	for _, path := range paths {
		entry := inventory[path]
		h.Write([]byte(path))
		h.Write([]byte{0})
		if entry.worktreeExists {
			h.Write([]byte("exists"))
			h.Write([]byte{0})
			h.Write([]byte(entry.worktreeMode))
			h.Write([]byte{0})
			if entry.symlinkTarget != "" {
				h.Write([]byte(entry.symlinkTarget))
				h.Write([]byte{0})
			}
			if len(entry.worktreeContent) > 0 {
				contentHash := sha256.Sum256(entry.worktreeContent)
				h.Write([]byte(hex.EncodeToString(contentHash[:])))
				h.Write([]byte{0})
			} else {
				h.Write([]byte("0"))
				h.Write([]byte{0})
			}
		} else {
			h.Write([]byte("deleted"))
			h.Write([]byte{0})
		}
		if entry.indexExists {
			h.Write([]byte("indexed"))
			h.Write([]byte{0})
			h.Write([]byte(entry.indexMode))
			h.Write([]byte{0})
			h.Write([]byte(entry.indexContentHash))
			h.Write([]byte{0})
			h.Write([]byte(fmt.Sprintf("%d", entry.indexStage)))
			h.Write([]byte{0})
		} else {
			h.Write([]byte("untracked"))
			h.Write([]byte{0})
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// classifyWorktreeStateWithGit uses git status to determine worktree cleanliness.
func classifyWorktreeStateWithGit(root, exactExclude, tempPrefix string) (string, error) {
	out, err := runGit(root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}

	output := strings.TrimRight(out, "\x00")
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

// CollectSubjectIdentity computes identity from the repository at root.
// It produces a content-bound digest over the complete working subject.
// The metricsDestination, if provided, is excluded from the inventory.
func CollectSubjectIdentity(root string, metricsDestination string) (*SubjectIdentity, error) {
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

	// Compute exclusions for metrics destination
	exactExcl, tempPrefix, err := metricsExclusions(root, metricsDestination)
	if err != nil {
		return nil, fmt.Errorf("compute metrics exclusions: %w", err)
	}

	// Build complete working subject inventory
	inventory, err := buildWorkingSubjectInventory(root, exactExcl, tempPrefix)
	if err != nil {
		return nil, fmt.Errorf("build working subject inventory: %w", err)
	}

	// Classify worktree state using git status
	worktreeState, err := classifyWorktreeStateWithGit(root, exactExcl, tempPrefix)
	if err != nil {
		return nil, fmt.Errorf("classify worktree state: %w", err)
	}

	// Compute content-bound digest
	digest := computeSubjectDigest(headOID, treeOID, worktreeState, inventory)

	return &SubjectIdentity{
		HeadOID:            headOID,
		TreeOID:            treeOID,
		WorktreeState:      worktreeState,
		SubjectInputDigest: digest,
	}, nil
}
