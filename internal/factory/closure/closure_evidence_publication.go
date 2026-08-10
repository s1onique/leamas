// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication.go implements the durable B3
// publication authority for the canonical B2 closure evidence.
//
// The publisher accepts ONLY the typed `evidence.PublicationCandidate`
// produced by `evidence.PrepareClosureEvidenceForPublication`. It
// refuses any object that did not originate from the B2 barrier so
// the publication surface can never be tricked into persisting
// arbitrary bytes by a caller who re-implements the schema.
//
// Publication invariants:
//   - Destination parent is opened once via `os.OpenRoot`; all
//     staging and final publication use only `*os.Root` operations
//     so a symlink / rename-swap of the lexical path cannot
//     redirect the write.
//   - Temp basenames are 16 random bytes (hex) opened through the
//     rooted descriptor with O_CREATE|O_EXCL.
//   - Final publication is no-replace: `*os.Root.Link` delegates
//     to `link(2)` which refuses to overwrite an existing
//     non-directory entry, so the kernel enforces the no-replace
//     contract.
//   - JSON is published first; sidecar second. The state
//     machine reports `json_visible` if the JSON was made
//     visible but the sidecar was not, and `pair_visible` once
//     both files are visible. The state NEVER collapses back to
//     `not_published` after visibility.
//   - The parent directory is fsynced after publication to
//     upgrade to `pair_durable`; a failed fsync yields
//     `pair_visible_durability_unconfirmed`.
//
// State / result types live in `closure_evidence_publication_types.go`.
// The abstract I/O surface and the production implementation live
// in `closure_evidence_publication_io.go`. The end-to-end runner
// wiring lives in `closure_evidence_publication_orchestrator.go`.
package closure

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// EvidencePublication is the prepared rooted authority.
type EvidencePublication struct {
	root        *os.Root
	jsonName    string
	sidecarName string
	parent      string
	canonical   string
	closed      bool
	pfs         evidencePublicationFilesystem
}

