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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Phase 2: detached exact build. The binary's VCS
	// revision must equal FINAL_COMMIT and the dirty flag
	// must be false so the manifest's leamas_binary_identity
	// matches the dogfood binary.
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

	// Phase 3: build S and F in a fresh hermetic repository.
	repository := initMacHandoffRepo(t)
	subject, subjectTree, freeze, freezeTree := buildMacHandoffSF(t, repository, leamasRepoRoot)

	// Phase 3a: run the production v2 runner (same binary)
	// to produce a manifest M. The runner is invoked via the
	// bounded harness; the manifest lands in a detached path
	// outside the repository.
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
	if runnerResult.TimedOut {
		t.Fatalf("v2 runner timed out")
	}
	if runnerResult.StdoutTruncated || runnerResult.StderrTruncated {
		t.Fatalf("v2 runner output was truncated")
	}
	manifestBytes, err := os.ReadFile(manifestOutput)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(manifestBytes) == 0 {
		t.Fatalf("runner manifest is empty")
	}
	if !json.Valid(manifestBytes) {
		t.Fatalf("runner manifest is not valid JSON: %s", string(manifestBytes))
	}

	// Phase 3b: commit the manifest as C. The commit lives
	// on top of F in the hermetic repository. The commit
	// message names the act so the verifier can be
	// reproduced by the same act_id.
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

	// Independent resolution of plan and manifest blob OIDs
	// from C, plus raw byte reads so the verifier-side
	// binding can be compared against literal C:M / F:P
	// bytes.
	planBlobOID := runR2CRGit(t, repository, "rev-parse", freeze+":"+macHandoffPlanPath)
	planRaw := runR2CRGitRaw(t, repository, planBlobOID)
	planSHA := sha256HexBytes(planRaw)
	manifestBlobOID := runR2CRGit(t, repository, "rev-parse", closure+":"+macHandoffManifestPath)
	manifestRaw := runR2CRGitRaw(t, repository, manifestBlobOID)
	manifestSHA := sha256HexBytes(manifestRaw)
	if manifestSHA != sha256HexBytes(manifestBytes) {
		t.Fatalf("committed manifest SHA %s != runner output SHA %s",
			manifestSHA, sha256HexBytes(manifestBytes))
	}

	// Phase 4: capture caller state BEFORE, then invoke the
	// verifier from a temp working directory OUTSIDE the
	// Leamas checkout and OUTSIDE the hermetic repository.
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

	// Phase 5: independent state proof. The caller state
	// must be byte-for-byte unchanged.
	callerAfter := captureMacHandoffCallerState(t, repository)
	if callerAfter.headCommit != callerBefore.headCommit {
		t.Fatalf("caller HEAD drifted: before=%s after=%s",
			callerBefore.headCommit, callerAfter.headCommit)
	}
	if callerAfter.headTree != callerBefore.headTree {
		t.Fatalf("caller HEAD tree drifted: before=%s after=%s",
			callerBefore.headTree, callerAfter.headTree)
	}
	if callerAfter.statusSHA != callerBefore.statusSHA {
		t.Fatalf("caller porcelain-v2 status drifted: before=%s after=%s",
			callerBefore.statusSHA, callerAfter.statusSHA)
	}
	if callerAfter.worktreeSHA != callerBefore.worktreeSHA {
		t.Fatalf("caller worktree inventory drifted: before=%s after=%s",
			callerBefore.worktreeSHA, callerAfter.worktreeSHA)
	}
	if callerAfter.refsSHA != callerBefore.refsSHA {
		t.Fatalf("caller refs snapshot drifted: before=%s after=%s",
			callerBefore.refsSHA, callerAfter.refsSHA)
	}

	// Phase 4: verifier output assertions. The CLI is
	// invoked with --json so stdout is a single JSON
	// envelope; the text-summary keys are not on stdout.
	// The non-zero exit and the typed-diagnostics surface
	// remain the primary failure signal.
	if verifierResult.ExitCode != 0 {
		t.Fatalf("verifier exit %d: stdout=%s stderr=%s",
			verifierResult.ExitCode,
			string(verifierResult.Stdout),
			string(verifierResult.Stderr))
	}
	if verifierResult.TimedOut {
		t.Fatalf("verifier timed out")
	}
	if verifierResult.StdoutTruncated || verifierResult.StderrTruncated {
		t.Fatalf("verifier output was truncated")
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"ok\": true")) {
		t.Fatalf("verifier did not report ok=true: stdout=%q",
			string(verifierResult.Stdout))
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"valid\": true")) {
		t.Fatalf("verifier did not report valid=true: stdout=%q",
			string(verifierResult.Stdout))
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"subject_commit\": \""+subject+"\"")) {
		t.Fatalf("verifier stdout missing subject_commit=%s: %q",
			subject, string(verifierResult.Stdout))
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"freeze_commit\": \""+freeze+"\"")) {
		t.Fatalf("verifier stdout missing freeze_commit=%s: %q",
			freeze, string(verifierResult.Stdout))
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"closure_commit\": \""+closure+"\"")) {
		t.Fatalf("verifier stdout missing closure_commit=%s: %q",
			closure, string(verifierResult.Stdout))
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"manifest_sha256\": \""+manifestSHA+"\"")) {
		t.Fatalf("verifier stdout missing manifest_sha256=%s: %q",
			manifestSHA, string(verifierResult.Stdout))
	}
	if !bytes.Contains(verifierResult.Stdout, []byte("\"plan_sha256\": \""+planSHA+"\"")) {
		t.Fatalf("verifier stdout missing plan_sha256=%s: %q",
			planSHA, string(verifierResult.Stdout))
	}

	// Phase 4: --output file must be a single JSON document
	// on disk and must round-trip to a verification result
	// whose subject/freeze/closure/blobs/sha256 fields all
	// match the expected literal values.
	verifierOutput, err := os.ReadFile(verifierOutputPath)
	if err != nil {
		t.Fatalf("read verifier output: %v", err)
	}
	if !json.Valid(verifierOutput) {
		t.Fatalf("verifier output not valid JSON: %s", string(verifierOutput))
	}
	var envelope struct {
		OK           bool            `json:"ok"`
		Verification json.RawMessage `json:"verification"`
	}
	if err := json.Unmarshal(verifierOutput, &envelope); err != nil {
		t.Fatalf("decode verifier envelope: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("verifier envelope OK=false: %s", string(verifierOutput))
	}
	var v struct {
		SubjectCommit  string `json:"subject_commit"`
		FreezeCommit   string `json:"freeze_commit"`
		ClosureCommit  string `json:"closure_commit"`
		PlanBlob       string `json:"plan_blob"`
		PlanSHA256     string `json:"plan_sha256"`
		ManifestBlob   string `json:"manifest_blob"`
		ManifestSHA256 string `json:"manifest_sha256"`
		Valid          bool   `json:"valid"`
	}
	if err := json.Unmarshal(envelope.Verification, &v); err != nil {
		t.Fatalf("decode verification: %v", err)
	}
	if v.SubjectCommit != subject {
		t.Fatalf("verification subject_commit %q != %q",
			v.SubjectCommit, subject)
	}
	if v.FreezeCommit != freeze {
		t.Fatalf("verification freeze_commit %q != %q",
			v.FreezeCommit, freeze)
	}
	if v.ClosureCommit != closure {
		t.Fatalf("verification closure_commit %q != %q",
			v.ClosureCommit, closure)
	}
	if v.PlanBlob != planBlobOID {
		t.Fatalf("verification plan_blob %q != %q",
			v.PlanBlob, planBlobOID)
	}
	if v.PlanSHA256 != planSHA {
		t.Fatalf("verification plan_sha256 %q != %q",
			v.PlanSHA256, planSHA)
	}
	if v.ManifestBlob != manifestBlobOID {
		t.Fatalf("verification manifest_blob %q != %q",
			v.ManifestBlob, manifestBlobOID)
	}
	if v.ManifestSHA256 != manifestSHA {
		t.Fatalf("verification manifest_sha256 %q != %q",
			v.ManifestSHA256, manifestSHA)
	}
	if !v.Valid {
		t.Fatalf("verification Valid=false: %s", string(verifierOutput))
	}

	// Phase 6: literal evidence with detached sidecar.
	result := macHandoffDogfoodResult{
		FinalCommit:            finalCommit,
		FinalTree:              finalTree,
		BuildSourceCommit:      finalCommit,
		BuildSourceTree:        finalTree,
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
		Closure:                closure,
		ClosureTree:            closureTree,
		PlanPath:               macHandoffPlanPath,
		PlanBlob:               planBlobOID,
		PlanSHA256:             planSHA,
		ManifestPath:           macHandoffManifestPath,
		ManifestBlob:           manifestBlobOID,
		ManifestSHA256:         manifestSHA,
		CallerHeadBefore:       callerBefore.headCommit,
		CallerHeadAfter:        callerAfter.headCommit,
		CallerTreeBefore:       callerBefore.headTree,
		CallerTreeAfter:        callerAfter.headTree,
		CallerStatusBeforeSHA:  callerBefore.statusSHA,
		CallerStatusAfterSHA:   callerAfter.statusSHA,
		WorktreesBeforeSHA:     callerBefore.worktreeSHA,
		WorktreesAfterSHA:      callerAfter.worktreeSHA,
		RefsBeforeSHA:          callerBefore.refsSHA,
		RefsAfterSHA:           callerAfter.refsSHA,
		StdoutBytes:            len(verifierResult.Stdout),
		StderrBytes:            len(verifierResult.Stderr),
		StdoutSHA256:           sha256HexBytes(verifierResult.Stdout),
		StderrSHA256:           sha256HexBytes(verifierResult.Stderr),
		ExitCode:               verifierResult.ExitCode,
		TimedOut:               verifierResult.TimedOut,
		StdoutTruncated:        verifierResult.StdoutTruncated,
		StderrTruncated:        verifierResult.StderrTruncated,
		RunErr:                 verifierResult.Err,
		VerifierOutputPath:     verifierOutputPath,
		VerifierOutputSHA256:   sha256HexBytes(verifierOutput),
	}
	lastMacHandoffDogfood = result
	writeMacHandoffEvidence(t, &result)
}

