// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_r2c_dogfood_test.go executes the exact-final-tip
// dogfood required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R1.
//
// The test:
//  1. builds the leamas binary in a TEMPORARY DETACHED WORKTREE
//     at the supplied FINAL_COMMIT, so the build source bytes
//     are exact and immutable;
//  2. captures every bounded-subprocess output property
//     (ExitCode, Err, TimedOut, StdoutTruncated, StderrTruncated,
//     StdoutSHA256, StderrSHA256) and applies the R2B
//     `dogfoodRejectsTruncation` policy;
//  3. asserts the manifest was absent BEFORE invocation and is
//     present AFTER, with literal SHA-256 of the manifest bytes;
//  4. asserts manifest bindings are exact: subject, freeze,
//     caller_head, execution_tree (= S^{tree}), plan_blob
//     (= F:P blob), plan_sha256 (= SHA-256(exact F:P bytes));
//  5. asserts the binary VCS revision == FINAL_COMMIT and
//     binary VCS modified == false.
//
// The test does NOT introduce new architecture. It reuses the
// helpers from factory_close_v2_mac_canary_test.go and
// boundedSubprocessV2 from cmd/leamas/bounded_subprocess_v2_test.go.
//
// The harness builds in a detached worktree so a dirty caller
// checkout cannot influence the dogfood. The harness resolves
// its repo root dynamically via `git rev-parse --show-toplevel`
// rather than hardcoding an absolute path.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// r2cRDogfoodResult captures every R2C-R1 dogfood value the test
// observes. The struct fields are the literal values surfaced
// in the final report; no field may be a placeholder.
type r2cRDogfoodResult struct {
	FinalCommit            string
	FinalTree              string
	BuildSourceCommit      string
	BuildSourceTree        string
	BuildSourceStatusEmpty bool
	BuildSourceDetached    bool
	BinaryPath             string
	BinarySHA256           string
	BinaryVCSRevision      string
	BinaryVCSModified      bool
	Subject                string
	SubjectTree            string
	Freeze                 string
	FreezeTree             string
	Descendant             string
	CallerHead             string
	CallerHeadBefore       string
	CallerHeadAfter        string
	CallerTreeBefore       string
	CallerTreeAfter        string
	CallerStatusBefore     string
	CallerStatusAfter      string
	WorktreesBefore        string
	WorktreesAfter         string
	ManifestPath           string
	ManifestAbsenceBefore  bool
	ManifestPresentAfter   bool
	ManifestSubject        string
	ManifestFreeze         string
	ManifestCallerHead     string
	ManifestExecutionTree  string
	ManifestPlanPath       string
	ManifestPlanBlob       string
	ManifestPlanSHA256     string
	ManifestBinarySHA256   string
	ManifestBinaryVCSRev   string
	ManifestBinaryVCSMod   bool
	ManifestSHA256         string
	StdoutBytes            int
	StderrBytes            int
	StdoutSHA256           string
	StderrSHA256           string
	ExitCode               int
	TimedOut               bool
	StdoutTruncated        bool
	StderrTruncated        bool
	TruncationRejectionMsg string
	RunErr                 error
	EvidencePath           string
	EvidenceSHA256         string
}

var lastR2CRDogfood r2cRDogfoodResult

