// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// bindExactPlanPath requires that the supplied plan path resolves to the
// canonical repoRoot/docs/closure-plans/<ACT>.json path. It rejects any
// path that does not resolve to the canonical location. Relative paths
// are resolved against repoRoot.
func bindExactPlanPath(repoRoot, actID, suppliedPath string) (string, error) {
	canonical := filepath.Join(repoRoot, "docs", "closure-plans", actID+".json")

	// Resolve relative paths against repoRoot.
	absPath := suppliedPath
	if !filepath.IsAbs(suppliedPath) {
		absPath = filepath.Join(repoRoot, suppliedPath)
	}

	canonicalResolved, err := filepath.EvalSymlinks(canonical)
	if err != nil {
		return "", fmt.Errorf("resolve canonical plan path: %w", err)
	}
	suppliedResolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve supplied plan path: %w", err)
	}

	if suppliedResolved != canonicalResolved {
		return "", fmt.Errorf("plan path must be exactly %s, got %s", canonicalResolved, suppliedResolved)
	}

	return "docs/closure-plans/" + actID + ".json", nil
}

// ProvenanceTopology pins the precise, named roles of the four
// commits a closure proof MUST distinguish. Conflating them — for
// example by calling parent(F) the "base" — is the kind of bug that
// silently corrupts the patch-hygiene range and the closure-policy
// range.
//
//	plan baseline B       = historical implementation comparison
//	                        base (free parameter declared by the
//	                        plan; typically a much older release tag)
//	freeze parent P       = commit immediately before the frozen
//	                        plan F (verified by `git rev-list --parents
//	                        -n 1 F`)
//	freeze F              = commit that introduced the final plan
//	                        blob at the canonical plan path
//	subject S             = exactly one implementation commit after F
//	                        (F = parent(S), no merge, no root)
//
// In other words the historical ACT scope policy range is
// plan.baseline..S (B..S) while the subject-only patch hygiene range
// is F..S. The shared word "base" must NEVER select one of these by
// accident; see run_v2_policy.go for the explicit decision.
type ProvenanceTopology struct {
	B string // plan.baseline.commit_oid
	P string // parent(F)
	F string // freeze commit
	S string // subject commit
}

// bindExactPlanBytes requires that the executed bytes match the blob at
// F for the canonical plan path without any trimming or normalization.
//
// Invariants:
//
//	executedBytes == blob(F:path) (byte-for-byte)
//	blob(F:path) == blob(S:path)
//	blob(parent(F):path) != blob(F:path)
func bindExactPlanBytes(
	ctx context.Context,
	git gitClient,
	repoRoot string,
	objectFormat ObjectFormat,
	planPath string,
	freezeCommit string,
	subjectCommit string,
	executedBytes []byte,
) (planBlobOID string, err error) {
	blobAtF, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", freezeCommit+":"+planPath)
	if err != nil {
		return "", fmt.Errorf("resolve plan blob at F: %w", err)
	}
	if err := ValidateOIDWithFormat("blob(F:plan)", blobAtF, objectFormat); err != nil {
		return "", err
	}

	blobAtS, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", subjectCommit+":"+planPath)
	if err != nil {
		return "", fmt.Errorf("resolve plan blob at S: %w", err)
	}
	if err := ValidateOIDWithFormat("blob(S:plan)", blobAtS, objectFormat); err != nil {
		return "", err
	}

	if blobAtF != blobAtS {
		return "", fmt.Errorf("plan blob differs between F (%s) and S (%s)", blobAtF, blobAtS)
	}

	// F must itself have exactly one parent P (NOT necessarily equal to
	// plan.baseline B). The P -> F -> S geometry is what we prove here;
	// the historical B -> S range is a separate, plan-declared concern.
	freezeParentCommit, err := verifySingleParent(ctx, git, repoRoot, freezeCommit, objectFormat)
	if err != nil {
		return "", fmt.Errorf("freeze commit must be a non-merge single-parent commit: %w", err)
	}
	lsResult := git.Run(ctx, repoRoot,
		"ls-tree", "--full-tree",
		"--format=%(objecttype)%x09%(objectname)%x09%(path)",
		freezeParentCommit, "--", planPath,
	)
	if lsResult.Err != nil || lsResult.ExitCode != 0 {
		return "", fmt.Errorf("git ls-tree on P (parent of F) failed: %v: %s", lsResult.Err, lsResult.Stderr)
	}
	blobAtP, present, err := parsePlanTreeRecord(lsResult.Stdout, planPath, objectFormat)
	if err != nil {
		return "", err
	}
	if present && blobAtP == blobAtF {
		return "", fmt.Errorf("F did not introduce the final plan blob (parent of F has same blob)")
	}

	// Read authoritative bytes from Git (no trimming)
	authoritativeResult := git.Run(ctx, repoRoot, "cat-file", "blob", blobAtF)
	if authoritativeResult.Err != nil || authoritativeResult.ExitCode != 0 {
		return "", fmt.Errorf("read plan blob: %v", authoritativeResult.Err)
	}
	if !bytesEqual(authoritativeResult.Stdout, executedBytes) {
		return "", fmt.Errorf("executed plan bytes do not match blob at F")
	}

	return blobAtF, nil
}