// initMacHandoffRepo creates a fresh temp directory and
// initialises a Git repository with an empty initial
// commit. The returned path is the repository root.
func initMacHandoffRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runR2CRGit(t, repo, "init", "-b", "main")
	runR2CRGit(t, repo, "config", "user.name", "Mac Handoff Dogfood")
	runR2CRGit(t, repo, "config", "user.email", "mac-handoff@example.invalid")
	runR2CRGit(t, repo, "config", "commit.gpgsign", "false")
	runR2CRGit(t, repo, "config", "tag.gpgsign", "false")
	runR2CRGit(t, repo, "commit", "--allow-empty", "-m", "initial")
	return repo
}

// buildMacHandoffSF creates the S and F commits in the
// supplied repository. S contains only the subject-only
// file; F is a child of S that adds the frozen plan P and
// a freeze-only file. The function returns the four
// canonical OIDs the test asserts against.
func buildMacHandoffSF(t *testing.T, repo, _ string) (subject, subjectTree, freeze, freezeTree string) {
	t.Helper()

	// S: subject-only file present.
	mustWriteFile(t, filepath.Join(repo, "subject-only.txt"), "subject\n")
	runR2CRGit(t, repo, "add", "subject-only.txt")
	runR2CRGit(t, repo, "commit", "-m", "subject")
	subject = runR2CRGit(t, repo, "rev-parse", "HEAD")
	subjectTree = runR2CRGit(t, repo, "rev-parse", subject+"^{tree}")

	// F: plan P + freeze-only file.
	planDir := filepath.Join(repo, filepath.Dir(macHandoffPlanPath))
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	planJSON := buildMacHandoffPlan(subject, subjectTree)
	mustWriteFile(t, filepath.Join(repo, macHandoffPlanPath), planJSON)
	mustWriteFile(t, filepath.Join(repo, "freeze-only.txt"), "freeze-only\n")
	runR2CRGit(t, repo, "add", ".")
	runR2CRGit(t, repo, "commit", "-m", "freeze: add plan")
	freeze = runR2CRGit(t, repo, "rev-parse", "HEAD")
	freezeTree = runR2CRGit(t, repo, "rev-parse", freeze+"^{tree}")
	return subject, subjectTree, freeze, freezeTree
}

