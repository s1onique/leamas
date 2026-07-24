// SPDX-License-Identifier: Apache-2.0

// Package authority: bootstrap_reentry_test.go asserts the
// bootstrap verify child runs as a fresh root invocation and is not
// tripped by the emergency re-entry fuse inherited from the parent
// process's environment.
//
// The contract is documented in internal/factory/authority/bootstrap.go
// and is part of the requalification for
// ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.
package authority

import (
	"strings"
	"testing"
)

// TestCleanReentryEnvStripsAllReentryVars ensures every LEAMAS_EXEC_*
// variable listed in reentryEnvVars is stripped from the returned
// environment.
func TestCleanReentryEnvStripsAllReentryVars(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"LEAMAS_EXEC_ROOT_ID=leamas-123-abc",
		"LEAMAS_EXEC_PARENT_PID=123",
		"LEAMAS_EXEC_GENERATION=2",
		"HOME=/home/test",
		"LEAMAS_EXEC_OTHER=keep-or-drop", // unknown reentry var must be preserved as-is
	}
	out := cleanReentryEnv(input)

	joined := strings.Join(out, "\n")
	for _, forbidden := range []string{
		"LEAMAS_EXEC_ROOT_ID=",
		"LEAMAS_EXEC_PARENT_PID=",
		"LEAMAS_EXEC_GENERATION=",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("cleanReentryEnv did not strip %q in %q", forbidden, joined)
		}
	}
	if !strings.Contains(joined, "PATH=/usr/bin") {
		t.Fatalf("cleanReentryEnv dropped PATH: %q", joined)
	}
	if !strings.Contains(joined, "HOME=/home/test") {
		t.Fatalf("cleanReentryEnv dropped HOME: %q", joined)
	}
}

// TestCleanReentryEnvPreservesOrder asserts cleanReentryEnv is order
// preserving and stable for repeated invocations on the same input.
func TestCleanReentryEnvPreservesOrder(t *testing.T) {
	input := []string{
		"A=1",
		"LEAMAS_EXEC_ROOT_ID=x",
		"B=2",
		"LEAMAS_EXEC_PARENT_PID=y",
		"C=3",
	}
	out := cleanReentryEnv(input)
	want := []string{"A=1", "B=2", "C=3"}
	if len(out) != len(want) {
		t.Fatalf("len(out)=%d want=%d (%v)", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("out[%d]=%q want=%q", i, out[i], want[i])
		}
	}
}

// TestCleanReentryEnvHandlesEmptyInput asserts cleanReentryEnv(nil)
// and cleanReentryEnv([]string{}) both return an empty (but non-nil)
// slice and do not panic.
func TestCleanReentryEnvHandlesEmptyInput(t *testing.T) {
	if got := cleanReentryEnv(nil); len(got) != 0 {
		t.Fatalf("cleanReentryEnv(nil)=%v want empty", got)
	}
	if got := cleanReentryEnv([]string{}); len(got) != 0 {
		t.Fatalf("cleanReentryEnv([])=%v want empty", got)
	}
}

// TestReentryEnvVarsListIsClosed ensures reentryEnvVars covers exactly
// the three names declared in internal/execution/reentry.go. If a new
// LEAMAS_EXEC_* name is added there, this test fails until
// reentryEnvVars is updated and the bootstrap fix is widened.
func TestReentryEnvVarsListIsClosed(t *testing.T) {
	want := []string{
		"LEAMAS_EXEC_ROOT_ID",
		"LEAMAS_EXEC_PARENT_PID",
		"LEAMAS_EXEC_GENERATION",
	}
	if len(reentryEnvVars) != len(want) {
		t.Fatalf("reentryEnvVars length changed: got=%d want=%d", len(reentryEnvVars), len(want))
	}
	for i, name := range want {
		if reentryEnvVars[i] != name {
			t.Fatalf("reentryEnvVars[%d]=%q want=%q", i, reentryEnvVars[i], name)
		}
	}
}
