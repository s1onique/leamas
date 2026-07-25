// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	v2EvidenceIndexName = "evidence-index.json"
	v2EvidenceIndexV1   = 1
)

type v2EvidenceIndex struct {
	ContractVersion int                    `json:"contract_version"`
	Entries         []v2EvidenceIndexEntry `json:"entries"`
}

type v2EvidenceIndexEntry struct {
	RelativePath string `json:"relative_path"`
	MediaType    string `json:"media_type"`
	ByteSize     int64  `json:"byte_size"`
	SHA256       string `json:"sha256"`
}

type v2QualifiedEvidence struct {
	IndexBytes  []byte
	IndexSHA256 string
	FinalPath   string
}

// buildV2EvidenceIndex writes and verifies a canonical index in staging.
func buildV2EvidenceIndex(stagingPath string) (v2QualifiedEvidence, error) {
	entries, err := scanV2Evidence(stagingPath)
	if err != nil {
		return v2QualifiedEvidence{}, err
	}
	indexBytes, err := marshalCanonicalJSON(v2EvidenceIndex{ContractVersion: v2EvidenceIndexV1, Entries: entries})
	if err != nil {
		return v2QualifiedEvidence{}, fmt.Errorf("marshal evidence index: %w", err)
	}
	indexPath := filepath.Join(stagingPath, v2EvidenceIndexName)
	if err := writeExclusiveRegular(indexPath, indexBytes, 0o600); err != nil {
		return v2QualifiedEvidence{}, fmt.Errorf("write evidence index: %w", err)
	}
	if err := verifyV2EvidenceIndex(stagingPath, indexBytes); err != nil {
		return v2QualifiedEvidence{}, err
	}
	return v2QualifiedEvidence{IndexBytes: indexBytes, IndexSHA256: SHA256Hex(indexBytes)}, nil
}

// publishV2Evidence renames a qualified staging directory into its
// deterministic final location.
//
// Durability scope (Closure Protocol v2): the rename is atomic for
// same-directory moves on POSIX/Linux/macOS filesystems, which is the
// documented supported platform set. The orchestrator reads the published
// directory and re-verifies its index hash before any ref publication, so
// process-crash recovery after the rename still yields a qualified snapshot
// or none at all. We do NOT claim full power-loss atomicity: cross-platform
// guarantees are explicitly out of scope, and callers MUST NOT treat the
// publish step as durable until fsync of the evidence directory and parent
// directory is added (future work). For Unix filesystems with same-directory
// rename semantics, the current implementation is sufficient for the
// verifier-and-recovery model.
func publishV2Evidence(stagingPath, finalPath string, evidence v2QualifiedEvidence) (v2QualifiedEvidence, error) {
	if err := validateEvidencePathRelationship(stagingPath, finalPath); err != nil {
		return v2QualifiedEvidence{}, err
	}
	if evidence.IndexSHA256 == "" || SHA256Hex(evidence.IndexBytes) != evidence.IndexSHA256 {
		return v2QualifiedEvidence{}, fmt.Errorf("qualified evidence snapshot hash mismatch")
	}
	stagingInfo, err := os.Lstat(stagingPath)
	if err != nil {
		return v2QualifiedEvidence{}, fmt.Errorf("inspect qualified staging directory: %w", err)
	}
	if !stagingInfo.IsDir() {
		return v2QualifiedEvidence{}, fmt.Errorf("qualified staging path is not a directory")
	}
	if _, err := os.Lstat(finalPath); err == nil {
		return v2QualifiedEvidence{}, fmt.Errorf("final evidence directory already exists")
	} else if !os.IsNotExist(err) {
		return v2QualifiedEvidence{}, fmt.Errorf("inspect final evidence directory: %w", err)
	}
	if err := os.Rename(stagingPath, finalPath); err != nil {
		return v2QualifiedEvidence{}, fmt.Errorf("publish evidence directory: %w", err)
	}
	finalInfo, err := os.Lstat(finalPath)
	if err != nil || !os.SameFile(stagingInfo, finalInfo) {
		return v2QualifiedEvidence{}, fmt.Errorf("final evidence directory is not the qualified staging directory")
	}
	evidence.FinalPath = finalPath
	return evidence, nil
}

func validateEvidencePathRelationship(stagingPath, finalPath string) error {
	stagingAbs, err := filepath.Abs(stagingPath)
	if err != nil {
		return err
	}
	finalAbs, err := filepath.Abs(finalPath)
	if err != nil {
		return err
	}
	if filepath.Dir(stagingAbs) != filepath.Dir(finalAbs) ||
		!strings.HasPrefix(filepath.Base(stagingAbs), ".staging-") {
		return fmt.Errorf("evidence staging must be a .staging-* sibling of final path")
	}
	return nil
}

func scanV2Evidence(root string) ([]v2EvidenceIndexEntry, error) {
	entries := make([]v2EvidenceIndexEntry, 0)
	seen := make(map[string]struct{})
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == v2EvidenceIndexName {
			return nil
		}
		if err := validateEvidenceRelativePath(relative); err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("evidence path %q is not a regular file", relative)
		}
		if _, exists := seen[relative]; exists {
			return fmt.Errorf("duplicate normalized evidence path %q", relative)
		}
		seen[relative] = struct{}{}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries = append(entries, v2EvidenceIndexEntry{
			RelativePath: relative, MediaType: evidenceMediaType(relative),
			ByteSize: int64(len(data)), SHA256: SHA256Hex(data),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan evidence: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].RelativePath < entries[j].RelativePath })
	return entries, nil
}

func verifyV2EvidenceIndex(root string, indexBytes []byte) error {
	var index v2EvidenceIndex
	decoder := json.NewDecoder(strings.NewReader(string(indexBytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&index); err != nil {
		return fmt.Errorf("decode evidence index: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("evidence index has trailing data")
	}
	if index.ContractVersion != v2EvidenceIndexV1 {
		return fmt.Errorf("unsupported evidence index contract_version %d", index.ContractVersion)
	}
	actual, err := scanV2Evidence(root)
	if err != nil {
		return err
	}
	if len(actual) != len(index.Entries) {
		return fmt.Errorf("evidence index entry count mismatch")
	}
	for i := range actual {
		if actual[i] != index.Entries[i] {
			return fmt.Errorf("evidence index mismatch at entry %d", i)
		}
	}
	return nil
}

func validateEvidenceRelativePath(path string) error {
	if path == "" || strings.HasPrefix(path, "/") || strings.ContainsRune(path, 0) {
		return fmt.Errorf("invalid evidence relative path %q", path)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean != path || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("invalid evidence relative path %q", path)
	}
	return nil
}

func evidenceMediaType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "application/json"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".txt", ".stdout", ".stderr", ".log":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
