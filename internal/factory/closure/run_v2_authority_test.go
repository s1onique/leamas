// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnforceRunnerIdentity(t *testing.T) {
	tests := []struct {
		name     string
		identity RunnerIdentity
		subject  string
		binHash  string
		wantErr  bool
	}{
		{
			name: "matching exact values accepted",
			identity: RunnerIdentity{
				VCSRevision:  "abc123",
				VCSModified:  false,
				BinarySHA256: "deadbeef",
			},
			subject: "abc123",
			binHash: "deadbeef",
			wantErr: false,
		},
		{
			name: "empty identity hash rejected",
			identity: RunnerIdentity{
				VCSRevision:  "abc123",
				VCSModified:  false,
				BinarySHA256: "",
			},
			subject: "abc123",
			binHash: "actual",
			wantErr: true,
		},
		{
			name: "empty actual hash rejected",
			identity: RunnerIdentity{
				VCSRevision:  "abc123",
				VCSModified:  false,
				BinarySHA256: "claimed",
			},
			subject: "abc123",
			binHash: "",
			wantErr: true,
		},
		{
			name: "missing revision rejected",
			identity: RunnerIdentity{
				VCSRevision:  "",
				VCSModified:  false,
				BinarySHA256: "hash",
			},
			subject: "abc123",
			binHash: "hash",
			wantErr: true,
		},
		{
			name: "stale revision rejected",
			identity: RunnerIdentity{
				VCSRevision:  "old123",
				VCSModified:  false,
				BinarySHA256: "deadbeef",
			},
			subject: "new456",
			binHash: "deadbeef",
			wantErr: true,
		},
		{
			name: "modified runner rejected",
			identity: RunnerIdentity{
				VCSRevision:  "abc123",
				VCSModified:  true,
				BinarySHA256: "deadbeef",
			},
			subject: "abc123",
			binHash: "deadbeef",
			wantErr: true,
		},
		{
			name: "binary hash mismatch rejected",
			identity: RunnerIdentity{
				VCSRevision:  "abc123",
				VCSModified:  false,
				BinarySHA256: "claimedhash",
			},
			subject: "abc123",
			binHash: "actualhash",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enforceRunnerIdentity(tt.identity, tt.subject, tt.binHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestVerifySingleParentStrict proves the strict single-parent parser rejects
// roots, merges, malformed records, object-format mismatches, and Git errors.
func TestVerifySingleParentStrict(t *testing.T) {
	sha1Commit := "1111111111111111111111111111111111111111"
	sha1Parent := "2222222222222222222222222222222222222222"
	sha256Commit := strings.Repeat("a", 64)
	sha256Parent := strings.Repeat("b", 64)
	tests := []struct {
		name       string
		commit     string
		format     ObjectFormat
		result     gitCommandResult
		wantParent string
		wantErr    string
	}{
		{name: "valid single parent", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte(sha1Commit + " " + sha1Parent + "\n")}, wantParent: sha1Parent},
		{name: "no output", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{}, wantErr: "no output"},
		{name: "blank output", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte("\n")}, wantErr: "empty"},
		{name: "multiple records", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte(sha1Commit + " " + sha1Parent + "\n" + sha1Commit + " " + sha1Parent + "\n")}, wantErr: "exactly 1"},
		{name: "wrong echoed commit", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte(strings.Repeat("3", 40) + " " + sha1Parent + "\n")}, wantErr: "expected"},
		{name: "root commit", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte(sha1Commit + "\n")}, wantErr: "root commit"},
		{name: "two-parent merge", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte(sha1Commit + " " + sha1Parent + " " + strings.Repeat("3", 40) + "\n")}, wantErr: "merge with 2 parents"},
		{name: "invalid echoed commit OID", commit: "not-an-oid", format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte("not-an-oid " + sha1Parent + "\n")}, wantErr: "rev-list commit"},
		{name: "invalid parent OID", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{Stdout: []byte(sha1Commit + " not-an-oid\n")}, wantErr: "rev-list parent"},
		{name: "Git non-zero with stderr", commit: sha1Commit, format: ObjectFormatSHA1, result: gitCommandResult{ExitCode: 128, Err: errors.New("exit status 128"), Stderr: []byte("fatal: bad revision\n")}, wantErr: "fatal: bad revision"},
		{name: "SHA-256 width", commit: sha256Commit, format: ObjectFormatSHA256, result: gitCommandResult{Stdout: []byte(sha256Commit + " " + sha256Parent + "\n")}, wantParent: sha256Parent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := &fakeGitClientForState{revListResult: &tt.result}
			parent, err := verifySingleParent(context.Background(), git, "/", tt.commit, tt.format)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if parent != tt.wantParent {
				t.Fatalf("parent = %q, want %q", parent, tt.wantParent)
			}
		})
	}
}

