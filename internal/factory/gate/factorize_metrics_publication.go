// Package gate provides metrics publication with unique temp file handling.
package gate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PublishMetricsV3 writes the metrics document using a unique sibling temp file.
// Uses os.CreateTemp to ensure concurrent calls never select the same file.
func PublishMetricsV3(path string, doc *FactorizeMetricsV3) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	// Create a unique sibling temp file
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(base)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	// Rename to final path (atomic on POSIX)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename to %s: %w", path, err)
	}

	return nil
}
