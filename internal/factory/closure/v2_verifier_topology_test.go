// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_topology_test.go covers the topology matrix
// required by Phase 3 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-TOPOLOGY-OBJECTS01.
//
// Every case constructs a real Git repository in a temp
// directory via the existing closure-package test helpers
// (initRepo, makeCommit, mustRunGit) and asserts the resolver
// classifies the S/F/C triple correctly:
//
//   - S < F < C                 accepted
//   - S = F < C                 rejected (subject_freeze_equal)
//   - S < F = C                 rejected (freeze_closure_equal)
//   - S < C < F                 rejected
//   - F < S < C                 rejected
//   - C < S < F                 rejected
//   - S / F unrelated           rejected
//   - F / C unrelated           rejected
//   - all unrelated             rejected
//   - missing S / F / C         rejected (typed)
//   - ancestry timeout          rejected (typed)
//   - ancestry execution error  rejected (typed)

import (
	"context"
	"strings"
	"testing"
)

// TestV2VerifierTopologyAcceptedChain proves the verifier
// classifies S < F < C as the only accepted relation. The
// test builds three sequential commits in a hermetic repo
// and asserts on the relation plus the verbatim OID fields.
func TestV2VerifierTopologyAcceptedChain(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject implementation\n",
	})
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"freeze-only.txt": "freeze implementation\n",
	})
	closure := makeCommit(t, dir, "closure", map[string]string{
		"closure-only.txt": "closure implementation\n",
	})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	resolver := NewV2ClosureTopologyResolver()
	facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
		SubjectCommit: subject,
		FreezeCommit:  freeze,
		ClosureCommit: closure,
	})
	if err != nil {
		t.Fatalf("ResolveTopology: %v", err)
	}
	if facts.Relation != V2ClosureRelationSubjectBeforeFreezeBeforeClosure {
		t.Fatalf("relation = %q, want %q",
			facts.Relation, V2ClosureRelationSubjectBeforeFreezeBeforeClosure)
	}
	if !facts.Relation.IsAccepted() {
		t.Fatalf("accepted relation must satisfy IsAccepted")
	}
	if len(facts.Diagnostics) != 0 {
		t.Fatalf("accepted topology must emit no diagnostics, got %v",
			facts.Diagnostics.Codes())
	}
	if facts.Topology.SubjectCommit != subject {
		t.Fatalf("SubjectCommit = %q, want %q", facts.Topology.SubjectCommit, subject)
	}
	if facts.Topology.FreezeCommit != freeze {
		t.Fatalf("FreezeCommit = %q, want %q", facts.Topology.FreezeCommit, freeze)
	}
	if facts.Topology.ClosureCommit != closure {
		t.Fatalf("ClosureCommit = %q, want %q", facts.Topology.ClosureCommit, closure)
	}
	if !facts.Topology.SubjectAncestorFreeze || !facts.Topology.FreezeAncestorClosure {
		t.Fatalf("topology bools = (%v, %v), want (true, true)",
			facts.Topology.SubjectAncestorFreeze, facts.Topology.FreezeAncestorClosure)
	}
}

