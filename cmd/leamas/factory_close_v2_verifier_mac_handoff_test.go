// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_mac_handoff_test.go implements
// the installed-style final dogfood required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01.
//
// The test:
//
//  1. builds the leamas binary in a TEMPORARY DETACHED
//     WORKTREE at the current HEAD so the build source bytes
//     are exact and immutable;
//  2. constructs a meaningful hermetic S < F < C repository
//     where C carries a real committed v2 manifest M produced
//     by the same binary via `factory close run-v2-authority`;
//  3. invokes `factory close verify-v2-authority` from a temp
//     working directory OUTSIDE both the Leamas checkout and
//     the hermetic repository;
//  4. asserts the verifier reported valid=true, that the
//     literal plan/manifest blob and SHA-256 bindings are
//     exact, that the target repository state is byte-for-byte
//     unchanged, and that no worktree leaked;
//  5. writes literal, deterministic evidence outside the
//     Leamas checkout with a detached SHA-256 sidecar.
//
// The test does NOT introduce new verifier architecture. It
// reuses the build / git / SHA-256 helpers from
// factory_close_v2_r2c_helpers_test.go and the bounded
// subprocess harness from bounded_subprocess_v2_test.go.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// macHandoffPlanPath is the repository-relative path used by
// the dogfood for the frozen plan P. The path is
// canonically under docs/closure-plans/ so the closure-plan
// doctype does not drift between the fixture and the
// committed manifest.
const macHandoffPlanPath = "docs/closure-plans/MAC-HANDOFF.json"

// macHandoffManifestPath is the repository-relative path
// used by the dogfood for the committed manifest M at C.
// The path is canonically under docs/closure-manifests/ so
// the committed manifest doctype does not drift.
const macHandoffManifestPath = "docs/closure-manifests/MAC-HANDOFF.json"

// macHandoffCheckShell is a POSIX sh -c invocation that
// proves the executor ran against S^{tree} (not F^{tree}
// or C^{tree}):
//
//	test -f subject-only.txt   (S tree contains S's file)
//	test ! -e freeze-only.txt  (S tree does NOT contain F-only)
//	test ! -e closure-only.txt (S tree does NOT contain C-only)
const macHandoffCheckShell = "test -f subject-only.txt && test ! -e freeze-only.txt && test ! -e closure-only.txt"

