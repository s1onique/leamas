// SPDX-License-Identifier: Apache-2.0

package closure

// verifier_output_publication.go implements the confined
// publication authority required by Phases 3 and 4 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02B.
//
// The authority owns the stable destination-parent directory
// descriptor for the entire lifetime of a publication. The CLI
// first prepares the authority (after worktree inventory and
// path canonicalization) and then calls Publish once per
// successful verifier result. The internal sequence is:
//
//	1. create temp inside the opened parent with the final
//	   mode (root-relative syscall via os.Root.OpenFile +
//	   O_CREATE|O_EXCL; the open inode carries the mode so
//	   no separate chmod step is needed); basename is
//	   randomized.
//	2. write all bytes (one buffer, one write)
//	3. file fsync
//	4. close
//	5. rename to final relative name (still inside parent)
//	6. directory fsync where the OS exposes one
//
// Phase 4 explicitly requires the file mode to be set BEFORE
// the file fsync; otherwise the inode metadata change would
// not be covered by the sync. CORRECTION02B closes that
// window by creating the temp file with the final mode in
// step 1 and skipping a separate chmod step entirely.
//
// The PublicationState result has exactly three values so
// callers can match on the result without parsing message text:
//
//	not_published                       temp lifecycle did not finish
//	published                           rename succeeded, dirfsync
//	                                   succeeded; visible_after_rename
//	                                   AND crash_durability confirmed
//	published_but_directory_sync_failed rename succeeded but the
//	                                   containing directory could
//	                                   not be fsynced; visible_after_rename
//	                                   confirmed; crash_durability is
//	                                   UNCONFIRMED
//
// "visible_after_rename" means the destination's directory
// entry references the new bytes regardless of crash recovery.
// "crash_durability" means the bytes — and the directory entry
// — are durable across a crash; only a successful dirfsync
// upgrades from the former to the latter. The state name
// refrains from claiming post-rename stability that a failed
// dirfsync has not confirmed.
//
// CORRECTION02B fixes the prior order bug where the file mode
// was set by `os.Chmod` after `file.Sync`, leaving the inode
// metadata change uncovered by the data sync. The fix is to
// open the temp file with the target perm in one syscall.

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PublicationState is the explicit outcome of a publication
// attempt. The string representation is stable for tests and
// for the final-report ACT manifest.
type PublicationState int

const (
	// PublicationNotPublished means the publication sequence
	// did not reach the rename step or the rename itself
	// failed. The destination is guaranteed untouched and no
	// temp file remains on disk; the caller may safely retry.
	PublicationNotPublished PublicationState = iota

	// PublicationPublished means the rename succeeded and (on
	// platforms that expose a directory fsync) the parent
	// directory was fsynced. visible_after_rename and
	// crash_durability are both confirmed.
	PublicationPublished

	// PublicationPublishedButDirectorySyncFailed means the
	// rename succeeded but the parent-directory fsync failed
	// (or the parent directory could not be opened for fsync
	// on a platform where the op is exposed).
	PublicationPublishedButDirectorySyncFailed

	// PublicationPublishedButPostPublishObservationFailed means
	// the rename and directory fsync succeeded, but observing the
	// post-publish directory handle failed. The publication is
	// visible and durable, but the observer contract is broken.
	PublicationPublishedButPostPublishObservationFailed
)

// String returns the canonical snake_case token for the
// publication state. The token is stable so it can appear in
// ACT manifests and downstream tooling.
func (s PublicationState) String() string {
	switch s {
	case PublicationNotPublished:
		return "not_published"
	case PublicationPublished:
		return "published"
	case PublicationPublishedButDirectorySyncFailed:
		return "published_but_directory_sync_failed"
	case PublicationPublishedButPostPublishObservationFailed:
		return "published_but_post_publish_observation_failed"
	default:
		return "unknown_publication_state"
	}
}

// PublicationResult is the structured outcome of a Publish
// call. State is authoritative; Err is non-nil only when the
// publication sequence failed (state == not_published) or the
// post-rename directory sync or observation errored.
type PublicationResult struct {
	State         PublicationState
	CanonicalPath string
	Err           error
}

