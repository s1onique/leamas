// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence.go implements Phase 7 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// ClosureEvidence is the single typed document the closure
// command publishes. The publication step is atomic
// (temp + fsync + rename), sidecar-bearing, and never embeds
// its own hash.

package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ClosureEvidenceSchemaVersion is the schema identifier. New
// versions MUST be appended; existing versions are immutable.
const ClosureEvidenceSchemaVersion = 1

// EvidenceCompleteness is the derived validity of a closure
// evidence document. INCOMPLETE means the document lacks one or
// more authoritative observations (check results, caller state,
// binary authority, gate classification). COMPLETE means every
// authoritative observation is present and consistent.
type EvidenceCompleteness string

const (
	EvidenceIncomplete EvidenceCompleteness = "INCOMPLETE"
	EvidenceComplete   EvidenceCompleteness = "COMPLETE"
)

// CheckEvidence records the typed result of one runtime check.
// The schema mirrors the small matrix the closure command
// publishes alongside the aggregate gate capture.
type CheckEvidence struct {
	CheckID     string `json:"check_id"`
	SubjectTree string `json:"subject_tree"`
	Status      string `json:"status"`
	ExitCode    *int   `json:"exit_code,omitempty"`
	Message     string `json:"message,omitempty"`
}

// CallerState records the observable caller worktree state at
// the moment a publication step completes.
type CallerState struct {
	WorktreeClean bool   `json:"worktree_clean"`
	HeadCommit    string `json:"head_commit,omitempty"`
	HeadTree      string `json:"head_tree,omitempty"`
}

// ClosureEvidence is the canonical publication document.
type ClosureEvidence struct {
	SchemaVersion int `json:"schema_version"`

	Runtime RuntimeContextSubset `json:"runtime"`
	Gate    GateCapture          `json:"gate"`
	Binary  BuiltBinaryEvidence  `json:"binary"`

	Checks []CheckEvidence `json:"checks"`

	CallerStateBefore CallerState `json:"caller_state_before"`
	CallerStateAfter  CallerState `json:"caller_state_after"`

	Completeness EvidenceCompleteness `json:"completeness"`
}

// RuntimeContextSubset is the projected runtime context the
// closure document carries. It mirrors closure.RuntimeContext
// without the repository-internal fields.
type RuntimeContextSubset struct {
	ACTID             string `json:"act_id"`
	RepositoryRoot    string `json:"repository_root"`
	RunID             string `json:"run_id"`
	FreezeCommit      string `json:"freeze_commit"`
	FreezeTree        string `json:"freeze_tree"`
	SubjectCommit     string `json:"subject_commit"`
	SubjectTree       string `json:"subject_tree"`
	PlanPath          string `json:"plan_path"`
	PlanBlob          string `json:"plan_blob"`
	PlanSHA256        string `json:"plan_sha256"`
	EvidenceDirectory string `json:"evidence_directory"`
	StartedAt         string `json:"started_at"`
}

// PublicationRequest parameterises PublishClosureEvidence.
type PublicationRequest struct {
	OutputPath string
	Evidence   ClosureEvidence
	Now        func() time.Time
	// SidecarName is the basename (no extension) of the
	// detached SHA-256 sidecar. Empty defaults to
	// "<basename>.sha256".
	SidecarName string
}

// PublicationResult describes the outcome of PublishClosureEvidence.
type PublicationResult struct {
	DocumentPath  string `json:"document_path"`
	SidecarPath   string `json:"sidecar_path"`
	DocumentBytes []byte `json:"-"`
	DocumentSHA   string `json:"document_sha256"`
	SidecarSHA    string `json:"sidecar_sha256"`
	DirectorySync bool   `json:"directory_sync_ok"`
}