// TestClosureCLIV2VerifierMacHandoff is the installed-style
// final dogfood for the v2 closure verifier. It is the
// Linux-side proof that the public `verify-v2-authority`
// command succeeds end-to-end against a meaningful hermetic
// S < F < C repository when invoked from outside the Leamas
// checkout, and that no caller-state mutation is observable.
func TestClosureCLIV2VerifierMacHandoff(t *testing.T) {
	finalCommit, finalTree, leamasRepoRoot, detached := mustPrepareR2CRBuildEnv(t)
	defer func() {
		if detached != "" {
			runR2CRGit(t, leamasRepoRoot, "worktree", "remove", "--force", detached)
		}
	}()

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

	repository := initMacHandoffRepo(t)
	subject, subjectTree, freeze, freezeTree := buildMacHandoffSF(t, repository, leamasRepoRoot)

	// Run the production v2 runner to produce manifest M.
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
		"--plan-path", macHandoffPlanPath,
		"--evidence-directory", evidenceDir,
		"--manifest-output", manifestOutput,
	}, boundedSubprocessV2Options{
		Timeout:   60 * time.Second,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
	})
	if runnerResult.ExitCode != 0 {
		t.Fatalf("v2 runner exit %d: stderr=%s",
			runnerResult.ExitCode, string(runnerResult.Stderr))
	}
	manifestBytes, err := os.ReadFile(manifestOutput)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !json.Valid(manifestBytes) {
		t.Fatalf("runner manifest is not valid JSON: %s", string(manifestBytes))
	}

	// Commit the manifest as C on top of F.
	manifestDir := filepath.Join(repository, filepath.Dir(macHandoffManifestPath))
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	mustWriteFile(t, filepath.Join(repository, macHandoffManifestPath), string(manifestBytes))
	mustWriteFile(t, filepath.Join(repository, "closure-only.txt"), "closure-only\n")
	runR2CRGit(t, repository, "add", ".")
	runR2CRGit(t, repository, "commit", "-m", "factory: close v2 verifier mac handoff ACT")
	closure := runR2CRGit(t, repository, "rev-parse", "HEAD")
	closureTree := runR2CRGit(t, repository, "rev-parse", closure+"^{tree}")

	planBlobOID := runR2CRGit(t, repository, "rev-parse", freeze+":"+macHandoffPlanPath)
	planRaw := runR2CRGitRaw(t, repository, planBlobOID)
	planSHA := sha256HexBytes(planRaw)
	manifestBlobOID := runR2CRGit(t, repository, "rev-parse", closure+":"+macHandoffManifestPath)
	manifestRaw := runR2CRGitRaw(t, repository, manifestBlobOID)
	manifestSHA := sha256HexBytes(manifestRaw)

	// Invoke the verifier from a temp CWD outside both repos.
	callerBefore := captureMacHandoffCallerState(t, repository)
	verifierCWD := t.TempDir()
	verifierOutputPath := filepath.Join(verifierCWD, "verification.json")
	verifierResult := boundedSubprocessV2(binaryPath, []string{
		"factory", "close", "verify-v2-authority",
		"--protocol-version", "2",
		"--plan-contract-version", "1",
		"--repository", repository,
		"--subject", subject,
		"--freeze", freeze,
		"--closure", closure,
		"--plan-path", macHandoffPlanPath,
		"--manifest-path", macHandoffManifestPath,
		"--json",
		"--output", verifierOutputPath,
	}, boundedSubprocessV2Options{
		Timeout:   60 * time.Second,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
		WorkDir:   verifierCWD,
	})

	// Caller state must be byte-for-byte unchanged.
	callerAfter := captureMacHandoffCallerState(t, repository)
	if callerAfter.statusSHA != callerBefore.statusSHA {
		t.Fatalf("caller porcelain-v2 status drifted")
	}
	if callerAfter.worktreeSHA != callerBefore.worktreeSHA {
		t.Fatalf("caller worktree inventory drifted")
	}
	if callerAfter.refsSHA != callerBefore.refsSHA {
		t.Fatalf("caller refs snapshot drifted")
	}

	if verifierResult.ExitCode != 0 {
		t.Fatalf("verifier exit %d: stdout=%s stderr=%s",
			verifierResult.ExitCode,
			string(verifierResult.Stdout),
			string(verifierResult.Stderr))
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"ok\": true")) {
		t.Fatalf("verifier did not report ok=true: stdout=%q",
			string(verifierResult.Stdout))
	}

	verifierOutput, err := os.ReadFile(verifierOutputPath)
	if err != nil {
		t.Fatalf("read verifier output: %v", err)
	}
	var envelope struct {
		OK           bool            `json:"ok"`
		Verification json.RawMessage `json:"verification"`
	}
	if err := json.Unmarshal(verifierOutput, &envelope); err != nil {
		t.Fatalf("decode verifier envelope: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("verifier envelope OK=false")
	}

	result := macHandoffDogfoodResult{
		FinalCommit:           finalCommit,
		FinalTree:             finalTree,
		BinaryPath:            binaryPath,
		BinarySHA256:          binSHA,
		BinaryVCSRevision:     identity.Commit,
		BinaryVCSModified:     identity.Dirty,
		Subject:               subject,
		SubjectTree:           subjectTree,
		Freeze:                freeze,
		FreezeTree:            freezeTree,
		Closure:               closure,
		ClosureTree:           closureTree,
		PlanPath:              macHandoffPlanPath,
		PlanBlob:              planBlobOID,
		PlanSHA256:            planSHA,
		ManifestPath:          macHandoffManifestPath,
		ManifestBlob:          manifestBlobOID,
		ManifestSHA256:        manifestSHA,
		CallerStatusBeforeSHA: callerBefore.statusSHA,
		CallerStatusAfterSHA:  callerAfter.statusSHA,
		WorktreesBeforeSHA:    callerBefore.worktreeSHA,
		WorktreesAfterSHA:     callerAfter.worktreeSHA,
		RefsBeforeSHA:         callerBefore.refsSHA,
		RefsAfterSHA:          callerAfter.refsSHA,
		StdoutSHA256:          sha256HexBytes(verifierResult.Stdout),
		StderrSHA256:          sha256HexBytes(verifierResult.Stderr),
		ExitCode:              verifierResult.ExitCode,
		VerifierOutputPath:    verifierOutputPath,
		VerifierOutputSHA256:  sha256HexBytes(verifierOutput),
	}
	lastMacHandoffDogfood = result
	writeMacHandoffEvidence(t, &result)
}
