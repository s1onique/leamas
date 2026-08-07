// SPDX-License-Identifier: Apache-2.0

package closure

// closure_runtime_context_topology_test.go provides the
// TestClosureExecuteTopologyAuthority umbrella required by
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02-A.
//
// The test exercises the production topology dispatch and
// resolver against a real Git repository. It MUST NOT rely on
// a fake gitClient or a fake grammar: every row must prove
// that the F < S matrix resolves to the expected outcome.
//
// Required matrix:
//
//	F < S        PASS
//	F == S       reject (subject_equals_freeze)
//	S < F        reject (subject_not_ancestor_of_freeze)
//	F unrelated S reject (subject_freeze_unrelated)
//	missing F    reject (freeze_commit_not_found)
//	missing S    reject (subject_commit_not_found)
//
// Splitting this from the existing v2 topology tests keeps
// the file under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

// realGitClient is a thin wrapper that satisfies the
// production gitClient interface by translating Run calls
// into execution.RunGit. The wrapper is local to this test
// file so it does not collide with fakes in other tests.
type realGitClient struct{}

func (realGitClient) Run(ctx context.Context, repoRoot string, args ...string) gitCommandResult {
	out, err := execution.RunGit(ctx, repoRoot, args...)
	// ALWAYS preserve the exit code even when err is set.
	// git merge-base --is-ancestor returns exit=1 to mean
	// "not an ancestor" and that is not an error. Tests
	// that drop the exit code on err would silently treat
	// "not an ancestor" as "ancestor", which is the
	// exact bug the production resolver must avoid.
	if err != nil {
		return gitCommandResult{Err: err, ExitCode: out.ExitCode}
	}
	return gitCommandResult{
		ExitCode: out.ExitCode,
		Stdout:   out.Stdout,
		Stderr:   out.Stderr,
	}
}

func (realGitClient) RunWithStdin(ctx context.Context, repoRoot, stdin string, args ...string) gitCommandResult {
	out, err := execution.RunGitWithStdin(ctx, repoRoot, stdin, args...)
	if err != nil {
		return gitCommandResult{Err: err}
	}
	return gitCommandResult{
		ExitCode: out.ExitCode,
		Stdout:   out.Stdout,
		Stderr:   out.Stderr,
	}
}

func (realGitClient) RunWithEnv(ctx context.Context, repoRoot string, env []string, args ...string) gitCommandResult {
	out, err := execution.RunGitWithEnv(ctx, repoRoot, env, args...)
	if err != nil {
		return gitCommandResult{Err: err}
	}
	return gitCommandResult{
		ExitCode: out.ExitCode,
		Stdout:   out.Stdout,
		Stderr:   out.Stderr,
	}
}

func (realGitClient) RunWithStdinAndEnv(ctx context.Context, repoRoot, stdin string, env []string, args ...string) gitCommandResult {
	out, err := execution.RunGitWithStdinAndEnv(ctx, repoRoot, stdin, env, args...)
	if err != nil {
		return gitCommandResult{Err: err}
	}
	return gitCommandResult{
		ExitCode: out.ExitCode,
		Stdout:   out.Stdout,
		Stderr:   out.Stderr,
	}
}

