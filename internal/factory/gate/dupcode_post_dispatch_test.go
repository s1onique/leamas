// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestDupcodeTypedEntryPointsNoPostDispatchProtectedCalls is the
// production-source structural test. It parses every production .go
// file under internal/factory/gate and runs the unified
// validateTypedDispatchSource helper against the canonical
// defaultTypedDispatchContract.
func TestDupcodeTypedEntryPointsNoPostDispatchProtectedCalls(t *testing.T) {
	files, fset, err := parseGatePackageFiles()
	if err != nil {
		t.Fatalf("parseGatePackageFiles: %v", err)
	}
	result := validateTypedDispatchSource(files, fset, defaultTypedDispatchContract())
	if !result.OK() {
		for _, e := range result.Errors {
			t.Errorf("structural violation: %v", e)
		}
	}
}

// structuralFixture holds the parsed source for an adversarial test.
type structuralFixture struct {
	fset *token.FileSet
	file *ast.File
}

// parseFixture compiles the given Go source text into a single
// *ast.File. Adversarial fixtures exercise the validator's policies
// directly without touching the production source tree.
func parseFixture(t *testing.T, src string) structuralFixture {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return structuralFixture{fset: fset, file: file}
}

// runValidatorOnFixture runs the validator against a single parsed
// file and returns the resulting error list. The fixture is supplied
// as the only entry in the file map so the validator treats it as the
// entire production source for the contract.
func runValidatorOnFixture(t *testing.T, fix structuralFixture, contract typedDispatchContract) []error {
	t.Helper()
	files := map[string]*ast.File{"fixture.go": fix.file}
	result := validateTypedDispatchSource(files, fix.fset, contract)
	return result.Errors
}

// TestAdversarialStructuralMissingInternal proves the validator
// rejects a missing internal declaration.
func TestAdversarialStructuralMissingInternal(t *testing.T) {
	src := `package fixture
func DispatchDupcodeVerifyTyped() int {
	return 0
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "is missing") {
		t.Errorf("expected missing-declaration error, got: %v", errs)
	}
}

// TestAdversarialStructuralDuplicateInternal proves the validator
// rejects two declarations of the same internal name.
func TestAdversarialStructuralDuplicateInternal(t *testing.T) {
	src := `package fixture
func dispatchDupcodeVerifyTypedWith() int {
	return 0
}
func dispatchDupcodeVerifyTypedWith() int {
	return 1
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "appears 2 times") {
		t.Errorf("expected duplicate-declaration error, got: %v", errs)
	}
}

// TestAdversarialStructuralNilBody proves the validator rejects a
// declaration with a nil body.
func TestAdversarialStructuralNilBody(t *testing.T) {
	src := `package fixture
func dispatchDupcodeVerifyTypedWith() int
func DispatchDupcodeVerifyTyped() int { return 0 }
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "nil body") {
		t.Errorf("expected nil-body error, got: %v", errs)
	}
}

// TestAdversarialStructuralZeroDispatch proves the validator rejects
// an internal entry point with no dispatcher.Dispatch call.
func TestAdversarialStructuralZeroDispatch(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch() int { return 0 }
func dispatchDupcodeVerifyTypedWith() int {
	return 0
}
func DispatchDupcodeVerifyTyped() int {
	return dispatchDupcodeVerifyTypedWith()
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "dispatcher.Dispatch call not found") {
		t.Errorf("expected zero-dispatch error, got: %v", errs)
	}
}

// TestAdversarialStructuralTwoDispatch proves the validator rejects
// an internal entry point with two dispatcher.Dispatch calls.
func TestAdversarialStructuralTwoDispatch(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch() int { return 0 }
func dispatchDupcodeVerifyTypedWith() int {
	var dispatcher *disp
	_ = dispatcher.Dispatch()
	_ = dispatcher.Dispatch()
	return 0
}
func DispatchDupcodeVerifyTyped() int {
	return 0
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "called 2 times") {
		t.Errorf("expected two-dispatch error, got: %v", errs)
	}
}

// TestAdversarialStructuralProtectedAfterDispatch proves the
// validator flags a protected call after dispatch.end.
func TestAdversarialStructuralProtectedAfterDispatch(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch() int { return 0 }
type b struct{}
func (b) run() {}
func dispatchDupcodeVerifyTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch()
	binder.run()
	return 0
}
func DispatchDupcodeVerifyTyped() int {
	return 0
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "protected call") {
		t.Errorf("expected protected-call error, got: %v", errs)
	}
}

// TestAdversarialStructuralInlineClosure proves the validator rejects
// protected calls hidden inside an inline closure passed as another
// dispatch argument.
func TestAdversarialStructuralInlineClosure(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch(func()) int { return 0 }
func dispatchDupcodeVerifyTypedWith() int {
	var dispatcher *disp
	_ = dispatcher.Dispatch(func() {
		NewDupcodeRunner()
	})
	return 0
}
func DispatchDupcodeVerifyTyped() int {
	return 0
}
func NewDupcodeRunner() {}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "disallowed call") {
		t.Errorf("expected disallowed-call error, got: %v", errs)
	}
}

// TestAdversarialStructuralNestedHelper proves the validator rejects
// `helper(binder.BindRunner())` as a dispatch argument: only the
// direct call is permitted.
func TestAdversarialStructuralNestedHelper(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch(int) int { return 0 }
func helper(x int) int { return x }
type b struct{}
func (b) BindRunner() int { return 0 }
func dispatchDupcodeVerifyTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(helper(binder.BindRunner()))
	return 0
}
func DispatchDupcodeVerifyTyped() int {
	return 0
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "disallowed call") {
		t.Errorf("expected disallowed-call error for nested helper, got: %v", errs)
	}
}

// TestAdversarialStructuralDirectBindRunner proves the validator
// accepts the direct `dispatcher.Dispatch(..., binder.BindRunner())`
// pattern. The fixture is intentionally complete (all three internal
// entries and all three public wrappers) so the validator only flags
// the targeted policy.
func TestAdversarialStructuralDirectBindRunner(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch(int) int { return 0 }
type b struct{}
func (b) BindRunner() int { return 0 }
func dispatchDupcodeVerifyTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(binder.BindRunner())
	return 0
}
func dispatchDupcodeBaselineVerifyTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(binder.BindRunner())
	return 0
}
func dispatchDupcodeUpdateBaselineTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(binder.BindRunner())
	return 0
}
func DispatchDupcodeVerifyTyped() int {
	return dispatchDupcodeVerifyTypedWith()
}
func DispatchDupcodeBaselineVerifyTyped() int {
	return dispatchDupcodeBaselineVerifyTypedWith()
}
func DispatchDupcodeUpdateBaselineTyped() int {
	return dispatchDupcodeUpdateBaselineTypedWith()
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if len(errs) != 0 {
		t.Errorf("expected no errors for direct BindRunner, got: %v", errs)
	}
}

// TestAdversarialStructuralMissingPublic proves the validator rejects
// a missing public wrapper.
func TestAdversarialStructuralMissingPublic(t *testing.T) {
	src := `package fixture
func dispatchDupcodeVerifyTypedWith() int {
	var dispatcher struct{}
	_ = dispatcher
	return 0
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "DispatchDupcodeVerifyTyped") || !containsAny(errs, "is missing") {
		t.Errorf("expected missing-public error, got: %v", errs)
	}
}

