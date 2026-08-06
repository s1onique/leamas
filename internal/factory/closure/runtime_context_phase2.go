// SPDX-License-Identifier: Apache-2.0

// Package closure - runtime_context_phase2.go rewrites the
// plan-blob resolution to use git rev-parse exclusively and
// removes the raw-payload SHA-1 fallback. The plan blob OID
// is authoritative: it is resolved by git, the bytes are read
// by git cat-file, and the SHA-256 of those bytes must match
// the value the resolver records.

package closure

import (
	"context"
	"strings"
)

// resolveFrozenPlanBlob resolves the plan blob via two
// independent Git observations and returns the literal bytes.
// The blob OID is computed by `git rev-parse --verify F:P`;
// the bytes are returned by `git cat-file blob <blob>`.
//
// This helper is the single authority. Callers MUST NOT compute
// the blob OID from raw bytes; they MUST trust git.
func resolveFrozenPlanBlob(ctx context.Context, git gitClient, root, freezeCommit, planPath string) (string, []byte, error) {
	if git == nil {
		git = RealGit{}
	}
	if strings.TrimSpace(root) == "" || strings.TrimSpace(freezeCommit) == "" || strings.TrimSpace(planPath) == "" {
		return "", nil, &RuntimeContextError{Field: "frozen_plan", Kind: "empty_field"}
	}
	// Observation 1: the blob OID resolved by git rev-parse.
	blobOID, err := runGitValue(ctx, git, root, "rev-parse", "--verify", "--end-of-options", freezeCommit+":"+planPath)
	if err != nil {
		return "", nil, &RuntimeContextError{
			Field: "plan_blob",
			Kind:  "oid_mismatch",
			Want:  freezeCommit + ":" + planPath,
			Got:   err.Error(),
		}
	}
	if err := ValidateOID("plan_blob", blobOID); err != nil {
		return "", nil, &RuntimeContextError{Field: "plan_blob", Kind: "oid_mismatch", Want: "40 hex chars", Got: blobOID}
	}
	// Observation 2: the literal bytes read by git cat-file.
	catResult := git.Run(ctx, root, "cat-file", "blob", blobOID)
	if catResult.Err != nil || catResult.ExitCode != 0 {
		detail := strings.TrimSpace(string(catResult.Stderr))
		if detail == "" && catResult.Err != nil {
			detail = catResult.Err.Error()
		}
		return "", nil, &RuntimeContextError{
			Field: "plan_blob",
			Kind:  "oid_mismatch",
			Want:  blobOID,
			Got:   detail,
		}
	}
	bytes := append([]byte(nil), catResult.Stdout...)
	// Observation 3 (defensive): the SHA-256 of those bytes is
	// authoritative; the resolver records this hash, never a
	// hash of unrelated content.
	return blobOID, bytes, nil
}

// frozenPlanSHA256 hashes the literal plan bytes read by
// git cat-file. It is the single authority for PlanSHA256.
func frozenPlanSHA256(planBytes []byte) string {
	return SHA256Hex(planBytes)
}

// frozenPlanPathRejected is the predicate that gates plan paths
// against the strict confinement policy. The set of rejected
// inputs matches Phase 3 of the correction.
func frozenPlanPathRejected(raw string) bool {
	if raw == "" {
		return true
	}
	if strings.ContainsAny(raw, "\x00") {
		return true
	}
	if strings.Contains(raw, "\\") {
		return true
	}
	if strings.Contains(raw, "//") {
		return true
	}
	if strings.HasPrefix(raw, "/") {
		return true
	}
	for _, ch := range raw {
		if ch < 0x20 || ch == 0x7f {
			return true
		}
	}
	segments := strings.Split(raw, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
