package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func DecodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := decodeStrictBounded(data, MaxManifestBytes, &manifest); err != nil {
		return Manifest{}, err
	}
	if err := validateManifestStructure(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func LoadManifest(path string) (Manifest, []byte, error) {
	data, err := readBoundedFile(path, MaxManifestBytes)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("read closure manifest: %w", err)
	}
	manifest, err := DecodeManifest(data)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("decode closure manifest: %w", err)
	}
	return manifest, data, nil
}

func MarshalManifest(manifest Manifest, plan Plan) ([]byte, error) {
	if err := VerifyManifestAgainstPlan(manifest, plan); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal closure manifest: %w", err)
	}
	data = append(data, '\n')
	if len(data) > MaxManifestBytes {
		return nil, fmt.Errorf("manifest exceeds %d-byte limit", MaxManifestBytes)
	}
	return data, nil
}

func VerifyManifestFile(repositoryRoot, manifestPath string) (Manifest, []byte, error) {
	manifest, data, err := LoadManifest(manifestPath)
	if err != nil {
		return Manifest{}, nil, err
	}
	planPath := filepath.Join(repositoryRoot, filepath.FromSlash(manifest.Plan.Path))
	plan, planBytes, err := LoadPlan(planPath)
	if err != nil {
		return Manifest{}, nil, err
	}
	if SHA256Hex(planBytes) != manifest.Plan.SHA256 {
		return Manifest{}, nil, fmt.Errorf("plan SHA-256 does not match manifest")
	}
	if err := VerifyManifestAgainstPlan(manifest, plan); err != nil {
		return Manifest{}, nil, err
	}
	return manifest, data, nil
}

func WriteManifest(path string, data []byte) error {
	if len(data) > MaxManifestBytes {
		return fmt.Errorf("manifest exceeds %d-byte limit", MaxManifestBytes)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write closure manifest: %w", err)
	}
	return nil
}

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