// buildMacHandoffPlan returns a contract-valid Plan
// Contract v1 document whose run-mode check is the
// macHandoffCheckShell sh -c invocation. The check proves
// the executor ran against S^{tree}.
func buildMacHandoffPlan(subject, subjectTree string) string {
	plan := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-MAC-HANDOFF-DOGFOOD",
		"baseline": map[string]string{
			"commit_oid": subject,
			"tree_oid":   subjectTree,
		},
		"execution": map[string]any{
			"mode": "serial_fail_fast",
		},
		"checks": []map[string]any{{
			"id":                "mac_handoff_dogfood_proof",
			"mode":              "run",
			"argv":              []string{"sh", "-c", macHandoffCheckShell},
			"working_directory": ".",
			"timeout_seconds":   60,
			"environment":       map[string]string{},
		}},
		"artifacts": []any{},
		"policy": map[string]bool{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("marshal mac handoff plan: %v", err))
	}
	return string(raw)
}

// macHandoffCallerState is a deterministic snapshot of the
// caller-state facets the verifier must not change. The
// helper stores SHA-256 of each facet so a single-byte
// difference is detectable in the assertion.
type macHandoffCallerState struct {
	headCommit  string
	headTree    string
	status      string
	worktrees   string
	refs        string
	statusSHA   string
	worktreeSHA string
	refsSHA     string
}