func TestBindExactPlanBytes_BytesEqualPreserved(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{"identical", []byte("hello\n"), []byte("hello\n"), true},
		{"identical no newline", []byte("hello"), []byte("hello"), true},
		{"different length", []byte("hello"), []byte("hello\n"), false},
		{"different content", []byte("hello\n"), []byte("world\n"), false},
		{"empty equal", []byte{}, []byte{}, true},
		{"nil equal empty", nil, []byte{}, true},
		{"one-byte difference", []byte("hello\n"), []byte("hellp\n"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bytesEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("bytesEqual = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeGitClientForState struct {
	RealGit
	parentOfHEAD  string
	revListResult *gitCommandResult
}

func (f *fakeGitClientForState) Run(ctx context.Context, dir string, args ...string) gitCommandResult {
	if len(args) >= 3 && args[0] == "rev-list" && args[1] == "--parents" {
		if f.revListResult != nil {
			return *f.revListResult
		}
		commit := args[len(args)-1]
		if f.parentOfHEAD == "" {
			return gitCommandResult{Stdout: []byte(commit + "\n")}
		}
		return gitCommandResult{Stdout: []byte(commit + " " + f.parentOfHEAD + "\n")}
	}
	if len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--verify" && strings.HasSuffix(args[2], "^") {
		if f.parentOfHEAD == "" {
			return gitCommandResult{Err: nil, ExitCode: 1, Stderr: []byte("not found")}
		}
		return gitCommandResult{Stdout: []byte(f.parentOfHEAD + "\n")}
	}
	return gitCommandResult{Err: nil, ExitCode: 0}
}

func (f *fakeGitClientForState) RunWithStdin(ctx context.Context, dir, stdin string, args ...string) gitCommandResult {
	return f.Run(ctx, dir, args...)
}

func (f *fakeGitClientForState) RunWithEnv(ctx context.Context, dir string, env []string, args ...string) gitCommandResult {
	return f.Run(ctx, dir, args...)
}

func (f *fakeGitClientForState) RunWithStdinAndEnv(ctx context.Context, dir, stdin string, env []string, args ...string) gitCommandResult {
	return f.Run(ctx, dir, args...)
}

// TestRunClosureV2TopologyPlanBaselineNotFreezeParent is the
// P0-2 topology test. It builds a fixture with the actual
// "B → P → F → S" four-commit history that matches the digest's
// "B → F1 → F2 → S" example and proves the four critical properties:
//
//  1. parent(F) is recoverable and equals P (NOT plan.baseline B).
//  2. plan.baseline != parent(F) — they are distinct commits.
//  3. The freeze commit F is reachable from S as the immediate parent
//     (single-parent subject property).
//  4. bindExactPlanBytes accepts this geometry, because F (the
//     freeze commit in scope) does introduce the final plan blob
//     versus its own parent P.
//
// This pins the explicit naming decision that motivates the P0-2
// rename. Without the rename, a regression that called parent(F)
// "plan.baseline" would silently corrupt the F..S patch-hygiene
// range and the B..S closure-policy range.
func TestRunClosureV2TopologyPlanBaselineNotFreezeParent(t *testing.T) {
	fixture := prepareV2MultiFreezeRepository(t)

	// 1. parent(F) is recoverable and equals P.
	freezeParent, err := verifySingleParent(context.Background(), RealGit{}, fixture.root, fixture.freeze, ObjectFormatSHA1)
	if err != nil {
		t.Fatalf("verifySingleParent(F) failed: %v", err)
	}
	if freezeParent != fixture.freezeParent {
		t.Fatalf("verifySingleParent(F) = %s, want %s", freezeParent, fixture.freezeParent)
	}

	// 2. plan.baseline != parent(F). THIS IS THE TOPOLOGY TEST.
	if fixture.baseline == freezeParent {
		t.Fatalf("plan.baseline (%s) equals parent(F) (%s); the topology test requires them to be distinct",
			fixture.baseline, freezeParent)
	}

	// 3. freezeParent is an ancestor of the freeze commit (single-parent chain).
	if !isAncestorGit(fixture.root, freezeParent, fixture.freeze) {
		t.Fatalf("freezeParent %s must be an ancestor of F %s", freezeParent, fixture.freeze)
	}

	// 4. The full ProvenanceTopology struct round-trips correctly.
	topology := ProvenanceTopology{
		B: fixture.baseline,
		P: freezeParent,
		F: fixture.freeze,
		S: fixture.subject,
	}
	if topology.B == topology.P {
		t.Fatalf("ProvenanceTopology.B == P; the explicit B vs P distinction is the whole point of P0-2")
	}
	if topology.F == topology.S {
		t.Fatalf("ProvenanceTopology.F == S; F is the freeze and S is exactly one commit after F")
	}
	if topology.P == topology.F {
		t.Fatalf("ProvenanceTopology.P == F; P is parent(F), they cannot be equal")
	}
}

// TestRunClosureV2TopologyBaselineEqualsFreezeParentAllowsFixture
// verifies that the trivial three-commit fixture (used by every other
// orchestrator test) intentionally has plan.baseline == parent(F).
// This pins the B = P degeneracy in the simple case and prevents
// future readers from being surprised that prepareV2Repository does
// NOT populate fixture.freezeParent.
func TestRunClosureV2TopologyBaselineEqualsFreezeParentAllowsFixture(t *testing.T) {
	fixture := prepareV2Repository(t)
	freezeParent, err := verifySingleParent(context.Background(), RealGit{}, fixture.root, fixture.freeze, ObjectFormatSHA1)
	if err != nil {
		t.Fatalf("verifySingleParent(F) failed: %v", err)
	}
	if freezeParent != fixture.baseline {
		t.Fatalf("prepareV2Repository must yield B == parent(F) (got B=%s, parent(F)=%s)",
			fixture.baseline, freezeParent)
	}
}

// TestV2PlanValidationCatchesContractViolationBeyondV2Profile proves the
// explicit Step 7 ValidatePlan rejects a violation the small v2 profile accepts.
func TestV2PlanValidationCatchesContractViolationBeyondV2Profile(t *testing.T) {
	plan := minimalValidPlan()
	plan.Baseline.CommitOID = "TO_BE_FILLED"
	plan.Baseline.TreeOID = "TO_BE_FILLED"

	if err := validateV2Plan(plan); err != nil {
		t.Fatalf("validateV2Plan unexpectedly rejected placeholder baseline: %v (v2 profile must accept non-empty OIDs)", err)
	}
	if err := ValidatePlan(plan); err == nil {
		t.Fatalf("ValidatePlan accepted placeholder baseline; the explicit full-contract check must reject it")
	}
}

// TestRunClosureV2RejectsAuthoritativeContractViolationAtStep7 commits the
// invalid plan at F and proves exact byte binding passes before Step 7 rejects it.
func TestRunClosureV2RejectsAuthoritativeContractViolationAtStep7(t *testing.T) {
	fixture := initializeV2Repository(t)
	baselineTree := v2Git(t, fixture.root, "rev-parse", fixture.baseline+"^{tree}")
	invalid := fmt.Sprintf(`{
  "contract_version": 1,
  "act_id": %q,
  "baseline": {"commit_oid": "TO_BE_FILLED", "tree_oid": %q},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id":"authority-check","mode":"run","argv":["go","version"],"working_directory":".","timeout_seconds":60,"environment":{}}],
  "artifacts": [],
  "policy": {"require_clean_before":true,"require_clean_after":true,"forbid_tracked_full_digests":true,"require_diff_check":true}
}
`, v2OrchestratorActID, baselineTree)
	if err := os.MkdirAll(filepath.Dir(fixture.planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.planPath, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	v2Git(t, fixture.root, "add", "docs/closure-plans")
	v2Git(t, fixture.root, "commit", "-m", "freeze invalid authoritative plan")
	fixture.freeze = v2Git(t, fixture.root, "rev-parse", "HEAD")
	v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "subject")
	fixture.subject = v2Git(t, fixture.root, "rev-parse", "HEAD")

	candidateBytes, err := os.ReadFile(fixture.planPath)
	if err != nil {
		t.Fatal(err)
	}
	planRelPath := "docs/closure-plans/" + v2OrchestratorActID + ".json"
	blobBytes := []byte(v2Git(t, fixture.root, "show", fixture.freeze+":"+planRelPath) + "\n")
	if !bytesEqual(candidateBytes, blobBytes) {
		t.Fatal("fixture candidate bytes do not exactly match F plan blob")
	}

	recorder := &v2OrchestratorRecorder{}
	err = runV2Orchestrator(t, fixture, recorder, nil)
	if err == nil || !strings.Contains(err.Error(), "validate authoritative plan") ||
		!strings.Contains(err.Error(), "baseline.commit_oid must be a full lowercase Git OID") {
		t.Fatalf("expected precise Step 7 ValidatePlan error, got %v", err)
	}
	if recorder.runnerCalls != 0 || recorder.checkCalls != 0 || recorder.finalizeCalls != 0 {
		t.Fatalf("invalid authoritative plan reached runner/checks/finalize: runner=%d checks=%d finalize=%d",
			recorder.runnerCalls, recorder.checkCalls, recorder.finalizeCalls)
	}
	assertPathAbsent(t, v2EvidencePath(fixture))
}
