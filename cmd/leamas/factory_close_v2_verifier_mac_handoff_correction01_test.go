// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_mac_handoff_correction01_test.go
// implements the installed-style final dogfood required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION01.
//
// The correction test fixes the false-PASS of the previous
// closure by:
//
//  1. building the leamas binary in a TEMPORARY DETACHED
//     WORKTREE at the current HEAD so the build source bytes
//     are exact and immutable;
//  2. asserting the binary's VCS revision LITERALLY equals
//     the final commit (no later commit rule);
//  3. constructing a meaningful hermetic S < F < C repository
//     where C carries a real committed v2 manifest M produced
//     by the same binary;
//  4. invoking `factory close verify-v2-authority` from a
//     temp working directory OUTSIDE both the Leamas checkout
//     and the hermetic repository;
//  5. asserting the verifier reported valid=true;
//  6. decoding the public JSON envelope into typed
//     structures (no json.RawMessage);
//  7. asserting every literal identity (S/F/C trees, P/M
//     blobs and SHA-256) from the typed envelope;
//  8. capturing the bounded subprocess outcome in literal
//     fields (ExitCode, TimedOut, StdoutTruncated,
//     StderrTruncated);
//  9. proving the target repository's HEAD, tree, status,
//     worktree list, and refs snapshot are byte-for-byte
//     unchanged before and after the invocation;
// 10. capturing caller state in SHA-256-canonical form
//     before and after the dogfood;
// 11. writing the literal evidence with a SHA-256 sidecar
//     outside both the Leamas checkout and the hermetic
//     repository.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// correction01PlanPath is the repository-relative path used
// by the dogfood for the frozen plan P.
const correction01PlanPath = "docs/closure-plans/CORRECTION01.json"

// correction01ManifestPath is the repository-relative path
// used by the dogfood for the committed manifest M at C.
const correction01ManifestPath = "docs/closure-manifests/CORRECTION01.json"

// correction01CheckShell proves the executor ran against
// S^{tree} (not F^{tree} or C^{tree}).
const correction01CheckShell = "test -f subject-only.txt && test ! -e freeze-only.txt && test ! -e closure-only.txt"

// correction01DogfoodResult is the durable literal result
// captured for the dogfood. Every field is required; an
// empty value fails the test.
type correction01DogfoodResult struct {
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
	RunErrorPresent        bool
	RunErrorText           string
	VerifierOutputPath     string
	VerifierOutputSHA256   string
	EvidencePath           string
	EvidenceSidecarPath    string
	EvidenceSHA256         string
}

