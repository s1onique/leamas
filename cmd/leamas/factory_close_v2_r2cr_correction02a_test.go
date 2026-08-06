// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_r2cr_correction02a_test.go implements
// the build-and-subprocess evidence dogfood for
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02A.
//
// The test:
//
//  1. builds the leamas binary in a TEMPORARY DETACHED
//     WORKTREE at the current HEAD, with the binary
//     output written OUTSIDE the worktree;
//  2. observes (not fabricates) the detached build source
//     HEAD, tree, detached state, and porcelain-v2 status
//     before and after the build;
//  3. asserts the build source status was clean before
//     and after the build;
//  4. asserts the binary's VCS revision literally equals
//     the final commit;
//  5. runs the v2 runner with full bounded-subprocess
//     assertion (ExitCode, TimedOut, StdoutTruncated,
//     StderrTruncated, Err);
//  6. runs the v2 verifier with full bounded-subprocess
//     assertion;
//  7. decodes the verifier JSON envelope into typed
//     structures and asserts every S/F/C/P/M identity;
//  8. writes validated, atomically published evidence
//     with a coordinated SHA-256 sidecar.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/factory/closure"
)

const correction02aPlanPath = "docs/closure-plans/CORRECTION02A.json"
const correction02aManifestPath = "docs/closure-manifests/CORRECTION02A.json"
const correction02aCheckShell = "test -f subject-only.txt && test ! -e freeze-only.txt && test ! -e closure-only.txt"