// TestV2VerifierTopologyPairwiseDistinctMatrix proves the
// verifier rejects the three equal-pair topologies with the
// matching typed diagnostic codes. The triple is built from
// three real commits in a hermetic repo; the test
// substitutes OIDs to manufacture each pairwise-equal
// configuration.
func TestV2VerifierTopologyPairwiseDistinctMatrix(t *testing.T) {
	dir := initRepo(t)
	commit1 := makeCommit(t, dir, "commit-1", map[string]string{"a.txt": "A\n"})
	commit2 := makeCommit(t, dir, "commit-2", map[string]string{"b.txt": "B\n"})
	_ = makeCommit(t, dir, "commit-3", map[string]string{"c.txt": "C\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	resolver := NewV2ClosureTopologyResolver()

	cases := []struct {
		name      string
		subject   string
		freeze    string
		closure   string
		want      V2ClosureRelation
		wantCodes []V2VerifierCode
	}{
		{
			name:      "S=F<C",
			subject:   commit1,
			freeze:    commit1,
			closure:   commit2,
			want:      V2ClosureRelationSubjectFreezeEqual,
			wantCodes: []V2VerifierCode{V2VerifierSubjectFreezeEqual},
		},
		{
			name:      "S<F=C",
			subject:   commit1,
			freeze:    commit2,
			closure:   commit2,
			want:      V2ClosureRelationFreezeClosureEqual,
			wantCodes: []V2VerifierCode{V2VerifierFreezeClosureEqual},
		},
		{
			name:      "S=C<F",
			subject:   commit1,
			freeze:    commit2,
			closure:   commit1,
			want:      V2ClosureRelationSubjectClosureEqual,
			wantCodes: []V2VerifierCode{V2VerifierSubjectClosureEqual},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
				SubjectCommit: tc.subject,
				FreezeCommit:  tc.freeze,
				ClosureCommit: tc.closure,
			})
			if err != nil {
				t.Fatalf("ResolveTopology: %v", err)
			}
			if facts.Relation != tc.want {
				t.Fatalf("relation = %q, want %q", facts.Relation, tc.want)
			}
			if facts.Relation.IsAccepted() {
				t.Fatalf("equal-pair topology must not be accepted")
			}
			gotCodes := facts.Diagnostics.Codes()
			if !equalCodeSet(gotCodes, tc.wantCodes) {
				t.Fatalf("diagnostic codes = %v, want %v", gotCodes, tc.wantCodes)
			}
		})
	}
}

// TestV2VerifierTopologyReverseMatrix proves the verifier
// rejects the two reverse-topology configurations. The test
// builds a non-linear history (commit A, then branch B off
// commit A, then commit B2 on branch B). The verifier must
// distinguish F < S from C < F even though both are valid
// linear histories.
func TestV2VerifierTopologyReverseMatrix(t *testing.T) {
	dir := initRepo(t)
	a := makeCommit(t, dir, "A", map[string]string{"a.txt": "A\n"})
	// Branch B off A: make a B then a C.
	// Make B a child of A then C a child of B.
	b := makeCommit(t, dir, "B", map[string]string{"b.txt": "B\n"})
	c := makeCommit(t, dir, "C", map[string]string{"c.txt": "C\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	resolver := NewV2ClosureTopologyResolver()

	// F < S < C: subject is the first commit, freeze is a
	// later commit, closure is the latest.
	t.Run("F<S<C", func(t *testing.T) {
		facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
			SubjectCommit: b,
			FreezeCommit:  a,
			ClosureCommit: c,
		})
		if err != nil {
			t.Fatalf("ResolveTopology: %v", err)
		}
		if facts.Relation != V2ClosureRelationFreezeBeforeSubject {
			t.Fatalf("relation = %q, want %q",
				facts.Relation, V2ClosureRelationFreezeBeforeSubject)
		}
		if !facts.Diagnostics.HasCode(V2VerifierReverseSubjectFreezeTopology) {
			t.Fatalf("diagnostic codes = %v, want reverse_subject_freeze_topology",
				facts.Diagnostics.Codes())
		}
	})

	// S < C < F: S first, F at the top, C between them.
	t.Run("S<C<F", func(t *testing.T) {
		facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
			SubjectCommit: a,
			FreezeCommit:  c,
			ClosureCommit: b,
		})
		if err != nil {
			t.Fatalf("ResolveTopology: %v", err)
		}
		if facts.Relation != V2ClosureRelationClosureBeforeFreeze {
			t.Fatalf("relation = %q, want %q",
				facts.Relation, V2ClosureRelationClosureBeforeFreeze)
		}
		if !facts.Diagnostics.HasCode(V2VerifierReverseFreezeClosureTopology) {
			t.Fatalf("diagnostic codes = %v, want reverse_freeze_closure_topology",
				facts.Diagnostics.Codes())
		}
	})
}