// TestAdversarialStructuralPublicZeroDelegates proves the validator
// rejects a public wrapper that does not delegate to its internal.
func TestAdversarialStructuralPublicZeroDelegates(t *testing.T) {
	src := `package fixture
func dispatchDupcodeVerifyTypedWith() int { return 0 }
func DispatchDupcodeVerifyTyped() int { return 0 }
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "must delegate exactly once") {
		t.Errorf("expected zero-delegation error, got: %v", errs)
	}
}

// TestAdversarialStructuralPublicTwoDelegates proves the validator
// rejects a public wrapper that delegates twice.
func TestAdversarialStructuralPublicTwoDelegates(t *testing.T) {
	src := `package fixture
func dispatchDupcodeVerifyTypedWith() int { return 0 }
func DispatchDupcodeVerifyTyped() int {
	_ = dispatchDupcodeVerifyTypedWith()
	_ = dispatchDupcodeVerifyTypedWith()
	return 0
}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "got 2 calls") {
		t.Errorf("expected two-delegation error, got: %v", errs)
	}
}

// TestAdversarialStructuralPublicCallsProtected proves the validator
// rejects a public wrapper that calls a protected operation directly.
func TestAdversarialStructuralPublicCallsProtected(t *testing.T) {
	src := `package fixture
func dispatchDupcodeVerifyTypedWith() int { return 0 }
func DispatchDupcodeVerifyTyped() int {
	LoadBaseline()
	return dispatchDupcodeVerifyTypedWith()
}
func LoadBaseline() {}
`
	fix := parseFixture(t, src)
	errs := runValidatorOnFixture(t, fix, defaultTypedDispatchContract())
	if !containsAny(errs, "directly calls protected operation") {
		t.Errorf("expected direct-protected-call error, got: %v", errs)
	}
}

// containsAny returns true when any of the errors contain substr.
func containsAny(errs []error, substr string) bool {
	for _, e := range errs {
		if e == nil {
			continue
		}
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	return false
}

// completeDispatchFixture builds a structural fixture with all three
// internal entry points and all three public wrappers wired up.
// extraDecls is prepended at package level (for adversarial
// receiver types), and dispatchCall is used as the single argument
// to dispatcher.Dispatch inside every internal entry point.
func completeDispatchFixture(t *testing.T, dispatchCall, extraDecls string) structuralFixture {
	t.Helper()
	src := `package fixture
type disp struct{}
func (disp) Dispatch(int) int { return 0 }
type b struct{}
func (b) BindRunner() int { return 0 }
` + extraDecls + `
func dispatchDupcodeVerifyTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(` + dispatchCall + `)
	return 0
}
func dispatchDupcodeBaselineVerifyTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(` + dispatchCall + `)
	return 0
}
func dispatchDupcodeUpdateBaselineTypedWith() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(` + dispatchCall + `)
	return 0
}
func DispatchDupcodeVerifyTyped() int { return dispatchDupcodeVerifyTypedWith() }
func DispatchDupcodeBaselineVerifyTyped() int { return dispatchDupcodeBaselineVerifyTypedWith() }
func DispatchDupcodeUpdateBaselineTyped() int { return dispatchDupcodeUpdateBaselineTypedWith() }
`
	return parseFixture(t, src)
}

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
