// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// --- Operation contract tests ---

func TestVerifier_Validate_Operations_VerifyAccept(t *testing.T) {
	v := Verifier{
		Name:       "ord",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        noopRun,
		Operations: verifyOnly(),
	}
	if err := v.Validate(); err != nil {
		t.Errorf("[verify] must be accepted: %v", err)
	}
}

func TestVerifier_Validate_Operations_UpdateBaselineAccept(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err != nil {
		t.Errorf("[update_baseline] must be accepted on canonical identity: %v", err)
	}
}

func TestVerifier_Validate_Operations_EmptyRejected(t *testing.T) {
	v := Verifier{
		Name:       "ord",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        noopRun,
		Operations: nil,
	}
	err := v.Validate()
	if err == nil {
		t.Fatal("expected error for empty operations")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Field != "Operations" {
		t.Errorf("expected Operations field, got: %v", err)
	}
}

func TestVerifier_Validate_Operations_EmptyStringRejected(t *testing.T) {
	v := Verifier{
		Name:       "ord",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        noopRun,
		Operations: []verifierauthority.VerifierOperation{""},
	}
	err := v.Validate()
	if err == nil {
		t.Fatal("expected error for empty-string operation")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Field != "Operations" {
		t.Errorf("expected Operations field, got: %v", err)
	}
}

func TestVerifier_Validate_Operations_UnknownRejected(t *testing.T) {
	v := Verifier{
		Name:       "ord",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        noopRun,
		Operations: []verifierauthority.VerifierOperation{"unknown"},
	}
	err := v.Validate()
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Field != "Operations" {
		t.Errorf("expected Operations field, got: %v", err)
	}
}

func TestVerifier_Validate_Operations_WhitespaceRejected(t *testing.T) {
	v := Verifier{
		Name:       "ord",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        noopRun,
		Operations: []verifierauthority.VerifierOperation{" verify "},
	}
	err := v.Validate()
	if err == nil {
		t.Fatal("expected error for whitespace-padded operation")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Field != "Operations" {
		t.Errorf("expected Operations field, got: %v", err)
	}
}

func TestVerifier_Validate_Operations_CaseRejected(t *testing.T) {
	v := Verifier{
		Name:       "ord",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        noopRun,
		Operations: []verifierauthority.VerifierOperation{"VERIFY"},
	}
	err := v.Validate()
	if err == nil {
		t.Fatal("expected error for case-variant operation")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Field != "Operations" {
		t.Errorf("expected Operations field, got: %v", err)
	}
}

func TestVerifier_Validate_Operations_DuplicateRejected(t *testing.T) {
	v := Verifier{
		Name:      "ord",
		Lane:      VerifierLaneFast,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationGate,
		Run:       noopRun,
		Operations: []verifierauthority.VerifierOperation{
			verifierauthority.OperationVerify,
			verifierauthority.OperationVerify,
		},
	}
	err := v.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate operation")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Field != "Operations" {
		t.Errorf("expected Operations field, got: %v", err)
	}
}

// --- Reserved identity biconditional tests ---

func TestVerifier_Validate_ReservedIdentity_CompleteTupleAccept(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err != nil {
		t.Errorf("canonical tuple must be accepted: %v", err)
	}
	if !isCanonicalDupcodeUpdateDefinition(v) {
		t.Error("expected isCanonicalDupcodeUpdateDefinition to return true")
	}
}

func TestVerifier_Validate_ReservedIdentity_VerifyOperationsRejected(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: verifyOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when reserved name uses [verify] operations")
	}
}

func TestVerifier_Validate_ReservedIdentity_UnknownOperationRejected(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: []verifierauthority.VerifierOperation{"unknown"},
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when reserved name uses unknown operation")
	}
}

func TestVerifier_Validate_ReservedIdentity_TwoOperationsRejected(t *testing.T) {
	v := Verifier{
		Name:      "dupcode-update-baseline",
		Lane:      VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationCommandOnly,
		Run:       nil,
		Operations: []verifierauthority.VerifierOperation{
			verifierauthority.OperationVerify,
			verifierauthority.OperationUpdateBaseline,
		},
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when reserved name uses mixed operations")
	}
}

func TestVerifier_Validate_ReservedIdentity_WrongLaneRejected(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when reserved name has wrong lane")
	}
}

func TestVerifier_Validate_ReservedIdentity_WrongAuthorityRejected(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityCIExactCheckout,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when reserved name has wrong authority")
	}
}

func TestVerifier_Validate_ReservedIdentity_WrongScopeRejected(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        nil,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when reserved name has wrong scope")
	}
}

func TestVerifier_Validate_ReservedIdentity_NonNilRunRejected(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        noopRun,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when reserved name has non-nil Run")
	}
}

func TestVerifier_Validate_ReservedIdentity_WrongNameWithCanonicalTupleRejected(t *testing.T) {
	v := Verifier{
		Name:       "wrong-name",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when canonical tuple uses non-reserved name")
	}
}

func TestVerifier_Validate_ReservedIdentity_NonCanonicalNameRejected(t *testing.T) {
	v := Verifier{
		Name:       "dupcode-update-baseline-other",
		Lane:       VerifierLaneDupcode,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationCommandOnly,
		Run:        nil,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error for near-miss reserved name")
	}
}

// --- Ordinary verifier restriction tests ---

func TestVerifier_Validate_Ordinary_UpdateBaselineRejected(t *testing.T) {
	v := Verifier{
		Name:       "ordinary",
		Lane:       VerifierLaneFast,
		Authority:  verifierauthority.AuthorityLocalSafe,
		Scope:      InvocationGate,
		Run:        noopRun,
		Operations: updateBaselineOnly(),
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when ordinary verifier declares update_baseline")
	}
}

func TestVerifier_Validate_Ordinary_MultipleOperationsRejected(t *testing.T) {
	v := Verifier{
		Name:      "ordinary",
		Lane:      VerifierLaneFast,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationGate,
		Run:       noopRun,
		Operations: []verifierauthority.VerifierOperation{
			verifierauthority.OperationVerify,
			verifierauthority.OperationUpdateBaseline,
		},
	}
	if err := v.Validate(); err == nil {
		t.Error("expected error when ordinary verifier declares multiple operations")
	}
}

// --- AllowsOperation tests ---

func TestVerifier_AllowsOperation_AcceptsDeclared(t *testing.T) {
	v := Verifier{Operations: verifyOnly()}
	if !v.AllowsOperation(verifierauthority.OperationVerify) {
		t.Error("AllowsOperation must accept declared verify")
	}
}

func TestVerifier_AllowsOperation_RejectsUndeclared(t *testing.T) {
	v := Verifier{Operations: verifyOnly()}
	if v.AllowsOperation(verifierauthority.OperationUpdateBaseline) {
		t.Error("AllowsOperation must reject undeclared update_baseline")
	}
}

func TestVerifier_AllowsOperation_EmptyAlwaysFalse(t *testing.T) {
	v := Verifier{Operations: nil}
	if v.AllowsOperation(verifierauthority.OperationVerify) {
		t.Error("AllowsOperation must reject on empty list")
	}
}