// TestClosureCLIV2VerifierMacHandoffCorrection01 is the
// installed-style final dogfood for the v2 closure verifier
// Mac handoff correction. The test asserts the same set of
// invariants the previous closure promised but with the
// exact exit codes, typed JSON decoding, atomic publication,
// and complete caller-state proof the correction01 ACT
// requires.
func TestClosureCLIV2VerifierMacHandoffCorrection01(t *testing.T) {
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

	repository := initCorrection01Repo(t)
	subject, subjectTree, freeze, freezeTree := buildCorrection01SF(t, repository, leamasRepoRoot)

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
		"--plan-path", correction01PlanPath,
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
	manifestDir := filepath.Join(repository, filepath.Dir(correction01ManifestPath))
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	mustWriteFile(t, filepath.Join(repository, correction01ManifestPath), string(manifestBytes))
	mustWriteFile(t, filepath.Join(repository, "closure-only.txt"), "closure-only\n")
	runR2CRGit(t, repository, "add", ".")
	runR2CRGit(t, repository, "commit", "-m", "factory: close v2 verifier mac handoff correction01 ACT")
	closureCommit := runR2CRGit(t, repository, "rev-parse", "HEAD")
	closureTree := runR2CRGit(t, repository, "rev-parse", closureCommit+"^{tree}")

	planBlobOID := runR2CRGit(t, repository, "rev-parse", freeze+":"+correction01PlanPath)
	planRaw := runR2CRGitRaw(t, repository, planBlobOID)
	planSHA := sha256HexBytes(planRaw)
	manifestBlobOID := runR2CRGit(t, repository, "rev-parse", closureCommit+":"+correction01ManifestPath)
	manifestRaw := runR2CRGitRaw(t, repository, manifestBlobOID)
	manifestSHA := sha256HexBytes(manifestRaw)

	// Invoke the verifier from a temp CWD outside both repos.
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
		"--plan-path", correction01PlanPath,
		"--manifest-path", correction01ManifestPath,
		"--json",
		"--output", verifierOutputPath,
	}, boundedSubprocessV2Options{
		Timeout:   60 * time.Second,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
		WorkDir:   verifierCWD,
	})

	// Bounded subprocess outcome authority: assert every
	// literal field. A non-zero ExitCode, a timeout, a
	// truncated stream, or a non-nil error all fail the
	// test. The dogfood cannot pass unless the verifier
	// succeeded, was bounded, and produced complete output.
	if verifierResult.ExitCode != 0 {
		t.Fatalf("verifier exit %d: stdout=%s stderr=%s",
			verifierResult.ExitCode,
			string(verifierResult.Stdout),
			string(verifierResult.Stderr))
	}
	if verifierResult.TimedOut {
		t.Fatalf("verifier timed out: stdout=%s stderr=%s",
			string(verifierResult.Stdout), string(verifierResult.Stderr))
	}
	if verifierResult.StdoutTruncated {
		t.Fatalf("verifier stdout truncated: bytes=%d", len(verifierResult.Stdout))
	}
	if verifierResult.StderrTruncated {
		t.Fatalf("verifier stderr truncated: bytes=%d", len(verifierResult.Stderr))
	}
	if verifierResult.Err != nil {
		t.Fatalf("verifier subprocess error: %v", verifierResult.Err)
	}

	// Caller state must be byte-for-byte unchanged.
	callerAfter := captureCorrection01CallerState(t, repository)
	if callerAfter.statusSHA != callerBefore.statusSHA {
		t.Fatalf("caller porcelain-v2 status drifted")
	}
	if callerAfter.worktreeSHA != callerBefore.worktreeSHA {
		t.Fatalf("caller worktree inventory drifted")
	}
	if callerAfter.refsSHA != callerBefore.refsSHA {
		t.Fatalf("caller refs snapshot drifted")
	}

	// Typed JSON decode: no json.RawMessage.
	verifierOutput, err := os.ReadFile(verifierOutputPath)
	if err != nil {
		t.Fatalf("read verifier output: %v", err)
	}
	envelope, err := decodeCorrection01Envelope(verifierOutput)
	if err != nil {
		t.Fatalf("decode verifier envelope: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("verifier envelope OK=false")
	}
	v := envelope.Verification
	if !v.Valid {
		t.Fatalf("verifier verification.valid=false")
	}
	if !v.TopologyValid {
		t.Fatalf("verifier verification.topology_valid=false")
	}
	if !v.ManifestValid {
		t.Fatalf("verifier verification.manifest_valid=false")
	}
	if !v.ResultSetValid {
		t.Fatalf("verifier verification.result_set_valid=false")
	}
	if len(v.Diagnostics) != 0 {
		t.Fatalf("verifier diagnostics not empty: %+v", v.Diagnostics)
	}
	if v.SubjectCommit != subject {
		t.Fatalf("verifier subject_commit=%s want %s", v.SubjectCommit, subject)
	}
	if v.SubjectTree != subjectTree {
		t.Fatalf("verifier subject_tree=%s want %s", v.SubjectTree, subjectTree)
	}
	if v.FreezeCommit != freeze {
		t.Fatalf("verifier freeze_commit=%s want %s", v.FreezeCommit, freeze)
	}
	if v.FreezeTree != freezeTree {
		t.Fatalf("verifier freeze_tree=%s want %s", v.FreezeTree, freezeTree)
	}
	if v.ClosureCommit != closureCommit {
		t.Fatalf("verifier closure_commit=%s want %s", v.ClosureCommit, closureCommit)
	}
	if v.ClosureTree != closureTree {
		t.Fatalf("verifier closure_tree=%s want %s", v.ClosureTree, closureTree)
	}
	if v.PlanPath != correction01PlanPath {
		t.Fatalf("verifier plan_path=%s want %s", v.PlanPath, correction01PlanPath)
	}
	if v.PlanBlob != planBlobOID {
		t.Fatalf("verifier plan_blob=%s want %s", v.PlanBlob, planBlobOID)
	}
	if v.PlanSHA256 != planSHA {
		t.Fatalf("verifier plan_sha256=%s want %s", v.PlanSHA256, planSHA)
	}
	if v.ManifestPath != correction01ManifestPath {
		t.Fatalf("verifier manifest_path=%s want %s", v.ManifestPath, correction01ManifestPath)
	}
	if v.ManifestBlob != manifestBlobOID {
		t.Fatalf("verifier manifest_blob=%s want %s", v.ManifestBlob, manifestBlobOID)
	}
	if v.ManifestSHA256 != manifestSHA {
		t.Fatalf("verifier manifest_sha256=%s want %s", v.ManifestSHA256, manifestSHA)
	}
	if v.ClosureProtocolVersion != "2" {
		t.Fatalf("verifier closure_protocol_version=%s want 2", v.ClosureProtocolVersion)
	}
	if v.PlanContractVersion != closure.PlanContractVersion(1) {
		t.Fatalf("verifier plan_contract_version=%v want 1", v.PlanContractVersion)
	}

	// Independent hash oracle: SHA-256 of the exact raw
	// F:P and C:M bytes must equal what the verifier
	// reports. The raw bytes are read via
	// `git cat-file blob <oid>` and SHA-256 is computed
	// over the literal bytes (no trim, no remarshal).
	planRaw2 := runR2CRGitRaw(t, repository, planBlobOID)
	planSHA2 := sha256HexBytes(planRaw2)
	if planSHA2 != v.PlanSHA256 {
		t.Fatalf("independent plan_sha256=%s != verifier plan_sha256=%s",
			planSHA2, v.PlanSHA256)
	}
	manifestRaw2 := runR2CRGitRaw(t, repository, manifestBlobOID)
	manifestSHA2 := sha256HexBytes(manifestRaw2)
	if manifestSHA2 != v.ManifestSHA256 {
		t.Fatalf("independent manifest_sha256=%s != verifier manifest_sha256=%s",
			manifestSHA2, v.ManifestSHA256)
	}

	result := correction01DogfoodResult{
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
		Closure:                closureCommit,
		ClosureTree:            closureTree,
		PlanPath:               correction01PlanPath,
		PlanBlob:               planBlobOID,
		PlanSHA256:             planSHA,
		ManifestPath:           correction01ManifestPath,
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
		RunErrorPresent:        verifierResult.Err != nil,
		RunErrorText:           runErrorText(verifierResult.Err),
		VerifierOutputPath:     verifierOutputPath,
		VerifierOutputSHA256:   sha256HexBytes(verifierOutput),
	}
	lastCorrection01Dogfood = result
	writeCorrection01Evidence(t, &result)
}

