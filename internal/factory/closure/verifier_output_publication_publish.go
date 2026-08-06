// SPDX-License-Identifier: Apache-2.0

package closure

// verifier_output_publication_publish.go isolates the Publish
// lifecycle from the public PrepareVerifierOutput surface in
// verifier_output_publication.go. Splitting along the I/O
// boundary keeps each file under the LLM-friendliness 400-line
// threshold while preserving the race-resistant publication
// invariants from CORRECTION02B.

import (
	"fmt"
)

// Publish writes data via temp + rename within the
// previously opened parent directory. See the file comment
// for the precise ordering. The function mutates State to
// reflect the outcome:
//
//	not_published                       on any pre-rename failure
//	published                           on rename + parent fsync
//	published_but_directory_sync_failed on rename without parent fsync
//
// On every pre-rename failure the temp file is removed (best
// effort) and the destination's previous contents are
// guaranteed untouched. The function never partially renames a
// destination; either the rename succeeds in full or the
// destination keeps its prior state.
// Publish writes data via temp + rename within the
// previously opened parent directory. See the file comment
// for the precise ordering. The function mutates State to
// reflect the outcome:
//
//	not_published                       on any pre-rename failure
//	published                           on rename + parent fsync
//	published_but_directory_sync_failed on rename without parent fsync
//
// On every pre-rename failure the temp file is removed (best
// effort) and the destination's previous contents are
// guaranteed untouched. The function never partially renames a
// destination; either the rename succeeds in full or the
// destination keeps its prior state.
func (a *VerifierOutputAuthority) Publish(data []byte) PublicationResult {
	if a == nil {
		return PublicationResult{
			State: PublicationNotPublished,
			Err:   fmt.Errorf("VerifierOutputAuthority: nil authority"),
		}
	}
	if a.closed || a.root == nil {
		return PublicationResult{
			State: PublicationNotPublished,
			Err:   fmt.Errorf("VerifierOutputAuthority: parent descriptor already released"),
		}
	}
	if a.state != PublicationNotPublished {
		return PublicationResult{
			State:         a.state,
			CanonicalPath: a.canonical,
			Err:           fmt.Errorf("VerifierOutputAuthority: authority already in state %s", a.state),
		}
	}
	if a.publicationFS == nil {
		a.publicationFS = defaultPublicationFilesystem{}
	}

	tempRel, tmp, err := a.publicationFS.createTemp(a.root, a.perm)
	if err != nil {
		return a.failNotPublished(fmt.Errorf("VerifierOutputAuthority: create temp: %w", err), "")
	}
	if err := a.publicationFS.writeAll(tmp, data); err != nil {
		_ = a.publicationFS.closeFile(tmp)
		_ = a.publicationFS.remove(a.root, tempRel)
		return a.failNotPublished(fmt.Errorf("VerifierOutputAuthority: write bytes: %w", err), tempRel)
	}
	if err := a.publicationFS.syncFile(tmp); err != nil {
		_ = a.publicationFS.closeFile(tmp)
		_ = a.publicationFS.remove(a.root, tempRel)
		return a.failNotPublished(fmt.Errorf("VerifierOutputAuthority: fsync temp: %w", err), tempRel)
	}
	if err := a.publicationFS.closeFile(tmp); err != nil {
		_ = a.publicationFS.remove(a.root, tempRel)
		return a.failNotPublished(fmt.Errorf("VerifierOutputAuthority: close temp: %w", err), tempRel)
	}
	if err := a.publicationFS.rename(a.root, tempRel, a.relName); err != nil {
		_ = a.publicationFS.remove(a.root, tempRel)
		return a.failNotPublished(fmt.Errorf("VerifierOutputAuthority: rename temp to %q: %w", a.relName, err), tempRel)
	}
	a.state = PublicationPublished
	if !a.supportsFs {
		return PublicationResult{
			State:         PublicationPublished,
			CanonicalPath: a.canonical,
		}
	}
	dir, derr := a.publicationFS.openParent(a.root)
	if derr != nil {
		a.state = PublicationPublishedButDirectorySyncFailed
		return PublicationResult{
			State:         PublicationPublishedButDirectorySyncFailed,
			CanonicalPath: a.canonical,
			Err:           fmt.Errorf("VerifierOutputAuthority: open parent for fsync: %w", derr),
		}
	}
	syncErr, closeErr := a.publicationFS.syncDirFile(dir)
	if syncErr != nil {
		a.state = PublicationPublishedButDirectorySyncFailed
		return PublicationResult{
			State:         PublicationPublishedButDirectorySyncFailed,
			CanonicalPath: a.canonical,
			Err:           fmt.Errorf("VerifierOutputAuthority: dirfsync: %w", syncErr),
		}
	}
	if closeErr != nil {
		// The directory Sync completed successfully, so the
		// file-system state is durable; the close-on-file
		// handle failed, so we keep the state as PublicationPublished
		// (the bytes are durable on disk) but surface a typed
		// post-publish observation error. The state name does
		// not mislabel the file-system as unsafe.
		a.state = PublicationPublished
		return PublicationResult{
			State:         PublicationPublished,
			CanonicalPath: a.canonical,
			Err:           fmt.Errorf("VerifierOutputAuthority: post-publish observation failed: close parent: %w", closeErr),
		}
	}
	return PublicationResult{
		State:         PublicationPublished,
		CanonicalPath: a.canonical,
	}
}
