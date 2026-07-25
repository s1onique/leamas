// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// verifyExistingTransactionExact is the strict, parameterised
// verifier. It accepts the Git client and repository root explicitly
// so it cannot be misconfigured by a caller deriving the root
// internally.
//
// The verifier regenerates the expected C / E / T identities from
// the authoritative plan and the qualified runtime evidence, then
// proves every observable of the published state against the
// reconstructed values:
//
//   - manifest bytes and blob
//   - report bytes and blob
//   - C_TREE geometry (S_TREE + exactly two canonical additions)
//   - C commit bytes/identity/parent/message/author/committer/epoch
//   - evidence index and every indexed file
//   - tag bytes and tag-object OID
//   - branch ref, tag ref, HEAD
//   - index tree
//   - worktree files/status
//
// Any mismatch is a fatal error.
func verifyExistingTransactionExact(ctx context.Context, git gitClient, repoRoot, evidenceDir string,
	expected v2ExpectedTransaction, evidence v2EvidenceSnapshot) (*TransactionResult, error) {
	if git == nil {
		return nil, fmt.Errorf("git client is required")
	}
	if repoRoot == "" {
		return nil, fmt.Errorf("repository root is required")
	}
	if expected.Tag.ActID == "" {
		return nil, fmt.Errorf("expected transaction is missing act_id")
	}
	subject := expected.SubjectCommit
	if subject == "" {
		return nil, fmt.Errorf("expected transaction is missing subject")
	}
	if !evidence.Present {
		return nil, fmt.Errorf("qualified evidence snapshot is absent")
	}
	if evidence.Runtime.ActID != expected.Tag.ActID || evidence.Runtime.Runner.VCSRevision != subject {
		return nil, fmt.Errorf("qualified evidence snapshot identity mismatch")
	}
	if SHA256Hex(evidence.IndexBytes) != evidence.IndexHash || evidence.IndexHash != expected.EvidenceHash {
		return nil, fmt.Errorf("evidence index hash %s does not match expected %s",
			evidence.IndexHash, expected.EvidenceHash)
	}

	// 2. Manifest blob at C must match the reconstructed manifest bytes.
	manifestBytes, err := readBlobAtCommit(ctx, git, repoRoot, expected.CommitObject.OID,
		canonicalV2ManifestPath(expected.Tag.ActID))
	if err != nil {
		return nil, fmt.Errorf("read manifest blob at C: %w", err)
	}
	if !bytes.Equal(manifestBytes, expected.Artifacts.ManifestBytes) {
		return nil, fmt.Errorf("manifest blob at C does not match regenerated manifest bytes")
	}
	if SHA256Hex(manifestBytes) != expected.Artifacts.ManifestSHA256 {
		return nil, fmt.Errorf("manifest SHA-256 at C does not match regenerated SHA-256")
	}

	// 3. Report blob at C must match the reconstructed report bytes.
	reportBytes, err := readBlobAtCommit(ctx, git, repoRoot, expected.CommitObject.OID,
		canonicalV2ReportPath(expected.Tag.ActID))
	if err != nil {
		return nil, fmt.Errorf("read report blob at C: %w", err)
	}
	if !bytes.Equal(reportBytes, expected.Artifacts.ReportBytes) {
		return nil, fmt.Errorf("report blob at C does not match regenerated report bytes")
	}
	if SHA256Hex(reportBytes) != expected.Artifacts.ReportSHA256 {
		return nil, fmt.Errorf("report SHA-256 at C does not match regenerated SHA-256")
	}

	// 4. C_TREE == S_TREE + exactly two canonical additions.
	if err := verifyV2ClosureTree(ctx, git, repoRoot, ObjectFormatFromOID(subject),
		subjectTreeFor(ctx, git, repoRoot, subject), expected.Objects.TreeOID,
		canonicalV2ManifestPath(expected.Tag.ActID), expected.Objects.ManifestBlobOID,
		canonicalV2ReportPath(expected.Tag.ActID), expected.Objects.ReportBlobOID); err != nil {
		return nil, fmt.Errorf("verify closure tree: %w", err)
	}

	// 5. C commit must equal the reconstructed identity with parent=S.
	if err := verifySingleParentIs(ctx, git, repoRoot, expected.CommitObject.OID, subject); err != nil {
		return nil, fmt.Errorf("verify closure commit parent: %w", err)
	}
	if err := verifyCommitExact(ctx, git, repoRoot, expected.CommitObject.OID, expected.Objects.TreeOID,
		subject, "chore(closure): close "+expected.Tag.ActID+"\n", v2ClosureName, v2ClosureEmail); err != nil {
		return nil, fmt.Errorf("verify closure commit bytes: %w", err)
	}

	// 6. Tag must be present, annotated, peel to C, and have exact bytes.
	tagPeeled, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify",
		"refs/tags/"+expected.Tag.Name+"^{commit}")
	if err != nil || tagPeeled != expected.CommitObject.OID {
		return nil, fmt.Errorf("tag does not peel to expected C: peeled=%s expected=%s err=%v",
			tagPeeled, expected.CommitObject.OID, err)
	}
	tagObjectOID, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify",
		"refs/tags/"+expected.Tag.Name+"^{tag}")
	if err != nil || tagObjectOID != expected.TagObject.OID {
		return nil, fmt.Errorf("tag-object OID %s != expected %s (err=%v)",
			tagObjectOID, expected.TagObject.OID, err)
	}
	tagBytesFromGit, err := readTagObjectBytes(ctx, git, repoRoot, expected.TagObject.OID)
	if err != nil {
		return nil, fmt.Errorf("read tag object bytes: %w", err)
	}
	if !bytes.Equal(tagBytesFromGit, expected.Tag.Bytes) {
		return nil, fmt.Errorf("tag object bytes do not match reconstructed bytes")
	}

	// 7. Publication branch (recorded in runtime.json) must equal C, and HEAD
	// must be attached to that same branch. This is the publication-target
	// binding that prevents recovery from redirecting refs to an unauthorised
	// branch while reusing the original evidence snapshot.
	publicationBranch := evidence.Runtime.PublicationBranch
	if publicationBranch == "" {
		return nil, fmt.Errorf("runtime publication branch is empty")
	}
	pubOID, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "refs/heads/"+publicationBranch)
	if err != nil || pubOID != expected.CommitObject.OID {
		return nil, fmt.Errorf("publication branch %s resolves to %s, expected %s",
			publicationBranch, pubOID, expected.CommitObject.OID)
	}
	branchName, err := runGitValue(ctx, git, repoRoot, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || branchName == "" {
		return nil, fmt.Errorf("HEAD must be attached to a branch: %v", err)
	}
	if branchName != publicationBranch {
		return nil, fmt.Errorf("HEAD attached to %s, but publication branch is %s",
			branchName, publicationBranch)
	}
	branchOID, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "refs/heads/"+branchName)
	if err != nil || branchOID != expected.CommitObject.OID {
		return nil, fmt.Errorf("branch %s resolves to %s, expected %s", branchName, branchOID, expected.CommitObject.OID)
	}

	// 8. HEAD must equal C.
	headCommit, err := runGitValue(ctx, git, repoRoot, "rev-parse", "HEAD^{commit}")
	if err != nil || headCommit != expected.CommitObject.OID {
		return nil, fmt.Errorf("HEAD %s != expected C %s", headCommit, expected.CommitObject.OID)
	}

	// 9. Index tree must equal C_TREE.
	indexTree, err := runGitValue(ctx, git, repoRoot, "write-tree")
	if err != nil || indexTree != expected.Objects.TreeOID {
		return nil, fmt.Errorf("index tree %s != expected C_TREE %s", indexTree, expected.Objects.TreeOID)
	}

	// 10. Worktree must be clean and contain the canonical manifest/report
	//     files matching the blobs at C.
	status, err := runGitValue(ctx, git, repoRoot, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || status != "" {
		return nil, fmt.Errorf("worktree is not clean at C: status=%q err=%v", status, err)
	}
	for _, path := range []string{canonicalV2ManifestPath(expected.Tag.ActID), canonicalV2ReportPath(expected.Tag.ActID)} {
		worktreeBytes, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("missing canonical artifact %s: %w", path, err)
		}
		var refBytes []byte
		if path == canonicalV2ManifestPath(expected.Tag.ActID) {
			refBytes = manifestBytes
		} else {
			refBytes = reportBytes
		}
		if !bytes.Equal(worktreeBytes, refBytes) {
			return nil, fmt.Errorf("worktree %s diverges from blob at C", path)
		}
	}

	return &TransactionResult{
		ActID:            expected.Tag.ActID,
		FreezeCommit:     expected.FreezeCommit,
		SubjectCommit:    subject,
		ClosureCommit:    expected.CommitObject.OID,
		ClosureTree:      expected.Objects.TreeOID,
		TagName:          expected.Tag.Name,
		TagObject:        expected.TagObject.OID,
		TagTarget:        expected.CommitObject.OID,
		EvidencePath:     evidenceDir,
		EvidenceHash:     expected.EvidenceHash,
		Runner:           evidence.Runtime.Runner,
		Verdict:          VerdictPass,
		TransactionState: v2StateVerified,
	}, nil
}