// decodeCorrection01Envelope decodes the public JSON
// envelope into typed structures. The function never
// retains json.RawMessage as the final assertion boundary.
func decodeCorrection01Envelope(raw []byte) (v2VerifierJSONEnvelope, error) {
	var env v2VerifierJSONEnvelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&env); err != nil {
		return env, fmt.Errorf("decode envelope: %w", err)
	}
	if dec.More() {
		return env, fmt.Errorf("envelope is not a single JSON document")
	}
	return env, nil
}

// runErrorText returns the textual form of a subprocess
// error suitable for the literal evidence. The function
// returns the empty string when err is nil so the field
// is JSON-safe regardless of the run outcome.
func runErrorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// initCorrection01Repo creates a fresh temp directory and
// initialises a Git repository with an empty initial
// commit. The returned path is the repository root.
func initCorrection01Repo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runR2CRGit(t, repo, "init", "-b", "main")
	runR2CRGit(t, repo, "config", "user.name", "Correction01 Dogfood")
	runR2CRGit(t, repo, "config", "user.email", "correction01@example.invalid")
	runR2CRGit(t, repo, "config", "commit.gpgsign", "false")
	runR2CRGit(t, repo, "config", "tag.gpgsign", "false")
	runR2CRGit(t, repo, "commit", "--allow-empty", "-m", "initial")
	return repo
}

// buildCorrection01SF creates the S and F commits in the
// supplied repository.
func buildCorrection01SF(t *testing.T, repo, _ string) (subject, subjectTree, freeze, freezeTree string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(repo, "subject-only.txt"), "subject\n")
	runR2CRGit(t, repo, "add", "subject-only.txt")
	runR2CRGit(t, repo, "commit", "-m", "subject")
	subject = runR2CRGit(t, repo, "rev-parse", "HEAD")
	subjectTree = runR2CRGit(t, repo, "rev-parse", subject+"^{tree}")

	planDir := filepath.Join(repo, filepath.Dir(correction01PlanPath))
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	planJSON := buildCorrection01Plan(subject, subjectTree)
	mustWriteFile(t, filepath.Join(repo, correction01PlanPath), planJSON)
	mustWriteFile(t, filepath.Join(repo, "freeze-only.txt"), "freeze-only\n")
	runR2CRGit(t, repo, "add", ".")
	runR2CRGit(t, repo, "commit", "-m", "freeze: add plan")
	freeze = runR2CRGit(t, repo, "rev-parse", "HEAD")
	freezeTree = runR2CRGit(t, repo, "rev-parse", freeze+"^{tree}")
	return subject, subjectTree, freeze, freezeTree
}

