// SPDX-License-Identifier: Apache-2.0

// binary_gate_assertions_test.go owns the typed assertion
// helpers the CORRECTION07 failure matrix uses. The helpers
// replace the substring-match-on-error.Error() pattern with
// the production V2Error diagnostic-code authority so each
// row's owning typed failure is asserted through the same
// path the CLI uses.
//
// Splitting this file from the matrix file keeps the
// matrix file focused on row definitions while the helpers
// stay short and focused.

package closure

import (
	"errors"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// requireV2Code asserts that err is a *V2Error whose
// FIRST diagnostic code equals want. The first-code rule
// matches the production CLI render path: every
// RunClosureProtocolV2ExecuteWithDeps failure returns a
// V2Error whose first diagnostic is the owning failure
// family. A wrapped V2Error (Unwrap returns Cause) is
// unwrapped automatically; if the wrapped cause is itself
// a *V2Error, the assertion inspects that.
//
// The helper deliberately rejects substring matches on
// err.Error(): the ACT requires the typed-code authority
// so future code-message changes cannot silently shift
// the owning failure family.
func requireV2Code(t *testing.T, err error, want V2DiagnosticCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("requireV2Code(%q): err is nil", want)
	}
	v2err := unwrapV2Error(err)
	if v2err == nil {
		t.Fatalf("requireV2Code(%q): err = %T (%v), want *V2Error", want, err, err)
	}
	if len(v2err.Diags) == 0 {
		t.Fatalf("requireV2Code(%q): V2Error has no diagnostics", want)
	}
	got := v2err.Diags[0].Code
	if got != want {
		t.Fatalf("requireV2Code(%q): got first-code %q (diags=%v)", want, got, v2err.Diags.Codes())
	}
}

// unwrapV2Error walks the cause chain until it finds a
// *V2Error or runs out. The function returns nil if no
// V2Error is reachable.
func unwrapV2Error(err error) *V2Error {
	for err != nil {
		if v2err, ok := err.(*V2Error); ok {
			return v2err
		}
		err = errors.Unwrap(err)
	}
	return nil
}

// requireOwnedR6BFailure asserts the typed owning R6-B
// failure code is the ONLY diagnostic family on the
// returned error (no unrelated drift diagnostics). The
// rule keeps each matrix row honest: a row passes only if
// the R6-B integration owns the failure, not the R6-A
// caller-snapshot authority or any unrelated authority.
//
// The helper is the inverse of requireB2Incomplete: the
// owned-failure assertion runs BEFORE the B2 consequence
// so an incorrectly-owned row cannot be salvaged by a B2
// barrier rejection.
func requireOwnedR6BFailure(t *testing.T, err error, want V2DiagnosticCode) {
	t.Helper()
	requireV2Code(t, err, want)
	v2err := unwrapV2Error(err)
	for _, d := range v2err.Diags[1:] {
		if d.Code == V2CodeCallerHeadChanged ||
			d.Code == V2CodeCallerTreeChanged ||
			d.Code == V2CodeCallerWorktreeDirtyAfter ||
			d.Code == V2CodeCallerRefsChanged ||
			d.Code == V2CodeWorktreeRegistrationLeaked ||
			d.Code == V2CodeCallerStateUnavailable ||
			d.Code == V2CodeWorktreeInventoryUnavailable ||
			d.Code == V2CodeCallerWorktreeDirty {
			t.Fatalf("requireOwnedR6BFailure(%q): found non-R6-B owner %q in diags=%v",
				want, d.Code, v2err.Diags.Codes())
		}
	}
}

// b2Consequence is the closed test-metadata enum that names
// the B2 consequence each matrix row must prove. It is test
// metadata only; no production authority is introduced.
type b2Consequence string

const (
	// consequenceCandidateUnreachable marks a row whose
	// failure surfaces before a valid B2 candidate can
	// exist. The observation fields the B2 builder needs
	// are absent (BinaryPath empty AND Gate.SubjectRoot
	// empty). The test asserts the observation is
	// structurally empty; no candidate is built.
	consequenceCandidateUnreachable b2Consequence = "candidate_unreachable"
	// consequenceBarrierRejects marks a row whose
	// failure surfaces AFTER a valid candidate can be
	// constructed. The observation is populated; the B2
	// barrier must reject the candidate with
	// evidence.ErrIncompleteEvidence.
	consequenceBarrierRejects b2Consequence = "barrier_rejects"
)

// requireB2Consequence asserts the per-row B2 consequence
// the matrix schema declared. Every row MUST call this
// helper so the strict 12-row matrix proves B2 consequence
// for the full set, not a subset.
//
// The helper is the consequence-only assertion: it MUST
// be called AFTER the owned-failure assertion so the test
// distinguishes "B2 rejected because the row owns a real
// R6-B failure" from "B2 rejected because the row failed
// to produce an observation". The two paths are
// semantically different for R6-B and both must hold.
//
// For consequenceCandidateUnreachable the helper asserts
// the observation is structurally empty. For
// consequenceBarrierRejects the helper builds the candidate
// and asserts the B2 barrier rejects with
// evidence.ErrIncompleteEvidence.
func requireB2Consequence(t *testing.T, want b2Consequence, obs V2ExecutionObservation, buildCandidate func() error) {
	t.Helper()
	switch want {
	case consequenceCandidateUnreachable:
		// The integration failed before producing an
		// observation: the B2 candidate is unreachable.
		// The fields the B2 builder needs must be absent.
		if obs.Binary.BinaryPath != "" || obs.Gate.SubjectRoot != "" {
			t.Fatalf("requireB2Consequence(candidate_unreachable): observation is unexpectedly populated: binary=%q gate=%q",
				obs.Binary.BinaryPath, obs.Gate.SubjectRoot)
		}
	case consequenceBarrierRejects:
		// The observation is populated (B1 + gate ran
		// successfully); B2 must reject the candidate.
		if obs.Binary.BinaryPath == "" || obs.Gate.SubjectRoot == "" {
			t.Fatalf("requireB2Consequence(barrier_rejects): observation is empty: binary=%q gate=%q",
				obs.Binary.BinaryPath, obs.Gate.SubjectRoot)
		}
		err := buildCandidate()
		if err == nil {
			t.Fatalf("requireB2Consequence(barrier_rejects): B2 barrier accepted an invalid candidate")
		}
		if !errors.Is(err, evidence.ErrIncompleteEvidence) {
			t.Fatalf("requireB2Consequence(barrier_rejects): err = %v, want ErrIncompleteEvidence", err)
		}
	default:
		t.Fatalf("requireB2Consequence: unknown consequence %q", want)
	}
}