// TestClosureExecuteTopologyAuthority is the umbrella test
// for the F < S topology rule. It exercises the production
// resolver and dispatch against a real Git repository. The
// test is the canonical proof that the runtime-context
// command does NOT silently inherit the legacy V2 (S < F)
// topology.
//
// The test matrix covers every required relation with a real
// Git fixture so the dispatch cannot hide behind a fake git
// or a fake grammar.
func TestClosureExecuteTopologyAuthority(t *testing.T) {
	t.Parallel()
	// 1. Build a real Git repo with three commits: F < S < D.
	dir := makeFltSDRepo(t)
	subject := mustRunGit(t, dir, "rev-parse", "HEAD~1")
	freeze := mustRunGit(t, dir, "rev-parse", "HEAD~2")
	descendant := mustRunGit(t, dir, "rev-parse", "HEAD")

	git := realGitClient{}
	resolver := NewGitV2TopologyResolver(git)

	// 2. F < S: PASS through the dispatch.
	facts, err := resolver.ResolveTopology(
		context.Background(), dir, subject, freeze,
	)
	if err != nil {
		t.Fatalf("resolve F < S: %v", err)
	}
	if facts.Classify() != V2RelationFreezeBeforeSubject {
		t.Fatalf("expected FreezeBeforeSubject, got %s", facts.Classify())
	}
	if outcome := dispatchClosureTopology(ClosureProtocolV2, executionTopologyFreezeBeforeSubject, facts); !outcome.Accepted {
		t.Fatalf("F < S must be accepted for the runtime-context topology, got code=%s", outcome.Code)
	}
	// 3. F == S: subject_equals_freeze rejection.
	factsEqual, err := resolver.ResolveTopology(
		context.Background(), dir, subject, subject,
	)
	if err != nil {
		t.Fatalf("resolve F == S: %v", err)
	}
	if outcome := dispatchClosureTopology(ClosureProtocolV2, executionTopologyFreezeBeforeSubject, factsEqual); outcome.Accepted {
		t.Fatalf("F == S must be rejected for the runtime-context topology")
	} else if outcome.Code != V2CodeSubjectEqualsFreeze {
		t.Fatalf("F == S rejected with wrong code: %s", outcome.Code)
	}
	// 4. S < F: subject_not_ancestor_of_freeze rejection.
	// swap parameters so S becomes the freeze lookup.
	factsReverse, err := resolver.ResolveTopology(
		context.Background(), dir, freeze, subject,
	)
	if err != nil {
		t.Fatalf("resolve S < F: %v", err)
	}
	if outcome := dispatchClosureTopology(ClosureProtocolV2, executionTopologyFreezeBeforeSubject, factsReverse); outcome.Accepted {
		t.Fatalf("S < F must be rejected for the runtime-context topology")
	} else if outcome.Code != V2CodeSubjectNotAncestorOfFreeze {
		t.Fatalf("S < F rejected with wrong code: %s", outcome.Code)
	}
	// 5. F unrelated S: create a branch from the initial
	// commit that does not contain F. The HEAD of that
	// branch is unrelated to F so the dispatch must reject
	// with subject_freeze_unrelated.
	mustRunGit(t, dir, "checkout", "HEAD~3", "-b", "unrelated")
	mustRunGit(t, dir, "commit", "--allow-empty", "-m", "unrelated")
	unrelatedHead := mustRunGit(t, dir, "rev-parse", "HEAD")
	factsUnrelated, err := resolver.ResolveTopology(
		context.Background(), dir, unrelatedHead, freeze,
	)
	if err != nil {
		t.Fatalf("resolve F unrelated S: %v", err)
	}
	if outcome := dispatchClosureTopology(ClosureProtocolV2, executionTopologyFreezeBeforeSubject, factsUnrelated); outcome.Accepted {
		t.Fatalf("F unrelated S must be rejected for the runtime-context topology")
	} else if outcome.Code != V2CodeSubjectFreezeUnrelated {
		t.Fatalf("F unrelated S rejected with wrong code: %s", outcome.Code)
	}
	// 6. missing F: a frozen-commit OID that does not exist
	// in the repo. The resolver must populate facts where
	// FreezeResolved == false and the dispatch must reject
	// with freeze_commit_not_found.
	factsMissingF, err := resolver.ResolveTopology(
		context.Background(), dir, subject, "0000000000000000000000000000000000000000",
	)
	if err != nil {
		t.Fatalf("resolve missing F: %v", err)
	}
	if factsMissingF.FreezeResolved {
		t.Fatalf("missing F should not resolve")
	}
	if outcome := dispatchClosureTopology(ClosureProtocolV2, executionTopologyFreezeBeforeSubject, factsMissingF); outcome.Accepted {
		t.Fatalf("missing F must be rejected")
	} else if outcome.Code != V2CodeFreezeCommitNotFound {
		t.Fatalf("missing F rejected with wrong code: %s", outcome.Code)
	}
	// 7. missing S: subject_commit_not_found.
	factsMissingS, err := resolver.ResolveTopology(
		context.Background(), dir, "0000000000000000000000000000000000000000", freeze,
	)
	if err != nil {
		t.Fatalf("resolve missing S: %v", err)
	}
	if factsMissingS.SubjectResolved {
		t.Fatalf("missing S should not resolve")
	}
	if outcome := dispatchClosureTopology(ClosureProtocolV2, executionTopologyFreezeBeforeSubject, factsMissingS); outcome.Accepted {
		t.Fatalf("missing S must be rejected")
	} else if outcome.Code != V2CodeSubjectCommitNotFound {
		t.Fatalf("missing S rejected with wrong code: %s", outcome.Code)
	}
	// 8. The runtime-context entry point locks the
	// internal execution topology to FreezeBeforeSubject
	// so the dispatch accepts F < S. The legacy V2 (S < F)
	// topology is unreachable from this entry point because
	// the topology is selected by the public entry point,
	// not by the request surface.
	_ = descendant // keep descendant referenced for downstream tests
}