func parsePlanTreeRecord(output []byte, planPath string, objectFormat ObjectFormat) (string, bool, error) {
	if len(output) == 0 {
		return "", false, nil
	}
	lines := bytesSplitLines(output)
	if len(lines) != 1 {
		return "", false, fmt.Errorf("git ls-tree returned %d records; want exactly one", len(lines))
	}
	fields := bytes.Split(lines[0], []byte{'\t'})
	if len(fields) != 3 {
		return "", false, fmt.Errorf("malformed git ls-tree record %q", lines[0])
	}
	if string(fields[0]) != "blob" {
		return "", false, fmt.Errorf("git ls-tree object type is %q; want blob", fields[0])
	}
	blobOID := string(fields[1])
	if err := ValidateOIDWithFormat("blob(P:plan)", blobOID, objectFormat); err != nil {
		return "", false, err
	}
	if string(fields[2]) != planPath {
		return "", false, fmt.Errorf("git ls-tree path is %q; want %q", fields[2], planPath)
	}
	return blobOID, true, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func bytesSplitLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			lines = append(lines, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, b[start:])
	}
	return lines
}

func bytesSplitFields(b []byte) [][]byte {
	var fields [][]byte
	start := 0
	for i, c := range b {
		if c == ' ' || c == '\t' {
			if start < i {
				fields = append(fields, b[start:i])
			}
			start = i + 1
		}
	}
	if start < len(b) {
		fields = append(fields, b[start:])
	}
	return fields
}

// enforceRunnerIdentity requires runner identity to match S exactly.
// Production fail-closed: empty values are fatal.
func enforceRunnerIdentity(identity RunnerIdentity, subject, actualBinarySHA256 string) error {
	if identity.VCSRevision == "" {
		return fmt.Errorf("runner VCS revision is empty")
	}
	if identity.VCSRevision != subject {
		return fmt.Errorf("runner VCS revision (%s) does not match subject (%s)", identity.VCSRevision, subject)
	}
	if identity.VCSModified {
		return fmt.Errorf("runner is built from modified sources")
	}
	if identity.BinarySHA256 == "" {
		return fmt.Errorf("runner BinarySHA256 is empty")
	}
	if actualBinarySHA256 == "" {
		return fmt.Errorf("actual binary SHA256 is empty")
	}
	if identity.BinarySHA256 != actualBinarySHA256 {
		return fmt.Errorf("runner binary SHA256 mismatch: identity=%s actual=%s", identity.BinarySHA256, actualBinarySHA256)
	}
	return nil
}

// identifyRunningBinary computes the SHA256 of the running binary.
func identifyRunningBinary() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", err
	}
	f, err := os.Open(execPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// readBlobBytesViaGit reads the contents of a Git blob by OID.
func readBlobBytesViaGit(ctx context.Context, git gitClient, repoRoot, blobOID string) ([]byte, error) {
	result := git.Run(ctx, repoRoot, "cat-file", "blob", blobOID)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("git cat-file blob %s failed: %v", blobOID, result.Err)
	}
	return result.Stdout, nil
}
