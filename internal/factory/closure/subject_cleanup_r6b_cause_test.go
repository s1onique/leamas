// SPDX-License-Identifier: Apache-2.0

// subject_cleanup_r6b_cause_test.go owns the CORRECTION09
// R6-B cause-preservation tests. The tests prove the
// R6-B wrapper preserves the ACTUAL executor error as the
// typed V2CodeR6BSubjectCleanupFailed Cause, rather than
// synthesising a replacement from result.SubjectCleanupError.
//
// The R6-A direct-caller contract (populated result + non-
// nil V2CodeCleanupFailed) is exercised in
// subject_cleanup_contract_test.go. The R6-B wrapper must
// NOT collapse that chain into a string-reconstructed
// stub; the cause MUST be the actual executor error so
// downstream errors.Is / errors.As inspection reaches the
// owning lower-level codes.
//
// Splitting this from subject_cleanup_contract_test.go keeps
// the direct-contract file focused on the R6-A authority
// while the R6-B wrapper contract lives in its own file.
// Both files stay under the LLM-friendly 400-line threshold.

package closure

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// r6BPrimaryAndCleanupFakeGit is the CORRECTION09
// primary+cleanup fake git client. It returns a deliberately
// wrong tree for the subject commit's tree resolution (so
// the R6-A executor's observed-tree mismatch fires) and
// fails `git worktree remove --force` (so the cleanup
// authority surfaces V2CodeCleanupFailed). All other
// commands are delegated to the supplied git client
// (typically RealGit) so the topology, commit, and freeze-
// tree resolution paths remain grounded in real git output.
//
// The wrong-tree branch uses a configurable OID prefix to
// identify the freeze commit: the freeze-tree resolution
// MUST pass through to the delegate, otherwise the manifest
// builder would reject the run before the executor runs.
type r6BPrimaryAndCleanupFakeGit struct {
	delegate      gitClient
	freezeOID     string
	wrongTree     string
	cleanupStderr string
}

func (m *r6BPrimaryAndCleanupFakeGit) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	// Subject tree override: rev-parse --verify --end-of-options <commit>^{tree}
	// for the subject commit (NOT the freeze commit). The
	// executor's observed-tree check then fails with
	// V2CodeExecutionTreeMismatch.
	if len(args) >= 4 && args[0] == "rev-parse" && args[1] == "--verify" &&
		args[2] == "--end-of-options" && strings.HasSuffix(args[3], "^{tree}") {
		if m.freezeOID != "" && strings.HasPrefix(args[3], m.freezeOID) {
			// Freeze tree: delegate to real git so the
			// manifest builder accepts the freeze identity.
			if m.delegate == nil {
				return gitCommandResult{Err: errors.New("r6b primary+cleanup fake: no delegate")}
			}
			return m.delegate.Run(ctx, directory, args...)
		}
		return gitCommandResult{
			ExitCode: 0,
			Stdout:   []byte(m.wrongTree + "\n"),
		}
	}
	// Cleanup failure: worktree remove --force.
	if len(args) >= 4 && args[0] == "worktree" && args[1] == "remove" {
		return gitCommandResult{
			ExitCode: 1,
			Stderr:   []byte(m.cleanupStderr),
		}
	}
	if m.delegate == nil {
		return gitCommandResult{Err: errors.New("r6b primary+cleanup fake: no delegate")}
	}
	return m.delegate.Run(ctx, directory, args...)
}

func (m *r6BPrimaryAndCleanupFakeGit) RunWithStdin(ctx context.Context, directory, stdin string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithStdin(ctx, directory, stdin, args...)
}

func (m *r6BPrimaryAndCleanupFakeGit) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithEnv(ctx, directory, env, args...)
}

func (m *r6BPrimaryAndCleanupFakeGit) RunWithStdinAndEnv(ctx context.Context, directory, stdin string, env []string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithStdinAndEnv(ctx, directory, stdin, env, args...)
}

// r6bCauseV2 walks past the R6-B outer V2Error to the
// actual executor err. The helper exists because
// errors.As(err, &v2) on the R6-B wrapped err returns the
// OUTER V2Error (the wrapper's own), not the cause. The
// test assertions need the CAUSE so they can prove the
// wrapper did not synthesise a replacement.
func r6bCauseV2(t *testing.T, err error) *V2Error {
	t.Helper()
	outer := unwrapV2Error(err)
	if outer == nil {
		t.Fatalf("outer must be a *V2Error, got %T (%v)", err, err)
	}
	cause := errors.Unwrap(outer)
	if cause == nil {
		t.Fatalf("outer V2Error has nil Cause; the R6-B wrapper MUST wrap the actual executor err")
	}
	var causeV2 *V2Error
	if !errors.As(cause, &causeV2) {
		t.Fatalf("cause must be a *V2Error (reachable via errors.As), got %T (%v)", cause, cause)
	}
	return causeV2
}

