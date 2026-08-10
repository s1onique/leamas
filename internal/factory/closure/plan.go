// SPDX-License-Identifier: Apache-2.0

// Package closure - plan.go is the closure-side Plan
// Contract v1 entry surface.
//
// B2-R7 motivation: prior to B2-R7, this file owned the
// closure runner's typed Plan semantic validator and a
// hand-maintained set of helpers (validatePlanChecks,
// validatePlanArtifacts, validatePlanPolicy, etc.) that
// reproduced every wire-contract rule already enforced by
// the plancontract leaf. Two authorities cannot agree on
// whether a plan is valid; the closure runner therefore
// carried a duplicate semantic authority that the B2-R4
// "parity assertion" only loosely guarded.
//
// B2-R7 deletes every duplicate semantic helper in this
// package. The typed validators are gone; only the entry
// points remain. The leaf is now the single semantic
// authority. validatePlanTyped (the closure-side typed
// validator) is preserved as a thin adapter in
// plan_adapter.go so the existing PlanSemanticError
// diagnostics continue to flow through the legacy
// typed-error contract.
//
// This file owns only:
//   - the public typed Plan struct surface (kept in
//     model.go for historical reasons),
//   - the typed-decode + size-bounded file-load entry
//     points (DecodePlan, LoadPlan, LoadPlanFromBytes),
//   - the EncodePlanForValidation helper the adapter
//     uses to materialise bytes for the leaf,
//   - a few misc I/O helpers (readBoundedFile) shared
//     by the loaders.
//
// NO wire-contract rule lives here. NO semantic
// validator lives here. NO typed Plan field is mutated
// here.
package closure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// DecodePlan is the legacy public entry point. It preserves
// the documented contract: parse, decode, and ValidatePlan
// in sequence. The internal composed pipeline routes the
// bytes through parseBoundedClosurePlanDocument (the single
// bounded syntactic authority) and the typed-decoder
// through decodeTypedPlan; composition observability is
// invocation-local via the compositionObserver interface
// in plan_contract_validation.go.
func DecodePlan(data []byte) (Plan, error) {
	root, parseDiagnostics := parseBoundedClosurePlanDocument(data)
	if len(parseDiagnostics) > 0 {
		return Plan{}, errorFromDiagnostics(parseDiagnostics)
	}
	plan, err := decodeTypedPlan(root)
	if err != nil {
		return Plan{}, err
	}
	if err := ValidatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// decodeTypedPlan turns the already-parsed document root
// into a typed Plan. It uses the same canonical JSON
// encoder/decoder pair as the parser so no second
// syntactic parse occurs. The typed decoder uses
// DisallowUnknownFields so unknown JSON keys still
// surface as a typed decode error even when the
// structural validator has accepted the document.
func decodeTypedPlan(root any) (Plan, error) {
	return decodeTypedPlanWithObserver(root, noopCompositionObserver{})
}

// decodeTypedPlanWithObserver is the internal entry point
// the composed pipeline uses. The observer is invocation-
// local; tests pass a per-assertion counting observer and
// production passes the noop observer. The function is a
// pure typed-decoder wrapper: it does NOT validate any
// wire-contract rule (the canonical plancontract leaf
// owns those). DisallowUnknownFields surfaces unknown JSON
// keys as a typed decode error so a structural validator
// that accepted the document still rejects unknown fields.
func decodeTypedPlanWithObserver(root any, observer compositionObserver) (Plan, error) {
	observer.TypedDecoded()
	buf, err := json.Marshal(root)
	if err != nil {
		return Plan{}, fmt.Errorf("marshal parsed plan: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()
	var plan Plan
	if err := dec.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("typed decode: %w", err)
	}
	return plan, nil
}

// errorFromDiagnostics turns a list of structural
// diagnostics into a Go error so the legacy DecodePlan
// preserves its (Plan{}, error) return contract.
func errorFromDiagnostics(diags []PlanValidationError) error {
	if len(diags) == 0 {
		return nil
	}
	return fmt.Errorf("plan rejected by structural validation: %s", diags[0].Message)
}

func LoadPlan(path string) (Plan, []byte, error) {
	data, err := readBoundedFile(path, plancontract.MaxPlanBytes)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("read closure plan: %w", err)
	}
	plan, err := DecodePlan(data)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("validate closure plan: %w", err)
	}
	return plan, data, nil
}