// readTagObjectBytes uses `git cat-file tag <OID>` to read the exact
// raw annotated-tag bytes for a tag object.
func readTagObjectBytes(ctx context.Context, git gitClient, repoRoot, oid string) ([]byte, error) {
	result := git.Run(ctx, repoRoot, "cat-file", "tag", oid)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("git cat-file tag %s failed: %s", oid, gitFailureDetail(result))
	}
	return append([]byte(nil), result.Stdout...), nil
}

func verifySingleParentIs(ctx context.Context, git gitClient, repoRoot, commit, expectedParent string) error {
	parent, err := verifySingleParent(ctx, git, repoRoot, commit, ObjectFormatFromOID(commit))
	if err != nil {
		return err
	}
	if parent != expectedParent {
		return fmt.Errorf("commit %s has parent %s, expected %s", commit, parent, expectedParent)
	}
	return nil
}

func verifyCommitExact(ctx context.Context, git gitClient, repoRoot, commit, expectedTree, expectedParent, expectedMessage, expectedAuthor, expectedAuthorEmail string) error {
	format := "objectname %H%ntreename %T%nparent %P%nauthor %an <%ae> %ai%ncommitter %cn <%ce> %ci%nbody%n%B"
	details, err := runGitValue(ctx, git, repoRoot, "show", "-s", "--format="+format, commit)
	if err != nil {
		return fmt.Errorf("read commit metadata: %w", err)
	}
	if !strings.Contains(details, expectedTree) {
		return fmt.Errorf("commit %s tree does not match", commit)
	}
	if !strings.Contains(details, "parent "+expectedParent) {
		return fmt.Errorf("commit %s parent does not match", commit)
	}
	if !strings.Contains(details, expectedAuthor+" <"+expectedAuthorEmail+">") {
		return fmt.Errorf("commit %s author/committer does not match", commit)
	}
	if !strings.Contains(details, "Leamas Closure <closure@leamas.local>") {
		return fmt.Errorf("commit %s did not use the closure identity", commit)
	}
	expectedMessage = strings.TrimSuffix(expectedMessage, "\n")
	if !strings.HasSuffix(details, expectedMessage) {
		return fmt.Errorf("commit %s message does not match %q", commit, expectedMessage)
	}
	return nil
}

func subjectTreeFor(ctx context.Context, git gitClient, repoRoot, subject string) string {
	tree, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", subject+"^{tree}")
	if err != nil {
		return ""
	}
	return tree
}

// tagRefOID returns the OID of a tag reference, the existence flag,
// and a non-nil error if Git itself failed. Only "not found" results
// are classified as absence. All other failures are propagated.
func tagRefOID(ctx context.Context, git gitClient, repoRoot, tagName string) (string, bool, error) {
	result := git.Run(ctx, repoRoot, "rev-parse", "--verify", "refs/tags/"+tagName)
	stderr := string(result.Stderr)
	if result.Err != nil || result.ExitCode != 0 {
		if strings.Contains(stderr, "Needed a single revision") ||
			strings.Contains(stderr, "unknown revision") ||
			strings.Contains(stderr, "not found") {
			return "", false, nil
		}
		if result.Err != nil {
			return "", false, fmt.Errorf("git rev-parse %s: %w", tagName, result.Err)
		}
		return "", false, fmt.Errorf("git rev-parse %s failed (exit %d): %s",
			tagName, result.ExitCode, sanitizeDiagnostic(stderr))
	}
	return strings.TrimSpace(string(result.Stdout)), true, nil
}
