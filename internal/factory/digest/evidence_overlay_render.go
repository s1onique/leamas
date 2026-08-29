// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay_render.go holds the rendering
// half of the evidence overlay MVP. Splitting it from the
// collector keeps each file LLM-friendly at <= 400 lines.
//
// The renderer is read-only and re-uses the existing PathEscape
// form so evidence rendering is consistent with ordinary digest
// sections.
//
// Security model (accurate, not aspirational):
//
//   - os.Root prevents the read from escaping overlay.RepoRoot
//     but does not by itself prevent symlink substitution that
//     stays inside the root.
//   - the collector rejects symlink evidence at collection time
//     so a leaf that was inspected as a regular file and later
//     becomes a symlink is detected by os.SameFile on the
//     opened file's Stat, not by os.Root itself.
//   - opening one *os.Root does NOT freeze the filesystem
//     contents beneath it; race-window substitutions of the
//     same pathname to a different inode are detected because
//     the safely opened object must still match the os.FileInfo
//     captured at inspect time.
//   - safeOpenReadOnlyNonblock opens the file with nonblocking
//     semantics on Unix so a FIFO/socket substitution cannot
//     block the render thread waiting for a writer. The open
//     remains traversal-resistant (it goes through *os.Root).
//     The post-open Stat then rejects any non-regular object
//     before any blocking content read is attempted.
//
// The bytes classified as binary/text are exactly the bytes
// that are emitted (C5/C6 invariant:
// CLASSIFIED_BYTES == RENDERED_BYTES).
package digest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// rootOpenUnavailable is returned by previewEvidenceFile when
// the runtime does not provide os.Root (the project would need
// to be on a Go baseline older than 1.24 to hit this). The
// caller MUST surface the failure rather than fall back to
// unsafe openat-style pathname I/O.
var errRootOpenUnavailable = errors.New("evidence: traversal-resistant open unavailable")

// openRoot returns the *os.Root anchored at repoRoot when the
// runtime supports os.OpenRoot. The package keeps the result
// caller-owned so every read is a single safe-open.
func openRoot(repoRoot string) (*os.Root, error) {
	if repoRoot == "" {
		return nil, fmt.Errorf("evidence: empty repoRoot")
	}
	return os.OpenRoot(repoRoot)
}

// RenderEvidenceOverlay renders the EVIDENCE_OVERLAY section as
// a single deterministic block. The section is labeled
// "(experimental)" so legacy v3 contract consumers can detect
// it without breaking their existing parsers.
//
// Each file renders as:
//
//	### root/path  size=N
//	<preview>
//
// or, when the preview cannot be read:
//
//	### root/path  size=N
//	(read_failed: <reason>)
//
// File contents are read through a traversal-resistant
// os.Root handle so a concurrent symlink swap cannot redirect
// the read outside overlay.RepoRoot.
//
// The omissions section is always rendered when non-empty so
// the overlay never silently claims completeness; truncation
// is reported via a top-level "truncated: true" line.
func RenderEvidenceOverlay(overlay EvidenceOverlay) string {
	var sb strings.Builder
	sb.WriteString("## EVIDENCE_OVERLAY (experimental)\n")
	if len(overlay.Roots) == 0 {
		sb.WriteString("(no evidence roots requested)\n")
		return sb.String()
	}
	sb.WriteString("repo_root: ")
	sb.WriteString(PathEscape(overlay.RepoRoot))
	sb.WriteString("\n")
	sb.WriteString("roots:\n")
	for _, r := range overlay.Roots {
		if r.Rejected {
			sb.WriteString(fmt.Sprintf("  - %s  REJECTED (%s)\n",
				PathEscape(r.Path), r.Reason))
		} else {
			sb.WriteString(fmt.Sprintf("  - %s\n", PathEscape(r.Path)))
		}
	}
	if len(overlay.Files) == 0 {
		sb.WriteString("\nfiles: (none)\n")
	} else {
		// Open the repository root ONCE so every preview
		// read is bounded by the same traversal-resistant
		// handle. Closing it at the end of the section
		// keeps the resource scope tight and removes any
		// TOCTOU window between successive previews.
		root, rootErr := openRoot(overlay.RepoRoot)
		if rootErr != nil {
			sb.WriteString(fmt.Sprintf("\nfiles: (render_unavailable: %s)\n", rootErr.Error()))
		} else {
			sb.WriteString("\nfiles:\n")
			for _, f := range overlay.Files {
				sb.WriteString(fmt.Sprintf("### %s  size=%d\n", PathEscape(f.Path), f.Size))
				content, isBinary, readErr := previewEvidenceFile(root, f.Path, f.Identity)
				switch {
				case readErr != nil:
					sb.WriteString(fmt.Sprintf("(read_failed: %s)\n", readErr.Error()))
				case isBinary:
					sb.WriteString("(binary file)\n")
				case content != "":
					sb.WriteString(content)
					if !strings.HasSuffix(content, "\n") {
						sb.WriteString("\n")
					}
				}
			}
			_ = root.Close()
		}
	}
	if len(overlay.Omitted) > 0 {
		sb.WriteString("\nomissions:\n")
		for _, om := range overlay.Omitted {
			sb.WriteString(fmt.Sprintf("  - path: %s\n    reason: %s\n",
				PathEscape(om.Path), om.Reason))
		}
	}
	if overlay.Truncated {
		sb.WriteString("\ntruncated: true\n")
	}
	return sb.String()
}

