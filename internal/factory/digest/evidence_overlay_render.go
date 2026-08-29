// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay_render.go holds the rendering
// half of the evidence overlay MVP. Splitting it from the
// collector keeps each file LLM-friendly at <= 400 lines.
//
// The renderer is read-only and re-uses the existing PathEscape
// form so evidence rendering is consistent with ordinary digest
// sections. Critically, every preview read is opened through
// os.OpenRoot (when the Go runtime supports it) so a symlink
// or ancestor swap between collection and render cannot
// redirect the read outside the authorized repo root. The
// classifier and renderer share the same open file descriptor
// so the bytes classified as binary/text are exactly the
// bytes that are then emitted to the digest (C5/C6 invariant:
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
				content, isBinary, readErr := previewEvidenceFile(root, f.Path)
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
//     concurrent ancestor or leaf symlink swap;
//   - the bytes classified as binary/text are exactly the
//     bytes that are emitted (CLASSIFIED_BYTES == RENDERED_BYTES).
//
// Without a traversal-resistant root (root == nil) the
// function returns an explicit failure so callers can never
// silently fall back to pathname I/O.
func previewEvidenceFile(root *os.Root, relPath string) (string, bool, error) {
	if relPath == "" {
		return "", false, fmt.Errorf("empty relative path")
	}
	if root == nil {
		return "", false, errRootOpenUnavailable
	}
	rel := filepath.FromSlash(relPath)
	file, err := root.Open(rel)
	if err != nil {
		return "", false, fmt.Errorf("open %s via root: %w", rel, err)
	}
	defer file.Close()

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