// PublishClosureEvidence atomically writes the supplied evidence
// document. The function:
//
//  1. validates the typed structure;
//  2. marshals the document once;
//  3. writes it via temp + fsync + rename;
//  4. fsyncs the parent directory where supported;
//  5. writes a detached SHA-256 sidecar.
//
// The function never embeds the document's own hash inside the
// JSON.
func PublishClosureEvidence(req PublicationRequest) (PublicationResult, error) {
	if strings.TrimSpace(req.OutputPath) == "" {
		return PublicationResult{}, errors.New("evidence: output path is required")
	}
	if req.Now == nil {
		req.Now = time.Now
	}
	if req.Evidence.Completeness != EvidenceComplete {
		return PublicationResult{}, errors.New("evidence: cannot publish invalid closure evidence")
	}
	if req.Evidence.SchemaVersion != ClosureEvidenceSchemaVersion {
		return PublicationResult{}, fmt.Errorf("evidence: unsupported schema version %d", req.Evidence.SchemaVersion)
	}
	document, err := json.MarshalIndent(req.Evidence, "", "  ")
	if err != nil {
		return PublicationResult{}, fmt.Errorf("evidence: marshal: %w", err)
	}
	documentSHA := SHA256HexBytes(document)
	dir := filepath.Dir(req.OutputPath)
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return PublicationResult{}, fmt.Errorf("evidence: mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".closure-evidence-*.json")
	if err != nil {
		return PublicationResult{}, fmt.Errorf("evidence: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(document); err != nil {
		_ = tmp.Close()
		cleanup()
		return PublicationResult{}, fmt.Errorf("evidence: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return PublicationResult{}, fmt.Errorf("evidence: fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return PublicationResult{}, fmt.Errorf("evidence: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, req.OutputPath); err != nil {
		cleanup()
		return PublicationResult{}, fmt.Errorf("evidence: rename temp: %w", err)
	}
	dirSync := syncDirBestEffort(dir)
	sidecarName := req.SidecarName
	if sidecarName == "" {
		sidecarName = strings.TrimSuffix(filepath.Base(req.OutputPath), filepath.Ext(req.OutputPath)) + ".sha256"
	}
	sidecarPath := filepath.Join(dir, sidecarName)
	sidecarBytes := []byte(documentSHA + "  " + filepath.Base(req.OutputPath) + "\n")
	if err := os.WriteFile(sidecarPath, sidecarBytes, 0o600); err != nil {
		return PublicationResult{}, fmt.Errorf("evidence: write sidecar: %w", err)
	}
	sidecarSHA := SHA256HexBytes(sidecarBytes)
	return PublicationResult{
		DocumentPath:  req.OutputPath,
		SidecarPath:   sidecarPath,
		DocumentBytes: document,
		DocumentSHA:   documentSHA,
		SidecarSHA:    sidecarSHA,
		DirectorySync: dirSync == nil,
	}, nil
}

// ValidateClosureEvidence asserts that every required field is
// present and well-formed. It does NOT perform any IO.
func ValidateClosureEvidence(evidence ClosureEvidence) error {
	if evidence.SchemaVersion != ClosureEvidenceSchemaVersion {
		return fmt.Errorf("evidence: schema_version %d is not supported", evidence.SchemaVersion)
	}
	if evidence.Completeness != EvidenceComplete {
		return errors.New("evidence: completeness is not COMPLETE")
	}
	if evidence.Runtime.ACTID == "" {
		return errors.New("evidence: runtime.act_id is empty")
	}
	if !isValidOID(evidence.Runtime.FreezeCommit) {
		return errors.New("evidence: runtime.freeze_commit is not a 40-char hex OID")
	}
	if !isValidOID(evidence.Runtime.SubjectCommit) {
		return errors.New("evidence: runtime.subject_commit is not a 40-char hex OID")
	}
	if !isValidOID(evidence.Runtime.PlanBlob) {
		return errors.New("evidence: runtime.plan_blob is not a 40-char hex OID")
	}
	if !isHexSHA256(evidence.Runtime.PlanSHA256) {
		return errors.New("evidence: runtime.plan_sha256 is not a 64-char hex digest")
	}
	if evidence.Binary.BinaryPath == "" {
		return errors.New("evidence: binary.binary_path is empty")
	}
	if !isHexSHA256(evidence.Binary.BinarySHA256) {
		return errors.New("evidence: binary.binary_sha256 is not a 64-char hex digest")
	}
	if evidence.Gate.RawOutputPath == "" {
		return errors.New("evidence: gate.raw_output_path is empty")
	}
	return nil
}

// syncDirBestEffort is a small wrapper around the closure
// package's atomic writer; the indirection keeps the evidence
// package self-contained.
func syncDirBestEffort(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// isValidOID is a tiny regex-free OID validator.
func isValidOID(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

// isHexSHA256 reports whether value is a 64-character lowercase
// hex digest.
func isHexSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, ch := range value {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			return false
		}
	}
	return true
}

// DocumentJSON is the small helper that marshals the supplied
// evidence into the canonical byte form. It exists so tests can
// assert the publication output without re-implementing the
// marshalling rules.
func DocumentJSON(evidence ClosureEvidence) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(evidence); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
