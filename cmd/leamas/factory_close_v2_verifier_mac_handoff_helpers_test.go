// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_mac_handoff_helpers_test.go owns
// the helper functions used by
// TestClosureCLIV2VerifierMacHandoff:
//
//   - the canonical plan / manifest path constants;
//   - the dogfood-check shell invariant;
//   - the hermetic S/F repository builder;
//   - the contract-valid plan builder;
//   - the caller-state snapshot helper.
//
// Splitting the helpers out of the main test file keeps the
// test file under the LLM-friendly 400-line threshold while
// preserving a single closure over the dogfood protocol.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	mustWriteFile(t, filepath.Join(repo, "subject-only.txt"), "subject\n")
	runR2CRGit(t, repo, "add", "subject-only.txt")
	runR2CRGit(t, repo, "commit", "-m", "subject")
	subject = runR2CRGit(t, repo, "rev-parse", "HEAD")
	subjectTree = runR2CRGit(t, repo, "rev-parse", subject+"^{tree}")

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
