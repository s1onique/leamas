// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay_inspect.go isolates the
// per-path filesystem-classification primitive so the
// collector file stays LLM-friendly.
//
// The classifier refuses anything other than a non-symlink
// regular file. The error strings are intentionally simple
// ("symlink: ...", "non-regular: ...") so the collector can
// classify them into stable EvidenceOmission reason codes.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
)

// inspectEvidenceFile classifies a single relative path under
// canonicalRoot. Only non-symlink, regular files survive;
// symlinks, directories, FIFOs, sockets, device nodes, and
// missing entries return an error whose message lets the
// caller pick a stable omission reason.
func inspectEvidenceFile(canonicalRoot, relPath string) (evidenceEntry, error) {
	full := filepath.Join(canonicalRoot, filepath.FromSlash(relPath))
	lst, lerr := os.Lstat(full)
	if lerr != nil {
		return evidenceEntry{}, lerr
	}
	mode := lst.Mode()
	if mode&os.ModeSymlink != 0 {
		return evidenceEntry{}, fmt.Errorf("symlink: %s", relPath)
	}
	if !mode.IsRegular() {
		return evidenceEntry{}, fmt.Errorf("non-regular: %s", relPath)
	}
	return evidenceEntry{Size: lst.Size()}, nil
}
