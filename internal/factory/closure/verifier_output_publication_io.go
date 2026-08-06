// SPDX-License-Identifier: Apache-2.0

package closure

// verifier_output_publication_io.go isolates the publication
// lifecycle helpers (temp-name randomization, pre-rename
// failure result construction, parent-dir fsync) from the
// public PrepareVerifierOutput and VerifierOutputAuthority
// surface in verifier_output_publication.go. Splitting along
// the I/O boundary keeps each file under the LLM-friendliness
// 400-line threshold while preserving the race-resistant
// publication invariants from CORRECTION02B.
//
// The helpers here are deliberately small and deterministic.
// Temp-name randomization uses crypto/rand so an attacker that
// pre-creates verifier-output files cannot predict the next
// name; the production publication loop combines the random
// component with an internal counter for fast duplicate
// detection and exponential fall-back to the 2^20 bound.

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// randomizedTempName returns a basename for a temp file the
// verifier writes inside its opened parent. The basename
// combines a stable prefix and suffix for human readability
// with a 16-byte random hex component for collision resistance.
// A local actor that pre-creates files in the directory can
// delay the publication by one attempt but cannot predict the
// next name; the OpenFile(O_CREATE|O_EXCL) call atomically
// decides uniqueness.
//
// crypto/rand.Read is fail-closed: a CSPRNG failure is
// returned to the caller, the publication is aborted with
// not_published, and the verifier surfaces a typed diagnostic.
// We do NOT fall back to deterministic bytes because that
// would let an attacker pre-create the predictable name and
// break the openat race window.
func randomizedTempName(prefix, suffix string) (string, error) {
	return randomizedTempNameFromReader(prefix, suffix, rand.Reader)
}

func randomizedTempNameFromReader(prefix, suffix string, reader io.Reader) (string, error) {
	if reader == nil {
		reader = rand.Reader
	}
	var raw [16]byte
	if _, err := io.ReadFull(reader, raw[:]); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return prefix + hex.EncodeToString(raw[:]) + suffix, nil
}

// failNotPublished returns a PublicationResult whose state is
// PublicationNotPublished and whose error wraps the underlying
// cause. The function is reserved for the early-failure paths
// of Publish so every error in a single place returns the
// same shape. It also best-effort removes a stray temp file
// under the opened parent when one exists.
func (a *VerifierOutputAuthority) failNotPublished(err error, tempRel string) PublicationResult {
	if tempRel != "" && a != nil && a.root != nil && a.publicationFS != nil {
		_ = a.publicationFS.remove(a.root, tempRel)
	}
	return PublicationResult{
		State:         PublicationNotPublished,
		CanonicalPath: a.canonicalOrEmpty(),
		Err:           err,
	}
}

// canonicalOrEmpty returns a.canonical or the empty string when
// a is nil. Used from failNotPublished to keep the result
// canonical path field stable for non-nil and nil receivers.
func (a *VerifierOutputAuthority) canonicalOrEmpty() string {
	if a == nil {
		return ""
	}
	return a.canonical
}