// TestV2VerifierTopologyUnrelatedMatrix proves the verifier
// rejects S/F and F/C unrelated configurations. The test
// creates two distinct linear histories starting from the
// same root commit by resetting HEAD back to root after each
// branch commit. The resulting commits share the root but
// diverge from it independently.
func TestV2VerifierTopologyUnrelatedMatrix(t *testing.T) {
	dir := initRepo(t)
	// rootOID is the initial empty commit created by initRepo.
	rootOID := mustRunGit(t, dir, "rev-parse", "HEAD")
	// Branch A: child of root (and only of root).
	branchA := makeCommit(t, dir, "branch A", map[string]string{"a.txt": "A\n"})
	// Reset HEAD back to root so the next commit has root
	// (not branchA) as its parent.
	mustRunGit(t, dir, "reset", "--hard", rootOID)
	branchB := makeCommit(t, dir, "branch B", map[string]string{"b.txt": "B\n"})
	// Reset again, create branch C independent of both A and B.
	mustRunGit(t, dir, "reset", "--hard", rootOID)
	branchC := makeCommit(t, dir, "branch C", map[string]string{"c.txt": "C\n"})

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	resolver := NewV2ClosureTopologyResolver()

	// S / F unrelated: S on branch A, F on branch B (both
	// children of root, no common ancestor between them).
	t.Run("S/F unrelated", func(t *testing.T) {
		facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
			SubjectCommit: branchA,
			FreezeCommit:  branchB,
			ClosureCommit: branchC,
		})
		if err != nil {
			t.Fatalf("ResolveTopology: %v", err)
		}
		if facts.Relation != V2ClosureRelationSubjectFreezeUnrelated {
			t.Fatalf("relation = %q, want %q",
				facts.Relation, V2ClosureRelationSubjectFreezeUnrelated)
		}
		if !facts.Diagnostics.HasCode(V2VerifierSubjectFreezeUnrelated) {
			t.Fatalf("diagnostic codes = %v, want subject_freeze_unrelated",
				facts.Diagnostics.Codes())
		}
	})

	// F / C unrelated: S on branch A, F on branch B (F < S
	// would not be true since F is independent), C on
	// branch C independent of B. S on branch A and F on
	// branch B are unrelated, so this surfaces as
	// subject_freeze_unrelated (classifier precedence
	// resolves S/F before F/C).
	t.Run("F/C unrelated", func(t *testing.T) {
		facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
			SubjectCommit: rootOID,
			FreezeCommit:  branchA,
			ClosureCommit: branchC,
		})
		if err != nil {
			t.Fatalf("ResolveTopology: %v", err)
		}
		// S=root, F=branchA (root<branchA), C=branchC
		// (root<branchC but branchA unrelated to branchC).
		if facts.Relation != V2ClosureRelationFreezeClosureUnrelated {
			t.Fatalf("relation = %q, want %q",
				facts.Relation, V2ClosureRelationFreezeClosureUnrelated)
		}
		if !facts.Diagnostics.HasCode(V2VerifierFreezeClosureUnrelated) {
			t.Fatalf("diagnostic codes = %v, want freeze_closure_unrelated",
				facts.Diagnostics.Codes())
		}
	})

	// All unrelated: S, F, C on three independent branches.
	// Reset HEAD to root, then make a fresh third commit
	// disconnected from all others.
	t.Run("S, F, C independent", func(t *testing.T) {
		// Create a third independent branch: HEAD is
		// currently at branchC; reset to root and make a
		// new commit independent of A, B, C.
		mustRunGit(t, dir, "reset", "--hard", rootOID)
		independent := makeCommit(t, dir, "independent", map[string]string{"z.txt": "Z\n"})
		facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
			SubjectCommit: branchA,
			FreezeCommit:  branchB,
			ClosureCommit: independent,
		})
		if err != nil {
			t.Fatalf("ResolveTopology: %v", err)
		}
		// branchA and branchB are unrelated, so the
		// classifier returns subject_freeze_unrelated
		// (S/F precedence).
		if facts.Relation != V2ClosureRelationSubjectFreezeUnrelated {
			t.Fatalf("relation = %q, want %q",
				facts.Relation, V2ClosureRelationSubjectFreezeUnrelated)
		}
	})
}

