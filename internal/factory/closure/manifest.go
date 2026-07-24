package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// SHA256Hex computes the SHA-256 hex digest of data.
func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// LoadManifest loads and validates a manifest from a file with strict bounds.
func LoadManifest(path string) (Manifest, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("read manifest: %w", err)
	}
	if len(data) > MaxManifestBytes {
		return Manifest{}, data, fmt.Errorf("manifest exceeds %d byte limit", MaxManifestBytes)
	}
	m, err := UnmarshalManifest(data)
	if err != nil {
		return Manifest{}, data, err
	}
	return m, data, nil
}

// UnmarshalManifest decodes with bounds, strict field checking, and EOF requirement.
func UnmarshalManifest(data []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	// Check for trailing JSON by trying to decode more
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Manifest{}, fmt.Errorf("trailing JSON after manifest")
		}
		return Manifest{}, fmt.Errorf("decode error: %w", err)
	}
	if err := ValidateManifestStrict(m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// MarshalManifest marshals a manifest with plan to JSON.
func MarshalManifest(m Manifest, plan Plan) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// WriteManifest writes manifest bytes to a file.
func WriteManifest(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// DecodeManifest decodes manifest JSON bytes with strict field checking.
func DecodeManifest(data []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	return m, nil
}

// ValidateManifestStrict validates manifest structure and required fields.
func ValidateManifestStrict(m Manifest) error {
	// Contract version must be 1
	if m.ContractVersion != ContractVersionV1 {
		return fmt.Errorf("unsupported contract_version %d (expected %d)", m.ContractVersion, ContractVersionV1)
	}

	// ACT ID must be non-empty
	if m.ActID == "" {
		return fmt.Errorf("act_id is required")
	}

	// Plan path must be non-empty
	if m.Plan.Path == "" {
		return fmt.Errorf("plan.path is required")
	}

	// Plan SHA-256 must be valid hex (64 chars)
	if m.Plan.SHA256 == "" {
		return fmt.Errorf("plan.sha256 is required")
	}
	if len(m.Plan.SHA256) != 64 {
		return fmt.Errorf("plan.sha256 must be 64 hex characters, got %d", len(m.Plan.SHA256))
	}

	// Freeze commit must be valid OID
	if err := ValidateOID("plan_freeze.freeze_commit", m.PlanFreeze.FreezeCommit); err != nil {
		return err
	}

	// Subject commit and tree must be valid OIDs
	if err := ValidateOID("subject.commit_oid", m.Subject.CommitOID); err != nil {
		return err
	}
	if err := ValidateOID("subject.tree_oid", m.Subject.TreeOID); err != nil {
		return err
	}

	// Verdict must be valid
	if m.Verdict != VerdictPass && m.Verdict != VerdictFail {
		return fmt.Errorf("invalid verdict %q (expected pass or fail)", m.Verdict)
	}

	return nil
}

// VerifyManifestFile verifies a manifest file and returns the manifest.
func VerifyManifestFile(repositoryRoot, manifestPath string) (Manifest, []byte, error) {
	manifestPath = joinRepositoryPath(repositoryRoot, manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("read manifest: %w", err)
	}
	m, err := UnmarshalManifest(data)
	if err != nil {
		return Manifest{}, data, err
	}
	// Reject machine-local absolute paths
	if strings.HasPrefix(m.Plan.Path, "/") {
		return Manifest{}, data, fmt.Errorf("plan.path must not be absolute: %s", m.Plan.Path)
	}
	return m, data, nil
}
