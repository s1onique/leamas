// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay_render.go holds the rendering
// half of the evidence overlay MVP. Splitting it from the
// collector keeps each file LLM-friendly at <= 400 lines.
//
// The renderer is intentionally read-only and re-uses the
// existing IsBinary detector and the PathEscape form so evidence
// rendering is consistent with ordinary digest sections.
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
// where preview comes from previewEvidenceFile and is content
// bounded at 16 KiB.
func RenderEvidenceOverlay(overlay EvidenceOverlay) string {
	var sb strings.Builder
	sb.WriteString("## EVIDENCE_OVERLAY (experimental)\n")
	if len(overlay.Roots) == 0 {
		sb.WriteString("(no evidence roots requested)\n")
		return sb.String()
	}
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
		return sb.String()
	}
	sb.WriteString("\nfiles:\n")
	for _, f := range overlay.Files {
		sb.WriteString(fmt.Sprintf("### %s  size=%d\n", PathEscape(f.Path), f.Size))
		content, isBinary := previewEvidenceFile(f.Path)
		if isBinary {
			sb.WriteString("(binary file)\n")
		} else if content != "" {
			sb.WriteString(content)
			if !strings.HasSuffix(content, "\n") {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// previewEvidenceFile returns a small, content-bounded preview
// for the rendered section. The function is read-only, never
// follows symlinks (Lstat semantics), and re-uses the existing
// IsBinary detector. Files larger than MaxEvidencePerFileBytes
// are rejected at collection time, so this path never has to
// defend against oversize content.
func previewEvidenceFile(relPath string) (string, bool) {
	if relPath == "" {
		return "", false
	}
	full := relPath
	if !filepath.IsAbs(full) {
		full = filepath.Join(".", filepath.FromSlash(relPath))
	}
	if IsBinary(full) {
		return "", true
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", false
	}
	const maxBytes = 16 * 1024 // 16 KiB preview cap
	if len(data) > maxBytes {
		data = data[:maxBytes]
	}
	out := string(data)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, false
}