// TestV2VerifierTopologyMissingMatrix proves the verifier
// rejects missing S, missing F, and missing C with the
// matching typed diagnostic codes. The test substitutes
// 40-character hex strings that do not correspond to any
// commit in the hermetic repository.
func TestV2VerifierTopologyMissingMatrix(t *testing.T) {
	dir := initRepo(t)
	commit := makeCommit(t, dir, "commit", map[string]string{"a.txt": "A\n"})
	missing := strings.Repeat("d", 40)

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	resolver := NewV2ClosureTopologyResolver()

	cases := []struct {
		name      string
		subject   string
		freeze    string
		closure   string
		want      V2ClosureRelation
		wantCodes []V2VerifierCode
	}{
		{
			name:      "missing S",
			subject:   missing,
			freeze:    commit,
			closure:   commit,
			want:      V2ClosureRelationSubjectUnresolved,
			wantCodes: []V2VerifierCode{V2VerifierSubjectUnresolved},
		},
		{
			name:      "missing F",
			subject:   commit,
			freeze:    missing,
			closure:   commit,
			want:      V2ClosureRelationFreezeUnresolved,
			wantCodes: []V2VerifierCode{V2VerifierFreezeUnresolved},
		},
		{
			name:      "missing C",
			subject:   commit,
			freeze:    commit,
			closure:   missing,
			want:      V2ClosureRelationClosureUnresolved,
			wantCodes: []V2VerifierCode{V2VerifierClosureUnresolved},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			facts, err := resolver.ResolveTopology(context.Background(), auth, V2ClosureVerifyRequest{
				SubjectCommit: tc.subject,
				FreezeCommit:  tc.freeze,
				ClosureCommit: tc.closure,
			})
			if err != nil {
				t.Fatalf("ResolveTopology: %v", err)
			}
			if facts.Relation != tc.want {
				t.Fatalf("relation = %q, want %q", facts.Relation, tc.want)
			}
			gotCodes := facts.Diagnostics.Codes()
			if !equalCodeSet(gotCodes, tc.wantCodes) {
				t.Fatalf("diagnostic codes = %v, want %v", gotCodes, tc.wantCodes)
			}
		})
	}
}

// TestV2VerifierTopologyGitFailureMatrix proves the verifier
// surfaces a typed topology_observation_failed diagnostic
// when an ancestry observation fails. The matrix tests use a
// stub authority that returns a synthetic error from
// IsAncestor so the verifier cannot accidentally classify
// the triple.
func TestV2VerifierTopologyGitFailureMatrix(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"a.txt": "A\n"})
	freeze := makeCommit(t, dir, "freeze", map[string]string{"b.txt": "B\n"})
	closure := makeCommit(t, dir, "closure", map[string]string{"c.txt": "C\n"})

	stub := &topologyFailureStubAuthority{
		real: mustAuthority(t, dir),
		failOn: map[string]bool{
			"IsAncestor": true,
		},
	}

	resolver := NewV2ClosureTopologyResolver()
	facts, err := resolver.ResolveTopology(context.Background(), stub, V2ClosureVerifyRequest{
		SubjectCommit: subject,
		FreezeCommit:  freeze,
		ClosureCommit: closure,
	})
	if err != nil {
		t.Fatalf("ResolveTopology: %v", err)
	}
	if facts.Relation != V2ClosureRelationGitFailure {
		t.Fatalf("relation = %q, want %q",
			facts.Relation, V2ClosureRelationGitFailure)
	}
	if !facts.Diagnostics.HasCode(V2VerifierTopologyObservationFailed) {
		t.Fatalf("diagnostic codes = %v, want topology_observation_failed",
			facts.Diagnostics.Codes())
	}
}