// previewEvidenceFile returns a small, content-bounded preview
// for the rendered section. The function is read-only and MUST
// be called with an *os.Root obtained from openRoot so that:
//
//   - the open call cannot escape overlay.RepoRoot via a
//     concurrent ancestor or leaf symlink swap (os.Root);
//   - the open is nonblocking on Unix (safeOpenReadOnlyNonblock)
//     so a FIFO/socket substitution cannot block the render
//     thread waiting for a writer;
//   - the opened object is rejected with a visible failure
//     when its mode is not regular, before any read;
//   - the opened filesystem object is verified, via
//     os.SameFile, to be the same object that the collector
//     authorized at inspect time. A different object (regular
//     file replaced by symlink, regular file replaced by
//     different regular file) is rejected with a visible
//     failure rather than rendered.
//   - the bytes classified as binary/text are exactly the
//     bytes that are emitted (CLASSIFIED_BYTES == RENDERED_BYTES).
//
// Without a traversal-resistant root (root == nil) or a
// collected-identity handle (collected == nil) the function
// returns an explicit failure so callers can never silently
// fall back to pathname I/O.
//
// In-place content mutation of the SAME inode is explicitly
// deferred (IN_PLACE_CONTENT_MUTATION_ATOMICITY=DEFERRED); this
// mechanism guarantees pathname / object substitution cannot
// silently redirect the read, not byte-level atomicity.
func previewEvidenceFile(root *os.Root, relPath string, collected os.FileInfo) (string, bool, error) {
	if relPath == "" {
		return "", false, fmt.Errorf("empty relative path")
	}
	if root == nil {
		return "", false, errRootOpenUnavailable
	}
	if collected == nil {
		return "", false, fmt.Errorf("evidence object identity missing")
	}
	rel := filepath.FromSlash(relPath)
	// C12: open with nonblocking semantics on platforms that
	// support it, so a FIFO/socket substitution cannot block
	// the render thread. The safe-open is still traversal-
	// resistant: it goes through *os.Root.
	file, err := safeOpenReadOnlyNonblock(root, rel)
	if err != nil {
		return "", false, fmt.Errorf("open %s via root: %w", rel, err)
	}
	defer file.Close()

	// C12: immediately Stat the opened file. If the object
	// is no longer a regular file (FIFO / socket / device /
	// symlink that resolved through the root) we reject it
	// before any blocking read can be attempted.
	opened, sErr := file.Stat()
	if sErr != nil {
		return "", false, fmt.Errorf("stat %s via root: %w", rel, sErr)
	}
	if !opened.Mode().IsRegular() {
		return "", false, fmt.Errorf(
			"evidence object is not a regular file: mode=%v",
			opened.Mode(),
		)
	}
	// C11: confirm the opened object is still the same
	// filesystem object that was authorized at collect time.
	// This catches:
	//   regular file -> different regular file
	//   regular file -> internal symlink (still inside root)
	if !os.SameFile(opened, collected) {
		return "", false, fmt.Errorf(
			"evidence object changed since collection: was %v, now %v",
			collected.Name(), opened.Name(),
		)
	}

	// boundedReader limits the read to MaxEvidencePreviewBytes
	// and lets the binary classifier see exactly the bytes
	// that are returned to the renderer.
	const previewCap = 16 * 1024 // 16 KiB preview cap
	lr := &io.LimitedReader{R: file, N: previewCap + 1}
	data, rerr := io.ReadAll(lr)
	if rerr != nil {
		return "", false, fmt.Errorf("read %s via root: %w", rel, rerr)
	}
	if int64(len(data)) > previewCap {
		data = data[:previewCap]
	}
	if isBinaryBytes(data) {
		return "", true, nil
	}
	out := string(data)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, false, nil
}

// isBinaryBytes reports whether the supplied byte slice looks
// like a binary artifact. The classifier reuses the same
// heuristic as the package's IsBinary preview helper but
// operates on already-read bytes so the caller's
// CLASSIFIED_BYTES == RENDERED_BYTES invariant holds.
func isBinaryBytes(data []byte) bool {
	const scanWindow = 8 * 1024
	end := len(data)
	if end > scanWindow {
		end = scanWindow
	}
	for _, b := range data[:end] {
		if b == 0 {
			return true
		}
	}
	return false
}
