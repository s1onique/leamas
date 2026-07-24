package closure

import (
	"errors"
	"strings"
	"testing"
)

func TestExecutionModePresenceString(t *testing.T) {
	cases := map[ExecutionModePresence]string{
		ExecutionModeMissing:           "missing",
		ExecutionModePresentEmpty:      "empty",
		ExecutionModePresentWhitespace: "whitespace",
		ExecutionModePresentUnknown:    "unknown",
		ExecutionModePresentValid:      "valid",
	}
	for p, want := range cases {
		if got := p.String(); got != want {
			t.Fatalf("ExecutionModePresence(%d).String() = %q, want %q", int(p), got, want)
		}
	}
	// Out-of-range sentinel.
	if got := ExecutionModePresence(99).String(); !strings.HasPrefix(got, "invalid(") {
		t.Fatalf("out-of-range String() = %q, want invalid()", got)
	}
}

func TestSupportedExecutionModesListsSerialFailFastExactlyOnce(t *testing.T) {
	got := SupportedExecutionModes()
	if len(got) != 1 {
		t.Fatalf("len(SupportedExecutionModes()) = %d, want 1", len(got))
	}
	if got[0] != ExecutionModeSerialFailFast {
		t.Fatalf("SupportedExecutionModes()[0] = %q, want %q", got[0], ExecutionModeSerialFailFast)
	}
	// Defensive-copy contract.
	got[0] = "mutated"
	again := SupportedExecutionModes()
	if again[0] != ExecutionModeSerialFailFast {
		t.Fatalf("SupportedExecutionModes() returned mutable slice; %q", again[0])
	}
}

func TestParseExecutionModeValid(t *testing.T) {
	mode, err := ParseExecutionMode(planExecutionModePath, string(ExecutionModeSerialFailFast))
	if err != nil {
		t.Fatalf("ParseExecutionMode(SerialFailFast) error = %v", err)
	}
	if mode != ExecutionModeSerialFailFast {
		t.Fatalf("ParseExecutionMode(SerialFailFast) = %q, want %q", mode, ExecutionModeSerialFailFast)
	}
}

func TestParseExecutionModeEmptyReturnsPresentEmpty(t *testing.T) {
	_, err := ParseExecutionMode(planExecutionModePath, "")
	if err == nil {
		t.Fatalf("ParseExecutionMode(\"\") accepted")
	}
	var typed *ExecutionModeError
	if !errors.As(err, &typed) {
		t.Fatalf("ParseExecutionMode(\"\") error = %v, want *ExecutionModeError", err)
	}
	if typed.Presence != ExecutionModePresentEmpty {
		t.Fatalf("Presence = %v, want %v", typed.Presence, ExecutionModePresentEmpty)
	}
	if typed.Path != planExecutionModePath {
		t.Fatalf("Path = %q, want %q", typed.Path, planExecutionModePath)
	}
	if !strings.Contains(typed.Error(), "empty") {
		t.Fatalf("Error() = %q, want substring 'empty'", typed.Error())
	}
}

func TestParseExecutionModeWhitespaceReturnsPresentWhitespace(t *testing.T) {
	_, err := ParseExecutionMode(planExecutionModePath, "   ")
	if err == nil {
		t.Fatalf("ParseExecutionMode(\"   \") accepted")
	}
	var typed *ExecutionModeError
	if !errors.As(err, &typed) {
		t.Fatalf("ParseExecutionMode(\"   \") error = %v, want *ExecutionModeError", err)
	}
	if typed.Presence != ExecutionModePresentWhitespace {
		t.Fatalf("Presence = %v, want %v", typed.Presence, ExecutionModePresentWhitespace)
	}
	if !strings.Contains(typed.Error(), "whitespace") {
		t.Fatalf("Error() = %q, want substring 'whitespace'", typed.Error())
	}
}

func TestParseExecutionModeUnknownReturnsPresentUnknown(t *testing.T) {
	_, err := ParseExecutionMode(planExecutionModePath, "parallel")
	if err == nil {
		t.Fatalf("ParseExecutionMode(\"parallel\") accepted")
	}
	var typed *ExecutionModeError
	if !errors.As(err, &typed) {
		t.Fatalf("ParseExecutionMode(\"parallel\") error = %v, want *ExecutionModeError", err)
	}
	if typed.Presence != ExecutionModePresentUnknown {
		t.Fatalf("Presence = %v, want %v", typed.Presence, ExecutionModePresentUnknown)
	}
	if !strings.Contains(typed.Error(), "parallel") {
		t.Fatalf("Error() = %q, want substring 'parallel'", typed.Error())
	}
	if !strings.Contains(typed.Error(), string(ExecutionModeSerialFailFast)) {
		t.Fatalf("Error() = %q, want substring '%s'", typed.Error(), ExecutionModeSerialFailFast)
	}
}

func TestIsExecutionModeError(t *testing.T) {
	if !IsExecutionModeError(&ExecutionModeError{Presence: ExecutionModeMissing}) {
		t.Fatalf("IsExecutionModeError rejected typed sentinel")
	}
	wrapped := error(&ExecutionModeError{Presence: ExecutionModeMissing})
	wrapped = errors.Join(errors.New("outer"), wrapped)
	if !IsExecutionModeError(wrapped) {
		t.Fatalf("IsExecutionModeError rejected wrapped typed sentinel")
	}
	if IsExecutionModeError(errors.New("plain")) {
		t.Fatalf("IsExecutionModeError matched plain error")
	}
	if IsExecutionModeError(nil) {
		t.Fatalf("IsExecutionModeError(nil) = true")
	}
}

func TestClassifyExecutionModeContract(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantMode ExecutionMode
		wantPres ExecutionModePresence
	}{
		{"canonical", string(ExecutionModeSerialFailFast), ExecutionModeSerialFailFast, ExecutionModePresentValid},
		{"empty-string", "", "", ExecutionModeMissing},
		{"whitespace", "   ", "", ExecutionModePresentWhitespace},
		{"trailing-space", "serial_fail_fast ", "", ExecutionModePresentUnknown},
		{"leading-tab", "\tserial_fail_fast", "", ExecutionModePresentUnknown},
		{"different-value", "parallel", "", ExecutionModePresentUnknown},
		{"uppercase", "SERIAL_FAIL_FAST", "", ExecutionModePresentUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotMode, gotPres := ClassifyExecutionMode(tc.input)
			if gotMode != tc.wantMode || gotPres != tc.wantPres {
				t.Fatalf("ClassifyExecutionMode(%q) = (%q, %v), want (%q, %v)",
					tc.input, gotMode, gotPres, tc.wantMode, tc.wantPres)
			}
		})
	}
}