// TestR6BSubjectCleanupCausePreservesInnerErr proves the
// CORRECTION09 fix: when the R6-A executor returns a
// non-nil err with a cleanup failure, the R6-B wrapper
// preserves the actual executor err as V2Error.Cause. The
// test drives the production integration end-to-end with
// the cleanup-failure git seam and asserts the cause
// chain contains the lower-level V2CodeCleanupFailed code.
//
// Required chain:
//
//	outer: V2CodeR6BSubjectCleanupFailed
//	cause: actual executor error
//	contained lower-level code: V2CodeCleanupFailed
//
// Assertions use typed inspection (errors.As,
// v2ErrorContainsCode) — not error-string reconstruction.
func TestR6BSubjectCleanupCausePreservesInnerErr(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	runner := &r6BRecordingRunner{stdoutField: []byte("EXEC_GATE_OBSERVED_STATUS:OK\n")}
	collector := evidence.NewGateCollector(runner)
	// Subject cleanup authority: real git except the
	// worktree remove call, which fails deterministically.
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-cause-cleanup-only",
			EvidenceDir:    r6BEvidenceDir(t),
			GitClient:      r6BRealSubjectCleanupFailureGitClient(),
		})
	if err == nil {
		t.Fatalf("cleanup failure must surface a typed error")
	}
	requireV2Code(t, err, V2CodeR6BSubjectCleanupFailed)
	// Outer must be the R6-B wrapper code, NOT a
	// synthesised cleanup-only stub.
	outer := unwrapV2Error(err)
	if outer == nil {
		t.Fatalf("outer must be a *V2Error, got %T (%v)", err, err)
	}
	if len(outer.Diags) == 0 || outer.Diags[0].Code != V2CodeR6BSubjectCleanupFailed {
		t.Fatalf("outer first code = %v, want V2CodeR6BSubjectCleanupFailed",
			outer.Diags.Codes())
	}
	// The cause MUST be the actual executor err. The
	// executor returns V2CodeCleanupFailed (from the
	// cleanup-only failure path). The cause's first code
	// is the executor's code, NOT the R6-B wrapper code.
	causeV2 := r6bCauseV2(t, err)
	if !v2ErrorContainsCode(causeV2, V2CodeCleanupFailed) {
		t.Fatalf("cause must contain V2CodeCleanupFailed, got diags=%v",
			causeV2.Diags.Codes())
	}
	if causeV2.Diags[0].Code == V2CodeR6BSubjectCleanupFailed {
		t.Fatalf("cause first code = V2CodeR6BSubjectCleanupFailed; the R6-B wrapper MUST wrap the actual executor err, not itself")
	}
}

// TestR6BPrimaryAndCleanupCauseChainPreserved proves the
// R6-B wrapper preserves BOTH a primary failure code AND
// the cleanup code in the cause chain when the underlying
// executor returns a primary+cleanup err. The test uses
// a custom git client that returns a wrong subject tree
// (triggering V2CodeExecutionTreeMismatch as the primary
// failure) AND fails the worktree remove (triggering
// V2CodeCleanupFailed). The R6-A executor's wrapWithCleanup
// appends the cleanup diagnostic to the primary err, so
// the actual err has both codes. The R6-B wrapper must
// preserve both in the cause chain.
//
// Required:
//
//	outer code: V2CodeR6BSubjectCleanupFailed
//	cause chain still contains:
//	  V2CodeExecutionTreeMismatch (primary)
//	  V2CodeCleanupFailed (cleanup)
func TestR6BPrimaryAndCleanupCauseChainPreserved(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	// Wrong subject tree: 40 zero hex chars (a valid OID
	// shape but not the real subject tree).
	wrongTree := strings.Repeat("0", 40)
	// The cleanup stderr is the report-summary text the
	// executor embeds in the cleanup diagnostic. The test
	// asserts the cause carries BOTH the primary tree-
	// mismatch detail and the cleanup report summary.
	cleanupStderr := "r6b-cause-primary-cleanup: deterministic cleanup failure"
	fakeGit := &r6BPrimaryAndCleanupFakeGit{
		delegate:      RealGit{},
		freezeOID:     freeze,
		wrongTree:     wrongTree,
		cleanupStderr: cleanupStderr,
	}
	runner := &r6BRecordingRunner{stdoutField: []byte("EXEC_GATE_OBSERVED_STATUS:OK\n")}
	collector := evidence.NewGateCollector(runner)
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-cause-primary-cleanup",
			EvidenceDir:    r6BEvidenceDir(t),
			GitClient:      fakeGit,
		})
	if err == nil {
		t.Fatalf("primary + cleanup failure must surface a typed error")
	}
	requireV2Code(t, err, V2CodeR6BSubjectCleanupFailed)
	// Cause chain (walking outer + cause) must contain BOTH codes.
	if !v2ErrorContainsCode(err, V2CodeExecutionTreeMismatch) {
		t.Fatalf("cause chain must contain V2CodeExecutionTreeMismatch (primary), got full err=%v", err)
	}
	if !v2ErrorContainsCode(err, V2CodeCleanupFailed) {
		t.Fatalf("cause chain must contain V2CodeCleanupFailed (cleanup), got full err=%v", err)
	}
	// The cause is the actual executor err (reachable
	// via errors.As *V2Error), NOT a string-reconstructed
	// stub. The actual err carries the primary detail
	// ("observed tree ... does not match subject tree ...").
	// A synthesised replacement built from the cleanup
	// string alone would NOT contain this detail.
	causeV2 := r6bCauseV2(t, err)
	primaryDetailFound := false
	for _, d := range causeV2.Diags {
		if d.Code == V2CodeExecutionTreeMismatch && strings.Contains(d.Message, "observed tree") {
			primaryDetailFound = true
			break
		}
	}
	if !primaryDetailFound {
		t.Fatalf("cause's primary diagnostic must contain the executor's actual 'observed tree' detail; diags=%v",
			causeV2.Diags.Codes())
	}
}