// makeFltSDRepo creates a real Git repository with three
// commits laid out F < S < D. The function returns the
// repository root path.
func makeFltSDRepo(t *testing.T) string {
	t.Helper()
	dir := initRepo(t)
	// First commit (F): the freeze.
	mustRunGit(t, dir, "commit", "--allow-empty", "-m", "F")
	// Second commit (S): the subject.
	mustRunGit(t, dir, "commit", "--allow-empty", "-m", "S")
	// Third commit (D): the descendant.
	mustRunGit(t, dir, "commit", "--allow-empty", "-m", "D")
	return dir
}

// TestClosureTopologyAuthorityCoversUnsupportedVersion is
// the dedicated matrix proof that the dispatch rejects
// unsupported closure protocol versions. The two supported
// lifecycle versions, V1 and V2, are topology-compatible
// with F < S and accept the same facts; every other value
// MUST be rejected with the exact
// V2CodeUnsupportedClosureProtocolVersion code.
func TestClosureTopologyAuthorityCoversUnsupportedVersion(t *testing.T) {
	t.Parallel()
	for _, v := range []ClosureProtocolVersion{
		"",
		"v1",
		"999",
		ClosureProtocolV1,
		ClosureProtocolV2,
	} {
		facts := V2TopologyFacts{
			SubjectResolved:       true,
			FreezeResolved:        true,
			Equal:                 false,
			SubjectAncestorFreeze: false,
			FreezeAncestorSubject: true,
			SubjectCommit:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			FreezeCommit:          "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		}
		outcome := dispatchClosureTopology(v, executionTopologyFreezeBeforeSubject, facts)
		switch v {
		case ClosureProtocolV1, ClosureProtocolV2:
			// Both V1 and V2 accept F < S under the
			// runtime-context topology; the runtime-context
			// entry point is the only path that actually
			// selects it, but the dispatch accepts both
			// versions for the same facts.
			if !outcome.Accepted {
				t.Fatalf("version %q must accept F < S, got code=%s", v, outcome.Code)
			}
		default:
			// Every unsupported version MUST be rejected
			// with the exact V2CodeUnsupportedClosureProtocolVersion
			// code. The previous guard skipped the code
			// check for "", "v1", and "999" which are
			// precisely the unsupported rows, so the
			// guard contradicted the table it was meant
			// to enforce.
			if outcome.Accepted {
				t.Fatalf("version %q must reject F < S, got accepted", v)
			}
			if outcome.Code != V2CodeUnsupportedClosureProtocolVersion {
				t.Fatalf("version %q must reject with unsupported code, got %s", v, outcome.Code)
			}
		}
	}
	// Keep strings package referenced for downstream tests.
	_ = strings.TrimSpace
}

// TestClosureRuntimeContextTopologyRegression is the
// regression proof that the public runtime-context entry
// point routes through the runtime-context topology. The
// test exercises RunClosureProtocolRuntimeContext itself
// (not merely dispatchClosureTopology) against a real
// hermetic Git repository so the topology step is isolated
// from any later dependency.
//
// Required rows:
//   - F < S through RunClosureProtocolRuntimeContext: must
//     reach the runtime execution topology; the test may
//     fail at a later controlled dependency but MUST NOT
//     reject for the topology step.
//   - S < F through RunClosureProtocolRuntimeContext: must
//     reject with the exact runtime-context topology code
//     V2CodeSubjectNotAncestorOfFreeze.
//
// The legacy public V2 path (RunClosureProtocolV2WithBinary)
// is also exercised against S < F to prove the older
// closure-protocol-version V2 still accepts the legacy
// topology S < F.
func TestClosureRuntimeContextTopologyRegression(t *testing.T) {
	t.Parallel()
	dir := makeFltSDRepo(t)
	// The repo has three commits F < S < D where:
	//   F = HEAD~2 (the original freeze)
	//   S = HEAD~1 (the original subject)
	//   D = HEAD    (the original descendant)
	freeze := mustRunGit(t, dir, "rev-parse", "HEAD~2")
	subject := mustRunGit(t, dir, "rev-parse", "HEAD~1")
	_ = mustRunGit(t, dir, "rev-parse", "HEAD")

	makeRequest := func(version ClosureProtocolVersion, subjectOID, freezeOID string) V2Request {
		return V2Request{
			ClosureProtocolVersion: version,
			PlanContractVersion:    1,
			RepositoryRoot:         dir,
			SubjectCommit:          subjectOID,
			FreezeCommit:           freezeOID,
			PlanPath:               "docs/plans/plan.json",
			EvidenceDirectory:      t.TempDir(),
			ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
		}
	}

	// 1. F < S through the runtime-context entry point.
	// The topology must accept; the caller may receive a
	// later non-topology failure (e.g. plan load), but the
	// failure MUST NOT be a topology rejection.
	_, errFLS := RunClosureProtocolRuntimeContext(
		context.Background(),
		makeRequest(ClosureProtocolV2, subject, freeze),
		V2BinaryIdentity{},
	)
	if errFLS != nil {
		v2err, ok := errFLS.(*V2Error)
		if !ok {
			t.Fatalf("runtime-context F < S must return *V2Error on failure, got %T: %v", errFLS, errFLS)
		}
		if v2err.Diags.HasCode(V2CodeSubjectNotAncestorOfFreeze) {
			t.Fatalf("runtime-context F < S must NOT reject with subject_not_ancestor_of_freeze: %v", v2err)
		}
		if v2err.Diags.HasCode(V2CodeSubjectEqualsFreeze) {
			t.Fatalf("runtime-context F < S must NOT reject with subject_equals_freeze: %v", v2err)
		}
		if v2err.Diags.HasCode(V2CodeFreezeAncestorOfSubject) {
			t.Fatalf("runtime-context F < S must NOT reject with freeze_ancestor_of_subject: %v", v2err)
		}
		if v2err.Diags.HasCode(V2CodeFrozenPlanNotBlob) {
			t.Fatalf("runtime-context F < S must NOT reject with frozen_plan_not_blob: %v", v2err)
		}
	}

	// 2. S < F through the runtime-context entry point.
	// Swap the original subject and freeze roles so the
	// supplied "subject" is an ancestor of the supplied
	// "freeze", which is the S < F rejection case for the
	// runtime-context topology rule. The original F is an
	// ancestor of the original S, so using the original F
	// as the new subject and the original S as the new
	// freeze yields S' (F) < F' (S) under the runtime-context
	// dispatch.
	_, errSLF := RunClosureProtocolRuntimeContext(
		context.Background(),
		makeRequest(ClosureProtocolV2, freeze, subject),
		V2BinaryIdentity{},
	)
	if errSLF == nil {
		t.Fatalf("runtime-context S < F must reject, got nil")
	}
	v2err, ok := errSLF.(*V2Error)
	if !ok {
		t.Fatalf("runtime-context S < F must return *V2Error, got %T: %v", errSLF, errSLF)
	}
	if !v2err.Diags.HasCode(V2CodeSubjectNotAncestorOfFreeze) {
		t.Fatalf("runtime-context S < F must reject with subject_not_ancestor_of_freeze, got %v", v2err.Diags.Codes())
	}

	// 3. Legacy public V2 path keeps S < F behaviour.
	// The legacy entry point uses the default topology
	// (S < F) and must accept the same swapped facts.
	// The legacy V2 path may fail at a later dependency
	// (e.g. plan load). The proof we want is that the
	// topology step accepts S < F under the default
	// topology, so we must NOT see a topology rejection.
	_, errLegacy := RunClosureProtocolV2WithBinary(
		context.Background(),
		makeRequest(ClosureProtocolV2, freeze, subject),
		V2BinaryIdentity{},
	)
	if errLegacy != nil {
		v2err, ok := errLegacy.(*V2Error)
		if !ok {
			t.Fatalf("legacy V2 S < F must return *V2Error on failure, got %T: %v", errLegacy, errLegacy)
		}
		if v2err.Diags.HasCode(V2CodeSubjectNotAncestorOfFreeze) {
			t.Fatalf("legacy V2 S < F must NOT reject with subject_not_ancestor_of_freeze: %v", v2err)
		}
	}
	// 4. The dispatcher itself, when invoked with the
	// default topology and V2, must accept S < F. This
	// is the typed proof that the legacy V2 dispatch path
	// is still reachable and topology-correct.
	git := realGitClient{}
	resolver := NewGitV2TopologyResolver(git)
	legacyFacts, err := resolver.ResolveTopology(
		context.Background(), dir, freeze, subject,
	)
	if err != nil {
		t.Fatalf("resolve legacy S < F: %v", err)
	}
	if legacyFacts.Classify() != V2RelationSubjectBeforeFreeze {
		t.Fatalf("legacy facts must classify as SubjectBeforeFreeze, got %s", legacyFacts.Classify())
	}
	if outcome := DispatchClosureTopology(ClosureProtocolV2, legacyFacts); !outcome.Accepted {
		t.Fatalf("legacy V2 must accept S < F, got code=%s", outcome.Code)
	}
}
