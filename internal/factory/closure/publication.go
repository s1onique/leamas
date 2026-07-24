// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// This implementation deliberately scopes transactional publication to Linux
// and to one destination filesystem. Unsupported platforms fail closed rather
// than inheriting an unproved cross-platform atomicity claim.
const (
	publicationMarkerName = ".leamas-close-publication.json"
	publicationMaxBytes   = 8 << 20
)

var ErrUnsupportedPublicationPlatform = errors.New("closure publication requires Linux same-filesystem transaction support")

type PublicationFailurePoint string

const (
	PublicationFailureStagingDirectory       PublicationFailurePoint = "staging_directory_creation"
	PublicationFailureManifestStagedWrite    PublicationFailurePoint = "manifest_staged_write"
	PublicationFailureReportStagedWrite      PublicationFailurePoint = "report_staged_write"
	PublicationFailureErratumStagedWrite     PublicationFailurePoint = "erratum_staged_write"
	PublicationFailureSchemaValidation       PublicationFailurePoint = "schema_validation"
	PublicationFailureBoundValidation        PublicationFailurePoint = "bound_validation"
	PublicationFailureHashValidation         PublicationFailurePoint = "hash_validation"
	PublicationFailureFirstPublication       PublicationFailurePoint = "first_publication"
	PublicationFailureLaterPublication       PublicationFailurePoint = "later_publication"
	PublicationFailureInterruptedPublication PublicationFailurePoint = "interrupted_publication"
	PublicationFailureCleanup                PublicationFailurePoint = "cleanup"
)

type PublicationOptions struct {
	Destination  string
	Files        map[string][]byte
	FailurePoint PublicationFailurePoint
}

type publicationEntry struct {
	Path      string `json:"path"`
	StagePath string `json:"stage_path"`
	Published bool   `json:"published"`
}

type publicationMarker struct {
	Version     int                `json:"version"`
	Destination string             `json:"destination"`
	StageDir    string             `json:"stage_dir"`
	State       string             `json:"state"`
	Entries     []publicationEntry `json:"entries"`
}

