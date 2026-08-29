// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay_inspect.go isolates the
// per-path filesystem-classification primitive so the
// collector file stays LLM-friendly.
//
// The classifier returns typed sentinel errors so callers
// classify outcomes with errors.Is rather than by parsing
// error strings. Stable external reason codes remain the
// EvidenceOmission.Reason strings; the sentinel errors are
// package-private implementation details.
package digest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Typed sentinel errors for inspect-time classification. These
// values are stable across package versions and are the only
// authoritative way for the collector to decide which stable
// EvidenceOmission reason code to record.
var (
	errEvidenceSymlink    = errors.New("evidence: symlink at path")
	errEvidenceNonRegular = errors.New("evidence: non-regular file mode")
)

// InspectSymlink reports whether an inspect-time error
// indicates the path resolved to a symlink. Used by the
// collector to pick the OmitSymlink omission reason.
func InspectSymlink(err error) bool { return errors.Is(err, errEvidenceSymlink) }

// InspectNonRegular reports whether an inspect-time error
// indicates the path resolved to a non-regular file mode
// (directory, FIFO, socket, device, ...). Used by the
// collector to pick the OmitNonRegular omission reason.
func InspectNonRegular(err error) bool { return errors.Is(err, errEvidenceNonRegular) }

// inspectEvidenceFile classifies a single relative path under
// canonicalRoot. Only non-symlink, regular files survive;
// symlinks, directories, FIFOs, sockets, device nodes, and
// missing entries return a typed sentinel error so the
// collector can pick a stable omission reason without parsing
// strings. Any other I/O error is returned verbatim so the
// collector records OmitInspectFailed.
func inspectEvidenceFile(canonicalRoot, relPath string) (evidenceEntry, error) {
	full := filepath.Join(canonicalRoot, filepath.FromSlash(relPath))
	lst, lerr := os.Lstat(full)
	if lerr != nil {
		return evidenceEntry{}, fmt.Errorf("inspect %s: %w", relPath, lerr)
	}
	mode := lst.Mode()
	if mode&os.ModeSymlink != 0 {
		return evidenceEntry{}, errEvidenceSymlink
	}
	if !mode.IsRegular() {
		return evidenceEntry{}, errEvidenceNonRegular
	}
	return evidenceEntry{Size: lst.Size()}, nil
}
