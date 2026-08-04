// SPDX-License-Identifier: Apache-2.0

package gate

import "testing"

// TestAdversarialStructuralEvilBinderRejected proves the validator
// rejects `evil.BindRunner()` even though the selector name matches.
// The selector name and receiver identity must BOTH match; the
// rejection reason must mention "receiver identity differs" so the
// failure mode is clearly distinguishable from a generic "disallowed
// call".
func TestAdversarialStructuralEvilBinderRejected(t *testing.T) {
	fix := completeDispatchFixture(t,
		"evil.BindRunner()",
		"type evil struct{}\nfunc (evil) BindRunner() int { return 0 }\n")
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "receiver") {
		t.Errorf("expected receiver-identity error, got: %v", errs)
	}
	if !containsAny(errs, "receiver identity differs") {
		t.Errorf("expected 'receiver identity differs' diagnostic, got: %v", errs)
	}
}

// TestAdversarialStructuralOtherBinderRejected proves the validator
// rejects `other.BindRunner()` for the same reason: the receiver
// must be exactly `binder`.
func TestAdversarialStructuralOtherBinderRejected(t *testing.T) {
	fix := completeDispatchFixture(t,
		"other.BindRunner()",
		"type other struct{}\nfunc (other) BindRunner() int { return 0 }\n")
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "receiver") {
		t.Errorf("expected receiver-identity error, got: %v", errs)
	}
}

// TestAdversarialStructuralBinderReceiverAccepted proves the validator
// accepts `binder.BindRunner()` exactly — the receiver identity must
// be the literal identifier `binder`.
func TestAdversarialStructuralBinderReceiverAccepted(t *testing.T) {
	fix := completeDispatchFixture(t, "binder.BindRunner()", "")
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if len(errs) != 0 {
		t.Errorf("expected no errors for canonical binder.BindRunner(), got: %v", errs)
	}
}
