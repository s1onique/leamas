package closure

import (
	"errors"
	"fmt"
	"strings"
)

// ExecutionMode is the canonical, closed enumeration of execution modes
// accepted by Closure Protocol v1 plans. The zero value is invalid and
// must never be treated as a silent default.
//
// The single supported value is ExecutionModeSerialFailFast.
//
// Selection basis (see
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-PLAN-EXECUTION-MODE-RECONCILIATION01):
//
//  1. The Go plan model has used `plan.Execution.Mode` since v1 first
//     shipped; the JSON struct tag is `mode` nested in `execution`.
//  2. Every committed v1 closure plan in docs/closure-plans/*.json and
//     every committed producer in cmd/leamas emits this exact spelling.
//  3. The runtime validator has only ever accepted this spelling.
//  4. No compatible alias was ever emitted or consumed.
//
// Therefore the canonical path is `execution.mode` with one supported
// value, `"serial_fail_fast"`.
type ExecutionMode string

const (
	// ExecutionModeSerialFailFast is the only execution mode accepted
	// by Closure Protocol v1. The checks run sequentially in the order
	// they appear in the plan and stop on the first failure.
	ExecutionModeSerialFailFast ExecutionMode = "serial_fail_fast"
)

// ExecutionModePresence classifies how an execution mode was supplied
// in a closure plan. The categories are mutually exclusive and let the
// validator produce deterministic, distinct diagnostics for each kind
// of bad input.
type ExecutionModePresence int

const (
	// ExecutionModeMissing means the `mode` property was absent.
	// The validator must reject the plan because runtime cannot
	// infer a privileged mode.
	ExecutionModeMissing ExecutionModePresence = iota
	// ExecutionModePresentEmpty means the property was present but
	// its value was an empty string.
	ExecutionModePresentEmpty
	// ExecutionModePresentWhitespace means the property was present
	// but its value was composed only of whitespace, which Closure
	// Protocol v1 never treats as a valid mode.
	ExecutionModePresentWhitespace
	// ExecutionModePresentUnknown means the property was present
	// with a value that is not in the closed set of supported modes.
	ExecutionModePresentUnknown
	// ExecutionModePresentValid means the property was present and
	// its value matches a supported ExecutionMode constant.
	ExecutionModePresentValid
)

// String renders the presence category as a stable, lowercase label
// suitable for inclusion in error diagnostics and tests.
func (p ExecutionModePresence) String() string {
	switch p {
	case ExecutionModeMissing:
		return "missing"
	case ExecutionModePresentEmpty:
		return "empty"
	case ExecutionModePresentWhitespace:
		return "whitespace"
	case ExecutionModePresentUnknown:
		return "unknown"
	case ExecutionModePresentValid:
		return "valid"
	}
	return fmt.Sprintf("invalid(%d)", int(p))
}

// ExecutionModeError is returned when an execution mode is rejected by
// the validator. It identifies the failing property path, the rejected
// value (when one was supplied), and the precise presence category so
// callers can map each failure to a stable diagnostic.
type ExecutionModeError struct {
	// Path is the canonical JSON pointer at which the mode was
	// expected. It is always "/execution/mode" for plan decoding.
	Path string
	// Value is the textual value that was supplied, or the empty
	// string when the property was missing.
	Value string
	// Presence is the precise classification of the rejection.
	Presence ExecutionModePresence
	// Supported lists every accepted value. It is included so the
	// error message names every option the producer can choose.
	Supported []ExecutionMode
}

// PlanDiagnostics implements planDiagnosticSource. It returns a single
// diagnostic with exact InstancePath, Code, Keyword, and deep-copied
// AcceptedValues.
func (e *ExecutionModeError) PlanDiagnostics() []PlanValidationError {
	// Deep-copy AcceptedValues so callers can mutate the returned
	// slice without affecting the error's internal state.
	accepted := make([]string, len(e.Supported))
	for i, m := range e.Supported {
		accepted[i] = string(m)
	}
	return []PlanValidationError{clonePlanValidationError(PlanValidationError{
		InstancePath:   "/execution/mode",
		SchemaPath:     "/execution/mode",
		Code:           PlanCodeSemanticConstraintFailed,
		Keyword:        KeywordEnum,
		Message:        e.Error(),
		AcceptedValues: accepted,
	})}
}