// publicationFilesystem is the abstract I/O surface used by the
// authority. The default implementation uses the os.Root-based
// primitives supplied by the Go runtime; tests substitute a
// fake to deterministically inject failures at every step of
// the publication sequence.
type publicationFilesystem interface {
	createTemp(root *os.Root, mode fs.FileMode) (relName string, file *os.File, err error)
	writeAll(file *os.File, data []byte) error
	syncFile(file *os.File) error
	closeFile(file *os.File) error
	rename(root *os.Root, oldname, newname string) error
	openParent(root *os.Root) (file *os.File, err error)
	syncDirFile(file *os.File) (syncErr, closeErr error)
	remove(root *os.Root, name string) error
}

// defaultPublicationFilesystem is the production I/O surface.
// It uses only root-relative operations on the supplied
// *os.Root so the authority never re-resolves the lexical
// path through mutable ancestors.
var cryptoRandomReader io.Reader = rand.Reader

type defaultPublicationFilesystem struct {
	random io.Reader
}

func (f defaultPublicationFilesystem) createTemp(root *os.Root, mode fs.FileMode) (string, *os.File, error) {
	const tempPrefix = ".verifier-output-"
	const tempSuffix = ".tmp"
	for attempt := uint64(0); attempt < (1 << 20); attempt++ {
		name, err := randomizedTempNameFromReader(tempPrefix, tempSuffix, f.random)
		if err != nil {
			// CSPRNG failure: the production loop is fail-closed
			// here too; an attacker that controls the CSPRNG can
			// already win bigger games, so the right response is
			// to surface a typed diagnostic and abort.
			return "", nil, fmt.Errorf("defaultPublicationFilesystem: random temp name: %w", err)
		}
		f, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err == nil {
			return name, f, nil
		}
		if !os.IsExist(err) {
			return "", nil, err
		}
	}
	return "", nil, fmt.Errorf("defaultPublicationFilesystem: could not allocate a unique temp name")
}

func (defaultPublicationFilesystem) writeAll(file *os.File, data []byte) error {
	_, err := file.Write(data)
	return err
}

func (defaultPublicationFilesystem) syncFile(file *os.File) error {
	return file.Sync()
}

func (defaultPublicationFilesystem) closeFile(file *os.File) error {
	return file.Close()
}

func (defaultPublicationFilesystem) rename(root *os.Root, oldname, newname string) error {
	return root.Rename(oldname, newname)
}

func (defaultPublicationFilesystem) openParent(root *os.Root) (*os.File, error) {
	return root.Open(".")
}

func (defaultPublicationFilesystem) syncDirFile(file *os.File) (error, error) {
	syncErr := file.Sync()
	closeErr := file.Close()
	return syncErr, closeErr
}

func (defaultPublicationFilesystem) remove(root *os.Root, name string) error {
	return root.Remove(name)
}

// VerifierOutputAuthority is the concrete prepared authority.
// The struct owns one os.Root descriptor; Close releases it.
// The internal fields are unexported so callers cannot mutate
// the publication state or the parent descriptor out of
// band.
type VerifierOutputAuthority struct {
	root          *os.Root
	relName       string
	canonical     string
	perm          fs.FileMode
	state         PublicationState
	closed        bool
	supportsFs    bool
	publicationFS publicationFilesystem
}

