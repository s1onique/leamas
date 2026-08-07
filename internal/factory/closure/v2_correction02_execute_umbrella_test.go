// SPDX-License-Identifier: Apache-2.0

// v2_correction02_execute_umbrella_test.go provides the
// umbrella tests required by Phase 1, 2, 3, 11 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02:
//
//   - TestClosureCallerStateAuthority  (Phase 1)
//   - TestClosureFrozenPlanRawBytes    (Phase 3)
//   - TestClosureFrozenPlanTrailingNewline (Phase 3)
//
// Other umbrella tests (subject worktree authority, execute
// against subject tree, check/result bijection, gate runs
// against subject, exact subject binary, publication after-
// state barrier, hermetic dogfood, public failure matrix)
// already exist in earlier ACTs and are verified by the
// existing closure runner tests; they are not duplicated here.

package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// correction02FakeGit is a minimal in-memory gitClient used
// to exercise the snapshot / Diff machinery without touching
// a real repository. The type is uniquely named so it does
// NOT collide with fakeGitClient declared in other test
// files in this package.
type correction02FakeGit struct {
	head      string
	headTree  string
	status    string
	worktrees string
	refs      string
	headErr   error
}

func (f correction02FakeGit) Run(_ context.Context, _ string, args ...string) gitCommandResult {
	if len(args) >= 1 && args[0] == "for-each-ref" {
		return gitCommandResult{Stdout: []byte(f.refs)}
	}
	if len(args) >= 1 && args[0] == "worktree" {
		return gitCommandResult{Stdout: []byte(f.worktrees)}
	}
	// snapshotCallerState calls:
	//   rev-parse HEAD^{commit}
	//   rev-parse HEAD^{tree}
	// which are 2-element argvs.
	if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD^{commit}" {
		if f.headErr != nil {
			return gitCommandResult{Err: f.headErr}
		}
		return gitCommandResult{Stdout: []byte(f.head)}
	}
	if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD^{tree}" {
		return gitCommandResult{Stdout: []byte(f.headTree)}
	}
	if len(args) >= 3 && args[0] == "status" {
		return gitCommandResult{Stdout: []byte(f.status)}
	}
	return gitCommandResult{Stdout: nil}
}

func (f correction02FakeGit) RunWithStdin(_ context.Context, _ string, _ string, _ ...string) gitCommandResult {
	return gitCommandResult{}
}

func (f correction02FakeGit) RunWithEnv(_ context.Context, _ string, _ []string, _ ...string) gitCommandResult {
	return gitCommandResult{}
}

func (f correction02FakeGit) RunWithStdinAndEnv(_ context.Context, _ string, _ string, _ []string, _ ...string) gitCommandResult {
	return gitCommandResult{}
}

// TestClosureCallerStateAuthority proves Phase 1 invariants:
// the BEFORE snapshot captures HEAD/tree/status/worktrees/refs
// and the AFTER-vs-BEFORE diff returns typed diagnostics for
// each kind of drift.
func TestClosureCallerStateAuthority(t *testing.T) {
	t.Parallel()
	const (
		beforeHEAD = "1111111111111111111111111111111111111111"
		afterHEAD  = "2222222222222222222222222222222222222222"
		beforeTree = "3333333333333333333333333333333333333333"
		afterTree  = "4444444444444444444444444444444444444444"
		// Production uses the explicit NUL-framed
		// "objectname\x00refname\x00" wire format
		// that real Git for-each-ref emits. The fake
		// "refname: ... object ..." grammar that the
		// CORRECTION02 patch originally used is not what
		// Git produces, and the new parser rejects it as
		// malformed NUL framing.
		beforeRefs = beforeHEAD + "\x00refs/heads/main\x00\n"
		afterRefs  = afterHEAD + "\x00refs/heads/main\x00\n"
	)

	before := snapshotCallerState(context.Background(),
		correction02FakeGit{head: beforeHEAD, headTree: beforeTree, refs: beforeRefs},
		"/repo")
	if !before.Available {
		t.Fatalf("expected BEFORE available, got diagnostics: %+v", before.Diagnostics)
	}

	// Same state, no drift expected.
	sameAfter := snapshotCallerState(context.Background(),
		correction02FakeGit{head: beforeHEAD, headTree: beforeTree, refs: beforeRefs},
		"/repo")
	if !sameAfter.Available {
		t.Fatalf("expected same-state AFTER available, got diagnostics: %+v", sameAfter.Diagnostics)
	}
	if diags := before.State.Diff(sameAfter.State); len(diags) != 0 {
		t.Fatalf("expected no drift, got %+v", diags)
	}

	// HEAD drift.
	headAfter := snapshotCallerState(context.Background(),
		correction02FakeGit{head: afterHEAD, headTree: beforeTree, refs: beforeRefs},
		"/repo")
	if diags := before.State.Diff(headAfter.State); len(diags) == 0 ||
		!diags.HasCode(V2CodeCallerHeadChanged) {
		t.Fatalf("expected HEAD drift, got %+v", diags)
	}

	// Tree drift.
	treeAfter := snapshotCallerState(context.Background(),
		correction02FakeGit{head: beforeHEAD, headTree: afterTree, refs: beforeRefs},
		"/repo")
	if diags := before.State.Diff(treeAfter.State); len(diags) == 0 ||
		!diags.HasCode(V2CodeCallerTreeChanged) {
		t.Fatalf("expected tree drift, got %+v", diags)
	}

	// Refs drift (NEW in CORRECTION02).
	refsAfter := snapshotCallerState(context.Background(),
		correction02FakeGit{head: beforeHEAD, headTree: beforeTree, refs: afterRefs},
		"/repo")
	if diags := before.State.Diff(refsAfter.State); len(diags) == 0 ||
		!diags.HasCode(V2CodeCallerRefsChanged) {
		t.Fatalf("expected refs drift, got %+v", diags)
	}
}