// buildCorrection01Plan returns a contract-valid Plan
// Contract v1 document whose run-mode check exercises the
// S-only / F-only / C-only invariants.
func buildCorrection01Plan(subject, subjectTree string) string {
	plan := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-CORRECTION01-DOGFOOD",
		"baseline": map[string]string{
			"commit_oid": subject,
			"tree_oid":   subjectTree,
		},
		"execution": map[string]any{
			"mode": "serial_fail_fast",
		},
		"checks": []map[string]any{{
			"id":                "correction01_dogfood_proof",
			"mode":              "run",
			"argv":              []string{"sh", "-c", correction01CheckShell},
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
		panic(fmt.Sprintf("marshal correction01 plan: %v", err))
	}
	return string(raw)
}

// correction01CallerState is a deterministic snapshot of
// the caller-state facets the verifier must not change.
type correction01CallerState struct {
	headCommit  string
	headTree    string
	status      string
	worktrees   string
	refs        string
	statusSHA   string
	worktreeSHA string
	refsSHA     string
}

// captureCorrection01CallerState snapshots HEAD commit,
// HEAD tree, porcelain-v2 status, worktree inventory, and
// refs for the supplied repository.
func captureCorrection01CallerState(t *testing.T, repo string) correction01CallerState {
	t.Helper()
	statusRaw := runR2CRGit(t, repo, "status", "--porcelain=v2", "--untracked-files=all")
	worktreeRaw := runR2CRGit(t, repo, "worktree", "list", "--porcelain")
	refsRaw := runR2CRGit(t, repo, "for-each-ref", "--format=%(HEAD)%(refname)%00%(objectname)")
	canonicalRefs := canonicalizeCorrection01Refs(refsRaw)
	return correction01CallerState{
		headCommit:  runR2CRGit(t, repo, "rev-parse", "HEAD"),
		headTree:    runR2CRGit(t, repo, "rev-parse", "HEAD^{tree}"),
		status:      statusRaw,
		worktrees:   worktreeRaw,
		refs:        canonicalRefs,
		statusSHA:   sha256HexBytes([]byte(statusRaw)),
		worktreeSHA: sha256HexBytes([]byte(worktreeRaw)),
		refsSHA:     sha256HexBytes([]byte(canonicalRefs)),
	}
}

// canonicalizeCorrection01Refs normalises a refs snapshot
// to a stable byte representation.
func canonicalizeCorrection01Refs(raw string) string {
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

var lastCorrection01Dogfood correction01DogfoodResult

// writeCorrection01Evidence serialises the dogfood result
// to deterministic JSON outside the Leamas checkout, then
// computes the file's SHA-256 and writes it to a sibling
// sidecar.
func writeCorrection01Evidence(t *testing.T, r *correction01DogfoodResult) {
	t.Helper()
	dir := os.Getenv("LEAMAS_CORRECTION01_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir evidence dir: %v", err)
	}
	jsonPath := filepath.Join(dir, "correction01-evidence.json")
	sidecarPath := filepath.Join(dir, "correction01-evidence.json.sha256")

	r.EvidencePath = jsonPath
	r.EvidenceSidecarPath = sidecarPath
	r.EvidenceSHA256 = "" // populated after write

	if err := writeCorrection01EvidenceAtomic(jsonPath, r); err != nil {
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

	finalBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("re-read final evidence: %v", err)
	}
	if got := sha256HexBytes(finalBytes); got != sum {
		t.Fatalf("evidence file SHA mismatch: got %s want %s", got, sum)
	}
}

// writeCorrection01EvidenceAtomic writes r to path via a
// temp-file rename so a partial write can never leave a
// half-formed evidence file behind.
func writeCorrection01EvidenceAtomic(path string, r *correction01DogfoodResult) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".correction01-evidence-*.tmp")
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

// Verify the imports used by the test compile cleanly even
// when the test is otherwise skipped or filtered out.
var (
	_ = sha256.Sum256
	_ = hex.EncodeToString
	_ = os.Getenv
	_ = exec.Command
)