// LoadPlanFromBytes parses plan bytes without reading from
// the filesystem. It enforces the size bound and strict
// JSON syntax only; callers that need an executable plan
// must subsequently invoke ValidatePlan explicitly.
//
// B2-R2: the bounded syntactic decoder is the canonical
// plancontract.DecodeBytes. The closure runner and the
// evidence package both consume the same parser pass so
// the production decoder and the evidence decoder cannot
// diverge.
func LoadPlanFromBytes(data []byte) (Plan, []byte, error) {
	root, err := plancontract.DecodeBytes(data)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("plan rejected by structural validation: %s", convertPlanContractError(err))
	}
	plan, err := decodeTypedPlan(root)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("decode closure plan: %w", err)
	}
	return plan, data, nil
}

// convertPlanContractError adapts the plancontract leaf's
// typed errors to a single human-readable string that the
// closure package's legacy errorFromDiagnostics contract
// preserves. The function is a thin wrapper around the
// typed error so callers that want the type can switch on
// it; legacy callers just see the message.
func convertPlanContractError(err error) string {
	return err.Error()
}

// ValidatePlan is the typed-Plan entry point for the
// closure runner's plan validation.
//
// B2-R7 single-authority rule: the function delegates to
// plancontract.DecodeAndValidateFull and adapts the leaf's
// DecodeError back to the closure package's legacy
// PlanSemanticError contract. The function MUST NOT
// re-implement any wire-contract rule. The adapter body
// is in plan_adapter.go; this comment marks the entry
// point as a thin shim over the canonical leaf.
func ValidatePlan(plan Plan) error {
	return validatePlanTyped(plan)
}

// encodePlanForValidation re-encodes the typed Plan to
// the Plan Contract v1 wire shape. The adapter uses the
// re-encoded bytes as the canonical input to
// plancontract.DecodeAndValidateFull so the leaf's strict
// syntax + semantic pipeline is the single authority over
// every wire-contract rule.
//
// The re-encoded bytes are semantically equivalent to the
// source bytes (modulo whitespace) because every typed
// field is a value type or a pointer whose zero value is
// the canonical "absent" signal the leaf accepts.
func encodePlanForValidation(plan Plan) ([]byte, error) {
	return json.Marshal(plan)
}

// readBoundedFile reads up to limit bytes from path. The
// caller MUST supply limit; the cap is enforced before
// allocation so a runaway file cannot exhaust memory.
func readBoundedFile(path string, limit int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > int64(limit) {
		return nil, fmt.Errorf("file exceeds %d-byte limit", limit)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("file exceeds %d-byte limit", limit)
	}
	return data, nil
}

// planExecutionModePath is the canonical JSON pointer
// used in every diagnostic that names the execution-mode
// field. Centralising the string keeps the runtime,
// JSON Schema, and CLI subprocess tests aligned. B2-R7
// preserves the constant for diagnostic compatibility;
// the canonical semantic authority lives in plancontract.
const planExecutionModePath = "/execution/mode"

// exactClosurePlaceholders is the closure-package alias
// for the canonical plancontract placeholder set. The
// alias exists so existing test files that probe the
// set directly (for example, render tests) keep working
// without reaching into the plancontract package's
// unexported symbol.
//
// B2-R7 single-source rule: the underlying set lives in
// plancontract.exactClosurePlaceholders; this alias is
// the only mirror.
var exactClosurePlaceholders = map[string]struct{}{
	"TBD":            {},
	"TODO":           {},
	"UNKNOWN":        {},
	"RUNNING":        {},
	"TO BE RECORDED": {},
}
