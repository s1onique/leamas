// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// SidecarRecord is one observation recorded in a detached sidecar file.
type SidecarRecord struct {
	LogicalName string `json:"logical_name"`
	MediaType   string `json:"media_type"`
	SHA256      string `json:"sha256"`
	ByteCount   int64  `json:"byte_count"`
	Truncated   bool   `json:"truncated,omitempty"`
	Incomplete  bool   `json:"incomplete,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

// SidecarFile is the canonical schema for a detached evidence file.
type SidecarFile struct {
	SchemaVersion int             `json:"schema_version"`
	ActID         string          `json:"act_id"`
	Kind          string          `json:"kind"`
	SubjectTree   string          `json:"subject_tree_oid"`
	Records       []SidecarRecord `json:"records"`
}

// SidecarKind enumerates the supported sidecar kinds.
type SidecarKind string

const (
	SidecarKindChecks      SidecarKind = "checks"
	SidecarKindArtifacts   SidecarKind = "artifacts"
	SidecarKindDiagnostics SidecarKind = "diagnostics"
)

// SidecarSummary is a deterministic identity for a sidecar file.
type SidecarSummary struct {
	Path          string      `json:"path"`
	SchemaVersion int         `json:"schema_version"`
	MediaType     string      `json:"media_type"`
	SHA256        string      `json:"sha256"`
	ByteCount     int64       `json:"byte_count"`
	ItemCount     int         `json:"item_count"`
	Kind          SidecarKind `json:"kind"`
}

// sidecarFileExt is the canonical extension for every detached sidecar.
const sidecarFileExt = ".json"

// sidecarFileName returns the canonical sidecar name for the ACT.
func sidecarFileName(actID string, kind SidecarKind) string {
	if !actIDPattern.MatchString(actID) {
		panic("sidecarFileName: invalid act id")
	}
	return fmt.Sprintf("%s.%s.sidecar%s", actID, kind, sidecarFileExt)
}

// BuildSidecarFile constructs a deterministic, bound-checked sidecar.
func BuildSidecarFile(actID, subjectTree string, kind SidecarKind, records []SidecarRecord) (SidecarFile, error) {
	if !actIDPattern.MatchString(actID) {
		return SidecarFile{}, fmt.Errorf("sidecar act id %q is invalid", actID)
	}
	if strings.TrimSpace(subjectTree) == "" {
		return SidecarFile{}, errors.New("sidecar subject_tree_oid is required")
	}
	if len(records) > SidecarMaxRecordCount {
		return SidecarFile{}, fmt.Errorf("sidecar %s exceeds %d record cap", kind, SidecarMaxRecordCount)
	}
	cloned := append([]SidecarRecord(nil), records...)
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].LogicalName < cloned[j].LogicalName })
	for _, record := range cloned {
		if strings.TrimSpace(record.LogicalName) == "" {
			return SidecarFile{}, errors.New("sidecar record logical_name is required")
		}
		if err := validateSHA256Hex(record.SHA256); err != nil {
			return SidecarFile{}, fmt.Errorf("sidecar %s: %w", record.LogicalName, err)
		}
		if record.ByteCount < 0 {
			return SidecarFile{}, fmt.Errorf("sidecar %s: byte_count must be non-negative", record.LogicalName)
		}
		if len(record.Metadata) > SidecarMaxStdoutMetadataBytes {
			return SidecarFile{}, fmt.Errorf("sidecar %s: metadata exceeds %d bytes", record.LogicalName, SidecarMaxStdoutMetadataBytes)
		}
		if len(record.LogicalName) > SidecarMaxStringLength {
			return SidecarFile{}, fmt.Errorf("sidecar %s: logical_name exceeds %d bytes", record.LogicalName, SidecarMaxStringLength)
		}
		if record.ByteCount > SidecarPerFileMaxBytes {
			return SidecarFile{}, fmt.Errorf("sidecar %s: byte_count %d exceeds per-file bound", record.LogicalName, record.ByteCount)
		}
	}
	return SidecarFile{SchemaVersion: 1, ActID: actID, Kind: string(kind), SubjectTree: subjectTree, Records: cloned}, nil
}

// EncodeSidecarFile returns the deterministic bytes, content hash, and
// bound summary for a sidecar. The encoding is sorted and indented so
// the SHA-256 reflects the canonical shape, not a transient map order.
func EncodeSidecarFile(file SidecarFile) ([]byte, string, SidecarSummary, error) {
	if file.SchemaVersion != 1 {
		return nil, "", SidecarSummary{}, fmt.Errorf("sidecar schema version %d is not 1", file.SchemaVersion)
	}
	encoded, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, "", SidecarSummary{}, fmt.Errorf("marshal sidecar: %w", err)
	}
	encoded = append(encoded, '\n')
	sum := sha256.Sum256(encoded)
	hash := hex.EncodeToString(sum[:])
	return encoded, hash, SidecarSummary{
		SchemaVersion: file.SchemaVersion,
		MediaType:     "application/json",
		SHA256:        hash,
		ByteCount:     int64(len(encoded)),
		ItemCount:     len(file.Records),
		Kind:          SidecarKind(file.Kind),
	}, nil
}

// SidecarPath joins an evidence directory with the canonical sidecar name.
func SidecarPath(evidenceDirectory, actID string, kind SidecarKind) string {
	return filepath.Join(evidenceDirectory, sidecarFileName(actID, kind))
}

// WriteSidecarFile atomically writes a sidecar to evidenceDirectory using
// the canonical name. The function refuses to overwrite an existing file
// and returns the bound summary on success.
func WriteSidecarFile(evidenceDirectory, actID, subjectTree string, kind SidecarKind, records []SidecarRecord) (SidecarSummary, error) {
	file, err := BuildSidecarFile(actID, subjectTree, kind, records)
	if err != nil {
		return SidecarSummary{}, err
	}
	encoded, _, summary, err := EncodeSidecarFile(file)
	if err != nil {
		return SidecarSummary{}, err
	}
	summary.Path = SidecarPath(evidenceDirectory, actID, kind)
	if err := validateSidecarSummary(summary, file); err != nil {
		return SidecarSummary{}, err
	}
	if _, err := writeDetachedBytes(evidenceDirectory, sidecarFileName(actID, kind), "application/json", encoded); err != nil {
		return SidecarSummary{}, err
	}
	return summary, nil
}

// verifySidecarFile rebinds the summary against the bytes on disk and
// validates every sidecar limit.
func verifySidecarFile(evidenceDirectory, actID string, summary SidecarSummary, file SidecarFile) error {
	expected := SidecarPath(evidenceDirectory, actID, SidecarKind(file.Kind))
	if summary.Path != "" && summary.Path != expected {
		return fmt.Errorf("sidecar path mismatch: got %q want %q", summary.Path, expected)
	}
	if !path.IsAbs(expected) {
		return fmt.Errorf("sidecar path %q must be absolute", expected)
	}
	if summary.ItemCount != len(file.Records) {
		return fmt.Errorf("sidecar %s item count %d != records %d", file.Kind, summary.ItemCount, len(file.Records))
	}
	if summary.SHA256 == "" || len(summary.SHA256) != 64 {
		return fmt.Errorf("sidecar %s has invalid sha256", file.Kind)
	}
	if summary.ByteCount <= 0 || summary.ByteCount > SidecarPerFileMaxBytes {
		return fmt.Errorf("sidecar %s byte count %d exceeds bound", file.Kind, summary.ByteCount)
	}
	if summary.ItemCount > SidecarMaxRecordCount {
		return fmt.Errorf("sidecar %s record count %d exceeds bound", file.Kind, summary.ItemCount)
	}
	if summary.MediaType != "application/json" {
		return fmt.Errorf("sidecar %s media type %q not application/json", file.Kind, summary.MediaType)
	}
	if summary.SchemaVersion != 1 {
		return fmt.Errorf("sidecar %s schema version %d not 1", file.Kind, summary.SchemaVersion)
	}
	return nil
}

func validateSidecarSummary(summary SidecarSummary, file SidecarFile) error {
	if summary.ItemCount != len(file.Records) {
		return fmt.Errorf("sidecar %s summary record count mismatch", file.Kind)
	}
	if summary.ByteCount > SidecarPerFileMaxBytes {
		return fmt.Errorf("sidecar %s exceeds %d byte bound", file.Kind, SidecarPerFileMaxBytes)
	}
	return nil
}

func validateSHA256Hex(value string) error {
	if len(value) != 64 {
		return fmt.Errorf("sha256 must be 64 hex chars, got %d", len(value))
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return fmt.Errorf("sha256 contains non-hex character %q", r)
		}
	}
	return nil
}
