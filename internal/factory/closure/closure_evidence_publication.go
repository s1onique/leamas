// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// PublicationCandidate is the complete B2 result supplied to the B3
// publication barrier. Bytes are written verbatim; the JSON has no digest.
type PublicationCandidate struct {
	Evidence any
	Bytes    []byte
	SHA256   [32]byte
}

// EvidencePublication is the durable JSON/sidecar pair authority.
type EvidencePublication struct {
	root        *os.Root
	jsonName    string
	sidecarName string
	parent      string
	closed      bool
}

// PrepareEvidencePublication validates and opens the destination parent once.
func PrepareEvidencePublication(repositoryRoot, destination string, worktrees []CanonicalWorktree) (*EvidencePublication, error) {
	roots := worktreeRootsForCanonical(worktrees)
	inventory, err := newRepositoryWorktreeInventoryFromCanonical(roots)
	if err != nil {
		return nil, err
	}
	canonical, err := resolveDetachedDestination(repositoryRoot, destination, inventory)
	if err != nil {
		return nil, err
	}
	if filepath.Ext(canonical) != ".json" {
		return nil, fmt.Errorf("evidence destination must end in .json")
	}
	if _, err := os.Lstat(canonical); err == nil {
		return nil, fmt.Errorf("evidence JSON already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	sidecar := canonical + ".sha256"
	if _, err := os.Lstat(sidecar); err == nil {
		return nil, fmt.Errorf("evidence sidecar already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	parent := filepath.Dir(canonical)
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, fmt.Errorf("open evidence parent: %w", err)
	}
	return &EvidencePublication{root: root, jsonName: filepath.Base(canonical), sidecarName: filepath.Base(sidecar), parent: parent}, nil
}

func (p *EvidencePublication) Close() error {
	if p == nil || p.closed {
		return nil
	}
	p.closed = true
	if p.root == nil {
		return nil
	}
	err := p.root.Close()
	p.root = nil
	return err
}

// Publish stages both files, syncs each, then renames both without overwrite.
func (p *EvidencePublication) Publish(candidate PublicationCandidate) error {
	if p == nil || p.closed || p.root == nil {
		return fmt.Errorf("publication authority is closed")
	}
	if len(candidate.Bytes) == 0 {
		return fmt.Errorf("publication candidate bytes are empty")
	}
	sum := sha256.Sum256(candidate.Bytes)
	if candidate.SHA256 != sum {
		return fmt.Errorf("publication candidate digest mismatch")
	}
	for _, name := range []string{p.jsonName, p.sidecarName} {
		if _, err := p.root.Stat(name); err == nil {
			return fmt.Errorf("publication artifact %q already exists", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	type staged struct{ tmp string }
	stagedFiles := make([]staged, 0, 2)
	cleanup := func() {
		for _, f := range stagedFiles {
			_ = p.root.Remove(f.tmp)
		}
	}
	stage := func(_ string, data []byte) error {
		f, err := os.CreateTemp(p.root.Name(), ".tmp-closure-")
		if err != nil {
			return err
		}
		tmp := filepath.Base(f.Name())
		stagedFiles = append(stagedFiles, staged{tmp: tmp})
		if err = f.Chmod(0600); err == nil {
			_, err = f.Write(data)
		}
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		return err
	}
	if err := stage(p.jsonName, candidate.Bytes); err != nil {
		cleanup()
		return fmt.Errorf("stage JSON: %w", err)
	}
	sidecar := []byte(hex.EncodeToString(candidate.SHA256[:]) + "\n")
	if err := stage(p.sidecarName, sidecar); err != nil {
		cleanup()
		return fmt.Errorf("stage sidecar: %w", err)
	}
	if _, err := p.root.Stat(p.jsonName); err == nil {
		cleanup()
		return fmt.Errorf("JSON appeared before publication")
	} else if !errors.Is(err, os.ErrNotExist) {
		cleanup()
		return err
	}
	if _, err := p.root.Stat(p.sidecarName); err == nil {
		cleanup()
		return fmt.Errorf("sidecar appeared before publication")
	} else if !errors.Is(err, os.ErrNotExist) {
		cleanup()
		return err
	}
	if err := p.root.Rename(stagedFiles[0].tmp, p.jsonName); err != nil {
		cleanup()
		return fmt.Errorf("publish JSON: %w", err)
	}
	if err := p.root.Rename(stagedFiles[1].tmp, p.sidecarName); err != nil {
		return fmt.Errorf("publish sidecar after JSON visible: %w", err)
	}
	dir, err := p.root.Open(".")
	if err != nil {
		return fmt.Errorf("open parent after publication: %w", err)
	}
	syncErr := dir.Sync()
	closeErr := dir.Close()
	if syncErr != nil {
		return fmt.Errorf("directory durability unconfirmed after publication: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("directory close after publication: %w", closeErr)
	}
	jsonBytes, err := p.root.ReadFile(p.jsonName)
	if err != nil {
		return fmt.Errorf("observe published JSON: %w", err)
	}
	if string(jsonBytes) != string(candidate.Bytes) || sha256.Sum256(jsonBytes) != candidate.SHA256 {
		return fmt.Errorf("published JSON observation mismatch")
	}
	gotSidecar, err := p.root.ReadFile(p.sidecarName)
	if err != nil || string(gotSidecar) != string(sidecar) {
		return fmt.Errorf("published sidecar observation mismatch")
	}
	return nil
}
