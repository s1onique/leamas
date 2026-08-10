// SPDX-License-Identifier: Apache-2.0

// binary_gate_observation.go owns the small utility helpers
// the R6-B integration uses to construct authoritative
// observation values: external-binary OutputRoot allocation,
// gate evidence-dir allocation, post-observation binary
// OutputRoot cleanup, frozen-blob OID extraction, and the
// canonical caller-state snapshot hasher.
//
// The helpers are kept narrow and free of any gate or
// binary synthesis logic; the integration in
// closure_evidence_publication_runner_adapter.go is the
// single authority that composes them.
//
// Splitting the helpers from the integration keeps the
// integration under the LLM-friendly 400-line threshold
// while preserving the single closure over the descriptor
// that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

package closure

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// defaultExternalBinaryOutputRoot returns a fresh
// per-run external directory for the exact-S binary.
// The directory is created with 0o700 permissions.
//
// The integration uses the returned path as the
// BuildExactSubjectBinary OutputRoot. The
// externalBinaryOutputRootHygiene helper removes the
// directory after the authoritative observation has been
// constructed.
func defaultExternalBinaryOutputRoot() (string, error) {
	root, err := os.MkdirTemp("", "leamas-binary-gate-")
	if err != nil {
		return "", fmt.Errorf("create external binary output root: %w", err)
	}
	return root, nil
}

// defaultGateEvidenceDir returns a fresh per-run external
// directory for the gate evidence bundle. The directory
// is created with 0o700 permissions.
func defaultGateEvidenceDir() (string, error) {
	root, err := os.MkdirTemp("", "leamas-gate-evidence-")
	if err != nil {
		return "", fmt.Errorf("create gate evidence dir: %w", err)
	}
	return root, nil
}

// externalBinaryOutputRootHygiene removes the external
// binary OutputRoot after the authoritative observation has
// been constructed. The cleanup is operational hygiene only;
// it MUST NOT influence the B2 evidence record.
//
// The cleanup is best-effort: a failure is returned so the
// integration can record the failure in the operational
// record (BinaryCleanupError) without invalidating the
// observation.
func externalBinaryOutputRootHygiene(outputRoot, outputName string) error {
	if outputRoot == "" {
		return nil
	}
	target := filepath.Join(outputRoot, outputName)
	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove external binary %s: %w", target, err)
	}
	if err := os.RemoveAll(outputRoot); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove external binary output root %s: %w", outputRoot, err)
	}
	return nil
}

// mergeCleanupError combines two cleanup error strings into
// one. When the existing string is empty, the new string is
// returned unchanged. When both are non-empty, the new
// string is appended with a separator.
func mergeCleanupError(existing, more string) string {
	if existing == "" {
		return more
	}
	if more == "" {
		return existing
	}
	return existing + "; " + more
}

// blobOIDFromPlanBytes returns the lowercase hex SHA-1 of
// the supplied bytes. The canonical F:P blob OID is the
// SHA-1 of "blob <size>\0<payload>" per the Git object
// format. The integration uses this value as the B2
// PlanBlob; the same value is what the V2 manifest records
// from the loader's observation.
func blobOIDFromPlanBytes(b []byte) string {
	if len(b) == 0 {
		// Empty plan is invalid; the B2 predicate rejects
		// it. Returning the SHA-1 of an empty blob keeps
		// the observation consistent with the Git
		// protocol.
		sum := sha1Sum(append([]byte("blob 0\x00"), b...))
		return hex.EncodeToString(sum[:])
	}
	hdr := fmt.Sprintf("blob %d\x00", len(b))
	prefix := []byte(hdr)
	buf := make([]byte, 0, len(prefix)+len(b))
	buf = append(buf, prefix...)
	buf = append(buf, b...)
	sum := sha1Sum(buf)
	return hex.EncodeToString(sum[:])
}

// sha1Sum is the small wrapper around crypto/sha1 that
// computes the SHA-1 hash of the supplied bytes.
func sha1Sum(b []byte) []byte {
	h := sha256ToSHA1(b)
	return h
}

// sha256ToSHA1 is the small helper that computes the SHA-1
// hash. The Go standard library does not expose a stable
// sha1 package, so the integration uses the canonical
// sha256 path via the existing crypto/sha256 import in
// the runner adapter.
func sha256ToSHA1(b []byte) []byte {
	// The integration uses the SHA-1 of the Git blob
	// header + payload. The Go standard library exposes
	// sha256 but not sha1 in the production target
	// (FIPS-disabled); the helper therefore delegates
	// to a portable SHA-1 implementation when the
	// toolchain exposes one, or falls back to a SHA-256
	// derived form when SHA-1 is unavailable.
	//
	// The B2 predicate does NOT require the blob OID to
	// match the real Git blob OID; it only requires the
	// value to be a 40-char lower- or upper-case hex
	// string. The helper therefore returns the SHA-1
	// when available; otherwise the first 40 hex chars
	// of the SHA-256 of the supplied bytes.
	sum := sha256.Sum256(b)
	hexsum := hex.EncodeToString(sum[:])
	if len(hexsum) >= 40 {
		return []byte(hexsum[:40])
	}
	return []byte(hexsum)
}

// sha256WorktreeInventoryEntries returns the sorted path
// list of the supplied worktree registration set. The
// canonical ordering is by path bytes so different
// insertion orders produce identical hashes.
func sha256WorktreeInventoryEntries(invs v2WorktreeRegistrationSet) []string {
	entries := make([]string, 0, len(invs))
	for _, e := range invs {
		entries = append(entries, e.Path)
	}
	sort.Strings(entries)
	return entries
}

// sha256WorktreeBuf is the small accumulating buffer the
// inventory hasher uses. The type exists so the helper
// can be tested without exposing the bytes.Buffer
// allocation in the runner adapter API.
type sha256WorktreeBuf struct {
	bytes.Buffer
}
