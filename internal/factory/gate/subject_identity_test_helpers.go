// Package gate provides test helpers for subject identity.
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

	// Build complete working subject inventory
	inventory, err := buildWorkingSubjectInventory(root, metricsDestination)
	if err != nil {
		return nil, fmt.Errorf("build working subject inventory: %w", err)
	}

	// Classify worktree state
	worktreeState := classifyWorktreeState(inventory)

	// Compute content-bound digest
	digest := computeSubjectDigest(headOID, treeOID, worktreeState, inventory)

	return &SubjectIdentity{
		HeadOID:            headOID,
		TreeOID:            treeOID,
		WorktreeState:      worktreeState,
		SubjectInputDigest: digest,
	}, nil
}

// buildWorkingSubjectInventory constructs a complete inventory of the working subject.
func buildWorkingSubjectInventory(root string, metricsDestination string) (map[string]*inventoryEntry, error) {
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

	// Remove metrics destination
	if metricsDestination != "" {
		absMetrics, _ := filepath.Abs(metricsDestination)
		delete(allPaths, absMetrics)
		metricsDir := filepath.Dir(absMetrics)
		metricsBase := filepath.Base(absMetrics)
		for path := range allPaths {
			if strings.HasPrefix(filepath.Base(path), metricsBase) && filepath.Dir(path) == metricsDir {
				delete(allPaths, path)
			}
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

func getHEADPaths(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-rz", "--name-only", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	paths := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			result = append(result, p)
		}
	}
	return result, nil
}

func getIndexPaths(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--stage", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	entries := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		result = append(result, parts[1])
	}
	return result, nil
}

func getNonignoredUntrackedPaths(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	paths := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			result = append(result, p)
		}
	}
	return result, nil
}

func buildInventoryEntry(root, path string) (*inventoryEntry, error) {
	fullPath := filepath.Join(root, path)
	entry := &inventoryEntry{path: path, worktreeExists: false}

	info, err := os.Lstat(fullPath)
	if err == nil {
		entry.worktreeExists = true
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

type indexInfo struct {
	exists      bool
	contentHash string
	mode        string
	stage       int
}

func getIndexEntry(root, path string) (*indexInfo, error) {
	cmd := exec.Command("git", "ls-files", "--stage", "-z", "--", path)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	output := strings.TrimRight(string(out), "\x00")
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

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
