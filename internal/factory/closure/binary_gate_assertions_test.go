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

// requireB2Incomplete asserts the B2 publication barrier
// refuses an attempted candidate with
// evidence.ErrIncompleteEvidence (or returns a candidate
// that is INCOMPLETE because the observation was rejected
// before a candidate could be assembled).
//
// The helper is the consequence-only assertion: it MUST
// be called AFTER requireOwnedR6BFailure so the test
// distinguishes "B2 rejected because the row owns a real
// R6-B failure" from "B2 rejected because the row failed
// to produce an observation". The two paths are
// semantically different for R6-B and both must hold.
func requireB2Incomplete(t *testing.T, obs V2ExecutionObservation, buildCandidate func() error) {
	t.Helper()
	if obs.Binary.BinaryPath == "" && obs.Gate.SubjectRoot == "" {
		// The integration failed before producing an
		// observation: the B2 candidate is unreachable,
		// which is a valid consequence for rows whose
		// failure surfaces before any observation.
		return
	}
	err := buildCandidate()
	if err == nil {
		t.Fatalf("requireB2Incomplete: B2 barrier accepted an invalid candidate")
	}
	if !errors.Is(err, evidence.ErrIncompleteEvidence) {
		t.Fatalf("requireB2Incomplete: err = %v, want ErrIncompleteEvidence", err)
	}
}