// captureMacHandoffCallerState snapshots HEAD commit, HEAD
// tree, porcelain-v2 status, worktree inventory, and refs
// for the supplied repository. Each facet is also SHA-256
// hashed so the caller-state drift assertion is
// byte-exact.
func captureMacHandoffCallerState(t *testing.T, repo string) macHandoffCallerState {
	t.Helper()
	statusRaw := runR2CRGit(t, repo, "status", "--porcelain=v2", "--untracked-files=all")
	worktreeRaw := runR2CRGit(t, repo, "worktree", "list", "--porcelain")
	refsRaw := runR2CRGit(t, repo, "for-each-ref", "--format=%(HEAD)%(refname)%00%(objectname)")
	return macHandoffCallerState{
		headCommit:  runR2CRGit(t, repo, "rev-parse", "HEAD"),
		headTree:    runR2CRGit(t, repo, "rev-parse", "HEAD^{tree}"),
		status:      statusRaw,
		worktrees:   worktreeRaw,
		refs:        canonicalizeMacHandoffRefs(refsRaw),
		statusSHA:   sha256HexBytes([]byte(statusRaw)),
		worktreeSHA: sha256HexBytes([]byte(worktreeRaw)),
		refsSHA:     sha256HexBytes([]byte(canonicalizeMacHandoffRefs(refsRaw))),
	}
}

// canonicalizeMacHandoffRefs normalises a refs snapshot to
// a stable byte representation. The output is sorted by
// refname so a different discovery order does not change
// the digest.
func canonicalizeMacHandoffRefs(raw string) string {
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	pairs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		idx := strings.Index(line, "\x00")
		if idx < 0 {
			pairs = append(pairs, line)
			continue
		}
		ref := line[:idx]
		oid := strings.TrimSpace(line[idx+1:])
		if ref == "" || oid == "" {
			continue
		}
		pairs = append(pairs, ref+"\x00"+oid)
	}
	return strings.Join(pairs, "\n")
}

// macHandoffDogfoodResult captures every literal value the
// test observes. The struct is the JSON shape of the
// committed evidence file.
type macHandoffDogfoodResult struct {
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
	Closure                string
	ClosureTree            string
	PlanPath               string
	PlanBlob               string
	PlanSHA256             string
	ManifestPath           string
	ManifestBlob           string
	ManifestSHA256         string
	CallerHeadBefore       string
	CallerHeadAfter        string
	CallerTreeBefore       string
	CallerTreeAfter        string
	CallerStatusBeforeSHA  string
	CallerStatusAfterSHA   string
	WorktreesBeforeSHA     string
	WorktreesAfterSHA      string
	RefsBeforeSHA          string
	RefsAfterSHA           string
	StdoutBytes            int
	StderrBytes            int
	StdoutSHA256           string
	StderrSHA256           string
	ExitCode               int
	TimedOut               bool
	StdoutTruncated        bool
	StderrTruncated        bool
	RunErr                 error
	VerifierOutputPath     string
	VerifierOutputSHA256   string
	EvidencePath           string
	EvidenceSidecarPath    string
	EvidenceSHA256         string
}

var lastMacHandoffDogfood macHandoffDogfoodResult

// writeMacHandoffEvidence serialises the dogfood result
// to deterministic JSON outside the Leamas checkout, then
// computes the file's SHA-256 and writes it to a sibling
// sidecar. The same pattern as R2C-R3: the on-disk
// evidence file does NOT embed its own SHA-256; the
// sidecar holds the digest.
func writeMacHandoffEvidence(t *testing.T, r *macHandoffDogfoodResult) {
	t.Helper()
	dir := os.Getenv("LEAMAS_MAC_HANDOFF_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir evidence dir: %v", err)
	}
	jsonPath := filepath.Join(dir, "mac-handoff-evidence.json")
	sidecarPath := filepath.Join(dir, "mac-handoff-evidence.json.sha256")

	r.EvidencePath = jsonPath
	r.EvidenceSidecarPath = sidecarPath
	r.EvidenceSHA256 = "" // populated after write

	if err := writeMacHandoffEvidenceAtomic(jsonPath, r); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("re-read evidence: %v", err)
	}
	sum := sha256HexBytes(raw)
	if err := os.WriteFile(sidecarPath, []byte(sum+"\n"), 0o644); err != nil {
		t.Fatalf("write evidence sidecar: %v", err)
	}
	r.EvidenceSHA256 = sum

	// Re-read final bytes to verify the file is unchanged
	// after the sidecar was written; the on-disk file MUST
	// not contain the sidecar digest.
	finalBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("re-read final evidence: %v", err)
	}
	if got := sha256HexBytes(finalBytes); got != sum {
		t.Fatalf("evidence file SHA mismatch: got %s want %s", got, sum)
	}
}

// writeMacHandoffEvidenceAtomic writes r to path via a
// temp-file rename so a partial write can never leave a
// half-formed evidence file behind.
func writeMacHandoffEvidenceAtomic(path string, r *macHandoffDogfoodResult) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mac-handoff-evidence-*.tmp")
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

// silence unused imports if a future edit drops one of
// the package-level references above. Each symbol is
// intentionally used; this comment anchors the package
// contract for the next reviewer.
var (
	_ = sha256.Sum256
	_ = hex.EncodeToString
)