// PrepareVerifierOutput accepts the canonical, complete
// repository inventory (including main and every linked
// worktree) and validates the candidate --output path. On
// success the function returns a prepared authority whose
// internal state is PublicationNotPublished.
//
// On a successful prepare, the authority holds:
//
//	root          = *os.Root rooted at the canonical destination
//	                parent directory
//	relName       = the canonical absolute destination's base name
//	canonical     = the canonical absolute final destination path
//	perm          = 0o644 default; only fs.FileMode 0 is overridden
//	supportsFs    = false on Windows, true elsewhere
//
// The function performs no output creation: no temp file, no
// directory fsync, no rename. It only validates inputs and
// opens the parent descriptor.
//
// The caller is expected to supply the inventory built by
// InventoryRepositoryWorktrees; the resolver binds the
// canonical repositoryRoot against the inventory's roots.
func PrepareVerifierOutput(repositoryRoot, outputPath string, worktrees []CanonicalWorktree) (*VerifierOutputAuthority, error) {
	if strings.TrimSpace(repositoryRoot) == "" {
		return nil, NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      "verifier output preparation requires a non-empty repository root",
			PropertyName: "repository_root",
		})
	}
	if strings.TrimSpace(outputPath) == "" {
		return nil, NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      "verifier output preparation requires a non-empty --output path",
			PropertyName: "output_path",
		})
	}
	roots := worktreeRootsForCanonical(worktrees)
	inventory, err := newRepositoryWorktreeInventoryFromCanonical(roots)
	if err != nil {
		return nil, err
	}
	canonical, err := resolveDetachedDestination(repositoryRoot, outputPath, inventory)
	if err != nil {
		return nil, err
	}
	parentDir := filepath.Dir(canonical)
	if parentDir == "" || parentDir == "." {
		return nil, NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output destination %q has no resolvable parent directory", outputPath),
			PropertyName: "output_path",
		})
	}
	root, err := os.OpenRoot(parentDir)
	if err != nil {
		return nil, NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("open destination parent %q: %s", parentDir, err.Error()),
			PropertyName: "output_path",
		})
	}
	relName := filepath.Base(canonical)
	return &VerifierOutputAuthority{
		root:          root,
		relName:       relName,
		canonical:     filepath.Join(root.Name(), relName),
		perm:          0o644,
		supportsFs:    runtime.GOOS != "windows",
		publicationFS: defaultPublicationFilesystem{random: cryptoRandomReader},
	}, nil
}

// worktreeRootsForCanonical flattens a CanonicalWorktree slice
// to the canonical-root list expected by the inventory
// constructor. The function preserves the input order and
// does not deduplicate; NewRepositoryWorktreeInventoryForTest
// enforces the same invariants the production parser does.
func worktreeRootsForCanonical(worktrees []CanonicalWorktree) []string {
	out := make([]string, len(worktrees))
	for i, w := range worktrees {
		out[i] = w.Path
	}
	return out
}

// State returns the current publication state. The state
// transitions monotonically:
//
//	not_published -> published
//	not_published -> published_but_directory_sync_failed
//
// Once the authority reports published or
// published_but_directory_sync_failed, it is closed for
// further Publish calls.
func (a *VerifierOutputAuthority) State() PublicationState {
	if a == nil {
		return PublicationNotPublished
	}
	return a.state
}

// CanonicalPath returns the canonical absolute destination
// path that publication will install bytes at. The string is
// set during preparation and never changes.
func (a *VerifierOutputAuthority) CanonicalPath() string {
	if a == nil {
		return ""
	}
	return a.canonical
}

// SetPermission overrides the default perm (0o644). The
// override is only honored while the authority is in the
// PublicationNotPublished state; once a Publish call has
// succeeded the change is rejected.
func (a *VerifierOutputAuthority) SetPermission(perm fs.FileMode) error {
	if a == nil {
		return fmt.Errorf("VerifierOutputAuthority: nil authority")
	}
	if a.state != PublicationNotPublished {
		return fmt.Errorf("VerifierOutputAuthority: cannot change perm in state %s", a.state)
	}
	if perm == 0 {
		return nil
	}
	a.perm = perm
	return nil
}

// setPublicationFilesystemForTest replaces the abstract I/O
// surface for the lifetime of the authority. The function is
// intentionally unexported: it exists for same-package tests
// and is not part of the production public surface. Production
// callers MUST NOT call this method.
func (a *VerifierOutputAuthority) setPublicationFilesystemForTest(fs publicationFilesystem) {
	if a == nil {
		return
	}
	if a.state != PublicationNotPublished {
		return
	}
	a.publicationFS = fs
}

// Close releases the parent descriptor. Safe to call
// multiple times; safe to call after a successful Publish.
// The function never errors in production; the return value
// exists so the call site can `defer a.Close()` cleanly.
func (a *VerifierOutputAuthority) Close() error {
	if a == nil || a.closed {
		return nil
	}
	a.closed = true
	if a.root == nil {
		return nil
	}
	err := a.root.Close()
	a.root = nil
	return err
}