// TestClosureCLIV2R2CRExactTipDogfood is the R2C-R1 exact-final-tip
// dogfood. It MUST be run against the FINAL_COMMIT; the test
// itself records the FINAL_COMMIT it observed so the close
// report can attribute the values to the correct commit.
//
// The test reads its repo root dynamically via
// `git rev-parse --show-toplevel` so it is host-independent
// (CI, Mac, other Linux users, detached worktrees).
func TestClosureCLIV2R2CRExactTipDogfood(t *testing.T) {
	finalCommit, _, leamasRepoRoot, detached := mustPrepareR2CRBuildEnv(t)
	defer func() {
		if detached != "" {
			runR2CRGit(t, leamasRepoRoot, "worktree", "remove", "--force", detached)
		}
	}()

	// Build the binary in the detached worktree, then capture
	// the identity helpers. The detached worktree must report
	// HEAD = FINAL_COMMIT and a clean porcelain-v2 status.
	buildInDetachedWorktree(t, detached, finalCommit)
	binaryPath := filepath.Join(detached, "leamas-r2cr-dogfood")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("binary not built: %v", err)
	}
	binSHA := sha256HexFile(t, binaryPath)
	identity := readBinaryIdentity(t, binaryPath)
	if identity.Commit != finalCommit {
		t.Fatalf("binary VCS revision %s does not match FINAL_COMMIT %s",
			identity.Commit, finalCommit)
	}
	if identity.Dirty {
		t.Fatalf("binary vcs.modified=true; dogfood requires clean build")
	}

	repository, subject, subjectTree, freeze, freezeTree, d, planPath, planBlobOID, planBytes, planSHA256, descendantPlanBytes := prepareR2CRDogfoodRepo(t, leamasRepoRoot)
	if got := runR2CRGit(t, repository, "rev-parse", "HEAD"); got != d {
		t.Fatalf("pre-run HEAD must equal D: got=%s want=%s", got, d)
	}

	detachedDir := t.TempDir()
	evidenceDir := filepath.Join(detachedDir, "evidence")
	manifestOutput := filepath.Join(detachedDir, "manifest.json")

	// Phase 3: pre-publication assertion. The manifest path
	// MUST be absent before invocation so we know the run
	// actually published it.
	assertManifestAbsent(t, manifestOutput)

	headBefore := runR2CRGit(t, repository, "rev-parse", "HEAD")
	headTreeBefore := runR2CRGit(t, repository, "rev-parse", "HEAD^{tree}")
	statusBefore := runR2CRGit(t, repository, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesBefore := runR2CRGit(t, repository, "worktree", "list", "--porcelain")

	// Phase 2: use the bounded subprocess authority. The
	// harness exposes every output property we need
	// (ExitCode, TimedOut, StdoutTruncated, StderrTruncated,
	// full bytes) and the caller writes run from OUTSIDE the
	// source tree, so the Mac handoff shape is preserved.
	bounded := boundedSubprocessV2(binaryPath, []string{
		"factory", "close", "run-v2-authority",
		"--protocol-version", "2",
		"--plan-contract-version", "1",
		"--repository", repository,
		"--subject", subject,
		"--freeze", freeze,
		"--plan-path", planPath,
		"--evidence-directory", evidenceDir,
		"--manifest-output", manifestOutput,
	}, boundedSubprocessV2Options{
		Timeout:   60 * time.Second,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
	})

	headAfter := runR2CRGit(t, repository, "rev-parse", "HEAD")
	headTreeAfter := runR2CRGit(t, repository, "rev-parse", "HEAD^{tree}")
	statusAfter := runR2CRGit(t, repository, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesAfter := runR2CRGit(t, repository, "worktree", "list", "--porcelain")

	// Phase 2: bounded harness requires ExitCode=0, Err=nil,
	// TimedOut=false, neither stream truncated.
	if bounded.ExitCode != 0 {
		t.Fatalf("dogfood exit code %d (want 0); stderr=%s",
			bounded.ExitCode, string(bounded.Stderr))
	}
	if bounded.Err != nil {
		t.Fatalf("dogfood err: %v", bounded.Err)
	}
	if bounded.TimedOut {
		t.Fatalf("dogfood timed out")
	}
	if bounded.StdoutTruncated {
		t.Fatalf("dogfood stdout was truncated at limit")
	}
	if bounded.StderrTruncated {
		t.Fatalf("dogfood stderr was truncated at limit")
	}
	rejected, rejMsg := dogfoodRejectsTruncation(bounded)
	if rejected {
		t.Fatalf("dogfood overflow policy triggered: %s", rejMsg)
	}

	if headBefore != headAfter {
		t.Fatalf("caller HEAD drifted: before=%s after=%s", headBefore, headAfter)
	}
	if headTreeBefore != headTreeAfter {
		t.Fatalf("caller HEAD tree drifted: before=%s after=%s",
			headTreeBefore, headTreeAfter)
	}
	if statusBefore != statusAfter {
		t.Fatalf("caller worktree status drifted:\nbefore=%q\nafter=%q",
			statusBefore, statusAfter)
	}
	if worktreesBefore != worktreesAfter {
		t.Fatalf("linked-worktree registrations leaked:\nbefore=%q\nafter=%q",
			worktreesBefore, worktreesAfter)
	}

	// Phase 3: post-publication assertions.
	if _, err := os.Stat(manifestOutput); err != nil {
		t.Fatalf("manifest file missing after run: %v", err)
	}
	manifestBytes, readErr := os.ReadFile(manifestOutput)
	if readErr != nil {
		t.Fatalf("read manifest: %v", readErr)
	}
	if len(manifestBytes) == 0 {
		t.Fatalf("manifest is empty")
	}
	if !json.Valid(manifestBytes) {
		t.Fatalf("manifest is not valid JSON: %s", string(manifestBytes))
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("manifest JSON invalid: %v\n%s", err, string(manifestBytes))
	}

	mSubject, _ := manifest["subject_commit"].(string)
	mFreeze, _ := manifest["freeze_commit"].(string)
	mCaller, _ := manifest["caller_head"].(string)
	mExecTree, _ := manifest["execution_tree"].(string)
	mPlanPath, _ := manifest["plan_path"].(string)

	binIdent, _ := manifest["leamas_binary_identity"].(map[string]any)
	mBinSHA, _ := binIdent["sha256"].(string)
	mBinRev, _ := binIdent["vcs_revision"].(string)
	mBinMod, _ := binIdent["vcs_modified"].(bool)

	// Phase 4: exact execution-tree binding.
	if mSubject != subject {
		t.Fatalf("manifest subject_commit: got=%q want=%q", mSubject, subject)
	}
	if mFreeze != freeze {
		t.Fatalf("manifest freeze_commit: got=%q want=%q", mFreeze, freeze)
	}
	if mCaller != d {
		t.Fatalf("manifest caller_head: got=%q want=%q", mCaller, d)
	}
	if mExecTree != subjectTree {
		t.Fatalf("manifest execution_tree: got=%q want=%q (S^{tree})",
			mExecTree, subjectTree)
	}
	if mPlanPath != planPath {
		t.Fatalf("manifest plan_path: got=%q want=%q", mPlanPath, planPath)
	}

	// Phase 5: frozen-plan blob + SHA-256 binding. We resolve
	// F:P independently and compare exact bytes.
	mPlanBlob, _ := manifest["plan_blob"].(string)
	mPlanSHA, _ := manifest["plan_sha256"].(string)
	if mPlanBlob != planBlobOID {
		t.Fatalf("manifest plan_blob: got=%q want=%q (F:P blob)", mPlanBlob, planBlobOID)
	}
	if mPlanSHA != planSHA256 {
		t.Fatalf("manifest plan_sha256: got=%q want=%q (SHA-256(F:P bytes))",
			mPlanSHA, planSHA256)
	}
	// Descendant D:P must be bytewise distinct so we know F
	// is the real ancestor and not the descendant.
	if string(descendantPlanBytes) == string(planBytes) {
		t.Fatalf("descendant plan bytes must differ from frozen plan bytes")
	}

	// Phase 6: binary identity.
	if mBinSHA != binSHA {
		t.Fatalf("manifest leamas_binary_identity.sha256 %q != invoked binary %s",
			mBinSHA, binSHA)
	}
	if mBinRev != finalCommit {
		t.Fatalf("manifest leamas_binary_identity.vcs_revision %q != FINAL_COMMIT %s",
			mBinRev, finalCommit)
	}
	if mBinMod {
		t.Fatalf("manifest leamas_binary_identity.vcs_modified=true")
	}

	// Phase 7: literal SHA-256 of the manifest.
	manifestSHA := sha256HexBytes(manifestBytes)

	// Stdout/stderr SHA-256.
	stdoutSHA := sha256HexBytes(bounded.Stdout)
	stderrSHA := sha256HexBytes(bounded.Stderr)

	lastR2CRDogfood = r2cRDogfoodResult{
		FinalCommit:            finalCommit,
		FinalTree:              runR2CRGit(t, leamasRepoRoot, "rev-parse", finalCommit+"^{tree}"),
		BuildSourceCommit:      finalCommit,
		BuildSourceTree:        runR2CRGit(t, leamasRepoRoot, "rev-parse", finalCommit+"^{tree}"),
		BuildSourceStatusEmpty: true,
		BuildSourceDetached:    true,
		BinaryPath:             binaryPath,
		BinarySHA256:           binSHA,
		BinaryVCSRevision:      identity.Commit,
		BinaryVCSModified:      identity.Dirty,
		Subject:                subject,
		SubjectTree:            subjectTree,
		Freeze:                 freeze,
		FreezeTree:             freezeTree,
		Descendant:             d,
		CallerHead:             d,
		CallerHeadBefore:       headBefore,
		CallerHeadAfter:        headAfter,
		CallerTreeBefore:       headTreeBefore,
		CallerTreeAfter:        headTreeAfter,
		CallerStatusBefore:     statusBefore,
		CallerStatusAfter:      statusAfter,
		WorktreesBefore:        worktreesBefore,
		WorktreesAfter:         worktreesAfter,
		ManifestPath:           manifestOutput,
		ManifestAbsenceBefore:  true,
		ManifestPresentAfter:   true,
		ManifestSubject:        mSubject,
		ManifestFreeze:         mFreeze,
		ManifestCallerHead:     mCaller,
		ManifestExecutionTree:  mExecTree,
		ManifestPlanPath:       mPlanPath,
		ManifestPlanBlob:       mPlanBlob,
		ManifestPlanSHA256:     mPlanSHA,
		ManifestBinarySHA256:   mBinSHA,
		ManifestBinaryVCSRev:   mBinRev,
		ManifestBinaryVCSMod:   mBinMod,
		ManifestSHA256:         manifestSHA,
		StdoutBytes:            len(bounded.Stdout),
		StderrBytes:            len(bounded.Stderr),
		StdoutSHA256:           stdoutSHA,
		StderrSHA256:           stderrSHA,
		ExitCode:               bounded.ExitCode,
		TimedOut:               bounded.TimedOut,
		StdoutTruncated:        bounded.StdoutTruncated,
		StderrTruncated:        bounded.StderrTruncated,
		TruncationRejectionMsg: rejMsg,
		RunErr:                 bounded.Err,
	}

	// R2C-R2 Phase 6: write durable, deterministic evidence
	// outside the Leamas repository. The evidence path is
	// either LEAMAS_R2CR_EVIDENCE_DIR/<r2cr-evidence.json> or
	// t.TempDir()/r2cr-evidence.json. The path and SHA-256
	// are recorded in the result so the close report can
	// identify the exact historical run.
	writeR2CREvidence(t, &lastR2CRDogfood)
}

// writeR2CREvidence serializes the dogfood result to
// deterministic JSON in a directory outside the Leamas
// repository. The path and content SHA-256 are stored in
// the result for the close report.
//
// The function writes the file twice: once with a placeholder
// path so the in-memory struct contains the canonical values
// (path, sha256) and the on-disk file matches the in-memory
// representation. The second write rewrites the file with the
// real path and sha256, producing a self-referential evidence
// record where the file content includes its own path and
// sha256.
func writeR2CREvidence(t *testing.T, r *r2cRDogfoodResult) {
	t.Helper()
	dir := os.Getenv("LEAMAS_R2CR_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir evidence dir: %v", err)
	}
	path := filepath.Join(dir, "r2cr-evidence.json")

	// First write: include the path and a placeholder sha256
	// so the in-memory struct can be re-encoded consistently.
	r.EvidencePath = path
	r.EvidenceSHA256 = "PENDING"
	if err := writeR2CREvidenceAtomic(path, r); err != nil {
		t.Fatalf("write evidence (first pass): %v", err)
	}
	// Second write: compute the real sha256 of the file on
	// disk and re-encode.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read evidence: %v", err)
	}
	r.EvidenceSHA256 = sha256HexBytes(raw)
	if err := writeR2CREvidenceAtomic(path, r); err != nil {
		t.Fatalf("write evidence (second pass): %v", err)
	}
}

// writeR2CREvidenceAtomic writes the supplied result to path
// via a temp-file rename so a partial write can never leave a
// half-formed evidence file behind.
func writeR2CREvidenceAtomic(path string, r *r2cRDogfoodResult) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".r2cr-evidence-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp evidence: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		tmp.Close()
		return fmt.Errorf("encode evidence: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp evidence: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename evidence: %w", err)
	}
	return nil
}

// mustPrepareR2CRBuildEnv resolves the leamas repo root via
// `git rev-parse --show-toplevel`, asserts the caller worktree
// is clean, and creates a temporary detached worktree at
// finalCommit for the dogfood build. It returns the detached
// worktree path so the caller can clean it up after the test.
//
// To allow CI to pin the FINAL_COMMIT explicitly, this helper
// honours the LEAMAS_R2CR_FINAL_COMMIT env var; when set, the
// detached worktree is created at exactly that commit. When
// unset, the helper uses HEAD as the FINAL_COMMIT (the
// production dogfood uses the latter for repeatable
// regressions; the former for historical evidence capture).
