// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	publicationNone publicationState = iota
	publicationPartial
	publicationComplete
)

type publicationState int

// inspectExistingPublication classifies the existing set of canonical
// files under destination. The return is used to fail closed on partial
// or conflicting sets and to short-circuit when the published set is
// already byte-identical to the requested one.
func inspectExistingPublication(destination string, paths []string, files map[string][]byte) (publicationState, error) {
	existing := 0
	identical := 0
	missing := 0
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(path)))
		if errors.Is(err, os.ErrNotExist) {
			missing++
			continue
		}
		if err != nil {
			return publicationNone, fmt.Errorf("inspect existing publication %s: %w", path, err)
		}
		existing++
		if string(data) == string(files[path]) {
			identical++
		}
	}
	if existing == 0 && missing == 0 {
		return publicationNone, nil
	}
	if existing == len(paths) && identical == len(paths) && missing == 0 {
		return publicationComplete, nil
	}
	if existing == 0 {
		return publicationNone, nil
	}
	return publicationNone, fmt.Errorf("existing publication is partial or conflicting; refusing mixed canonical set")
}

// recoverInterruptedPublication restores the pre-publication state
// described by a marker left over by a previous interrupted run.
func recoverInterruptedPublication(destination string) error {
	path := filepath.Join(destination, publicationMarkerName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read publication recovery marker: %w", err)
	}
	var marker publicationMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return fmt.Errorf("decode publication recovery marker: %w", err)
	}
	if marker.State == "complete" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if marker.StageDir != "" {
			_ = os.RemoveAll(marker.StageDir)
		}
		return nil
	}
	for index := len(marker.Entries) - 1; index >= 0; index-- {
		entry := marker.Entries[index]
		if entry.Published {
			_ = os.Remove(filepath.Join(destination, filepath.FromSlash(entry.Path)))
		}
	}
	if marker.StageDir != "" {
		if err := os.RemoveAll(marker.StageDir); err != nil {
			return fmt.Errorf("remove interrupted staging directory: %w", err)
		}
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove publication recovery marker: %w", err)
	}
	return nil
}

// recoverInterruptedMarker walks the entries of an interrupted
// publication marker in reverse order and removes any published files
// before clearing the staging directory and the marker itself.
func recoverInterruptedMarker(destination string, marker publicationMarker, cause error) error {
	for index := len(marker.Entries) - 1; index >= 0; index-- {
		entry := marker.Entries[index]
		if entry.Published {
			if err := os.Remove(filepath.Join(destination, filepath.FromSlash(entry.Path))); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	if marker.StageDir != "" {
		if err := os.RemoveAll(marker.StageDir); err != nil {
			return err
		}
	}
	if err := os.Remove(filepath.Join(destination, publicationMarkerName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// rollbackPublication returns a wrapped rollback error. It is the single
// point where interrupted publication transitions into a failed run.
func rollbackPublication(destination string, marker publicationMarker, cause error) error {
	if err := recoverInterruptedMarker(destination, marker, cause); err != nil {
		return fmt.Errorf("%w; rollback incomplete: %v", cause, err)
	}
	return cause
}
