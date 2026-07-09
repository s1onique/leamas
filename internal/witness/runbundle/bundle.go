// Package runbundle provides local run bundle creation and validation for
// Leamas verification witness evidence.
package runbundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Standard subdirectory names for run bundles.
var subdirs = []string{
	"claims",
	"evidence",
	"digests",
	"traces",
	"verifier-results",
}

// Default tool name if none provided.
const DefaultToolName = "leamas"

// Errors for bundle operations.
var (
	ErrSchemaVersionMismatch = errors.New("metadata schema version mismatch")
	ErrRunIDMismatch         = errors.New("metadata run ID does not match requested ID")
	ErrMissingMetadata       = errors.New("metadata.json not found")
	ErrMetadataReadError     = errors.New("failed to read metadata.json")
	ErrMetadataDecodeError   = errors.New("failed to decode metadata.json")
)

// Bundle represents an open run bundle.
type Bundle struct {
	Root string
	ID   RunID
	Path string
}

// CreateOptions contains options for creating a new run bundle.
type CreateOptions struct {
	Root     string
	RunID    RunID
	Now      func() time.Time
	ToolName string
	Version  string
}

// Create creates a new run bundle with the given options.
// It creates the bundle directory, all subdirectories, and writes metadata.json.
func Create(opts CreateOptions) (Bundle, error) {
	// Validate root
	if opts.Root == "" {
		return Bundle{}, ErrEmptyRoot
	}

	// Validate run ID
	if err := ValidateRunID(opts.RunID); err != nil {
		return Bundle{}, err
	}

	// Use default time function if not provided
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	// Use default tool name if not provided
	toolName := opts.ToolName
	if toolName == "" {
		toolName = DefaultToolName
	}

	// Compute bundle path
	bundlePath, err := BundlePath(opts.Root, opts.RunID)
	if err != nil {
		return Bundle{}, err
	}

	// Create the bundle directory
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return Bundle{}, fmt.Errorf("failed to create bundle directory: %w", err)
	}

	// Create all subdirectories
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(bundlePath, subdir)
		if err := os.MkdirAll(subdirPath, 0755); err != nil {
			return Bundle{}, fmt.Errorf("failed to create subdirectory %s: %w", subdir, err)
		}
	}

	// Create metadata
	metadata := NewMetadata(opts.RunID, now(), toolName, opts.Version)

	// Write metadata.json
	metadataPath := filepath.Join(bundlePath, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return Bundle{}, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return Bundle{}, fmt.Errorf("failed to write metadata.json: %w", err)
	}

	return Bundle{
		Root: opts.Root,
		ID:   opts.RunID,
		Path: bundlePath,
	}, nil
}

// Open opens an existing run bundle and returns its metadata.
func Open(root string, id RunID) (Bundle, *Metadata, error) {
	// Validate inputs
	if root == "" {
		return Bundle{}, nil, ErrEmptyRoot
	}
	if err := ValidateRunID(id); err != nil {
		return Bundle{}, nil, err
	}

	// Compute bundle path
	bundlePath, err := BundlePath(root, id)
	if err != nil {
		return Bundle{}, nil, err
	}

	// Read metadata.json
	metadataPath := filepath.Join(bundlePath, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Bundle{}, nil, ErrMissingMetadata
		}
		return Bundle{}, nil, fmt.Errorf("%w: %v", ErrMetadataReadError, err)
	}

	// Strict decode metadata
	meta, err := StrictDecode(data)
	if err != nil {
		return Bundle{}, nil, fmt.Errorf("%w: %v", ErrMetadataDecodeError, err)
	}

	// Validate schema version
	if meta.SchemaVersion != SchemaVersion {
		return Bundle{}, nil, fmt.Errorf("%w: got %q, want %q", ErrSchemaVersionMismatch, meta.SchemaVersion, SchemaVersion)
	}

	// Validate run ID matches
	if meta.RunID != id {
		return Bundle{}, nil, fmt.Errorf("%w: got %q, want %q", ErrRunIDMismatch, meta.RunID, id)
	}

	return Bundle{
		Root: root,
		ID:   id,
		Path: bundlePath,
	}, meta, nil
}
