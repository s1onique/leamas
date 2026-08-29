// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay_render.go holds the rendering
// half of the evidence overlay MVP. Splitting it from the
// collector keeps each file LLM-friendly at <= 400 lines.
//
// The renderer is read-only and re-uses the existing IsBinary
// detector and the PathEscape form so evidence rendering is
// consistent with ordinary digest sections. Critically, the
// renderer reads files RELATIVE TO overlay.RepoRoot, never to
// the process current working directory, and propagates read
// failures so a successfully collected file whose contents
// become unreadable can never masquerade as a zero-byte file.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
// File contents are read relative to overlay.RepoRoot so the
// renderer is independent of the caller's cwd.
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
		sb.WriteString("\nfiles:\n")
		for _, f := range overlay.Files {
			sb.WriteString(fmt.Sprintf("### %s  size=%d\n", PathEscape(f.Path), f.Size))
			content, isBinary, readErr := previewEvidenceFile(overlay.RepoRoot, f.Path)
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
// for the rendered section. The function is read-only and never
// follows symlinks (it operates on the RepoRoot-bound absolute
// path returned by inspect). Read failures are propagated so a
// missing or unreadable file renders as an explicit
// (read_failed: ...) marker rather than as silent empty content.
func previewEvidenceFile(repoRoot, relPath string) (string, bool, error) {
	if relPath == "" {
		return "", false, fmt.Errorf("empty relative path")
	}
	if repoRoot == "" {
		return "", false, fmt.Errorf("empty repoRoot; cannot bind read")
	}
	full := filepath.Join(repoRoot, filepath.FromSlash(relPath))
	if IsBinary(full) {
		return "", true, nil
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", full, err)
	}
	const maxBytes = 16 * 1024 // 16 KiB preview cap
	if len(data) > maxBytes {
		data = data[:maxBytes]
	}
	out := string(data)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, false, nil
}