// TestR6BNoCauseReconstructionWhenErrExists is the
// CORRECTION09 regression guard. It proves the
// ERROR_OBJECT_AUTHORITY > STRING_RECONSTRUCTION rule:
// when the R6-A executor returns a non-nil err AND the
// result.SubjectCleanupError is populated, the R6-B
// wrapper uses the actual err as Cause — NOT a
// string-reconstructed replacement built from
// result.SubjectCleanupError.
//
// The test sets:
//
//	result.SubjectCleanupError = "r6b-cause-no-reconstruct: text-A"
//
//	actual executor err = primary + cleanup, the primary
//	  diagnostic's message contains "observed tree" (the
//	  executor's authority detail)
//
// The assertion: the cause's first diagnostic message
// must contain the executor's authority detail
// ("observed tree"), proving the cause is the actual
// executor err. A synthesised replacement would carry
// ONLY the cleanup string.
func TestR6BNoCauseReconstructionWhenErrExists(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	wrongTree := strings.Repeat("0", 40)
	// The cleanup stderr IS the string a synthesised
	// replacement would use. The test proves the cause
	// carries the executor's actual primary detail
	// instead.
	cleanupStderr := "r6b-cause-no-reconstruct: text-A"
	fakeGit := &r6BPrimaryAndCleanupFakeGit{
		delegate:      RealGit{},
		freezeOID:     freeze,
		wrongTree:     wrongTree,
		cleanupStderr: cleanupStderr,
	}
	runner := &r6BRecordingRunner{stdoutField: []byte("EXEC_GATE_OBSERVED_STATUS:OK\n")}
	collector := evidence.NewGateCollector(runner)
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-cause-no-reconstruct",
			EvidenceDir:    r6BEvidenceDir(t),
			GitClient:      fakeGit,
		})
	if err == nil {
		t.Fatalf("primary + cleanup failure must surface a typed error")
	}
	requireV2Code(t, err, V2CodeR6BSubjectCleanupFailed)
	// The cause must be the actual executor err. The
	// proof: the cause's primary diagnostic (the first
	// non-cleanup diagnostic the executor built) carries
	// the "observed tree" detail. A synthesised
	// replacement built from the cleanup string alone
	// would NOT carry this detail.
	causeV2 := r6bCauseV2(t, err)
	hasPrimaryDetail := false
	for _, d := range causeV2.Diags {
		if d.Code == V2CodeExecutionTreeMismatch && strings.Contains(d.Message, "observed tree") {
			hasPrimaryDetail = true
			break
		}
	}
	if !hasPrimaryDetail {
		t.Fatalf("cause must carry the executor's actual primary detail ('observed tree'); the R6-B wrapper MUST NOT reconstruct from result.SubjectCleanupError; got diags=%v",
			causeV2.Diags.Codes())
	}
	// The cause must also carry the cleanup code, proving
	// the R6-A executor's wrapWithCleanup produced both
	// diagnostics and the R6-B wrapper preserved the full
	// err.
	if !v2ErrorContainsCode(causeV2, V2CodeCleanupFailed) {
		t.Fatalf("cause must also carry V2CodeCleanupFailed; got diags=%v",
			causeV2.Diags.Codes())
	}
}