func TestClosureCLIV2VerifierMacHandoffCorrection02A(t *testing.T) {
	// 1. Resolve the leamas repo root and the final
	//    commit; create the detached worktree.
	finalCommit, finalTree, leamasRepoRoot, detached := mustPrepareR2CRBuildEnv(t)
	defer func() {
		if detached != "" {
			runR2CRGit(t, leamasRepoRoot, "worktree", "remove", "--force", detached)
		}
	}()

	// 2. Observe the build source BEFORE the build.
	beforeSnap := captureCorrection02aBuildSourceSnapshot(t, detached)
	if !beforeSnap.IsEmpty {
		t.Fatalf("detached build source dirty BEFORE build:\n%s", beforeSnap.PorcelainV2)
	}
	if !beforeSnap.Detached {
		t.Fatalf("detached worktree HEAD is not detached before build")
	}
	if beforeSnap.HeadCommit != finalCommit {
		t.Fatalf("detached HEAD before build = %s, want %s",
			beforeSnap.HeadCommit, finalCommit)
	}

	// 3. Build the binary OUTSIDE the worktree.
	outputDir := filepath.Join(filepath.Dir(detached), "correction02a-binary")
	binaryPath := buildInDetachedWorktreeTo(t, detached, outputDir, finalCommit)

	// 4. Observe the build source AFTER the build.
	afterSnap := captureCorrection02aBuildSourceSnapshot(t, detached)
	if !afterSnap.IsEmpty {
		t.Fatalf("detached build source dirty AFTER build:\n%s", afterSnap.PorcelainV2)
	}
	if !afterSnap.Detached {
		t.Fatalf("detached worktree HEAD is not detached after build")
	}

	// 5. Confirm the binary is outside the worktree.
	absBinary, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("abs binary: %v", err)
	}
	absDetached, err := filepath.Abs(detached)
	if err != nil {
		t.Fatalf("abs detached: %v", err)
	}
	if isPathInsideDir(absDetached, absBinary) {
		t.Fatalf("binary %s is inside detached worktree %s", absBinary, absDetached)
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

	// 6. Build the hermetic S < F < C repository.
	repository := initCorrection01Repo(t)
	subject, subjectTree, freeze, freezeTree := buildSFWithPlan(t, repository, leamasRepoRoot, correction02aPlanPath)

	// 7. Run the production v2 runner with full bounded
	//    subprocess assertion.
	detachedDir := t.TempDir()
	evidenceDir := filepath.Join(detachedDir, "evidence")
	manifestOutput := filepath.Join(detachedDir, "manifest.json")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatalf("mkdir evidence: %v", err)
	}
	runnerResult := boundedSubprocessV2(binaryPath, []string{
		"factory", "close", "run-v2-authority",
		"--protocol-version", "2",
		"--plan-contract-version", "1",
		"--repository", repository,
		"--subject", subject,
		"--freeze", freeze,
		"--plan-path", correction02aPlanPath,
		"--evidence-directory", evidenceDir,
		"--manifest-output", manifestOutput,
	}, boundedSubprocessV2Options{
		Timeout:   60 * time.Second,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
	})
	assertCorrection02aBounded(t, "runner", runnerResult)
	manifestBytes, err := os.ReadFile(manifestOutput)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !json.Valid(manifestBytes) {
		t.Fatalf("runner manifest is not valid JSON: %s", string(manifestBytes))
	}

	// 8. Commit the manifest as C on top of F.
	manifestDir := filepath.Join(repository, filepath.Dir(correction02aManifestPath))
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	mustWriteFile(t, filepath.Join(repository, correction02aManifestPath), string(manifestBytes))
	mustWriteFile(t, filepath.Join(repository, "closure-only.txt"), "closure-only\n")
	runR2CRGit(t, repository, "add", ".")
	runR2CRGit(t, repository, "commit", "-m", "factory: close v2 verifier mac handoff correction02a ACT")
	closureCommit := runR2CRGit(t, repository, "rev-parse", "HEAD")
	closureTree := runR2CRGit(t, repository, "rev-parse", closureCommit+"^{tree}")

	planBlobOID := runR2CRGit(t, repository, "rev-parse", freeze+":"+correction02aPlanPath)
	planRaw := runR2CRGitRaw(t, repository, planBlobOID)
	planSHA := sha256HexBytes(planRaw)
	manifestBlobOID := runR2CRGit(t, repository, "rev-parse", closureCommit+":"+correction02aManifestPath)
	manifestRaw := runR2CRGitRaw(t, repository, manifestBlobOID)
	manifestSHA := sha256HexBytes(manifestRaw)

	// 9. Invoke the verifier with full bounded subprocess
	//    assertion. Capture caller state before.
	callerBefore := captureCorrection01CallerState(t, repository)
	verifierCWD := t.TempDir()
	verifierOutputPath := filepath.Join(verifierCWD, "verification.json")
	verifierResult := boundedSubprocessV2(binaryPath, []string{
		"factory", "close", "verify-v2-authority",
		"--protocol-version", "2",
		"--plan-contract-version", "1",
		"--repository", repository,
		"--subject", subject,
		"--freeze", freeze,
		"--closure", closureCommit,
		"--plan-path", correction02aPlanPath,
		"--manifest-path", correction02aManifestPath,
		"--json",
		"--output", verifierOutputPath,
	}, boundedSubprocessV2Options{
		Timeout:   60 * time.Second,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
		WorkDir:   verifierCWD,
	})
	assertCorrection02aBounded(t, "verifier", verifierResult)
	callerAfter := captureCorrection01CallerState(t, repository)
	if callerAfter.statusSHA != callerBefore.statusSHA {
		t.Fatalf("caller status drifted")
	}
	if callerAfter.worktreeSHA != callerBefore.worktreeSHA {
		t.Fatalf("caller worktree drifted")
	}
	if callerAfter.refsSHA != callerBefore.refsSHA {
		t.Fatalf("caller refs drifted")
	}

	// 10. Decode the verifier envelope into typed
	//     structures.
	verifierOutput, err := os.ReadFile(verifierOutputPath)
	if err != nil {
		t.Fatalf("read verifier output: %v", err)
	}
	envelope, err := decodeCorrection01Envelope(verifierOutput)
	if err != nil {
		t.Fatalf("decode verifier envelope: %v", err)
	}
	if !envelope.OK || !envelope.Verification.Valid ||
		!envelope.Verification.TopologyValid ||
		!envelope.Verification.ManifestValid ||
		!envelope.Verification.ResultSetValid ||
		len(envelope.Verification.Diagnostics) != 0 {
		t.Fatalf("verifier envelope rejected: %+v", envelope)
	}
	v := envelope.Verification
	if v.SubjectCommit != subject || v.SubjectTree != subjectTree ||
		v.FreezeCommit != freeze || v.FreezeTree != freezeTree ||
		v.ClosureCommit != closureCommit || v.ClosureTree != closureTree ||
		v.PlanBlob != planBlobOID || v.PlanSHA256 != planSHA ||
		v.ManifestBlob != manifestBlobOID || v.ManifestSHA256 != manifestSHA ||
		v.ClosureProtocolVersion != "2" ||
		v.PlanContractVersion != closure.PlanContractVersion(1) {
		t.Fatalf("verifier S/F/C/P/M bindings do not match dogfood: %+v", v)
	}

	// 11. Validate and publish the evidence.
	result := correction02aDogfoodResult{
		ACTID:                      "ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02A",
		Status:                     "PASS",
		BaseCommit:                 "c62d110c01ab3d4786a53ef8fea1d11b4c8e80e8",
		BaseTree:                   "5a463325fdbd6532a6b3df1507e7de385c67c524",
		FinalCommit:                finalCommit,
		FinalTree:                  finalTree,
		BuildSourceHeadBefore:      beforeSnap.HeadCommit,
		BuildSourceHeadAfter:       afterSnap.HeadCommit,
		BuildSourceTreeBefore:      beforeSnap.HeadTree,
		BuildSourceTreeAfter:       afterSnap.HeadTree,
		BuildSourceDetachedBefore:  beforeSnap.Detached,
		BuildSourceDetachedAfter:   afterSnap.Detached,
		BuildSourceStatusBeforeSHA: beforeSnap.PorcelainV2SHA,
		BuildSourceStatusAfterSHA:  afterSnap.PorcelainV2SHA,
		BuildOutputPath:            outputDir,
		BuildOutputOutsideSource:   !isPathInsideDir(absDetached, absBinary),
		BinaryPath:                 binaryPath,
		BinarySHA256:               binSHA,
		BinaryVCSRevision:          identity.Commit,
		BinaryVCSModified:          identity.Dirty,
		RunnerResult: correction02aSubprocessResult{
			ExitCode:        runnerResult.ExitCode,
			TimedOut:        runnerResult.TimedOut,
			StdoutTruncated: runnerResult.StdoutTruncated,
			StderrTruncated: runnerResult.StderrTruncated,
			StdoutSHA256:    sha256HexBytes(runnerResult.Stdout),
			StderrSHA256:    sha256HexBytes(runnerResult.Stderr),
			ErrorPresent:    runnerResult.Err != nil,
			ErrorText:       runErrorText(runnerResult.Err),
		},
		VerifierResult: correction02aSubprocessResult{
			ExitCode:        verifierResult.ExitCode,
			TimedOut:        verifierResult.TimedOut,
			StdoutTruncated: verifierResult.StdoutTruncated,
			StderrTruncated: verifierResult.StderrTruncated,
			StdoutSHA256:    sha256HexBytes(verifierResult.Stdout),
			StderrSHA256:    sha256HexBytes(verifierResult.Stderr),
			ErrorPresent:    verifierResult.Err != nil,
			ErrorText:       runErrorText(verifierResult.Err),
		},
		DogfoodSubject:             subject,
		DogfoodSubjectTree:         subjectTree,
		DogfoodFreeze:              freeze,
		DogfoodFreezeTree:          freezeTree,
		DogfoodClosure:             closureCommit,
		DogfoodClosureTree:         closureTree,
		DogfoodPlanPath:            correction02aPlanPath,
		DogfoodPlanBlob:            planBlobOID,
		DogfoodPlanSHA256:          planSHA,
		DogfoodManifestPath:        correction02aManifestPath,
		DogfoodManifestBlob:        manifestBlobOID,
		DogfoodManifestSHA256:      manifestSHA,
		CallerHeadBefore:           callerBefore.headCommit,
		CallerHeadAfter:            callerAfter.headCommit,
		CallerTreeBefore:           callerBefore.headTree,
		CallerTreeAfter:            callerAfter.headTree,
		CallerStatusBeforeSHA:      callerBefore.statusSHA,
		CallerStatusAfterSHA:       callerAfter.statusSHA,
		WorktreeInventoryBeforeSHA: callerBefore.worktreeSHA,
		WorktreeInventoryAfterSHA:  callerAfter.worktreeSHA,
		RefsBeforeSHA:              callerBefore.refsSHA,
		RefsAfterSHA:               callerAfter.refsSHA,
	}
	lastCorrection02aDogfood = result
	writeCorrection02aEvidence(t, &result)
}

// assertCorrection02aBounded asserts the bounded
// subprocess result has the required clean shape: zero
// exit, no timeout, no truncation, no error.
func assertCorrection02aBounded(t *testing.T, label string, r boundedSubprocessV2Result) {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("%s exit %d: stdout=%s stderr=%s",
			label, r.ExitCode, string(r.Stdout), string(r.Stderr))
	}
	if r.TimedOut {
		t.Fatalf("%s timed out", label)
	}
	if r.StdoutTruncated {
		t.Fatalf("%s stdout truncated", label)
	}
	if r.StderrTruncated {
		t.Fatalf("%s stderr truncated", label)
	}
	if r.Err != nil {
		t.Fatalf("%s error: %v", label, r.Err)
	}
}

// isPathInsideDir reports whether path is inside the
// directory parent. Both arguments are expected to be
// absolute and Clean()-ed.
func isPathInsideDir(parent, path string) bool {
	if parent == "" || path == "" {
		return false
	}
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || (len(rel) >= 3 && rel[:3] == "../") {
		return false
	}
	return true
}