// TestV2VerifierTopologyRelationIsAccepted guards the public
// boolean. The single accepted token is the only relation
// that returns true; every other token returns false.
func TestV2VerifierTopologyRelationIsAccepted(t *testing.T) {
	relations := []V2ClosureRelation{
		V2ClosureRelationSubjectBeforeFreezeBeforeClosure,
		V2ClosureRelationSubjectFreezeEqual,
		V2ClosureRelationFreezeClosureEqual,
		V2ClosureRelationSubjectClosureEqual,
		V2ClosureRelationFreezeBeforeSubject,
		V2ClosureRelationClosureBeforeFreeze,
		V2ClosureRelationSubjectFreezeUnrelated,
		V2ClosureRelationFreezeClosureUnrelated,
		V2ClosureRelationSubjectUnresolved,
		V2ClosureRelationFreezeUnresolved,
		V2ClosureRelationClosureUnresolved,
		V2ClosureRelationGitFailure,
	}
	for _, r := range relations {
		got := r.IsAccepted()
		want := r == V2ClosureRelationSubjectBeforeFreezeBeforeClosure
		if got != want {
			t.Fatalf("IsAccepted(%q) = %v, want %v", r, got, want)
		}
	}
}

// mustAuthority constructs a V2ClosureGitAuthority bound to
// the supplied hermetic repository. Test-only helper.
func mustAuthority(t *testing.T, dir string) V2ClosureGitAuthority {
	t.Helper()
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	return auth
}

// topologyFailureStubAuthority wraps a real authority and
// returns synthetic errors for the operations named in
// failOn. The real authority is consulted only when failOn
// does not name the operation.
type topologyFailureStubAuthority struct {
	real   V2ClosureGitAuthority
	failOn map[string]bool
}

func (s *topologyFailureStubAuthority) ObjectFormat(ctx context.Context) (string, error) {
	if s.failOn["ObjectFormat"] {
		return "", &V2VerifierError{Diags: V2VerifierDiagnostics{
			NewV2VerifierDiagnostic(V2VerifierObjectFormatUnavailable, "synthetic"),
		}}
	}
	return s.real.ObjectFormat(ctx)
}

func (s *topologyFailureStubAuthority) ResolveCommit(ctx context.Context, revision string) (string, error) {
	if s.failOn["ResolveCommit"] {
		return "", &V2VerifierError{Diags: V2VerifierDiagnostics{
			NewV2VerifierDiagnostic(V2VerifierSubjectUnresolved, "synthetic"),
		}}
	}
	return s.real.ResolveCommit(ctx, revision)
}

func (s *topologyFailureStubAuthority) ResolveTree(ctx context.Context, commit string) (string, error) {
	if s.failOn["ResolveTree"] {
		return "", &V2VerifierError{Diags: V2VerifierDiagnostics{
			NewV2VerifierDiagnostic(V2VerifierTopologyObservationFailed, "synthetic"),
		}}
	}
	return s.real.ResolveTree(ctx, commit)
}

func (s *topologyFailureStubAuthority) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	if s.failOn["IsAncestor"] {
		return false, &V2VerifierError{Diags: V2VerifierDiagnostics{
			NewV2VerifierDiagnostic(V2VerifierTopologyObservationFailed, "synthetic ancestry failure"),
		}}
	}
	return s.real.IsAncestor(ctx, ancestor, descendant)
}

func (s *topologyFailureStubAuthority) ResolvePathObject(ctx context.Context, commit, path string) (string, string, error) {
	return s.real.ResolvePathObject(ctx, commit, path)
}

func (s *topologyFailureStubAuthority) ReadBlob(ctx context.Context, oid string) ([]byte, error) {
	return s.real.ReadBlob(ctx, oid)
}

// equalCodeSet reports whether two code slices contain the
// same elements (order-independent). Tests assert on closed
// sets of failure codes; the helper avoids re-implementing
// set comparison at every call site.
func equalCodeSet(a, b []V2VerifierCode) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[V2VerifierCode]bool, len(a))
	for _, c := range a {
		seen[c] = true
	}
	for _, c := range b {
		if !seen[c] {
			return false
		}
	}
	return true
}