// Error implements the error interface.
func (e *ExecutionModeError) Error() string {
	switch e.Presence {
	case ExecutionModeMissing:
		return fmt.Sprintf("closure plan: %s is required and must be one of %v", e.Path, e.Supported)
	case ExecutionModePresentEmpty:
		return fmt.Sprintf("closure plan: %s is empty; expected one of %v", e.Path, e.Supported)
	case ExecutionModePresentWhitespace:
		return fmt.Sprintf("closure plan: %s %q is whitespace-only; expected one of %v", e.Path, e.Value, e.Supported)
	default:
		return fmt.Sprintf("closure plan: %s %q is not a supported execution mode; expected one of %v", e.Path, e.Value, e.Supported)
	}
}

// supportedExecutionModes is the canonical, ordered set of accepted
// ExecutionMode values. New values MUST be appended here and only
// here; no other site in the repository may enumerate supported modes.
var supportedExecutionModes = []ExecutionMode{ExecutionModeSerialFailFast}

// ClassifyExecutionMode separates "what was supplied" from "is it
// valid". The function is the single entry point that other validators
// (including the JSON Schema parity tests) must use to ensure schema
// and runtime agree on every classification.
//
// The value argument is the raw string the producer supplied. Callers
// MUST pass strings.TrimSpace(value) when they want to treat "   " and
// "" identically; otherwise the empty and whitespace categories are
// reported verbatim.
func ClassifyExecutionMode(rawValue string) (ExecutionMode, ExecutionModePresence) {
	if rawValue == "" {
		return "", ExecutionModeMissing
	}
	if rawValue != "" && strings.TrimSpace(rawValue) == "" {
		return "", ExecutionModePresentWhitespace
	}
	for _, candidate := range supportedExecutionModes {
		if ExecutionMode(rawValue) == candidate {
			return candidate, ExecutionModePresentValid
		}
	}
	return "", ExecutionModePresentUnknown
}

// ParseExecutionMode is the strict parsing entry point used by runtime
// validators and by the canonical-plan producer. It returns the
// normalized ExecutionMode value on success and a typed *ExecutionModeError
// on any failure. Callers can inspect err.(*ExecutionModeError).Presence
// to dispatch different diagnostics.
//
// Unlike ClassifyExecutionMode, ParseExecutionMode treats the empty
// string as the "property omitted" case (ExecutionModeMissing), not as
// "present with empty value". The strict JSON decoder guarantees that
// a present-but-empty string round-trips through DecodePlan unchanged,
// but the empty-mode failure category must still distinguish the two
// cases because the schema and CLI diagnostics need to report them
// differently.
//
// The asymmetry is explicit: omitting the property means "I forgot the
// field" while an empty string means "I set the field to garbage". Both
// are rejected, but the validator names the bad input precisely.
func ParseExecutionMode(path, rawValue string) (ExecutionMode, error) {
	if rawValue == "" {
		return "", &ExecutionModeError{
			Path:      path,
			Value:     rawValue,
			Presence:  ExecutionModePresentEmpty,
			Supported: append([]ExecutionMode(nil), supportedExecutionModes...),
		}
	}
	mode, presence := ClassifyExecutionMode(rawValue)
	if presence == ExecutionModePresentValid {
		return mode, nil
	}
	return "", &ExecutionModeError{
		Path:      path,
		Value:     rawValue,
		Presence:  presence,
		Supported: append([]ExecutionMode(nil), supportedExecutionModes...),
	}
}

// IsExecutionModeError reports whether err is an *ExecutionModeError
// so callers can route the failure through a dedicated diagnostic
// path without exposing the concrete error type.
func IsExecutionModeError(err error) bool {
	var target *ExecutionModeError
	return errors.As(err, &target)
}

// SupportedExecutionModes returns a defensive copy of the canonical,
// ordered list of supported ExecutionMode values. Callers that need
// to enumerate modes (for example to render JSON Schema enum arrays)
// must use this helper so they always see the current authoritative
// set rather than a captured slice.
func SupportedExecutionModes() []ExecutionMode {
	out := make([]ExecutionMode, len(supportedExecutionModes))
	copy(out, supportedExecutionModes)
	return out
}