// PublishArtifactSet stages, validates, and publishes a complete set of
// relative files. A pre-existing complete byte-identical set is idempotent;
// a partial or conflicting set fails closed.
func PublishArtifactSet(options PublicationOptions) error {
	if runtime.GOOS != "linux" {
		return ErrUnsupportedPublicationPlatform
	}
	if options.Destination == "" || !filepath.IsAbs(options.Destination) {
		return fmt.Errorf("publication destination must be an absolute directory")
	}
	if len(options.Files) == 0 {
		return fmt.Errorf("publication set is empty")
	}
	if err := os.MkdirAll(options.Destination, 0o700); err != nil {
		return fmt.Errorf("create publication destination: %w", err)
	}
	if err := recoverInterruptedPublication(options.Destination); err != nil {
		return err
	}

	paths := make([]string, 0, len(options.Files))
	for path := range options.Files {
		if err := validatePublicationPath(path); err != nil {
			return err
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if state, err := inspectExistingPublication(options.Destination, paths, options.Files); err != nil {
		return err
	} else if state == publicationComplete {
		return nil
	}

	if options.FailurePoint == PublicationFailureStagingDirectory {
		return fmt.Errorf("injected publication failure: %s", options.FailurePoint)
	}
	stageDir, err := os.MkdirTemp(options.Destination, ".leamas-close-stage-*")
	if err != nil {
		return fmt.Errorf("create publication staging directory: %w", err)
	}
	keepStage := false
	defer func() {
		if !keepStage {
			_ = os.RemoveAll(stageDir)
		}
	}()

	entries := make([]publicationEntry, 0, len(paths))
	for index, path := range paths {
		failure := stagedWriteFailure(path)
		if failure != "" && options.FailurePoint == failure {
			return fmt.Errorf("injected publication failure: %s", options.FailurePoint)
		}
		stagePath := filepath.Join(stageDir, fmt.Sprintf("entry-%04d", index))
		if err := writePublicationFile(stagePath, options.Files[path]); err != nil {
			return err
		}
		entries = append(entries, publicationEntry{Path: path, StagePath: stagePath})
	}
	if options.FailurePoint == PublicationFailureSchemaValidation {
		return fmt.Errorf("injected publication failure: %s", options.FailurePoint)
	}
	if err := validateStagedPublication(entries, options.Files); err != nil {
		return err
	}
	if options.FailurePoint == PublicationFailureBoundValidation {
		return fmt.Errorf("injected publication failure: %s", options.FailurePoint)
	}
	if err := validatePublicationBounds(options.Files); err != nil {
		return err
	}
	if options.FailurePoint == PublicationFailureHashValidation {
		return fmt.Errorf("injected publication failure: %s", options.FailurePoint)
	}
	if err := validatePublicationHashes(entries, options.Files); err != nil {
		return err
	}

	marker := publicationMarker{Version: 1, Destination: options.Destination, StageDir: stageDir, State: "publishing", Entries: entries}
	if err := writePublicationMarker(options.Destination, marker); err != nil {
		return err
	}
	published := 0
	for index := range entries {
		if index == 0 && options.FailurePoint == PublicationFailureFirstPublication {
			return rollbackPublication(options.Destination, marker, fmt.Errorf("injected publication failure: %s", options.FailurePoint))
		}
		if index > 0 && options.FailurePoint == PublicationFailureLaterPublication {
			return rollbackPublication(options.Destination, marker, fmt.Errorf("injected publication failure: %s", options.FailurePoint))
		}
		entry := &entries[index]
		target := filepath.Join(options.Destination, filepath.FromSlash(entry.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return rollbackPublication(options.Destination, marker, fmt.Errorf("create publication parent: %w", err))
		}
		if err := os.Rename(entry.StagePath, target); err != nil {
			return rollbackPublication(options.Destination, marker, fmt.Errorf("publish %s: %w", entry.Path, err))
		}
		entry.Published = true
		published++
		marker.Entries = entries
		if err := writePublicationMarker(options.Destination, marker); err != nil {
			return rollbackPublication(options.Destination, marker, err)
		}
		if published == 1 && options.FailurePoint == PublicationFailureInterruptedPublication {
			keepStage = true
			return fmt.Errorf("interrupted publication left recovery marker")
		}
	}
	marker.State = "complete"
	marker.Entries = entries
	if err := writePublicationMarker(options.Destination, marker); err != nil {
		return rollbackPublication(options.Destination, marker, err)
	}
	if options.FailurePoint == PublicationFailureCleanup {
		keepStage = true
		return fmt.Errorf("injected publication cleanup failure; published set is valid")
	}
	if err := os.Remove(filepath.Join(options.Destination, publicationMarkerName)); err != nil {
		return fmt.Errorf("cleanup publication marker: %w", err)
	}
	if err := os.RemoveAll(stageDir); err != nil {
		return fmt.Errorf("cleanup publication staging directory: %w", err)
	}
	keepStage = true
	return nil
}

func validatePublicationPath(value string) error {
	if value == "" || filepath.IsAbs(value) || strings.ContainsRune(value, 0) {
		return fmt.Errorf("publication path %q is not relative", value)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("publication path %q escapes destination or is not clean", value)
	}
	return nil
}

func stagedWriteFailure(path string) PublicationFailurePoint {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "closure-manifests/") || strings.HasSuffix(lower, "manifest.json") || strings.HasSuffix(lower, ".attestation.json"):
		return PublicationFailureManifestStagedWrite
	case strings.Contains(lower, "close-reports/") || strings.HasSuffix(lower, ".report.md"):
		return PublicationFailureReportStagedWrite
	case strings.Contains(lower, "lifecycle-errata/") || strings.Contains(lower, "erratum"):
		return PublicationFailureErratumStagedWrite
	default:
		return ""
	}
}

func writePublicationFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create staged publication file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write staged publication file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync staged publication file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close staged publication file: %w", err)
	}
	return nil
}

func validateStagedPublication(entries []publicationEntry, files map[string][]byte) error {
	for _, entry := range entries {
		data, err := os.ReadFile(entry.StagePath)
		if err != nil {
			return fmt.Errorf("read staged %s: %w", entry.Path, err)
		}
		if strings.HasSuffix(strings.ToLower(entry.Path), ".json") && !json.Valid(data) {
			return fmt.Errorf("staged JSON %s is invalid", entry.Path)
		}
		if strings.HasSuffix(strings.ToLower(entry.Path), ".md") && len(data) > MaxReportBytes {
			return fmt.Errorf("staged report %s exceeds bound", entry.Path)
		}
		if string(data) != string(files[entry.Path]) {
			return fmt.Errorf("staged %s differs from requested bytes", entry.Path)
		}
	}
	return nil
}

func validatePublicationBounds(files map[string][]byte) error {
	for path, data := range files {
		if len(data) > publicationMaxBytes {
			return fmt.Errorf("publication %s exceeds %d-byte bound", path, publicationMaxBytes)
		}
	}
	return nil
}

func validatePublicationHashes(entries []publicationEntry, files map[string][]byte) error {
	for _, entry := range entries {
		data, err := os.ReadFile(entry.StagePath)
		if err != nil {
			return err
		}
		a := sha256.Sum256(data)
		b := sha256.Sum256(files[entry.Path])
		if hex.EncodeToString(a[:]) != hex.EncodeToString(b[:]) {
			return fmt.Errorf("staged hash mismatch for %s", entry.Path)
		}
	}
	return nil
}

func writePublicationMarker(destination string, marker publicationMarker) error {
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := filepath.Join(destination, publicationMarkerName)
	return os.WriteFile(path, data, 0o600)
}

var _ = io.EOF