// PrepareEvidencePublication validates the destination, opens
// the parent directory once, and returns a prepared authority.
// The destination MUST be outside every supplied worktree, MUST
// not yet exist, and MUST be a path that resolves inside an
// existing parent directory that can be opened for fsync.
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
	sidecar := canonical + ".sha256"
	if _, err := os.Lstat(canonical); err == nil {
		return nil, fmt.Errorf("evidence JSON already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
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
	return &EvidencePublication{
		root:        root,
		jsonName:    filepath.Base(canonical),
		sidecarName: filepath.Base(sidecar),
		parent:      parent,
		canonical:   canonical,
		pfs:         defaultEvidencePublicationFilesystem{random: rand.Reader},
	}, nil
}

// Close releases the parent descriptor.
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

// CanonicalJSON returns the canonical absolute JSON path.
func (p *EvidencePublication) CanonicalJSON() string {
	if p == nil {
		return ""
	}
	return p.canonical
}

// CanonicalSidecar returns the canonical absolute sidecar path.
func (p *EvidencePublication) CanonicalSidecar() string {
	if p == nil {
		return ""
	}
	return p.canonical + ".sha256"
}

// setPublicationFilesystemForTest replaces the abstract I/O
// surface. Tests only; production never calls this.
func (p *EvidencePublication) setPublicationFilesystemForTest(fs evidencePublicationFilesystem) {
	if p == nil || p.closed {
		return
	}
	p.pfs = fs
}

// Publish stages both files through the prepared root, syncs
// each, publishes the JSON via `link` (no-replace at the kernel
// boundary), then publishes the sidecar, and finally fsyncs
// the parent. The state machine is monotonic: the state never
// returns to not_published after a successful link.
func (p *EvidencePublication) Publish(candidate evidence.PublicationCandidate) EvidencePublicationResult {
	if p == nil || p.closed || p.root == nil {
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("evidence publication authority is closed")}
	}
	if candidate.Bytes == nil {
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("candidate bytes are nil")}
	}
	wantSum := sha256.Sum256(candidate.Bytes)
	if candidate.SHA256 != hex.EncodeToString(wantSum[:]) {
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("candidate SHA-256 does not match candidate bytes")}
	}
	if _, err := p.pfs.readFile(p.root, p.jsonName); err == nil {
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("evidence JSON already exists")}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: err}
	}
	if _, err := p.pfs.readFile(p.root, p.sidecarName); err == nil {
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("evidence sidecar already exists")}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: err}
	}

	type stagedFile struct{ tmp string }
	staged := []stagedFile{}
	cleanup := func() {
		for _, s := range staged {
			_ = p.pfs.unlink(p.root, s.tmp)
		}
	}
	stage := func(mode fs.FileMode, data []byte) (string, error) {
		name, file, err := p.pfs.createTemp(p.root, mode)
		if err != nil {
			return "", err
		}
		staged = append(staged, stagedFile{tmp: name})
		if err := p.pfs.writeAll(file, data); err != nil {
			_ = p.pfs.closeFile(file)
			return "", err
		}
		if err := p.pfs.syncFile(file); err != nil {
			_ = p.pfs.closeFile(file)
			return "", err
		}
		if err := p.pfs.closeFile(file); err != nil {
			return "", err
		}
		return name, nil
	}

	jsonTmp, err := stage(0o600, candidate.Bytes)
	if err != nil {
		cleanup()
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("stage JSON: %w", err)}
	}
	sidecarBytes := []byte(candidate.SHA256 + "\n")
	sidecarTmp, err := stage(0o600, sidecarBytes)
	if err != nil {
		cleanup()
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("stage sidecar: %w", err)}
	}

	if err := p.pfs.link(p.root, jsonTmp, p.jsonName); err != nil {
		cleanup()
		return EvidencePublicationResult{State: EvidencePublicationNotPublished, Err: fmt.Errorf("publish JSON (link): %w", err)}
	}
	if err := p.pfs.unlink(p.root, jsonTmp); err != nil {
		// The temp lingers; not fatal.
	}
	if err := p.pfs.link(p.root, sidecarTmp, p.sidecarName); err != nil {
		return EvidencePublicationResult{
			State:         EvidencePublicationJSONVisible,
			CanonicalJSON: p.canonical,
			Err:           fmt.Errorf("publish sidecar (link): %w", err),
		}
	}
	if err := p.pfs.unlink(p.root, sidecarTmp); err != nil {
		// not fatal
	}
	gotJSON, err := p.pfs.readFile(p.root, p.jsonName)
	if err != nil || string(gotJSON) != string(candidate.Bytes) || sha256.Sum256(gotJSON) != wantSum {
		return EvidencePublicationResult{
			State:         EvidencePublicationPairVisible,
			CanonicalJSON: p.canonical,
			CanonicalSide: p.canonical + ".sha256",
			Err:           fmt.Errorf("post-publish JSON observation mismatch: %v", err),
		}
	}
	gotSide, err := p.pfs.readFile(p.root, p.sidecarName)
	if err != nil || string(gotSide) != string(sidecarBytes) {
		return EvidencePublicationResult{
			State:         EvidencePublicationPairVisible,
			CanonicalJSON: p.canonical,
			CanonicalSide: p.canonical + ".sha256",
			Err:           fmt.Errorf("post-publish sidecar observation mismatch: %v", err),
		}
	}
	dir, err := p.pfs.openDir(p.root)
	if err != nil {
		return EvidencePublicationResult{
			State:         EvidencePublicationPairVisibleDurabilityUnconfirmed,
			CanonicalJSON: p.canonical,
			CanonicalSide: p.canonical + ".sha256",
			Err:           fmt.Errorf("open parent for fsync: %w", err),
		}
	}
	syncErr := p.pfs.syncDir(dir)
	closeErr := p.pfs.closeDir(dir)
	if syncErr != nil {
		return EvidencePublicationResult{
			State:         EvidencePublicationPairVisibleDurabilityUnconfirmed,
			CanonicalJSON: p.canonical,
			CanonicalSide: p.canonical + ".sha256",
			Err:           fmt.Errorf("directory fsync: %w", syncErr),
		}
	}
	if closeErr != nil {
		return EvidencePublicationResult{
			State:         EvidencePublicationPairVisibleDurabilityUnconfirmed,
			CanonicalJSON: p.canonical,
			CanonicalSide: p.canonical + ".sha256",
			Err:           fmt.Errorf("directory close: %w", closeErr),
		}
	}
	return EvidencePublicationResult{State: EvidencePublicationPairDurable, CanonicalJSON: p.canonical, CanonicalSide: p.canonical + ".sha256"}
}
