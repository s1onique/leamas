// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// v2EvidenceSnapshot is the one qualified evidence view captured by a v2
// invocation. Classification and verification consume these bytes and must not
// reopen evidence independently.
type v2EvidenceSnapshot struct {
	Present    bool
	Runtime    v2RuntimeEvidence
	IndexBytes []byte
	IndexHash  string
}

func v2EvidencePresent(evidenceDir string) (bool, error) {
	_, err := os.Lstat(evidenceDir)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("inspect evidence directory: %w", err)
	}
}

func readQualifiedV2Evidence(evidenceDir, actID, subject string) (v2EvidenceSnapshot, error) {
	var snapshot v2EvidenceSnapshot
	indexBytes, err := os.ReadFile(filepath.Join(evidenceDir, v2EvidenceIndexName))
	if err != nil {
		return snapshot, fmt.Errorf("read evidence index: %w", err)
	}
	if err := verifyV2EvidenceIndex(evidenceDir, indexBytes); err != nil {
		return snapshot, err
	}
	runtimeBytes, err := os.ReadFile(filepath.Join(evidenceDir, "runtime.json"))
	if err != nil {
		return snapshot, fmt.Errorf("read runtime evidence: %w", err)
	}
	entries, err := decodeV2EvidenceEntries(indexBytes)
	if err != nil {
		return snapshot, err
	}
	runtimeEntry, ok := entries["runtime.json"]
	if !ok || runtimeEntry.MediaType != "application/json" ||
		runtimeEntry.ByteSize != int64(len(runtimeBytes)) || runtimeEntry.SHA256 != SHA256Hex(runtimeBytes) {
		return snapshot, fmt.Errorf("runtime evidence bytes do not match captured index")
	}
	decoder := json.NewDecoder(strings.NewReader(string(runtimeBytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot.Runtime); err != nil {
		return snapshot, fmt.Errorf("decode runtime evidence: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return snapshot, fmt.Errorf("runtime evidence has trailing data")
	}
	if snapshot.Runtime.ContractVersion != 1 || snapshot.Runtime.ActID != actID {
		return snapshot, fmt.Errorf("runtime evidence identity mismatch")
	}
	if snapshot.Runtime.Runner.VCSRevision != subject || snapshot.Runtime.Runner.VCSModified {
		return snapshot, fmt.Errorf("runtime runner subject authority is invalid")
	}
	if err := validateSHA256("runtime runner binary SHA-256", snapshot.Runtime.Runner.BinarySHA256); err != nil {
		return snapshot, err
	}
	snapshot.Present = true
	snapshot.IndexBytes = append([]byte(nil), indexBytes...)
	snapshot.IndexHash = SHA256Hex(indexBytes)
	return snapshot, nil
}

func ObjectFormatFromOID(oid string) ObjectFormat {
	if len(oid) == 64 {
		return ObjectFormatSHA256
	}
	return ObjectFormatSHA1
}

func readBlobAtCommit(ctx context.Context, git gitClient, repoRoot, commit, path string) ([]byte, error) {
	result := git.Run(ctx, repoRoot, "show", commit+":"+path)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("read %s from closure: %s", path, gitFailureDetail(result))
	}
	return result.Stdout, nil
}