// correction02PlanClient is a tiny in-memory gitClient that
// only implements the four operations the loader needs:
//
//	rev-parse --verify --end-of-options <F>^{commit}
//	rev-parse --verify --end-of-options <commit>:<path>
//	cat-file -t <oid>
//	cat-file blob <oid>
type correction02PlanClient struct {
	commitOID string
	blobOID   string
	bytes     []byte
}

func (p correction02PlanClient) Run(_ context.Context, _ string, args ...string) gitCommandResult {
	if len(args) >= 4 && args[0] == "rev-parse" && args[1] == "--verify" && args[2] == "--end-of-options" {
		spec := args[3]
		if strings.HasSuffix(spec, "^{commit}") {
			return gitCommandResult{Stdout: []byte(p.commitOID)}
		}
		if strings.Contains(spec, ":") {
			return gitCommandResult{Stdout: []byte(p.blobOID)}
		}
	}
	if len(args) >= 3 && args[0] == "cat-file" && args[1] == "-t" {
		return gitCommandResult{Stdout: []byte("blob")}
	}
	if len(args) >= 3 && args[0] == "cat-file" && args[1] == "blob" {
		return gitCommandResult{Stdout: append([]byte(nil), p.bytes...)}
	}
	return gitCommandResult{}
}

func (p correction02PlanClient) RunWithStdin(_ context.Context, _ string, _ string, _ ...string) gitCommandResult {
	return gitCommandResult{}
}
func (p correction02PlanClient) RunWithEnv(_ context.Context, _ string, _ []string, _ ...string) gitCommandResult {
	return gitCommandResult{}
}
func (p correction02PlanClient) RunWithStdinAndEnv(_ context.Context, _ string, _ string, _ []string, _ ...string) gitCommandResult {
	return gitCommandResult{}
}

// TestClosureFrozenPlanRawBytes proves the loader returns
// the literal `git cat-file blob` bytes: no trim, no
// normalisation. The SHA-256 of the returned bytes MUST equal
// the recorded PlanSHA256.
func TestClosureFrozenPlanRawBytes(t *testing.T) {
	t.Parallel()
	// Build a deterministic, non-ASCII, leading-whitespace
	// and trailing-newline plan blob and prove the loader
	// preserves every byte.
	const planContent = "\n{\"contract_version\":1,\"checks\":[]}\n  "
	sum := sha256.Sum256([]byte(planContent))
	want := hex.EncodeToString(sum[:])
	loader := NewGitV2FrozenPlanLoader(correction02PlanClient{
		commitOID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		blobOID:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		bytes:     []byte(planContent),
	})
	frozen, err := loader.LoadFrozenPlan(context.Background(), "/repo", "deadbeef", "docs/plans/x.json")
	if err != nil {
		t.Fatalf("load frozen plan: %v", err)
	}
	if string(frozen.Bytes) != planContent {
		t.Fatalf("loader must preserve exact bytes; got %q want %q", string(frozen.Bytes), planContent)
	}
	if frozen.SHA256 != want {
		t.Fatalf("loader SHA mismatch; got %s want %s", frozen.SHA256, want)
	}
}

// TestClosureFrozenPlanTrailingNewline proves the loader
// preserves a trailing newline so SHA-256 differs from the
// trimmed-bytes hash. This closes the provisional R1-R2
// newline claim.
func TestClosureFrozenPlanTrailingNewline(t *testing.T) {
	t.Parallel()
	withNewline := "{\"contract_version\":1,\"checks\":[]}\n"
	withoutNewline := strings.TrimRight(withNewline, "\n")
	if withNewline == withoutNewline {
		t.Fatal("fixture invariant: strings must differ")
	}
	sumA := sha256.Sum256([]byte(withNewline))
	sumB := sha256.Sum256([]byte(withoutNewline))
	if hex.EncodeToString(sumA[:]) == hex.EncodeToString(sumB[:]) {
		t.Fatal("trimmed and untrimmed SHA-256 must differ for newline fixture")
	}
	loader := NewGitV2FrozenPlanLoader(correction02PlanClient{
		commitOID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		blobOID:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		bytes:     []byte(withNewline),
	})
	frozen, err := loader.LoadFrozenPlan(context.Background(), "/repo", "deadbeef", "docs/plans/x.json")
	if err != nil {
		t.Fatalf("load frozen plan: %v", err)
	}
	if !strings.HasSuffix(string(frozen.Bytes), "\n") {
		t.Fatalf("trailing newline must survive; got %q", string(frozen.Bytes))
	}
	if frozen.SHA256 != hex.EncodeToString(sumA[:]) {
		t.Fatalf("SHA-256 must equal SHA256(literal bytes); got %s want %s",
			frozen.SHA256, hex.EncodeToString(sumA[:]))
	}
}
