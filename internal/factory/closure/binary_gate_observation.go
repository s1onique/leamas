// SPDX-License-Identifier: Apache-2.0

// binary_gate_observation.go owns the small utility helpers
// the R6-B integration uses to construct authoritative
// observation values: external-binary OutputRoot allocation,
// gate evidence-dir allocation, and post-observation binary
// OutputRoot operational cleanup.
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
	"fmt"
	"os"
	"path/filepath"
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
// record (separately from the B2 CleanupAuthority) without
// invalidating the observation.
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
